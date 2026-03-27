# 02 — Clean Architecture 核心概念

## 什麼是 Clean Architecture？

Clean Architecture 是 Robert C. Martin（Uncle Bob）提出的軟體架構設計原則。核心思想只有一句話：

> **讓業務邏輯不依賴於任何外部細節（框架、資料庫、UI）。**

## 為什麼需要它？

想像一個常見的情境：

```
❌ 沒有分層的寫法
handler.go:
  func CreateGame(c *gin.Context) {
      var req struct { Name string `json:"name"` }
      c.BindJSON(&req)

      // 直接操作資料庫
      collection := mongoClient.Database("db").Collection("games")
      collection.InsertOne(c, bson.M{"name": req.Name})

      c.JSON(200, gin.H{"success": true})
  }
```

這樣寫有什麼問題？

1. **換框架就要重寫**：如果從 Gin 換成 Echo，業務邏輯全部耦合在 handler 裡面
2. **換資料庫就要重寫**：如果從 MongoDB 換成 PostgreSQL，每個 handler 都要改
3. **無法單獨測試業務邏輯**：測試必須啟動完整的 HTTP 伺服器和資料庫
4. **職責混亂**：一個函式同時處理 HTTP 解析、業務邏輯、資料存取

## Clean Architecture 的解法：四層同心圓

```
    ┌──────────────────────────────────────┐
    │          Infrastructure              │ ← 最外層：框架、DB Driver
    │    ┌────────────────────────────┐    │
    │    │         Adapter            │    │ ← 轉接層：Handler、Repository 實作
    │    │    ┌──────────────────┐    │    │
    │    │    │     UseCase      │    │    │ ← 業務邏輯（應用層）
    │    │    │   ┌──────────┐   │    │    │
    │    │    │   │  Domain  │   │    │    │ ← 最內層：核心實體
    │    │    │   └──────────┘   │    │    │
    │    │    └──────────────────┘    │    │
    │    └────────────────────────────┘    │
    └──────────────────────────────────────┘
```

### 依賴規則（最重要的一條規則）

> **依賴方向只能由外向內，內層永遠不知道外層的存在。**

| 層 | 可以依賴 | 不可以依賴 |
|----|---------|-----------|
| Domain | 無（只依賴 Go 標準庫） | UseCase、Adapter、Infrastructure |
| UseCase | Domain | Adapter、Infrastructure |
| Adapter | Domain、UseCase | Infrastructure（除了實作需要的套件） |
| Infrastructure | 全部 | — |

## 對應到本專案的程式碼

以 `admin-backend/server/` 為例：

```
internal/
├── domain/entity/          ← Domain Layer
│   ├── game.go             # Game 實體，零外部依賴
│   ├── admin_user.go       # 使用者實體
│   └── ...
│
├── usecase/                ← UseCase Layer
│   ├── game_mgmt/
│   │   ├── contract.go     # 定義介面（Input/Output）
│   │   ├── uc.go           # 業務邏輯實作
│   │   └── ports/          # 定義需要的外部能力（介面）
│   └── auth/
│       ├── contract.go
│       └── uc.go
│
├── adapter/                ← Adapter Layer
│   ├── in/http/            # 入站轉接器（接收請求）
│   │   ├── game_handler.go # HTTP Handler
│   │   └── dto/            # 資料傳輸物件
│   └── out/                # 出站轉接器（對外連線）
│       ├── repository/     # 資料庫操作實作
│       └── gateway/        # 呼叫外部 API
│
└── infrastructure/         ← Infrastructure Layer
    ├── config/             # 環境變數設定
    ├── di/                 # 依賴注入容器
    ├── mongodb/            # MongoDB 連線建立
    └── redis/              # Redis 連線建立
```

## 實際資料流

以「管理員建立一款新遊戲」為例：

```
1. HTTP Request 進來
   POST /api/admin/v1/games {"name": "Slot A", "rtp": 96.5}
              │
              ▼
2. Adapter Layer: game_handler.go
   - 解析 JSON → DTO (Request)
   - 轉換 DTO → UseCase Command
   - 呼叫 UseCase
              │
              ▼
3. UseCase Layer: game_mgmt/uc.go
   - 驗證業務規則（名稱不能重複等）
   - 建立 Domain Entity
   - 透過 Port（介面）呼叫 Repository
              │
              ▼
4. Adapter Layer: repository/mongo/game_repo.go
   - 將 Entity 轉換為 MongoDB Document（加上 bson tags）
   - 執行 InsertOne
   - 將結果轉回 Entity
              │
              ▼
5. 回到 UseCase → 回到 Handler → 回傳 HTTP Response
```

## 為什麼值得這麼做？

| 好處 | 說明 |
|------|------|
| **可測試性** | UseCase 只依賴介面，可以用 Mock 測試，不需要真的資料庫 |
| **可替換性** | 換資料庫只要寫新的 Repository 實作，業務邏輯完全不動 |
| **可理解性** | 每一層職責清楚，新人看 Domain 就知道有哪些核心概念 |
| **可維護性** | 修改 UI 不會影響業務邏輯，修改業務邏輯不會影響資料存取 |

## 常見的 Junior 疑問

### Q：這樣分層不會太多 boilerplate 嗎？
A：是的，初期會覺得「明明幾行就能寫完的東西為什麼要跨四個檔案」。但當專案成長到有 20+ 個功能、多人協作時，這些分層帶來的可維護性遠大於初期的開銷。

### Q：Domain Entity 和 DTO 長得幾乎一樣，為什麼不共用？
A：因為它們的變更原因不同。DTO 跟隨 API 規格變化，Entity 跟隨業務規則變化。今天 API 想多回傳一個欄位，不應該影響到 Entity。

### Q：我要怎麼判斷邏輯該放在哪一層？
A：問自己：**「如果今天換一個前端框架 / 換一個資料庫，這段邏輯還需要嗎？」**
- 需要 → 放 UseCase 或 Domain
- 不需要 → 放 Adapter 或 Infrastructure

## 延伸閱讀

- Robert C. Martin, *Clean Architecture* (書籍)
- [The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) (Uncle Bob 原文)
- Alistair Cockburn, *Hexagonal Architecture*（六邊形架構，概念相似）
