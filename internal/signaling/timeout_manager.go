package signaling

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/metrics"
	"github.com/poki/netlib/internal/signaling/stores"
	"go.uber.org/zap"
)

type TimeoutManager struct {
	DisconnectThreshold time.Duration

	Store stores.Store
}

func (manager *TimeoutManager) Run(ctx context.Context) {
	if manager.DisconnectThreshold == 0 {
		// We update peer activity every 30 seconds. Make it possible to somehow miss
		// two updates before we consider a peer timed out.
		manager.DisconnectThreshold = time.Second * 90
	}

	// Allow clients to reconnect after a restart without refreshing stale rows.
	grace := time.NewTimer(manager.DisconnectThreshold)
	defer grace.Stop()
	select {
	case <-ctx.Done():
		return
	case <-grace.C:
	}
	for ctx.Err() == nil {
		manager.RunOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (manager *TimeoutManager) RunOnce(ctx context.Context) {
	logger := logging.GetLogger(ctx)

	for ctx.Err() == nil {
		peers, err := manager.Store.ClaimTimedOutPeers(ctx, manager.DisconnectThreshold, 250)
		if err != nil {
			logger.Error("failed to claim timedout peers", zap.Error(err))
			return
		}
		if len(peers) == 0 {
			return
		}
		// Several expired peers can share a lobby. Elect once after all removals.
		type lobbyKey struct{ game, code string }
		lobbies := make(map[lobbyKey]struct{})
		for _, peer := range peers {
			for game, codes := range peer.GameLobbies {
				for _, code := range codes {
					logger.Info("peer timeout", zap.String("peer", peer.ID), zap.String("game", game), zap.String("lobby", code))
					if err := manager.disconnectPeerInLobby(ctx, peer.ID, game, code); err != nil {
						logger.Error("failed to disconnect peer", zap.Error(err), zap.String("peer", peer.ID))
					}
					lobbies[lobbyKey{game, code}] = struct{}{}
					go metrics.Record(ctx, "client", "timeout", game, peer.ID, code)
				}
			}
		}
		for lobby := range lobbies {
			if err := manager.doLeaderElectionAndPublish(ctx, lobby.game, lobby.code); err != nil {
				logger.Error("failed to do leader election", zap.Error(err), zap.String("game", lobby.game), zap.String("lobby", lobby.code))
			}
		}
	}
}

func (manager *TimeoutManager) disconnectPeerInLobby(ctx context.Context, peerID string, gameID string, lobby string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	packet := DisconnectPacket{
		Type: "disconnect",
		ID:   peerID,
	}
	data, err := json.Marshal(packet)
	if err != nil {
		return err
	}

	err = manager.Store.Publish(ctx, gameID+lobby, data)
	if err != nil {
		return err
	}
	return nil
}

func (manager *TimeoutManager) Disconnected(ctx context.Context, p *Peer) {
	logger := logging.GetLogger(ctx)

	if p.ID == "" {
		return
	}

	logger.Debug("peer marked as disconnected", zap.String("id", p.ID), zap.String("lobby", p.Lobby))
	err := manager.Store.MarkPeerAsDisconnected(ctx, p.ID)
	if err != nil {
		logger.Error("failed to record timeout peer", zap.Error(err))
		return
	}

	if p.Lobby == "" {
		return
	}

	err = manager.doLeaderElectionAndPublish(ctx, p.Game, p.Lobby)
	if err != nil {
		logger.Error("failed to do leader election", zap.Error(err), zap.String("game", p.Game), zap.String("lobby", p.Lobby))
	}
}

func (manager *TimeoutManager) Reconnected(ctx context.Context, peerID, secret, gameID string) (bool, []string, error) {
	logger := logging.GetLogger(ctx)

	logger.Debug("peer marked as reconnected", zap.String("peer", peerID))
	return manager.Store.MarkPeerAsReconnected(ctx, peerID, secret, gameID)
}

func (manager *TimeoutManager) MarkPeerAsActive(ctx context.Context, peerID string) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	logger := logging.GetLogger(ctx)

	err := manager.Store.MarkPeerAsActive(ctx, peerID)

	// context.Canceled is expected when the connection is closed right as this function
	// is called. We don't want to log this as an error.
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("failed to mark peer as active", zap.Error(err), zap.String("peer", peerID))
	}
}

func (manager *TimeoutManager) doLeaderElectionAndPublish(ctx context.Context, gameID, lobbyCode string) error {
	result, err := manager.Store.DoLeaderElection(ctx, gameID, lobbyCode)
	if err != nil {
		return err
	}

	if result == nil {
		return nil
	}

	packet := LeaderPacket{
		Type:   "leader",
		Leader: result.Leader,
		Term:   result.Term,
	}
	data, err := json.Marshal(packet)
	if err != nil {
		return err
	}

	err = manager.Store.Publish(ctx, gameID+lobbyCode, data)
	if err != nil {
		return err
	}

	return nil
}
