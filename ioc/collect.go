package ioc

import (
	"context"
	collectv1 "github.com/MuxiKeStack/be-api/gen/proto/collect/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitCollectClient(ecli *clientv3.Client) collectv1.CollectServiceClient {
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc.client.collect", &cfg)
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
	client := collectv1.NewCollectServiceClient(cc)
	return client
}
