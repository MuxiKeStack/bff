package web

import (
	"errors"
	"fmt"
	staticv1 "github.com/MuxiKeStack/be-api/gen/proto/static/v1"
	"github.com/MuxiKeStack/bff/errs"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/pkg/htmlx"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"io"
	"path"
)

type StaticHandler struct {
	staticClient           staticv1.StaticServiceClient
	fileToHTMLConverterMap map[string]htmlx.FileToHTMLConverter
	Administrators         map[string]struct{}
}

func NewStaticHandler(staticClient staticv1.StaticServiceClient, fileToHTMLConverterMap map[string]htmlx.FileToHTMLConverter,
	administrators map[string]struct{}) *StaticHandler {
	return &StaticHandler{staticClient: staticClient, fileToHTMLConverterMap: fileToHTMLConverterMap, Administrators: administrators}
}

func (h *StaticHandler) RegisterRoutes(s *gin.Engine, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/statics")
	sg.GET("/:name", ginx.Wrap(h.GetStatic))
	// 因为没有管理员系统，所以直接将管理员写入配置文件
	sg.POST("/save", authMiddleware, ginx.WrapClaimsAndReq(h.SaveStatic))
	sg.POST("/save_file", authMiddleware, ginx.WrapClaimsAndReq(h.SaveStaticByFile))
}

// @Summary 获取静态资源
// @Description 根据静态资源名称获取静态资源的内容。
// @Tags 静态
// @Accept json
// @Produce json
// @Param name path string true "静态资源名称"
// @Success 200 {object} ginx.Result{data=StaticVo} "成功"
// @Router /statics/{name} [get]
func (h *StaticHandler) GetStatic(ctx *gin.Context) (ginx.Result, error) {
	name := ctx.Param("name")
	if name == "" {
		return ginx.Result{
			Code: errs.StaticInvalidInput,
			Msg:  "静态名称不合法",
		}, errors.New("静态名称不合法")
	}
	res, err := h.staticClient.GetStaticByName(ctx, &staticv1.GetStaticByNameRequest{Name: name})
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	return ginx.Result{
		Msg: "Success",
		Data: StaticVo{
			Content: res.GetStatic().GetContent(),
		},
	}, nil
}

// @Summary 保存静态内容
// @Description 保存静态内容
// @Tags 静态
// @Accept json
// @Produce json
// @Param request body SaveStaticReq true "保存静态内容请求参数"
// @Success 200 {object} ginx.Result "成功"
// @Router /statics/save [post]
func (h *StaticHandler) SaveStatic(ctx *gin.Context, req SaveStaticReq, uc ijwt.UserClaims) (ginx.Result, error) {
	// 管理员身份验证
	if !h.isAdmin(uc.StudentId) {
		return ginx.Result{
			Code: errs.StaticPermissionDenied,
			Msg:  "没有访问权限",
		}, fmt.Errorf("没有访问权限: %s", uc.StudentId)
	}
	if req.Name == "" {
		return ginx.Result{
			Code: errs.StaticInvalidInput,
			Msg:  "静态名称不合法",
		}, errors.New("静态名称不合法")
	}
	_, err := h.staticClient.SaveStatic(ctx, &staticv1.SaveStaticRequest{
		Static: &staticv1.Static{
			Name:    req.Name,
			Content: req.Content,
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

// @Summary 保存静态内容[文件]
// @Description 通过上传文件保存静态内容，目前仅支持.html文件
// @Tags 静态
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "静态名称"
// @Param content formData file true "静态内容文件"
// @Success 200 {object} ginx.Result "成功"
// @Router /statics/save_file [post]
func (h *StaticHandler) SaveStaticByFile(ctx *gin.Context, req SaveStaticByFileReq, uc ijwt.UserClaims) (ginx.Result, error) {
	// 管理员身份验证
	if !h.isAdmin(uc.StudentId) {
		return ginx.Result{
			Code: errs.StaticPermissionDenied,
			Msg:  "没有访问权限",
		}, fmt.Errorf("没有访问权限: %s", uc.StudentId)
	}
	formFile, err := ctx.FormFile("content")
	if err != nil {
		return ginx.Result{
			Code: errs.InternalServerError,
			Msg:  "系统异常",
		}, err
	}
	if req.Name == "" {
		return ginx.Result{
			Code: errs.StaticInvalidInput,
			Msg:  "静态名称不合法",
		}, errors.New("静态名称不合法")
	}
	var htmlContent string
	if ext := path.Ext(formFile.Filename); ext == ".html" {
		file, er := formFile.Open()
		if er != nil {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, er
		}
		data, er := io.ReadAll(file)
		if er != nil {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, er
		}
		htmlContent = string(data)
	} else {
		// 非.html得其他文件要进行转换，如docx
		converter, exists := h.fileToHTMLConverterMap[ext]
		if !exists {
			return ginx.Result{
				Code: errs.StaticInvalidInput,
				Msg:  "不支持的文件类型",
			}, fmt.Errorf("不支持文件类型: %s", ext)
		}
		file, er := formFile.Open()
		if er != nil {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, er
		}
		htmlContent, er = converter.ConvertToHTML(file)
		if er != nil {
			return ginx.Result{
				Code: errs.InternalServerError,
				Msg:  "系统异常",
			}, er
		}
	}
	_, err = h.staticClient.SaveStatic(ctx, &staticv1.SaveStaticRequest{
		Static: &staticv1.Static{
			Name:    req.Name,
			Content: htmlContent,
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

func (h *StaticHandler) isAdmin(studentId string) bool {
	_, exists := h.Administrators[studentId]
	return exists
}
