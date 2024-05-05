package ioc

import (
	"context"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/pkg/logger"
	"github.com/MuxiKeStack/bff/web"
	"github.com/MuxiKeStack/bff/web/evaluation"
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/MuxiKeStack/bff/web/middleware"
	"github.com/MuxiKeStack/bff/web/search"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"strings"
	"time"
)

func InitGinServer(l logger.Logger, jwtHdl ijwt.Handler, user *web.UserHandler,
	course *web.CourseHandler, question *web.QuestionHandler, evaluation *evaluation.EvaluationHandler,
	comment *web.CommentHandler, search *search.SearchHandler, grade *web.GradeHandler, static *web.StaticHandler,
	answer *web.AnswerHandler, point *web.PointHandler, feed *web.FeedHandler, tube *web.TubeHandler) *ginx.Server {
	engine := gin.Default()
	engine.Use(
		corsHdl(),
		//middleware.NewLoginMiddleWareBuilder(jwtHdl).Build(),
	)
	authMiddleware := middleware.NewLoginMiddleWareBuilder(jwtHdl).Build()
	user.RegisterRoutes(engine, authMiddleware)
	course.RegisterRoutes(engine, authMiddleware)
	question.RegisterRoutes(engine, authMiddleware)
	evaluation.RegisterRoutes(engine, authMiddleware)
	comment.RegisterRoutes(engine, authMiddleware)
	search.RegisterRoutes(engine, authMiddleware)
	grade.RegisterRoutes(engine, authMiddleware)
	static.RegisterRoutes(engine, authMiddleware)
	answer.RegisterRoutes(engine, authMiddleware)
	point.RegisterRoutes(engine, authMiddleware)
	feed.RegisterRoutes(engine, authMiddleware)
	tube.RegisterRoutes(engine, authMiddleware)
	addr := viper.GetString("http.addr")
	ginx.InitCounter(prometheus.CounterOpts{
		Namespace: "muxi",
		Subsystem: "kstack_bff",
		Name:      "http",
	})
	ginx.SetLogger(l)
	return &ginx.Server{
		Engine: engine,
		Addr:   addr,
	}
}

func timeout() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		_, ok := ctx.Request.Context().Deadline()
		if !ok {
			// 强制给一个超时，省得我前端调试等得不耐烦
			newCtx, cancel := context.WithTimeout(ctx.Request.Context(), time.Second*10)
			defer cancel()
			ctx.Request = ctx.Request.Clone(newCtx)
		}
		ctx.Next()
	}
}

func corsHdl() gin.HandlerFunc {
	return cors.New(cors.Config{
		//AllowOrigins: []string{"*"},
		//AllowMethods: []string{"POST", "GET"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"x-jwt-token", "x-refresh-token"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "localhost") {
				// 你的开发环境
				return true
			}
			return strings.Contains(origin, "bigdust.space")
		},
		MaxAge: 12 * time.Hour,
	})
}
