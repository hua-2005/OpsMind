# 申告状态机与生命周期 (Ticket State Machine & Lifecycle)

> **涉及文件：** `handler/ticket.go` → `service/ticket_service.go` → `repository/ticket_repo.go`
> **调度器：** `service/scheduler.go` (TicketAutoCloseJob)
> **消息：** `service/message_service.go` (NotifySupplement)

---

## 1. 完整申告生命周期

```mermaid
sequenceDiagram
    actor R as 报障人 (门户端)
    actor O as 运维人员 (后台)
    participant PH as TicketHandler<br/>(Portal)
    participant AH as TicketHandler<br/>(Admin)
    participant TS as TicketService<br/>service/ticket_service.go
    participant TR as TicketRepo<br/>repository/ticket_repo.go
    participant MS as MessageService<br/>service/message_service.go
    participant DB as PostgreSQL

    Note over R,DB: === 阶段1: 创建申告 ===
    R->>PH: POST /api/v1/portal/tickets<br/>{title, description, urgency, contact_phone, ...}
    PH->>TS: s.TicketService.CreateTicket(req, userID)
    
    TS->>TS: 参数校验: title/description/contact_phone 必填<br/>urgency 必须 1-3
    
    TS->>TS: 生成 ticket_no:<br/>TK-YYYYMMDD-XXXX (随机后缀)
    TS->>TS: Status=1 (待处理), Source=1 (门户提交)
    
    TS->>TR: Create(&Ticket{...})
    TR->>DB: INSERT INTO tickets
    DB-->>TR: ok
    TS-->>PH: nil
    PH-->>R: {"code": 0, "data": {ticket_no, status:"待处理"}}

    Note over R,DB: === 阶段2: 开始处理 ===
    O->>AH: PATCH /api/v1/admin/tickets/:id/status<br/>{action: "start"}
    AH->>TS: s.TicketService.UpdateStatus(id, operatorID, req)
    
    TS->>TR: FindByID(id) → ticket
    TR->>DB: SELECT * FROM tickets WHERE id = ?
    DB-->>TR: *Ticket{Status: 1}
    
    TS->>TS: switch action "start":<br/>  校验 ticket.Status == 1<br/>  newStatus = 2, recordAction = "start"
    
    TS->>TR: UpdateStatus(id, 2)
    TR->>DB: UPDATE tickets SET status = 2
    TS->>TR: CreateRecord({action:"start", operatorID, content})
    TR->>DB: INSERT INTO ticket_records
    
    TS-->>AH: nil
    AH-->>O: {"code": 0}

    Note over R,DB: === 阶段3: 需补充信息（循环，最多3次）===
    O->>AH: PATCH /api/v1/admin/tickets/:id/status<br/>{action: "request_info"}
    AH->>TS: s.TicketService.UpdateStatus(id, operatorID, req)
    
    TS->>TR: FindByID(id) → ticket
    TS->>TS: switch action "request_info":<br/>  校验 ticket.Status == 2<br/>  校验 supplement_count < 3
    
    alt supplement_count >= 3
        TS-->>AH: AppError{10003, "补充信息次数已达上限（3次）"}
        AH-->>O: {"code": 10003}
    end
    
    TS->>TR: IncrementSupplementCount(id)
    TR->>DB: UPDATE tickets SET supplement_count = supplement_count + 1
    TS->>TR: UpdateStatus(id, 3)
    TS->>TR: CreateRecord({action:"request_info"})
    
    Note over TS,MS: 同步触发站内消息通知
    TS->>MS: NotifySupplement(ticketID, userID)
    MS->>DB: INSERT INTO messages<br/>{type:"ticket_supplement", user_id, ...}
    
    TS-->>AH: nil
    
    Note over R,DB: 报障人补充信息
    R->>PH: PATCH /api/v1/portal/tickets/:id/supplement<br/>{content: "补充内容"}
    PH->>TS: s.TicketService.SupplementTicket(id, userID, req)
    
    TS->>TR: FindByID(id)
    TS->>TS: 校验: ticket.UserID == userID (仅本人)
    TS->>TS: 校验: ticket.Status == 3 (需补充信息)
    
    TS->>TR: CreateRecord({action:"supplement", content})
    TS->>TR: UpdateStatus(id, 2) (回到处理中)
    TS-->>PH: nil
    
    Note over R,DB: === 阶段4: 处理完成 ===
    O->>AH: PATCH /api/v1/admin/tickets/:id/status<br/>{action: "resolve", result: "处理结果"}
    AH->>TS: s.TicketService.UpdateStatus(id, operatorID, req)
    
    TS->>TS: switch action "resolve":<br/>  校验 ticket.Status == 2<br/>  newStatus = 4, recordAction = "resolve"
    
    TS->>TR: UpdateStatus(id, 4)
    TS->>TR: CreateRecord({action:"resolve", content: result})
    TS-->>AH: nil
    AH-->>O: {"code": 0}

    Note over R,DB: === 可选: 关闭申告 ===
    O->>AH: PATCH /api/v1/admin/tickets/:id/status<br/>{action: "close"}
    AH->>TS: UpdateStatus(id, operatorID, req)
    
    TS->>TS: switch action "close":<br/>  任意状态 → newStatus = 5
    TS->>TR: UpdateStatus(id, 5)
    TS->>TR: CreateRecord({action:"close"})
```

---

## 2. 状态机转换图

```mermaid
stateDiagram-v2
    [*] --> 待处理: CreateTicket()<br/>status=1
    
    待处理 --> 处理中: UpdateStatus(action:"start")<br/>仅运维人员
    
    处理中 --> 需补充信息: UpdateStatus(action: "request_info")<br/>supplement_count 自增 (不超过3次)
    需补充信息 --> 处理中: SupplementTicket()<br/>仅报障人本人
    
    处理中 --> 已解决: UpdateStatus(action:"resolve")<br/>仅运维人员
    
    待处理 --> 已关闭: UpdateStatus(action:"close")<br/>手动关闭
    处理中 --> 已关闭: UpdateStatus(action:"close")<br/>手动 / Scheduler.AutoCloseJob
    需补充信息 --> 已关闭: UpdateStatus(action:"close")<br/>手动 / Scheduler.AutoCloseJob
    
    已解决 --> [*]
    已关闭 --> [*]

    note right of 需补充信息
        每次 request_info 触发:
        MessageService.NotifySupplement()
        → 写入 messages 表
        → 门户端顶部提示条
    end note
    
    note right of 已关闭
        Scheduler.TicketAutoCloseJob
        每小时执行:
        status IN (1,2,3)
        AND created_at < NOW() - 7天
        → 批量 status=5
    end note
```

---

## 3. 后台调度器：7 天自动关闭

```mermaid
flowchart TD
    Start([Scheduler.Start]) --> GoRoutine[go runAutoCloseLoop(ctx)]
    GoRoutine --> Ticker[time.NewTicker(1h)]
    
    Ticker --> Loop{收到 tick?}
    Loop -->|是| Query[TicketRepo.AutoCloseTickets(7天前)]
    Query --> SQL["SELECT ... FROM tickets<br/>WHERE status IN (1,2,3)<br/>AND created_at < NOW() - INTERVAL '7 days'"]
    SQL --> Update["UPDATE tickets SET status = 5<br/>WHERE ..."]
    Update --> Log["写入 audit_log<br/>action='auto_close'"]
    Log --> CheckCtx{ctx.Done()?}
    
    Loop -->|否| CheckCtx
    
    CheckCtx -->|否| Loop
    CheckCtx -->|是 (Scheduler.Stop)| Stop([退出 goroutine])
```

---

## 4. 申告查询流程（门户端/后台）

```mermaid
sequenceDiagram
    actor R as 报障人
    actor O as 运维人员
    participant PH as TicketHandler (Portal)
    participant AH as TicketHandler (Admin)
    participant TS as TicketService
    participant TR as TicketRepo
    participant DB as PostgreSQL

    Note over R,DB: 门户端: 查询我的申告
    R->>PH: GET /api/v1/portal/tickets?page=1&page_size=20
    PH->>TS: s.TicketService.ListByUser(userID, page, pageSize)
    TS->>TR: ListByUser(userID, page, pageSize)
    TR->>DB: SELECT * FROM tickets<br/>WHERE user_id = ?<br/>ORDER BY created_at DESC<br/>LIMIT ? OFFSET ?
    DB-->>TR: []Ticket, total
    TR-->>TS: ([]Ticket, total)
    TS-->>PH: *TicketListResponse
    PH-->>R: {"code": 0, "data": {tickets, total, page, page_size}}

    Note over R,DB: 后台: 全量筛选查询
    O->>AH: GET /api/v1/admin/tickets?status=1&urgency=2&page=1
    AH->>TS: s.TicketService.ListAll(status, urgency, page, pageSize)
    TS->>TR: ListAll(status, urgency, page, pageSize)
    TR->>DB: SELECT * FROM tickets<br/>WHERE status = ? AND urgency = ?<br/>ORDER BY created_at DESC
    DB-->>TR: []Ticket, total
    TR-->>TS: ([]Ticket, total)
    TS-->>AH: *TicketListResponse
    AH-->>O: {"code": 0, "data": {...}}
```
