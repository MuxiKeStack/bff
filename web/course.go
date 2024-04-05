package web

import (
	ccnuv1 "github.com/MuxiKeStack/be-api/gen/proto/ccnu/v1"
	coursev1 "github.com/MuxiKeStack/be-api/gen/proto/course/v1"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type CourseHandler struct {
	ijwt.Handler
	course     coursev1.CourseServiceClient
	evaluation evaluationv1.EvaluationServiceClient
}

func (h *CourseHandler) RegisterRoutes(s *gin.Engine) {
	cg := s.Group("/course")
	cg.GET("/list", ginx.WrapClaimsAndReq(h.List))
}

func (h *CourseHandler) List(ctx *gin.Context, req CourseListReq, uc ijwt.UserClaims) (ginx.Result, error) {
	// 查询course
	res, err := h.course.List(ctx, &coursev1.ListRequest{
		StudentId: uc.StudentId,
		Password:  uc.Password,
		Year:      req.Year,
		Term:      req.Term,
	})
	// from ccnu or 降级成功
	if err == nil || ccnuv1.IsNetworkToXkError(err) {
		courseVos := slice.Map(res.Courses, func(idx int, src *coursev1.Course) ProfileCourseVo {
			return ProfileCourseVo{
				StudentId: uc.StudentId,
				CourseId:  src.CourseId,
				Name:      src.Name,
				Teacher:   src.Teacher,
				Year:      src.Year,
				Term:      src.Term,
			}
		})
		// 这里要去聚合课评服务
		var eg errgroup.Group
		// 基于go1.22的新特性，这里的迭代变量i没有在内层重新copy
		for i := range courseVos {
			eg.Go(func() error {
				res, er := h.evaluation.Evaluated(ctx, &evaluationv1.EvaluatedRequest{
					StudentId: uc.StudentId,
					CourseId:  courseVos[i].CourseId,
					Name:      courseVos[i].Name,
					Teacher:   courseVos[i].Teacher,
				})
				courseVos[i].Evaluated = res.Evaluated
				return er
			})
		}
		er := eg.Wait()
		if er != nil {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, err
		}
		return ginx.Result{
			Msg:  "Success",
			Data: courseVos,
		}, nil
	}

	switch {
	case ccnuv1.IsInvalidSidOrPwd(err):
		// 学号密码错误，登出
		er := h.ClearToken(ctx)
		if er != nil {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, err
		}
		return ginx.Result{
			Code: errs.UserInvalidSidOrPassword,
			Msg:  "学号或密码错误，账号已登出",
		}, nil
	default:
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
}
