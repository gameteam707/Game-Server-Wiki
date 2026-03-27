# Training Plan Implementation

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an agent-driven junior developer training system with quiz + incremental implementation, progress tracking, and mentor checkpoints.

**Architecture:** Wiki repo hosts curriculum definition (YAML), question banks (YAML), scaffold templates (Go), agent skills (Markdown), and trainee progress folders. A separate practice repo is generated per trainee at `/init` time. Two Claude Code skills (`/init`, `/train`) drive all interactions.

**Tech Stack:** YAML (curriculum, questions, progress), Markdown (skills, rules), Go (scaffold project with Gin)

**Spec:** `docs/superpowers/specs/2026-03-27-training-plan-design.md`

---

## Chunk 1: Curriculum, Checkpoints, and Directory Scaffolding

### Task 1: Create curriculum.yaml

**Files:**
- Create: `training/curriculum.yaml`

- [ ] **Step 1: Write curriculum.yaml with all 8 stages**

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

  - id: 2
    name: "核心分層"
    type: quiz
    required: true
    chapters: ["04", "05", "06", "07"]
    pass_criteria:
      core_questions: 80
      extension_questions: 60

  - id: 3
    name: "基礎實作"
    type: implementation
    required: true
    tasks:
      - id: "3-1"
        name: "建立 domain entity（Item + ValueObject）"
        description: "在練習專案中建立 Item entity，包含 Name, Description, Price (float64), Status (ItemStatus) 欄位。Status 使用自訂型別 + 常數（active/inactive）。不可有任何 struct tag。"
        acceptance:
          - "檔案位於 internal/domain/entity/item.go"
          - "Item struct 無任何 json/bson tag"
          - "ItemStatus 為自訂型別，有 Active/Inactive 常數"
      - id: "3-2"
        name: "建立 create_item usecase + contract + ports"
        description: "建立 create_item usecase，包含 contract.go（UseCase 介面 + CreateCommand + CreateResult）、ports/repository.go（ItemRepository 介面）、uc.go（實作）。UseCase 應檢查 Name 不可為空。"
        acceptance:
          - "目錄結構：internal/usecase/create_item/{contract.go, uc.go, ports/repository.go}"
          - "CreateCommand 使用業務語言，無 json tag"
          - "UseCase 透過 ports.ItemRepository 介面操作，不直接依賴 MongoDB"
          - "NewCreateItemUseCase 接收介面、回傳介面"
      - id: "3-3"
        name: "建立 HTTP handler + DTO + 路由"
        description: "建立 item_handler.go、dto/request/item.go、dto/response/item.go。Handler 的 Create 方法：解析 JSON → 轉換為 Command → 呼叫 UseCase → 回傳 Response DTO。在 server.go 中註冊路由 POST /api/v1/items。"
        acceptance:
          - "Request DTO 有 json tag 和 binding 驗證"
          - "Response DTO 有 FromEntity 轉換函式"
          - "Handler 不包含業務邏輯"
          - "路由已註冊在 server.go"
      - id: "3-4"
        name: "手動測試 API 可正常呼叫"
        description: "啟動服務，用 curl 或 Postman 發送 POST /api/v1/items，確認回傳正確的 JSON 回應。此階段可以用 in-memory 假實作（stub）來滿足 ItemRepository 介面。"
        acceptance:
          - "服務可啟動無 panic"
          - "POST /api/v1/items 回傳 201 + JSON body"
    review: auto
    checkpoint: false

  - id: 4
    name: "橫切關注點"
    type: quiz
    required: true
    chapters: ["08", "09", "10", "11"]
    pass_criteria:
      core_questions: 80
      extension_questions: 60

  - id: 5
    name: "進階實作"
    type: implementation
    required: true
    tasks:
      - id: "5-1"
        name: "實作 MongoDB repository"
        description: "建立 adapter/out/repository/mongo/item_repo.go，實作 ports.ItemRepository 介面。包含 itemDocument struct（有 bson tag）和 toDocument/toEntity 轉換函式。實作 Create、FindByID 方法。"
        acceptance:
          - "itemDocument 有 bson tag，與 entity.Item 分離"
          - "有 toDocument 和 toEntity 轉換函式"
          - "NewItemRepository 回傳 ports.ItemRepository 介面"
      - id: "5-2"
        name: "加入 DI container 組裝所有元件"
        description: "建立 infrastructure/di/container.go，按照由內而外的順序組裝：Repository → UseCase → Handler。Container struct 暴露 Handler 給 Router 使用。"
        acceptance:
          - "組裝順序正確：Repository → UseCase → Handler"
          - "所有依賴透過建構函式注入"
          - "main.go 使用 Container 啟動服務"
      - id: "5-3"
        name: "完成 CRUD usecase（list, get, update, delete）"
        description: "為 Item 新增 list_items、get_item、update_item、delete_item 四個 usecase。每個都有 contract.go + uc.go。更新 ItemRepository 介面加入 List、FindByID、Update、Delete 方法。更新 Handler 加入對應路由。"
        acceptance:
          - "4 個新 usecase 各有 contract.go + uc.go"
          - "ItemRepository 介面有完整 CRUD 方法"
          - "路由：GET /items, GET /items/:id, PUT /items/:id, DELETE /items/:id"
      - id: "5-4"
        name: "加 middleware（request logging, error handling）"
        description: "建立 adapter/in/http/middleware/ 目錄，加入 request_logger.go（記錄每個請求的 method、path、status、duration）和 error_handler.go（統一 panic recovery + 錯誤格式）。在 server.go 中套用。"
        acceptance:
          - "每個請求都有結構化日誌輸出"
          - "Panic 會被 recovery 攔截，回傳 500 JSON"
          - "Middleware 套用在路由群組上"
    review: auto
    checkpoint: false

  - id: 6
    name: "營運面"
    type: quiz
    required: false
    chapters: ["12", "13"]
    pass_criteria:
      core_questions: 80
      extension_questions: 60

  - id: 7
    name: "測試"
    type: mixed
    required: true
    chapters: ["14"]
    tasks:
      - id: "7-1"
        name: "為 create_item usecase 寫 unit test（mock repo）"
        description: "在 usecase/create_item/uc_test.go 中，手寫 mockItemRepository，測試：成功建立、Name 為空時回錯誤。使用 AAA 模式。"
        acceptance:
          - "Mock 實作 ports.ItemRepository 介面"
          - "至少 2 個測試案例（成功 + 驗證失敗）"
          - "go test 通過"
      - id: "7-2"
        name: "為 MongoDB repository 寫 integration test"
        description: "在 adapter/out/repository/mongo/item_repo_integration_test.go 中，使用真實的 MongoDB（docker-compose 提供）測試 Create + FindByID 的完整流程。"
        acceptance:
          - "測試連接真實 MongoDB"
          - "測試 Create → FindByID 的往返"
          - "測試後清理資料"
    review: auto
    checkpoint: true
    checkpoint_review: mentor

  - id: 8
    name: "學習路線"
    type: reading
    required: false
    chapters: ["15"]
```

- [ ] **Step 2: Commit**

```bash
git add training/curriculum.yaml
git commit -m "feat(training): add curriculum definition with 8 stages"
```

---

### Task 2: Create checkpoints.yaml

**Files:**
- Create: `training/checkpoints.yaml`

- [ ] **Step 1: Write checkpoints.yaml**

```yaml
# Mentor checkpoint configuration
# review_type: auto | mentor
#
# auto: agent reviews and approves automatically
# mentor: agent reviews first, then sets status to awaiting_mentor
#         mentor must create trainees/<name>/reviews/stage-<id>-review.md
#         with "approved: true" to pass the stage

stages:
  1: auto
  2: auto
  3: auto
  4: auto
  5: auto
  6: auto
  7: mentor    # 畢業關卡，需 mentor 確認
  8: auto

# Mentor review file format:
# ---
# stage: 7
# approved: true
# reviewer: "<mentor name>"
# date: "YYYY-MM-DD"
# ---
# <review comments>
```

- [ ] **Step 2: Commit**

```bash
git add training/checkpoints.yaml
git commit -m "feat(training): add mentor checkpoint configuration"
```

---

### Task 3: Create trainees directory

**Files:**
- Create: `trainees/.gitkeep`

- [ ] **Step 1: Create .gitkeep**

Create an empty file at `trainees/.gitkeep`.

- [ ] **Step 2: Commit**

```bash
git add trainees/.gitkeep
git commit -m "feat(training): add trainees directory"
```

---

## Chunk 2: Question Banks (chapters 00-07)

### Task 4: Create questions for Stage 1 — 全局觀 (chapters 00-03)

**Files:**
- Create: `training/questions/00.yaml`
- Create: `training/questions/01.yaml`
- Create: `training/questions/02.yaml`
- Create: `training/questions/03.yaml`

- [ ] **Step 1: Write training/questions/00.yaml**

```yaml
chapter: "00"
title: "導讀指南"
questions:
  - id: "00-core-1"
    type: concept
    question: "這份 wiki 建議的閱讀方式有哪些重點？為什麼要「搭配程式碼一起看」？"
    hints:
      - "想想看，只看文字描述和實際看到程式碼結構，哪個更容易理解架構？"
    pass_criteria: "提到搭配 IDE 對照程式碼、理解為什麼比記住怎麼做更重要、動手試做中的至少兩項"

  - id: "00-core-2"
    type: concept
    question: "文件將閱讀分成五個階段，為什麼建議先讀全局觀（01-03），再讀各層深入（04-07），而不是反過來？"
    hints:
      - "想想學習一個新城市，你會先看地圖還是先走進一條小巷子？"
    pass_criteria: "提到需要先建立整體架構的心智模型，才能理解每一層的設計決策和存在意義"
```

- [ ] **Step 2: Write training/questions/01.yaml**

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
    question: "如果 gameservice 需要知道某款遊戲的 RTP 設定被更新了，它是怎麼得知的？請描述完整的資料流。"
    hints:
      - "看看 config-service 的職責是什麼"
      - "想想 SSE 在這裡扮演什麼角色"
    pass_criteria: "描述出 admin-backend 修改 MongoDB → config-service 透過 Change Stream 偵測變更 → 透過 SSE 推播 → gameservice 收到更新的流程"

  - id: "01-core-3"
    type: comparison
    question: "admin-backend 用 HTTP，gameservice 用 gRPC，為什麼不統一用一種協定？"
    hints:
      - "想想兩個服務各自面對的使用者是誰"
      - "瀏覽器可以直接呼叫 gRPC 嗎？"
    pass_criteria: "提到 admin-backend 面向前端瀏覽器適合 HTTP REST，gameservice 是服務間高頻內部通訊適合 gRPC 的效能優勢（二進位傳輸、型別安全）"

  - id: "01-core-4"
    type: scenario
    question: "如果今天要新增一款遊戲，會涉及哪些服務？各自負責什麼？"
    hints:
      - "管理員在哪裡操作？操作完後其他服務怎麼知道？"
    pass_criteria: "提到 admin-backend（管理員透過前端建立遊戲）→ 寫入 MongoDB → config-service 偵測變更並推播 → gameservice 更新內部快取"
```

- [ ] **Step 3: Write training/questions/02.yaml**

```yaml
chapter: "02"
title: "Clean Architecture 核心概念"
questions:
  - id: "02-core-1"
    type: concept
    question: "Clean Architecture 最重要的一條規則是什麼？這條規則解決了什麼問題？"
    hints:
      - "回想同心圓圖中箭頭的方向"
    pass_criteria: "說出依賴方向只能由外向內，內層不知道外層的存在。解決了業務邏輯耦合框架/資料庫的問題，使得換框架或換資料庫時業務邏輯不需要修改"

  - id: "02-core-2"
    type: scenario
    question: "一段「檢查遊戲名稱是否重複」的邏輯，應該放在哪一層？為什麼？"
    hints:
      - "問自己：如果今天從 Gin 換成 Echo，或從 MongoDB 換成 PostgreSQL，這段邏輯還需要嗎？"
    pass_criteria: "放在 UseCase 層。因為這是業務規則，不管換什麼框架或資料庫，名稱不能重複這個規則都不變"

  - id: "02-core-3"
    type: code_review
    question: |
      以下程式碼有什麼架構問題？
      ```go
      func CreateGame(c *gin.Context) {
          var req struct { Name string `json:"name"` }
          c.BindJSON(&req)
          collection := mongoClient.Database("db").Collection("games")
          collection.InsertOne(c, bson.M{"name": req.Name})
          c.JSON(200, gin.H{"success": true})
      }
      ```
    hints:
      - "這個函式同時做了幾件事？各自屬於哪一層？"
    pass_criteria: "指出職責混亂：HTTP 解析（Adapter）、業務邏輯（UseCase）、資料庫操作（Adapter/out）全部混在一個函式。違反依賴規則，無法單獨測試業務邏輯，換框架或資料庫要整個重寫"

  - id: "02-core-4"
    type: comparison
    question: "Domain Entity 和 DTO 看起來欄位很像，為什麼不共用同一個 struct？"
    hints:
      - "想想兩者的「變更原因」是否相同"
    pass_criteria: "提到變更原因不同：DTO 跟隨 API 規格變化，Entity 跟隨業務規則變化。共用會導致改 API 欄位名時影響到 Domain 層"
```

- [ ] **Step 4: Write training/questions/03.yaml**

```yaml
chapter: "03"
title: "目錄結構與分層"
questions:
  - id: "03-core-1"
    type: concept
    question: "Go 語言中 `internal/` 目錄有什麼特殊意義？為什麼本專案把所有業務程式碼都放在 internal/ 下？"
    hints:
      - "想想如果其他服務可以直接 import 你的內部程式碼，會有什麼問題"
    pass_criteria: "internal/ 目錄下的程式碼只能被同一個 module 的程式碼 import，是 Go 編譯器強制執行的規則。確保服務之間的邊界清晰，防止跨服務直接依賴內部實作"

  - id: "03-core-2"
    type: concept
    question: "adapter/ 目錄下分成 in/ 和 out/，這兩者各自代表什麼？請各舉一個例子。"
    hints:
      - "想想資料流動的方向：誰在呼叫誰？"
    pass_criteria: "in（入站）：外部世界呼叫系統，例如 HTTP Handler 接收 REST 請求。out（出站）：系統呼叫外部世界，例如 Repository 存取 MongoDB 或 Gateway 呼叫外部 API"

  - id: "03-core-3"
    type: scenario
    question: "如果要為 admin-backend 新增一個「公告管理」功能，你需要在哪些目錄下新增檔案？請列出至少四個。"
    hints:
      - "回想新功能開發流程：Entity → UseCase → Adapter"
      - "別忘了 DTO 和 DI Container"
    pass_criteria: "至少提到：domain/entity/（公告 Entity）、usecase/announcement/（UseCase + contract + ports）、adapter/in/http/（Handler）、adapter/in/http/dto/（Request/Response DTO）、adapter/out/repository/mongo/（Repository 實作）。提到要更新 DI Container 更好"
```

- [ ] **Step 5: Commit**

```bash
git add training/questions/
git commit -m "feat(training): add question banks for stage 1 (chapters 00-03)"
```

---

### Task 5: Create questions for Stage 2 — 核心分層 (chapters 04-07)

**Files:**
- Create: `training/questions/04.yaml`
- Create: `training/questions/05.yaml`
- Create: `training/questions/06.yaml`
- Create: `training/questions/07.yaml`

- [ ] **Step 1: Write training/questions/04.yaml**

```yaml
chapter: "04"
title: "Domain Layer"
questions:
  - id: "04-core-1"
    type: concept
    question: "Domain Entity 為什麼不能有 json 或 bson 等序列化標籤（struct tag）？"
    hints:
      - "想想 Entity 和「外部怎麼使用它」之間的關係"
      - "如果 API 要改欄位名，應該影響到 Domain 層嗎？"
    pass_criteria: "提到序列化標籤暗示 Entity 知道外部如何使用它，違反 Domain 層的獨立性。不同 Adapter 有不同的序列化需求（HTTP 用 camelCase JSON，DB 用 snake_case BSON），不應該把這些耦合在 Entity 上"

  - id: "04-core-2"
    type: comparison
    question: "Entity 和 Value Object 有什麼區別？本專案中 GameStatus 是 Entity 還是 Value Object？"
    hints:
      - "想想 GameStatus 有沒有自己的 ID"
    pass_criteria: "Entity 有唯一識別（ID），Value Object 沒有（用值本身區分）。GameStatus 是 Value Object（自訂型別 + 常數），它沒有 ID，靠值（active/inactive）來區分"

  - id: "04-core-3"
    type: concept
    question: "本專案的 Domain Entity 採用「貧血模型」而非「充血模型」，兩者有什麼差異？各有什麼優缺點？"
    hints:
      - "想想業務邏輯放在 Entity 裡面 vs 放在 UseCase 裡面"
    pass_criteria: "貧血模型：Entity 主要是資料載體，業務邏輯放在 UseCase。優點是結構簡單、容易理解，缺點是業務邏輯分散。充血模型：Entity 包含業務方法，優點是業務邏輯內聚，缺點是需要更深的 DDD 功力"
```

- [ ] **Step 2: Write training/questions/05.yaml**

```yaml
chapter: "05"
title: "UseCase Layer"
questions:
  - id: "05-core-1"
    type: concept
    question: "UseCase 目錄下有三個核心檔案：contract.go、ports/*.go、uc.go。它們各自的職責是什麼？"
    hints:
      - "想想「對外宣告我能做什麼」vs「我需要什麼外部能力」vs「我怎麼做」"
    pass_criteria: "contract.go：定義 UseCase 介面 + Command（輸入）+ Result（輸出），是對外的契約。ports/：定義 UseCase 需要的外部依賴介面（如 Repository），只有介面沒有實作。uc.go：業務邏輯的實際實作"

  - id: "05-core-2"
    type: concept
    question: "什麼是「依賴反轉 (Dependency Inversion)」？在 UseCase 和 Repository 的關係中是怎麼體現的？"
    hints:
      - "傳統做法是 UseCase 直接依賴 MongoRepository，依賴反轉後呢？"
    pass_criteria: "UseCase 定義介面（Port），Repository 實作介面。UseCase 不知道實際用的是 MongoDB 還是 PostgreSQL。高層模組（UseCase）不依賴低層模組（Repository 實作），兩者都依賴抽象（Port 介面）"

  - id: "05-core-3"
    type: code_review
    question: |
      以下 UseCase Command 的定義有什麼問題？
      ```go
      type CreateCommand struct {
          GameCode string `json:"game_code"`
          Names    bson.M
      }
      ```
    hints:
      - "Command 屬於 UseCase 層，它應該依賴什麼？"
    pass_criteria: "兩個問題：1) json tag 是 HTTP 層的關注點，不該出現在 UseCase 的 Command 中。2) bson.M 是 MongoDB 的型別，UseCase 不應該依賴資料庫套件。Command 應該使用純粹的業務語言定義"

  - id: "05-core-4"
    type: scenario
    question: "Auth Login UseCase 同時依賴了 AdminUserRepository 和 LoginLimiter 兩個 Port。為什麼不把限流邏輯直接寫在 UseCase 裡面，而是抽成 Port？"
    hints:
      - "想想限流的實作可能用 Redis，也可能用 in-memory，UseCase 應該知道嗎？"
      - "如果要測試登入邏輯但不想啟動 Redis，該怎麼辦？"
    pass_criteria: "抽成 Port 的好處：1) UseCase 不知道限流用什麼技術實作（Redis/in-memory）。2) 測試時可以注入 Mock，不需要真的 Redis。3) 未來換限流方案只需要寫新的實作，UseCase 不用改"
```

- [ ] **Step 3: Write training/questions/06.yaml**

```yaml
chapter: "06"
title: "Adapter Layer"
questions:
  - id: "06-core-1"
    type: concept
    question: "HTTP Handler 的職責應該只有哪些步驟？如果在 Handler 裡面看到業務規則的判斷，代表什麼問題？"
    hints:
      - "回想 Handler 的四個 Step"
    pass_criteria: "Handler 只做：1) 解析 HTTP Request 為 DTO 2) DTO 轉 UseCase Command 3) 呼叫 UseCase 4) Entity/Result 轉 Response DTO 回傳。如果有業務規則判斷（如檢查名稱是否重複），代表邏輯放錯地方了，應該在 UseCase 層"

  - id: "06-core-2"
    type: concept
    question: "在本專案中，一個完整的 HTTP 請求會經過幾次資料轉換？請依序列出每次轉換涉及的資料結構。"
    hints:
      - "從 JSON 進來到存入 MongoDB，中間經過了幾個不同的 struct？"
    pass_criteria: "至少描述出：HTTP JSON → Request DTO (json tag) → UseCase Command (無 tag) → Domain Entity (無 tag) → MongoDB Document (bson tag)，回程反向。每次轉換都是刻意的，確保每層的資料結構只服務自己的關注點"

  - id: "06-core-3"
    type: scenario
    question: "Repository 的 toDocument() 和 toEntity() 轉換函式為什麼是必要的？如果直接讓 Entity 同時有 bson tag 不是更簡單嗎？"
    hints:
      - "想想 API 回傳的欄位名和資料庫存的欄位名是否總是一樣的"
    pass_criteria: "轉換函式確保 Entity 保持純粹不依賴 MongoDB。不同 Adapter 可能有不同的命名慣例（API 用 camelCase，DB 用 snake_case）。若直接加 bson tag 在 Entity 上，改 DB 欄位名就會影響 Domain 層，違反依賴規則"
```

- [ ] **Step 4: Write training/questions/07.yaml**

```yaml
chapter: "07"
title: "Infrastructure Layer"
questions:
  - id: "07-core-1"
    type: concept
    question: "為什麼本專案使用環境變數來管理設定，而不是設定檔（如 config.json）？"
    hints:
      - "想想 Docker 和 Kubernetes 怎麼傳設定給容器"
    pass_criteria: "提到 12-Factor App 建議設定存在環境中。Docker/Kubernetes 原生支援環境變數注入。不同環境（開發/測試/正式）只需切換環境變數，不需改程式碼或維護多份設定檔"

  - id: "07-core-2"
    type: scenario
    question: "main.go 是整個服務的「組裝點」。請描述啟動流程的順序，以及為什麼這個順序很重要。"
    hints:
      - "如果 Handler 在 MongoDB 連線之前就被建立，會發生什麼？"
    pass_criteria: "順序：1) 載入設定 2) 建立基礎設施連線（MongoDB, Redis）3) 建立 DI Container（組裝 Repository → UseCase → Handler）4) 建立路由 5) 啟動伺服器。順序重要因為每一步都依賴前一步的產物，例如 Repository 需要 DB 連線，UseCase 需要 Repository"

  - id: "07-core-3"
    type: concept
    question: "Infrastructure Layer 是「最外層」，可以依賴所有內層。但為什麼 MongoDB 的連線建立（mongodb/client.go）和使用 MongoDB 的 Repository（adapter/out/repository/）要分開放？"
    hints:
      - "連線建立是「基礎設施」，使用連線做 CRUD 是「轉接」，它們的變更原因一樣嗎？"
    pass_criteria: "分開是因為職責不同：mongodb/client.go 負責建立和管理連線（基礎設施），Repository 負責使用連線做業務相關的 CRUD 操作（轉接）。連線設定變了（如換連線池大小）不需要改 Repository，業務查詢變了不需要改連線邏輯"
```

- [ ] **Step 5: Commit**

```bash
git add training/questions/
git commit -m "feat(training): add question banks for stage 2 (chapters 04-07)"
```

---

## Chunk 3: Question Banks (chapters 08-14)

### Task 6: Create questions for Stage 4 — 橫切關注點 (chapters 08-11)

**Files:**
- Create: `training/questions/08.yaml`
- Create: `training/questions/09.yaml`
- Create: `training/questions/10.yaml`
- Create: `training/questions/11.yaml`

- [ ] **Step 1: Write training/questions/08.yaml**

```yaml
chapter: "08"
title: "依賴注入"
questions:
  - id: "08-core-1"
    type: comparison
    question: "「自己建立依賴」和「依賴注入」有什麼差別？為什麼後者更好？"
    hints:
      - "如果 UseCase 自己 new 一個 MongoRepository，測試時怎麼辦？"
    pass_criteria: "自己建立依賴：UseCase 內部直接建立 MongoRepository，造成緊耦合，無法用 Mock 測試。依賴注入：從外部傳入介面，UseCase 不知道具體實作，可以注入 Mock 測試，也容易替換實作"

  - id: "08-core-2"
    type: concept
    question: "本專案選擇手動 DI（Container Pattern）而非 DI 框架（如 Wire、Dig），原因是什麼？"
    hints:
      - "想想可讀性和錯誤檢測時機"
    pass_criteria: "手動 DI 程式碼明確一目了然、編譯時就能檢查錯誤、沒有額外依賴。對中小型專案，可讀性優勢大於框架的自動化便利性"

  - id: "08-core-3"
    type: scenario
    question: "DI Container 的組裝順序是「由內而外」。如果你把 UseCase 的建立放在 Repository 之前，會發生什麼？"
    hints:
      - "UseCase 的建構函式需要什麼參數？"
    pass_criteria: "會編譯失敗或 panic，因為 UseCase 的建構函式需要 Repository 介面作為參數。組裝順序必須是：基礎設施連線 → Repository（Adapter/out）→ UseCase → Handler（Adapter/in）→ Router"
```

- [ ] **Step 2: Write training/questions/09.yaml**

```yaml
chapter: "09"
title: "API 設計與 Middleware"
questions:
  - id: "09-core-1"
    type: concept
    question: "Gin 的 Middleware 中 c.Next() 和 c.Abort() 各自的作用是什麼？如果忘記呼叫 c.Next() 會怎樣？"
    hints:
      - "想想 Middleware 是一個鏈條，c.Next() 是往下走"
    pass_criteria: "c.Next() 繼續執行下一個 Middleware 或 Handler。c.Abort() 中斷鏈條，不再執行後續 Middleware。忘記呼叫 c.Next() 會導致請求停在這個 Middleware，Handler 永遠不會被執行"

  - id: "09-core-2"
    type: scenario
    question: "JWT Middleware 驗證完 Token 後，用 c.Set('userID', claims.UserID) 把資訊存入 Context。為什麼不直接把 Token 傳給 Handler，讓 Handler 自己解析？"
    hints:
      - "如果有 10 個 Handler 都需要 userID，每個都自己解析 Token？"
    pass_criteria: "Middleware 做一次驗證和解析，所有 Handler 都可以直接用 c.Get('userID') 取得。避免重複程式碼，確保驗證邏輯統一。職責分離：Middleware 負責認證，Handler 只需專注業務邏輯"

  - id: "09-core-3"
    type: code_review
    question: "Audit Log Middleware 使用 goroutine 非同步記錄日誌。這樣做有什麼好處和潛在風險？"
    hints:
      - "想想如果同步記錄，對回應時間有什麼影響"
      - "goroutine 裡面的錯誤，主請求能知道嗎？"
    pass_criteria: "好處：不會阻塞 HTTP 回應，使用者不需要等日誌寫入完成。風險：goroutine 裡的錯誤不會回傳給使用者，日誌寫入可能靜默失敗。另外 goroutine 使用了 context.Background() 而非請求的 Context，因為請求可能已經結束"
```

- [ ] **Step 3: Write training/questions/10.yaml**

```yaml
chapter: "10"
title: "資料庫與 Repository"
questions:
  - id: "10-core-1"
    type: concept
    question: "Repository 的 FindByID 在找不到資料時回傳 (nil, nil) 而非 error。為什麼這樣設計？"
    hints:
      - "「找不到資料」和「資料庫出錯」是同一種情況嗎？"
    pass_criteria: "找不到資料是正常的業務情況（使用者可能查了一個不存在的 ID），不算錯誤。真正的 error 是資料庫連線斷了、查詢語法錯誤等技術問題。呼叫者可以用 if result == nil 判斷是否存在，用 if err != nil 判斷是否出錯"

  - id: "10-core-2"
    type: scenario
    question: "本專案的 Redis 只用在登入限流，而不是用來快取 MongoDB 的查詢結果。為什麼？"
    hints:
      - "想想 admin-backend 的使用量和即時性需求"
    pass_criteria: "admin-backend 是管理後台，使用量不高，不需要快取來提升查詢效能。登入限流需要極高的讀寫速度和 TTL（自動過期）功能，Redis 原生支援這些特性。不必要的快取反而增加資料一致性的複雜度"

  - id: "10-core-3"
    type: concept
    question: "MongoDB 的 Change Streams 在 config-service 中扮演什麼角色？它和輪詢（Polling）相比有什麼優勢？"
    hints:
      - "想想 gameservice 需要多快知道設定變更了"
    pass_criteria: "Change Streams 讓 config-service 即時監聽 MongoDB 的資料變更，一有更新就透過 SSE 推播。相比輪詢：零無效請求、即時通知（不需等待輪詢間隔）、降低資料庫負載"
```

- [ ] **Step 4: Write training/questions/11.yaml**

```yaml
chapter: "11"
title: "微服務通訊"
questions:
  - id: "11-core-1"
    type: comparison
    question: "同步通訊（HTTP/gRPC）和非同步通訊（SSE/Kafka）各適合什麼場景？請各舉一個本專案中的例子。"
    hints:
      - "哪些操作需要立即得到結果？哪些可以稍後處理？"
    pass_criteria: "同步：需要立即回應的場景，如 gameapi 透過 gRPC 呼叫 gameservice 的 Spin（玩家等著看結果）。非同步：不需要立即處理的場景，如 gameservice 透過 Kafka 發送下注事件給下游統計系統，或 config-service 透過 SSE 推播設定更新"

  - id: "11-core-2"
    type: concept
    question: "Outbox Pattern 解決了什麼問題？如果不使用 Outbox，直接在業務操作後發 Kafka 訊息，會有什麼風險？"
    hints:
      - "如果 MongoDB 寫入成功但 Kafka 發送失敗呢？反過來呢？"
    pass_criteria: "解決資料庫操作和訊息發送之間的一致性問題。不用 Outbox：DB 寫入成功但 Kafka 失敗 → 資料庫有紀錄但下游不知道；或 Kafka 成功但 DB 失敗 → 下游收到了不存在的資料。Outbox 把訊息和業務資料放在同一個 Transaction 中，由背景 Worker 負責發送"

  - id: "11-core-3"
    type: scenario
    question: "gRPC 使用 Protocol Buffers 定義服務介面（.proto 檔案）。這和 HTTP REST 直接用 JSON 相比，在「型別安全」方面有什麼差異？"
    hints:
      - "JSON 的數字是 number，沒有 int 和 float 的區分，.proto 呢？"
    pass_criteria: "Proto 有嚴格的型別定義（int32、double、string 等），編譯時就能檢查型別錯誤，client 和 server 共用同一份 .proto 定義確保一致。JSON 是動態的，欄位名打錯、型別不匹配只有在執行時才會發現"
```

- [ ] **Step 5: Commit**

```bash
git add training/questions/
git commit -m "feat(training): add question banks for stage 4 (chapters 08-11)"
```

---

### Task 7: Create questions for Stage 6 and 7 (chapters 12-14)

**Files:**
- Create: `training/questions/12.yaml`
- Create: `training/questions/13.yaml`
- Create: `training/questions/14.yaml`

- [ ] **Step 1: Write training/questions/12.yaml**

```yaml
chapter: "12"
title: "可觀測性"
questions:
  - id: "12-core-1"
    type: concept
    question: "可觀測性的三大支柱是什麼？它們各自回答什麼問題？"
    hints:
      - "Logs、Metrics、Traces 各自的用途"
    pass_criteria: "Logs（日誌）：發生了什麼事。Metrics（指標）：系統目前的數字是多少（如請求數、延遲）。Traces（追蹤）：一個請求經過了哪些服務、各花了多少時間"

  - id: "12-core-2"
    type: comparison
    question: "結構化日誌（slog）和 fmt.Printf 有什麼差異？為什麼生產環境要用結構化日誌？"
    hints:
      - "如果你想在幾百萬行日誌中找到某個 sessionId 的紀錄，哪種格式比較容易搜尋？"
    pass_criteria: "結構化日誌輸出 JSON 格式的 key-value，可以被 Loki/ELK 等工具索引和搜尋。fmt.Printf 輸出純文字，難以自動解析和搜尋。生產環境需要快速定位問題，結構化日誌的可搜尋性是必要的"

  - id: "12-core-3"
    type: concept
    question: "Prometheus 的 Counter、Gauge、Histogram 三種指標類型各適合記錄什麼？"
    hints:
      - "有些數字只會往上加，有些會上下變動，有些需要看分佈"
    pass_criteria: "Counter：只增不減的累計值，如請求總數、錯誤數。Gauge：可增可減的即時值，如當前連線數、佇列長度。Histogram：數值的分佈，如請求延遲（可以看 P50/P95/P99）"
```

- [ ] **Step 2: Write training/questions/13.yaml**

```yaml
chapter: "13"
title: "部署與容器化"
questions:
  - id: "13-core-1"
    type: concept
    question: "Docker 多階段建置（Multi-stage Build）為什麼能大幅縮小映像大小？兩個階段各做什麼？"
    hints:
      - "Go SDK 本身就有好幾百 MB"
    pass_criteria: "第一階段（builder）：使用完整的 Go SDK 編譯程式碼。第二階段（runtime）：只複製編譯好的二進位檔到精簡的 Alpine 映像。最終映像不包含 Go SDK 和原始碼，只有約 20MB（vs ~1GB）"

  - id: "13-core-2"
    type: concept
    question: "Kubernetes 的 Deployment 和 Service 各自的職責是什麼？為什麼需要 Service？"
    hints:
      - "Pod 的 IP 是會變的"
    pass_criteria: "Deployment 管理 Pod 的副本數量、更新策略和自動修復。Service 提供穩定的網路入口（固定 DNS 名稱），因為 Pod 可能被重啟、IP 會改變，Service 抽象了這個變動，讓其他服務用固定名稱存取"

  - id: "13-core-3"
    type: scenario
    question: "Dockerfile 中為什麼先 COPY go.mod + go.sum 並 RUN go mod download，然後才 COPY 原始碼？而不是一次全部 COPY？"
    hints:
      - "Docker 的每一行指令都是一個快取層"
    pass_criteria: "利用 Docker 的層快取機制。依賴檔案（go.mod/go.sum）不常改變，把它們放前面可以讓 go mod download 這一層被快取。只有原始碼改變時才需要重新編譯，不需要重新下載依賴，大幅加速建置速度"
```

- [ ] **Step 3: Write training/questions/14.yaml**

```yaml
chapter: "14"
title: "測試策略"
questions:
  - id: "14-core-1"
    type: concept
    question: "測試金字塔中，為什麼 Unit Test 應該最多、E2E Test 應該最少？"
    hints:
      - "想想每種測試的速度、可靠度和維護成本"
    pass_criteria: "Unit Test：速度快（毫秒）、可靠度高、維護成本低，適合大量使用來覆蓋各種邏輯分支。E2E Test：速度慢（分鐘）、容易因環境問題而 flaky、維護成本高，只用來驗證關鍵的完整流程"

  - id: "14-core-2"
    type: concept
    question: "AAA 模式（Arrange-Act-Assert）是什麼？為什麼每個測試案例應該只有一個 Act？"
    hints:
      - "如果一個測試有兩個 Act，失敗時你知道是哪一步出錯嗎？"
    pass_criteria: "Arrange：準備測試資料和依賴。Act：執行被測試的操作。Assert：驗證結果是否符合預期。每個測試只有一個 Act 才能精確定位失敗原因，測試名稱也能清楚描述在測什麼情境"

  - id: "14-core-3"
    type: scenario
    question: "本專案用手寫 Mock 而非 Mock 框架（gomock、mockery）。在什麼情況下手寫 Mock 會變得不實際？"
    hints:
      - "如果一個介面有 20 個方法呢？"
    pass_criteria: "當介面方法很多時（>5-10 個），手寫 Mock 很繁瑣。Go 的介面通常方法很少所以手寫可行，但如果介面膨脹就該考慮框架或重構介面（Interface Segregation Principle——拆成更小的介面）"

  - id: "14-core-4"
    type: concept
    question: "Table-Driven Tests 是什麼？相比為每個案例寫獨立的函式，有什麼好處？"
    hints:
      - "新增一個測試案例需要做什麼？"
    pass_criteria: "用一個 struct slice 列出所有測試案例（輸入 + 預期結果），用迴圈 + t.Run() 逐一執行。好處：新增案例只需加一行到表格，不需寫新函式；測試結構統一，容易閱讀和維護；共用 Arrange/Assert 邏輯，減少重複程式碼"
```

- [ ] **Step 4: Commit**

```bash
git add training/questions/
git commit -m "feat(training): add question banks for stages 6-7 (chapters 12-14)"
```

---

## Chunk 4: Practice Project Scaffold

### Task 8: Create scaffold Go project

**Files:**
- Create: `training/scaffold/go.mod`
- Create: `training/scaffold/cmd/server/main.go`
- Create: `training/scaffold/internal/domain/entity/.gitkeep`
- Create: `training/scaffold/internal/usecase/.gitkeep`
- Create: `training/scaffold/internal/adapter/in/http/.gitkeep`
- Create: `training/scaffold/internal/adapter/in/http/dto/request/.gitkeep`
- Create: `training/scaffold/internal/adapter/in/http/dto/response/.gitkeep`
- Create: `training/scaffold/internal/adapter/out/repository/.gitkeep`
- Create: `training/scaffold/internal/adapter/presenter/.gitkeep`
- Create: `training/scaffold/internal/infrastructure/config/config.go`
- Create: `training/scaffold/internal/infrastructure/server/server.go`
- Create: `training/scaffold/docker-compose.yaml`
- Create: `training/scaffold/Makefile`
- Create: `training/scaffold/README.md`

- [ ] **Step 1: Write go.mod**

```
module kfc-training

go 1.25

require github.com/gin-gonic/gin v1.10.0
```

Note: Exact dependency versions will be resolved when the trainee runs `go mod tidy`.

- [ ] **Step 2: Write cmd/server/main.go**

```go
package main

import (
	"log/slog"
	"os"

	"kfc-training/internal/infrastructure/config"
	"kfc-training/internal/infrastructure/server"
)

func main() {
	// 1. 載入設定
	cfg := config.Load()

	// 2. 設定結構化日誌
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// 3. 建立並啟動 HTTP 伺服器
	// TODO: 建立 DI Container，將依賴注入到 server
	srv := server.New(cfg.Port)

	slog.Info("server starting", "port", cfg.Port)
	if err := srv.Run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Write infrastructure/config/config.go**

```go
package config

import "os"

type Config struct {
	Port     string
	MongoURI string
	MongoDB  string
}

func Load() *Config {
	return &Config{
		Port:     getEnv("SERVER_PORT", "8080"),
		MongoURI: getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:  getEnv("MONGO_DB", "kfc_training"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
```

- [ ] **Step 4: Write infrastructure/server/server.go**

```go
package server

import (
	"github.com/gin-gonic/gin"
)

type Server struct {
	port   string
	router *gin.Engine
}

func New(port string) *Server {
	r := gin.Default()

	// API v1 路由群組
	v1 := r.Group("/api/v1")
	_ = v1 // TODO: 在這裡註冊你的 Handler 路由

	return &Server{
		port:   port,
		router: r,
	}
}

func (s *Server) Run() error {
	return s.router.Run(":" + s.port)
}
```

- [ ] **Step 5: Write docker-compose.yaml**

```yaml
services:
  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  mongo_data:
```

- [ ] **Step 6: Write Makefile**

```makefile
.PHONY: run test lint clean

run:
	go run ./cmd/server/

test:
	go test ./... -v

lint:
	golangci-lint run ./...

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

infra-up:
	docker compose up -d

infra-down:
	docker compose down
```

- [ ] **Step 7: Write README.md**

```markdown
# KFC Training — Item Service

這是 KFC Project 的訓練練習專案。你將在這個專案中，用 Clean Architecture 從零建立一個「物品管理服務」。

## 快速開始

### 啟動基礎設施（MongoDB + Redis）

    make infra-up

### 啟動服務

    make run

### 執行測試

    make test

## 目錄結構

    internal/
    ├── domain/entity/         ← Domain Layer：定義你的 Entity
    ├── usecase/               ← UseCase Layer：業務邏輯
    ├── adapter/
    │   ├── in/http/           ← 入站：HTTP Handler + DTO
    │   └── out/repository/    ← 出站：MongoDB Repository
    └── infrastructure/
        ├── config/            ← 環境變數設定（已完成）
        └── server/            ← Gin Server 啟動（已完成）

## 訓練進度

請回到 Game-Server-Wiki repo 執行 `/train` 查看你的訓練進度和下一步任務。
```

- [ ] **Step 8: Create .gitkeep files for empty directories**

Create empty `.gitkeep` files in:
- `training/scaffold/internal/domain/entity/.gitkeep`
- `training/scaffold/internal/usecase/.gitkeep`
- `training/scaffold/internal/adapter/in/http/.gitkeep`
- `training/scaffold/internal/adapter/in/http/dto/request/.gitkeep`
- `training/scaffold/internal/adapter/in/http/dto/response/.gitkeep`
- `training/scaffold/internal/adapter/out/repository/.gitkeep`
- `training/scaffold/internal/adapter/presenter/.gitkeep`

- [ ] **Step 9: Commit**

```bash
git add training/scaffold/
git commit -m "feat(training): add practice project scaffold (Item Service)"
```

---

## Chunk 5: Agent Skills and Rules

### Task 9: Create training-context.md (global rules)

**Files:**
- Create: `.claude/rules/training-context.md`

- [ ] **Step 1: Write training-context.md**

```markdown
# Training Agent Context

## 身份

你是 KFC Project 的訓練導師 agent。你的目標是引導 junior developer 透過閱讀 wiki + 動手實作，達到能獨立開發的程度。

## 核心原則：絕對不直接給答案

### 知識問答時

- Junior 回答不完整 → 提供引導性提示（相關章節段落、類比說明、反問），讓 junior 自己推導出答案
- Junior 回答錯誤 → 指出矛盾點或與章節內容的衝突，引導重新思考
- Junior 完全卡住 → 提示去重讀哪個章節的哪個段落，但不直接說出答案
- **可以給**：概念解釋、思考方向、章節參考、類比
- **不可以給**：題目的完整答案、直接告訴 junior 正確選項

### 引導策略

1. 先問 junior 自己的理解是什麼
2. 方向正確但不完整 → 追問細節（「你提到了 X，那 Y 呢？」）
3. 方向錯誤 → 指出矛盾點（「你說的和第 N 章描述的 X 概念有衝突，想想看為什麼？」）
4. 完全卡住 → 給出最小提示，引導重讀特定段落

### 實作 review 時

- 指出問題的方向和原因，不直接給修正後的 code
- **可以給**：概念提示、設計模式參考、相關 wiki 章節、錯誤方向的描述
- **不可以給**：完整實作 code、直接幫 junior 改 code、複製貼上的解答

## 進度管理

- 讀取 `trainees/<name>/progress.yaml` 了解當前進度
- 每次 task 狀態變更都要更新 `progress.yaml` 並 commit 到 trainee 的 branch
- 階段必須按順序完成，不可跳過必修
- 選修階段（6, 8）可以跳過

## Mentor Checkpoint

- `checkpoint: true` 的階段（Stage 7），agent review 完成後狀態設為 `awaiting_mentor`
- Agent 不可自行將 `awaiting_mentor` 改為 `passed`
- Mentor 在 `trainees/<name>/reviews/stage-<id>-review.md` 標記 approved 後才算通過

## 練習專案位置

- 練習專案在 wiki repo 的上層目錄：`../kfc-training-<name>/`
- 實作 review 時讀取該目錄的檔案

## 問答紀錄

- 每章的問答紀錄存在 `trainees/<name>/answers/<chapter>.md`
- 格式包含：題目、junior 的回答、是否通過、agent 的回饋
```

- [ ] **Step 2: Commit**

```bash
git add .claude/rules/training-context.md
git commit -m "feat(training): add training agent context rules"
```

---

### Task 10: Create /init skill

**Files:**
- Create: `.claude/skills/init.md`

- [ ] **Step 1: Write init.md**

```markdown
---
name: init
description: 初始化 junior developer 的訓練環境（進度 folder + branch + 練習專案）
---

# /init — 訓練環境初始化

## 執行步驟

### 1. 檢查是否已初始化

讀取 `trainees/` 目錄，檢查是否已有該使用者的目錄。如果已存在，提示使用者：

> 你的訓練環境已經存在（`trainees/<name>/`）。如果要重新開始，請先手動刪除該目錄和對應的 branch。

若已初始化則停止，不重複建立。

### 2. 詢問名字

用 AskUserQuestion 詢問 junior 的名字（英文，用於目錄和 branch 命名）：

> 歡迎加入 KFC Project 訓練計畫！請輸入你的英文名字（用於建立你的訓練進度目錄和 branch）：

### 3. 建立 branch

```bash
git checkout -b trainee/<name>
```

### 4. 建立進度目錄結構

在 wiki repo 中建立：

```
trainees/<name>/
├── progress.yaml    ← 從模板生成，填入名字和日期
├── answers/         ← 空目錄（.gitkeep）
└── reviews/         ← 空目錄（.gitkeep）
```

`progress.yaml` 根據 `training/curriculum.yaml` 生成，包含所有階段和 task 的初始狀態。Stage 1 狀態為 `in_progress`，其餘為 `pending`。

### 5. 生成練習專案

將 `training/scaffold/` 的內容複製到 `../kfc-training-<name>/`：

```bash
cp -r training/scaffold/ ../kfc-training-<name>/
```

在練習專案目錄中初始化 git：

```bash
cd ../kfc-training-<name>
git init
git add .
git commit -m "feat: init training project scaffold"
```

### 6. 回到 wiki repo，commit 並 push

```bash
cd <wiki-repo>
git add trainees/<name>/
git commit -m "feat(training): init trainee <name>"
git push -u origin trainee/<name>
```

### 7. 歡迎訊息

顯示：

> 訓練環境已建立完成！
>
> - 進度追蹤：`trainees/<name>/progress.yaml`
> - 練習專案：`../kfc-training-<name>/`
> - 目前階段：**Stage 1 — 全局觀**
>
> 請先閱讀以下章節：
> 1. `junior-developer-guide/00-導讀指南.md`
> 2. `junior-developer-guide/01-專案總覽.md`
> 3. `junior-developer-guide/02-Clean-Architecture.md`
> 4. `junior-developer-guide/03-目錄結構與分層.md`
>
> 閱讀完成後，執行 `/train` 開始測驗。
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/init.md
git commit -m "feat(training): add /init skill for trainee onboarding"
```

---

### Task 11: Create /train skill

**Files:**
- Create: `.claude/skills/train.md`

- [ ] **Step 1: Write train.md**

```markdown
---
name: train
description: 訓練互動入口 — 根據進度自動導向問答、實作指派或 review
---

# /train — 訓練互動

## 前置檢查

1. 找到當前 trainee 的 `progress.yaml`。如果 `trainees/` 下只有一個 trainee，直接使用；如果有多個，詢問名字。
2. 如果找不到任何 trainee 目錄，提示執行 `/init`。

## 路由邏輯

讀取 `progress.yaml` 的 `current_stage`，根據該階段的類型執行對應流程。

### Quiz 型階段（Stage 1, 2, 4, 6）

1. **檢查未讀章節**：找 `chapters` 中 `read: false` 的章節
   - 如果有 → 提示 junior 先閱讀該章節，給出檔案路徑和重點導讀（用 2-3 句話概述該章核心內容）
   - 詢問 junior 是否已讀完，確認後標記 `read: true`

2. **檢查未通過章節**：找 `chapters` 中 `quiz_passed: false` 的章節
   - 讀取 `training/questions/<chapter>.yaml`
   - 依序出核心必考題：
     - 一次出一題
     - Junior 回答後，根據 `pass_criteria` 判定
     - 通過 → 記錄到 `answers/<chapter>.md`，下一題
     - 不完整 → 釋放 `hints`（逐個，從第一個開始），引導補充。**絕對不直接說出答案**
     - 錯誤 → 指出矛盾點，引導重新思考
   - 核心題全部通過後，agent 根據章節內容動態出 1-2 題延伸題
   - 該章通過 → 標記 `quiz_passed: true`

3. **所有章節通過**：
   - 標記階段 `status: passed`
   - 推進 `current_stage` 到下一階段
   - 更新 `progress.yaml`，commit & push

### Implementation 型階段（Stage 3, 5）

1. **檢查 pending task**：
   - 讀取 `curriculum.yaml` 中該階段的 task 定義
   - 找到第一個 `status: pending` 的 task
   - 說明任務需求（`description`）和驗收標準（`acceptance`）
   - 提供引導方向，**不給完整 code**
   - 標記該 task 為 `in_progress`

2. **檢查 in_progress task**：
   - 讀取練習專案（`../kfc-training-<name>/`）中對應的檔案
   - 根據 `acceptance` 標準 review：
     - 全部通過 → 標記 `status: completed`，給予正面回饋
     - 部分不通過 → 指出問題方向，提供引導，**不直接給修正 code**
   - 更新 `progress.yaml`

3. **所有 task 完成**：
   - 讀取 `training/checkpoints.yaml` 判斷是否需要 mentor review
   - 不需要 → 標記階段 `status: passed`，推進下一階段
   - 需要 → 標記階段 `status: awaiting_mentor`，提示 junior 通知 mentor

### Mixed 型階段（Stage 7）

先執行 Quiz 流程（chapter 14），再執行 Implementation 流程（test tasks）。

### Reading 型階段（Stage 8）

提示 junior 閱讀該章節。確認後標記 `status: passed`（或 `skipped`）。

### 選修階段跳過

如果 `current_stage` 對應的階段 `required: false`，詢問 junior：

> 這是選修階段「<name>」。你可以現在學習，或暫時跳過先完成必修。要跳過嗎？

跳過 → 標記 `status: skipped`，推進下一階段。

### 畢業

當所有 `graduation_required` 階段都是 `passed` 時：

> 🎓 恭喜畢業！你已經完成了 KFC Project 的 Junior Developer 訓練計畫。
>
> 你的訓練成果：
> - 知識問答：X 個章節通過
> - 實作完成：Item Service CRUD + 測試
> - 總耗時：N 天
>
> 接下來你可以：
> 1. 回去完成選修章節（可觀測性、部署）
> 2. 開始在正式專案上貢獻
> 3. 和 mentor 討論你的下一步成長方向

## 進度持久化

每次狀態變更後：

```bash
# 在 wiki repo 中
cd <wiki-repo>
git add trainees/<name>/
git commit -m "progress(<name>): <description of change>"
git push
```

## 問答紀錄格式

`trainees/<name>/answers/<chapter>.md`：

```markdown
# Chapter <chapter> — <title>

## 核心題

### <question-id>

**題目：** <question text>

**回答：** <junior's answer>

**結果：** ✅ 通過 / ❌ 需補充

**回饋：** <agent feedback>

---

## 延伸題

### ext-1

**題目：** <dynamically generated question>

**回答：** <junior's answer>

**結果：** ✅ / ❌

**回饋：** <agent feedback>
```
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/train.md
git commit -m "feat(training): add /train skill for training interactions"
```

---

## Chunk 6: Final Integration

### Task 12: Update CLAUDE.md with training system info

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Add training system section to CLAUDE.md**

Append the following section to the existing CLAUDE.md:

```markdown

## Training System

This repo includes an agent-driven training system for junior developers.

### Key Commands
- `/init` — Initialize a trainee's environment (progress folder, branch, practice project)
- `/train` — Main training interaction (quiz, implementation tasks, review)

### Training Structure
- `training/curriculum.yaml` — Course definition (8 stages, required/optional)
- `training/questions/*.yaml` — Core quiz questions per chapter
- `training/checkpoints.yaml` — Mentor review checkpoint configuration
- `training/scaffold/` — Practice project template (Go + Gin)
- `trainees/<name>/` — Per-trainee progress, answers, and reviews (on trainee branches)

### Agent Behavior
- The training agent follows rules in `.claude/rules/training-context.md`
- Never gives direct answers to quiz questions — uses hints and guided questioning
- Never provides complete implementation code — gives directional feedback
- Progress is tracked in `progress.yaml` and committed to the trainee's branch
```

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add training system section to CLAUDE.md"
```

---

### Task 13: Create .gitignore for trainees

**Files:**
- Create: `.gitignore` (or modify if exists)

- [ ] **Step 1: Create .gitignore**

```gitignore
# Practice project repos are outside this repo
# but ensure no OS/editor artifacts are committed
.DS_Store
Thumbs.db
*.swp
*.swo
*~
```

- [ ] **Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: add .gitignore"
```
