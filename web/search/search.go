package search

import (
	"errors"
	"fmt"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	searchv1 "github.com/MuxiKeStack/be-api/gen/proto/search/v1"
	tagv1 "github.com/MuxiKeStack/be-api/gen/proto/tag/v1"
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
	sg.GET("/history", authMiddleware, ginx.WrapClaimsAndReq(h.GetHistory))    // 历史记录，写死，返回十条
	sg.PUT("/history", authMiddleware, ginx.WrapClaimsAndReq(h.DeleteHistory)) // 删除历史记录
}

func NewSearchHandler(client searchv1.SearchServiceClient, tagClient tagv1.TagServiceClient,
	evaluationClient evaluationv1.EvaluationServiceClient) *SearchHandler {
	strategies := map[string]SearchStrategy{
		"Course": &CourseSearchStrategy{
			searchClient:     client,
			tagClient:        tagClient,
			evaluationClient: evaluationClient,
		},
	}
	return &SearchHandler{
		client:     client,
		strategies: strategies,
	}
}

// Search 搜索请求处理
// @Summary 执行搜索
// @Description 根据提供的业务类型和关键词执行搜索操作
// @Tags 搜索
// @Accept json
// @Produce json
// @Param biz query string true "业务类型，Course"
// @Param keyword query string true "搜索关键词"
// @Param search_location query string true "搜索位置: Home，Collections"
// @Success 200 {object} ginx.Result{data=[]CourseVo} "返回搜索结果"
// @Router /search [get]
func (h *SearchHandler) Search(ctx *gin.Context, req SearchReq, uc ijwt.UserClaims) (ginx.Result, error) {
	if len([]rune(req.Keyword)) > 15 {
		return ginx.Result{
			Code: errs.SearchInvalidInput,
			Msg:  "搜索长度不应大于15字",
		}, errors.New("搜索长度过长")
	}
	// 可以约束一下boxId
	strategy, exists := h.strategies[req.Biz]
	if !exists {
		return ginx.Result{
			Code: errs.SearchInvalidInput,
			Msg:  "非法业务类型",
		}, fmt.Errorf("不支持的业务类型: %s", req.Biz)
	}
	if len(req.Keyword) <= 0 {
		return ginx.Result{
			Code: errs.SearchInvalidInput,
			Msg:  "keyword不能为空",
		}, errors.New("keyword为空")
	}
	return strategy.Search(ctx, req.Keyword, uc.Uid, req.SearchLocation)
}

// GetHistory 获取搜索历史
// @Summary 获取搜索历史
// @Description 返回用户的搜索历史记录
// @Tags 搜索
// @Accept json
// @Produce json
// @Param search_location query string true "搜索位置: Home，Collections"
// @Success 200 {object} ginx.Result{data=[]HistoryVo} "返回搜索历史记录"
// @Router /search/history [get]
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

// DeleteHistory 删除搜索历史
// @Summary 删除搜索历史
// @Description 根据请求删除用户的搜索历史记录
// @Tags 搜索
// @Accept json
// @Produce json
// @Param body body DeleteHistoryReq true "请求体"
// @Success 200 {object} ginx.Result "成功删除历史记录"
// @Router /search/history [put]
func (h *SearchHandler) DeleteHistory(ctx *gin.Context, req DeleteHistoryReq, uc ijwt.UserClaims) (ginx.Result, error) {
	location, exists := searchv1.SearchLocation_value[req.SearchLocation]
	if !exists {
		return ginx.Result{
			Code: errs.SearchInvalidInput,
			Msg:  "非法的location",
		}, fmt.Errorf("不支持的location: %d", location)
	}
	// 标记为不可见
	_, err := h.client.HideUserSearchHistories(ctx, &searchv1.HideUserSearchHistoriesRequest{
		Uid:        uc.Uid,
		Location:   searchv1.SearchLocation(location),
		RemoveAll:  req.RemoveAll,
		HistoryIds: req.RemoveHistoryIds,
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
