package signaling

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/cloudflare"
	"github.com/poki/netlib/internal/metrics"
	"github.com/poki/netlib/internal/signaling/stores"
	"github.com/poki/netlib/internal/util"
	"go.uber.org/zap"
)

const LobbyCleanInterval = 30 * time.Minute
const LobbyCleanThreshold = 24 * time.Hour
const peerPingDuration = 2 * time.Second
const peerActiveUpdateInterval = 30 * time.Second

// Countries to track states/regions for the avg-latency-at-10s event.
// United States
// Canada
// Australia
// Brazil
// India
// Mexico
// Argentina
// Chile
// China
// Russia
// Indonesia
// Japan
var countriesToTrackStates = []string{"US", "CA", "AU", "BR", "IN", "MX", "AR", "CL", "CN", "RU", "ID", "JP"}

func Handler(ctx context.Context, store stores.Store, cloudflare *cloudflare.CredentialsClient) (*sync.WaitGroup, http.HandlerFunc) {
	manager := &TimeoutManager{
		Store: store,
	}
	go manager.Run(ctx)

	go func() {
		logger := logging.GetLogger(ctx)
		ticker := time.NewTicker(LobbyCleanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				logger.Debug("cleaning empty lobbies")
				if err := store.CleanEmptyLobbies(ctx, util.NowUTC(ctx).Add(-LobbyCleanThreshold)); err != nil {
					logger.Error("failed to clean empty lobbies", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	wg := &sync.WaitGroup{}
	return wg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.GetLogger(ctx)
		logger.Debug("upgrading connection")

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		acceptOptions := &websocket.AcceptOptions{
			InsecureSkipVerify: true, // Allow any origin/game to connect.
			CompressionMode:    websocket.CompressionDisabled,
		}
		conn, err := websocket.Accept(w, r, acceptOptions)
		if err != nil {
			util.ErrorAndAbort(w, r, http.StatusBadRequest, "", err)
		}

		wg.Add(1)
		defer wg.Done()

		country := r.Header.Get("CF-IPCountry")
		region := r.Header.Get("X-Geo-Region")
		if country == "" || region == "" {
			q := r.URL.Query()
			if country == "" {
				country = q.Get("country")
			}
			if region == "" {
				region = q.Get("region")
			}
		}

		peer := &Peer{
			store: store,
			conn:  conn,

			retrievedIDCallback: manager.Reconnected,

			Country: country,
			Region:  region,
		}
		defer func() {
			logger.Debug("peer websocket closed", zap.String("peer", peer.ID), zap.String("game", peer.Game), zap.String("origin", r.Header.Get("Origin")))
			conn.Close(websocket.StatusInternalError, "unexpected closure") // nolint:errcheck

			if !peer.closedPacketReceived {
				// At this point ctx has already been cancelled, so we create a new one to use for the disconnect.
				nctx, cancel := context.WithTimeout(logging.WithLogger(context.Background(), logger), time.Second*10)
				defer cancel()
				manager.Disconnected(nctx, peer)
			}
		}()

		go func() { // Sending ping packet every X to check if the tcp connection is still alive.
			ticker := time.NewTicker(peerPingDuration)
			defer ticker.Stop()
			var lastActiveUpdate time.Time
			for {
				select {
				case <-ticker.C:
					if err := peer.Send(ctx, PingPacket{Type: "ping"}); err != nil {
						if !util.ShouldIgnoreNetworkError(err) {
							if strings.Contains(err.Error(), "write: broken pipe") || strings.Contains(err.Error(), "connection reset by peer") {
								logger.Warn("failed to send ping packet", zap.String("peer", peer.ID), zap.Error(err))
							} else {
								logger.Error("failed to send ping packet", zap.String("peer", peer.ID), zap.Error(err))
							}
						}
					} else {
						// If we can send a ping packet, and the peer has an ID, we update the peer as being active.
						// If the peer doesn't have an ID yet, it's still in the process of connecting, so we don't update it.
						if peer.ID != "" {
							now := util.NowUTC(ctx)
							if lastActiveUpdate.IsZero() || now.Sub(lastActiveUpdate) >= peerActiveUpdateInterval {
								manager.MarkPeerAsActive(ctx, peer.ID)
								lastActiveUpdate = now
							}
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		for ctx.Err() == nil {
			var raw []byte
			if _, raw, err = conn.Read(ctx); err != nil {
				util.ErrorAndDisconnect(ctx, conn, err)
			}

			base := struct {
				Type      string `json:"type"`
				RequestID string `json:"rid"`
			}{}
			if err := json.Unmarshal(raw, &base); err != nil {
				util.ErrorAndDisconnect(ctx, conn, err)
			}

			if base.RequestID != "" {
				ctx = util.WithRequestID(ctx, base.RequestID)
			}

			if peer.closedPacketReceived {
				if base.Type != "disconnect" && base.Type != "disconnected" { // expected lingering packets after closure.
					logger.Warn("received packet after close", zap.String("peer", peer.ID), zap.String("type", base.Type))
				}
				continue
			}

			switch base.Type {
			case "credentials":
				credentials, err := cloudflare.GetCredentials(ctx)
				if err != nil {
					util.ReplyError(ctx, conn, err)
				} else {
					packet := CredentialsPacket{
						Type:        "credentials",
						Credentials: *credentials,
						RequestID:   base.RequestID,
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

				// Add country and region to event data of the avg-latency-at-10s event.
				// We want to use this data to build a latency world map.
				if params.Action == "avg-latency-at-10s" && params.Data != nil && peer != nil {
					params.Data["country"] = peer.Country

					// For big countries, also track the region/state so we can try and use this to
					// estimate latencies more accurately. We only do this for a select set of countries
					// to limit the amount of data we collect.
					if slices.Contains(countriesToTrackStates, peer.Country) {
						params.Data["region"] = peer.Region
					}
				}

				go metrics.RecordEvent(ctx, params)

			case "ping", "pong":
				// ignore, ping/pong is just for the tcp keepalive.

			default:
				if err := peer.HandlePacket(ctx, base.Type, raw); err != nil {
					if err == ErrUnknownPacketType {
						logger.Warn("unknown packet type received", zap.String("type", base.Type), zap.String("peer", peer.ID), zap.String("game", peer.Game), zap.String("origin", r.Header.Get("Origin")))
					} else {
						util.ErrorAndDisconnect(ctx, conn, err)
					}
				}
			}
		}
	})
}
