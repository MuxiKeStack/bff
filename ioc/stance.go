package ioc

import (
	"context"
	stancev1 "github.com/MuxiKeStack/be-api/gen/proto/stance/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitStanceClient(ecli *clientv3.Client) stancev1.StanceServiceClient {
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc.client.stance", &cfg)
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
	client := stancev1.NewStanceServiceClient(cc)
	return client
}
