//go:build wireinject

package main

import (
	"github.com/MuxiKeStack/bff/ioc"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web"
	"github.com/google/wire"
)

func InitWebServer() *ginx.Server {
	wire.Build(
		ioc.InitGinServer,
		web.NewUserHandler, web.NewCourseHandler, ioc.InitJwtHandler,
		ioc.InitCCNUClient,
		ioc.InitCourseClient,
		ioc.InitEvaluationClient,
		ioc.InitUserClient,
		// 组件
		ioc.InitEtcdClient,
		ioc.InitLogger,
		ioc.InitRedis,
	)
	return &ginx.Server{}
}
