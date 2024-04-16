package web

import (
	"context"
	"errors"
	coursev1 "github.com/MuxiKeStack/be-api/gen/proto/course/v1"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	tagv1 "github.com/MuxiKeStack/be-api/gen/proto/tag/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"github.com/seata/seata-go/pkg/tm"
	"strconv"
	"time"
)

type EvaluationHandler struct {
	evaluationClient evaluationv1.EvaluationServiceClient
	tagClient        tagv1.TagServiceClient
}

func NewEvaluationHandler(evaluationClient evaluationv1.EvaluationServiceClient, tagClient tagv1.TagServiceClient) *EvaluationHandler {
	return &EvaluationHandler{evaluationClient: evaluationClient, tagClient: tagClient}
}

func (h *EvaluationHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	eg := s.Group("/evaluation")
	eg.POST("/save", authMiddleware, ginx.WrapClaimsAndReq(h.Save))
	eg.POST("/:evaluationId/status", authMiddleware, ginx.WrapClaimsAndReq(h.UpdateStatus))
	eg.GET("/list/all", ginx.WrapReq(h.ListRecent))              // 最近的课程评价
	eg.GET("/list/course/:courseId", ginx.WrapReq(h.ListCourse)) // 指定课程的课程评价
	eg.GET("/list/mine", authMiddleware, ginx.WrapClaimsAndReq(h.ListMine))
	eg.GET("/count/course/:courseId/invisible", ginx.Wrap(h.CountCourseInvisible))
	eg.GET("/count/mine", authMiddleware, ginx.WrapClaimsAndReq(h.CountMine))
	eg.GET("/:evaluationId/detail", authMiddleware, ginx.WrapClaims(h.Detail))
}

// @Summary 发布课评
// @Description
// @Tags 课评
// @Accept json
// @Produce json
// @Param request body EvaluationSaveReq true "发布课评请求体"
// @Success 200 {object} ginx.Result{data=int64} "Success"
// @Router /evaluation/save [post]
func (h *EvaluationHandler) Save(ctx *gin.Context, req EvaluationSaveReq, uc ijwt.UserClaims) (ginx.Result, error) {
	// 这里要校验参数 1. content 长度 2. 星级是必选项
	if len([]rune(req.Content)) > 450 {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "评课内容长度过长，不能超过450个字符",
		}, errors.New("不合法课评内容长度")
	}
	if req.StarRating < 1 || req.StarRating > 5 {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "星级不合理，应为1到5",
		}, errors.New("不合法的课评星级")
	}
	assessmentTags := make([]tagv1.AssessmentTag, 0, len(req.Assessments))
	if len(req.Assessments) > 0 {
		for _, assessment := range req.Assessments {
			tag, ok := tagv1.AssessmentTag_value[assessment]
			if !ok {
				return ginx.Result{
					Code: errs.EvaluationInvalidInput,
					Msg:  "不合法的考核方式",
				}, errors.New("不合法的考核方式")
			}
			assessmentTags = append(assessmentTags, tagv1.AssessmentTag(tag))
		}
	}
	featureTags := make([]tagv1.FeatureTag, 0, len(req.Features))
	if len(req.Features) > 0 {
		for _, feature := range req.Features {
			tag, ok := tagv1.FeatureTag_value[feature]
			if !ok {
				return ginx.Result{
					Code: errs.EvaluationInvalidInput,
					Msg:  "不合法的课程特点",
				}, errors.New("不合法的课程特点")
			}
			featureTags = append(featureTags, tagv1.FeatureTag(tag))
		}
	}
	// status 不能是folded
	status, ok := evaluationv1.EvaluationStatus_value[req.Status]
	if !ok || req.Status == evaluationv1.EvaluationStatus_Folded.String() {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "不合法的课评状态",
		}, errors.New("不合法的课评状态")
	}

	var res *evaluationv1.SaveResponse
	// 下面涉及两个服务的原子性调用，需要使用分布式事务，这里的bff其实起到了聚合服务的作用...，引入实际意义聚合服务，目前没必要
	err := tm.WithGlobalTx(ctx,
		&tm.GtxConfig{
			Timeout: 1000 * time.Second, // todo
			Name:    "ATPublishAndTagTx",
		},
		func(ctx context.Context) error {
			var er error
			res, er = h.evaluationClient.Save(ctx, &evaluationv1.SaveRequest{
				Evaluation: &evaluationv1.Evaluation{
					Id:          req.Id,
					PublisherId: uc.Uid,
					CourseId:    req.CourseId, // TODO 下面这个地方要用外键
					StarRating:  uint32(req.StarRating),
					Content:     req.Content,
					Status:      evaluationv1.EvaluationStatus(status),
				},
			})
			if er != nil {
				return er
			}
			// 这里要去聚合 tag 服务，打两类标签
			_, er = h.tagClient.AttachAssessmentTags(ctx, &tagv1.AttachAssessmentTagsRequest{
				TaggerId: uc.Uid,
				Biz:      tagv1.Biz_Course,
				BizId:    req.CourseId, // 外键
				Tags:     assessmentTags,
			})
			if er != nil {
				return er
			}
			_, er = h.tagClient.AttachFeatureTags(ctx, &tagv1.AttachFeatureTagsRequest{
				TaggerId: uc.Uid,
				Biz:      tagv1.Biz_Course,
				BizId:    req.CourseId,
				Tags:     featureTags,
			})
			return er
		})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetEvaluationId(), // 这里给前端标明是evaluationId
	}, nil
}

func (h *EvaluationHandler) UpdateStatus(ctx *gin.Context, req EvaluationUpdateStatusReq, uc ijwt.UserClaims) (ginx.Result, error) {
	eidStr := ctx.Param("evaluationId")
	eid, err := strconv.ParseInt(eidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	status, ok := evaluationv1.EvaluationStatus_value[req.Status]
	if !ok || req.Status == evaluationv1.EvaluationStatus_Folded.String() {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "不合法的课评状态",
		}, errors.New("不合法的课评状态")
	}
	_, err = h.evaluationClient.UpdateStatus(ctx, &evaluationv1.UpdateStatusRequest{
		EvaluationId: eid,
		Status:       evaluationv1.EvaluationStatus(status),
		Uid:          uc.Uid,
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

func (h *EvaluationHandler) ListRecent(ctx *gin.Context, req ListRecentReq) (ginx.Result, error) {
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
	return ginx.Result{
		Msg: "Success",
		Data: slice.Map(res.Evaluation, func(idx int, src *evaluationv1.Evaluation) EvaluationVo {
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
		}),
	}, nil
}

func (h *EvaluationHandler) ListCourse(ctx *gin.Context, req ListCourseReq) (ginx.Result, error) {
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
	return ginx.Result{
		Msg: "Success",
		Data: slice.Map(res.Evaluation, func(idx int, src *evaluationv1.Evaluation) EvaluationVo {
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
		}),
	}, nil
}

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
	return ginx.Result{
		Msg: "Success",
		Data: slice.Map(res.Evaluation, func(idx int, src *evaluationv1.Evaluation) EvaluationVo {
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
		}),
	}, nil
}

func (h *EvaluationHandler) CountCourseInvisible(ctx *gin.Context) (ginx.Result, error) {
	cidStr := ctx.Param("courseId")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	res, err := h.evaluationClient.CountCourseInvisible(ctx, &evaluationv1.CountCourseInvisibleRequest{CourseId: cid})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetCount(), // 标明是count
	}, nil
}

func (h *EvaluationHandler) CountMine(ctx *gin.Context, req CountMineReq, uc ijwt.UserClaims) (ginx.Result, error) {
	status, ok := evaluationv1.EvaluationStatus_value[req.Status]
	if !ok {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "不合法的课评状态",
		}, errors.New("不合法的课评状态")
	}
	res, err := h.evaluationClient.CountMine(ctx, &evaluationv1.CountMineRequest{
		Uid:    uc.Uid,
		Status: evaluationv1.EvaluationStatus(status),
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetCount(), // 标明是count
	}, nil
}

func (h *EvaluationHandler) Detail(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	eidStr := ctx.Param("evaluationId")
	eid, err := strconv.ParseInt(eidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	res, err := h.evaluationClient.Detail(ctx, &evaluationv1.DetailRequest{
		EvaluationId: eid,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	if res.GetEvaluation().GetStatus() != evaluationv1.EvaluationStatus_Public &&
		res.GetEvaluation().GetPublisherId() != uc.Uid {
		return ginx.Result{
			Code: errs.EvaluationPermissionDenied,
			Msg:  "无法访问他人不可见的课评",
		}, nil
	}
	// 哦不，这里还要去聚合tags，但是似乎不用开分布式事务，因为只存在查询，没什么好事务的

	return ginx.Result{
		Msg: "Success",
		Data: EvaluationVo{
			Id:          res.Evaluation.GetId(),
			PublisherId: res.Evaluation.GetPublisherId(),
			CourseId:    res.Evaluation.GetCourseId(),
			StarRating:  res.Evaluation.GetStarRating(),
			Content:     res.Evaluation.GetContent(),
			Status:      res.Evaluation.GetStatus().String(),
		},
	}, nil
}
