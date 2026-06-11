# 智能问答 RAG 流程 (Smart Q&A with RAG Pipeline)

> **涉及文件：** `handler/chat.go` → `service/chat_service.go` → `adapter/rag_client.go` → AnythingLLM → vLLM
> **降级规则：** 与 ANYTHINGLLM_AI_INTEGRATION.md §7.1 完全对齐

---

## 1. 完整问答链路（含降级分支）

```mermaid
sequenceDiagram
    actor U as 用户 (门户端)
    participant CV as Chat.vue<br/>views/portal/
    participant CH as ChatHandler<br/>handler/chat.go
    participant CS as ChatService<br/>service/chat_service.go
    participant KR as KnowledgeRepo<br/>repository/knowledge_repo.go
    participant Rag as RagClient<br/>adapter/rag_client.go
    participant AL as AnythingLLM<br/>:3001/api
    participant vLLM as vLLM<br/>:8000/v1
    participant CR as ChatRepo<br/>repository/chat_repo.go
    participant DB as PostgreSQL

    U->>CV: 输入问题 + 选择知识库
    CV->>CH: POST /api/v1/portal/chat-sessions<br/>{question, kb_id}
    
    CH->>CH: c.ShouldBindJSON(&CreateChatRequest)
    CH->>CH: c.Get("currentUser") → userID
    CH->>CS: s.ChatService.CreateChatSession(req, userID)
    
    Note over CS: === 步骤1: 参数校验 ===
    CS->>CS: strings.TrimSpace(req.Question) == ""
    alt 问题为空
        CS-->>CH: AppError{10003, "问题不能为空"}
        CH-->>CV: {"code": 10003}
    end
    
    Note over CS: === 步骤2: 查询知识库 ===
    CS->>KR: FindKBByID(req.KBID)
    KR->>DB: SELECT * FROM knowledge_bases WHERE id = ?
    DB-->>KR: *KnowledgeBase (含 RAGWorkspaceSlug)
    KR-->>CS: kb
    
    alt 知识库不存在
        CS-->>CH: AppError{10004, "知识库不存在"}
        CH-->>CV: {"code": 10004}
    end
    
    Note over CS: === 步骤3: RAG 检索 + AI 生成 ===
    CS->>CS: start = time.Now()
    CS->>Rag: Query(ctx, RAGQueryRequest{<br/>  WorkspaceSlug, Question, TopK:5<br/>})
    
    Rag->>Rag: context.WithTimeout(30s)
    Rag->>AL: POST /api/v1/workspace/{slug}/chat<br/>Authorization: Bearer <api_key><br/>{message, mode:"query"}
    
    AL->>AL: 向量检索 (LanceDB)
    AL->>vLLM: POST /v1/chat/completions<br/>(OpenAI-compatible, RAG 增强 prompt)
    vLLM-->>AL: {choices[0].message.content}
    AL-->>Rag: {textResponse, sources[], chatId, error}
    
    Rag->>Rag: 字段映射:<br/>textResponse→Answer<br/>sources[].title→DocName<br/>sources[].text→ChunkContent<br/>max(sources[].score)→Confidence
    
    Rag-->>CS: *RAGQueryResponse{Answer, Sources, Confidence, Error}

    alt RagClient 不可达 (网络错误)
        Rag-->>CS: error
        CS-->>CH: AppError{20001, "当前 AI 服务暂不可用..."}
        CH-->>CV: {"code": 20001, "message": "AI 服务不可用"}
    end
    
    Note over CS: === 步骤4: 置信度判断 ===
    CS->>CS: durationMS = time.Since(start)
    
    alt ragResp.Error != "" 或 len(Sources)==0 或 Confidence < 0.6
        CS->>CS: answer = fallbackLowConfidence<br/>"暂未找到足够匹配的知识..."
        CS->>CS: canSubmit = true
    else 正常且置信度达标
        CS->>CS: answer = ragResp.Answer
        CS->>CS: sources = ragResp.Sources (映射)
        CS->>CS: canSubmit = false
    end
    
    Note over CS: === 步骤5: 持久化 ===
    CS->>CS: json.Marshal(sources) → sourcesJSON
    
    CS->>CR: Create(&ChatSession{<br/>  UserID, KBID, Question, Answer,<br/>  Sources(JSONB), Confidence, DurationMs<br/>})
    CR->>DB: INSERT INTO chat_sessions
    DB-->>CR: session.ID
    
    CS->>CR: CreateBatch([]ChatMessage{<br/>  {role:"user", content:question},<br/>  {role:"assistant", content:answer, sources, confidence}<br/>})
    CR->>DB: INSERT INTO chat_messages (批量)
    
    CS-->>CH: *ChatSessionResponse{<br/>  SessionID, Answer, Sources,<br/>  Confidence, CanSubmitTicket, DurationMS<br/>}
    CH-->>CV: {"code": 0, "data": {...}}
    
    CV->>U: 展示答案 + 来源文档 + 置信度<br/>canSubmit? → 展示"提交申告"按钮
```

---

## 2. 降级决策树

```mermaid
flowchart TD
    Start([ChatService.CreateChatSession]) --> FindKB[KnowledgeRepo.FindKBByID]
    FindKB --> KBExists{知识库存在?}
    KBExists -->|否| Err404[返回 10004]
    KBExists -->|是| CallRAG[RagClient.Query<br/>ctx 30s timeout]
    
    CallRAG --> NetOK{网络可达?}
    NetOK -->|否| Err20001[返回 code=20001<br/>AI 服务不可用]
    
    NetOK -->|是| HasError{ragResp.Error != ""?}
    HasError -->|是| Fallback[降级: 兜底答案<br/>can_submit_ticket=true]
    
    HasError -->|否| HasSources{sources 非空?}
    HasSources -->|否| Fallback
    
    HasSources -->|是| CheckConf{confidence >= 0.6?}
    CheckConf -->|否| Fallback
    
    CheckConf -->|是| Normal[正常返回<br/>answer + sources + confidence<br/>can_submit_ticket=false]
    
    Fallback --> SaveSession[保存 ChatSession]
    Normal --> SaveSession
    
    SaveSession --> SaveMsg[保存 ChatMessage x2<br/>user + assistant]
    SaveMsg --> Response[返回 ChatSessionResponse]
```

---

## 3. RagClient 字段映射（AnythingLLM → OpsMind）

```mermaid
flowchart LR
    subgraph AL["AnythingLLM 返回"]
        TR["textResponse"]
        SRC["sources[]"]
        TITLE["sources[].title"]
        TEXT["sources[].text"]
        SCORE["sources[].score"]
        ERR["error"]
    end
    
    subgraph RC["RagClient 解析<br/>adapter/rag_client.go"]
        MAP["字段映射逻辑"]
    end
    
    subgraph OM["OpsMind 字段"]
        ANS["Answer"]
        DOC["DocName"]
        CHK["ChunkContent"]
        CONF["Confidence"]
        ERR2["Error"]
    end
    
    TR --> MAP
    TITLE --> MAP
    TEXT --> MAP
    SCORE --> MAP
    ERR --> MAP
    SRC --> MAP
    
    MAP --> ANS
    MAP --> DOC
    MAP --> CHK
    MAP -->|"max(sources[].score)"| CONF
    MAP --> ERR2
```

---

## 4. 问答反馈流程

```mermaid
sequenceDiagram
    actor U as 用户
    participant CH as ChatHandler
    participant CS as ChatService<br/>SubmitFeedback
    participant CR as ChatRepo
    participant DB as PostgreSQL

    U->>CH: POST /api/v1/portal/chat-sessions/:id/feedback<br/>{feedback: 1(已解决) 或 2(未解决)}
    CH->>CS: s.ChatService.SubmitFeedback(sessionID, feedback)
    
    CS->>CR: FindByID(sessionID)
    CR->>DB: SELECT * FROM chat_sessions WHERE id = ?
    DB-->>CR: *ChatSession
    
    alt 会话不存在
        CR-->>CS: gorm.ErrRecordNotFound
        CS-->>CH: AppError{10004, "会话不存在"}
        CH-->>U: {"code": 10004}
    end
    
    CS->>CR: UpdateFeedback(sessionID, feedback)
    CR->>DB: UPDATE chat_sessions SET feedback = ?
    DB-->>CR: ok
    
    CS-->>CH: nil
    CH-->>U: {"code": 0}
    
    alt feedback == 2 (未解决)
        U->>U: 前端展示"提交申告"入口<br/>携带原始问题和问答上下文
    end
```
