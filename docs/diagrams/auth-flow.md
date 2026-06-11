# 认证与授权流程 (Authentication & Authorization Flow)

> **涉及文件：** `handler/auth.go` → `service/auth_service.go` → `repository/user_repo.go` → `pkg/jwt/` → `pkg/hash/`
> **中间件：** `middleware/auth.go` (JWTAuth), `middleware/rbac.go` (RequirePermission)

---

## 1. 用户登录完整流程

```mermaid
sequenceDiagram
    actor U as 用户 (浏览器)
    participant H as AuthHandler<br/>handler/auth.go
    participant S as AuthService<br/>service/auth_service.go
    participant R as UserRepo<br/>repository/user_repo.go
    participant J as pkg/jwt
    participant B as pkg/hash
    participant DB as PostgreSQL

    U->>H: POST /api/v1/auth/login<br/>{username, password}
    H->>H: c.ShouldBindJSON(&LoginRequest)
    H->>S: s.AuthService.Login(username, password)
    
    S->>R: GetByUsername(username)
    R->>DB: SELECT * FROM users WHERE username = ?
    DB-->>R: user row
    R-->>S: *model.User
    
    alt 用户不存在
        S-->>H: AppError{Code: 10003, "用户名或密码错误"}
        H-->>U: {"code": 10003, "message": "用户名或密码错误"}
    end
    
    S->>B: CheckPassword(user.PasswordHash, password)
    alt 密码错误
        S-->>H: AppError{Code: 10003, "用户名或密码错误"}
        H-->>U: {"code": 10003, "message": "用户名或密码错误"}
    end
    
    alt 账号已冻结 (status=2)
        S-->>H: AppError{Code: 10002, "账号已被冻结"}
        H-->>U: {"code": 10002, "message": "账号已被冻结"}
    end
    
    S->>S: buildLoginResponse(user)
    
    rect rgb(30, 40, 60)
        Note over S,DB: buildLoginResponse 内部调用链
        S->>R: GetUserRoles(userID)
        R->>DB: SELECT r.* FROM roles r<br/>JOIN user_roles ur ON r.id = ur.role_id<br/>WHERE ur.user_id = ?
        DB-->>R: []Role
        R-->>S: []Role
        
        S->>R: GetUserPermissions(userID)
        R->>DB: 聚合所有角色的 permissions JSONB
        DB-->>R: []string (permissions)
        R-->>S: []string
        
        S->>S: buildMenuTree(userID, roles)
        Note over S: 系统管理员 → ListMenus() 全量<br/>其他角色 → 按角色聚合 + 去重
        
        S->>J: GenerateAccessToken(userID, username, roles, secret, 2h)
        J-->>S: accessToken (JWT)
        
        S->>J: GenerateRefreshToken(userID, username, roles, secret, 7d)
        J-->>S: refreshToken (JWT)
    end
    
    S-->>H: *LoginResponse{AccessToken, RefreshToken, User, Roles, Permissions, Menus}
    H-->>U: {"code": 0, "data": {access_token, refresh_token, user, roles, permissions, menus}}
```

---

## 2. JWT 认证中间件

```mermaid
sequenceDiagram
    actor C as 客户端
    participant MW as JWTAuth<br/>middleware/auth.go
    participant J as pkg/jwt.ParseToken
    participant Next as 下一个 Handler

    C->>MW: 请求 + Authorization: Bearer <token>
    
    MW->>MW: c.GetHeader("Authorization")
    
    alt 缺失 Authorization header
        MW-->>C: {"code": 10001, "message": "未登录或令牌过期"}
    end
    
    MW->>MW: strings.TrimPrefix(authHeader, "Bearer ")
    
    alt 格式错误（无 Bearer 前缀）
        MW-->>C: {"code": 10001, "message": "未登录或令牌过期"}
    end
    
    MW->>J: ParseToken(tokenString, jwtSecret())
    
    alt 令牌过期或无效
        J-->>MW: error
        MW-->>C: {"code": 10001, "message": "未登录或令牌过期"}
    end
    
    J-->>MW: *Claims{UserID, Username, Roles}
    MW->>MW: c.Set("currentUser", CurrentUser{UserID, Username, Roles})
    MW->>Next: c.Next()
```

---

## 3. RBAC 权限中间件

```mermaid
sequenceDiagram
    actor C as 客户端
    participant MW as RequirePermission<br/>middleware/rbac.go
    participant R as UserRepo<br/>repository/user_repo.go
    participant Next as 下一个 Handler

    C->>MW: 请求 (已通过 JWTAuth)
    MW->>MW: c.Get("currentUser") → currentUser
    
    alt 未找到 currentUser (JWTAuth 未执行)
        MW-->>C: {"code": 10001, "message": "未登录或令牌过期"}
    end
    
    Note over MW: 检查是否为系统管理员<br/>遍历 roles，匹配 "系统管理员"
    
    alt 是系统管理员
        MW->>Next: c.Next() (自动放行)
    end
    
    MW->>R: GetUserPermissions(userID)
    R-->>MW: []string (用户所有权限)
    
    alt 用户不包含目标权限
        MW-->>C: {"code": 10002, "message": "无权限"}
    end
    
    MW->>Next: c.Next() (权限校验通过)
```

---

## 4. 修改密码流程

```mermaid
sequenceDiagram
    actor U as 已登录用户
    participant H as AuthHandler<br/>handler/auth.go
    participant S as AuthService<br/>service/auth_service.go
    participant R as UserRepo<br/>repository/user_repo.go
    participant B as pkg/hash

    U->>H: POST /api/v1/auth/change-password<br/>{old_password, new_password}
    H->>H: c.Get("currentUser") → userID
    H->>S: s.AuthService.ChangePassword(userID, oldPwd, newPwd)
    
    S->>R: GetByID(userID)
    R-->>S: *model.User
    
    S->>B: CheckPassword(user.PasswordHash, oldPwd)
    alt 旧密码错误
        S-->>H: AppError{Code: 10003, "旧密码错误"}
        H-->>U: {"code": 10003}
    end
    
    S->>B: ValidatePassword(newPwd)
    Note over B: 正则: ^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,32}$
    alt 密码不符合策略
        S-->>H: AppError{Code: 10003, "密码必须包含..."}
        H-->>U: {"code": 10003}
    end
    
    S->>B: HashPassword(newPwd)
    B-->>S: passwordHash (bcrypt cost=10)
    
    S->>S: user.FirstLogin = false
    S->>R: Update(user)
    R->>DB: UPDATE users SET password_hash=?, first_login=false
    
    S-->>H: nil (成功)
    H-->>U: {"code": 0, "message": "success"}
```

---

## 5. 令牌刷新流程

```mermaid
sequenceDiagram
    actor C as 客户端
    participant H as AuthHandler
    participant S as AuthService<br/>RefreshToken
    participant J as pkg/jwt

    C->>H: POST /api/v1/auth/refresh<br/>{refresh_token}
    H->>S: s.AuthService.RefreshToken(refreshToken)
    
    S->>J: ParseToken(refreshToken, jwtSecret())
    alt 令牌无效
        J-->>S: error
        S-->>H: AppError{Code: 10001, "刷新令牌无效或已过期"}
        H-->>C: {"code": 10001}
    end
    
    J-->>S: *Claims{UserID, Username, Roles}
    S->>S: userRepo.GetByID(claims.UserID)
    S->>S: 检查 user.Status ≠ 2 (冻结)
    S->>S: buildLoginResponse(user) → 生成新令牌对
    
    S-->>H: *LoginResponse (新 access_token + refresh_token)
    H-->>C: {"code": 0, "data": {...}}
```
