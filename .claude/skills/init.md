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
