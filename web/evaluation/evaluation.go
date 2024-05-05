package evaluation

import (
	"context"
	"errors"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	stancev1 "github.com/MuxiKeStack/be-api/gen/proto/stance/v1"
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
	stanceClient     stancev1.StanceServiceClient
	commentClient    commentv1.CommentServiceClient
}

func NewEvaluationHandler(evaluationClient evaluationv1.EvaluationServiceClient, tagClient tagv1.TagServiceClient,
	interactClient stancev1.StanceServiceClient, commentClient commentv1.CommentServiceClient) *EvaluationHandler {
	return &EvaluationHandler{
		evaluationClient: evaluationClient,
		tagClient:        tagClient,
		stanceClient:     interactClient,
		commentClient:    commentClient,
	}
}

func (h *EvaluationHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	eg := s.Group("/evaluations")
	eg.POST("/save", authMiddleware, ginx.WrapClaimsAndReq(h.Save))
	eg.POST("/:evaluationId/status", authMiddleware, ginx.WrapClaimsAndReq(h.UpdateStatus))
	eg.GET("/list/all", authMiddleware, ginx.WrapClaimsAndReq(h.ListRecent))               // 广场
	eg.GET("/list/courses/:courseId", authMiddleware, ginx.WrapClaimsAndReq(h.ListCourse)) // 指定课程的课程评价
	eg.GET("/list/mine", authMiddleware, ginx.WrapClaimsAndReq(h.ListMine))
	eg.GET("/count/courses/:courseId/invisible", ginx.Wrap(h.CountCourseInvisible))
	eg.GET("/count/mine", authMiddleware, ginx.WrapClaimsAndReq(h.CountMine))
	eg.GET("/:evaluationId/detail", authMiddleware, ginx.WrapClaims(h.Detail))
	eg.POST("/:evaluationId/endorse", authMiddleware, ginx.WrapClaimsAndReq(h.Endorse))
}

// @Summary 发布课评
// @Description
// @Tags 课评
// @Accept json
// @Produce json
// @Param request body SaveReq true "发布课评请求体"
// @Success 200 {object} ginx.Result{data=int64} "Success"
// @Router /evaluations/save [post]
func (h *EvaluationHandler) Save(ctx *gin.Context, req SaveReq, uc ijwt.UserClaims) (ginx.Result, error) {
	// 这里要校验参数 1. content 长度 2. 星级是必选项
	status, ok := evaluationv1.EvaluationStatus_value[req.Status]
	if !ok || req.Status == evaluationv1.EvaluationStatus_Folded.String() {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "不合法的课评状态",
		}, errors.New("不合法的课评状态")
	}
	if req.Id == 0 && req.Status != evaluationv1.EvaluationStatus_Public.String() {
		// 创建时 status 必须为 public
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "创建时必须以Public状态创建",
		}, errors.New("非Public创建")
	}
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

	var (
		res     *evaluationv1.SaveResponse
		saveErr error
	)
	// 下面涉及两个服务的原子性调用，需要使用分布式事务，这里的bff其实起到了聚合服务的作用...，引入实际意义聚合服务，目前没必要
	// go的seatago框架相当不成熟，比如这个事务内部不能用errgroup并发这两个attach tag
	err := tm.WithGlobalTx(ctx,
		&tm.GtxConfig{
			Timeout: 1000 * time.Second, // todo
			Name:    "ATPublishAndTagTx",
		},
		func(ctx context.Context) error {
			res, saveErr = h.evaluationClient.Save(ctx, &evaluationv1.SaveRequest{
				Evaluation: &evaluationv1.Evaluation{
					Id:          req.Id,
					PublisherId: uc.Uid,
					CourseId:    req.CourseId,
					StarRating:  uint32(req.StarRating),
					Content:     req.Content,
					Status:      evaluationv1.EvaluationStatus(status),
				},
			})
			if saveErr != nil {
				return saveErr
			}
			var er error
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
	switch {
	case err == nil:
		return ginx.Result{
			Msg:  "Success",
			Data: res.GetEvaluationId(), // 这里给前端标明是evaluationId
		}, nil
		// 检验saveErr
	case evaluationv1.IsCanNotEvaluateUnattendedCourse(saveErr):
		return ginx.Result{
			Code: errs.EvaluationPermissionDenied,
			Msg:  "不能评价未上过的课程",
		}, saveErr
	default:
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
}

// @Summary 变更课评状态
// @Description
// @Tags 课评
// @Accept json
// @Produce json
// @Param request body UpdateStatusReq true "变更课评状态请求体"
// @Success 200 {object} ginx.Result "Success"
// @Router /evaluations/{evaluationId}/status [post]
func (h *EvaluationHandler) UpdateStatus(ctx *gin.Context, req UpdateStatusReq, uc ijwt.UserClaims) (ginx.Result, error) {
	eidStr := ctx.Param("evaluationId")
	eid, err := strconv.ParseInt(eidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
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
			Code: errs.EvaluationInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	res, err := h.evaluationClient.Detail(ctx, &evaluationv1.DetailRequest{
		EvaluationId: eid,
	})
	if err != nil {
		if evaluationv1.IsEvaluationNotFound(err) {
			return ginx.Result{
				Code: errs.EvaluationNotFound,
				Msg:  "课评不存在",
			}, err
		} else {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, err
		}
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
		eg           errgroup.Group
		evaluationVo = EvaluationVo{
			Id:          res.GetEvaluation().GetId(),
			PublisherId: res.GetEvaluation().GetPublisherId(),
			CourseId:    res.GetEvaluation().GetCourseId(),
			StarRating:  res.GetEvaluation().GetStarRating(),
			Content:     res.GetEvaluation().GetContent(),
			Status:      res.GetEvaluation().GetStatus().String(),
			Utime:       res.GetEvaluation().GetUtime(),
			Ctime:       res.GetEvaluation().GetCtime(),
		}
	)
	// 聚合考核方式
	eg.Go(func() error {
		atRes, er := h.tagClient.GetAssessmentTagsByTaggerBiz(ctx, &tagv1.GetAssessmentTagsByTaggerBizRequest{
			TaggerId: res.GetEvaluation().GetPublisherId(),
			Biz:      tagv1.Biz_Course,
			BizId:    res.GetEvaluation().GetCourseId(),
		})
		if er != nil {
			return er
		}
		evaluationVo.Assessments = slice.Map(atRes.GetTags(), func(idx int, src tagv1.AssessmentTag) string {
			return src.String()
		})
		return nil
	})
	// 聚合课程特点
	eg.Go(func() error {
		ftRes, er := h.tagClient.GetFeatureTagsByTaggerBiz(ctx, &tagv1.GetFeatureTagsByTaggerBizRequest{
			TaggerId: res.GetEvaluation().GetPublisherId(),
			Biz:      tagv1.Biz_Course,
			BizId:    res.GetEvaluation().GetCourseId(),
		})
		if er != nil {
			return er
		}
		evaluationVo.Features = slice.Map(ftRes.GetTags(), func(idx int, src tagv1.FeatureTag) string {
			return src.String()
		})
		return nil
	})
	// 还要聚合interact数据，voting -1、0、1
	// 支持数，反对数，评论数
	// 聚合表态信息
	// 设置了可受限访问，所以要区分游客和登录用户
	if uc.Uid != 0 {
		eg.Go(func() error {
			stanceRes, er := h.stanceClient.GetUserStance(ctx, &stancev1.GetUserStanceRequest{
				Uid:   uc.Uid,
				Biz:   stancev1.Biz_Evaluation,
				BizId: eid,
			})
			if er != nil {
				return er
			}
			evaluationVo.Stance = int32(stanceRes.GetStance())
			evaluationVo.TotalSupportCount = stanceRes.GetTotalSupports()
			evaluationVo.TotalOpposeCount = stanceRes.GetTotalOpposes()
			return nil
		})
	} else {
		eg.Go(func() error {
			countStanceRes, er := h.stanceClient.CountStance(ctx, &stancev1.CountStanceRequest{
				Biz:   stancev1.Biz_Evaluation,
				BizId: eid,
			})
			if er != nil {
				return er
			}
			evaluationVo.TotalSupportCount = countStanceRes.GetTotalSupports()
			evaluationVo.TotalOpposeCount = countStanceRes.GetTotalOpposes()
			return nil
		})
	}
	// 评论数
	eg.Go(func() error {
		countCommentRes, er := h.commentClient.CountComment(ctx, &commentv1.CountCommentRequest{
			Biz:   commentv1.Biz_Evaluation,
			BizId: eid,
		})
		if er != nil {
			return er
		}
		evaluationVo.TotalCommentCount = countCommentRes.GetCount()
		return nil
	})
	err = eg.Wait()
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: evaluationVo,
	}, nil
}
