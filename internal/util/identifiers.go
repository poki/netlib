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

var isTestEnv = os.Getenv("ENV") == "test"

// deterministicRand is used while running tests to ensure that the generated
// identifiers are deterministic.
// Each feature test will restart the signaling server, so running tests in a random
// order will work as expected.
var deterministicRand = rand.New(rand.NewSource(0))

var alphabet = []string{"2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F", "G", "H", "J", "K", "M", "N", "P", "R", "S", "T", "V", "W", "X", "Y", "Z"}

func GeneratePeerID(ctx context.Context) string {
	if isTestEnv {
		return strconv.FormatInt(deterministicRand.Int63(), 36) // deterministic for testing
	}
	return xid.New().String()
}

func GenerateSecret(ctx context.Context) string {
	if isTestEnv {
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
	randInt63 := rand.Int63
	if isTestEnv {
		randInt63 = deterministicRand.Int63
	}

	return strconv.FormatInt(randInt63(), 36)
}

// GenerateShortLobbyCode generates a short lobby code with the given number of characters.
// The code is generated using al alpabet of characters that are easy to distinguish from each other (e.g. no 0 and O).
func GenerateShortLobbyCode(ctx context.Context, chars int) string {
	randIntn := rand.Intn
	if isTestEnv {
		randIntn = deterministicRand.Intn
	}

	ss := make([]string, chars)
	for i := range chars {
		ss[i] = alphabet[randIntn(len(alphabet))]
	}
	return strings.Join(ss, "")
}
