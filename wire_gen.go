// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/MuxiKeStack/bff/ioc"
	"github.com/MuxiKeStack/bff/pkg/ginx"
	"github.com/MuxiKeStack/bff/web"
	"github.com/MuxiKeStack/bff/web/evaluation"
	"github.com/MuxiKeStack/bff/web/search"
)

// Injectors from wire.go:

func InitWebServer() *ginx.Server {
	logger := ioc.InitLogger()
	cmdable := ioc.InitRedis()
	handler := ioc.InitJwtHandler(cmdable)
	client := ioc.InitEtcdClient()
	userServiceClient := ioc.InitUserClient(client)
	ccnuServiceClient := ioc.InitCCNUClient(client)
	userHandler := web.NewUserHandler(handler, userServiceClient, ccnuServiceClient)
	courseServiceClient := ioc.InitCourseClient(client)
	evaluationServiceClient := ioc.InitEvaluationClient(client)
	tagServiceClient := ioc.InitTagClient(client)
	collectServiceClient := ioc.InitCollectClient(client)
	courseHandler := web.NewCourseHandler(handler, courseServiceClient, evaluationServiceClient, userServiceClient, tagServiceClient, logger, collectServiceClient)
	questionServiceClient := ioc.InitQuestionClient(client)
	questionHandler := web.NewQuestionHandler(questionServiceClient, userServiceClient, logger)
	stanceServiceClient := ioc.InitStanceClient(client)
	commentServiceClient := ioc.InitCommentClient(client)
	evaluationHandler := evaluation.NewEvaluationHandler(evaluationServiceClient, tagServiceClient, stanceServiceClient, commentServiceClient)
	commentHandler := web.NewCommentHandler(commentServiceClient)
	searchServiceClient := ioc.InitSearchClient(client)
	searchHandler := search.NewSearchHandler(searchServiceClient, tagServiceClient, evaluationServiceClient)
	server := ioc.InitGinServer(logger, handler, userHandler, courseHandler, questionHandler, evaluationHandler, commentHandler, searchHandler)
	return server
}
