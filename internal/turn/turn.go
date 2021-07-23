package turn

import (
	"context"
	"net"

	"github.com/koenbollen/logging"
	"github.com/pion/turn/v2"
)

func Run(ctx context.Context, addr string) error {
	logger := logging.GetLogger(ctx)
	logger.Info("starting turn server")

	listener, err := net.ListenPacket("udp4", addr)
	if err != nil {
		return err
	}

	server, err := turn.NewServer(turn.ServerConfig{
		LoggerFactory: nil,
		Realm:         "poki.com",
		AuthHandler: func(username, realm string, srcAddr net.Addr) (key []byte, ok bool) {
			return turn.GenerateAuthKey(username, realm, "secret"), true
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: listener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP("127.0.0.1"),
					Address:      "0.0.0.0",
				},
			},
		},
	})
	if err != nil {
		return err
	}

	<-ctx.Done()
	logger.Info("closing turn server")
	return server.Close()
}
