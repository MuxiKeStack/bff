package web

import (
	"fmt"
	questionv1 "github.com/MuxiKeStack/be-api/gen/proto/question/v1"
	userv1 "github.com/MuxiKeStack/be-api/gen/proto/user/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/pkg/logger"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"strconv"
)

type QuestionHandler struct {
	question questionv1.QuestionServiceClient
	user     userv1.UserServiceClient
	l        logger.Logger
}

func NewQuestionHandler(question questionv1.QuestionServiceClient, user userv1.UserServiceClient, l logger.Logger) *QuestionHandler {
	return &QuestionHandler{question: question, user: user, l: l}
}

func (h *QuestionHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	qg := s.Group("/questions")
	qg.POST("/publish", authMiddleware, ginx.WrapClaimsAndReq(h.Publish))
	qg.GET("/:questionId/detail", ginx.Wrap(h.Detail))
	qg.GET("/:questionId/recommendation_invitees", ginx.WrapReq(h.RecommendationInvitees))
	qg.POST("/:questionId/invitees", authMiddleware, ginx.WrapClaimsAndReq(h.InviteUserToAnswer))
}

// Publish 发布一个新问题
// @Summary 发布新问题
// @Description
// @Tags 问题
// @Accept json
// @Produce json
// @Param request body QuestionPublishReq true "发布问题请求体"
// @Success 200 {object} ginx.Result{data=int64} "Success"
// @Router /questions/publish [post]
func (h *QuestionHandler) Publish(ctx *gin.Context, req QuestionPublishReq, uc ijwt.UserClaims) (ginx.Result, error) {
	bizId, ok := questionv1.Biz_value[req.Biz]
	if !ok {
		return ginx.Result{
			Code: errs.QuestionBizNotFound,
			Msg:  "未找到业务",
		}, fmt.Errorf("未找到业务: %s", req.Biz)
	}
	res, err := h.question.Publish(ctx, &questionv1.PublishRequest{
		Question: &questionv1.Question{
			QuestionerId: uc.Uid,
			Biz:          questionv1.Biz(bizId),
			BizId:        req.BizId,
			Content:      req.Content,
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
		Data: res.GetQuestionId(), // 这里给前端标明是questionId
	}, nil
}

// RecommendationInvitees 获取问题的推荐邀请。
// @Summary 获取推荐邀请人
// @Description 为特定问题检索推荐的邀请人列表。
// @Tags 问题
// @Accept json
// @Produce json
// @Param questionId path int true "问题ID"
// @Param limit query int true "邀请人数限制" default(10)
// @Param cur_uid query int true "当前id，游标的感觉" default(0)
// @Success 200 {object} ginx.Result{data=[]InviteesVo} "Success"
// @Router /questions/{questionId}/recommendation_invitees [get]
func (h *QuestionHandler) RecommendationInvitees(ctx *gin.Context, req RecommendationInviteesReq) (ginx.Result, error) {
	qidStr := ctx.Param("questionId")
	qid, err := strconv.ParseInt(qidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	// 用cid拿到，上过该课的用户uids
	res, err := h.question.GetRecommendationInviteeUids(ctx, &questionv1.GetRecommendationInviteeUidsRequest{
		QuestionId: qid,
		CurUid:     req.CurUid,
		Limit:      req.Limit,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	// 在这里聚合用户信息
	inviteesVos := slice.Map(res.GetInviteeUids(), func(idx int, src int64) InviteesVo {
		// 降级了的话可以直接不聚合
		res, err := h.user.Profile(ctx, &userv1.ProfileRequest{
			Uid: src,
		})
		if err != nil {
			// 因为用户具体信息作为可选项，这里error也不是很影响，我采取不return
			h.l.Error("聚合用户信息失败", logger.Error(err), logger.Int64("uid", src))
		}
		return InviteesVo{
			Uid:      src,
			Nickname: res.User.Nickname,
			Avatar:   res.User.Avatar,
		}
	})
	return ginx.Result{
		Msg:  "Success",
		Data: inviteesVos,
	}, nil
}

// Detail 获取特定问题的详情。
// @Summary 获取问题详情
// @Description 通过问题ID检索特定问题的详情。
// @Tags 问题
// @Accept json
// @Produce json
// @Param questionId path int true "问题ID"
// @Success 200 {object} ginx.Result{data=QuestionVo} "Success"
// @Router /questions/{questionId}/detail [get]
func (h *QuestionHandler) Detail(ctx *gin.Context) (ginx.Result, error) {
	qidStr := ctx.Param("questionId")
	qid, err := strconv.ParseInt(qidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	res, err := h.question.GetDetailById(ctx, &questionv1.GetDetailByIdRequest{
		QuestionId: qid,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: QuestionVo{
			Id:           res.GetQuestion().GetId(),
			QuestionerId: res.GetQuestion().GetQuestionerId(),
			Biz:          res.GetQuestion().GetBiz().String(),
			BizId:        res.GetQuestion().GetBizId(),
			Content:      res.GetQuestion().GetContent(),
		},
	}, nil
}

// InviteUserToAnswer 邀请用户回答问题。
// @Summary 邀请回答问题
// @Description 邀请一个或多个用户回答特定问题。
// @Tags 问题
// @Accept json
// @Produce json
// @Param questionId path int true "问题ID"
// @Param request body InviteUserToAnswerReq true "邀请回答问题请求体"
// @Success 200 {object} ginx.Result "Success"
// @Router /questions/{questionId}/invitees [post]
func (h *QuestionHandler) InviteUserToAnswer(ctx *gin.Context, req InviteUserToAnswerReq, uc ijwt.UserClaims) (ginx.Result, error) {
	qidStr := ctx.Param("questionId")
	qid, err := strconv.ParseInt(qidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	_, err = h.question.InviteUserToAnswer(ctx, &questionv1.InviteUserToAnswerRequest{
		Inviter:    uc.Uid,
		Invitees:   req.Invitees,
		QuestionId: qid,
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
