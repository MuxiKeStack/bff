package ioc

import (
	"context"
	searchv1 "github.com/MuxiKeStack/be-api/gen/proto/search/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitSearchClient(ecli *clientv3.Client) searchv1.SearchServiceClient {
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc.client.search", &cfg)
	if err != nil {
		panic(err)
	}
	r := etcd.New(ecli)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(10*time.Second), // TODO
	)
	if err != nil {
		panic(err)
	}
	client := searchv1.NewSearchServiceClient(cc)
	return client
}
