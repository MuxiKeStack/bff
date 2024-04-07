package web

import "github.com/gin-gonic/gin"

type handler interface {
	RegisterRoutes(s *gin.Engine, AuthMiddleware gin.HandlerFunc)
}
