# 文档上传与异步处理流程

> 涉及文件：`handler/knowledge.go` → `service/knowledge_service.go` → `rag/document_parser.go` / `rag/chunker.go` / `rag/embedder.go` / `rag/processor.go`

## 1. 完整上传→处理链路

```mermaid
sequenceDiagram
    actor A as 知识库管理员
    participant KH as KnowledgeHandler<br/>handler/knowledge.go
    participant KS as KnowledgeService<br/>service/knowledge_service.go
    participant DP as DocParser<br/>rag/document_parser.go
    participant KR as KnowledgeRepo<br/>repository/knowledge_repo.go
    participant PR as Processor<br/>rag/processor.go
    participant CH as Chunker<br/>rag/chunker.go
    participant EM as Embedder<br/>rag/embedder.go
    participant VS as PgvectorStore<br/>adapter/vector_store.go
    participant S3 as MinIO
    participant DB as PostgreSQL

    Note over A,DB: ====== 同步阶段 (Handler 线程) ======

    A->>KH: POST /api/v1/admin/knowledge-bases/:kb_id/documents/upload<br/>multipart/form-data: file=xxx.pdf

    KH->>KH: c.FormFile("file") → 获取上传文件
    KH->>KH: filepath.Ext(file.Filename) → 校验扩展名<br/>(.pdf/.docx/.md/.txt)
    KH->>KH: file.Size > 50MB → 拒绝
    KH->>KH: file.Open() → 文件流
    KH->>KH: getCurrentUserID(c) → userID

    KH->>KS: KnowledgeService.UploadDocuments(<br/>  kbID, userID, filename, fileType, src)

    Note over KS: === 步骤1: 解析文档文本 ===
    KS->>DP: DocParser.Parse(fileReader, fileType)

    alt fileType == "pdf"
        DP->>DP: pdf.NewReader(reader) → 遍历页 → 提取文本
    else fileType == "docx"
        DP->>DP: archive/zip 解压 → encoding/xml 解析<br/>→ 提取 w:t 元素文本
    else fileType == "md" or "txt"
        DP->>DP: io.ReadAll → 直接读取
    end

    DP-->>KS: 纯文本内容 string

    KS->>KS: strings.TrimSpace(text) → 空内容检查

    Note over KS: === 步骤2: 创建文章记录 ===
    KS->>KR: KnowledgeRepo.CreateArticle(&KnowledgeArticle{<br/>  KBID, Question:filename, Answer:text,<br/>  Category:"文档上传", Status:1(草稿)})
    KR->>DB: INSERT INTO knowledge_articles
    DB-->>KR: article.ID

    Note over KS,PR: === 步骤3: 提交异步处理 ===
    KS->>PR: Processor.Submit(ProcessTask{<br/>  ArticleID, KBID, Content: text,<br/>  OnStatusChange: callback})

    KS-->>KH: *KnowledgeArticle{ID, Question, Answer}
    KH-->>A: 200 {article_id, filename, kb_id, message:"文档已接收，正在后台处理"}

    Note over PR,DB: ====== 异步阶段 (goroutine pool) ======

    PR->>PR: goroutine pool 取任务

    PR->>PR: callback(articleID, "parsing", "")
    PR->>KR: KnowledgeRepo.UpdateArticleStatus(id, 1)

    PR->>PR: callback(articleID, "chunking", "")
    PR->>CH: Chunker.Split(text)
    CH->>CH: RecursiveCharacterTextSplitter<br/>(chunk_size=1000, overlap=200)
    CH-->>PR: []string chunks

    PR->>PR: callback(articleID, "embedding", "")
    PR->>EM: Embedder.Embed(ctx, chunks)
    EM-->>PR: ([][]float32 vectors, dimension, nil)

    PR->>VS: VectorStore.DeleteByArticle(ctx, articleID)
    VS->>DB: DELETE FROM knowledge_chunks WHERE article_id=?

    PR->>VS: VectorStore.BatchInsert(ctx, vectorChunks)
    VS->>DB: INSERT INTO knowledge_chunks (article_id, kb_id, content,<br/>  chunk_index, embedding, embedding_model, vector_dimension)

    alt 全部成功
        PR->>PR: callback(articleID, "completed", "")
        PR->>KR: KnowledgeRepo.UpdateArticleStatus(id, 3)
        KR->>DB: UPDATE knowledge_articles SET status=3
    else 任一步骤失败
        PR->>PR: callback(articleID, "failed", errorMsg)
        PR->>KR: KnowledgeRepo.UpdateArticleStatus(id, 1)
    end
```

## 2. 处理状态流转

```mermaid
stateDiagram-v2
    [*] --> pending : UploadDocuments()<br/>创建文章 + 入队
    pending --> parsing : Processor 取任务
    parsing --> chunking : Chunker.Split()
    chunking --> embedding : Embedder.Embed()
    embedding --> completed : VectorStore.BatchInsert()<br/>文章状态→已通过(3)

    parsing --> failed : 解析异常
    chunking --> failed : 分块异常
    embedding --> failed : API 调用异常

    failed --> pending : RetryDocument()<br/>重置→重新入队
    completed --> [*]
```

## 3. 支持的文件格式

```mermaid
flowchart LR
    Upload["UploadDocuments()"] --> Ext{filepath.Ext}
    Ext -->|".pdf"| PDF["DocParser.Parse(reader, \"pdf\")<br/>pdf.NewReader → 遍历页提取文本"]
    Ext -->|".docx"| DOCX["DocParser.Parse(reader, \"docx\")<br/>archive/zip → encoding/xml"]
    Ext -->|".md"| MD["DocParser.Parse(reader, \"md\")<br/>io.ReadAll 直接读取"]
    Ext -->|".txt"| TXT["DocParser.Parse(reader, \"txt\")<br/>io.ReadAll 直接读取"]
    Ext -->|其他| Reject["返回 10003<br/>不支持的文件格式"]

    PDF --> Text["纯文本 string"]
    DOCX --> Text
    MD --> Text
    TXT --> Text
    Text --> Article["CreateArticle()"]
```
