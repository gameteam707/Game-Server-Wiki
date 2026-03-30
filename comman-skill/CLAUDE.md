# CLAUDE.md — KFC 遊戲平台共通技能指南

> 本文件為 Claude Code 在整個 KFC 專案中工作時的共通參考。涵蓋新架構全貌、各服務職責、跨服務協作模式，以及開發慣例。

---

## 專案全貌

KFC 是一套線上遊戲平台後端系統。新架構從舊專案 (Backend-Core-Project) 的 **Go 單機 DDD 模式**，遷移為**依賴 K8s 部署於 GCP 的微服務架構**，以支持水平擴展。

### 舊架構 (Backend-Core-Project) — 已棄用，僅供參考

| 項目 | 說明 |
|------|------|
| 架構 | Go 單機 DDD (Domain-Driven Design) |
| 資料庫 | MySQL (Users/Platforms/Orders) + MongoDB (BetRecords) |
| 即時通訊 | Gorilla WebSocket (單機 Hub) |
| 數學引擎 | gRPC 呼叫外部 Math-Lab 服務 |
| 限制 | 單機架構無法水平擴展，WebSocket Hub 不支持多節點 |

### 新架構 — 微服務 + K8s

```
┌─────────────────────────────────────────────────────────────┐
│                        GCP / Kubernetes                      │
│                                                              │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────────┐   │
│  │ admin-backend│  │config-service│  │  sfc-stream-game  │   │
│  │  (Go + Vue) │  │    (Go)      │  │                   │   │
│  │  Port: 8090 │  │  Port: 8095  │  │ ┌───────────┐     │   │
│  └──────┬──────┘  └──────┬───────┘  │ │ connector │     │   │
│         │                │          │ │ (WS:8081) │     │   │
│         │   REST + SSE   │          │ └─────┬─────┘     │   │
│         └───────────────►│◄─────────│       │gRPC       │   │
│                          │          │ ┌─────▼─────┐     │   │
│                          │  SSE     │ │gameservice│     │   │
│                          ├─────────►│ │(gRPC:50051)│    │   │
│                          │          │ └───────────┘     │   │
│                          │          │ ┌───────────┐     │   │
│                          │          │ │  gameapi  │     │   │
│                          │          │ │(REST:8082)│     │   │
│                          │          │ └───────────┘     │   │
│                          │          │ ┌─────────────┐   │   │
│                          │          │ │push-producer│   │   │
│                          │          │ │(gRPC:50052) │   │   │
│                          │          │ └─────────────┘   │   │
│                          │          └───────────────────┘   │
│                                                              │
│  ┌──────────┐  ┌───────┐  ┌───────┐  ┌──────────────────┐  │
│  │ MongoDB  │  │ Redis │  │ Kafka │  │ Observability    │  │
│  │(資料持久化)│  │(快取)  │  │(事件流)│  │(OTel+Grafana)   │  │
│  └──────────┘  └───────┘  └───────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘

共用 Library:
  ├── math-lib          (RNG/機率引擎，作為 Go module 引入)
  └── go-observability  (OpenTelemetry 封裝，作為 Go module 引入)

本地開發/測試:
  └── Special_Game_Pipeline  (docker-compose 完整環境)

K8s 部署設定:
  └── kfc-k8s  (Kustomize base + overlays)
```

---

## 各服務職責

### 1. sfc-stream-game — 核心遊戲服務群

包含 4 個子服務，共用一個 Go module：

| 子服務 | 協議 | Port | 職責 |
|--------|------|------|------|
| **connector** | WebSocket | 8081 | 玩家即時連線閘道，自訂二進位協議，狀態機管理 |
| **gameservice** | gRPC | 50051 | 遊戲核心邏輯：扣款 → RNG → 派彩 → 事件發布 |
| **gameapi** | REST | 8082 | 遊戲查詢 API (報表、歷史記錄) |
| **push-producer** | gRPC | 50052 | 即時推播事件產生器 (Kafka WsEventPush) |

**關鍵設計：**
- Connector 為 **StatefulSet** (3 replicas)，支持 sessionAffinity
- GameService 為 **Deployment**，無狀態可水平擴展
- 透過 Redis SpinLock 防止併發 spin
- 透過 Kafka 解耦推播事件
- Config Service SSE 即時同步遊戲設定

### 2. admin-backend — 管理後台

| 元件 | 技術 | 職責 |
|------|------|------|
| server | Go + Gin | 管理 API：遊戲設定、玩家管理、報表、帳號權限 |
| web | Vue 3 + Vben Admin | 管理介面，支持 i18n、時區轉換、ECharts 報表 |

**與其他服務關係：** 匯率/平台端點設定 CRUD → 同步至 config-service → SSE 推送至 gameservice

### 3. config-service — 設定中心

- 提供 REST API 供 admin-backend 寫入、gameservice 全量同步
- 提供 SSE `/events/config` 供 gameservice 即時增量更新
- 管理三類設定：CurrencyConfig、IntegratorConfig、PlatformEndpointConfig

### 4. go-observability — 可觀測性共用模組

以 Go module 方式引入各服務，封裝 OpenTelemetry：

| 模組 | 用途 |
|------|------|
| `observability.Init()` | 初始化 OTel Tracer/Meter Providers |
| `logging.Init()` | 結構化日誌 + trace context 關聯 |
| `ginmw.Trace()` | Gin HTTP 自動 span |
| `grpcmw.ServerOption()` / `ClientOption()` | gRPC 自動 span |
| `mongotrace.NewMonitoredClientOptions()` | MongoDB 操作追蹤 |
| `redistrace.NewClient()` | Redis 操作追蹤 |
| `tracing.Start()` | 手動業務 span |

### 5. math-lib — RNG/機率引擎

- 純 Go library，以 Go module 引入 gameservice
- 提供 `engine.PlayRound()` 進行遊戲 RNG 計算
- 私有 repo，Docker build 需 GITHUB_TOKEN

### 6. kfc-k8s — Kubernetes 部署設定

```
kfc-k8s/
├── base/
│   ├── app/           # 各服務 Deployment/StatefulSet
│   ├── data/          # MongoDB, Redis, Kafka
│   ├── observability/ # Prometheus, Grafana, Tempo, Loki
│   ├── hpa/           # Horizontal Pod Autoscaling
│   └── keda/          # Event-driven Autoscaling (Kafka lag)
└── overlays/          # dev, staging, prod 環境差異
```

### 7. Special_Game_Pipeline — 本地整合開發環境

提供 docker-compose 一鍵啟動完整環境：
- **dummyPlatform** (Go)：模擬遊戲平台 (登入/錢包/玩家管理)
- **MockGameFrontEnd** (Vue 3)：測試用遊戲前端
- **IntegrationTool** (Vue 3)：遊戲啟動器 (選擇遊戲/語系/幣別)
- 基礎設施：MongoDB Replica Set、Redis、Kafka、ELK、Grafana 全套

---

## 統一架構原則 — Clean Architecture

所有 Go 服務遵循相同的 Clean Architecture 分層：

```
internal/
├── domain/              # 領域層 (最內層，零外部依賴)
│   ├── entity/          # 核心業務實體 (禁止 json/bson tag)
│   └── port/            # 全局外部服務介面
│
├── usecase/             # 應用層 (業務邏輯)
│   └── {feature}/
│       ├── contract.go  # UseCase 介面 + Command/Result
│       ├── uc.go        # 實作
│       └── ports/       # 功能專用依賴介面
│
├── adapter/             # 介面轉接層
│   ├── in/              # 入站 (HTTP handler, gRPC server, WS handler, Kafka consumer)
│   │   └── {protocol}/dto/  # Request/Response DTO (序列化 tag 在此層)
│   ├── out/             # 出站 (Repository, Gateway, ConfigCache)
│   └── presenter/       # 回應格式轉換
│
└── infrastructure/      # 基礎設施層 (最外層)
    ├── config/          # 環境設定
    ├── mongodb/         # MongoDB 連線 + migration
    ├── redis/           # Redis 客戶端
    └── server/          # HTTP/gRPC server 設定
```

### 依賴規則

```
Infrastructure → Adapter → Use Case → Domain
     (外層)                           (內層)
```

- **內層不可依賴外層**
- **Domain Entity 禁止序列化 tag** (json/bson 放 Adapter 層 DTO)
- **UseCase 定義介面 (ports/)**，Adapter 實作介面
- **Handler 薄層化**：解析請求 → 呼叫 UseCase → 回傳結果

### 開發新功能標準流程

1. 定義 Entity (`domain/entity/`)
2. 定義功能專用 Port (`usecase/{feature}/ports/`)
3. 定義 UseCase 介面 + Command/Result (`usecase/{feature}/contract.go`)
4. 實作 UseCase (`usecase/{feature}/uc.go`)
5. 定義 DTO (`adapter/in/{protocol}/dto/`)
6. 實作 Repository/Gateway (`adapter/out/`)
7. 實作 Handler (`adapter/in/{protocol}/`)
8. 在 `main.go` 或 DI 容器中組裝依賴

---

## 跨服務資料流

### 玩家遊戲流程

```
1. 平台登入 → dummyPlatform POST /auth/login → UserToken (JWT)
2. 遊戲連線 → Connector (WS upgrade + token verify via Platform Auth)
3. 遊戲初始化 → Connector → GameService gRPC InitGame → ConfigCache 取設定
4. 下注旋轉 → Connector → GameService gRPC Spin:
   ├── SpinLock (Redis 分佈式鎖)
   ├── Debit (HTTP → Platform Wallet)
   ├── RNG (math-lib engine.PlayRound())
   ├── Credit (HTTP → Platform Wallet，失敗走 Outbox 補償)
   └── Event (Kafka: bet.confirmed / bet.settled)
5. 推播 → PushProducer → Kafka WsEventPush → Connector → Player
```

### 設定同步流程

```
admin-backend (CRUD) → config-service (REST 寫入)
config-service → gameservice:
  ├── 啟動時 Full Sync (REST API)
  └── 運行時 SSE /events/config (即時增量更新)
        ├── currency_updated / currency_deleted
        ├── integrator_updated / integrator_deleted
        └── platform_endpoint_updated / platform_endpoint_deleted
斷線重連: 等待 5s → Full Sync → 重新 SSE 訂閱
```

### 匯率換算

```
admin-backend 管理匯率表 → config-service → gameservice configcache
gameservice 內部: 平台幣別 × exchangeRate = USD → 遊戲邏輯以 USD 計算 → USD / exchangeRate = 平台幣別回傳
```

---

## 技術棧速查

| 分類 | 技術 |
|------|------|
| 語言 | Go 1.25-1.26 (後端)、TypeScript (前端) |
| HTTP 框架 | Gin (所有 Go 服務) |
| RPC | gRPC + Protocol Buffers |
| WebSocket | Gorilla WebSocket + 自訂二進位協議 |
| 資料庫 | MongoDB (遊戲記錄/設定)、Redis (快取/鎖) |
| 訊息佇列 | Apache Kafka (事件流) |
| 前端 | Vue 3 + Vite + Vben Admin |
| 可觀測性 | OpenTelemetry (tracing)、Prometheus (metrics)、Grafana + Loki + Tempo (視覺化) |
| 容器 | Docker (Alpine 多階段建置) |
| 編排 | Kubernetes + Kustomize (HPA, KEDA) |
| 認證 | JWT (UserToken 由平台發行，遊戲商不解析) |

---

## 關鍵設計模式

### 1. 無縫錢包 (Seamless Wallet)
- 遊戲商不管理玩家資金，所有金流操作透過 HTTP 呼叫平台端點
- 端點設定來自 config-service，支持按 IID + GameType 細粒度設定
- Wallet Gateway 含 exponential backoff + jitter 重試

### 2. Outbox Pattern (補償機制)
- 扣款成功但派彩失敗時，事件寫入 MongoDB event_outbox
- Background worker 定期重試，確保最終一致性

### 3. SpinLock (分佈式鎖)
- Redis 實現，防止同一玩家併發 spin
- 狀態機：idle → ready → spinning

### 4. Config Cache + SSE
- 記憶體快取遊戲設定，啟動 Full Sync + SSE 即時更新
- Fallback：config-service 未設定時，用 `PLATFORM_URL` 環境變數自動組合端點

### 5. WebSocket 二進位協議
```
Flag (1B) | ID (2B) | RouteLen (1B) | Route (NB) | Payload (JSON)
```
- 路由：`game.init`、`game.spin`、`wallet.balance`
- 推播路由：`wallet.balance.update`、`system.shutdown`、`system.force_logout`

### 6. GLI 合規
- 數學引擎獨立部署 (math-lib 作為 library)
- CanSpin 驗證確保審計軌跡
- BetRecord 結算後不可變

---

## APM / 可觀測性開發指引

### 自動化 (不需手動加)

- HTTP 入站：`ginmw.Trace()` 自動 trace
- gRPC 入站/出站：`grpcmw.ServerOption()` / `ClientOption()` 自動 trace
- MongoDB/Redis：driver 層自動 trace
- HTTP 出站 (Wallet)：`otelhttp.NewTransport()` 自動 trace

### 手動加 Span (UseCase 業務邏輯)

```go
import "github.com/gameteam707/go-observability/tracing"

ctx, span := tracing.Start(ctx, "spin.debitWallet")
defer span.End()
```

**Span 命名慣例：** `{useCase}.{step}`，例如 `spin.debitWallet`、`initGame.getBalance`

### 錯誤記錄

```go
slog.ErrorContext(ctx, "debit failed", "roundID", cmd.RoundID, "error", err)
// 透過 ctx 自動關聯 trace，不需額外呼叫 span.RecordError()
```

---

## MongoDB Migration 規範

各服務使用統一的 migration 框架：

1. 在 `infrastructure/mongodb/migration/migrations/` 下新增檔案，編號遞增
2. 實作 `Up()` / `Down()`
3. 在 `registry.go` 的 `All()` 中註冊
4. **啟動時自動執行**，失敗則服務不啟動

---

## 開發通用規則

- **語言**: 一律使用繁體中文溝通
- **編譯檢查**: 每次開發完成後執行 `go build` 確保編譯成功
- **開發文件**: 每次功能開發前後需在對應目錄 (`/prd` 或 `/docs`) 寫完整計畫及報告
- **測試**: 新功能需撰寫測試，UseCase 層覆蓋率目標 90%+
- **前端 i18n**: 所有 UI 文字使用 `t()` 翻譯函式，禁止硬編碼
- **時區**: 後端一律 UTC 儲存/傳輸，前端負責時區轉換
- **Domain Entity**: 禁止序列化 tag，序列化結構定義在 Adapter 層

---

## 各 Repo 速查

| 目錄 | 用途 | 主要技術 | CLAUDE.md |
|------|------|---------|-----------|
| `sfc-stream-game/` | 核心遊戲服務 (connector, gameservice, gameapi, push-producer) | Go, gRPC, WS, Kafka | 有 |
| `admin-backend/` | 管理後台 (server + web) | Go + Gin, Vue 3 + Vben | 有 |
| `config-service/` | 設定中心 (REST + SSE) | Go + Gin | 無 |
| `go-observability/` | OTel 共用模組 | Go + OpenTelemetry | 無 |
| `math-lib/` | RNG/機率引擎 | Go | 無 |
| `kfc-k8s/` | K8s 部署設定 | Kustomize YAML | 無 |
| `Special_Game_Pipeline/` | 本地整合環境 | Docker Compose | 有 |
| `Backend-Core-Project/` | 舊架構 (已棄用，僅供參考) | Go DDD 單機 | 無 |
| `Game-Server-Wiki/` | 專案維基百科 | Markdown | — |

---

## 新人入門建議

1. 先閱讀 `Game-Server-Wiki/junior-developer-guide/` 的 00-15 系列文件
2. 用 `Special_Game_Pipeline/docker-compose.yml` 啟動本地完整環境
3. 從 connector 的 WebSocket handler 追蹤一次完整的 game.spin 流程
4. 理解 config-service → gameservice 的 SSE 同步機制
5. 理解 Wallet Gateway 的重試與 Outbox 補償邏輯
