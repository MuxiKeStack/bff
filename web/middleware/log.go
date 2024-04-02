package middleware

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"io"
	"time"
)

type LogMiddlewareBuilder struct {
	logFn     func(ctx context.Context, l AccessLog)
	allowReq  bool
	allowResp bool
}

func NewLogMiddlewareBuilder(logFn func(ctx context.Context, l AccessLog)) *LogMiddlewareBuilder {
	return &LogMiddlewareBuilder{logFn: logFn}
}

type AccessLog struct {
	Path     string        `json:"path"`
	Method   string        `json:"method"`
	ReqBody  string        `json:"req_body"`
	Status   int           `json:"status"`
	RespBody string        `json:"resp_body"`
	Duration time.Duration `json:"duration"`
}

func (m *LogMiddlewareBuilder) AllowReq() *LogMiddlewareBuilder {
	m.allowReq = true
	return m
}

func (m *LogMiddlewareBuilder) AllowResp() *LogMiddlewareBuilder {
	m.allowResp = true
	return m
}

func (m *LogMiddlewareBuilder) Build() func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		if len(path) > 1024 {
			path = path[:1024]
		}
		method := ctx.Request.Method
		al := AccessLog{
			Path:   path,
			Method: method,
		}
		if m.allowReq {
			// request 是一个stream对象，只能读一次
			body, _ := ctx.GetRawData()
			if len(body) > 2048 {
				al.ReqBody = string(body[:2048])
			} else {
				al.RespBody = string(body)
			}
			// 放回去
			ctx.Request.Body = io.NopCloser(bytes.NewReader([]byte(body)))
		}
		if m.allowResp {
			ctx.Writer = &responseWriter{
				ResponseWriter: ctx.Writer,
				al:             &al,
			}
		}

		start := time.Now()
		defer func() {
			al.Duration = time.Since(start)
			m.logFn(ctx, al)
		}()
		ctx.Next()
	}
}

type responseWriter struct {
	gin.ResponseWriter
	al *AccessLog
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.al.RespBody += string(data)
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.al.Status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
