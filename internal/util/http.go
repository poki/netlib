package util

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

type requestIDContextKeyType int

const requestIDContextKey requestIDContextKeyType = 0

type errorResponse struct {
	Status int      `json:"status"`
	Key    string   `json:"key"`
	Errors []string `json:"errors,omitempty"`
}

type errorCodeError struct {
	err  error
	code string
}

func (e *errorCodeError) Error() string {
	return e.err.Error()
}

func (e *errorCodeError) ErrorCode() string {
	return e.code
}

func ErrorWithCode(err error, code string) error {
	return &errorCodeError{err: err, code: code}
}

func ErrorAndAbort(w http.ResponseWriter, r *http.Request, status int, key string, errs ...error) {
	if status/100 == 5 && len(errs) != 0 {
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
	ReplyError(ctx, conn, err)
	panic(http.ErrAbortHandler)
}

func ReplyError(ctx context.Context, conn *websocket.Conn, err error) {
	payload := struct {
		Type      string `json:"type"`
		RequestID string `json:"rid,omitempty"`
		Message   string `json:"message"`
		Error     any    `json:"error,omitempty"`
		Code      string `json:"code,omitempty"`
	}{
		Type:    "error",
		Message: err.Error(),
		Error:   err,
	}
	if rid, ok := ctx.Value(requestIDContextKey).(string); ok {
		payload.RequestID = rid
	}
	if cerr, ok := err.(interface{ ErrorCode() string }); ok {
		payload.Code = cerr.ErrorCode()
	}
	err = wsjson.Write(ctx, conn, &payload)
	if err != nil && !IsPipeError(err) {
		logger := logging.GetLogger(ctx)
		logger.Warn("uncaught server error", zap.Error(err), zap.Stack("stack"))
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

// WithRequestID returns a new context with the given request ID attached, this ID
// can be used when replying errors etc.
func WithRequestID(ctx context.Context, id string) context.Context {
	if id == "" || len(id) > 64 {
		return ctx
	}
	return context.WithValue(ctx, requestIDContextKey, id)
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
		if errors.Is(err, context.DeadlineExceeded) {
			return true
		}
		if errors.Is(err, io.EOF) {
			return true
		}
		closeErr := websocket.CloseError{}
		if errors.As(err, &closeErr) {
			return true
		}
	}
	return false
}
