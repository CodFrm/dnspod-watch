package main

import (
	"context"
	"log"

	"github.com/codfrm/cago"
	"github.com/codfrm/cago/configs"
	"github.com/codfrm/cago/pkg/component"
	"github.com/codfrm/cago/pkg/logger"
	"github.com/codfrm/dnspod-watch/internal/watch"
	"github.com/codfrm/dnspod-watch/pkg/pushcat"
)

func main() {
	ctx := context.Background()
	cfg, err := configs.NewConfig("dnspod-watch")
	if err != nil {
		log.Fatalf("load config err: %v", err)
	}

	err = cago.New(ctx, cfg).
		Registry(component.Logger()).
		Registry(cago.FuncComponent(logger.Logger)).
		Registry(pushcat.Pushcat()).
		Registry(watch.Watch()).
		Start()
	if err != nil {
		log.Fatalf("start err: %v", err)
		return
	}
}
