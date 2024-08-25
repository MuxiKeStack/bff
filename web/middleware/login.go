package middleware

import (
	"errors"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/ecodeclub/ekit/set"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
)

type LoginMiddlewareBuilder struct {
	allowRestrictedAccessPaths set.Set[string]
	ijwt.Handler
}

func NewLoginMiddleWareBuilder(hdl ijwt.Handler) *LoginMiddlewareBuilder {
	s := set.NewMapSet[string](3)
	s.Add("/evaluations/list/all")
	l := &LoginMiddlewareBuilder{
		allowRestrictedAccessPaths: s,
		Handler:                    hdl,
	}
	return l
}

func (m *LoginMiddlewareBuilder) allowRestrictedAccess(path string) bool {
	if m.allowRestrictedAccessPaths.Exist(path) {
		return true
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 3 && parts[0] == "evaluations" && parts[2] == "detail" {
		return true
	}
	return false
}

func (m *LoginMiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 放行游客可访问的路由，
		uc, err := m.extractUserClaimsFromAuthorizationHeader(ctx)
		if err == nil {
			ctx.Set("user", uc)
		} else {
			if m.allowRestrictedAccess(ctx.Request.URL.Path) {
				ctx.Set("user", uc)
			} else {
				ctx.AbortWithStatus(http.StatusUnauthorized)
			}
		}
	}
}

func (m *LoginMiddlewareBuilder) extractUserClaimsFromAuthorizationHeader(ctx *gin.Context) (ijwt.UserClaims, error) {
	authCode := ctx.GetHeader("Authorization")
	// 没token
	if authCode == "" {
		return ijwt.UserClaims{}, errors.New("authorization为空")
	}
	// Bearer xxxx
	segs := strings.Split(authCode, " ")
	if len(segs) != 2 {
		return ijwt.UserClaims{}, errors.New("authorization为空格式不合理")
	}
	tokenStr := segs[1]
	uc := ijwt.UserClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, &uc, func(*jwt.Token) (interface{}, error) {
		// 可以根据具体情况给出不同的key
		return m.JWTKey(), nil
	})
	if err != nil {
		return ijwt.UserClaims{}, err
	}
	if token == nil || !token.Valid {
		return ijwt.UserClaims{}, errors.New("token无效")
	}
	// token有效
	//User-Agent
	//if uc.UserAgent != ctx.GetHeader("User-Agent") {
	//	// 大概率是攻击者才会进入这个分支
	//	return ijwt.UserClaims{}, errors.New("User-Agent验证：不安全")
	//}
	ok, err := m.CheckSession(ctx, uc.Ssid)
	if err != nil || ok {
		// err如果是redis崩溃导致，考虑进行降级，不再验证是否退出 refresh_token降级的话收益会很少，因为是低频接口
		// 这里 != nil 就是异常，可能崩溃，或连不上
		return ijwt.UserClaims{}, errors.New("session检验：失败")
	}
	return uc, nil
}
