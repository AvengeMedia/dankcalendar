package ipc

import (
	dankipc "github.com/AvengeMedia/dankgo/ipc"
	"github.com/AvengeMedia/dankgo/ipc/params"
)

const APIVersion = 1

type (
	Request         = dankipc.Request
	Response[T any] = dankipc.Response[T]
	ConnWriter      = dankipc.ConnWriter
	Subscriber      = dankipc.Subscriber
	EventBus        = dankipc.EventBus
	Client          = dankipc.Client
)

var (
	NewEventBus   = dankipc.NewEventBus
	NewConnWriter = dankipc.NewConnWriter
	Dial          = dankipc.Dial
)

func Respond[T any](w *ConnWriter, id int, result T) { dankipc.Respond(w, id, result) }

func RespondError(w *ConnWriter, id int, msg string) { dankipc.RespondError(w, id, msg) }

func FindRunningSocket() (string, error) { return dankipc.FindRunningSocket("dankcal") }

func ParamString(p map[string]any, key string) string { return params.StringOpt(p, key, "") }

func ParamInt(p map[string]any, key string) int { return params.IntOpt(p, key, 0) }

func ParamBool(p map[string]any, key string) bool { return params.BoolLoose(p, key) }

func ParamStringSlice(p map[string]any, key string) []string { return params.StringSlice(p, key) }
