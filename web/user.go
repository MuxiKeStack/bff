package web

import (
	userv1 "github.com/MuxiKeStack/be-api/gen/proto/user/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	ijwt.Handler
	svc userv1.UserServiceClient
}

func (h *UserHandler) RegisterRoutes(s *gin.Engine) {
	ug := s.Group("/users")
	ug.POST("/login_ccnu", ginx.WrapBody(h.LoginByCCNU))
}

func (h *UserHandler) LoginByCCNU(ctx *gin.Context, req LoginByCCNUReq) (ginx.Result, error) {
	resp, err := h.svc.LoginByCCNU(ctx, &userv1.LoginByCCNURequest{
		StudentId: req.StudentId,
		Password:  req.Password,
	})
	if err == nil {
		err := h.SetLoginToken(ctx, resp.User.Id)
		if err != nil {
			return ginx.Result{
				Code: errs.UserInternalServerError,
				Msg:  "系统异常",
				Data: nil,
			}, err
		}
		return ginx.Result{
			Msg:  "登录成功",
			Data: nil,
		}, nil
	}

	switch {
	case userv1.IsInvalidSidOrPwd(err):
		return ginx.Result{
			Code: errs.UserInvalidSidOrPassword,
			Msg:  "学号或密码错误",
			Data: nil,
		}, err
	default:
		return ginx.Result{
			Code: errs.UserInternalServerError,
			Msg:  "系统异常",
			Data: nil,
		}, err
	}
}
