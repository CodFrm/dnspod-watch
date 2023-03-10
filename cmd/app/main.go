package main

import (
	"context"
	"log"

	"github.com/codfrm/cago"
	"github.com/codfrm/cago/configs"
	"github.com/codfrm/cago/pkg/logger"
	"github.com/codfrm/dnspod-watch/internal/watch"
)

func main() {
	ctx := context.Background()
	cfg, err := configs.NewConfig("dnspod-watch")
	if err != nil {
		log.Fatalf("load config err: %v", err)
	}

	err = cago.New(ctx, cfg).
		Registry(cago.FuncComponent(logger.Logger)).
		Registry(watch.Watch()).
		Start()
	if err != nil {
		log.Fatalf("start err: %v", err)
		return
	}
}
