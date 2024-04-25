package web

import (
	"errors"
	ccnuv1 "github.com/MuxiKeStack/be-api/gen/proto/ccnu/v1"
	gradev1 "github.com/MuxiKeStack/be-api/gen/proto/grade/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"strconv"
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
	g := s.Group("/grades")
	g.POST("/sign", authMiddleware, ginx.WrapClaimsAndReq(h.Sign)) //签约
	g.POST("/share", authMiddleware, ginx.WrapClaims(h.Share))
	g.GET("/courses/:courseId", authMiddleware, ginx.WrapClaims(h.GetCourseGrades))
}

// Sign 成绩签约或取消
// @Summary 签约或取消签约
// @Description 用户选择是否签约分享成绩
// @Tags 成绩
// @Accept json
// @Produce json
// @Param SignReq body SignReq true "签约请求信息"
// @Success 200 {object} ginx.Result "成功返回结果"
// @Router /grades/sign [post]
func (h *GradeHandler) Sign(ctx *gin.Context, req SignReq, uc ijwt.UserClaims) (ginx.Result, error) {
	_, err := h.gradeClient.SignForGradeSharing(ctx, &gradev1.SignForGradeSharingRequest{
		Uid:         uc.Uid,
		WantsToSign: req.WantsToSign,
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
	case gradev1.IsRepeatCancelSigning(err):
		return ginx.Result{
			Code: errs.GradeRepeatSigning,
			Msg:  "重复取消签约",
		}, err
	default:
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
}

// Share 共享成绩信息
// @Summary 分享成绩
// @Description 用户主动发起一次分享自己的最新成绩
// @Tags 成绩
// @Accept json
// @Produce json
// @Success 200 {object} ginx.Result "成功返回结果"
// @Router /grades/share [post]
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

// GetCourseGrades 获取指定课程的成绩详情
// @Summary 获取课程成绩分布
// @Description 根据课程ID获取该课程的成绩详情
// @Tags 成绩
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param courseId path int true "课程ID"
// @Success 200 {object} ginx.Result{data=[]GradeVo} "成功返回成绩数组"
// @Failure 400 "参数错误或未签约"
// @Router /grades/courses/{courseId} [get]
func (h *GradeHandler) GetCourseGrades(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	cidStr := ctx.Param("courseId")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	statusRes, err := h.gradeClient.GetSignStatus(ctx, &gradev1.GetSignStatusRequest{Uid: uc.Uid})
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	if !statusRes.GetIsSigned() {
		return ginx.Result{
			Code: errs.GradeNotSigned,
			Msg:  "未签约无法获取成绩详情",
		}, errors.New("未签约无法获取成绩详情")
	}
	gradesRes, err := h.gradeClient.GetGradesByCourseId(ctx, &gradev1.GetGradesByCourseIdRequest{CourseId: cid})
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: slice.Map(gradesRes.GetGrades(), func(idx int, src *ccnuv1.Grade) GradeVo {
			return GradeVo{
				Regular: src.Regular,
				Final:   src.Final,
				Total:   src.Total,
				Year:    src.Year,
				Term:    src.Term,
			}
		}),
	}, nil
}
