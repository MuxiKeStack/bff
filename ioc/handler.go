package ioc

import (
	staticv1 "github.com/MuxiKeStack/be-api/gen/proto/static/v1"
	"github.com/MuxiKeStack/bff/pkg/htmlx"
	"github.com/MuxiKeStack/bff/web"
	"github.com/ecodeclub/ekit/slice"
	"github.com/spf13/viper"
)

func InitStaticHandler(staticClient staticv1.StaticServiceClient) *web.StaticHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return web.NewStaticHandler(staticClient,
		map[string]htmlx.FileToHTMLConverter{
			//"docx": &htmlx.DocxToHTMLConverter{},
		},
		slice.ToMapV(administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}
