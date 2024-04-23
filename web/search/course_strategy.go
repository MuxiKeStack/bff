package search

import (
	"context"
	"fmt"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	searchv1 "github.com/MuxiKeStack/be-api/gen/proto/search/v1"
	tagv1 "github.com/MuxiKeStack/be-api/gen/proto/tag/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/ecodeclub/ekit/slice"
	"golang.org/x/sync/errgroup"
)

type CourseSearchStrategy struct {
	searchClient     searchv1.SearchServiceClient
	tagClient        tagv1.TagServiceClient
	evaluationClient evaluationv1.EvaluationServiceClient
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
	res, err := c.searchClient.SearchCourse(ctx, &searchv1.SearchCourseRequest{
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
	courseVos := slice.Map(res.GetCourses(), func(idx int, src *searchv1.Course) CourseVo {
		return CourseVo{
			Id:             src.GetId(),
			Name:           src.GetName(),
			Teacher:        src.GetTeacher(),
			CompositeScore: src.GetCompositeScore(),
		}
	})
	// 要去聚合一下标签信息，因为es里面没这个
	var eg errgroup.Group
	for i, _ := range courseVos {
		eg.Go(func() error {
			publishersRes, er := c.evaluationClient.VisiblePublishersCourse(ctx, &evaluationv1.VisiblePublishersCourseRequest{
				CourseId: courseVos[i].Id,
			})
			if er != nil {
				return er
			}
			caRes, er := c.tagClient.CountAssessmentTagsByCourseTagger(ctx, &tagv1.CountAssessmentTagsByCourseTaggerRequest{
				CourseId:  courseVos[i].Id,
				TaggerIds: publishersRes.GetPublishers(),
			})
			if er != nil {
				return er
			}
			cfRes, er := c.tagClient.CountFeatureTagsByCourseTagger(ctx, &tagv1.CountFeatureTagsByCourseTaggerRequest{
				CourseId:  courseVos[i].Id,
				TaggerIds: publishersRes.GetPublishers(),
			})
			if er != nil {
				return er
			}
			courseVos[i].Assessments = slice.ToMapV(caRes.GetItems(), func(element *tagv1.CountAssessmentItem) (string, int64) {
				return element.GetTag().String(), element.GetCount()
			})
			courseVos[i].Features = slice.ToMapV(cfRes.GetItems(), func(element *tagv1.CountFeatureItem) (string, int64) {
				return element.GetTag().String(), element.GetCount()
			})
			return nil
		})
	}
	err = eg.Wait()
	return ginx.Result{
		Msg:  "Success",
		Data: courseVos,
	}, nil
}
