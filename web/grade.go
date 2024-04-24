package web

import (
	gradev1 "github.com/MuxiKeStack/be-api/gen/proto/grade/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

type GradeHandler struct {
	gradeClient gradev1.GradeServiceClient
}

func NewGradeHandler(gradeClient gradev1.GradeServiceClient) *GradeHandler {
	return &GradeHandler{gradeClient: gradeClient}
}

func (h *GradeHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	g := s.Group("/grade")
	g.POST("/sign", authMiddleware, ginx.WrapClaims(h.Sign)) //签约
	g.POST("/share", authMiddleware, ginx.WrapClaims(h.Share))
}

func (h *GradeHandler) Sign(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	_, err := h.gradeClient.SignupForGradeSharing(ctx, &gradev1.SignupForGradeSharingRequest{
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
	}, nil
}

func (h *GradeHandler) Share(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	_, err := h.gradeClient.ShareGrade(ctx, &gradev1.ShareGradeRequest{
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
	}, nil
}
