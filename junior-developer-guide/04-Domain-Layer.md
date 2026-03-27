# 04 — Domain Layer（領域層）

## 這一層是什麼？

Domain Layer 是整個架構中 **最核心、最穩定** 的一層。它定義了系統中的「核心概念」——也就是即使你換掉框架、換掉資料庫、換掉 API 協定，這些概念依然存在。

## 位置

```
internal/domain/entity/*.go
```

## 核心原則：零外部依賴

Domain Entity **不允許** 有任何外部 import（除了 Go 標準庫如 `time`）。

```go
// ✅ 正確：Domain Entity
package entity

import "time"

type Game struct {
    ID          string
    Code        string
    NameI18n    map[string]string    // 多語系名稱
    GameTypeID  string
    Tags        []string
    RTP         float64
    Status      GameStatus
    IconURL     string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type GameStatus string

const (
    GameStatusActive   GameStatus = "active"
    GameStatusInactive GameStatus = "inactive"
)
```

```go
// ❌ 錯誤：Domain Entity 不該有 json/bson 標籤
type Game struct {
    ID   string `json:"id" bson:"_id"`    // 不要這樣做！
    Name string `json:"name" bson:"name"` // 標籤屬於 Adapter 層
}
```

### 為什麼不能有標籤（tags）？

- `json:"..."` 是 HTTP 序列化的關注點 → 屬於 Adapter/in/http
- `bson:"..."` 是 MongoDB 序列化的關注點 → 屬於 Adapter/out/repository
- Entity 只關心「業務上這個物件有什麼屬性」，不關心怎麼存、怎麼傳

## 本專案的 Domain Entity 範例

### AdminUser（管理員）

```
檔案：admin-backend/server/internal/domain/entity/admin_user.go
```

```go
type Role string

const (
    RoleSuperAdmin    Role = "super_admin"
    RoleAdmin         Role = "admin"
    RolePlatformAdmin Role = "platform_admin"
    RoleReadOnly      Role = "readonly"
)

type AdminUser struct {
    ID                string
    Username          string
    HashedPassword    string
    Role              Role
    PlatformID        string
    PermissionGroupID string
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

觀察重點：
- `Role` 使用自訂型別 + 常數，而非普通的 string → **型別安全**
- `HashedPassword` 已經是 hash 過的，Entity 不負責 hash 邏輯
- 沒有任何序列化標籤

### PermissionGroup（權限群組）

```
檔案：admin-backend/server/internal/domain/entity/permission_group.go
```

```go
type Module string
type Action string

const (
    ModuleGames        Module = "games"
    ModuleCurrencies   Module = "currencies"
    ModuleIntegrators  Module = "integrators"
    // ...
)

const (
    ActionView   Action = "view"
    ActionCreate Action = "create"
    ActionEdit   Action = "edit"
    ActionDelete Action = "delete"
)

type PermissionGroup struct {
    ID          string
    Name        string
    Permissions map[Module][]Action   // 每個模組可執行的動作
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

觀察重點：
- 用 `map[Module][]Action` 表達「哪個模組可以做哪些操作」
- 這個結構直接體現了 RBAC 的業務概念
- Module 和 Action 都是具名型別，避免傳錯參數

### Currency（幣別）

```
檔案：admin-backend/server/internal/domain/entity/currency.go
```

```go
type Currency struct {
    ID            string
    Code          string     // "USD", "TWD", "JPY"
    ExchangeRate  float64    // 對 USD 的匯率
    DecimalPlaces int        // 小數位數（JPY=0, USD=2）
    Status        string
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

## Domain Layer 的設計思考

### 1. Entity 不應該包含業務方法嗎？

在經典的 DDD（Domain-Driven Design）中，Entity 可以包含業務方法。本專案採用的是比較「貧血模型」的做法——Entity 主要是資料載體，業務邏輯放在 UseCase 層。

兩種做法各有優劣：

| 做法 | 優點 | 缺點 |
|------|------|------|
| 貧血模型（本專案） | 結構簡單、容易理解 | 業務邏輯分散在 UseCase |
| 充血模型（DDD） | 業務邏輯內聚在 Entity | 需要更深的 DDD 功力 |

### 2. Value Object 在哪裡？

本專案中有些概念比較像 Value Object（值物件）：
- `GameStatus`、`Role`、`Module`、`Action` 這些常數定義
- `map[string]string` 表示的 I18n 名稱

嚴格的 DDD 會把這些抽成獨立的 Value Object 型別並加上驗證邏輯，但本專案保持了比較務實的做法。

## 給 Junior 的學習重點

### 你需要理解的概念

1. **Entity vs Value Object**：Entity 有唯一識別（ID），Value Object 沒有（用值本身來區分）
2. **依賴方向**：Domain 是最內層，不依賴任何其他層
3. **序列化分離**：資料怎麼存、怎麼傳，是外層的事

### 動手練習

1. 打開 `admin-backend/server/internal/domain/entity/` 目錄，瀏覽所有 Entity
2. 確認每個 Entity 都沒有 import 外部套件
3. 思考：如果要新增一個「遊戲供應商 (Game Provider)」Entity，你會定義哪些欄位？

### 延伸閱讀

- Eric Evans, *Domain-Driven Design*（藍皮書）
- Vaughn Vernon, *Implementing Domain-Driven Design*（紅皮書）
- Martin Fowler, [AnemicDomainModel](https://martinfowler.com/bliki/AnemicDomainModel.html)
