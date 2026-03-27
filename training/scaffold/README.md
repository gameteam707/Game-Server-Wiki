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
