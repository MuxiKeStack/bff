package evaluation

import (
	stancev1 "github.com/MuxiKeStack/be-api/gen/proto/stance/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"strconv"
)

// @Summary 支持或反对课评
// @Description 根据评价ID支持或反对指定的评价
// @Tags 课评
// @Accept json
// @Produce json
// @Param evaluationId path integer true "评价ID"
// @Param body body EndorseReq true "支持或反对的请求体，stance为态度标识，-1反对，1支持，0表示无可用于取消表态"
// @Success 200 {object} ginx.Result{data=nil} "操作成功"
// @Router /evaluations/{evaluationId}/endorse [post]
func (h *EvaluationHandler) Endorse(ctx *gin.Context, req EndorseReq, uc ijwt.UserClaims) (ginx.Result, error) {
	eidStr := ctx.Param("evaluationId")
	eid, err := strconv.ParseInt(eidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	_, ok := stancev1.Stance_name[req.Stance]
	if !ok {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "不合法的立场",
		}, err
	}
	_, err = h.stanceClient.Endorse(ctx, &stancev1.EndorseRequest{
		Uid:    uc.Uid,
		Biz:    stancev1.Biz_Evaluation,
		BizId:  eid,
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
