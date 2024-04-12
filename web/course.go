package web

import (
	ccnuv1 "github.com/MuxiKeStack/be-api/gen/proto/ccnu/v1"
	coursev1 "github.com/MuxiKeStack/be-api/gen/proto/course/v1"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	userv1 "github.com/MuxiKeStack/be-api/gen/proto/user/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/pkg/logger"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"strconv"
)

type CourseHandler struct {
	ijwt.Handler
	course     coursev1.CourseServiceClient
	evaluation evaluationv1.EvaluationServiceClient
	user       userv1.UserServiceClient
	l          logger.Logger
}

func NewCourseHandler(handler ijwt.Handler, course coursev1.CourseServiceClient,
	evaluation evaluationv1.EvaluationServiceClient, user userv1.UserServiceClient, l logger.Logger) *CourseHandler {
	return &CourseHandler{Handler: handler, course: course, evaluation: evaluation, user: user, l: l}
}

func (h *CourseHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	cg := s.Group("/courses")
	cg.GET("/profile/list", authMiddleware, ginx.WrapClaims(h.List))
	cg.GET("/:courseId/detail", ginx.Wrap(h.Detail))
}

// @Summary 我的课程列表
// @Description 获取用户的课程列表
// @Tags 课程
// @Accept json
// @Produce json
// @Param year query string false "年份，格式为YYYY"
// @Param term query string false "学期，如1、2、3"
// @Success 200 {object} ginx.Result{data=[]ProfileCourseVo} "Success"
// @Router /courses/profile/list [get]
func (h *CourseHandler) List(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	year := ctx.Query("year")
	term := ctx.Query("term")
	// 查询course
	res, err := h.course.SubscriptionList(ctx, &coursev1.SubscriptionListRequest{
		Uid:       uc.Uid,
		StudentId: uc.StudentId,
		Password:  uc.Password,
		Year:      year,
		Term:      term,
	})
	// from ccnu or 降级成功
	if err == nil || ccnuv1.IsNetworkToXkError(err) {
		courseVos := slice.Map(res.GetCourseSubscriptions(), func(idx int, src *coursev1.CourseSubscription) ProfileCourseVo {
			return ProfileCourseVo{
				Id:      src.GetCourse().GetId(),
				Name:    src.GetCourse().GetName(),
				Teacher: src.GetCourse().GetTeacher(),
				Year:    src.GetYear(),
				Term:    src.GetTerm(),
			}
		})
		// 这里要去聚合课评服务
		var eg errgroup.Group
		// 基于go1.22的新特性，这里的迭代变量i没有在内层重新copy
		for i := range courseVos {
			eg.Go(func() error {
				res, er := h.evaluation.Evaluated(ctx, &evaluationv1.EvaluatedRequest{
					CourseId: courseVos[i].Id,
					UserId:   uc.Uid,
				})
				courseVos[i].Evaluated = res.GetEvaluated()
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

// @Summary 获取课程详情
// @Description 根据课程ID获取课程详情信息,包括成绩
// @Tags 课程
// @Accept json
// @Produce json
// @Param courseId path integer true "课程ID"
// @Success 200 {object} ginx.Result{data=PublicCourseVo} "Success"
// @Router /courses/{courseId}/detail [get]
func (h *CourseHandler) Detail(ctx *gin.Context) (ginx.Result, error) {
	idStr := ctx.Param("courseId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	// 去查
	res, err := h.course.GetDetailById(ctx, &coursev1.GetDetailByIdRequest{
		CourseId: id,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}

	return ginx.Result{
		Msg: "Success",
		Data: PublicCourseVo{
			Id:       res.GetCourse().GetId(),
			Name:     res.GetCourse().GetName(),
			Teacher:  res.GetCourse().GetTeacher(),
			School:   res.GetCourse().GetSchool(),
			Property: res.GetCourse().GetProperty(),
			Credit:   res.GetCourse().GetCredit(),
			Grades: slice.Map(res.GetCourse().GetGrades(), func(idx int, src *coursev1.Grade) Grade {
				return Grade{
					Regular: src.GetRegular(),
					Final:   src.GetFinal(),
					Total:   src.GetTotal(),
					Year:    src.GetYear(),
					Term:    src.GetTerm(),
				}
			}),
		},
	}, nil
}
