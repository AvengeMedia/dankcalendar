package ipc

import (
	"strconv"

	"github.com/AvengeMedia/dankcalendar/core/internal/log"
)

const APIVersion = 1

type Capabilities struct {
	APIVersion   int      `json:"apiVersion"`
	Capabilities []string `json:"capabilities"`
}

type Request struct {
	ID     int            `json:"id,omitempty"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

type Response[T any] struct {
	ID     int    `json:"id,omitempty"`
	Result *T     `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func RespondError(w *ConnWriter, id int, msg string) {
	log.Errorf("ipc error: id=%d method-error=%s", id, msg)
	w.WriteResponse(Response[any]{ID: id, Error: msg})
}

func Respond[T any](w *ConnWriter, id int, result T) {
	w.WriteResponse(Response[T]{ID: id, Result: &result})
}

func ParamString(p map[string]any, key string) string {
	v, ok := p[key].(string)
	if !ok {
		return ""
	}
	return v
}

func ParamInt(p map[string]any, key string) int {
	switch v := p[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func ParamBool(p map[string]any, key string) bool {
	switch v := p[key].(type) {
	case bool:
		return v
	case string:
		b, _ := strconv.ParseBool(v)
		return b
	}
	return false
}

func ParamStringSlice(p map[string]any, key string) []string {
	raw, ok := p[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
