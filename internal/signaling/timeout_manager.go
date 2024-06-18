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
	if i.DisconnectThreshold == 0 {
		i.DisconnectThreshold = time.Minute
	}

	for ctx.Err() == nil {
		time.Sleep(time.Second)
		i.RunOnce(ctx)
	}
}

func (i *TimeoutManager) RunOnce(ctx context.Context) {
	logger := logging.GetLogger(ctx)

	for ctx.Err() == nil {
		hasNext, err := i.Store.ClaimNextTimedOutPeer(ctx, i.DisconnectThreshold, func(peerID, gameID string, lobbies []string) error {
			logger.Info("peer timed out closing peer", zap.String("id", peerID))

			for _, lobby := range lobbies {
				if err := i.disconnectPeerInLobby(ctx, peerID, gameID, lobby, logger); err != nil {
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
}

func (i *TimeoutManager) disconnectPeerInLobby(ctx context.Context, peerID string, gameID string, lobby string, logger *zap.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	packet := DisconnectPacket{
		Type: "disconnect",
		ID:   peerID,
	}
	data, err := json.Marshal(packet)
	if err != nil {
		logger.Error("failed to marshal disconnect packet", zap.Error(err))
		return err
	}

	err = i.Store.Publish(ctx, gameID+lobby, data)
	if err != nil {
		logger.Error("failed to publish disconnect packet", zap.Error(err))
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
			result, err := i.Store.DoLeaderElection(ctx, p.Game, lobby)
			if err != nil {
				logger.Error("failed to do leader election", zap.Error(err))
			} else if result != nil {
				packet := LeaderPacket{
					Type:   "leader",
					Leader: result.Leader,
					Term:   result.Term,
				}
				data, err := json.Marshal(packet)
				if err != nil {
					logger.Error("failed to marshal leader packet", zap.Error(err))
				} else {
					err = i.Store.Publish(ctx, p.Game+lobby, data)
					if err != nil {
						logger.Error("failed to publish leader packet", zap.Error(err))
					}
				}
			}
		}
	}
}

func (i *TimeoutManager) Reconnected(ctx context.Context, id, secret, game string) (bool, []string, error) {
	logger := logging.GetLogger(ctx)

	logger.Debug("peer marked as reconnected", zap.String("id", id))
	return i.Store.ReconnectPeer(ctx, id, secret, game)
}
