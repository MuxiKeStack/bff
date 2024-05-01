package web

import (
	"errors"
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
	ag.DELETE("/:answerId", authMiddleware, ginx.WrapClaims(h.DelAnswer))
	ag.GET("/:answerId/detail", authMiddleware, ginx.WrapClaims(h.Detail))
	ag.GET("/list/questions/:questionId", authMiddleware, ginx.WrapClaimsAndReq(h.ListForQuestion))
	ag.GET("/list/mine", authMiddleware, ginx.WrapClaimsAndReq(h.ListForMine))
	ag.POST("/:answerId/endorse", authMiddleware, ginx.WrapClaimsAndReq(h.Endorse))
}

// Publish 发布一个新答案
// @Summary 发布新回答
// @Description 用户发布答案到指定问题
// @Tags 回答
// @Accept json
// @Produce json
// @Param request body AnswerPublishReq true "发布答案请求体"
// @Success 200 {object} ginx.Result{data=int64} "成功返回"
// @Router /answers/publish [post]
func (h *AnswerHandler) Publish(ctx *gin.Context, req AnswerPublishReq, uc ijwt.UserClaims) (ginx.Result, error) {
	if len([]rune(req.Content)) > 200 {
		return ginx.Result{
			Code: errs.AnswerInvalidInput,
			Msg:  "回答不能超过200个字符",
		}, errors.New("回答超过了200个字符")
	}
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
	publishRes, err := h.answerClient.Publish(ctx, &answerv1.PublishRequest{
		Answer: &answerv1.Answer{
			PublisherId: uc.Uid,
			QuestionId:  req.QuestionId,
			Content:     req.Content,
		},
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: publishRes.GetAnswerId(),
	}, nil
}

// Detail 获取回答详情
// @Summary 获取回答详情
// @Description 通过答案ID检索特定回答的详情
// @Tags 回答
// @Accept json
// @Produce json
// @Param answerId path int64 true "答案ID"
// @Success 200 {object} ginx.Result{data=AnswerVo} "成功返回答案详情"
// @Router /answers/{answerId}/detail [get]
func (h *AnswerHandler) Detail(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	aidStr := ctx.Param("answerId")
	aid, err := strconv.ParseInt(aidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.AnswerInvalidInput,
			Msg:  "不合法的answerId",
		}, err
	}
	var (
		eg         errgroup.Group
		answerRes  *answerv1.DetailResponse
		commentRes *commentv1.CountCommentResponse
		stanceRes  *stancev1.GetUserStanceResponse
	)
	eg.Go(func() error {
		var er error
		answerRes, er = h.answerClient.Detail(ctx, &answerv1.DetailRequest{
			AnswerId: aid,
		})
		return er
	})
	eg.Go(func() error {
		var er error
		commentRes, er = h.commentClient.CountComment(ctx, &commentv1.CountCommentRequest{
			Biz:   commentv1.Biz_Answer,
			BizId: aid,
		})
		return er
	})
	eg.Go(func() error {
		var er error
		stanceRes, er = h.stanceClient.GetUserStance(ctx, &stancev1.GetUserStanceRequest{
			Uid:   uc.Uid,
			Biz:   stancev1.Biz_Answer,
			BizId: aid,
		})
		return er
	})
	err = eg.Wait()
	switch {
	case err == nil:
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
				Utime:             answerRes.GetAnswer().GetUtime(),
				Ctime:             answerRes.GetAnswer().GetCtime(),
			},
		}, nil
	case answerv1.IsAnswerNotFound(err):
		return ginx.Result{
			Code: errs.AnswerNotFound,
			Msg:  "回答不存在",
		}, err
	default:
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
}

// ListForQuestion 回答列表[问题]
// @Summary 回答列表[问题]
// @Description 为特定问题检索所有相关回答的列表
// @Tags 回答
// @Accept json
// @Produce json
// @Param questionId path int64 true "问题ID"
// @Param cur_answer_id query int64 false "当前答案ID"
// @Param limit query int64 false "返回答案数量限制" default(10)
// @Success 200 {object} ginx.Result{data=[]AnswerVo} "成功返回答案列表"
// @Router /answers/list/questions/{questionId} [get]
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
			Utime:       src.GetUtime(),
			Ctime:       src.GetCtime(),
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

// ListForMine 回答列表[自己]
// @Summary 回答列表[自己]
// @Description 获取当前用户发布的所有回答的列表
// @Tags 回答
// @Accept json
// @Produce json
// @Param cur_answer_id query int64 false "当前答案ID"
// @Param limit query int64 false "返回答案数量限制" default(10)
// @Success 200 {object} ginx.Result{data=[]AnswerVo} "成功返回答案列表"
// @Router /answers/list/mine [get]
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
			Utime:       src.GetUtime(),
			Ctime:       src.GetCtime(),
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

// Endorse 为回答背书
// @Summary 为回答背书
// @Description 为指定回答表达支持或反对
// @Tags 回答
// @Accept json
// @Produce json
// @Param answerId path int64 true "答案ID"
// @Param stance body EndorseReq true "立场（支持或反对）"
// @Success 200 {object} ginx.Result "成功返回"
// @Router /answers/{answerId}/endorse [post]
func (h *AnswerHandler) Endorse(ctx *gin.Context, req EndorseReq, uc ijwt.UserClaims) (ginx.Result, error) {
	aidStr := ctx.Param("answerId")
	aid, err := strconv.ParseInt(aidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.AnswerInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	_, ok := stancev1.Stance_name[req.Stance]
	if !ok {
		return ginx.Result{
			Code: errs.AnswerInvalidInput,
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

func (h *AnswerHandler) DelAnswer(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	aidStr := ctx.Param("answerId")
	aid, err := strconv.ParseInt(aidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.AnswerInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	_, err = h.answerClient.DelAnswerById(ctx, &answerv1.DelAnswerByIdRequest{
		AnswerId: aid,
		Uid:      uc.Uid,
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
