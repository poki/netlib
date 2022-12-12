package signaling

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/cloudflare"
	"github.com/poki/netlib/internal/metrics"
	"github.com/poki/netlib/internal/signaling/stores"
	"github.com/poki/netlib/internal/util"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

const MaxConnectionTime = 1 * time.Hour

type Store interface {
	CreateLobby(ctx context.Context, game, lobby, id string) error
	JoinLobby(ctx context.Context, game, lobby, id string) ([]string, error)
	IsPeerInLobby(ctx context.Context, game, lobby, id string) (bool, error)
	LeaveLobby(ctx context.Context, game, lobby, id string) ([]string, error)
	GetLobby(ctx context.Context, game, lobby string) ([]string, error)
	ListLobbies(ctx context.Context, game, filter string) ([]stores.Lobby, error)

	// Subscribe subscribes to topic. callback should never block!
	Subscribe(ctx context.Context, topic string, callback func(context.Context, []byte))
	Publish(ctx context.Context, topic string, data []byte) error
}

func Handler(ctx context.Context, store Store, cloudflare *cloudflare.CredentialsClient) http.HandlerFunc {
	acceptOptions := &websocket.AcceptOptions{
		// Allow any origin/game to connect.
		InsecureSkipVerify: true,
	}

	manager := &TimeoutManager{}
	go manager.Run(ctx)

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

			retrievedIDCallback: manager.Reconnected,
		}
		defer func() {
			logger.Debug("peer websocket closed", zap.String("id", peer.ID))
			manager.Disconnected(ctx, peer)
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

			switch typeOnly.Type {
			case "credentials":
				credentials, err := cloudflare.GetCredentials(ctx)
				if err != nil {
					util.ReplyError(ctx, conn, err)
				} else {
					packet := CredentialsPacket{
						Type:        "credentials",
						Credentials: *credentials,
					}
					if err := peer.Send(ctx, packet); err != nil {
						util.ErrorAndDisconnect(ctx, conn, err)
					}
				}

			case "event":
				params := metrics.EventParams{}
				if err := json.Unmarshal(raw, &params); err != nil {
					util.ErrorAndDisconnect(ctx, conn, err)
				}
				go metrics.RecordEvent(ctx, params)

			default:
				if err := peer.HandlePacket(ctx, typeOnly.Type, raw); err != nil {
					util.ErrorAndDisconnect(ctx, conn, err)
				}
			}
		}
	})
}
