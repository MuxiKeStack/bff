package middleware

import (
	"github.com/big-dust/ZhiJing-BE/internal/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
)

type LoginMiddlewareBuilder struct {
	ijwt.Handler
}

func NewLoginMiddleWareBuilder(hdl ijwt.Handler) *LoginMiddlewareBuilder {
	return &LoginMiddlewareBuilder{Handler: hdl}
}

func (m *LoginMiddlewareBuilder) CheckLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		if path == "/users/login" ||
			path == "/users/signup" ||
			path == "/users/login_email" ||
			path == "/users/login_email/code/send" ||
			path == "/users/refresh_token" ||
			path == "/oauth2/wechat/authurl" ||
			path == "/oauth2/wechat/callback/login" {
			return
		}
		// 改为jwt鉴权
		authCode := ctx.GetHeader("Authorization")
		// 没token
		if authCode == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Bearer xxxx
		segs := strings.Split(authCode, " ")
		if len(segs) != 2 {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		tokenStr := segs[1]
		uc := &ijwt.UserClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, uc, func(*jwt.Token) (interface{}, error) {
			// 可以根据具体情况给出不同的key
			return m.JWTKey(), nil
		})
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if token == nil || !token.Valid {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// token有效
		// User-Agent
		//if uc.UserAgent != ctx.GetHeader("User-Agent") {
		//	// 大概率是攻击者才会进入这个分支
		//	ctx.AbortWithStatus(http.StatusUnauthorized)
		//	return
		//}
		ok, err := m.CheckSession(ctx, uc.Ssid)
		if err != nil || !ok {
			// err如果是redis崩溃导致，考虑进行降级，不再验证是否退出 refresh_token降级的话收益会很少，因为是低频接口
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// 刷新过期时间,固定每分钟只会刷一次
		//expireAt := uc.ExpiresAt.Time
		//if expireAt.Sub(time.Now()) < time.Minute*29 {
		//	// 刷新
		//	uc.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute * 30))
		//	tokenStr, err = token.SignedString(ijwt.JWTKey)
		//	if err != nil {
		//		// 刷新失败，但是校验已成功，不应该影响正常访问
		//		log.Println(err)
		//	}
		//	ctx.Header("x-ijwt-token", tokenStr)
		//}
		//ctx.Set("user", uc)
	}
}
