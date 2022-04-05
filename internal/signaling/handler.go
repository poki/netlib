package signaling

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/util"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

const MaxConnectionTime = 1 * time.Hour

type Store interface {
	CreateLobby(ctx context.Context, game, lobby, id string) error
	JoinLobby(ctx context.Context, game, lobby, id string) ([]string, error)
	GetLobby(ctx context.Context, game, lobby string) ([]string, error)

	Subscribe(ctx context.Context, topic string, callback func(context.Context, []byte))
	Publish(ctx context.Context, topic string, data []byte) error
}

func Handler(store Store) http.HandlerFunc {
	acceptOptions := &websocket.AcceptOptions{
		// Allow any origin/game to connect.
		InsecureSkipVerify: true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.GetLogger(ctx)
		logger.Debug("upgrading connection")

		ctx, cancel := context.WithTimeout(ctx, MaxConnectionTime)
		defer cancel()

		conn, err := websocket.Accept(w, r, acceptOptions)
		if err != nil {
			util.ErrorAndAbort(w, r, http.StatusBadRequest, "", err)
		}

		peer := &Peer{
			store: store,
			conn:  conn,
		}
		defer peer.Close()
		defer func() {
			logger.Debug("peer closed", zap.String("id", peer.ID))
		}()

		for ctx.Err() == nil {
			var raw []byte
			if _, raw, err = conn.Read(ctx); err != nil {
				util.ErrorAndDisconnect(ctx, conn, err)
			}

			typeOnly := struct{ Type string }{}
			if err := json.Unmarshal(raw, &typeOnly); err != nil {
				util.ErrorAndDisconnect(ctx, conn, err)
			}

			if typeOnly.Type == "credentials" {
				credentials, err := GetCredentials(ctx)
				if err != nil {
					util.ReplyError(ctx, conn, err)
					continue
				}
				if err := peer.Send(ctx, credentials); err != nil {
					util.ErrorAndDisconnect(ctx, conn, err)
				}
			}

			if err := peer.HandlePacket(ctx, typeOnly.Type, raw); err != nil {
				util.ErrorAndDisconnect(ctx, conn, err)
			}
		}
	})
}
