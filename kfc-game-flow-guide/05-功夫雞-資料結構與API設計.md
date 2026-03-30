# 05 — 功夫雞：資料結構與 API 設計

## Domain Entities

> 所有 Entity 遵循 Clean Architecture 原則：**禁止序列化標籤**（json/bson tag），序列化結構定義在 Adapter 層。

### Round Entity

**位置**：`gameservice/internal/domain/entity/round.go`

```go
// Round 代表一個多人共享的遊戲回合。
// 是功夫雞遊戲的 Aggregate Root。
type Round struct {
    RoundID     string
    GameID      string      // "kungfuchicken"
    Status      RoundStatus
    Result      *BetOption  // nil = 尚未開獎
    HouseEdge   float64     // 抽水比例，例 0.05 = 5%
    BetPool     BetPool     // 各選項投注總額
    TotalPool   float64     // 投注池總額
    BetCount    int         // 總下注筆數
    CreatedBy   string      // 建立者（Admin ID）
    OpenedAt    *time.Time
    ClosedAt    *time.Time
    SettledAt   *time.Time
    CompletedAt *time.Time
    CancelledAt *time.Time
    CreatedAt   time.Time
}

// BetPool 各投注選項的金額池。
type BetPool struct {
    Wala float64
    Melo float64
    Draw float64
}
```

### RoundStatus（Value Object）

**位置**：`gameservice/internal/domain/entity/round_status.go`

```go
type RoundStatus string

const (
    RoundStatusCreated   RoundStatus = "created"
    RoundStatusOpen      RoundStatus = "open"
    RoundStatusClosed    RoundStatus = "closed"
    RoundStatusSettled   RoundStatus = "settled"
    RoundStatusCompleted RoundStatus = "completed"
    RoundStatusCancelled RoundStatus = "cancelled"
)
```

### BetOption（Value Object）

**位置**：`gameservice/internal/domain/entity/bet_option.go`

```go
type BetOption string

const (
    BetOptionWala BetOption = "wala"
    BetOptionMelo BetOption = "melo"
    BetOptionDraw BetOption = "draw"
)

// ValidBetOptions 所有合法的投注選項。
var ValidBetOptions = []BetOption{BetOptionWala, BetOptionMelo, BetOptionDraw}

// IsValid 驗證投注選項是否合法。
func (o BetOption) IsValid() bool {
    for _, v := range ValidBetOptions {
        if o == v {
            return true
        }
    }
    return false
}
```

### KFCBet Entity

**位置**：`gameservice/internal/domain/entity/kfc_bet.go`

```go
// KFCBet 代表單一玩家在某回合中的一筆投注。
type KFCBet struct {
    BetID        string
    RoundID      string
    UserID       string
    GameID       string      // "kungfuchicken"
    IID          string      // 平台商 ID
    Currency     string
    Option       BetOption   // 投注選項：wala / melo / draw
    BetAmount    float64
    BetAmountUSD float64     // USD 等值
    ExchangeRate float64     // 下注當下匯率
    Odds         float64     // 下注時的即時賠率（參考用）
    FinalOdds    float64     // 結算時的最終賠率
    WinAmount    float64
    WinAmountUSD float64
    Status       BetStatus   // 複用現有 BetStatus（Confirmed/Settled/Distributed/Refunded）
    DebitTxID    string
    CreditTxID   string
    RefundTxID   string
    RefundReason string
    IsTest       bool        // 壓測標記
    CreatedAt    time.Time
    SettledAt    *time.Time
    DistributedAt *time.Time
    RefundedAt    *time.Time
}
```

### RoundOdds（計算結果 Value Object，非持久化）

```go
// RoundOdds 即時賠率計算結果。
type RoundOdds struct {
    WalaOdds  float64
    MeloOdds  float64
    DrawOdds  float64
    TotalPool float64
    WalaPool  float64
    MeloPool  float64
    DrawPool  float64
}
```

### Domain Events

**位置**：`gameservice/internal/domain/entity/kfc_events.go`

```go
// KFC 遊戲的領域事件。

type RoundStateChangedEvent struct {
    RoundID   string
    GameID    string
    OldStatus RoundStatus
    NewStatus RoundStatus
    Timestamp time.Time
}

type BetPlacedEvent struct {
    BetID    string
    RoundID  string
    UserID   string
    Option   BetOption
    Amount   float64
    Odds     float64
    Timestamp time.Time
}

type RoundSettledEvent struct {
    RoundID    string
    GameID     string
    Result     BetOption
    TotalPool  float64
    BetCount   int
    FinalOdds  RoundOdds
    Timestamp  time.Time
}

type BetResultEvent struct {
    BetID     string
    RoundID   string
    UserID    string
    Option    BetOption
    BetAmount float64
    WinAmount float64
    FinalOdds float64
    Won       bool
    Timestamp time.Time
}
```

---

## Ports（UseCase 依賴介面）

### Global Ports — `domain/port/`

```go
// RoundStateStore Redis 中的回合狀態快取。
type RoundStateStore interface {
    // InitPool 初始化投注池（所有選項歸零）。
    InitPool(ctx context.Context, roundID string) error
    // IncrementPool 原子增量投注池。
    IncrementPool(ctx context.Context, roundID string, option entity.BetOption, amount float64) error
    // GetPool 讀取當前投注池。
    GetPool(ctx context.Context, roundID string) (*entity.BetPool, error)
    // DeletePool 刪除投注池。
    DeletePool(ctx context.Context, roundID string) error
}
```

### UseCase-Level Ports

#### `usecase/kfc_round_mgmt/ports/`

```go
// RoundRepository 回合持久化。
type RoundRepository interface {
    Create(ctx context.Context, round *entity.Round) error
    FindByID(ctx context.Context, roundID string) (*entity.Round, error)
    UpdateStatus(ctx context.Context, roundID string, from, to entity.RoundStatus, updates map[string]interface{}) error
    FindByStatus(ctx context.Context, gameID string, status entity.RoundStatus) ([]*entity.Round, error)
}
```

#### `usecase/kfc_place_bet/ports/`

```go
// KFCBetRepository 投注紀錄持久化。
type KFCBetRepository interface {
    Save(ctx context.Context, bet *entity.KFCBet) error
    FindByRoundID(ctx context.Context, roundID string) ([]*entity.KFCBet, error)
    FindByUserAndRound(ctx context.Context, userID, roundID string) ([]*entity.KFCBet, error)
    ExistsByUserRoundOption(ctx context.Context, userID, roundID string, option entity.BetOption) (bool, error)
}
```

#### `usecase/kfc_settle/ports/`

```go
// KFCBetSettler 批次結算。
type KFCBetSettler interface {
    FindByRoundID(ctx context.Context, roundID string) ([]*entity.KFCBet, error)
    UpdateSettlement(ctx context.Context, betID string, status entity.BetStatus, finalOdds, winAmount float64, creditTxID string) error
    UpdateRefund(ctx context.Context, betID string, refundTxID, reason string) error
}
```

---

## gRPC API

### Proto 定義

**位置**：`proto/kungfuchicken.proto`

```protobuf
syntax = "proto3";
package kfc;
option go_package = "github.com/game-pipeline/proto/kfcpb";

// ══════════════════════════════════════════════════════════
//  KungFuChickenService — 功夫雞遊戲核心服務
// ══════════════════════════════════════════════════════════

service KungFuChickenService {
  // ── 回合管理（Admin 操作）──────────────────────────────
  rpc CreateRound(CreateRoundRequest) returns (RoundResponse);
  rpc OpenRound(RoundActionRequest) returns (RoundResponse);
  rpc CloseRound(RoundActionRequest) returns (RoundResponse);
  rpc SettleRound(SettleRoundRequest) returns (SettleRoundResponse);
  rpc CancelRound(RoundActionRequest) returns (RoundResponse);
  rpc GetRound(GetRoundRequest) returns (RoundResponse);

  // ── 玩家操作 ─────────────────────────────────────────
  rpc PlaceBet(PlaceBetRequest) returns (PlaceBetResponse);
  rpc GetOdds(GetOddsRequest) returns (OddsResponse);
  rpc GetRoundInfo(GetRoundInfoRequest) returns (RoundInfoResponse);
}

// ══════════════════════════════════════════════════════════
//  Messages
// ══════════════════════════════════════════════════════════

// ── 回合管理 ────────────────────────────────────────────

message CreateRoundRequest {
  string game_id = 1;          // "kungfuchicken"
  double house_edge = 2;       // 抽水比例，例 0.05
  string created_by = 3;       // Admin ID
}

message RoundActionRequest {
  string round_id = 1;
}

message SettleRoundRequest {
  string round_id = 1;
  string result = 2;           // "wala" / "melo" / "draw"
}

message GetRoundRequest {
  string round_id = 1;
}

message RoundResponse {
  string round_id = 1;
  string game_id = 2;
  string status = 3;
  string result = 4;           // 空字串 = 尚未開獎
  double house_edge = 5;
  double total_pool = 6;
  int32 bet_count = 7;
  OddsInfo odds = 8;
  string created_at = 9;       // RFC3339
  string opened_at = 10;
  string closed_at = 11;
  string settled_at = 12;
}

message SettleRoundResponse {
  RoundResponse round = 1;
  int32 winners_count = 2;
  int32 losers_count = 3;
  double total_payout = 4;
}

// ── 玩家操作 ────────────────────────────────────────────

message PlaceBetRequest {
  string round_id = 1;
  string user_id = 2;
  string game_id = 3;          // "kungfuchicken"
  string option = 4;           // "wala" / "melo" / "draw"
  double bet_amount = 5;
  string currency = 6;
  string iid = 7;              // 平台商 ID
  string player_token = 8;     // JWT Token
  bool is_bot = 9;             // 壓測標記
}

message PlaceBetResponse {
  string bet_id = 1;
  string round_id = 2;
  string option = 3;
  double bet_amount = 4;
  double odds = 5;             // 下注時的即時賠率
  double balance = 6;          // 扣款後餘額
  OddsInfo current_odds = 7;   // 最新全局賠率
}

message GetOddsRequest {
  string round_id = 1;
}

message OddsResponse {
  OddsInfo odds = 1;
}

message OddsInfo {
  double wala_odds = 1;
  double melo_odds = 2;
  double draw_odds = 3;
  double wala_pool = 4;
  double melo_pool = 5;
  double draw_pool = 6;
  double total_pool = 7;
}

message GetRoundInfoRequest {
  string round_id = 1;
  string game_id = 2;          // 可用 game_id 查當前 Open 的回合
}

message RoundInfoResponse {
  RoundResponse round = 1;
  OddsInfo odds = 2;
}
```

---

## WebSocket Routes（Connector）

### 新增路由定義

**位置**：`connector/internal/adapter/in/websocket/codec/packet.go`（新增常量）

| Route 常量 | 路由字串 | 方向 | 說明 |
|-----------|---------|------|------|
| `RouteKFCBetPlace` | `kfc.bet.place` | Request | 玩家下注 |
| `RouteKFCRoundInfo` | `kfc.round.info` | Request | 查詢回合資訊 + 賠率 |
| `RouteKFCOddsGet` | `kfc.odds.get` | Request | 查詢即時賠率 |
| `RouteKFCRoundState` | `kfc.round.state` | Push | 回合狀態變更推播 |
| `RouteKFCOddsUpdate` | `kfc.odds.update` | Push | 賠率更新推播 |
| `RouteKFCBetResult` | `kfc.bet.result` | Push | 個人結算結果推播 |

### Request Payloads

#### `kfc.bet.place`

```json
{
  "roundId": "round-uuid",
  "option": "wala",
  "betAmount": 100
}
```

Response：
```json
{
  "betId": "bet-uuid",
  "roundId": "round-uuid",
  "option": "wala",
  "betAmount": 100,
  "odds": 1.85,
  "balance": 9900,
  "currentOdds": {
    "walaOdds": 1.85,
    "meloOdds": 2.10,
    "drawOdds": 8.50,
    "totalPool": 5000
  }
}
```

#### `kfc.round.info`

```json
{
  "roundId": "round-uuid"
}
```

Response：
```json
{
  "roundId": "round-uuid",
  "gameId": "kungfuchicken",
  "status": "open",
  "odds": {
    "walaOdds": 1.85,
    "meloOdds": 2.10,
    "drawOdds": 8.50,
    "totalPool": 5000
  },
  "betCount": 42,
  "openedAt": "2026-03-30T10:00:00Z"
}
```

### Push Payloads

#### `kfc.round.state`（Broadcast）

```json
{
  "roundId": "round-uuid",
  "gameId": "kungfuchicken",
  "status": "closed",
  "previousStatus": "open",
  "timestamp": "2026-03-30T10:05:00Z"
}
```

#### `kfc.odds.update`（Broadcast）

```json
{
  "roundId": "round-uuid",
  "walaOdds": 1.78,
  "meloOdds": 2.25,
  "drawOdds": 9.10,
  "totalPool": 8500,
  "timestamp": "2026-03-30T10:03:15Z"
}
```

#### `kfc.bet.result`（SendToUser）

```json
{
  "betId": "bet-uuid",
  "roundId": "round-uuid",
  "option": "wala",
  "result": "wala",
  "betAmount": 100,
  "finalOdds": 1.583,
  "winAmount": 158.30,
  "won": true,
  "balance": 10058.30,
  "timestamp": "2026-03-30T10:10:00Z"
}
```

---

## Admin REST API（admin-backend）

### 新增端點

| Method | Path | 說明 | 權限 |
|--------|------|------|------|
| POST | `/api/admin/v1/kfc/rounds` | 建立回合 | `kfc_round:create` |
| PUT | `/api/admin/v1/kfc/rounds/:id/open` | 開盤 | `kfc_round:manage` |
| PUT | `/api/admin/v1/kfc/rounds/:id/close` | 封盤 | `kfc_round:manage` |
| PUT | `/api/admin/v1/kfc/rounds/:id/settle` | 結算（帶結果） | `kfc_round:manage` |
| PUT | `/api/admin/v1/kfc/rounds/:id/cancel` | 取消回合 | `kfc_round:manage` |
| GET | `/api/admin/v1/kfc/rounds/:id` | 查詢回合詳情 | `kfc_round:read` |
| GET | `/api/admin/v1/kfc/rounds` | 回合列表（分頁） | `kfc_round:read` |
| GET | `/api/admin/v1/kfc/rounds/:id/bets` | 查詢回合內所有下注 | `kfc_round:read` |

### Request/Response 範例

#### POST `/api/admin/v1/kfc/rounds`

Request：
```json
{
  "gameId": "kungfuchicken",
  "houseEdge": 0.05
}
```

Response：
```json
{
  "success": true,
  "data": {
    "roundId": "round-uuid",
    "gameId": "kungfuchicken",
    "status": "created",
    "houseEdge": 0.05,
    "createdAt": "2026-03-30T09:55:00Z"
  }
}
```

#### PUT `/api/admin/v1/kfc/rounds/:id/settle`

Request：
```json
{
  "result": "wala"
}
```

Response：
```json
{
  "success": true,
  "data": {
    "roundId": "round-uuid",
    "status": "completed",
    "result": "wala",
    "winnersCount": 15,
    "losersCount": 27,
    "totalPayout": 4750.00,
    "totalPool": 10000.00,
    "settledAt": "2026-03-30T10:10:00Z"
  }
}
```

---

## Kafka Topics

| Topic | Producer | Consumer | Payload | 用途 |
|-------|----------|----------|---------|------|
| `kfc-round-events` | GameService | Connector / Analytics | RoundStateChangedEvent | 回合狀態變更 |
| `kfc-odds-updates` | GameService | Connector (PushProducer) | RoundOdds | 賠率更新廣播 |
| `kfc-bet-records` | GameService | GameService (Consumer) | KFCBet | 投注紀錄非同步持久化 |
| `kfc-bet-results` | GameService | Connector (PushProducer) | BetResultEvent | 個人結算結果推播 |

---

## MongoDB Collections + Indexes

### `kfc_rounds` — 回合紀錄

```go
// adapter/out/repository/mongo/round_document.go
type roundDocument struct {
    RoundID     string    `bson:"_id"`
    GameID      string    `bson:"game_id"`
    Status      string    `bson:"status"`
    Result      *string   `bson:"result,omitempty"`
    HouseEdge   float64   `bson:"house_edge"`
    WalaPool    float64   `bson:"wala_pool"`
    MeloPool    float64   `bson:"melo_pool"`
    DrawPool    float64   `bson:"draw_pool"`
    TotalPool   float64   `bson:"total_pool"`
    BetCount    int       `bson:"bet_count"`
    CreatedBy   string    `bson:"created_by"`
    OpenedAt    *time.Time `bson:"opened_at,omitempty"`
    ClosedAt    *time.Time `bson:"closed_at,omitempty"`
    SettledAt   *time.Time `bson:"settled_at,omitempty"`
    CompletedAt *time.Time `bson:"completed_at,omitempty"`
    CancelledAt *time.Time `bson:"cancelled_at,omitempty"`
    CreatedAt   time.Time  `bson:"created_at"`
}
```

**Indexes**：

| Index | 用途 |
|-------|------|
| `{game_id: 1, status: 1}` | 查詢特定遊戲的 Active 回合 |
| `{status: 1, settled_at: 1}` | 結算補償 Worker 掃描 |
| `{created_at: -1}` | 回合列表分頁 |

### `kfc_bets` — 投注紀錄

```go
// adapter/out/repository/mongo/kfc_bet_document.go
type kfcBetDocument struct {
    BetID         string     `bson:"_id"`
    RoundID       string     `bson:"round_id"`
    UserID        string     `bson:"user_id"`
    GameID        string     `bson:"game_id"`
    IID           string     `bson:"iid"`
    Currency      string     `bson:"currency"`
    Option        string     `bson:"option"`
    BetAmount     float64    `bson:"bet_amount"`
    BetAmountUSD  float64    `bson:"bet_amount_usd"`
    ExchangeRate  float64    `bson:"exchange_rate"`
    Odds          float64    `bson:"odds"`
    FinalOdds     float64    `bson:"final_odds"`
    WinAmount     float64    `bson:"win_amount"`
    WinAmountUSD  float64    `bson:"win_amount_usd"`
    Status        string     `bson:"status"`
    DebitTxID     string     `bson:"debit_tx_id"`
    CreditTxID    string     `bson:"credit_tx_id,omitempty"`
    RefundTxID    string     `bson:"refund_tx_id,omitempty"`
    RefundReason  string     `bson:"refund_reason,omitempty"`
    IsTest        bool       `bson:"is_test"`
    CreatedAt     time.Time  `bson:"created_at"`
    SettledAt     *time.Time `bson:"settled_at,omitempty"`
    DistributedAt *time.Time `bson:"distributed_at,omitempty"`
    RefundedAt    *time.Time `bson:"refunded_at,omitempty"`
}
```

**Indexes**：

| Index | 用途 |
|-------|------|
| `{round_id: 1, user_id: 1, option: 1}` | 唯一性檢查（同一玩家同一回合同一選項） |
| `{round_id: 1, status: 1}` | 結算時查詢 |
| `{user_id: 1, created_at: -1}` | 玩家投注歷史 |
| `{round_id: 1, created_at: -1}` | 回合投注列表 |

---

## Redis Key 設計

| Key Pattern | Type | TTL | 用途 |
|-------------|------|-----|------|
| `kfc:round:{roundID}:pool` | Hash | 回合結束後刪除 | 投注池計數器（wala/melo/draw） |
| `kfc:round:{roundID}:status` | String | 回合結束後刪除 | Round 當前狀態快取 |
| `kfc:round:{roundID}:meta` | Hash | 回合結束後刪除 | Round 基本資訊快取（houseEdge 等） |

### Redis 操作

```
# 初始化投注池
HSET kfc:round:{id}:pool wala 0 melo 0 draw 0

# 原子增量（玩家下注 Wala $100）
HINCRBY kfc:round:{id}:pool wala 10000    ← 以分為單位避免浮點數

# 讀取投注池
HGETALL kfc:round:{id}:pool

# 回合結束後清理
DEL kfc:round:{id}:pool kfc:round:{id}:status kfc:round:{id}:meta
```

> **金額精度**：Redis 中以**最小幣別單位（分/cent）的整數**儲存，避免浮點數精度問題。例 $100.50 → 10050。
