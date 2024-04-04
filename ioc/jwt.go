package ioc

import (
	"github.com/MuxiKeStack/bff/web/ijwt"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitJwtHandler(cmd redis.Cmdable) ijwt.Handler {
	type Config struct {
		JwtKey     string `yaml:"jwtKey"`
		RefreshKey string `yaml:"refreshKey"`
	}
	var cfg Config
	err := viper.UnmarshalKey("jwt", &cfg)
	if err != nil {
		panic(err)
	}
	return ijwt.NewRedisJWTHandler(cmd, cfg.JwtKey, cfg.RefreshKey)
}
