package web

import (
	pointv1 "github.com/MuxiKeStack/be-api/gen/proto/point/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

type PointHandler struct {
	pointClient pointv1.PointServiceClient
}

func NewPointHandler(pointClient pointv1.PointServiceClient) *PointHandler {
	return &PointHandler{pointClient: pointClient}
}

func (h *PointHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	pg := s.Group("/points")
	pg.GET("/users/mine", authMiddleware, ginx.WrapClaims(h.GetPointInfoMine))
}

// GetPointInfoMine 获取用户的积分信息[自己]。
// @Summary 获取用户的积分信息
// @Description 获取包括积分、下一级所需积分和等级在内的当前用户的积分信息。
// @Tags 积分
// @Produce json
// @Success 200 {object} ginx.Result{data=PointInfoVo} "成功响应"
// @Router /points/users/mine [get]
func (h *PointHandler) GetPointInfoMine(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	res, err := h.pointClient.GetPointInfoOfUser(ctx, &pointv1.GetPointInfoOfUserRequest{
		Uid: uc.Uid,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: PointInfoVo{
			Points:          res.GetPoints(),
			NextLevelPoints: res.GetNextLevelPoints(),
			Level:           res.GetLevel(),
		},
	}, nil
}
