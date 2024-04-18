package web

import (
	"errors"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	commentClient commentv1.CommentServiceClient
}

func NewCommentHandler(commentClient commentv1.CommentServiceClient) *CommentHandler {
	return &CommentHandler{commentClient: commentClient}
}

func (h *CommentHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	cg := s.Group("/comments")
	cg.POST("/publish", ginx.WrapClaimsAndReq(h.Publish))
	cg.GET("/list", ginx.WrapReq(h.List))
	cg.GET("/replies/list", ginx.WrapReq(h.ListRelies))
	cg.GET("/count", ginx.WrapReq(h.Count)) // 这个数目要缓存好
}

func (h *CommentHandler) Publish(ctx *gin.Context, req CommentPublishReq, uc ijwt.UserClaims) (ginx.Result, error) {
	biz, ok := commentv1.Biz_value[req.Biz]
	if !ok {
		return ginx.Result{
			Code: errs.CommentInvalidInput,
			Msg:  "不合法的Biz(资源)类型",
		}, errors.New("不合法的Biz(资源)类型")
	}
	contentLen := len([]rune(req.Content))
	if contentLen < 1 {
		return ginx.Result{
			Code: errs.CommentInvalidInput,
			Msg:  "内容不能为空",
		}, errors.New("内容为空")
	}
	if contentLen > 300 {
		return ginx.Result{
			Code: errs.CommentInvalidInput,
			Msg:  "内容过长，不能超过300字符",
		}, errors.New("内容过长")
	}
	res, err := h.commentClient.CreateComment(ctx, &commentv1.CreateCommentRequest{
		Comment: &commentv1.Comment{
			CommentatorId: uc.Uid,
			Biz:           commentv1.Biz(biz),
			BizId:         req.BizId,
			Content:       req.Content,
			RootComment:   &commentv1.Comment{Id: req.RootId},
			ParentComment: &commentv1.Comment{Id: req.ParentId}, // 这里内部会根据pid来判断要不要聚合一个回复对象
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
		Data: res.GetCommentId(),
	}, nil
}

func (h *CommentHandler) List(ctx *gin.Context, req CommentListReq) (ginx.Result, error) {
	biz, ok := commentv1.Biz_value[req.Biz]
	if !ok {
		return ginx.Result{
			Code: errs.CommentInvalidInput,
			Msg:  "不合法的Biz(资源)类型",
		}, errors.New("不合法的Biz(资源)类型")
	}
	res, err := h.commentClient.GetCommentList(ctx, &commentv1.CommentListRequest{
		Biz:          commentv1.Biz(biz),
		BizId:        req.BizId,
		CurCommentId: req.CurCommentId,
		Limit:        req.Limit,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: slice.Map(res.GetComments(), func(idx int, src *commentv1.Comment) CommentVo {
			return CommentVo{
				Id:              src.GetId(),
				CommentatorId:   src.GetCommentatorId(),
				Biz:             src.GetBiz().String(),
				BizId:           src.GetBizId(),
				Content:         src.GetContent(),
				RootCommentId:   src.GetRootComment().GetId(),
				ParentCommentId: src.GetParentComment().GetId(),
				ReplyToUserId:   src.GetReplyToUserId(),
				Utime:           src.GetUtime(),
				Ctime:           src.GetCtime(),
			}
		}),
	}, nil
}

func (h *CommentHandler) ListRelies(ctx *gin.Context, req CommentListReliesReq) (ginx.Result, error) {
	res, err := h.commentClient.GetMoreReplies(ctx, &commentv1.GetMoreRepliesRequest{
		Rid:          req.RootId,
		CurCommentId: req.CurCommentId,
		Limit:        req.Limit,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: slice.Map(res.GetReplies(), func(idx int, src *commentv1.Comment) CommentVo {
			return CommentVo{
				Id:              src.GetId(),
				CommentatorId:   src.GetCommentatorId(),
				Biz:             src.GetBiz().String(),
				BizId:           src.GetBizId(),
				Content:         src.GetContent(),
				RootCommentId:   src.GetRootComment().GetId(),
				ParentCommentId: src.GetParentComment().GetId(),
				ReplyToUserId:   src.GetReplyToUserId(),
				Utime:           src.GetUtime(),
				Ctime:           src.GetCtime(),
			}
		}),
	}, nil
}

func (h *CommentHandler) Count(ctx *gin.Context, req CommentCountReq) (ginx.Result, error) {
	biz, ok := commentv1.Biz_value[req.Biz]
	if !ok {
		return ginx.Result{
			Code: errs.CommentInvalidInput,
			Msg:  "不合法的Biz(资源)类型",
		}, errors.New("不合法的Biz(资源)类型")
	}
	res, err := h.commentClient.CountComment(ctx, &commentv1.CountCommentRequest{
		Biz:   commentv1.Biz(biz),
		BizId: req.BizId,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetCount(),
	}, nil
}
