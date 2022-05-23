package util

import (
	"context"
	"time"
)

func Now(ctx context.Context) time.Time {
	return time.Now()
}
