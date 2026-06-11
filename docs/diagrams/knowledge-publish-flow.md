# 知识发布与同步流程 (Knowledge Publish & Sync Flow)

> **涉及文件：** `handler/knowledge.go` → `service/knowledge_service.go` → `adapter/rag_client.go` → AnythingLLM
> **同步状态：** pending → synced / failed / disabled

---

## 1. 知识库创建流程

```mermaid
sequenceDiagram
    actor A as 知识库管理员
    participant KH as KnowledgeHandler<br/>handler/knowledge.go
    participant KS as KnowledgeService<br/>service/knowledge_service.go
    participant KR as KnowledgeRepo<br/>repository/knowledge_repo.go
    participant Rag as RagClient<br/>adapter/rag_client.go
    participant AL as AnythingLLM<br/>:3001/api
    participant DB as PostgreSQL

    A->>KH: POST /api/v1/admin/knowledge-bases<br/>{name, description}
    KH->>KS: s.KnowledgeService.CreateKB(req, userID)
    
    Note over KS: === 步骤1: 创建 AnythingLLM 工作区 ===
    KS->>Rag: CreateWorkspace(ctx, RAGCreateWorkspaceRequest{<br/>  Name, EmbeddingModel, EmbeddingEngine:"generic-openai"<br/>})
    Rag->>AL: POST /api/v1/workspace/new<br/>{name, chatMode:"query", topN:5,<br/> similarityThreshold:0.6}
    AL-->>Rag: {workspace: {slug}}
    Rag-->>KS: *RAGCreateWorkspaceResponse{Slug}
    
    alt 创建失败
        KS-->>KH: AppError{20002, "创建 RAG 工作区失败"}
        KH-->>A: {"code": 20002}
    end
    
    Note over KS: === 步骤2: 持久化知识库 ===
    KS->>KR: CreateKB(&KnowledgeBase{<br/>  Name, Description,<br/>  RAGWorkspaceSlug, EmbeddingModel, CreatedBy<br/>})
    KR->>DB: INSERT INTO knowledge_bases
    DB-->>KR: ok
    
    KS-->>KH: nil
    KH-->>A: {"code": 0}
```

---

## 2. 知识文章完整生命周期

```mermaid
stateDiagram-v2
    [*] --> 草稿: CreateArticle()<br/>status=1
    
    草稿 --> 待审核: SubmitReview()<br/>仅草稿可提交
    
    待审核 --> 已通过: Review(approved=true)<br/>审核人≠创建人
    
    待审核 --> 已驳回: Review(approved=false)<br/>必须填写 review_comment
    
    已驳回 --> 草稿: UpdateArticle()<br/>驳回后可重新编辑
    已驳回 --> 待审核: SubmitReview()<br/>重新提交
    
    已通过 --> 已发布: Publish()<br/>调用 RagClient.SyncDocument
    
    已发布 --> 已停用: Disable()<br/>调用 RagClient.DisableDocument
    
    已停用 --> [*]

    note right of 已发布
        Publish 内部流程:
        1. RagClient.SyncDocument()
        2. 保存 rag_document_location
        3. 更新 sync_status
        4. 写入 pgvector chunks
        ---
        失败时:
        article.status 仍为 4 (已发布)
        chunk.sync_status = 'failed'
        chunk.sync_error = 错误详情
        支持 RetrySync() 重试
    end note
```

---

## 3. 发布同步详细流程

```mermaid
sequenceDiagram
    actor A as 知识库管理员
    participant KH as KnowledgeHandler
    participant KS as KnowledgeService<br/>Publish()
    participant KR as KnowledgeRepo
    participant Rag as RagClient<br/>SyncDocument
    participant AL as AnythingLLM
    participant DB as PostgreSQL

    A->>KH: POST /api/v1/admin/articles/:id/publish
    KH->>KS: s.KnowledgeService.Publish(id, publisherID)
    
    KS->>KR: FindArticleByID(id) → article<br/>(预加载 KnowledgeBase)
    KR->>DB: SELECT * FROM knowledge_articles<br/>JOIN knowledge_bases
    DB-->>KR: *KnowledgeArticle{KnowledgeBase}
    
    alt article.Status != 3 (已审核通过)
        KS-->>KH: AppError{10003, "仅已审核通过的文章可发布"}
        KH-->>A: {"code": 10003}
    end
    
    Note over KS: === 同步到 AnythingLLM ===
    KS->>Rag: SyncDocument(ctx, RAGSyncRequest{<br/>  WorkspaceSlug, Title, Content, Mode:"raw-text"<br/>})
    Rag->>AL: POST /api/v1/document/raw-text<br/>{textContent, addToWorkspaces, metadata}
    AL-->>Rag: {success:true, documents:[{location}]}
    Rag-->>KS: *RAGSyncResponse{DocumentLocation}
    
    alt 同步失败
        Rag-->>KS: error
    end
    
    Note over KS: === 更新文章状态 ===
    KS->>KS: article.Status = 4 (已发布)
    KS->>KS: article.PublishedBy = &publisherID
    alt 同步成功
        KS->>KS: article.RAGDocumentLocation = syncResp.DocumentLocation
    end
    KS->>KR: UpdateArticle(article)
    KR->>DB: UPDATE knowledge_articles SET status=4, ...
    
    Note over KS: === 更新切片同步状态 ===
    alt 同步成功
        KS->>KR: FindChunksByArticleID(id)
        alt 已有 chunks
            KS->>KR: UpdateChunkSyncStatus(id, "synced", "")
        else 无 chunks (首次发布)
            KS->>KR: CreateChunks([{content, embedding,<br/>  embedding_model, vector_dimension,<br/>  sync_status:"synced"}])
        end
    else 同步失败
        KS->>KR: UpdateChunkSyncStatus(id, "failed", syncErr.Error())
    end
    
    KS-->>KH: nil
    KH-->>A: {"code": 0}
```

---

## 4. 停用与重试流程

```mermaid
sequenceDiagram
    actor A as 知识库管理员
    participant KH as KnowledgeHandler
    participant KS as KnowledgeService
    participant KR as KnowledgeRepo
    participant Rag as RagClient
    participant AL as AnythingLLM
    participant DB as PostgreSQL

    Note over A,DB: ===== 停用流程 =====
    A->>KH: POST /api/v1/admin/articles/:id/disable
    KH->>KS: s.KnowledgeService.Disable(id)
    
    KS->>KR: FindArticleByID(id) → article
    
    alt article.RAGDocumentLocation != ""
        KS->>Rag: DisableDocument(ctx, RAGDisableRequest{<br/>  WorkspaceSlug, DocumentLocations: [article.RAGDocumentLocation]<br/>})
        Rag->>AL: POST /api/v1/workspace/{slug}/update-embeddings<br/>{deletes: [document_location]}
        AL-->>Rag: ok (忽略错误)
    end
    
    KS->>KR: UpdateArticle(article) → status=0
    KS->>KR: UpdateChunkStatusByArticleID(id, "disabled")
    KR->>DB: UPDATE knowledge_articles SET status=0<br/>UPDATE knowledge_chunks SET sync_status='disabled'
    
    KS-->>KH: nil
    KH-->>A: {"code": 0}

    Note over A,DB: ===== 重试同步 =====
    A->>KH: POST /api/v1/admin/articles/:id/retry-sync
    KH->>KS: s.KnowledgeService.Publish(id, publisherID)
    Note over KS: 重新执行完整 Publish 流程<br/>（同上，重新调用 RagClient.SyncDocument）
```

---

## 5. Embedding 配置管理

```mermaid
flowchart TD
    Start([系统管理员]) --> List[GET /admin/embedding-configs<br/>KnowledgeService.ListEmbeddingConfigs]
    List --> Table[展示所有 Embedding 配置]
    
    Table --> Create[POST 新增]
    Table --> Update[PUT 更新]
    Table --> Delete[DELETE 删除]
    
    Create --> Validate{model_type?}
    Validate -->|1: API 接入| CheckAPI["校验 api_endpoint 必填"]
    Validate -->|2: 本地部署| CheckLocal["校验 local_path 必填"]
    
    CheckAPI --> SetDefault{is_default == true?}
    CheckLocal --> SetDefault
    
    SetDefault -->|是| UnsetOthers["将所有其他配置的<br/>is_default 设为 false"]
    SetDefault -->|否| Save["INSERT INTO embedding_configs"]
    UnsetOthers --> Save
```
