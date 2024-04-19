package evaluation

import (
	interactv1 "github.com/MuxiKeStack/be-api/gen/proto/interact/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"strconv"
)

func (h *EvaluationHandler) Endorse(ctx *gin.Context, req EndorseReq, uc ijwt.UserClaims) (ginx.Result, error) {
	eidStr := ctx.Param("evaluationId")
	eid, err := strconv.ParseInt(eidStr, 10, 64)
	if err != nil {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "输入参数有误",
		}, err
	}
	_, ok := interactv1.Stance_name[req.Stance]
	if !ok {
		return ginx.Result{
			Code: errs.EvaluationInvalidInput,
			Msg:  "不合法的立场",
		}, err
	}
	_, err = h.interactClient.Endorse(ctx, &interactv1.EndorseRequest{
		Uid:    uc.Uid,
		Biz:    interactv1.Biz_Evaluation,
		BizId:  eid,
		Stance: interactv1.Stance(req.Stance),
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
