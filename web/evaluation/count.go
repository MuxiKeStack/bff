package evaluation

import (
	"errors"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"strconv"
)

// CountCourseInvisible 计算指定课程的不可见评价数量。
// @Summary 不可见课评数
// @Description 根据课程ID计算该课程的不可见评价数量。
// @Tags 课评
// @Accept json
// @Produce json
// @Param courseId path int64 true "课程ID"
// @Success 200 {object} ginx.Result{data=int64} "Success"
// @Router /evaluations/count/courses/{courseId}/invisible [get]
func (h *EvaluationHandler) CountCourseInvisible(ctx *gin.Context) (ginx.Result, error) {
	cidStr := ctx.Param("courseId")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	res, err := h.evaluationClient.CountCourseInvisible(ctx, &evaluationv1.CountCourseInvisibleRequest{CourseId: cid})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetCount(), // 标明是count
	}, nil
}

// CountMine 统计用户的课评数量。
// @Summary 用户课评数
// @Description 根据用户ID和课评状态分类统计用户的课评数量。
// @Tags 课评
// @Accept json
// @Produce json
// @Param status query string true "课评状态"
// @Success 200 {object} ginx.Result{data=int64} "Success"
// @Router /evaluations/count/mine [get]
func (h *EvaluationHandler) CountMine(ctx *gin.Context, req CountMineReq, uc ijwt.UserClaims) (ginx.Result, error) {
	status, ok := evaluationv1.EvaluationStatus_value[req.Status]
	if !ok {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "不合法的课评状态",
		}, errors.New("不合法的课评状态")
	}
	res, err := h.evaluationClient.CountMine(ctx, &evaluationv1.CountMineRequest{
		Uid:    uc.Uid,
		Status: evaluationv1.EvaluationStatus(status),
	})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg:  "Success",
		Data: res.GetCount(), // 标明是count
	}, nil
}
