package ipc

import (
	"context"

	dankipc "github.com/AvengeMedia/dankgo/ipc"
)

type Server = dankipc.Server

func NewServer(deps Deps) *Server {
	if deps.Bus == nil {
		deps.Bus = NewEventBus()
	}

	cfg := dankipc.Config{
		AppName:                "dankcal",
		APIVersion:             APIVersion,
		Capabilities:           []string{"accounts", "calendars", "events", "reminders", "subscribe", "ui", "system"},
		DefaultSubscribeTopics: []string{"accounts", "calendars", "events", "tasks", "sync"},
		Bus:                    deps.Bus,
		OnSubscribe: func(topics []string, _ *Subscriber) {
			publishPending(deps, topics)
		},
	}

	return dankipc.NewServer(cfg, func(ctx context.Context, w *ConnWriter, req Request, _ *Subscriber) {
		Route(ctx, w, req, deps)
	})
}

func publishPending(deps Deps, topics []string) {
	if deps.Pending == nil {
		return
	}
	for _, t := range topics {
		if t != "ui" {
			continue
		}
		if payload := deps.Pending.Take(); payload != nil {
			deps.Bus.Publish("ui", payload)
		}
		return
	}
}
