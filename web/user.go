package web

import (
	"errors"
	ccnuv1 "github.com/MuxiKeStack/be-api/gen/proto/ccnu/v1"
	gradev1 "github.com/MuxiKeStack/be-api/gen/proto/grade/v1"
	pointv1 "github.com/MuxiKeStack/be-api/gen/proto/point/v1"
	userv1 "github.com/MuxiKeStack/be-api/gen/proto/user/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/sync/errgroup"
	"maps"
	"net/http"
	"strconv"
)

type UserHandler struct {
	ijwt.Handler
	userSvc       userv1.UserServiceClient
	ccnuSvc       ccnuv1.CCNUServiceClient
	gradeSvc      gradev1.GradeServiceClient
	pointSvc      pointv1.PointServiceClient
	allPointTitle map[string]bool
}

func NewUserHandler(hdl ijwt.Handler, userSvc userv1.UserServiceClient, ccnuSvc ccnuv1.CCNUServiceClient,
	gradeSvc gradev1.GradeServiceClient, pointSvc pointv1.PointServiceClient) *UserHandler {
	allPointTitle := make(map[string]bool, len(pointv1.Title_value))
	for title := range pointv1.Title_value {
		allPointTitle[title] = false
	}
	return &UserHandler{
		Handler:       hdl,
		userSvc:       userSvc,
		ccnuSvc:       ccnuSvc,
		gradeSvc:      gradeSvc,
		pointSvc:      pointSvc,
		allPointTitle: allPointTitle,
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
	usingTitle, exists := pointv1.Title_value[req.UsingTitle]
	if !exists || req.UsingTitle == pointv1.Title_None.String() {
		return ginx.Result{
			Code: errs.UserInvalidInput,
			Msg:  "无效的title",
		}, errors.New("无效的title")
	}
	var eg errgroup.Group
	eg.Go(func() error {
		_, err := h.pointSvc.SaveUsingTitleOfUser(ctx, &pointv1.SaveUsingTitleOfUserRequest{
			Uid:   uc.Uid,
			Title: pointv1.Title(usingTitle),
		})
		return err
	})
	eg.Go(func() error {
		_, err := h.userSvc.UpdateNonSensitiveInfo(ctx, &userv1.UpdateNonSensitiveInfoRequest{
			User: &userv1.User{
				Id:       uc.Uid,
				Avatar:   req.Avatar,
				Nickname: req.Nickname,
			},
		})
		return err
	})
	err := eg.Wait()
	switch {
	case err == nil:
		return ginx.Result{
			Msg: "Success",
		}, nil
	case pointv1.IsPointNotEnough(err):
		return ginx.Result{
			Code: errs.PointsNotEnough,
			Msg:  "积分不足",
		}, err
	default:
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
}

// @Summary 获取用户信息[自己]
// @Description
// @Tags 用户
// @Accept json
// @Produce json
// @Success 200 {object} ginx.Result{data=UserProfileVo} "Success"
// @Router /users/profile [get]
func (h *UserHandler) Profile(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	var (
		eg             errgroup.Group
		userRes        *userv1.ProfileResponse
		statusRes      *gradev1.GetSignStatusResponse
		titleOwnership map[string]bool
		usingTitle     string
	)
	eg.Go(func() error {
		var er error
		userRes, er = h.userSvc.Profile(ctx, &userv1.ProfileRequest{Uid: uc.Uid})
		return er
	})
	eg.Go(func() error {
		var er error
		statusRes, er = h.gradeSvc.GetSignStatus(ctx, &gradev1.GetSignStatusRequest{Uid: uc.Uid})
		return er
	})
	// 聚合title信息
	eg.Go(func() error {
		titleRes, er := h.pointSvc.GetTitleOfUser(ctx, &pointv1.GetTitleOfUserRequest{Uid: uc.Uid})
		if er != nil {
			return er
		}
		titleOwnership = maps.Clone(h.allPointTitle)
		delete(titleOwnership, pointv1.Title_None.String())
		for _, title := range titleRes.GetOwnedTitles() {
			titleOwnership[title.String()] = true
		}
		usingTitle = titleRes.GetUsingTitle().String()
		return nil
	})
	err := eg.Wait()
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: UserProfileVo{
			Id:                   userRes.GetUser().GetId(),
			StudentId:            userRes.GetUser().GetStudentId(),
			Avatar:               userRes.GetUser().GetAvatar(),
			Nickname:             userRes.GetUser().GetNickname(),
			New:                  userRes.GetUser().GetNew(),
			GradeSharingIsSigned: statusRes.GetIsSigned(),
			UsingTitle:           usingTitle,
			TitleOwnership:       titleOwnership,
			Utime:                userRes.GetUser().GetUtime(),
			Ctime:                userRes.GetUser().GetCtime(),
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
	switch {
	case err == nil:
		return ginx.Result{
			Msg: "Success",
			Data: UserPublicProfileVo{
				Id:       res.GetUser().GetId(),
				Avatar:   res.GetUser().GetAvatar(),
				Nickname: res.GetUser().GetNickname(),
			},
		}, nil
	case userv1.IsUserNotFound(err):
		return ginx.Result{
			Code: errs.UserNotFound,
			Msg:  "用户不存在",
		}, err
	default:
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
}
