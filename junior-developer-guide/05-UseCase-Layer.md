# 05 — UseCase Layer（應用層 / 業務邏輯層）

## 這一層是什麼？

UseCase Layer 負責 **編排業務邏輯**。它知道「建立一款遊戲需要做哪些步驟」，但不知道「怎麼連 MongoDB」或「HTTP request 長什麼樣子」。

## 位置

```
internal/usecase/{feature_name}/
├── contract.go    ← 介面定義 + 輸入/輸出結構
├── uc.go          ← 業務邏輯實作
└── ports/         ← 需要的外部能力（介面）
    └── *.go
```

## 三個核心檔案的職責

### 1. contract.go — 對外的契約

定義這個 UseCase「能做什麼」以及「需要什麼輸入、會給什麼輸出」。

```go
// 檔案：admin-backend/server/internal/usecase/game_mgmt/contract.go

package game_mgmt

import "context"

// UseCase 介面 — 其他層只認這個介面
type CreateUseCase interface {
    Execute(ctx context.Context, cmd *CreateCommand) (*CreateResult, error)
}

type ListUseCase interface {
    Execute(ctx context.Context, cmd *ListCommand) (*ListResult, error)
}

// Command — 輸入（呼叫者需要提供什麼）
type CreateCommand struct {
    Code       string
    NameI18n   map[string]string
    GameTypeID string
    Tags       []string
    RTP        float64
}

// Result — 輸出（呼叫者會拿到什麼）
type CreateResult struct {
    Game *entity.Game
}
```

**為什麼要有 Command/Result？**

- **Command** 把 UseCase 需要的參數打包成一個結構，避免參數列表無限增長
- **Result** 把回傳值打包成一個結構，未來要多回傳東西時不用改介面簽名
- 它們是 UseCase 層的「語言」，與 HTTP 的 Request/Response DTO 完全分離

### 2. ports/ — 需要的外部能力

定義 UseCase 需要哪些外部依賴，但 **只定義介面，不做實作**。

```go
// 檔案：admin-backend/server/internal/usecase/game_mgmt/ports/repository.go

package ports

import (
    "context"
    "admin-backend/server/internal/domain/entity"
)

// GameRepository — UseCase 只知道「我需要能存取遊戲資料的東西」
type GameRepository interface {
    Create(ctx context.Context, game *entity.Game) error
    FindByID(ctx context.Context, id string) (*entity.Game, error)
    FindByCode(ctx context.Context, code string) (*entity.Game, error)
    List(ctx context.Context, filter GameFilter) ([]*entity.Game, int64, error)
    Update(ctx context.Context, game *entity.Game) error
    Delete(ctx context.Context, id string) error
}

type GameFilter struct {
    Status     *string
    GameTypeID *string
    Page       int
    PageSize   int
}
```

**這就是「依賴反轉 (Dependency Inversion)」的精髓**：
- UseCase 定義「我需要什麼」（Port / 介面）
- Adapter 決定「怎麼提供」（Repository 實作）
- UseCase 永遠不知道實際用的是 MongoDB、PostgreSQL 還是記憶體

### 3. uc.go — 業務邏輯實作

```go
// 檔案：admin-backend/server/internal/usecase/game_mgmt/uc.go

package game_mgmt

import (
    "context"
    "errors"
    "time"

    "admin-backend/server/internal/domain/entity"
    "admin-backend/server/internal/usecase/game_mgmt/ports"
)

type createUseCase struct {
    gameRepo ports.GameRepository    // 依賴注入的介面
}

// 建構函式：接收介面，回傳介面
func NewCreateUseCase(gameRepo ports.GameRepository) CreateUseCase {
    return &createUseCase{gameRepo: gameRepo}
}

func (uc *createUseCase) Execute(ctx context.Context, cmd *CreateCommand) (*CreateResult, error) {
    // 1. 業務規則驗證
    existing, _ := uc.gameRepo.FindByCode(ctx, cmd.Code)
    if existing != nil {
        return nil, errors.New("game code already exists")
    }

    // 2. 建立 Domain Entity
    game := &entity.Game{
        Code:       cmd.Code,
        NameI18n:   cmd.NameI18n,
        GameTypeID: cmd.GameTypeID,
        Tags:       cmd.Tags,
        RTP:        cmd.RTP,
        Status:     entity.GameStatusActive,
        CreatedAt:  time.Now(),
        UpdatedAt:  time.Now(),
    }

    // 3. 透過 Port 存入資料庫
    if err := uc.gameRepo.Create(ctx, game); err != nil {
        return nil, err
    }

    // 4. 回傳結果
    return &CreateResult{Game: game}, nil
}
```

觀察重點：
- `createUseCase` struct 的欄位是 **介面** 型別（`ports.GameRepository`）
- `NewCreateUseCase` 接收介面、回傳介面 → 符合 Go 的慣例
- Execute 裡面只有 **業務邏輯**，沒有 HTTP 解析、沒有 BSON 操作

## 更複雜的 UseCase 範例：Auth Login

```go
// admin-backend/server/internal/usecase/auth/ 的概念

type loginUseCase struct {
    userRepo     ports.AdminUserRepository
    loginLimiter ports.LoginLimiter     // 登入失敗限流
    jwtSecret    string
    jwtExpiry    time.Duration
}

func (uc *loginUseCase) Execute(ctx context.Context, cmd *LoginCommand) (*LoginResult, error) {
    // 1. 檢查是否被限流（連續登入失敗太多次）
    if blocked := uc.loginLimiter.IsBlocked(ctx, cmd.IP); blocked {
        return nil, errors.New("too many failed attempts")
    }

    // 2. 查詢使用者
    user, err := uc.userRepo.FindByUsername(ctx, cmd.Username)
    if err != nil {
        uc.loginLimiter.RecordFailure(ctx, cmd.IP)
        return nil, errors.New("invalid credentials")
    }

    // 3. 驗證密碼
    if !verifyPassword(user.HashedPassword, cmd.Password) {
        uc.loginLimiter.RecordFailure(ctx, cmd.IP)
        return nil, errors.New("invalid credentials")
    }

    // 4. 產生 JWT Token
    token, err := generateJWT(user, uc.jwtSecret, uc.jwtExpiry)
    if err != nil {
        return nil, err
    }

    // 5. 清除失敗紀錄
    uc.loginLimiter.ClearFailures(ctx, cmd.IP)

    return &LoginResult{Token: token, User: user}, nil
}
```

注意這裡有 **多個 Port**：UserRepository 和 LoginLimiter。UseCase 可以組合多個外部能力來完成業務流程。

## UseCase 的設計準則

### 1. 單一職責
每個 UseCase 做一件事。「建立遊戲」和「列出遊戲」是兩個不同的 UseCase，即使它們都操作 Game。

### 2. 不要洩漏技術細節
UseCase 裡面不應該出現：
- `gin.Context`（HTTP 框架）
- `bson.M`（MongoDB）
- `kafka.Message`（訊息佇列）

### 3. Command 不是 DTO
Command 的欄位名稱用 **業務語言**，不是 API 語言：

```go
// ✅ 好的 Command
type CreateCommand struct {
    Code     string              // 業務概念：遊戲代碼
    NameI18n map[string]string   // 業務概念：多語系名稱
}

// ❌ 不好的 Command（混入了 API/DB 語言）
type CreateCommand struct {
    GameCode string `json:"game_code"`  // 不該有 json tag
    Names    bson.M                     // 不該有 bson 型別
}
```

## 給 Junior 的學習重點

### 你需要理解的概念

1. **依賴反轉 (Dependency Inversion Principle)**：高層模組不依賴低層模組，兩者都依賴抽象（介面）
2. **介面 (Interface) 的設計**：Go 的介面是隱式實作的，理解 `interface` 在 Go 中的使用方式
3. **Command Pattern**：把操作的輸入封裝成物件

### 動手練習

1. 選一個 UseCase（如 `currency_mgmt`），追蹤它的 `contract.go` → `ports/*.go` → `uc.go`
2. 找出對應的 Repository 實作在 `adapter/out/repository/mongo/` 的哪個檔案
3. 試著畫出 UseCase 的依賴關係圖
