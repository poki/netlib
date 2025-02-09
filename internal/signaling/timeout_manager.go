package signaling

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/signaling/stores"
	"go.uber.org/zap"
)

type TimeoutManager struct {
	DisconnectThreshold time.Duration

	Store stores.Store
}

func (manager *TimeoutManager) Run(ctx context.Context) {
	logger := logging.GetLogger(ctx)

	if err := manager.Store.ResetAllPeerLastSeen(ctx); err != nil {
		logger.Error("failed to reset all peer last seen", zap.Error(err))
	}

	if manager.DisconnectThreshold == 0 {
		// We update peer activity every 30 seconds. Make it possible to somehow miss
		// two updates before we consider a peer timed out.
		manager.DisconnectThreshold = time.Second * 90
	}

	for ctx.Err() == nil {
		time.Sleep(time.Second)
		manager.RunOnce(ctx)
	}
}

func (manager *TimeoutManager) RunOnce(ctx context.Context) {
	logger := logging.GetLogger(ctx)

	for ctx.Err() == nil {
		peerID, disconnected, gameLobbies, err := manager.Store.ClaimNextTimedOutPeer(ctx, manager.DisconnectThreshold)
		if err != nil {
			logger.Error("failed to claim next timedout peer", zap.Error(err))
		}
		if peerID == "" {
			break
		}

		for gameID, lobbies := range gameLobbies {
			for _, lobbyCode := range lobbies {
				logger.Info("peer timeout", zap.String("peer", peerID), zap.String("game", gameID), zap.String("lobby", lobbyCode))

				if err := manager.disconnectPeerInLobby(ctx, peerID, gameID, lobbyCode); err != nil {
					logger.Error("failed to disconnect peer", zap.Error(err), zap.String("peer", peerID), zap.String("game", gameID), zap.String("lobby", lobbyCode))
				}

				// If the peer wasn't disconnected normally, they might still be the leader of a lobby.
				// Just to be sure, do a leader election.
				if !disconnected {
					if err := manager.doLeaderElectionAndPublish(ctx, gameID, lobbyCode); err != nil {
						logger.Error("failed to do leader election", zap.Error(err), zap.String("peer", peerID), zap.String("game", gameID), zap.String("lobby", lobbyCode))
					}
				}
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
	} else {
		err := manager.doLeaderElectionAndPublish(ctx, p.Game, p.Lobby)
		if err != nil {
			logger.Error("failed to do leader election", zap.Error(err), zap.String("game", p.Game), zap.String("lobby", p.Lobby))
		}
	}
}

func (manager *TimeoutManager) Reconnected(ctx context.Context, peerID, secret, gameID string) (bool, []string, error) {
	logger := logging.GetLogger(ctx)

	logger.Debug("peer marked as reconnected", zap.String("peer", peerID))
	return manager.Store.MarkPeerAsReconnected(ctx, peerID, secret, gameID)
}

func (manager *TimeoutManager) MarkPeerAsActive(ctx context.Context, peerID string) {
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
