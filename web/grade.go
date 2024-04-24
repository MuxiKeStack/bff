package web

import (
	ccnuv1 "github.com/MuxiKeStack/be-api/gen/proto/ccnu/v1"
	gradev1 "github.com/MuxiKeStack/be-api/gen/proto/grade/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

type GradeHandler struct {
	gradeClient gradev1.GradeServiceClient
	ijwt.Handler
}

func NewGradeHandler(gradeClient gradev1.GradeServiceClient, jwtHdl ijwt.Handler) *GradeHandler {
	return &GradeHandler{
		gradeClient: gradeClient,
		Handler:     jwtHdl,
	}
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
	switch {
	case err == nil:
		return ginx.Result{
			Msg: "Success",
		}, nil
	case gradev1.IsRepeatSigning(err):
		return ginx.Result{
			Code: errs.GradeRepeatSigning,
			Msg:  "重复签约",
		}, err
	default:
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
}

func (h *GradeHandler) Share(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	_, err := h.gradeClient.ShareGrade(ctx, &gradev1.ShareGradeRequest{
		Uid:       uc.Uid,
		StudentId: uc.StudentId,
		Password:  uc.Password,
	})
	switch {
	case err == nil:
		return ginx.Result{
			Msg: "Success",
		}, nil
	case gradev1.IsNotSigned(err):
		return ginx.Result{
			Code: errs.GradeNotSigned,
			Msg:  "尚未签约",
		}, err
	case ccnuv1.IsInvalidSidOrPwd(err):
		// 登出
		er := h.ClearToken(ctx)
		if er != nil {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, err
		}
		return ginx.Result{
			Code: errs.UserInvalidSidOrPassword,
			Msg:  "登录失效，请重新登录",
		}, err
	default:
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
}
