# 04 — 功夫雞 (Kung Fu Chicken) 設計規格

## 遊戲概述

| 項目 | 規格 |
|------|------|
| 遊戲名稱 | Kung Fu Chicken（功夫雞） |
| 遊戲 ID | `kungfuchicken` |
| 遊戲類型 | 多人共享回合投注 |
| 遊戲類別 | 鬥雞 |
| 投注選項 | Wala（藍）/ Melo（紅）/ Draw（平手） |
| 結果來源 | 外部真實鬥雞賽事信號 |
| 賠率模型 | 動態 Parimutuel（依投注池分佈即時計算） |
| 回合控制 | 管理員手動操作（開盤 / 封盤 / 結算 / 取消） |

---

## 架構全景圖

### 與現有架構的差異對照

```
┌──────────────────────────────────────────────────────────────┐
│                     功夫雞架構新增/變更                         │
│                                                              │
│  ┌──────────────┐                  ┌────────────────────┐    │
│  │ Admin Backend │   新增           │   GameService      │    │
│  │              │   回合管理 API    │                    │    │
│  │ [KFC Round   │──────────────▶  │ [KFC UseCases]     │    │
│  │  Management] │   gRPC          │  ├ kfc_round_mgmt  │    │
│  └──────────────┘                  │  ├ kfc_place_bet   │    │
│                                    │  ├ kfc_settle      │    │
│  ┌──────────────┐                  │  └ kfc_odds        │    │
│  │  Connector   │   新增           │                    │    │
│  │              │   KFC WS 路由    │ [KFC Entities]     │    │
│  │ [KFC Handler]│──────────────▶  │  ├ Round           │    │
│  │  kfc.bet.*   │   gRPC          │  └ KFCBet          │    │
│  └──────────────┘                  └────────┬───────────┘    │
│                                             │                │
│  ┌────────┐  ┌────────┐  ┌────────┐       │                │
│  │ Redis  │  │MongoDB │  │ Kafka  │       │                │
│  │ 投注池  │  │回合/投注│  │事件推播 │◀──────┘                │
│  │ 計數器  │  │紀錄    │  │        │                        │
│  └────────┘  └────────┘  └────────┘                        │
│                                                              │
│  ✅ 完全複用：WalletClient、ConfigCache、Outbox、OTel         │
│  ✅ 介面複用：EventPublisher、PushProducer                    │
│  🆕 全新：Round Entity、KFCBet Entity、賠率引擎               │
└──────────────────────────────────────────────────────────────┘
```

---

## 回合 (Round) 生命週期

### Round 狀態機

```
Created ──▶ Open ──▶ Closed ──▶ Settled ──▶ Completed
              │         │
              └────┬────┘
                   ▼
              Cancelled
```

### 每個狀態的業務規則

| 狀態 | 允許的操作 | 業務規則 |
|------|-----------|---------|
| **Created** | OpenRound | 回合已建立但尚未對外開放，可做預設定（houseEdge 等） |
| **Open** | PlaceBet, CloseRound, CancelRound | 玩家可下注，每次下注後更新投注池和動態賠率 |
| **Closed** | SettleRound, CancelRound | 不再接受下注，等待外部賽事結果 |
| **Settled** | （系統自動 → Completed） | 結果已確定，系統正在逐筆派彩 |
| **Completed** | （終態） | 所有派彩完成，回合歸檔 |
| **Cancelled** | （終態） | 回合取消，所有已下注玩家全額退款 |

### 管理員操作 vs 系統操作

| 操作 | 觸發者 | 說明 |
|------|--------|------|
| CreateRound | Admin | 建立新回合 |
| OpenRound | Admin | 開盤（允許下注） |
| CloseRound | Admin | 封盤（禁止下注） |
| SettleRound | Admin | 提交賽事結果，啟動結算 |
| CancelRound | Admin | 取消回合（緊急情況） |
| → Completed | System | 所有派彩完成後自動轉換 |

---

## Parimutuel 賠率引擎

### 核心公式

```
                totalPool × (1 - houseEdge)
odds(outcome) = ─────────────────────────────
                      outcomePool
```

| 符號 | 意義 |
|------|------|
| `totalPool` | 所有選項的投注總額 |
| `outcomePool` | 特定選項（Wala/Melo/Draw）的投注總額 |
| `houseEdge` | 抽水比例（例：5% = 0.05） |
| `odds(outcome)` | 該選項的賠率（含本金） |

### 計算範例

假設 `houseEdge = 5%`：

| 選項 | 投注總額 | 占比 | 賠率計算 | 最終賠率 |
|------|---------|------|---------|---------|
| Wala | $6,000 | 60% | 10000 × 0.95 / 6000 | **1.583** |
| Melo | $3,000 | 30% | 10000 × 0.95 / 3000 | **3.167** |
| Draw | $1,000 | 10% | 10000 × 0.95 / 1000 | **9.500** |
| **合計** | **$10,000** | | | |

### 抽水比例 (House Edge) 設計

| 項目 | 設計 |
|------|------|
| 預設值 | 5%（可在 GameConfig 中設定） |
| 設定粒度 | Per IntegratorGame（不同平台商可有不同抽水） |
| 來源 | ConfigCache（IntegratorGameConfig.CustomRTP → `houseEdge = 1 - RTP`） |

### 動態賠率更新策略

```
每次 PlaceBet 成功後：
1. Redis HINCRBY 更新投注池（原子操作）
2. HGETALL 讀取最新投注池
3. 計算新賠率
4. 回傳給下注玩家（同步）
5. 廣播最新賠率（非同步，可節流）
```

**節流策略**：
- 每 2 秒最多廣播一次賠率更新
- 或每累積 10 筆下注後廣播一次
- 結算時使用最終池子重新計算精確賠率（不受推播頻率影響）

### 邊界情況處理

| 情況 | 處理方式 |
|------|---------|
| 某選項投注額 = 0 | 賠率顯示為 0（表示無人投注，不會有贏家） |
| 總投注額 = 0 | 無法計算賠率，顯示初始佔位值 |
| 極端分佈（99% 押同一邊） | 正常計算，賠率趨近 1.0（低回報） |
| 只有一個選項有人投注 | 若該選項贏，賠率 = 1 - houseEdge；其他選項無人投注不會觸發 |

---

## 結算邏輯

### 單一回合批次結算流程

```
Admin 提交結果 (result = "wala")
    │
    ├── 1. 更新 Round: status = Settled, result = "wala"
    │
    ├── 2. 從 Redis 讀取最終投注池
    │      HGETALL round:{id}:pool → {wala: 6000, melo: 3000, draw: 1000}
    │
    ├── 3. 計算最終賠率
    │      walaOdds = 10000 × 0.95 / 6000 = 1.583
    │
    ├── 4. 查詢所有 Bets (MongoDB)
    │
    ├── 5. 分類處理
    │      ├── 贏家 (option == result)：
    │      │   └── winAmount = betAmount × finalOdds
    │      │       → Wallet Credit → Bet status = Distributed
    │      │
    │      └── 輸家 (option != result)：
    │          └── winAmount = 0
    │              → Bet status = Settled
    │
    ├── 6. 所有 Bet 處理完成 → Round status = Completed
    │
    └── 7. 推播結果
           ├── Broadcast: kfc.round.state {status: "completed", result: "wala"}
           └── SendToUser: kfc.bet.result {betId, option, win, payout}
```

### 派彩金額計算

```
winAmount = betAmount × finalOdds
```

範例：玩家押 Wala $100，最終 walaOdds = 1.583

```
winAmount = 100 × 1.583 = $158.30
淨利 = 158.30 - 100 = $58.30
```

### Draw（平手）結果處理

Draw 與 Wala / Melo 處理方式完全相同：
- 押中 Draw 的玩家 → Credit `betAmount × drawOdds`
- 未押 Draw 的玩家 → winAmount = 0

### 全額退款場景（CancelRound）

| 觸發條件 | 處理 |
|---------|------|
| 管理員取消回合 | 所有已 Confirmed 的 Bet → Wallet Cancel → status = Refunded |
| 結算過程中系統故障 | Outbox Worker 重試；無法恢復則由管理員介入取消 |

---

## 異常處理

### 下注階段異常

| 異常 | 處理 |
|------|------|
| Debit 成功但 Round 已被封盤 | Cancel Debit 退款（Double Check 機制） |
| Debit 超時 | Cancel + Outbox（同 Spin V2） |
| Redis 投注池更新失敗 | Cancel Debit 退款（無法記錄投注） |
| MongoDB Bet 寫入失敗 | Cancel Debit + Redis HINCRBY 反向扣減 |

### 結算階段異常

| 異常 | 處理 |
|------|------|
| 個別 Credit 失敗 | 寫入 Outbox，Worker 重試；不影響其他 Bet 結算 |
| 批次結算中途服務重啟 | Round status = Settled 但 Completed 尚未設定 → 重啟後掃描 Settled 回合，繼續未完成的派彩 |
| 外部結果錯誤 | 管理員手動修正（需要額外的管理 API） |

### 結算補償 Worker

```
定期掃描：
  Round status == Settled AND settledAt > 5 分鐘前 AND status != Completed
  → 檢查是否有 Bet 尚未 Distributed/Settled
  → 重新執行派彩
```

---

## 與現有遊戲的關鍵差異總覽表

| 維度 | CashMachine / XiaoMali | Kung Fu Chicken |
|------|----------------------|-----------------|
| 遊戲 ID | `cashmachine` / `xiaomali` | `kungfuchicken` |
| 結果引擎 | math-lib RNG | 外部信號 |
| 賠率 | 固定 RTP 95%/94% | 動態 Parimutuel（~5% house edge） |
| gRPC Service | `GameService.Spin` | `KungFuChickenService.PlaceBet` |
| WS Route | `game.spin` | `kfc.bet.place` |
| 回合管理 | 無（每次 Spin 即一局） | Admin API 全生命週期管理 |
| 投注選項 | 無（自動 Spin） | Wala / Melo / Draw |
| Redis 用途 | SpinState 暫存（10min TTL） | 投注池計數器 + Round 快取 |
| Kafka Topics | `bet-records`, `bet-events` | `kfc-bet-records`, `kfc-round-events`, `kfc-odds-updates` |
| MongoDB Collections | `bet_records`, `users` | `kfc_rounds`, `kfc_bets` |
| 推播 | 僅個人回傳 | 廣播（賠率/狀態）+ 個人（結算結果） |
| 併發模型 | 每用戶 SpinLock | 多用戶同時下注（Redis 原子操作） |
