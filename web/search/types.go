package search

import (
	"context"
	"github.com/MuxiKeStack/bff/pkg/ginx"
)

type SearchStrategy interface {
	Search(ctx context.Context, keyword string, uid int64, searchLocation string) (ginx.Result, error)
}

type SearchStrategyMap map[string]SearchStrategy
