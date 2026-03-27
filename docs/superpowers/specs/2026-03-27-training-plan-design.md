# Training Plan Design — KFC Junior Developer 訓練計畫

## 概述

在 Game-Server-Wiki repo 中建立一套 agent 驅動的訓練系統，讓 junior developer 透過「知識問答 + 漸進式實作」完成 onboarding，畢業後具備獨立開發能力。

## 設計決策摘要

| 項目 | 決定 |
|------|------|
| 完成形式 | 混合型：前半知識問答 + 後半動手實作 |
| 互動環境 | wiki repo 放教材/rules/進度，練習專案在獨立 repo |
| 進度儲存 | 存 wiki repo、push trainee branch，mentor 可在 GitHub 檢視 |
| 出題方式 | 核心必考題（預設題庫）+ agent 動態延伸題 |
| 練習專案 | 一起設計骨架（scaffold），`/init` 時生成 |
| Mentor 角色 | 預設 agent 自動推進，可配置 checkpoint 需人工 review |
| 章節範圍 | 全部涵蓋，分必修/選修，必修通過才畢業 |
| 實作規模 | 漸進式累積：同一個服務上逐階段疊加功能 |

---

## 1. 課程結構

### 階段劃分

| 階段 | 名稱 | 對應章節 | 類型 | 必修/選修 |
|------|------|----------|------|-----------|
| 1 | 全局觀 | 00-03 | quiz | 必修 |
| 2 | 核心分層 | 04-07 | quiz | 必修 |
| 3 | 基礎實作 | — | implementation | 必修 |
| 4 | 橫切關注點 | 08-11 | quiz | 必修 |
| 5 | 進階實作 | — | implementation | 必修 |
| 6 | 營運面 | 12-13 | quiz | 選修 |
| 7 | 測試 | 14 | mixed | 必修 |
| 8 | 學習路線 | 15 | 自行閱讀 | 選修 |

### 設計邏輯

- 讀完理論就實作：階段 1-2 讀完架構概念後，階段 3 馬上動手建骨架
- 漸進累積：階段 3 建出的服務在階段 5 繼續疊加功能，階段 7 再補測試
- 選修不擋畢業：階段 6 和 8 標記選修，junior 可畢業後再回來看
- 階段按順序解鎖：完成階段 N 才能進入 N+1

### curriculum.yaml 結構

```yaml
version: 1
graduation_required: [1, 2, 3, 4, 5, 7]

stages:
  - id: 1
    name: "全局觀"
    type: quiz
    required: true
    chapters: ["00", "01", "02", "03"]
    pass_criteria:
      core_questions: 80
      extension_questions: 60

  - id: 3
    name: "基礎實作"
    type: implementation
    required: true
    tasks:
      - "建立 domain entity（Item + ValueObject）"
      - "建立 usecase（create_item）+ contract + ports"
      - "建立 HTTP handler + DTO + 路由"
      - "手動測試 API 可正常呼叫"
    review: auto
    checkpoint: false

  - id: 7
    name: "測試"
    type: mixed
    required: true
    chapters: ["14"]
    tasks:
      - "為 usecase 寫 unit test（mock ports）"
      - "為 repository 寫 integration test"
    review: auto
    checkpoint: true
```

---

## 2. 進度追蹤與 `/init` 流程

### `/init` 執行步驟

1. 問 junior 名字（用於 branch 和 folder 命名）
2. 在 wiki repo 建立進度結構：
   ```
   trainees/<name>/
   ├── progress.yaml
   ├── answers/
   └── reviews/
   ```
3. 建立 branch `trainee/<name>` 並 commit
4. 用 scaffold 在 wiki repo 的上層目錄生成練習專案：
   ```
   ../kfc-training-<name>/
   ```
5. 偵測已存在則提示，不重複建立（冪等）

### progress.yaml 結構

```yaml
trainee: "john"
started_at: "2026-03-27"
current_stage: 1

stages:
  1:
    status: in_progress
    chapters:
      "00": { read: false, quiz_passed: false }
      "01": { read: false, quiz_passed: false }
      "02": { read: false, quiz_passed: false }
      "03": { read: false, quiz_passed: false }
  2:
    status: pending
    chapters:
      "04": { read: false, quiz_passed: false }
      "05": { read: false, quiz_passed: false }
      "06": { read: false, quiz_passed: false }
      "07": { read: false, quiz_passed: false }
  3:
    status: pending
    tasks:
      - { name: "建立 domain entity", status: pending }
      - { name: "建立 usecase", status: pending }
      - { name: "建立 HTTP handler", status: pending }
      - { name: "手動測試 API", status: pending }
    mentor_review: null
  4:
    status: pending
    chapters:
      "08": { read: false, quiz_passed: false }
      "09": { read: false, quiz_passed: false }
      "10": { read: false, quiz_passed: false }
      "11": { read: false, quiz_passed: false }
  5:
    status: pending
    tasks:
      - { name: "實作 MongoDB repository", status: pending }
      - { name: "加入 DI container", status: pending }
      - { name: "完成 CRUD usecase", status: pending }
      - { name: "加 middleware", status: pending }
    mentor_review: null
  6:
    status: pending
    required: false
    chapters:
      "12": { read: false, quiz_passed: false }
      "13": { read: false, quiz_passed: false }
  7:
    status: pending
    chapters:
      "14": { read: false, quiz_passed: false }
    tasks:
      - { name: "usecase unit test", status: pending }
      - { name: "repository integration test", status: pending }
    mentor_review: pending
  8:
    status: pending
    required: false
    chapters:
      "15": { read: false }
```

### 階段解鎖規則

- 完成階段 N → 解鎖 N+1
- 選修階段可標記 `skipped`
- `checkpoint: true` 的階段，agent review 完成後狀態變 `awaiting_mentor`
- Mentor 在 `trainees/<name>/reviews/` 下建立 `stage-<id>-review.md` 標記 approved

---

## 3. `/train` 互動流程

### 路由邏輯

```
/train
  → 讀取 progress.yaml，找 current_stage
  → quiz 型：
      未讀章節 → 提示閱讀 + 重點導讀
      未通過章節 → 出題（核心必考 → 動態延伸）
      全部通過 → 推進下一階段
  → implementation 型：
      pending task → 說明任務需求、驗收標準、提示方向
      in_progress task → 讀練習 repo code，給 review
      全部完成 → 若 checkpoint 則等 mentor，否則推進
  → mixed 型：
      先 quiz，再 implementation
  → 所有必修通過 → 畢業
```

### 問答互動流

1. Agent 出核心必考題（從 `questions/<chapter>.yaml`）
2. Junior 回答
3. Agent 判定：
   - 正確 → 記錄，下一題
   - 不完整 → 釋放 hint（逐步），引導補充
   - 錯誤 → 指出矛盾點，引導重新思考
4. 核心題通過 → agent 動態出 1-2 題延伸題
5. 該章通過 → 更新 progress.yaml，commit & push

### 實作互動流

1. Agent 說明 task 需求和驗收標準
2. 提供引導方向，不給完整 code
3. Junior 自行寫 code，完成後跑 `/train`
4. Agent 讀練習 repo 對應檔案，review
5. 通過 → 標記完成；不通過 → 給引導式建議

---

## 4. Agent 行為規則

### 核心原則：不直接給答案

**知識問答時：**
- Junior 回答不完整 → 提供引導性提示（相關章節段落、類比、反問）
- 可以給：概念解釋、文件連結、思考方向、章節參考
- 不可以給：完整答案

**引導策略：**
1. 先問 junior 自己的理解
2. 方向正確但不完整 → 追問細節
3. 方向錯誤 → 指出矛盾點，引導重新思考
4. 完全卡住 → 提示重讀哪個章節的哪個段落

**實作 review 時：**
- 指出問題的方向和原因，不直接給修正後的 code
- 可以給：概念提示、設計模式參考、相關 wiki 章節
- 不可以給：完整實作 code、直接幫 junior 改 code

### 進度管理

- 每次 task 狀態變更都要更新 progress.yaml 並 commit
- 階段必須按順序完成，不可跳過必修
- Agent 不可自行將 `awaiting_mentor` 改為 `passed`

---

## 5. 題目設計

### 題型

| 題型 | 用途 | 範例 |
|------|------|------|
| 概念題 | 確認讀懂核心概念 | 「依賴規則為什麼要求內層不能依賴外層？」 |
| 情境判斷題 | 確認能應用概念 | 「驗證信用卡格式的邏輯該放在哪一層？」 |
| 程式碼判讀題 | 確認看得懂架構 code | 給違反依賴規則的 code，問「哪裡有問題？」 |
| 比較題 | 確認理解 trade-off | 「Entity 和 DTO 為什麼不共用？」 |

每章 3-5 題核心必考題，agent 動態補 1-2 題延伸題。

### 題目檔案格式 (questions/*.yaml)

```yaml
chapter: "01"
title: "專案總覽"
questions:
  - id: "01-core-1"
    type: concept
    question: "KFC 平台為什麼選擇微服務架構而非單體應用？請說出至少兩個原因。"
    hints:
      - "想想看當遊戲流量暴增時，單體架構會遇到什麼問題"
      - "再想想多人協作開發時，單體架構有什麼不便"
    pass_criteria: "提到獨立部署/擴展、團隊獨立開發、故障隔離中的至少兩項"

  - id: "01-core-2"
    type: scenario
    question: "如果 gameservice 需要知道某款遊戲的 RTP 設定被更新了，它是怎麼得知的？"
    hints:
      - "看看 config-service 的職責是什麼"
      - "想想 SSE 在這裡扮演什麼角色"
    pass_criteria: "描述出 admin-backend → MongoDB → config-service (Change Stream) → SSE → gameservice 的流程"
```

- `hints`：逐步釋放給 junior，不直接揭露答案
- `pass_criteria`：僅供 agent 判定用，不展示給 junior

---

## 6. 練習專案

### 主題：物品管理服務 (Item Service)

選擇原因：簡單易懂、能完整走過所有架構層、與 KFC 的 game management 模式平行。

### 漸進式 Task 安排

| 階段 | Task | 產出 |
|------|------|------|
| 3：基礎實作 | 建立 `Item` entity（Name, Description, Price, Status） | `domain/entity/item.go` |
| | 建立 `create_item` usecase + contract + ports | `usecase/create_item/` |
| | 建立 HTTP handler + request DTO + 路由 | `adapter/in/http/` |
| | 手動測試 POST /api/v1/items 可正常回應 | 可運行的 API |
| 5：進階實作 | 實作 MongoDB repository（實作 port 介面） | `adapter/out/repository/` |
| | 加入 DI container 組裝所有元件 | `infrastructure/di/` |
| | 加 list/get/update/delete usecase（完整 CRUD） | 更多 usecase |
| | 加 middleware（request logging, error handling） | `adapter/in/http/middleware/` |
| 7：測試 | 為 `create_item` usecase 寫 unit test（mock repo） | `*_test.go` |
| | 為 MongoDB repository 寫 integration test | `*_integration_test.go` |

### Scaffold 骨架

生成路徑：`../kfc-training-<name>/`

```
kfc-training-<name>/
├── go.mod
├── cmd/server/main.go              ← 最小可運行進入點
├── internal/
│   ├── domain/entity/.gitkeep
│   ├── usecase/.gitkeep
│   ├── adapter/
│   │   ├── in/http/.gitkeep
│   │   └── out/repository/.gitkeep
│   └── infrastructure/
│       ├── config/config.go        ← 環境變數讀取骨架
│       └── server/server.go        ← Gin server 啟動 + 空路由群組
├── docker-compose.yaml             ← MongoDB + Redis
├── Makefile                        ← run, test, lint
└── README.md
```

只有 `config.go`、`server.go`、`main.go` 有實際 code，其餘為空目錄。

---

## 7. 檔案總覽

### Wiki Repo 最終結構

```
Game-Server-Wiki/
├── junior-developer-guide/         ← 既有 16 篇
├── training/
│   ├── curriculum.yaml
│   ├── checkpoints.yaml
│   ├── questions/
│   │   ├── 00.yaml ... 14.yaml
│   └── scaffold/
│       ├── go.mod
│       ├── cmd/server/main.go
│       ├── internal/...
│       ├── docker-compose.yaml
│       ├── Makefile
│       └── README.md
├── .claude/
│   ├── rules/
│   │   └── training-context.md
│   └── skills/
│       ├── init.md
│       └── train.md
├── trainees/
│   └── .gitkeep
├── CLAUDE.md
└── LICENSE
```

### Skill 職責邊界

| | `/init` | `/train` |
|--|---------|----------|
| 何時用 | 第一次加入訓練 | 每次訓練互動 |
| 做什麼 | 問名字 → 建 folder → 建 branch → scaffold 練習 repo → commit & push | 讀進度 → 路由到對應互動 |
| 冪等性 | 偵測已存在則提示，不重複建立 | 每次讀最新 progress.yaml |
| 寫入 | progress.yaml 初始狀態 | 更新 progress.yaml + answers/ |
