package signaling

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/metrics"
	"github.com/poki/netlib/internal/signaling/stores"
	"go.uber.org/zap"
)

type peerConnection struct {
	ID           string
	ConnectionID string
}

type TimeoutManager struct {
	DisconnectThreshold time.Duration

	Store stores.Store

	activityMutex sync.Mutex
	activePeers   map[peerConnection]struct{}
}

const peerActivityFlushInterval = 2 * time.Second

func (manager *TimeoutManager) Run(ctx context.Context) {
	if manager.DisconnectThreshold == 0 {
		// Retain disconnected sessions for reconnects, independently of matchmaking.
		manager.DisconnectThreshold = stores.DefaultPeerTimeout
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
	disconnected, err := manager.Store.MarkPeerAsDisconnected(ctx, p.ID, p.connectionID)
	if err != nil {
		logger.Error("failed to record timeout peer", zap.Error(err))
		return
	}

	if !disconnected || p.Lobby == "" {
		return
	}

	err = manager.doLeaderElectionAndPublish(ctx, p.Game, p.Lobby)
	if err != nil {
		logger.Error("failed to do leader election", zap.Error(err), zap.String("game", p.Game), zap.String("lobby", p.Lobby))
	}
}

func (manager *TimeoutManager) Reconnected(ctx context.Context, peerID, secret, gameID, connectionID string) (bool, []string, error) {
	logger := logging.GetLogger(ctx)

	logger.Debug("peer marked as reconnected", zap.String("peer", peerID))
	return manager.Store.MarkPeerAsReconnected(ctx, peerID, secret, gameID, connectionID)
}

func (manager *TimeoutManager) RecordPeerActivity(peerID, connectionID string) {
	if peerID == "" {
		return
	}

	manager.activityMutex.Lock()
	defer manager.activityMutex.Unlock()

	if manager.activePeers == nil {
		manager.activePeers = make(map[peerConnection]struct{})
	}

	manager.activePeers[peerConnection{ID: peerID, ConnectionID: connectionID}] = struct{}{}
}

func (manager *TimeoutManager) RunActivityUpdates(ctx context.Context) {
	ticker := time.NewTicker(peerActivityFlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			manager.activityMutex.Lock()
			active := manager.activePeers
			manager.activePeers = nil
			manager.activityMutex.Unlock()

			if len(active) == 0 {
				continue
			}

			peerIDs := make([]string, 0, len(active))
			connectionIDs := make([]string, 0, len(active))

			for peer := range active {
				peerIDs = append(peerIDs, peer.ID)
				connectionIDs = append(connectionIDs, peer.ConnectionID)
			}

			// Bound flush time so batches cannot build up behind a slow database.
			flushCtx, cancel := context.WithTimeout(ctx, peerActivityFlushInterval)
			err := manager.Store.MarkPeersAsActive(flushCtx, peerIDs, connectionIDs)
			cancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				logging.GetLogger(ctx).Error("failed to update peer activity", zap.Error(err))
			}

			// Failed batches are discarded. Retrying with NOW() would make old
			// traffic appear fresh; subsequent messages populate the next batch.
			// Since it takes many flush intervals for a peer to be seen as inactive,
			// occasional missed updates are not critical.
		}
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
