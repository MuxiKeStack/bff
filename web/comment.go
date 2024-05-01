package web

import (
	"errors"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"strconv"
)

type CommentHandler struct {
	commentClient commentv1.CommentServiceClient
}

func NewCommentHandler(commentClient commentv1.CommentServiceClient) *CommentHandler {
	return &CommentHandler{commentClient: commentClient}
}

func (h *CommentHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	cg := s.Group("/comments")
	cg.POST("/publish", authMiddleware, ginx.WrapClaimsAndReq(h.Publish))
	cg.GET("/list", ginx.WrapReq(h.List))
	cg.GET("/replies/list", ginx.WrapReq(h.ListReplies))
	cg.GET("/count", ginx.WrapReq(h.Count)) // 这个数目要缓存好
	cg.GET("/:commentId/detail", ginx.Wrap(h.GetDetailById))
	cg.DELETE("/:commentId", authMiddleware, ginx.WrapClaims(h.Delete))

}

// Publish 发布一个新评论
// @Summary 发布评论
// @Description 根据业务类型、业务ID、rootId，parentId发布评论。
// @Tags 评论
// @Accept json
// @Produce json
// @Param request body CommentPublishReq true "发布评论请求"
// @Success 200 {object} ginx.Result "Success"
// @Router /comments/publish [post]
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
	_, err := h.commentClient.CreateComment(ctx, &commentv1.CreateCommentRequest{
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
		Msg: "Success",
	}, nil
}

// List 列出评论
// @Summary 评论列表[一级]
// @Description 根据业务类型和业务ID列出一级评论。
// @Tags 评论
// @Accept json
// @Produce json
// @Param biz query string true "业务类型"
// @Param biz_id query int64 true "业务ID"
// @Param cur_comment_id query int64 false "当前评论ID"
// @Param limit query int64 false "返回数量限制"
// @Success 200 {object} ginx.Result{data=[]CommentVo} "成功返回评论列表"
// @Router /comments/list [get]
func (h *CommentHandler) List(ctx *gin.Context, req CommentListReq) (ginx.Result, error) {
	biz, ok := commentv1.Biz_value[req.Biz]
	if !ok {
		return ginx.Result{
			Code: errs.CommentInvalidInput,
			Msg:  "不合法的Biz(资源)类型",
		}, errors.New("不合法的Biz(资源)类型")
	}
	if req.Limit > 100 {
		req.Limit = 100
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
				ReplyToUid:      src.GetReplyToUid(),
				Utime:           src.GetUtime(),
				Ctime:           src.GetCtime(),
			}
		}),
	}, nil
}

// ListReplies 列出回复
// @Summary 评论列表[二级]
// @Description 实际上二级及其以下的所有都拍平到二级进行返回了。
// @Tags 评论
// @Accept json
// @Produce json
// @Param root_id query int64 true "根评论ID"
// @Param cur_comment_id query int64 false "当前评论ID"
// @Param limit query int64 false "返回数量限制"
// @Success 200 {object} ginx.Result{data=[]CommentVo} "成功返回评论列表"
// @Router /comments/replies/list [get]
func (h *CommentHandler) ListReplies(ctx *gin.Context, req CommentListReliesReq) (ginx.Result, error) {
	if req.Limit > 100 {
		req.Limit = 100
	}
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
				ReplyToUid:      src.GetReplyToUid(),
				Utime:           src.GetUtime(),
				Ctime:           src.GetCtime(),
			}
		}),
	}, nil
}

// Count 计数评论
// @Summary 评论数
// @Description 根据业务类型和业务ID计数评论。
// @Tags 评论
// @Accept json
// @Produce json
// @Param biz query string true "业务类型"
// @Param biz_id query int64 true "业务ID"
// @Success 200 {object} ginx.Result{data=int64} "成功返回评论数"
// @Router /comments/count [get]
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

// Delete 删除评论
// @Summary 删除评论
// @Description 根据评论ID删除评论。
// @Tags 评论
// @Accept json
// @Produce json
// @Param commentId path int64 true "评论ID"
// @Success 200 {object} ginx.Result "成功删除评论"
// @Router /comments/{commentId} [delete]
func (h *CommentHandler) Delete(ctx *gin.Context, uc ijwt.UserClaims) (ginx.Result, error) {
	cidStr := ctx.Param("commentId")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	_, err = h.commentClient.DeleteComment(ctx, &commentv1.DeleteCommentRequest{
		CommentId: cid,
		Uid:       uc.Uid,
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

// GetDetailById 获取评论详情
// @Summary 获取评论详情
// @Description 根据评论ID获取评论。
// @Tags 评论
// @Accept json
// @Produce json
// @Param commentId path int64 true "评论ID"
// @Success 200 {object} ginx.Result{data=CommentVo} "成功删除评论"
// @Router /comments/{commentId}/detail [get]
func (h *CommentHandler) GetDetailById(ctx *gin.Context) (ginx.Result, error) {
	cidStr := ctx.Param("commentId")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.CourseInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	res, err := h.commentClient.GetComment(ctx, &commentv1.GetCommentRequest{
		CommentId: cid,
	})
	switch {
	case err == nil:
		return ginx.Result{
			Msg: "Success",
			Data: CommentVo{
				Id:              res.GetComment().GetId(),
				CommentatorId:   res.GetComment().GetCommentatorId(),
				Biz:             res.GetComment().GetBiz().String(),
				BizId:           res.GetComment().GetBizId(),
				Content:         res.GetComment().GetContent(),
				RootCommentId:   res.GetComment().GetRootComment().GetId(),
				ParentCommentId: res.GetComment().GetParentComment().GetId(),
				ReplyToUid:      res.GetComment().GetReplyToUid(),
				Utime:           res.GetComment().GetUtime(),
				Ctime:           res.GetComment().GetCtime(),
			},
		}, nil
	case commentv1.IsCommentNotFound(err):
		return ginx.Result{
			Code: errs.CommentNotFound,
			Msg:  "评论不存在",
			Data: nil,
		}, err
	default:
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
}
