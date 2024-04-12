package web

import (
	"errors"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	tagv1 "github.com/MuxiKeStack/be-api/gen/proto/tag/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

type EvaluationHandler struct {
	evaluationClient evaluationv1.EvaluationServiceClient
	tagClient        tagv1.TagServiceClient
}

func (h *EvaluationHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	eg := s.Group("/evaluation")
	eg.POST("/publish", ginx.WrapClaimsAndReq(h.Publish))
}

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
	res, err := h.evaluationClient.Publish(ctx, &evaluationv1.PublishRequest{
		Evaluation: &evaluationv1.Evaluation{
			PublisherId: uc.Uid,
			CourseId:    req.CourseId, // TODO 下面这个地方要用外键
			StarRating:  uint32(req.StarRating),
			Content:     req.Content,
		},
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	// 这里要去聚合 tag 服务，打两类标签
	if len(req.Assessments) > 0 {
		tags := make([]tagv1.AssessmentTag, 0, len(req.Assessments))
		for _, assessment := range req.Assessments {
			tag, ok := tagv1.AssessmentTag_value[assessment]
			if !ok {
				return ginx.Result{
					Code: errs.EvaluationInvalidInput,
					Msg:  "不合法的考核方式",
				}, errors.New("不合法的考核方式")
			}
			tags = append(tags, tagv1.AssessmentTag(tag))
		}
		_, er := h.tagClient.AttachAssessmentTags(ctx, &tagv1.AttachAssessmentTagsRequest{
			TaggerId: uc.Uid,
			Biz:      tagv1.Biz_Course,
			BizId:    req.CourseId, // 外键
			Tags:     tags,
		})
		if er != nil {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, er
		}
	}
	if len(req.Features) > 0 {
		tags := make([]tagv1.FeaturesTag, 0, len(req.Features))
		for _, feature := range req.Features {
			tag, ok := tagv1.FeaturesTag_value[feature]
			if !ok {
				return ginx.Result{
					Code: errs.EvaluationInvalidInput,
					Msg:  "不合法的课程特点",
				}, errors.New("不合法的课程特点")
			}
			tags = append(tags, tagv1.FeaturesTag(tag))
		}
		_, er := h.tagClient.AttachFeaturesTags(ctx, &tagv1.AttachFeaturesTagsRequest{
			TaggerId: uc.Uid,
			Biz:      tagv1.Biz_Course,
			BizId:    req.CourseId,
			Tags:     tags,
		})
		if er != nil {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, er
		}
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetEvaluationId(), // 这里给前端标明是evaluationId
	}, nil
}
