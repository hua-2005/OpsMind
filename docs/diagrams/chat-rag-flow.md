# 智能问答 RAG 管道流程

> 涉及文件：`handler/chat.go` → `service/chat_service.go` → `rag/pipeline.go` → `adapter/llm_client.go`
> 管道：查询改写 → 多路检索 → 向量检索 → BM25检索 → RRF融合 → 重排序 → LLM生成

## 1. 完整 SSE 流式问答链路

```mermaid
sequenceDiagram
    actor U as 用户
    participant CV as Chat.vue
    participant API as streamChatSession()<br/>api/chat.ts
    participant CH as ChatHandler<br/>handler/chat.go
    participant CS as ChatService<br/>service/chat_service.go
    participant Pipe as Pipeline<br/>rag/pipeline.go
    participant Vec as PgvectorStore<br/>adapter/vector_store.go
    participant BM25 as BM25Retriever<br/>rag/bm25.go
    participant Rerank as Rerank<br/>rag/rerank.go
    participant LLM as OpenAIClient<br/>adapter/llm_client.go
    participant CR as ChatRepo<br/>repository/chat_repo.go
    participant DB as PostgreSQL

    U->>CV: 输入问题 + 选择知识库
    CV->>API: streamChatSession({question, kb_id, rag_options})
    API->>CH: POST /api/v1/portal/chat-sessions/stream<br/>Authorization: Bearer {token}

    CH->>CH: c.ShouldBindJSON(&CreateChatRequest)
    CH->>CH: getCurrentUserID(c) → userID
    CH->>CH: 设置 SSE headers<br/>Content-Type: text/event-stream

    CH->>CS: ChatService.CreateChatSession(req, userID)

    Note over CS: === 1. 参数校验 ===
    CS->>CS: strings.TrimSpace(req.Question) — 非空检查
    CS->>CS: knowledgeRepo.FindKBByID(req.KBID) — 知识库存在检查

    Note over CS,Pipe: === 2. RAG 管道 ===
    CS->>Pipe: Pipeline.Execute(ctx, question, kbID, RAGOptions, nil)

    alt QueryRewrite = true
        Pipe->>LLM: QueryRewrite(ctx, question, history)
        LLM-->>Pipe: 改写后查询
    end

    alt MultiRoute = true
        Pipe->>LLM: MultiRoute(ctx, rewrittenQuery) → []subQueries
        LLM-->>Pipe: 2-4 个子查询
    end

    par 每个查询独立检索
        Pipe->>Vec: VectorStore.CosineSearch(ctx, kbID, embedding, topK)
        Vec->>DB: SELECT * FROM knowledge_chunks<br/>ORDER BY embedding <=> $1 LIMIT $2
        DB-->>Vec: []SearchResult{ChunkID, Content, Score}
        Vec-->>Pipe: 向量检索结果
    and BM25 检索 (hybrid=true)
        Pipe->>BM25: BM25Retriever.Retrieve(kbID, query)
        BM25->>BM25: gse 中文分词 → 倒排索引查询
        BM25->>BM25: Okapi BM25 计分 (k1=1.5, b=0.75)
        BM25-->>Pipe: BM25 检索结果
    end

    alt Hybrid = true
        Pipe->>Pipe: HybridFuse(vectorResults, bm25Results)<br/>RRF_score(d) = Σ 1/(k+rank_i(d)), k=60
    end

    alt Rerank = true
        Pipe->>LLM: Rerank(ctx, question, topCandidates)
        LLM-->>Pipe: 重新排序后的 topK 分块
    end

    Pipe-->>CS: *RAGResult{Chunks, Metrics}

    Note over CS,LLM: === 3. LLM 生成 ===
    CS->>CS: 构造 System Prompt + Context (最多 3 条)
    CS->>LLM: LLMClient.ChatCompletion(ctx, ChatRequest{<br/>  Messages: [{system, context}, {user, question}],<br/>  Model, MaxTokens, Temperature: 0.3})

    alt LLM 成功
        LLM-->>CS: ChatResponse{Content, FinishReason}
    else LLM 失败
        CS-->>CH: AppError{20001, "AI 服务不可用"}
    end

    Note over CS: === 4. 保存会话 ===
    CS->>CR: ChatRepo.Create(&ChatSession{<br/>  UserID, KBID, Question, Answer, Confidence})
    CR->>DB: INSERT INTO chat_sessions
    DB-->>CR: session.ID

    CS-->>CH: *ChatSessionResponse{SessionID, Answer, Sources, Confidence}

    Note over CH: === 5. SSE 流式输出 ===
    CH->>CH: LLMClient.ChatCompletionStream(ctx, streamReq)

    loop 逐 token
        LLM-->>CH: StreamChunk{Content, FinishReason}
        CH->>CH: escapeSSE(chunk.Content)
        CH->>CH: fmt.Fprintf(w, "data: {\"type\":\"token\",\"content\":\"%s\"}\n\n")
        CH->>API: SSE event: token
        API->>CV: onToken(content) → 追加到 UI
    end

    CH->>CH: json.Marshal(resp) → metadata
    CH->>CH: fmt.Fprintf(w, "data: {\"type\":\"done\",\"metadata\":{...}}\n\n")
    CH->>API: SSE event: done
    API->>CV: onDone(session) → 展示来源/置信度

    Note over CV,U: 渲染完整答案 + 知识来源 + 管道耗时
```

## 2. 非流式（同步）问答

```mermaid
sequenceDiagram
    actor U as 用户
    participant CH as ChatHandler
    participant CS as ChatService
    participant Pipe as Pipeline
    participant LLM as LLMClient
    participant CR as ChatRepo

    U->>CH: POST /api/v1/portal/chat-sessions
    CH->>CS: ChatService.CreateChatSession(req, userID)
    CS->>Pipe: Pipeline.Execute(ctx, question, kbID, opts, nil)
    Pipe-->>CS: *RAGResult
    CS->>LLM: LLMClient.ChatCompletion(ctx, request)
    LLM-->>CS: ChatResponse
    CS->>CR: ChatRepo.Create(session)
    CS-->>CH: *ChatSessionResponse
    CH-->>U: 200 {"code":0, "data":{session_id, answer, sources, confidence, pipeline}}
```

## 3. 降级矩阵

```mermaid
flowchart TD
    Start([Pipeline.Execute]) --> QR{QueryRewrite?}
    QR -->|true| QR_LLM[QueryRewrite → LLM]
    QR -->|false| MR
    QR_LLM -->|成功| MR{MultiRoute?}
    QR_LLM -->|失败| QR_DG[降级：使用原始 question]
    QR_DG --> MR

    MR -->|true| MR_LLM[MultiRoute → LLM]
    MR -->|false| VR
    MR_LLM -->|成功| VR[VectorRetrieve]
    MR_LLM -->|失败| VR_DG[降级：单路检索]
    VR_DG --> VR

    VR -->|成功| BM{Hybrid?}
    VR -->|失败 ❌| VRFail[返回 code=20002]

    BM -->|true| BM25[BM25Retrieve]
    BM -->|false| Rerank
    BM25 -->|成功| Fuse[HybridFuse → RRF k=60]
    BM25 -->|失败| BM_DG[降级：仅向量结果]
    BM_DG --> Rerank
    Fuse --> Rerank

    Rerank{Rerank?} -->|true| Rerank_LLM[Rerank → LLM]
    Rerank -->|false| LLMGen
    Rerank_LLM -->|成功| LLMGen[LLM Generate]
    Rerank_LLM -->|失败| Rerank_DG[降级：RRF 排序结果]
    Rerank_DG --> LLMGen

    LLMGen -->|成功| Done([返回答案])
    LLMGen -->|失败 ❌| LLMFail[返回 code=20001]

    style VRFail fill:#ef444420,stroke:#ef4444
    style LLMFail fill:#ef444420,stroke:#ef4444
    style QR_DG fill:#f59e0b20,stroke:#f59e0b
    style VR_DG fill:#f59e0b20,stroke:#f59e0b
    style BM_DG fill:#f59e0b20,stroke:#f59e0b
    style Rerank_DG fill:#f59e0b20,stroke:#f59e0b
```
