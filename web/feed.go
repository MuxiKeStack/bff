package web

import (
	feedv1 "github.com/MuxiKeStack/be-api/gen/proto/feed/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

type FeedHandler struct {
	feedClient feedv1.FeedServiceClient
}

func NewFeedHandler(feedClient feedv1.FeedServiceClient) *FeedHandler {
	return &FeedHandler{feedClient: feedClient}
}

func (h *FeedHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	fg := s.Group("/feed")
	fg.GET("/events_list", authMiddleware, ginx.WrapClaimsAndReq(h.GetFeedEventsList))

}

// GetFeedEventsList 拉取feed事件
// @Summary 拉取feed事件
// @Description 根据上一次事件ctime，进行增量拉取
// @Tags feed
// @Accept json
// @Produce json
// @Param last_time query int64 true "上一条消息提醒的发生时间ctime"
// @Param direction query string true "查询方向 Before 或 After last_time"
// @Param limit query int64 true "返回消息数量限制"
// @Success 200 {object} ginx.Result{data=[]feedv1.FeedEvent} "成功返回结果"
// @Router /feed/events_list [get]
func (h *FeedHandler) GetFeedEventsList(ctx *gin.Context, req GetFeedEventsListReq, uc ijwt.UserClaims) (ginx.Result, error) {
	if req.Limit > 100 {
		req.Limit = 100
	}
	direction, ok := feedv1.Direction_value[req.Direction]
	if !ok {
		return ginx.Result{
			Code: errs.FeedInvalidInput,
			Msg:  "不合法的查询方向",
		}, nil
	}
	res, err := h.feedClient.FindFeedEvents(ctx, &feedv1.FindFeedEventsRequest{
		Uid:       uc.Uid,
		LastTime:  req.LastTime,
		Direction: feedv1.Direction(direction),
		Limit:     req.Limit,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetFeedEvents(),
	}, nil
}
