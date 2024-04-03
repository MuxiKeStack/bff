package ijwt

import (
	"github.com/gin-gonic/gin"
)

//go:generate mockgen -source=./types.go -package=ijwtmocks -destination=./mocks/ijwt.mock.go Handler
type Handler interface {
	ClearToken(ctx *gin.Context) error
	ExtractToken(ctx *gin.Context) string
	SetLoginToken(ctx *gin.Context, uid int64) error
	SetJWTToken(ctx *gin.Context, uid int64, ssid string, userAgent string) error
	CheckSession(ctx *gin.Context, ssid string) (bool, error)
	JWTKey() []byte
	RCJWTKey() []byte
}
