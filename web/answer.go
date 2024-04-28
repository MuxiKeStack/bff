package web

import (
	answerv1 "github.com/MuxiKeStack/be-api/gen/proto/answer/v1"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	coursev1 "github.com/MuxiKeStack/be-api/gen/proto/course/v1"
	questionv1 "github.com/MuxiKeStack/be-api/gen/proto/question/v1"
	stancev1 "github.com/MuxiKeStack/be-api/gen/proto/stance/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"strconv"
)

type AnswerHandler struct {
	answerClient   answerv1.AnswerServiceClient
	courseClient   coursev1.CourseServiceClient
	questionClient questionv1.QuestionServiceClient
	commentClient  commentv1.CommentServiceClient
	stanceClient   stancev1.StanceServiceClient
}

func NewAnswerHandler(answerClient answerv1.AnswerServiceClient, courseClient coursev1.CourseServiceClient,
	questionClient questionv1.QuestionServiceClient, commentClient commentv1.CommentServiceClient,
	stanceClient stancev1.StanceServiceClient) *AnswerHandler {
	return &AnswerHandler{
		answerClient:   answerClient,
		courseClient:   courseClient,
		questionClient: questionClient,
		commentClient:  commentClient,
		stanceClient:   stanceClient,
	}
}

func (h *AnswerHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	ag := s.Group("/answers")
	ag.POST("/publish", authMiddleware, ginx.WrapClaimsAndReq(h.Publish))
	ag.GET("/:answerId/detail", authMiddleware, ginx.WrapClaims(h.Detail))
	ag.GET("/list/questions/:questionId", authMiddleware, ginx.WrapClaimsAndReq(h.ListForQuestion))
	ag.GET("/list/mine", authMiddleware, ginx.WrapClaimsAndReq(h.ListForMine))
	ag.POST("/:answerId/endorse", authMiddleware, ginx.WrapClaimsAndReq(h.Endorse))
}

func (h *AnswerHandler) Publish(ctx *gin.Context, req AnswerPublishReq, uc ijwt.UserClaims) (ginx.Result, error) {
	questionRes, err := h.questionClient.GetDetailById(ctx, &questionv1.GetDetailByIdRequest{
		QuestionId: req.QuestionId,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	if questionRes.GetQuestion().GetBiz() == questionv1.Biz_Course {
		// 如果是课程的问题那么需要，上过才能回答
		subscribedRes, er := h.courseClient.Subscribed(ctx, &coursev1.SubscribedRequest{
			Uid:      uc.Uid,
			CourseId: questionRes.GetQuestion().GetBizId(),
		})
		if er != nil {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, er
		}
		if !subscribedRes.GetSubscribed() {
			return ginx.Result{
				Code: errs.AnswerPermissionDenied,
				Msg:  "不能回答未上过的课",
			}, er
		}
	}
	_, err = h.answerClient.Publish(ctx, &answerv1.PublishRequest{
		PublisherId: uc.Uid,
		QuestionId:  req.QuestionId,
		Content:     req.Content,
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

func (h *AnswerHandler) Detail(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	aidStr := ctx.Param("answerId")
	aid, err := strconv.ParseInt(aidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.AnswerInvalidInput,
			Msg:  "不合法的answerId",
		}, err
	}
	answerRes, err := h.answerClient.Detail(ctx, &answerv1.DetailRequest{
		AnswerId: aid,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	commentRes, err := h.commentClient.CountComment(ctx, &commentv1.CountCommentRequest{
		Biz:   commentv1.Biz_Answer,
		BizId: aid,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	stanceRes, err := h.stanceClient.GetUserStance(ctx, &stancev1.GetUserStanceRequest{
		Uid:   uc.Uid,
		Biz:   stancev1.Biz_Answer,
		BizId: aid,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: AnswerVo{
			Id:                answerRes.GetAnswer().GetId(),
			PublisherId:       answerRes.GetAnswer().GetPublisherId(),
			QuestionId:        answerRes.GetAnswer().GetQuestionId(),
			Content:           answerRes.GetAnswer().GetContent(),
			Stance:            int32(stanceRes.GetStance()),
			TotalSupportCount: stanceRes.GetTotalSupports(),
			TotalOpposeCount:  stanceRes.GetTotalOpposes(),
			TotalCommentCount: commentRes.GetCount(),
		},
	}, nil
}

func (h *AnswerHandler) ListForQuestion(ctx *gin.Context, req AnswerListReq, uc ijwt.UserClaims) (ginx.Result, error) {
	qidStr := ctx.Param("questionId")
	qid, err := strconv.ParseInt(qidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.AnswerInvalidInput,
			Msg:  "不合法的answerId",
		}, err
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	res, err := h.answerClient.ListForQuestion(ctx, &answerv1.ListForQuestionRequest{
		QuestionId:  qid,
		CurAnswerId: req.CurAnswerId,
		Limit:       req.Limit,
	})
	answerVos := slice.Map(res.GetAnswers(), func(idx int, src *answerv1.Answer) AnswerVo {
		return AnswerVo{
			Id:          src.GetId(),
			PublisherId: src.GetPublisherId(),
			QuestionId:  src.GetQuestionId(),
			Content:     src.GetContent(),
		}
	})
	var eg errgroup.Group
	for i := range answerVos {
		eg.Go(func() error {
			commentRes, er := h.commentClient.CountComment(ctx, &commentv1.CountCommentRequest{
				Biz:   commentv1.Biz_Answer,
				BizId: answerVos[i].Id,
			})
			if er != nil {
				return er
			}
			answerVos[i].TotalCommentCount = commentRes.GetCount()
			stanceRes, er := h.stanceClient.GetUserStance(ctx, &stancev1.GetUserStanceRequest{
				Uid:   uc.Uid,
				Biz:   stancev1.Biz_Answer,
				BizId: answerVos[i].Id,
			})
			if er != nil {
				return er
			}
			answerVos[i].Stance = int32(stanceRes.GetStance())
			answerVos[i].TotalSupportCount = stanceRes.GetTotalSupports()
			answerVos[i].TotalOpposeCount = stanceRes.GetTotalOpposes()
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
		Data: answerVos,
	}, nil
}

func (h *AnswerHandler) ListForMine(ctx *gin.Context, req AnswerListReq, uc ijwt.UserClaims) (ginx.Result, error) {
	if req.Limit > 100 {
		req.Limit = 100
	}
	res, err := h.answerClient.ListForUser(ctx, &answerv1.ListForUserRequest{
		Uid:         uc.Uid,
		CurAnswerId: req.CurAnswerId,
		Limit:       req.Limit,
	})
	answerVos := slice.Map(res.GetAnswers(), func(idx int, src *answerv1.Answer) AnswerVo {
		return AnswerVo{
			Id:          src.GetId(),
			PublisherId: src.GetPublisherId(),
			QuestionId:  src.GetQuestionId(),
			Content:     src.GetContent(),
		}
	})
	var eg errgroup.Group
	for i := range answerVos {
		eg.Go(func() error {
			commentRes, er := h.commentClient.CountComment(ctx, &commentv1.CountCommentRequest{
				Biz:   commentv1.Biz_Answer,
				BizId: answerVos[i].Id,
			})
			if er != nil {
				return er
			}
			answerVos[i].TotalCommentCount = commentRes.GetCount()
			stanceRes, er := h.stanceClient.GetUserStance(ctx, &stancev1.GetUserStanceRequest{
				Uid:   uc.Uid,
				Biz:   stancev1.Biz_Answer,
				BizId: answerVos[i].Id,
			})
			if er != nil {
				return er
			}
			answerVos[i].Stance = int32(stanceRes.GetStance())
			answerVos[i].TotalSupportCount = stanceRes.GetTotalSupports()
			answerVos[i].TotalOpposeCount = stanceRes.GetTotalOpposes()
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
		Data: answerVos,
	}, nil
}

func (h *AnswerHandler) Endorse(ctx *gin.Context, req EndorseReq, uc ijwt.UserClaims) (ginx.Result, error) {
	aidStr := ctx.Param("answerId")
	aid, err := strconv.ParseInt(aidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	_, ok := stancev1.Stance_name[req.Stance]
	if !ok {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "不合法的立场",
		}, err
	}
	_, err = h.stanceClient.Endorse(ctx, &stancev1.EndorseRequest{
		Uid:    uc.Uid,
		Biz:    stancev1.Biz_Answer,
		BizId:  aid,
		Stance: stancev1.Stance(req.Stance),
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
