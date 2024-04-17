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
	"golang.org/x/sync/errgroup"
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
	eg := s.Group("/evaluations")
	eg.POST("/save", authMiddleware, ginx.WrapClaimsAndReq(h.Save))
	eg.POST("/:evaluationId/status", authMiddleware, ginx.WrapClaimsAndReq(h.UpdateStatus))
	eg.GET("/list/all", ginx.WrapReq(h.ListRecent))               // 广场
	eg.GET("/list/courses/:courseId", ginx.WrapReq(h.ListCourse)) // 指定课程的课程评价
	eg.GET("/list/mine", authMiddleware, ginx.WrapClaimsAndReq(h.ListMine))
	eg.GET("/count/courses/:courseId/invisible", ginx.Wrap(h.CountCourseInvisible))
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
// @Router /evaluations/save [post]
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
	// go的seatago框架相当不成熟，比如这个事务内部不能用errgroup并发这两个attach tag
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

// @Summary 变更课评状态
// @Description
// @Tags 课评
// @Accept json
// @Produce json
// @Param request body EvaluationUpdateStatusReq true "变更课评状态请求体"
// @Success 200 {object} ginx.Result "Success"
// @Router /evaluations/{evaluationId}/status [post]
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
func (h *EvaluationHandler) ListRecent(ctx *gin.Context, req EvaluationListRecentReq) (ginx.Result, error) {
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
		Data: slice.Map(res.GetEvaluations(), func(idx int, src *evaluationv1.Evaluation) EvaluationVo {
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
func (h *EvaluationHandler) ListCourse(ctx *gin.Context, req EvaluationListCourseReq) (ginx.Result, error) {
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
func (h *EvaluationHandler) ListMine(ctx *gin.Context, req EvaluationListMineReq, uc ijwt.UserClaims) (ginx.Result, error) {
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
		Data: slice.Map(res.GetEvaluations(), func(idx int, src *evaluationv1.Evaluation) EvaluationVo {
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

// CountCourseInvisible 计算指定课程的不可见评价数量。
// @Summary 不可见课评数
// @Description 根据课程ID计算该课程的不可见评价数量。
// @Tags 课评
// @Accept json
// @Produce json
// @Param courseId path int64 true "课程ID"
// @Success 200 {object} ginx.Result{data=int64} "Success"
// @Router /evaluations/count/courses/{courseId}/invisible [get]
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

// CountMine 统计用户的课评数量。
// @Summary 用户课评数
// @Description 根据用户ID和课评状态分类统计用户的课评数量。
// @Tags 课评
// @Accept json
// @Produce json
// @Param status query string true "课评状态"
// @Success 200 {object} ginx.Result{data=int64} "Success"
// @Router /evaluations/count/mine [get]
func (h *EvaluationHandler) CountMine(ctx *gin.Context, req EvaluationCountMineReq, uc ijwt.UserClaims) (ginx.Result, error) {
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

// Detail 课评详情。
// @Summary 课评详情
// @Description 根据课评ID获取详情，详情包括标签
// @Tags 课评
// @Accept json
// @Produce json
// @Param evaluationId path int64 true "课评ID"
// @Success 200 {object} ginx.Result{data=EvaluationVo} "Success"
// @Router /evaluations/{evaluationId}/detail [get]
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
	var (
		eg    errgroup.Group
		atRes *tagv1.GetAssessmentTagsByTaggerBizResponse
		ftRes *tagv1.GetFeatureTagsByTaggerBizResponse
	)
	eg.Go(func() error {
		var er error
		atRes, er = h.tagClient.GetAssessmentTagsByTaggerBiz(ctx, &tagv1.GetAssessmentTagsByTaggerBizRequest{
			TaggerId: res.GetEvaluation().GetPublisherId(),
			Biz:      tagv1.Biz_Course,
			BizId:    res.GetEvaluation().GetCourseId(),
		})
		return er
	})
	eg.Go(func() error {
		var er error
		ftRes, er = h.tagClient.GetFeatureTagsByTaggerBiz(ctx, &tagv1.GetFeatureTagsByTaggerBizRequest{
			TaggerId: res.GetEvaluation().GetPublisherId(),
			Biz:      tagv1.Biz_Course,
			BizId:    res.GetEvaluation().GetCourseId(),
		})
		return er
	})
	err = eg.Wait()
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: EvaluationVo{
			Id:          res.GetEvaluation().GetId(),
			PublisherId: res.GetEvaluation().GetPublisherId(),
			CourseId:    res.GetEvaluation().GetCourseId(),
			StarRating:  res.GetEvaluation().GetStarRating(),
			Content:     res.GetEvaluation().GetContent(),
			Status:      res.GetEvaluation().GetStatus().String(),
			Assessments: slice.Map(atRes.GetTags(), func(idx int, src tagv1.AssessmentTag) string {
				return src.String()
			}),
			Features: slice.Map(ftRes.GetTags(), func(idx int, src tagv1.FeatureTag) string {
				return src.String()
			}),
			Utime: res.GetEvaluation().GetUtime(),
			Ctime: res.GetEvaluation().GetCtime(),
		},
	}, nil
}
