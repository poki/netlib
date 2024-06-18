package util

import (
	"context"
	"time"
)

func NowUTC(ctx context.Context) time.Time {
	return time.Now().UTC()
}
