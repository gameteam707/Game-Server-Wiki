# 09 — API 設計與 Middleware

## API 設計風格

本專案的 HTTP API 遵循 RESTful 慣例。

### URL 結構

```
/api/{service}/v1/{resource}
     │          │   │
     │          │   └── 資源名稱（複數）
     │          └── API 版本號
     └── 服務名稱

範例：
/api/admin/v1/games           ← 遊戲列表
/api/admin/v1/games/:gameId   ← 特定遊戲
/api/admin/v1/auth/login      ← 登入（動作類）
```

### HTTP Method 對應 CRUD

| Method | 用途 | 範例 |
|--------|------|------|
| GET | 查詢（不改變狀態） | `GET /games` 列出遊戲 |
| POST | 新增 | `POST /games` 建立遊戲 |
| PUT | 更新 | `PUT /games/:id` 更新遊戲 |
| DELETE | 刪除 | `DELETE /games/:id` 刪除遊戲 |

### 統一回應格式

所有 API 回傳相同的信封格式：

```json
// 成功
{
    "success": true,
    "data": { ... },
    "meta": {
        "total": 100,
        "page": 1,
        "pageSize": 20
    }
}

// 失敗
{
    "success": false,
    "error": "game code already exists",
    "code": "DUPLICATE_CODE"
}
```

## Middleware 架構

Middleware 是 HTTP 請求的「攔截器」，在 Handler 執行之前或之後做額外處理。

### 執行順序

```
HTTP Request
     │
     ▼
┌─── CORS ───────────────────--┐  ← 所有請求
│    │                         │
│    ▼                         │
│    JWT Auth ───────────────┐ │  ← 受保護的路由
│    │                       │ │
│    ▼                       │ │
│    ReadOnly Check ────────┐│ │  ← 檢查唯讀角色
│    │                      ││ │
│    ▼                      ││ │
│    Audit Log ────────────┐││ │  ← 記錄操作日誌
│    │                     │││ │
│    ▼                     │││ │
│    RequirePermission ───┐│││││  ← 特定路由群組
│    │                    ││││││
│    ▼                    ││││││
│    Handler              ││││││
│    │                    ││││││
│    ▼（回程）             ││││││
│    Audit Log 記錄 ◀─────┘│││││
│    ◀────────────────────┘││││
│    ◀─────────────────────┘│││
│    ◀──────────────────────┘││
│    ◀───────────────────────┘│
└──── ◀───────────────────────┘
     │
     ▼
HTTP Response
```

### 1. JWT 認證 Middleware

```go
// 檔案：adapter/in/http/middleware/auth.go

func JWTAuth(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 從 Header 取得 Token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.AbortWithStatusJSON(401, gin.H{
                "success": false,
                "error":   "missing authorization header",
            })
            return
        }

        // 2. 解析 "Bearer <token>"
        token := strings.TrimPrefix(authHeader, "Bearer ")

        // 3. 驗證 JWT 簽名並解析 Claims
        claims, err := parseJWT(token, secret)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{
                "success": false,
                "error":   "invalid token",
            })
            return
        }

        // 4. 將使用者資訊存入 Context，後續 Handler 可以取用
        c.Set("userID", claims.UserID)
        c.Set("username", claims.Username)
        c.Set("role", claims.Role)
        c.Set("permissionGroupId", claims.PermissionGroupID)

        // 5. 繼續執行下一個 Middleware 或 Handler
        c.Next()
    }
}
```

**重點概念：`c.Set()` 和 `c.Get()`**

Gin 的 Context 可以在 Middleware 和 Handler 之間傳遞資料。JWT Middleware 驗證完成後，把使用者資訊「放進去」，Handler 就可以「拿出來」用。

### 2. RBAC 角色檢查

```go
// 檔案：adapter/in/http/middleware/rbac.go

// RequireRole — 檢查使用者是否具備指定角色
func RequireRole(allowedRoles ...entity.Role) gin.HandlerFunc {
    return func(c *gin.Context) {
        role := c.GetString("role")

        for _, allowed := range allowedRoles {
            if entity.Role(role) == allowed {
                c.Next()
                return
            }
        }

        c.AbortWithStatusJSON(403, gin.H{
            "success": false,
            "error":   "insufficient role",
        })
    }
}

// ReadOnly — 唯讀角色只能執行 GET 請求
func ReadOnly() gin.HandlerFunc {
    return func(c *gin.Context) {
        role := c.GetString("role")
        if entity.Role(role) == entity.RoleReadOnly && c.Request.Method != "GET" {
            c.AbortWithStatusJSON(403, gin.H{
                "success": false,
                "error":   "readonly users cannot modify data",
            })
            return
        }
        c.Next()
    }
}
```

### 3. 細粒度權限檢查

```go
// 檔案：adapter/in/http/middleware/permission.go

func RequirePermission(
    repo ports.PermissionGroupRepository,
    module entity.Module,
    action entity.Action,
) gin.HandlerFunc {
    return func(c *gin.Context) {
        role := c.GetString("role")

        // SuperAdmin 擁有所有權限，直接放行
        if entity.Role(role) == entity.RoleSuperAdmin {
            c.Next()
            return
        }

        // 查詢使用者的權限群組
        groupID := c.GetString("permissionGroupId")
        group, err := repo.FindByID(c.Request.Context(), groupID)
        if err != nil {
            c.AbortWithStatusJSON(403, gin.H{"error": "permission group not found"})
            return
        }

        // 檢查是否有該模組的該動作權限
        actions, exists := group.Permissions[module]
        if !exists {
            c.AbortWithStatusJSON(403, gin.H{"error": "no access to this module"})
            return
        }

        for _, a := range actions {
            if a == action {
                c.Next()
                return
            }
        }

        c.AbortWithStatusJSON(403, gin.H{"error": "insufficient permission"})
    }
}
```

### 4. 審計日誌

```go
// 檔案：adapter/in/http/middleware/audit.go

func AuditLog(createUC audit.CreateUseCase) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 讀取 Request Body（限制 1KB 避免記錄過大的內容）
        body, _ := io.ReadAll(io.LimitReader(c.Request.Body, 1024))
        c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

        // 先讓 Handler 執行
        c.Next()

        // 只記錄寫入操作（POST/PUT/DELETE）且成功的請求（2xx）
        if c.Request.Method == "GET" || c.Writer.Status() >= 300 {
            return
        }

        // 非同步記錄審計日誌
        go func() {
            createUC.Execute(context.Background(), &audit.CreateCommand{
                UserID:       c.GetString("userID"),
                Username:     c.GetString("username"),
                Action:       c.Request.Method,
                ResourceType: extractResourceType(c.Request.URL.Path),
                ResourceID:   extractResourceID(c.Request.URL.Path),
                Detail:       string(body),
                IP:           c.ClientIP(),
            })
        }()
    }
}
```

注意 Audit Log 使用了 **非同步（goroutine）** 記錄，不會阻塞回應。

### 路由群組的 Middleware 套用

```go
// Router 設定中，Middleware 按路由群組套用

// 公開路由（不需要認證）
public := r.Group("/api/admin/v1")
public.POST("/auth/login", authHandler.Login)

// 受保護路由（需要認證 + 唯讀檢查 + 審計日誌）
protected := r.Group("/api/admin/v1",
    middleware.JWTAuth(jwtSecret),
    middleware.ReadOnly(),
    middleware.AuditLog(auditLogUC),
)

// 遊戲路由（繼承 protected 的 middleware + 加上權限檢查）
games := protected.Group("/games",
    middleware.RequirePermission(permRepo, entity.ModuleGames, entity.ActionView),
)
games.GET("", gameHandler.List)
games.POST("",
    middleware.RequirePermission(permRepo, entity.ModuleGames, entity.ActionCreate),
    gameHandler.Create,
)
```

## 認證流程圖

```
Client                    Server
  │                         │
  │  POST /auth/login       │
  │  {username, password}   │
  │────────────────────────▶│
  │                         │ 1. 驗證帳密
  │                         │ 2. 產生 JWT Token
  │  {token: "eyJhbG..."}  │
  │◀────────────────────────│
  │                         │
  │  GET /games             │
  │  Authorization:         │
  │    Bearer eyJhbG...     │
  │────────────────────────▶│
  │                         │ 1. JWT Middleware 驗證 Token
  │                         │ 2. ReadOnly 檢查
  │                         │ 3. Permission 檢查
  │                         │ 4. Handler 執行
  │  {success: true, ...}   │ 5. Audit Log 記錄
  │◀────────────────────────│
```

## 給 Junior 的學習重點

### 你需要理解的概念

1. **JWT (JSON Web Token)**：無狀態的認證機制，Token 自帶使用者資訊
2. **Middleware Pattern**：請求處理的管線模式，可以任意組合
3. **RBAC**：基於角色的存取控制
4. **`c.Next()` 和 `c.Abort()`**：Gin 的中介軟體控制流程

### 動手練習

1. 用 curl 或 Postman 呼叫 login API，取得 JWT Token
2. 用 [jwt.io](https://jwt.io/) 解碼 Token，看裡面有什麼 Claims
3. 追蹤 `RequirePermission` middleware，理解權限檢查的完整流程
4. 思考：如果要新增一個「操作日誌的查詢權限」，需要修改哪些地方？
