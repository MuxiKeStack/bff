package ioc

import (
	"context"
	ccnuv1 "github.com/MuxiKeStack/be-api/gen/proto/ccnu/v1"
	webclient "github.com/MuxiKeStack/bff/web/client"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/spf13/viper"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"time"
)

func InitCCNUClient(etcdClient *etcdv3.Client) ccnuv1.CCNUServiceClient {
	type Config struct {
		Endpoint string `yaml:"endpoint"`
		RetryCnt int    `yaml:"retryCnt"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc.client.ccnu", &cfg)
	if err != nil {
		panic(err)
	}
	r := etcd.New(etcdClient)
	cc, err := grpc.DialInsecure(context.Background(),
		grpc.WithEndpoint(cfg.Endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(10*time.Second), // TODO
	)
	if err != nil {
		panic(err)
	}
	ccnuClient := ccnuv1.NewCCNUServiceClient(cc)
	retryCCNUClient := webclient.NewRetryCCNUClient(ccnuClient, cfg.RetryCnt)
	return retryCCNUClient
}
