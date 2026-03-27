# 10 — 資料庫與 Repository Pattern

## 為什麼需要 Repository Pattern？

Repository Pattern 的核心價值是 **把資料存取的細節隱藏在介面背後**。

```go
// UseCase 只看到這個介面
type GameRepository interface {
    Create(ctx context.Context, game *entity.Game) error
    FindByID(ctx context.Context, id string) (*entity.Game, error)
    List(ctx context.Context, filter GameFilter) ([]*entity.Game, int64, error)
}

// UseCase 不知道也不關心：
// - 用的是 MongoDB 還是 PostgreSQL
// - Collection/Table 叫什麼名字
// - 查詢語法長什麼樣子
```

## MongoDB 基礎概念

本專案使用 MongoDB 作為主要資料庫。如果你之前只用過 SQL 資料庫，這裡是快速對照：

| SQL 概念 | MongoDB 概念 | 說明 |
|----------|-------------|------|
| Database | Database | 資料庫 |
| Table | Collection | 資料表/集合 |
| Row | Document | 一筆資料 |
| Column | Field | 欄位 |
| PRIMARY KEY | `_id` | 唯一識別 |
| JOIN | Embedding / Reference | 關聯方式不同 |
| Schema | Flexible | MongoDB 不強制 Schema |

### MongoDB Document 範例

```json
// games collection 中的一筆 document
{
    "_id": "game_001",
    "code": "slot-fortune",
    "nameI18n": {
        "zh-TW": "幸運老虎機",
        "en": "Fortune Slot",
        "ja": "フォーチュンスロット"
    },
    "gameTypeId": "type_slot",
    "tags": ["hot", "new"],
    "rtp": 96.5,
    "status": "active",
    "iconUrl": "/uploads/slot-fortune.png",
    "createdAt": "2024-01-15T08:30:00Z",
    "updatedAt": "2024-03-20T14:22:00Z"
}
```

## Entity → Document → Entity 轉換

這是本專案最重要的模式之一。三層資料結構各司其職：

```
Domain Entity (零標籤)
     ↕ toDocument() / toEntity()
MongoDB Document (bson 標籤)
     ↕ MongoDB Driver 自動處理
MongoDB 資料庫
```

### 實際程式碼

```go
// === Domain Entity（domain/entity/game.go）===
type Game struct {
    ID         string
    Code       string
    NameI18n   map[string]string
    GameTypeID string
    Tags       []string
    RTP        float64
    Status     GameStatus
    IconURL    string
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// === MongoDB Document（adapter/out/repository/mongo/models.go）===
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

// === 轉換函式 ===
func (r *gameRepository) toDocument(e *entity.Game) *gameDocument {
    return &gameDocument{
        ID:         e.ID,
        Code:       e.Code,
        NameI18n:   e.NameI18n,
        GameTypeID: e.GameTypeID,
        Tags:       e.Tags,
        RTP:        e.RTP,
        Status:     string(e.Status),
        IconURL:    e.IconURL,
        CreatedAt:  e.CreatedAt,
        UpdatedAt:  e.UpdatedAt,
    }
}

func (r *gameRepository) toEntity(d *gameDocument) *entity.Game {
    return &entity.Game{
        ID:         d.ID,
        Code:       d.Code,
        NameI18n:   d.NameI18n,
        GameTypeID: d.GameTypeID,
        Tags:       d.Tags,
        RTP:        d.RTP,
        Status:     entity.GameStatus(d.Status),
        IconURL:    d.IconURL,
        CreatedAt:  d.CreatedAt,
        UpdatedAt:  d.UpdatedAt,
    }
}
```

## Repository 實作

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

// ── Create ──
func (r *gameRepository) Create(ctx context.Context, game *entity.Game) error {
    doc := r.toDocument(game)
    _, err := r.collection.InsertOne(ctx, doc)
    return err
}

// ── FindByID ──
func (r *gameRepository) FindByID(ctx context.Context, id string) (*entity.Game, error) {
    var doc gameDocument
    err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return nil, nil  // 找不到不算錯誤，回傳 nil
        }
        return nil, err
    }
    return r.toEntity(&doc), nil
}

// ── List（分頁查詢）──
func (r *gameRepository) List(ctx context.Context, filter ports.GameFilter) ([]*entity.Game, int64, error) {
    // 建立查詢條件
    query := bson.M{}
    if filter.Status != nil {
        query["status"] = *filter.Status
    }
    if filter.GameTypeID != nil {
        query["gameTypeId"] = *filter.GameTypeID
    }

    // 計算總數
    total, err := r.collection.CountDocuments(ctx, query)
    if err != nil {
        return nil, 0, err
    }

    // 分頁查詢
    opts := options.Find().
        SetSkip(int64((filter.Page - 1) * filter.PageSize)).
        SetLimit(int64(filter.PageSize)).
        SetSort(bson.M{"createdAt": -1})  // 按建立時間倒序

    cursor, err := r.collection.Find(ctx, query, opts)
    if err != nil {
        return nil, 0, err
    }
    defer cursor.Close(ctx)

    // 解碼所有結果
    var docs []gameDocument
    if err := cursor.All(ctx, &docs); err != nil {
        return nil, 0, err
    }

    // 轉換為 Entity
    games := make([]*entity.Game, len(docs))
    for i, doc := range docs {
        games[i] = r.toEntity(&doc)
    }

    return games, total, nil
}

// ── Update ──
func (r *gameRepository) Update(ctx context.Context, game *entity.Game) error {
    doc := r.toDocument(game)
    _, err := r.collection.ReplaceOne(ctx, bson.M{"_id": game.ID}, doc)
    return err
}

// ── Delete ──
func (r *gameRepository) Delete(ctx context.Context, id string) error {
    _, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
    return err
}
```

## Redis 的使用場景

本專案中 Redis 不是主要資料庫，而是用於特定場景：

### 登入限流（Login Rate Limiting）

```go
// 檔案：adapter/out/provider/login_limiter.go

type RedisLoginLimiter struct {
    client *redis.Client
}

func (l *RedisLoginLimiter) RecordFailure(ctx context.Context, ip string) {
    key := fmt.Sprintf("login:fail:%s", ip)
    l.client.Incr(ctx, key)
    l.client.Expire(ctx, key, 15*time.Minute)  // 15 分鐘後自動清除
}

func (l *RedisLoginLimiter) IsBlocked(ctx context.Context, ip string) bool {
    key := fmt.Sprintf("login:fail:%s", ip)
    count, _ := l.client.Get(ctx, key).Int()
    return count >= 5  // 連續失敗 5 次就封鎖
}

func (l *RedisLoginLimiter) ClearFailures(ctx context.Context, ip string) {
    key := fmt.Sprintf("login:fail:%s", ip)
    l.client.Del(ctx, key)
}
```

**為什麼用 Redis 而不是 MongoDB？**
- 登入限流需要極高的讀寫速度
- 資料是短暫的（15 分鐘後自動過期）
- Redis 原生支援 TTL（Time To Live）

## MongoDB Change Streams（config-service）

config-service 使用了 MongoDB 的特殊功能——Change Streams，即時監聽資料變更：

```go
// 檔案：config-service/internal/adapter/out/repository/mongo/change_stream.go

func WatchChanges(ctx context.Context, db *mongo.Database, hub *sse.Hub) {
    // 監聽指定的 collections
    collections := []string{"integrators", "games", "currencies"}

    for _, collName := range collections {
        go func(name string) {
            coll := db.Collection(name)
            stream, _ := coll.Watch(ctx, mongo.Pipeline{})
            defer stream.Close(ctx)

            for stream.Next(ctx) {
                // 有資料變更時，透過 SSE Hub 廣播
                hub.Broadcast(sse.Event{
                    Type: name + "_updated",
                    Data: stream.Current.String(),
                })
            }
        }(collName)
    }
}
```

## 本專案的 Collections 一覽

| Collection | 服務 | 用途 |
|-----------|------|------|
| `admin_users` | admin-backend | 管理員帳號 |
| `games` | admin-backend | 遊戲設定 |
| `game_types` | admin-backend | 遊戲類型 |
| `game_tags` | admin-backend | 遊戲標籤 |
| `currencies` | admin-backend | 幣別與匯率 |
| `integrators` | admin-backend | 整合商 |
| `integrator_games` | admin-backend | 整合商-遊戲關聯 |
| `platform_endpoint_configs` | admin-backend | 平台端點設定 |
| `game_maintenances` | admin-backend | 維護排程 |
| `permission_groups` | admin-backend | 權限群組 |
| `whitelists` | admin-backend | IP 白名單 |
| `audit_logs` | admin-backend | 審計日誌 |
| `notifications` | admin-backend | 通知 |
| `languages` | admin-backend | 語系 |
| `bet_records` | gameservice | 下注紀錄 |
| `users` | gameservice | 玩家資料 |

## 給 Junior 的學習重點

### 你需要理解的概念

1. **Repository Pattern**：隔離資料存取細節
2. **MongoDB CRUD 操作**：InsertOne、FindOne、Find、ReplaceOne、DeleteOne
3. **分頁查詢**：Skip + Limit 的模式
4. **`context.Context`**：Go 的請求上下文，用於超時控制和取消操作

### 動手練習

1. 選一個 Repository（如 `currency_repo.go`），完整閱讀所有方法
2. 用 MongoDB Compass 連線到本地資料庫，觀察 document 的結構
3. 嘗試寫一個新的 Repository 方法（如「按狀態統計遊戲數量」）

### 延伸閱讀

- [MongoDB Go Driver v2 官方文件](https://www.mongodb.com/docs/drivers/go/current/)
- Martin Fowler, [Repository Pattern](https://martinfowler.com/eaaCatalog/repository.html)
