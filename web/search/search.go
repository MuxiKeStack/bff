package search

import (
	"fmt"
	searchv1 "github.com/MuxiKeStack/be-api/gen/proto/search/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	// 路由策略表
	client     searchv1.SearchServiceClient
	strategies map[string]SearchStrategy
}

func (h *SearchHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/search")
	sg.GET("", authMiddleware, ginx.WrapClaimsAndReq(h.Search))
	sg.GET("/history", authMiddleware, ginx.WrapClaimsAndReq(h.GetHistory))       // 历史记录，写死，返回十条
	sg.DELETE("/history", authMiddleware, ginx.WrapClaimsAndReq(h.DeleteHistory)) // 删除历史记录
}

func NewSearchHandler(client searchv1.SearchServiceClient) *SearchHandler {
	strategies := map[string]SearchStrategy{
		"Course": &CourseSearchStrategy{client: client},
	}
	return &SearchHandler{
		client:     client,
		strategies: strategies,
	}
}

func (h *SearchHandler) Search(ctx *gin.Context, req SearchReq, uc ijwt.UserClaims) (ginx.Result, error) {
	// 可以约束一下boxId
	strategy, exists := h.strategies[req.Biz]
	if !exists {
		return ginx.Result{
			Code: errs.SearchInvalidInput,
			Msg:  "非法业务类型",
		}, fmt.Errorf("不支持的业务类型: %s", req.Biz)
	}
	return strategy.Search(ctx, req.Keyword, uc.Uid, req.SearchLocation)
}

func (h *SearchHandler) GetHistory(ctx *gin.Context, req GetHistoryReq, uc ijwt.UserClaims) (ginx.Result, error) {
	location, exists := searchv1.SearchLocation_value[req.SearchLocation]
	if !exists {
		return ginx.Result{
			Code: errs.SearchInvalidInput,
			Msg:  "非法的location",
		}, fmt.Errorf("不支持的location: %d", location)
	}
	// 写死，返回十条，包括被删除的，然后筛掉
	res, err := h.client.GetUserSearchHistories(ctx, &searchv1.GetUserHistoryRequest{
		Uid:      uc.Uid,
		Location: searchv1.SearchLocation(location),
		Offset:   0,
		Limit:    10,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	// 排除不可见
	historyVos := make([]HistoryVo, 0, len(res.GetHistories()))
	for _, history := range res.GetHistories() {
		if history.GetStatus() == searchv1.VisibilityStatus_Visible {
			historyVos = append(historyVos, HistoryVo{
				Id:      history.Id,
				Keyword: history.Keyword,
			})
		}
	}
	return ginx.Result{
		Msg:  "Success",
		Data: historyVos,
	}, nil
}

func (h *SearchHandler) DeleteHistory(ctx *gin.Context, req DeleteHistoryReq, uc ijwt.UserClaims) (ginx.Result, error) {
	// 标记为不可见
	location, exists := searchv1.SearchLocation_value[req.SearchLocation]
	if !exists {
		return ginx.Result{
			Code: errs.SearchInvalidInput,
			Msg:  "非法的location",
		}, fmt.Errorf("不支持的location: %d", location)
	}
	_, err := h.client.HideUserSearchHistories(ctx, &searchv1.HideUserSearchHistoriesRequest{
		Uid:        uc.Uid,
		Location:   searchv1.SearchLocation(location),
		RemoveAll:  req.RemoveAll,
		HistoryIds: req.HistoryIds,
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
