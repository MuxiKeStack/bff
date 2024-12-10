package ioc

import (
	"context"
	tagv1 "github.com/MuxiKeStack/be-api/gen/proto/tag/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitTagClient(ecli *clientv3.Client) tagv1.TagServiceClient {
	type Config struct {
		Endpoint string `yaml:"endpoint"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc.client.tag", &cfg)
	if err != nil {
		panic(err)
	}
	r := etcd.New(ecli)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(time.Second*100),
	)
	if err != nil {
		panic(err)
	}
	client := tagv1.NewTagServiceClient(cc)
	return client
}
