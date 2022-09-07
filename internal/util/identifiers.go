package util

import (
	"context"
	"encoding/base32"
	"math/rand"
	"os"
	"strconv"
	"strings"

	crand "crypto/rand"

	"github.com/koenbollen/logging"
	"github.com/rs/xid"
	"go.uber.org/zap"
)

func GeneratePeerID(ctx context.Context) string {
	if os.Getenv("ENV") == "test" {
		return strconv.FormatInt(rand.Int63(), 36) // deterministic for testing
	}
	return xid.New().String()
}

func GenerateSecret(ctx context.Context) string {
	if os.Getenv("ENV") == "test" {
		return "secret" // deterministic for testing
	}

	buf := make([]byte, 15)
	if _, err := crand.Read(buf[:]); err != nil {
		logger := logging.GetLogger(ctx)
		logger.Error("error generating secret", zap.Error(err))
		panic(err)
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(buf))
}

func GenerateLobbyCode(ctx context.Context) string {
	return strconv.FormatInt(rand.Int63(), 36)
}
