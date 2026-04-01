# 03 — Arena 賽事遊戲：系統架構與 GameService 職責

## 遊戲概述

| 項目 | 規格 |
|------|------|
| 遊戲名稱 | Battle Arena |
| 遊戲類型 | 多人對戰賽事投注 |
| 投注選項 | A 隊 / B 隊 / 和局（Draw） |
| 結果來源 | 外部真實賽事信號（管理員輸入結果） |
| 賠率模型 | RTP-based 初始賠率 + 投注量動態調整 |
| 回合控制 | 管理員手動操作（開盤 / 封盤 / 結算 / 取消） |

---

## 系統架構全景圖

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Arena 賽事遊戲架構                                │
│                                                                         │
│  ┌──────────┐     ┌──────────────┐     ┌────────────────────────┐      │
│  │  Player   │     │  Connector   │     │    GameService         │      │
│  │ (Browser) │◀──▶│  (WS:8081)   │────▶│    (gRPC:50051)        │      │
│  └──────────┘     └──────────────┘     │                        │      │
│       │                                 │  職責：                 │      │
│       │                                 │  ・驗證玩家 (verify)    │      │
│       │                                 │  ・押注上限驗證          │      │
│       │                                 │  ・Wallet Debit (扣款)  │      │
│       │                                 │  ・投注紀錄持久化        │      │
│       │                                 └──────────┬─────────────┘      │
│       │                                            │                    │
│       │                                   Kafka: stream_place_bets          │
│       │                                            │                    │
│       │                                            ▼                    │
│       │  ┌──────────────────────────────────────────────────────────┐   │
│       │  │                  Arena Service                            │   │
│       │  │                                                          │   │
│       │  │  ┌──────────┐  ┌──────────────┐  ┌────────────────┐    │   │
│       │  │  │ CRUD API │  │odds-consumer │  │   CronJob      │    │   │
│       │  │  │ 賽事管理  │  │Kafka→Redis   │  │ 賠率推播       │    │   │
│       │  │  │ 桌台管理  │  │Lua 賠率計算   │  │ (每5秒比對)    │    │   │
│       │  │  │ 隊伍管理  │  └──────────────┘  └────────────────┘    │   │
│       │  │  └────┬─────┘                                            │   │
│       │  │       │ 狀態變更時                                        │   │
│       │  │       ▼                                                   │   │
│       │  │  Game-API 通知                                            │   │
│       │  └──────┬───────────────────────────────────────────────────┘   │
│       │         │ POST /api/v1/game/arena/notify                        │
│       │         ▼                                                       │
│       │  ┌──────────────────────────────────────────────────────────┐   │
│       │  │                    GameAPI                                │   │
│       │  │                                                          │   │
│       │  │  職責：                                                   │   │
│       │  │  ・接收 arena-service 的 Game-API 通知                    │   │
│       │  │  ・MATCH_RESULT → 查詢投注 → 計算派彩 → Wallet Credit     │   │
│       │  │  ・MATCH_CANCELLED → 查詢投注 → Wallet Cancel (全額退款)  │   │
│       │  │  ・狀態變更 → 推播給前端玩家                               │   │
│       │  └──────────────────────────────────────────────────────────┘   │
│       │                                                                 │
│  ┌────┴───────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐    │
│  │ Platform   │  │ MongoDB  │  │  Redis   │  │  Admin Backend   │    │
│  │ (Wallet)   │  │ (持久化)  │  │ (賠率)   │  │  (認證代理層)     │    │
│  └────────────┘  └──────────┘  └──────────┘  └──────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 服務職責分工（明確邊界）

### arena-service（另一位工程師負責）

| 職責 | 說明 |
|------|------|
| 賽事生命週期管理 | 建立、開盤、封盤、結算、取消 |
| 桌台管理 | 桌台 CRUD、狀態切換 |
| 隊伍管理 | 隊伍 CRUD、戰績統計 |
| 賠率引擎 | 初始賠率計算（RTP-based）、動態賠率調整（Redis Lua Script） |
| odds-consumer | 消費 Kafka `debit_records` → 更新 Redis 投注統計 + 賠率 |
| CronJob 推播 | 每 5 秒比對賠率快照 → 變動時呼叫 push-producer |
| Game-API 通知 | 狀態變更時主動 POST 通知 GameAPI |
| Query API | 前台查詢（桌台、賽事、隊伍、投注紀錄） |

### GameService（我們負責）

| 職責 | 說明 |
|------|------|
| 玩家身份驗證 | 呼叫平台 `verify_endpoint` 確認 Token 有效 |
| 押注上限驗證 | 從 ConfigCache 讀取 `ArenaBetLimitConfig`，驗證單注/單玩家/單場上限 |
| Wallet Debit | 下注時呼叫平台 `debit_endpoint` 扣款 |
| 投注紀錄持久化 | 儲存 ArenaBet 到 MongoDB |
| Kafka 發布 | 下注成功後發布到 `stream_place_bets` topic，供 arena-service odds-consumer 消費 |

### GameAPI（我們負責）

| 職責 | 說明 |
|------|------|
| 接收 Game-API 通知 | POST `/api/v1/game/arena/notify` |
| 結算派彩 | 收到 MATCH_RESULT → 查詢投注 → 計算派彩 → Wallet Credit → 發布 `stream_credit` |
| 取消退款 | 收到 MATCH_CANCELLED → 查詢投注 → Wallet Cancel → 發布 `stream_bet_cancel` |
| 狀態推播 | 所有通知事件整理後推播到前端（Kafka → PushProducer → Connector → WS） |

### admin-backend

| 職責 | 說明 |
|------|------|
| 認證代理 | JWT 認證後轉發請求至 arena-service |
| 押注上限 CRUD | 直接呼叫 config-service（不經 arena-service） |

---

## Match（賽事）狀態機

> 由 arena-service 管理，GameService/GameAPI 不直接操作狀態，僅接收通知。

```
待開賽 ──→ 下注中 ──→ 進行中 ──→ 已結算
(pending)  (betting)  (in_progress)  (settled)
              │          │
              └────┬─────┘
                   ▼
              已取消
             (cancelled)
```

| 狀態 | Game-API 事件 | GameAPI 動作 |
|------|--------------|-------------|
| pending → betting | `BETTING_OPEN` | 推播賽事資訊 + 開放下注 |
| betting → in_progress | `BETTING_CLOSED` | 推播封盤通知 + 鎖定賠率 |
| in_progress → settled | `MATCH_RESULT` | 計算派彩 → Wallet Credit → 推播結果 |
| 任何 → cancelled | `MATCH_CANCELLED` | 全額退款 → Wallet Cancel → 推播取消 |

### 桌台（Table）狀態

| 狀態 | Game-API 事件 | GameAPI 動作 |
|------|--------------|-------------|
| maintenance → open | `TABLE_OPEN` | 推播桌台開放 |
| open → maintenance | `TABLE_CLOSED` | 推播桌台關閉 |

---

## 資料流總覽

### 下注流程

```
Player → WS → Connector → GameService (gRPC)
                              │
                              ├── 1. verify_endpoint（驗證玩家）
                              ├── 2. ConfigCache（押注上限驗證）
                              ├── 3. debit_endpoint（扣款）
                              ├── 4. MongoDB（儲存 ArenaBet）
                              ├── 5. Kafka stream_place_bets（通知 arena-service）
                              └── 6. 回傳結果 → Connector → WS → Player
                                              │
                                              ▼
                              arena-service odds-consumer
                                              │
                                              ▼
                              Redis 更新賠率 → CronJob → 推播
```

### 結算流程

```
Admin → Admin Backend → Arena Service
                            │
                            ├── 更新 Match status = settled
                            └── POST Game-API: MATCH_RESULT
                                        │
                                        ▼
                                    GameAPI
                                        │
                                        ├── 1. 查詢該場所有 ArenaBet
                                        ├── 2. 計算派彩（betAmount × settlementOdds）
                                        ├── 3. credit_endpoint（贏家入帳）
                                        ├── 4. 更新 ArenaBet 狀態
                                        ├── 5. Kafka stream_credit（供 arena-service 統計 RTP）
                                        └── 6. 推播結果 → Player
```

### 取消流程

```
Admin → Admin Backend → Arena Service
                            │
                            └── POST Game-API: MATCH_CANCELLED
                                        │
                                        ▼
                                    GameAPI
                                        │
                                        ├── 1. 查詢該場所有 Confirmed ArenaBet
                                        ├── 2. cancel_endpoint（全額退款）
                                        ├── 3. 更新 ArenaBet 狀態 = Refunded
                                        ├── 4. Kafka stream_bet_cancel（修正累積押注數值）
                                        └── 5. 推播取消通知 → Player
```

---

## 與 Spin 遊戲的共用 / 差異對照

| 面向 | 單人 Spin | Arena 賽事 | 共用程度 |
|------|----------|-----------|---------|
| Wallet Debit | 下注時扣款 | 下注時扣款 | **完全複用** |
| Wallet Credit | Spin 完成時 | 賽事結算時 | **完全複用** |
| Wallet Cancel | 異常補償 | 異常補償 + 賽事取消 | **完全複用** |
| ConfigCache | 取 Integrator endpoints | 取 endpoints + 押注上限 | **擴展複用** |
| Outbox Pattern | Credit 失敗補償 | Credit 失敗補償 | **完全複用** |
| EventPublisher | bet.completed | 各種 arena 事件 | **介面複用** |
| 結果來源 | math-lib RNG | arena-service 通知 | **完全不同** |
| 賠率計算 | GameService 內部 | arena-service（Redis Lua） | **完全不同** |
| 回合管理 | 無 | arena-service + Admin | **完全不同** |
| 推播 | 僅回傳給個人 | 廣播 + 個人推播 | **PushProducer 複用** |
