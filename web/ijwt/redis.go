package ijwt

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

type RedisJWTHandler struct {
	cmd           redis.Cmdable
	signingMethod jwt.SigningMethod
	rcExpiration  time.Duration
	jwtKey        string
	rcJWTKey      string
}

func (r *RedisJWTHandler) JWTKey() string {
	return r.jwtKey
}

func (r *RedisJWTHandler) RCJWTKey() string {
	return r.rcJWTKey
}

func (r *RedisJWTHandler) ClearToken(ctx *gin.Context) error {
	// 要求客户端设置为空
	ctx.Header("x-jwt-token", "")
	ctx.Header("x-refresh-token", "")
	// 在session上记录已过期
	uc := ctx.MustGet("user").(UserClaims)
	return r.cmd.Set(ctx, fmt.Sprintf("users:ssid:%s", uc.Ssid), "", r.rcExpiration).Err()
}

func (r *RedisJWTHandler) ExtractToken(ctx *gin.Context) string {
	authCode := ctx.GetHeader("Authorization")
	if authCode == "" {
		return ""
	}
	segs := strings.Split(authCode, " ")
	if len(segs) != 2 {
		return ""
	}
	return segs[1]
}

func (r *RedisJWTHandler) SetLoginToken(ctx *gin.Context, uid int64) error {
	ssid := uuid.New().String()
	err := r.setRefreshToken(ctx, uid, ssid)
	if err != nil {
		return err
	}
	return r.SetJWTToken(ctx, uid, ssid)
}

func (r *RedisJWTHandler) setRefreshToken(ctx *gin.Context, uid int64, ssid string) error {
	rc := RefreshClaims{
		Uid:  uid,
		Ssid: ssid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(r.rcExpiration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES512, rc)
	tokenStr, err := token.SignedString(r.RCJWTKey())
	if err != nil {
		return err
	}
	ctx.Header("x-refresh-token", tokenStr)
	return nil
}

func (r *RedisJWTHandler) SetJWTToken(ctx *gin.Context, uid int64, ssid string) error {
	uc := UserClaims{
		Uid:       uid,
		Ssid:      ssid,
		UserAgent: ctx.GetHeader("User-Agent"),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES512, uc)
	tokenStr, err := token.SignedString(r.JWTKey())
	if err != nil {
		return err
	}
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

// 布隆过滤器可以优化-> 询问得到不在就确定未退出登录  询问在还是需要redis兜底进一步确定状态
func (r *RedisJWTHandler) CheckSession(ctx *gin.Context, ssid string) (bool, error) {
	val, err := r.cmd.Exists(ctx, fmt.Sprintf("users:ssid:%s", ssid)).Result()
	return val > 0, err
}

func NewRedisJWTHandler(cmd redis.Cmdable, jwtKey string, rcJWTKey string) Handler {
	return &RedisJWTHandler{
		cmd:           cmd,
		signingMethod: jwt.SigningMethodES512,
		rcExpiration:  time.Hour * 24 * 7,
		jwtKey:        jwtKey,
		rcJWTKey:      rcJWTKey,
	}
}

type UserClaims struct {
	jwt.RegisteredClaims
	Uid       int64
	Ssid      string
	UserAgent string
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Uid  int64
	Ssid string
}
