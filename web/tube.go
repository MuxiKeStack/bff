package web

import (
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/qiniu/api.v7/v7/auth/qbox"
	"github.com/qiniu/api.v7/v7/storage"
)

type TubeHandler struct {
	putPolicy  storage.PutPolicy
	mac        *qbox.Mac
	domainName string
}

func NewTubeHandler(putPolicy storage.PutPolicy, mac *qbox.Mac, domainName string) *TubeHandler {
	return &TubeHandler{
		putPolicy:  putPolicy,
		mac:        mac,
		domainName: domainName,
	}
}

func (t *TubeHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	tg := s.Group("/tube")
	tg.GET("/access_token", authMiddleware, ginx.WrapClaims(t.GetTubeToken))
}

// @Summary 获取图床访问令牌
// @Description
// @Tags 图床
// @Accept json
// @Produce json
// @Success 200 {object} ginx.Result{data=GetTubeTokenData} "成功"
// @Router /tube/access_token [get]
func (t *TubeHandler) GetTubeToken(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	accessToken := t.putPolicy.UploadToken(t.mac)
	return ginx.Result{
		Msg: "Success",
		Data: GetTubeTokenData{
			AccessToken: accessToken,
			DomainName:  t.domainName,
		},
	}, nil
}
