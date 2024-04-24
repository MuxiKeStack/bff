//go:build wireinject

package main

import (
	"github.com/MuxiKeStack/bff/ioc"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web"
	"github.com/MuxiKeStack/bff/web/evaluation"
	"github.com/MuxiKeStack/bff/web/search"
	"github.com/google/wire"
)

func InitWebServer() *ginx.Server {
	wire.Build(
		ioc.InitGinServer,
		web.NewUserHandler, web.NewCourseHandler, ioc.InitJwtHandler, web.NewQuestionHandler,
		evaluation.NewEvaluationHandler, web.NewCommentHandler, search.NewSearchHandler,
		web.NewGradeHandler,
		ioc.InitGradeClient,
		ioc.InitSearchClient,
		ioc.InitCommentClient,
		ioc.InitStanceClient,
		ioc.InitCollectClient,
		ioc.InitTagClient,
		ioc.InitCCNUClient,
		ioc.InitCourseClient,
		ioc.InitEvaluationClient,
		ioc.InitUserClient,
		ioc.InitQuestionClient,
		// 组件
		ioc.InitEtcdClient,
		ioc.InitLogger,
		ioc.InitRedis,
	)
	return &ginx.Server{}
}
