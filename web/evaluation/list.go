package evaluation

import (
	"errors"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	coursev1 "github.com/MuxiKeStack/be-api/gen/proto/course/v1"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	stancev1 "github.com/MuxiKeStack/be-api/gen/proto/stance/v1"
	tagv1 "github.com/MuxiKeStack/be-api/gen/proto/tag/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"strconv"
)

// @Summary 课评列表[广场]
// @Description
// @Tags 课评
// @Accept json
// @Produce json
// @Param cur_evaluation_id query int64 true "当前ID"
// @Param limit query int64 true "课评数量限制"
// @Param property query string false "用于过滤课评的课程性质（可选）"
// @Success 200 {object} ginx.Result{data=[]EvaluationVo} "Success"
// @Router /evaluations/list/all [get]
func (h *EvaluationHandler) ListRecent(ctx *gin.Context, req ListRecentReq, uc ijwt.UserClaims) (ginx.Result, error) {
	var property coursev1.CourseProperty
	if req.Property == "" {
		property = coursev1.CourseProperty_CoursePropertyAny
	} else {
		propertyUint32, ok := coursev1.CourseProperty_value[req.Property]
		if !ok {
			return ginx.Result{
				Code: errs.EvaluationInvalidInput,
				Msg:  "不合法的课程性质",
			}, errors.New("不合法的课程性质")
		}
		property = coursev1.CourseProperty(propertyUint32)
	}
	// 单次最多查一百
	if req.Limit > 100 {
		req.Limit = 100
	}
	res, err := h.evaluationClient.ListRecent(ctx, &evaluationv1.ListRecentRequest{
		CurEvaluationId: req.CurEvaluationId,
		Limit:           req.Limit,
		Property:        property,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	evaluationVos := slice.Map(res.GetEvaluations(), func(idx int, src *evaluationv1.Evaluation) EvaluationVo {
		return EvaluationVo{
			Id:          src.GetId(),
			PublisherId: src.GetPublisherId(),
			CourseId:    src.GetCourseId(),
			StarRating:  src.GetStarRating(),
			Content:     src.GetContent(),
			Status:      src.GetStatus().String(),
			Utime:       src.GetUtime(),
			Ctime:       src.GetCtime(),
		}
	})
	var eg errgroup.Group
	for i := range evaluationVos {
		eg.Go(func() error {
			// 因为这个路径被设置为了可以受限访问，也就是游客访问，所以这里做了区分
			if uc.Uid != 0 {
				stanceRes, er := h.stanceClient.GetUserStance(ctx, &stancev1.GetUserStanceRequest{
					Uid:   uc.Uid,
					Biz:   stancev1.Biz_Evaluation,
					BizId: evaluationVos[i].Id,
				})
				if er != nil {
					return er
				}
				evaluationVos[i].Stance = int32(stanceRes.GetStance())
				evaluationVos[i].TotalSupportCount = stanceRes.GetTotalSupports()
				evaluationVos[i].TotalOpposeCount = stanceRes.GetTotalOpposes()
			} else {
				countStanceRes, er := h.stanceClient.CountStance(ctx, &stancev1.CountStanceRequest{
					Biz:   stancev1.Biz_Evaluation,
					BizId: evaluationVos[i].Id,
				})
				if er != nil {
					return er
				}
				evaluationVos[i].TotalSupportCount = countStanceRes.GetTotalSupports()
				evaluationVos[i].TotalOpposeCount = countStanceRes.GetTotalOpposes()
			}
			countCommentRes, er := h.commentClient.CountComment(ctx, &commentv1.CountCommentRequest{
				Biz:   commentv1.Biz_Evaluation,
				BizId: evaluationVos[i].Id,
			})
			if er != nil {
				return er
			}
			evaluationVos[i].TotalCommentCount = countCommentRes.GetCount()
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	// 要聚合评论数，支持反对数，是否支持了
	return ginx.Result{
		Msg:  "Success",
		Data: evaluationVos,
	}, nil
}

// ListCourse 根据课程ID列出课评
// @Summary 课评列表[指定课程]
// @Description 根据课程ID获取课程评价列表，支持分页。
// @Tags 课评
// @Accept json
// @Produce json
// @Param courseId path int64 true "课程ID"
// @Param cur_evaluation_id query int64 true "当前课评ID"
// @Param limit query int64 true "返回课评的最大数量，上限为100"
// @Success 200 {object} ginx.Result{data=[]EvaluationVo} "Success"
// @Router /evaluations/list/courses/{courseId} [get]
func (h *EvaluationHandler) ListCourse(ctx *gin.Context, req ListCourseReq, uc ijwt.UserClaims) (ginx.Result, error) {
	cidStr := ctx.Param("courseId")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	// 单次最多查一百
	if req.Limit > 100 {
		req.Limit = 100
	}
	res, err := h.evaluationClient.ListCourse(ctx, &evaluationv1.ListCourseRequest{
		CurEvaluationId: req.CurEvaluationId,
		Limit:           req.Limit,
		CourseId:        cid,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	evaluationVos := slice.Map(res.GetEvaluations(), func(idx int, src *evaluationv1.Evaluation) EvaluationVo {
		return EvaluationVo{
			Id:          src.GetId(),
			PublisherId: src.GetPublisherId(),
			CourseId:    src.GetCourseId(),
			StarRating:  src.GetStarRating(),
			Content:     src.GetContent(),
			Status:      src.GetStatus().String(),
			Utime:       src.GetUtime(),
			Ctime:       src.GetCtime(),
		}
	})
	// 这里要为，每个课评，聚合标签
	var eg errgroup.Group
	for i := range evaluationVos {
		eg.Go(func() error {
			atRes, er := h.tagClient.GetAssessmentTagsByTaggerBiz(ctx, &tagv1.GetAssessmentTagsByTaggerBizRequest{
				TaggerId: evaluationVos[i].PublisherId,
				Biz:      tagv1.Biz_Course,
				BizId:    evaluationVos[i].CourseId,
			})
			if er != nil {
				return er
			}
			evaluationVos[i].Assessments = slice.Map(atRes.GetTags(), func(idx int, src tagv1.AssessmentTag) string {
				return src.String()
			})
			ftRes, er := h.tagClient.GetFeatureTagsByTaggerBiz(ctx, &tagv1.GetFeatureTagsByTaggerBizRequest{
				TaggerId: evaluationVos[i].PublisherId,
				Biz:      tagv1.Biz_Course,
				BizId:    evaluationVos[i].CourseId,
			})
			if er != nil {
				return er
			}
			evaluationVos[i].Features = slice.Map(ftRes.GetTags(), func(idx int, src tagv1.FeatureTag) string {
				return src.String()
			})
			stanceRes, er := h.stanceClient.GetUserStance(ctx, &stancev1.GetUserStanceRequest{
				Uid:   uc.Uid,
				Biz:   stancev1.Biz_Evaluation,
				BizId: evaluationVos[i].Id,
			})
			if er != nil {
				return er
			}
			countCommentRes, er := h.commentClient.CountComment(ctx, &commentv1.CountCommentRequest{
				Biz:   commentv1.Biz_Evaluation,
				BizId: evaluationVos[i].Id,
			})
			if er != nil {
				return er
			}
			evaluationVos[i].Stance = int32(stanceRes.GetStance())
			evaluationVos[i].TotalSupportCount = stanceRes.GetTotalSupports()
			evaluationVos[i].TotalOpposeCount = stanceRes.GetTotalOpposes()
			evaluationVos[i].TotalCommentCount = countCommentRes.GetCount()
			return nil
		})
	}
	// 要聚合评论数，支持反对数，是否支持了
	err = eg.Wait()
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: evaluationVos,
	}, nil
}

// ListMine 我的历史
// @Summary 课评列表[我的历史]
// @Description 根据课程ID获取课程评价列表，支持分页。
// @Tags 课评
// @Accept json
// @Produce json
// @Param courseId path int64 true "课程ID"
// @Param cur_evaluation_id query int64 true "当前评估ID，用于分页"
// @Param limit query int64 true "上限为100"
// @Param status query string true "课评状态: Public/Private/Folded"
// @Success 200 {object} ginx.Result{data=[]EvaluationVo} "Success"
// @Router /evaluations/list/mine [get]
func (h *EvaluationHandler) ListMine(ctx *gin.Context, req ListMineReq, uc ijwt.UserClaims) (ginx.Result, error) {
	status, ok := evaluationv1.EvaluationStatus_value[req.Status]
	if !ok {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "不合法的课评状态",
		}, errors.New("不合法的课评状态")
	}
	// 单次最多查一百
	if req.Limit > 100 {
		req.Limit = 100
	}
	res, err := h.evaluationClient.ListMine(ctx, &evaluationv1.ListMineRequest{
		CurEvaluationId: req.CurEvaluationId,
		Limit:           req.Limit,
		Uid:             uc.Uid,
		Status:          evaluationv1.EvaluationStatus(status),
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	evaluationVos := slice.Map(res.GetEvaluations(), func(idx int, src *evaluationv1.Evaluation) EvaluationVo {
		return EvaluationVo{
			Id:          src.GetId(),
			PublisherId: src.GetPublisherId(),
			CourseId:    src.GetCourseId(),
			StarRating:  src.GetStarRating(),
			Content:     src.GetContent(),
			Status:      src.GetStatus().String(),
			Utime:       src.GetUtime(),
			Ctime:       src.GetCtime(),
		}
	})
	var eg errgroup.Group
	for i := range evaluationVos {
		eg.Go(func() error {
			stanceRes, er := h.stanceClient.GetUserStance(ctx, &stancev1.GetUserStanceRequest{
				Uid:   uc.Uid,
				Biz:   stancev1.Biz_Evaluation,
				BizId: evaluationVos[i].Id,
			})
			if er != nil {
				return er
			}
			countCommentRes, er := h.commentClient.CountComment(ctx, &commentv1.CountCommentRequest{
				Biz:   commentv1.Biz_Evaluation,
				BizId: evaluationVos[i].Id,
			})
			if er != nil {
				return er
			}
			evaluationVos[i].Stance = int32(stanceRes.GetStance())
			evaluationVos[i].TotalSupportCount = stanceRes.GetTotalSupports()
			evaluationVos[i].TotalOpposeCount = stanceRes.GetTotalOpposes()
			evaluationVos[i].TotalCommentCount = countCommentRes.GetCount()
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	// 要聚合评论数，支持反对数，是否支持了
	return ginx.Result{
		Msg:  "Success",
		Data: evaluationVos,
	}, nil
}
