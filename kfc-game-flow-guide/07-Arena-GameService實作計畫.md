# 07 — Arena 賽事遊戲：GameService 實作計畫

## 實作範圍

僅涵蓋 **GameService**（gRPC，下注流程）與 **GameAPI**（HTTP，通知接收 + 結算 + 推播）。
arena-service 由另一位工程師負責，不在本文件範圍內。

---

## 實作階段總覽

```
Phase 1: Domain Layer          ← Entity、Value Object、Port 定義
    ↓
Phase 2: UseCase Layer         ← 下注、結算、取消、通知處理
    ↓
Phase 3: Adapter Layer         ← gRPC Server、HTTP Handler、Repository、Gateway
    ↓
Phase 4: Connector 整合        ← WebSocket Handler + Routes
    ↓
Phase 5: 推播整合              ← Kafka → PushProducer → Connector → WS
    ↓
Phase 6: 測試                  ← 單元測試 + 整合測試
```

---

## Phase 1：Domain Layer

**目標**：定義 Arena 相關的 Entity、Value Object。

### 新增檔案

```
gameservice/internal/domain/entity/
├── arena_bet.go              # ArenaBet Entity
├── arena_bet_side.go         # BetSide value object (A/B/DRAW)
└── arena_events.go           # Arena 領域事件定義
```

### ArenaBet Entity

```go
// arena_bet.go
type ArenaBet struct {
    BetID         string
    RoundID       string      // matchCode
    TableID       string
    UserID        string
    PlayerName    string
    GameID        string      // "arena"
    IID           string      // 平台商 ID
    Currency      string
    BetSide       BetSide     // A / B / DRAW
    BetAmount     float64
    BetAmountUSD  float64
    ExchangeRate  float64
    OddsAtBet     float64     // 下注時即時賠率（參考用）
    OddsAtClose   float64     // 封盤賠率（結算時填入）
    WinAmount     float64
    Status        BetStatus   // 複用現有 BetStatus
    DebitTxID     string
    CreditTxID    string
    RefundTxID    string
    RefundReason  string
    CreatedAt     time.Time
    SettledAt     *time.Time
    DistributedAt *time.Time
    RefundedAt    *time.Time
}
```

### BetSide Value Object

```go
// arena_bet_side.go
type BetSide string

const (
    BetSideA    BetSide = "A"
    BetSideB    BetSide = "B"
    BetSideDraw BetSide = "DRAW"
)

func (s BetSide) IsValid() bool {
    return s == BetSideA || s == BetSideB || s == BetSideDraw
}
```

### 驗收標準

- [ ] 所有 Entity 無 json/bson tag
- [ ] BetSide.IsValid() 驗證邏輯正確
- [ ] 複用現有 BetStatus（Confirmed/Settled/Distributed/Refunded）
- [ ] `go build` 通過

---

## Phase 2：UseCase Layer

**目標**：實作三個 UseCase + 定義 Ports。

### UseCase 1：`arena_place_bet`（GameService — 下注）

```
gameservice/internal/usecase/arena_place_bet/
├── contract.go
├── uc.go
└── ports/
    ├── bet_repository.go
    └── player_verifier.go
```

**Contract**：

```go
type UseCase interface {
    Execute(ctx context.Context, cmd *Command) (*Result, error)
}

type Command struct {
    TableID     string
    RoundID     string      // matchCode
    BetSide     entity.BetSide
    BetAmount   float64
    Currency    string
    IID         string
    PlayerToken string
}

type Result struct {
    BetID   string
    RoundID string
    BetSide entity.BetSide
    Amount  float64
    Balance float64
}
```

**核心邏輯（uc.go）**：

```
1. verify_endpoint 驗證玩家
2. ConfigCache 取得 Wallet endpoints + 押注上限
3. 驗證：單注 ≤ MaxBetPerTicket
4. 查詢 MongoDB：該玩家該場累計 → 驗證 ≤ MaxBetPerPlayer
5. 查詢 MongoDB：該場總投注 → 驗證 ≤ MaxBetPerMatch
6. Wallet Debit
7. MongoDB Insert ArenaBet (status=Confirmed)
   └── 失敗 → Cancel Debit
8. Kafka Publish stream_place_bets（非同步，失敗走 Outbox）
9. 回傳結果
```

**Ports**：

```go
// ports/bet_repository.go
type ArenaBetRepository interface {
    Save(ctx context.Context, bet *entity.ArenaBet) error
    FindByRoundID(ctx context.Context, roundID string) ([]*entity.ArenaBet, error)
    FindConfirmedByRoundID(ctx context.Context, roundID string) ([]*entity.ArenaBet, error)
    SumByUserAndRound(ctx context.Context, userID, roundID string) (float64, error)
    SumByRound(ctx context.Context, roundID string) (float64, error)
    UpdateSettlement(ctx context.Context, betID string, status entity.BetStatus, winAmount, oddsAtClose float64, creditTxID string) error
    UpdateRefund(ctx context.Context, betID string, refundTxID, reason string) error
}

// ports/player_verifier.go
type PlayerVerifier interface {
    Verify(ctx context.Context, endpoint, playerToken string) (*PlayerInfo, error)
}

type PlayerInfo struct {
    UserID     string
    PlayerName string
    Valid      bool
}
```

### UseCase 2：`arena_settle`（GameAPI — 結算派彩）

```
gameservice/internal/usecase/arena_settle/
├── contract.go
├── uc.go
└── ports/
    └── bet_repository.go   // 複用 arena_place_bet 的 Port
```

**Contract**：

```go
type UseCase interface {
    Execute(ctx context.Context, cmd *Command) (*Result, error)
}

type Command struct {
    RoundID         string
    Winner          string            // "A" / "B" / "DRAW"
    WinReason       string
    SettlementOdds  SettlementOdds
}

type SettlementOdds struct {
    A    float64
    Draw float64
    B    float64
}

type Result struct {
    TotalBets    int
    WinnersCount int
    LosersCount  int
    TotalPayout  float64
    TotalBet     float64
    GGR          float64
    ActualRTP    float64
}
```

**核心邏輯（uc.go）**：

```
1. 查詢所有 Confirmed ArenaBet（by roundID）
2. 遍歷每筆 Bet：
   ├── betSide == winner：
   │   payout = betAmount × settlementOdds[side]
   │   → Wallet Credit（獨立 context）
   │   → 成功：Update status=Distributed, winAmount, creditTxId
   │   → 失敗：寫入 Outbox
   └── betSide != winner：
       → Update status=Settled, winAmount=0
3. 每筆 Credit 成功後 → Kafka Publish stream_credit
4. 計算財務指標（GGR, actualRTP）
5. 推播結果（非同步）
```

### UseCase 3：`arena_cancel`（GameAPI — 取消退款）

```
gameservice/internal/usecase/arena_cancel/
├── contract.go
├── uc.go
└── ports/
    └── bet_repository.go   // 複用
```

**Contract**：

```go
type UseCase interface {
    Execute(ctx context.Context, cmd *Command) (*Result, error)
}

type Command struct {
    RoundID      string
    Reason       string    // PRE_MATCH_CANCEL / MID_MATCH_SUSPENSION / SYSTEM_ERROR
    CancelReason string    // 人類可讀原因
}

type Result struct {
    TotalRefunded int
    TotalAmount   float64
}
```

**核心邏輯（uc.go）**：

```
1. 查詢所有 Confirmed ArenaBet（by roundID）
2. 逐筆 Wallet Cancel（獨立 context）
   → 成功：Update status=Refunded, refundTxId, refundReason
   → 失敗：寫入 Outbox
3. 每筆 Cancel 成功後 → Kafka Publish stream_bet_cancel
4. 推播退款通知（非同步）
```

---

## Phase 3：Adapter Layer

### 新增檔案

```
gameservice/
├── internal/adapter/
│   ├── in/grpc/
│   │   └── arena_server.go                  # gRPC PlaceArenaBet 實作
│   ├── out/repository/mongo/
│   │   ├── arena_bet_repo.go                # ArenaBet MongoDB Repository
│   │   └── arena_bet_document.go            # ArenaBet bson Document
│   └── out/gateway/
│       └── player_verifier.go               # 平台 verify 實作（如未複用現有）

gameapi/
├── internal/adapter/
│   ├── in/http/
│   │   └── arena_notify_handler.go          # POST /api/v1/game/arena/notify
│   └── out/messaging/
│       └── arena_push_publisher.go          # 推播事件發布

共用：
├── internal/infrastructure/
│   ├── di/container.go                      # 新增 Arena 相關依賴注入
│   └── mongodb/migration/migrations/
│       └── 00X_arena_bets_indexes.go        # arena_bets indexes
```

### MongoDB Collection：`arena_bets`

```go
// arena_bet_document.go
type arenaBetDocument struct {
    BetID         string     `bson:"_id"`
    RoundID       string     `bson:"round_id"`
    TableID       string     `bson:"table_id"`
    UserID        string     `bson:"user_id"`
    PlayerName    string     `bson:"player_name"`
    GameID        string     `bson:"game_id"`
    IID           string     `bson:"iid"`
    Currency      string     `bson:"currency"`
    BetSide       string     `bson:"bet_side"`
    BetAmount     float64    `bson:"bet_amount"`
    BetAmountUSD  float64    `bson:"bet_amount_usd"`
    ExchangeRate  float64    `bson:"exchange_rate"`
    OddsAtBet     float64    `bson:"odds_at_bet"`
    OddsAtClose   float64    `bson:"odds_at_close"`
    WinAmount     float64    `bson:"win_amount"`
    Status        string     `bson:"status"`
    DebitTxID     string     `bson:"debit_tx_id"`
    CreditTxID    string     `bson:"credit_tx_id,omitempty"`
    RefundTxID    string     `bson:"refund_tx_id,omitempty"`
    RefundReason  string     `bson:"refund_reason,omitempty"`
    CreatedAt     time.Time  `bson:"created_at"`
    SettledAt     *time.Time `bson:"settled_at,omitempty"`
    DistributedAt *time.Time `bson:"distributed_at,omitempty"`
    RefundedAt    *time.Time `bson:"refunded_at,omitempty"`
}
```

**Indexes**：

| Index | 用途 |
|-------|------|
| `{round_id: 1, status: 1}` | 結算 / 取消時查詢 |
| `{round_id: 1, user_id: 1}` | 查詢玩家在該場的累計投注 |
| `{user_id: 1, created_at: -1}` | 玩家投注歷史 |
| `{round_id: 1, created_at: -1}` | 賽事投注列表 |

### GameAPI HTTP Handler

```go
// arena_notify_handler.go
func (h *ArenaNotifyHandler) HandleNotify(c *gin.Context) {
    var req ArenaNotifyRequest
    // parse request...

    switch req.Event {
    case "BETTING_OPEN":
        h.handleBettingOpen(c, req)
    case "BETTING_CLOSED":
        h.handleBettingClosed(c, req)
    case "MATCH_RESULT":
        h.handleMatchResult(c, req)     // → arena_settle UseCase
    case "MATCH_CANCELLED":
        h.handleMatchCancelled(c, req)  // → arena_cancel UseCase
    case "TABLE_OPEN":
        h.handleTableOpen(c, req)
    case "TABLE_CLOSED":
        h.handleTableClosed(c, req)
    default:
        c.JSON(400, errorResponse("INVALID_EVENT"))
    }
}
```

### DI Container 擴展

```go
// container.go 新增
type Container struct {
    // ... 現有欄位 ...

    // Arena UseCases
    ArenaPlaceBetUC  arena_place_bet.UseCase
    ArenaSettleUC    arena_settle.UseCase
    ArenaCancelUC    arena_cancel.UseCase
}
```

---

## Phase 4：Connector 整合

### 新增檔案

```
connector/internal/
├── adapter/in/websocket/
│   ├── codec/packet.go         # 新增 RouteArena* 常量
│   └── handler/
│       └── handler_arena.go    # Arena WebSocket Handler
│
├── adapter/out/grpc/
│   └── arena_service_client.go # gRPC GameService Arena Client
│
└── domain/port/
    └── arena_service_client.go # Arena gRPC Client Port
```

### 路由常量

| Route 常量 | 路由字串 | 方向 | 說明 |
|-----------|---------|------|------|
| `RouteArenaBetPlace` | `arena.bet.place` | Request | 玩家下注 |
| `RouteArenaRoundState` | `arena.round.state` | Push | 賽事狀態變更推播 |
| `RouteArenaTableState` | `arena.table.state` | Push | 桌台狀態推播 |
| `RouteArenaBetResult` | `arena.bet.result` | Push | 個人結算/退款結果推播 |

### Handler 註冊

```go
func (h *Handler) RegisterRoutes(router *ws.Router) {
    // ... 現有路由 ...
    router.Handle(codec.RouteArenaBetPlace, h.HandleArenaBetPlace)
}
```

---

## Phase 5：推播整合

### 推播事件與路由

| WS Route | 觸發 Game-API 事件 | 推播方式 | 內容 |
|---------|-------------------|---------|------|
| `arena.round.state` | BETTING_OPEN / BETTING_CLOSED / MATCH_RESULT / MATCH_CANCELLED | Broadcast | 賽事狀態 + 賠率 |
| `arena.table.state` | TABLE_OPEN / TABLE_CLOSED | Broadcast | 桌台狀態 |
| `arena.bet.result` | MATCH_RESULT / MATCH_CANCELLED | SendToUser | 個人結算/退款結果 |

### 推播資料流

```
GameAPI 收到通知
  → 處理完業務邏輯（結算/退款）
  → Kafka topic（arena-push-events）
  → PushProducer Consumer
  → PushProducer.Broadcast / SendToUser (gRPC)
  → Kafka ws-event-push
  → Connector Kafka Consumer
  → WebSocket Push to clients
```

### 賠率更新推播（由 arena-service 直接驅動）

```
arena-service CronJob（每 5 秒）
  → 比對 Redis 賠率快照
  → 有變動 → push-producer gRPC
  → PushProducer → Connector → WS Broadcast
```

> 賠率推播不經過 GameAPI，由 arena-service 的 CronJob 直接呼叫 push-producer。

---

## Phase 6：測試策略

### UseCase 單元測試（P0）

| UseCase | 測試重點 |
|---------|---------|
| `arena_place_bet` | 正常下注、各種押注上限超過、Verify 失敗、Debit 失敗、MongoDB 失敗回滾 |
| `arena_settle` | 正常結算（贏家 Credit + 輸家 Settled）、Credit 失敗走 Outbox、空投注列表、重複通知冪等 |
| `arena_cancel` | 正常退款、Cancel 失敗走 Outbox、空投注列表、重複通知冪等 |

### 整合測試（P1）

| 測試場景 | 涵蓋範圍 |
|---------|---------|
| 完整生命週期 | PlaceBet × N → MATCH_RESULT → 所有 Bet 正確結算 |
| 取消退款 | PlaceBet × N → MATCH_CANCELLED → 所有 Bet Refunded |
| 併發下注 | 多 goroutine 同時 PlaceBet → 押注上限正確 |
| 結算補償 | Settle 中途 Credit 失敗 → Outbox → 重試成功 |
| 重複通知 | 同一 MATCH_RESULT 送兩次 → 第二次無副作用 |

### E2E 測試（P2）

使用 `Special_Game_Pipeline` docker-compose 環境：
1. dummyPlatform 提供 Wallet API
2. Connector → GameService → Wallet 下注鏈路
3. GameAPI → Wallet 結算/退款鏈路
4. 模擬 arena-service 發送 Game-API 通知

---

## 服務變更矩陣

| 服務 | 變更範圍 | 新增檔案（估） | 修改檔案（估） |
|------|---------|-------------|-------------|
| **gameservice** | Domain + UseCase(place_bet) + Adapter(gRPC + Repo) + DI | ~10 | ~3 |
| **gameapi** | UseCase(settle + cancel) + Adapter(HTTP Handler + Push) + DI | ~8 | ~3 |
| **connector** | WS Handler + Routes + gRPC Client | ~4 | ~3 |

---

## 檔案結構樹（完整）

```
gameservice/
├── internal/
│   ├── domain/entity/
│   │   ├── bet_status.go            (現有，Arena 複用)
│   │   ├── arena_bet.go            🆕
│   │   ├── arena_bet_side.go       🆕
│   │   └── arena_events.go         🆕
│   │
│   ├── usecase/
│   │   ├── spin/                    (現有)
│   │   ├── spin_v2/                 (現有)
│   │   └── arena_place_bet/        🆕
│   │       ├── contract.go
│   │       ├── uc.go
│   │       └── ports/
│   │           ├── bet_repository.go
│   │           └── player_verifier.go
│   │
│   ├── adapter/
│   │   ├── in/grpc/
│   │   │   ├── server.go            (現有)
│   │   │   └── arena_server.go     🆕
│   │   └── out/
│   │       ├── repository/mongo/
│   │       │   ├── arena_bet_repo.go      🆕
│   │       │   └── arena_bet_document.go  🆕
│   │       └── gateway/
│   │           └── player_verifier.go     🆕 (或複用現有)
│   │
│   └── infrastructure/
│       ├── di/container.go          📝 修改
│       └── mongodb/migration/
│           └── 00X_arena_bets_indexes.go  🆕

gameapi/
├── internal/
│   ├── usecase/
│   │   ├── arena_settle/           🆕
│   │   │   ├── contract.go
│   │   │   └── uc.go
│   │   └── arena_cancel/           🆕
│   │       ├── contract.go
│   │       └── uc.go
│   │
│   ├── adapter/
│   │   ├── in/http/
│   │   │   └── arena_notify_handler.go    🆕
│   │   └── out/messaging/
│   │       └── arena_push_publisher.go    🆕
│   │
│   └── infrastructure/
│       └── di/container.go          📝 修改

connector/
├── internal/
│   ├── adapter/in/websocket/
│   │   ├── codec/packet.go          📝 修改（新增路由常量）
│   │   └── handler/
│   │       ├── handler.go           📝 修改（註冊 Arena 路由）
│   │       └── handler_arena.go    🆕
│   ├── adapter/out/grpc/
│   │   └── arena_service_client.go 🆕
│   └── domain/port/
│       └── arena_service_client.go 🆕
```

---

## 命名慣例

| 項目 | 慣例 | 範例 |
|------|------|------|
| 遊戲 ID | `arena` | — |
| WS Route 前綴 | `arena.` | `arena.bet.place` |
| Kafka Topic | `stream_` + 用途 | `stream_place_bets`, `stream_credit`, `stream_bet_cancel` |
| MongoDB Collection | `arena_` + 用途 | `arena_bets` |
| UseCase 目錄 | `arena_` + 功能 | `arena_place_bet`, `arena_settle` |
| Tracing Span | `arena.` + 步驟 | `arena.placeBet.debitWallet` |
