package ginx

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Server struct {
	*gin.Engine
	Addr string
}

func (s *Server) Start() error {
	return s.Engine.Run(s.Addr)
}

// Result 你可以通过在 Result 里面定义更加多的字段，来配合 Wrap 方法
type Result struct {
	Code int    `json:"code"` // 错误码，非 0 表示失败
	Msg  string `json:"msg"`  // 错误或成功 描述
	Data any    `json:"data"`
}

type UserClaims struct {
	Id        int64
	UserAgent string
	Ssid      string
	jwt.RegisteredClaims
}
