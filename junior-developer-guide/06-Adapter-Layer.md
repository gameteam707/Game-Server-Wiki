# 06 — Adapter Layer（轉接層）

## 這一層是什麼？

Adapter Layer 是系統與外部世界之間的 **翻譯官**。它負責：
- 把外部的資料格式（JSON、BSON、Protobuf）轉換成內部的 Domain Entity
- 把內部的 Domain Entity 轉換成外部的資料格式
- 實作 UseCase 定義的 Port（介面）

## 位置與分類

```
internal/adapter/
├── in/          ← 入站轉接器：外部世界 → 系統
│   ├── http/    # HTTP Handler（接收 REST API 請求）
│   └── grpc/    # gRPC Handler（接收 RPC 呼叫）
├── out/         ← 出站轉接器：系統 → 外部世界
│   ├── repository/   # 資料庫操作
│   ├── gateway/      # 呼叫外部 API
│   └── messaging/    # 發送訊息（Kafka）
└── presenter/   ← 回應格式化
```

## 入站轉接器（in/）

### HTTP Handler

Handler 的職責很單純：**解析請求 → 呼叫 UseCase → 格式化回應**。

```go
// 檔案：admin-backend/server/internal/adapter/in/http/game_handler.go

type GameHandler struct {
    createUC game_mgmt.CreateUseCase
    listUC   game_mgmt.ListUseCase
}

func NewGameHandler(createUC game_mgmt.CreateUseCase, listUC game_mgmt.ListUseCase) *GameHandler {
    return &GameHandler{createUC: createUC, listUC: listUC}
}

func (h *GameHandler) Create(c *gin.Context) {
    // Step 1: 解析 HTTP Request → DTO
    var req request.CreateGameRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        presenter.RespondError(c, 400, "INVALID_REQUEST", err.Error())
        return
    }

    // Step 2: DTO → UseCase Command（翻譯）
    cmd := &game_mgmt.CreateCommand{
        Code:       req.Code,
        NameI18n:   req.NameI18n,
        GameTypeID: req.GameTypeID,
        Tags:       req.Tags,
        RTP:        req.RTP,
    }

    // Step 3: 呼叫 UseCase
    result, err := h.createUC.Execute(c.Request.Context(), cmd)
    if err != nil {
        presenter.RespondError(c, 500, "CREATE_FAILED", err.Error())
        return
    }

    // Step 4: Entity → Response DTO → HTTP Response
    presenter.RespondSuccess(c, 201, response.FromGameEntity(result.Game))
}
```

**Handler 裡面不應該有業務邏輯！** 如果你發現 Handler 裡面有 `if` 判斷業務規則、有資料庫操作，那就是放錯地方了。

### DTO（Data Transfer Object）

DTO 是 HTTP 層專屬的資料結構，帶有 `json` 標籤。

```go
// 檔案：adapter/in/http/dto/request/game.go

type CreateGameRequest struct {
    Code       string            `json:"code" binding:"required"`
    NameI18n   map[string]string `json:"nameI18n" binding:"required"`
    GameTypeID string            `json:"gameTypeId" binding:"required"`
    Tags       []string          `json:"tags"`
    RTP        float64           `json:"rtp" binding:"required,gt=0,lte=100"`
}
```

```go
// 檔案：adapter/in/http/dto/response/game.go

type GameResponse struct {
    ID         string            `json:"id"`
    Code       string            `json:"code"`
    NameI18n   map[string]string `json:"nameI18n"`
    GameTypeID string            `json:"gameTypeId"`
    Status     string            `json:"status"`
    CreatedAt  string            `json:"createdAt"`
}

// Entity → Response 的轉換函式
func FromGameEntity(g *entity.Game) *GameResponse {
    return &GameResponse{
        ID:         g.ID,
        Code:       g.Code,
        NameI18n:   g.NameI18n,
        GameTypeID: g.GameTypeID,
        Status:     string(g.Status),
        CreatedAt:  g.CreatedAt.Format(time.RFC3339),
    }
}
```

### gRPC Handler（gameservice）

```go
// 檔案：sfc-stream-game/gameservice/internal/adapter/in/grpc/server.go

type GameServer struct {
    gamepb.UnimplementedGameServiceServer
    spinUC     spin.UseCase
    initGameUC init_game.UseCase
    balanceUC  balance.UseCase
}

func (s *GameServer) Spin(ctx context.Context, req *gamepb.SpinRequest) (*gamepb.SpinResponse, error) {
    // Protobuf → UseCase Command
    cmd := &spin.Command{
        SessionID: req.SessionId,
        BetAmount: req.BetAmount,
        // ...
    }

    result, err := s.spinUC.Execute(ctx, cmd)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "spin failed: %v", err)
    }

    // UseCase Result → Protobuf Response
    return &gamepb.SpinResponse{
        WinAmount: result.WinAmount,
        // ...
    }, nil
}
```

## 出站轉接器（out/）

### Repository — 資料庫操作

Repository 實作 UseCase 定義的 Port 介面。

```go
// 檔案：adapter/out/repository/mongo/game_repo.go

type gameRepository struct {
    collection *mongo.Collection
}

func NewGameRepository(db *mongo.Database) ports.GameRepository {
    return &gameRepository{
        collection: db.Collection("games"),
    }
}

// 實作 ports.GameRepository 介面
func (r *gameRepository) Create(ctx context.Context, game *entity.Game) error {
    doc := r.toDocument(game)    // Entity → MongoDB Document
    _, err := r.collection.InsertOne(ctx, doc)
    return err
}

func (r *gameRepository) FindByID(ctx context.Context, id string) (*entity.Game, error) {
    var doc gameDocument
    err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
    if err != nil {
        return nil, err
    }
    return r.toEntity(&doc), nil  // MongoDB Document → Entity
}
```

### MongoDB Document Model

```go
// 檔案：adapter/out/repository/mongo/models.go

// BSON 標籤只出現在這裡，不會污染 Domain Entity
type gameDocument struct {
    ID         string            `bson:"_id"`
    Code       string            `bson:"code"`
    NameI18n   map[string]string `bson:"nameI18n"`
    GameTypeID string            `bson:"gameTypeId"`
    Tags       []string          `bson:"tags"`
    RTP        float64           `bson:"rtp"`
    Status     string            `bson:"status"`
    IconURL    string            `bson:"iconUrl"`
    CreatedAt  time.Time         `bson:"createdAt"`
    UpdatedAt  time.Time         `bson:"updatedAt"`
}

// 轉換函式
func (r *gameRepository) toDocument(e *entity.Game) *gameDocument {
    return &gameDocument{
        ID:         e.ID,
        Code:       e.Code,
        NameI18n:   e.NameI18n,
        // ...
    }
}

func (r *gameRepository) toEntity(d *gameDocument) *entity.Game {
    return &entity.Game{
        ID:         d.ID,
        Code:       d.Code,
        NameI18n:   d.NameI18n,
        // ...
    }
}
```

### Gateway — 呼叫外部 API

```go
// 檔案：adapter/out/gateway/gameapi_client.go

type GameAPIClient struct {
    baseURL    string
    httpClient *http.Client
}

func (c *GameAPIClient) GetBetRecords(ctx context.Context, filter BetFilter) ([]BetRecord, error) {
    url := fmt.Sprintf("%s/api/v1/bet-records?page=%d", c.baseURL, filter.Page)
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data []BetRecord `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    return result.Data, nil
}
```

### Presenter — 統一回應格式

```go
// 檔案：adapter/presenter/response.go

func RespondSuccess(c *gin.Context, statusCode int, data interface{}) {
    c.JSON(statusCode, gin.H{
        "success": true,
        "data":    data,
    })
}

func RespondError(c *gin.Context, statusCode int, code string, msg string) {
    c.JSON(statusCode, gin.H{
        "success": false,
        "error":   msg,
        "code":    code,
    })
}

func RespondList(c *gin.Context, data interface{}, total int64, page, pageSize int) {
    c.JSON(200, gin.H{
        "success": true,
        "data":    data,
        "meta": gin.H{
            "total":    total,
            "page":     page,
            "pageSize": pageSize,
        },
    })
}
```

## 轉換流程總覽

```
HTTP Request (JSON)
     │
     ▼
Request DTO (json tags)  ──  adapter/in/http/dto/request/
     │
     ▼ (手動轉換)
UseCase Command          ──  usecase/{feature}/contract.go
     │
     ▼ (UseCase 處理)
Domain Entity            ──  domain/entity/
     │
     ▼ (Repository 轉換)
MongoDB Document (bson tags) ── adapter/out/repository/mongo/models.go
     │
     ▼ (存取完成，回傳)
Domain Entity
     │
     ▼ (手動轉換)
Response DTO (json tags) ──  adapter/in/http/dto/response/
     │
     ▼
HTTP Response (JSON)
```

每一次轉換都是刻意為之——確保每一層的資料結構只為自己的關注點服務。

## 給 Junior 的學習重點

### 你需要理解的概念

1. **DTO vs Entity vs Document**：三者各自服務不同的層，不要共用
2. **介面實作**：Go 的隱式介面——只要你的 struct 有實作介面的所有方法，就自動滿足介面
3. **Adapter Pattern**：經典的設計模式，用一個轉接層讓不相容的介面可以一起工作

### 動手練習

1. 追蹤一個完整的 CRUD 請求，從 Handler → DTO → Command → UseCase → Port → Repository → Document → MongoDB
2. 比較 Request DTO、Entity、MongoDB Document 三者的欄位差異
3. 嘗試為一個新的 Entity 寫出完整的 Adapter 層（Handler + DTO + Repository）
