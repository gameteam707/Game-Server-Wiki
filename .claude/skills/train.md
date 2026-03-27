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

> 恭喜畢業！你已經完成了 KFC Project 的 Junior Developer 訓練計畫。
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

**結果：** 通過 / 需補充

**回饋：** <agent feedback>

---

## 延伸題

### ext-1

**題目：** <dynamically generated question>

**回答：** <junior's answer>

**結果：** 通過 / 需補充

**回饋：** <agent feedback>
```
