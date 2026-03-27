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
