package util

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/koenbollen/logging"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type errorResponse struct {
	Status int      `json:"status"`
	Key    string   `json:"key"`
	Errors []string `json:"errors,omitempty"`
}

func ErrorAndAbort(w http.ResponseWriter, r *http.Request, status int, key string, errs ...error) {
	if status/100 == 500 && len(errs) != 0 {
		logger := logging.GetLogger(r.Context())
		logger.Error("uncaught server error", zap.Errors("errors", errs))
	}
	if key == "" {
		key = strings.ToLower(strings.Join(strings.Fields(http.StatusText(status)), "-"))
	}
	resp := errorResponse{
		Status: status,
		Key:    key,
	}
	for _, e := range errs {
		if e != nil {
			resp.Errors = append(resp.Errors, e.Error())
		}
	}
	RenderJSON(w, r, status, resp)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	panic(http.ErrAbortHandler)
}

func ErrorAndDisconnect(ctx context.Context, conn *websocket.Conn, err error) {
	logger := logging.GetLogger(ctx)
	if !IsPipeError(err) {
		logger.Warn("error during connection", zap.Error(err))
	}
	payload := struct {
		Type  string `json:"type"`
		Error string `json:"error"`
	}{
		Type:  "error",
		Error: err.Error(),
	}
	err = wsjson.Write(ctx, conn, &payload)
	if err != nil && !IsPipeError(err) {
		logger.Warn("uncaught server error", zap.Error(err))
	}
	panic(http.ErrAbortHandler)
}

func ReplyError(ctx context.Context, conn *websocket.Conn, err error) {
	payload := struct {
		Type  string `json:"type"`
		Error string `json:"error"`
	}{
		Type:  "error",
		Error: err.Error(),
	}
	err = wsjson.Write(ctx, conn, &payload)
	if err != nil && !IsPipeError(err) {
		logger := logging.GetLogger(ctx)
		logger.Warn("uncaught server error", zap.Error(err))
	}
}

// RenderJSON will write a json response to the given ResponseWriter.
func RenderJSON(w http.ResponseWriter, r *http.Request, status int, val interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(val)
	if err != nil && !IsPipeError(err) {
		logger := logging.GetLogger(r.Context())
		logger.Warn("uncaught server error", zap.Error(err))
	}
}

func IsPipeError(err error) bool {
	switch v := err.(type) {
	case syscall.Errno:
		return v == syscall.EPIPE
	case *net.OpError:
		return IsPipeError(v.Err)
	case *os.SyscallError:
		return IsPipeError(v.Err)
	default:
		if errors.Is(err, context.Canceled) {
			return true
		}
		closeErr := websocket.CloseError{}
		if errors.As(err, &closeErr) {
			return true
		}
	}
	return false
}
