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
