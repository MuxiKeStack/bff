package web

import (
	ccnuv1 "github.com/MuxiKeStack/be-api/gen/proto/ccnu/v1"
	userv1 "github.com/MuxiKeStack/be-api/gen/proto/user/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strconv"
)

type UserHandler struct {
	ijwt.Handler
	userSvc userv1.UserServiceClient
	ccnuSvc ccnuv1.CCNUServiceClient
}

func NewUserHandler(hdl ijwt.Handler, userSvc userv1.UserServiceClient, ccnuSvc ccnuv1.CCNUServiceClient) *UserHandler {
	return &UserHandler{
		Handler: hdl,
		userSvc: userSvc,
		ccnuSvc: ccnuSvc,
	}
}

func (h *UserHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	ug := s.Group("/users")
	ug.POST("/login_ccnu", ginx.WrapReq(h.LoginByCCNU))
	ug.POST("/logout", authMiddleware, ginx.Wrap(h.Logout))
	ug.GET("/refresh_token", h.RefreshToken)
	ug.POST("/edit", authMiddleware, ginx.WrapClaimsAndReq(h.Edit))
	ug.GET("/profile", authMiddleware, ginx.WrapClaims(h.Profile))
	ug.GET("/:userId/profile", ginx.Wrap(h.ProfileById))
}

// @Summary ccnu登录
// @Description 通过学号和密码进行登录认证
// @Tags 用户
// @Accept json
// @Produce json
// @Param body body LoginByCCNUReq true "登录请求体"
// @Success 200 {object} ginx.Result "Success"
// @Router /users/login_ccnu [post]
func (h *UserHandler) LoginByCCNU(ctx *gin.Context, req LoginByCCNUReq) (ginx.Result, error) {
	_, err := h.ccnuSvc.Login(ctx, &ccnuv1.LoginRequest{
		StudentId: req.StudentId,
		Password:  req.Password,
	})
	switch {
	case err == nil:
	// 直接向下执行
	case ccnuv1.IsInvalidSidOrPwd(err):
		return ginx.Result{
			Code: errs.UserInvalidSidOrPassword,
			Msg:  "学号或密码错误",
		}, nil
	default:
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	// FindOrCreate
	fcRes, err := h.userSvc.FindOrCreateByStudentId(ctx, &userv1.FindOrCreateByStudentIdRequest{StudentId: req.StudentId})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	err = h.SetLoginToken(ctx, fcRes.GetUser().GetId(), req.StudentId, req.Password)
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

// @Summary 登出(销毁token)
// @Description 通过短token登出
// @Tags 用户
// @Accept json
// @Produce json
// @Success 200 {object} ginx.Result "Success"
// @Router /users/logout [post]
func (h *UserHandler) Logout(ctx *gin.Context) (ginx.Result, error) {
	err := h.ClearToken(ctx)
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

// @Summary 刷新短token
// @Description 通过长token刷新短token
// @Tags 用户
// @Accept json
// @Produce json
// @Success 200 {object} ginx.Result "Success"
// @Router /users/refresh_token [get]
func (h *UserHandler) RefreshToken(ctx *gin.Context) {
	tokenStr := h.ExtractToken(ctx)
	rc := &ijwt.RefreshClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, rc, func(*jwt.Token) (interface{}, error) {
		// 可以根据具体情况给出不同的key
		return h.RCJWTKey(), nil
	})
	if err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if token == nil || !token.Valid {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	ok, err := h.CheckSession(ctx, rc.Ssid)
	if err != nil || ok {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	err = h.SetJWTToken(ctx, ijwt.ClaimParams{
		Uid:       rc.Uid,
		StudentId: rc.StudentId,
		Password:  rc.Password,
		Ssid:      rc.Ssid,
		UserAgent: rc.UserAgent,
	})
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{
		Msg: "Success",
	})
}

// @Summary 编辑个人信息
// @Description
// @Tags 用户
// @Accept json
// @Produce json
// @Param body body UserEditReq true "编辑用户信息请求体"
// @Success 200 {object} ginx.Result "Success"
// @Router /users/edit [post]
func (h *UserHandler) Edit(ctx *gin.Context, req UserEditReq, uc ijwt.UserClaims) (ginx.Result, error) {
	_, err := h.userSvc.UpdateNonSensitiveInfo(ctx, &userv1.UpdateNonSensitiveInfoRequest{
		User: &userv1.User{
			Id:       uc.Uid,
			Avatar:   req.Avatar,
			Nickname: req.Nickname,
		},
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

// @Summary 获取用户信息[自己]
// @Description
// @Tags 用户
// @Accept json
// @Produce json
// @Success 200 {object} ginx.Result{data=UserProfileVo} "Success"
// @Router /users/profile [get]
func (h *UserHandler) Profile(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	res, err := h.userSvc.Profile(ctx, &userv1.ProfileRequest{Uid: uc.Uid})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: UserProfileVo{
			Id:        res.GetUser().GetId(),
			StudentId: res.GetUser().GetStudentId(),
			Avatar:    res.GetUser().GetAvatar(),
			Nickname:  res.GetUser().GetNickname(),
			New:       res.GetUser().GetNew(),
			Utime:     res.GetUser().GetUtime(),
			Ctime:     res.GetUser().GetCtime(),
		},
	}, nil
}

// @Summary 获取用户信息[公开]
// @Description
// @Tags 用户
// @Accept json
// @Produce json
// @Success 200 {object} ginx.Result{data=UserPublicProfileVo} "Success"
// @Router /users/:userId/profile [get]
func (h *UserHandler) ProfileById(ctx *gin.Context) (ginx.Result, error) {
	uidStr := ctx.Param("userId")
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.UserInvalidInput,
			Msg:  "无效的输入参数",
		}, err
	}
	res, err := h.userSvc.Profile(ctx, &userv1.ProfileRequest{Uid: uid})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: UserPublicProfileVo{
			Id:       res.GetUser().GetId(),
			Avatar:   res.GetUser().GetAvatar(),
			Nickname: res.GetUser().GetNickname(),
		},
	}, nil
}
