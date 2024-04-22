package search

import (
	"context"
	"fmt"
	searchv1 "github.com/MuxiKeStack/be-api/gen/proto/search/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
)

type CourseSearchStrategy struct {
	client searchv1.SearchServiceClient
}

// Search 可用于普通搜索和收藏搜索
func (c *CourseSearchStrategy) Search(ctx context.Context, keyword string, uid int64, searchLocation string) (ginx.Result, error) {
	location, exists := searchv1.SearchLocation_value[searchLocation]
	if !exists {
		return ginx.Result{
			Code: errs.SearchInvalidInput,
			Msg:  "非法的location",
		}, fmt.Errorf("不支持的location: %d", location)
	}
	res, err := c.client.SearchCourse(ctx, &searchv1.SearchCourseRequest{
		Keyword:  keyword,
		Uid:      uid,
		Location: searchv1.SearchLocation(location),
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetCourses(),
	}, nil
}
