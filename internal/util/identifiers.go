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

func GenerateShortLobbyCode(ctx context.Context) string {
	numbers := []string{"2", "3", "4", "5", "6", "7", "8", "9"}
	alphabet := []string{"A", "B", "C", "D", "E", "F", "G", "H", "J", "K", "M", "N", "P", "R", "S", "T", "V", "W", "X", "Y", "Z"}
	return numbers[rand.Intn(len(numbers))] + numbers[rand.Intn(len(numbers))] + alphabet[rand.Intn(len(alphabet))] + alphabet[rand.Intn(len(alphabet))]
}
