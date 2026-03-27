# 07 — Infrastructure Layer（基礎設施層）

## 這一層是什麼？

Infrastructure Layer 處理「技術細節」——怎麼連資料庫、怎麼讀設定、怎麼啟動伺服器。它是最外層，可以依賴所有其他層。

## 位置

```
internal/infrastructure/
├── config/        ← 設定管理
├── di/            ← 依賴注入容器
├── mongodb/       ← MongoDB 連線
├── redis/         ← Redis 連線
├── server/        ← HTTP/gRPC 伺服器與路由
└── metrics/       ← Prometheus 指標（gameservice）
```

## 設定管理（config/）

所有服務使用相同模式：**環境變數 + 預設值**。

```go
// 檔案：admin-backend/server/internal/infrastructure/config/config.go

package config

import "os"

type Config struct {
    Port              string
    MongoURI          string
    MongoDB           string
    JWTSecret         string
    JWTExpiry         string
    RedisAddr         string
    GameAPIURL        string
    SuperAdminUser    string
    SuperAdminPass    string
    DummyPlatformURL  string
}

func Load() *Config {
    return &Config{
        Port:             getEnv("SERVER_PORT", "8090"),
        MongoURI:         getEnv("MONGO_URI", "mongodb://localhost:27017"),
        MongoDB:          getEnv("MONGO_DB", "admin_backend"),
        JWTSecret:        getEnv("JWT_SECRET", "default-secret"),
        JWTExpiry:        getEnv("JWT_EXPIRY", "24h"),
        RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
        GameAPIURL:       getEnv("GAME_API_URL", "http://localhost:8082"),
        SuperAdminUser:   getEnv("SUPER_ADMIN_USERNAME", "admin"),
        SuperAdminPass:   getEnv("SUPER_ADMIN_PASSWORD", "admin123"),
        DummyPlatformURL: getEnv("DUMMY_PLATFORM_URL", "http://localhost:8080"),
    }
}

func getEnv(key, fallback string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return fallback
}
```

**為什麼用環境變數而非設定檔？**

- [12-Factor App](https://12factor.net/config) 建議：設定應該存在環境中，而非程式碼裡
- Docker/Kubernetes 原生支援環境變數注入
- 不同環境（開發、測試、正式）只需要切換環境變數，不需要改程式碼

## 資料庫連線（mongodb/、redis/）

```go
// 檔案：admin-backend/server/internal/infrastructure/mongodb/client.go

package mongodb

import (
    "context"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
    client, err := mongo.Connect(options.Client().ApplyURI(uri))
    if err != nil {
        return nil, err
    }

    // Ping 確認連線正常
    if err := client.Ping(ctx, nil); err != nil {
        return nil, err
    }

    return client, nil
}
```

注意：連線邏輯是「基礎設施」，但 **使用** 連線的是 Adapter 層的 Repository。

## 路由設定（server/）

```go
// 檔案：admin-backend/server/internal/infrastructure/server/router.go

func NewRouter(c *di.Container) *gin.Engine {
    r := gin.Default()

    // CORS
    r.Use(cors.New(cors.Config{
        AllowOrigins: []string{"*"},
        AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders: []string{"Authorization", "Content-Type"},
    }))

    // 公開路由
    admin := r.Group("/api/admin/v1")
    admin.POST("/auth/login", c.AuthHandler.Login)

    // 受保護路由（需要登入）
    protected := admin.Group("",
        middleware.JWTAuth(c.JWTSecret),
        middleware.ReadOnly(),
        middleware.AuditLog(c.AuditLogCreateUC),
    )

    // 遊戲管理路由（需要權限）
    games := protected.Group("/games",
        middleware.RequirePermission(c.PermGroupRepo, entity.ModuleGames, entity.ActionView),
    )
    games.GET("", c.GameHandler.List)
    games.POST("", c.GameHandler.Create)
    games.PUT("/:gameId", c.GameHandler.Update)
    games.DELETE("/:gameId", c.GameHandler.Delete)

    // ... 更多路由
    return r
}
```

路由設定集中在一個地方，清楚呈現：
- 哪些路由是公開的、哪些需要認證
- 每組路由需要什麼權限
- Middleware 的套用順序

## Main Entry Point（cmd/server/main.go）

main.go 是整個服務的「組裝點」——把所有層串起來：

```go
// 簡化版的 main.go 流程

func main() {
    // 1. 載入設定
    cfg := config.Load()

    // 2. 建立基礎設施連線
    mongoClient, _ := mongodb.Connect(context.Background(), cfg.MongoURI)
    db := mongoClient.Database(cfg.MongoDB)

    redisClient := redis.NewClient(cfg.RedisAddr)

    // 3. 建立依賴注入容器（組裝所有層）
    container := di.NewContainer(db, redisClient, cfg)

    // 4. 建立路由
    router := server.NewRouter(container)

    // 5. 啟動伺服器
    srv := &http.Server{
        Addr:    ":" + cfg.Port,
        Handler: router,
    }
    srv.ListenAndServe()

    // 6. 優雅關閉（Graceful Shutdown）
    // ... 處理 OS Signal，等待進行中的請求完成
}
```

## 依賴注入容器（di/）

> 詳見 [08-依賴注入.md](08-依賴注入.md)

## 給 Junior 的學習重點

### 你需要理解的概念

1. **12-Factor App**：現代雲原生應用的設計原則，特別是 Config、Port Binding、Logs 這幾項
2. **Graceful Shutdown**：伺服器收到關閉信號時，不是直接斷線，而是等進行中的請求完成
3. **Connection Pooling**：MongoDB Driver 和 Redis Client 內部都有連線池管理

### 動手練習

1. 閱讀 `admin-backend/server/cmd/server/main.go`，畫出啟動流程圖
2. 看 `router.go`，列出所有路由及其需要的 middleware
3. 修改一個環境變數的預設值，觀察對系統的影響
