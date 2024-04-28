package web

import (
	answerv1 "github.com/MuxiKeStack/be-api/gen/proto/answer/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"strconv"
)

type AnswerHandler struct {
	answerClient answerv1.AnswerServiceClient
}

func (h *AnswerHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	ag := s.Group("/answers")
	ag.POST("/publish", authMiddleware, ginx.WrapClaimsAndReq(h.Publish))
	ag.GET("/:answerId/detail", ginx.Wrap(h.Detail))
	ag.GET("/list/questions/:questionId", ginx.WrapReq(h.ListForQuestion))
}

func (h *AnswerHandler) Publish(ctx *gin.Context, req AnswerPublishReq, uc ijwt.UserClaims) (ginx.Result, error) {
	_, err := h.answerClient.Publish(ctx, &answerv1.PublishRequest{
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

func (h *AnswerHandler) Detail(ctx *gin.Context) (ginx.Result, error) {
	aidStr := ctx.Param("answerId")
	aid, err := strconv.ParseInt(aidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.AnswerInvalidInput,
			Msg:  "不合法的answerId",
		}, err
	}
	res, err := h.answerClient.Detail(ctx, &answerv1.DetailRequest{
		AnswerId: aid,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetAnswer(),
	}, nil

}

func (h *AnswerHandler) ListForQuestion(ctx *gin.Context, req AnswerListForQuestionReq) (ginx.Result, error) {
	qidStr := ctx.Param("questionId")
	qid, err := strconv.ParseInt(qidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.AnswerInvalidInput,
			Msg:  "不合法的answerId",
		}, err
	}
	res, err := h.answerClient.ListForQuestion(ctx, &answerv1.ListForQuestionRequest{
		QuestionId:  qid,
		CurAnswerId: req.CurAnswerId,
		Limit:       req.Limit,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetAnswers(),
	}, nil
}
