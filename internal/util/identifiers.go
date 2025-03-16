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

// deterministicRand is used while running tests to ensure that the generated
// identifiers are deterministic.
// Each feature test will restart the signaling server, so running tests in a random
// order will work as expected.
var deterministicRand = rand.New(rand.NewSource(0))

func GeneratePeerID(ctx context.Context) string {
	if os.Getenv("ENV") == "test" {
		return strconv.FormatInt(deterministicRand.Int63(), 36) // deterministic for testing
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
	randInt63 := rand.Int63
	if os.Getenv("ENV") == "test" {
		randInt63 = deterministicRand.Int63
	}

	return strconv.FormatInt(randInt63(), 36)
}

func GenerateShortLobbyCode(ctx context.Context, chars int) string {
	randIntn := rand.Intn
	if os.Getenv("ENV") == "test" {
		randIntn = deterministicRand.Intn
	}

	numbers := []string{"2", "3", "4", "5", "6", "7", "8", "9"}
	alphabet := []string{"A", "B", "C", "D", "E", "F", "G", "H", "J", "K", "M", "N", "P", "R", "S", "T", "V", "W", "X", "Y", "Z"}

	ss := make([]string, chars)
	for i := range chars {
		if i/2%2 == 0 {
			ss[i] = numbers[randIntn(len(numbers))]
		} else {
			ss[i] = alphabet[randIntn(len(alphabet))]
		}
	}
	return strings.Join(ss, "")
}
