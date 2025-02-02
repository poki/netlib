package signaling

import (
	"context"
	"encoding/json"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/signaling/stores"
	"go.uber.org/zap"
)

type TimeoutManager struct {
	DisconnectThreshold time.Duration

	Store stores.Store
}

func (i *TimeoutManager) Run(ctx context.Context) {
	logger := logging.GetLogger(ctx)

	if i.DisconnectThreshold == 0 {
		i.DisconnectThreshold = time.Minute
	}

	// The first time the manager starts it marks all peers from all lobbies
	// as active so no peers are missed.
	if err := i.Store.MarkAllPeersAsActive(ctx); err != nil {
		logger.Error("failed to mark all peers as active", zap.Error(err))
	}

	for ctx.Err() == nil {
		time.Sleep(time.Second)
		i.RunOnce(ctx)
	}
}

func (i *TimeoutManager) RunOnce(ctx context.Context) {
	logger := logging.GetLogger(ctx)

	// First remove all timed out peers that we expect to reconnect.
	for ctx.Err() == nil {
		hasNext, err := i.Store.ClaimNextTimedOutPeer(ctx, i.DisconnectThreshold, func(peerID, gameID string, lobbies []string) error {
			logger.Info("peer timed out closing peer", zap.String("peer", peerID), zap.Strings("lobbies", lobbies))

			if err := i.Store.RemovePeerActivity(ctx, peerID); err != nil {
				return err
			}

			for _, lobby := range lobbies {
				if err := i.disconnectPeerInLobby(ctx, peerID, gameID, lobby); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			logger.Error("failed to claim next timed out peer", zap.Error(err))
		}
		if !hasNext {
			break
		}
	}

	// If a peer still exists a 2 minutes after it should have been disconnected, then
	// we assume the disconnect somehow failed.
	inactiveThreshold := i.DisconnectThreshold + 2*time.Minute

	// Then remove all peers that haven't seen any activity in a while.
	// These are mostly peers where a disconnect failed (for example after a pod restart).
	for ctx.Err() == nil {
		peerID, gameIDs, lobbies, err := i.Store.ClaimNextInactivePeer(ctx, inactiveThreshold)
		if err != nil {
			logger.Error("failed to claim next inactive peer", zap.Error(err))
		}
		if peerID == "" {
			break
		}

		for _, gameID := range gameIDs {
			for _, lobbyCode := range lobbies {
				logger.Info("peer inactive", zap.String("peer", peerID), zap.String("game", gameID), zap.String("lobby", lobbyCode))

				if err := i.disconnectPeerInLobby(ctx, peerID, gameID, lobbyCode); err != nil {
					logger.Error("failed to disconnect peer", zap.Error(err), zap.String("peer", peerID), zap.String("game", gameID), zap.String("lobby", lobbyCode))
				}

				if err := i.doLeaderElectionAndPublish(ctx, gameID, lobbyCode); err != nil {
					logger.Error("failed to do leader election", zap.Error(err), zap.String("peer", peerID), zap.String("game", gameID), zap.String("lobby", lobbyCode))
				}
			}
		}
	}
}

func (i *TimeoutManager) disconnectPeerInLobby(ctx context.Context, peerID string, gameID string, lobby string) error {
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

	err = i.Store.Publish(ctx, gameID+lobby, data)
	if err != nil {
		return err
	}
	return nil
}

func (i *TimeoutManager) Disconnected(ctx context.Context, p *Peer) {
	logger := logging.GetLogger(ctx)

	if p.ID == "" {
		return
	}

	logger.Debug("peer marked as disconnected", zap.String("id", p.ID), zap.String("lobby", p.Lobby))
	lobbies := []string{}
	if p.Lobby != "" {
		lobbies = []string{p.Lobby}
	}
	err := i.Store.TimeoutPeer(ctx, p.ID, p.Secret, p.Game, lobbies)
	if err != nil {
		logger.Error("failed to record timeout peer", zap.Error(err))
	} else {
		for _, lobby := range lobbies {
			err := i.doLeaderElectionAndPublish(ctx, p.Game, lobby)
			if err != nil {
				logger.Error("failed to do leader election", zap.Error(err), zap.String("game", p.Game), zap.String("lobby", lobby))
			}
		}
	}
}

func (i *TimeoutManager) Reconnected(ctx context.Context, peerID, secret, game string) (bool, []string, error) {
	logger := logging.GetLogger(ctx)

	logger.Debug("peer marked as reconnected", zap.String("peer", peerID))
	return i.Store.ReconnectPeer(ctx, peerID, secret, game)
}

func (i *TimeoutManager) MarkPeerAsActive(ctx context.Context, peerID string) {
	logger := logging.GetLogger(ctx)

	err := i.Store.UpdatePeerActivity(ctx, peerID)
	if err != nil {
		logger.Error("failed to mark peer as active", zap.Error(err), zap.String("peer", peerID))
	}
}

func (i *TimeoutManager) doLeaderElectionAndPublish(ctx context.Context, gameID, lobbyCode string) error {
	result, err := i.Store.DoLeaderElection(ctx, gameID, lobbyCode)
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

	err = i.Store.Publish(ctx, gameID+lobbyCode, data)
	if err != nil {
		return err
	}

	return nil
}
