package ioc

import (
	"context"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitCommentClient(ecli *clientv3.Client) commentv1.CommentServiceClient {
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc.client.comment", &cfg)
	if err != nil {
		panic(err)
	}
	r := etcd.New(ecli)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(100*time.Second), // TODO
	)
	if err != nil {
		panic(err)
	}
	client := commentv1.NewCommentServiceClient(cc)
	return client
}
