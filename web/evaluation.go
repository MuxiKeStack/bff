package web

import (
	"context"
	"errors"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	tagv1 "github.com/MuxiKeStack/be-api/gen/proto/tag/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/seata/seata-go/pkg/tm"
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
	eg.POST("/publish", authMiddleware, ginx.WrapClaimsAndReq(h.Publish))
}

// @Summary 发布课评
// @Description
// @Tags 课评
// @Accept json
// @Produce json
// @Param request body EvaluationPublishReq true "发布课评请求体"
// @Success 200 {object} ginx.Result{data=int64} "Success"
// @Router /evaluation/publish [post]
func (h *EvaluationHandler) Publish(ctx *gin.Context, req EvaluationPublishReq, uc ijwt.UserClaims) (ginx.Result, error) {
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
	var (
		res *evaluationv1.PublishResponse
		err error
	)
	// 下面涉及两个服务的原子性调用，需要使用分布式事务，所以这里的bff其实起到了聚合服务的作用...，引入实际意义聚合服务，目前没必要
	err = tm.WithGlobalTx(ctx,
		&tm.GtxConfig{
			Timeout: 1000 * time.Second, // todo
			Name:    "PublishAndTagTx",
		},
		func(ctx context.Context) error {
			var er error
			res, er = h.evaluationClient.Publish(ctx, &evaluationv1.PublishRequest{
				Evaluation: &evaluationv1.Evaluation{
					PublisherId: uc.Uid,
					CourseId:    req.CourseId, // TODO 下面这个地方要用外键
					StarRating:  uint32(req.StarRating),
					Content:     req.Content,
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
