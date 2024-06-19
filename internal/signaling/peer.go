package signaling

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/metrics"
	"github.com/poki/netlib/internal/signaling/stores"
	"github.com/poki/netlib/internal/util"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Peer struct {
	store stores.Store
	conn  *websocket.Conn

	closedPacketReceived bool

	retrievedIDCallback func(context.Context, string, string, string) (bool, []string, error)

	ID     string
	Secret string
	Game   string
	Lobby  string
}

func (p *Peer) Send(ctx context.Context, packet interface{}) error {
	return wsjson.Write(ctx, p.conn, packet)
}

func (p *Peer) RequestConnection(ctx context.Context, otherID string) error {
	toMe := ConnectPacket{
		Type:   "connect",
		ID:     otherID,
		Polite: true,
	}
	toThem := ConnectPacket{
		Type:   "connect",
		ID:     p.ID,
		Polite: false,
	}

	err := wsjson.Write(ctx, p.conn, toMe)
	if err != nil {
		return err
	}

	data, err := json.Marshal(toThem)
	if err != nil {
		return err
	}

	err = p.store.Publish(ctx, p.Game+p.Lobby+otherID, data)
	if err != nil {
		return err
	}

	go metrics.Record(ctx, "rtc", "attempt", p.Game, p.ID, p.Lobby, "target", otherID)
	go metrics.Record(ctx, "rtc", "attempt", p.Game, otherID, p.Lobby, "target", p.ID)

	return nil
}

func (p *Peer) ForwardMessage(ctx context.Context, raw []byte) {
	logger := logging.GetLogger(ctx)
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	err := p.conn.Write(ctx, websocket.MessageText, raw)
	if err != nil && !util.IsPipeError(err) {
		logger.Warn("failed to forward message", zap.Error(err))
	}
}

func (p *Peer) HandlePacket(ctx context.Context, typ string, raw []byte) error {
	logger := logging.GetLogger(ctx).With(zap.String("peer", p.ID))
	logger.Debug("handling packet", zap.String("type", typ), zap.ByteString("data", raw))

	var err error
	switch typ {
	case "hello":
		packet := HelloPacket{}
		if err := json.Unmarshal(raw, &packet); err != nil {
			return fmt.Errorf("unable to unmarshal json: %w", err)
		}
		err = p.HandleHelloPacket(ctx, packet)
		if err != nil {
			return fmt.Errorf("unable to handle packet: %w", err)
		}

	case "leave": // legacy, backwards compatibility
		fallthrough
	case "close":
		packet := ClosePacket{}
		if err := json.Unmarshal(raw, &packet); err != nil {
			return fmt.Errorf("unable to unmarshal json: %w", err)
		}
		err = p.HandleClosePacket(ctx, packet)
		if err != nil {
			return fmt.Errorf("unable to handle packet: %w", err)
		}

	case "list":
		packet := ListPacket{}
		if err := json.Unmarshal(raw, &packet); err != nil {
			return fmt.Errorf("unable to unmarshal json: %w", err)
		}
		err = p.HandleListPacket(ctx, packet)
		if err != nil {
			return fmt.Errorf("unable to handle packet: %w", err)
		}

	case "create":
		packet := CreatePacket{}
		if err := json.Unmarshal(raw, &packet); err != nil {
			return fmt.Errorf("unable to unmarshal json: %w", err)
		}
		err = p.HandleCreatePacket(ctx, packet)
		if err != nil {
			return fmt.Errorf("unable to handle packet: %w", err)
		}

	case "join":
		packet := JoinPacket{}
		if err := json.Unmarshal(raw, &packet); err != nil {
			return fmt.Errorf("unable to unmarshal json: %w", err)
		}
		err = p.HandleJoinPacket(ctx, packet)
		if err != nil {
			return fmt.Errorf("unable to handle packet: %w", err)
		}

	case "update":
		packet := UpdatePacket{}
		if err := json.Unmarshal(raw, &packet); err != nil {
			return fmt.Errorf("unable to unmarshal json: %w", err)
		}
		err = p.HandleUpdatePacket(ctx, packet)
		if err != nil {
			return fmt.Errorf("unable to handle packet: %w", err)
		}

	// case "leave":

	case "connected": // TODO: Do we want to keep track of connections between peers?
	case "disconnected": // TODO: Do we want to keep track of connections between peers?

	case "candidate":
		fallthrough
	case "description":
		routing := ForwardablePacket{}
		if err := json.Unmarshal(raw, &routing); err != nil {
			util.ErrorAndDisconnect(ctx, p.conn, err)
		}
		if routing.Source != p.ID {
			util.ErrorAndDisconnect(ctx, p.conn, fmt.Errorf("invalid source set"))
		}
		err = p.store.Publish(ctx, p.Game+p.Lobby+routing.Recipient, raw)
		if err == stores.ErrNoSuchTopic {
			util.ReplyError(ctx, p.conn, &MissingRecipientError{
				Recipient: routing.Recipient,
				Cause:     err,
			})
		} else if err != nil {
			return fmt.Errorf("unable to publish packet to forward: %w", err)
		}

	default:
		logger.Warn("unknown packet type received", zap.String("type", typ))
	}

	return nil
}

func (p *Peer) HandleHelloPacket(ctx context.Context, packet HelloPacket) error {
	logger := logging.GetLogger(ctx)
	if p.Game != "" {
		return fmt.Errorf("already introduced %s for game %s", p.ID, p.Game)
	}
	if !util.IsUUID(packet.Game) {
		return fmt.Errorf("no game id supplied")
	}

	hasReconnected := false
	var reconnectingLobbies []string
	if packet.ID != "" && packet.Secret != "" {
		logger.Info("peer reconnecting", zap.String("game", packet.Game), zap.String("peer", packet.ID))
		var err error
		hasReconnected, reconnectingLobbies, err = p.retrievedIDCallback(ctx, packet.ID, packet.Secret, packet.Game)
		if err != nil {
			return fmt.Errorf("unable to reconnect: %w", err)
		}
		if !hasReconnected {
			logger.Info("peer failed reconnecting", zap.String("game", p.Game), zap.String("peer", p.ID))

			err := fmt.Errorf("failed to reconnect, missing pid or invalid secret")
			err = util.ErrorWithCode(err, "reconnect-failed")
			util.ReplyError(ctx, p.conn, err)

			// Return nil. Peers with old code will stay connected, but will not be able to do anything. Peers with new
			// code will close their network completely based on the error send above.
			// This is to prevent the peer from being disconnected by the server making it reconnect again right away.
			return nil
		}

		p.Game = packet.Game
		p.ID = packet.ID
		p.Secret = packet.Secret
	} else {
		p.Game = packet.Game
		p.ID = util.GeneratePeerID(ctx)
		p.Secret = util.GenerateSecret(ctx)
		logger.Info("peer connecting", zap.String("game", p.Game), zap.String("peer", p.ID))
	}

	err := p.Send(ctx, WelcomePacket{
		Type:   "welcome",
		ID:     p.ID,
		Secret: p.Secret,
	})
	if err != nil {
		return err
	}

	if hasReconnected {
		for _, lobbyID := range reconnectingLobbies {
			logger.Info("peer rejoining lobby", zap.String("game", p.Game), zap.String("peer", p.ID), zap.String("lobby", p.Lobby))
			p.Lobby = lobbyID
			p.store.Subscribe(ctx, p.ForwardMessage, p.Game, p.Lobby, p.ID)

			go metrics.Record(ctx, "lobby", "reconnected", p.Game, p.ID, p.Lobby)

			// We just reconnected, and we might be the only peer in the lobby.
			// So do an election to make sure we then become the leader.
			// This won't do anything if there's already a leader.
			changed, err := p.doLeaderElectionAndPublish(ctx)
			if err != nil {
				return err
			} else if !changed {
				// No new leader was elected, but we might still have missed
				// changes in leadership while we were disconnected.
				// So send the current leader to the client just in case.

				lobbyInfo, err := p.store.GetLobby(ctx, p.Game, lobbyID)
				if err != nil {
					return err
				}

				err = p.Send(ctx, LeaderPacket{
					Type:   "leader",
					Leader: lobbyInfo.Leader,
					Term:   lobbyInfo.Term,
				})
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (p *Peer) HandleClosePacket(ctx context.Context, packet ClosePacket) error {
	logger := logging.GetLogger(ctx)
	go metrics.Record(ctx, "client", "close", p.Game, p.ID, p.Lobby)

	p.closedPacketReceived = true

	logger.Info("client closed",
		zap.String("game", p.Game),
		zap.String("peer", p.ID),
		zap.String("lobby", p.Lobby),
		zap.String("reason", packet.Reason),
	)

	if p.Lobby != "" {
		err := p.store.LeaveLobby(ctx, p.Game, p.Lobby, p.ID)
		if err != nil {
			return fmt.Errorf("unable to leave lobby: %w", err)
		}
		packet := DisconnectPacket{
			Type: "disconnect",
			ID:   p.ID,
		}
		data, err := json.Marshal(packet)
		if err == nil {
			err := p.store.Publish(ctx, p.Game+p.Lobby, data)
			if err != nil {
				logger.Error("failed to publish disconnect packet", zap.Error(err))
			}
		}

		_, err = p.doLeaderElectionAndPublish(ctx)
		if err != nil {
			return err
		}

		p.Lobby = ""
	}

	return nil
}

func (p *Peer) HandleListPacket(ctx context.Context, packet ListPacket) error {
	logger := logging.GetLogger(ctx)
	if p.ID == "" {
		return fmt.Errorf("peer not connected")
	}
	logger.Debug("listing lobbies", zap.String("game", p.Game), zap.String("peer", p.ID))
	lobbies, err := p.store.ListLobbies(ctx, p.Game, packet.Filter)
	if err != nil {
		return err
	}
	if lobbies == nil {
		lobbies = []stores.Lobby{}
	}
	return p.Send(ctx, LobbiesPacket{
		RequestID: packet.RequestID,
		Type:      "lobbies",
		Lobbies:   lobbies,
	})
}

func (p *Peer) HandleCreatePacket(ctx context.Context, packet CreatePacket) error {
	logger := logging.GetLogger(ctx)
	if p.ID == "" {
		return fmt.Errorf("peer not connected")
	}
	if p.Lobby != "" {
		return fmt.Errorf("already in a lobby %s:%s as %s", p.Game, p.Lobby, p.ID)
	}

	if packet.CanUpdateBy == "" {
		packet.CanUpdateBy = stores.CanUpdateByCreator
	}

	attempts := 20
	for ; attempts > 0; attempts-- {
		switch packet.CodeFormat {
		case "short":
			p.Lobby = util.GenerateShortLobbyCode(ctx)
		default:
			p.Lobby = util.GenerateLobbyCode(ctx)
		}

		err := p.store.CreateLobby(ctx, p.Game, p.Lobby, p.ID, packet.Public, packet.CustomData, packet.CanUpdateBy)
		if err != nil {
			if err == stores.ErrLobbyExists {
				continue
			}
			return err
		}
		break
	}
	if attempts <= 0 {
		return fmt.Errorf("unable to create lobby, too many attempts to find a unique code")
	}

	p.store.Subscribe(ctx, p.ForwardMessage, p.Game, p.Lobby, p.ID)

	lobby, err := p.store.GetLobby(ctx, p.Game, p.Lobby)
	if err != nil {
		return err
	}

	logger.Info("created lobby", zap.String("game", p.Game), zap.String("lobby", p.Lobby), zap.String("peer", p.ID))
	go metrics.Record(ctx, "lobby", "created", p.Game, p.ID, p.Lobby)

	return p.Send(ctx, JoinedPacket{
		RequestID: packet.RequestID,
		Type:      "joined",
		LobbyCode: p.Lobby, // backwards compatibility
		LobbyInfo: lobby,
	})
}

func (p *Peer) HandleJoinPacket(ctx context.Context, packet JoinPacket) error {
	logger := logging.GetLogger(ctx)
	if p.ID == "" {
		return fmt.Errorf("peer not connected")
	}
	if p.Lobby != "" {
		return fmt.Errorf("already in a lobby %s:%s as %s", p.Game, p.Lobby, p.ID)
	}
	if packet.Lobby == "" {
		return fmt.Errorf("no lobby code supplied")
	}
	if len(packet.Lobby) > 20 {
		return fmt.Errorf("lobby code too long")
	}

	err := p.store.JoinLobby(ctx, p.Game, packet.Lobby, p.ID)
	if err != nil {
		return err
	}

	p.Lobby = packet.Lobby
	p.store.Subscribe(ctx, p.ForwardMessage, p.Game, p.Lobby, p.ID)

	// Lobby might be empty when joining, then you need to become the leader.
	_, err = p.doLeaderElectionAndPublish(ctx)
	if err != nil {
		return err
	}

	lobby, err := p.store.GetLobby(ctx, p.Game, p.Lobby)
	if err != nil {
		return err
	}

	err = p.Send(ctx, JoinedPacket{
		RequestID: packet.RequestID,
		Type:      "joined",
		LobbyCode: p.Lobby, // backwards compatibility
		LobbyInfo: lobby,
	})
	if err != nil {
		return err
	}

	for _, otherID := range lobby.Peers {
		if otherID == p.ID {
			continue
		}

		err := p.RequestConnection(ctx, otherID)
		if err != nil {
			return err
		}
	}

	logger.Info("joined lobby",
		zap.String("game", p.Game),
		zap.String("lobby", p.Lobby),
		zap.String("peer", p.ID),
		zap.Strings("peers", lobby.Peers))
	go metrics.Record(ctx, "lobby", "joined", p.Game, p.ID, p.Lobby)

	return nil
}

func (p *Peer) HandleUpdatePacket(ctx context.Context, packet UpdatePacket) error {
	logger := logging.GetLogger(ctx)
	if p.ID == "" {
		return fmt.Errorf("peer not connected")
	}
	if p.Lobby == "" {
		return fmt.Errorf("not in a lobby")
	}
	if packet.CanUpdateBy != nil {
		if *packet.CanUpdateBy != stores.CanUpdateByCreator &&
			*packet.CanUpdateBy != stores.CanUpdateByLeader &&
			*packet.CanUpdateBy != stores.CanUpdateByAnyone &&
			*packet.CanUpdateBy != stores.CanUpdateByNone {
			return fmt.Errorf("invalid canUpdateBy value")
		}
	}

	err := p.store.UpdateCustomData(ctx, p.Game, p.Lobby, p.ID, packet.Public, packet.CustomData, packet.CanUpdateBy)
	if err != nil {
		logger.Error("failed to update lobby", zap.Error(err), zap.Any("customData", packet.CustomData))

		return p.Send(ctx, UpdatedPacket{
			RequestID: packet.RequestID,
			Type:      "updated",
			Error:     fmt.Sprintf("unable to update lobby: %v", err),
		})
	}

	lobbyInfo, err := p.store.GetLobby(ctx, p.Game, p.Lobby)
	if err != nil {
		return err
	}

	data, err := json.Marshal(UpdatedPacket{
		// Include the request ID for the peer that requested the update.
		// Other peers will ignore this.
		RequestID: packet.RequestID,

		Type:      "updated",
		LobbyInfo: lobbyInfo,
	})
	if err != nil {
		return err
	}
	return p.store.Publish(ctx, p.Game+p.Lobby, data)
}

// doLeaderElectionAndPublish will do a leader election and publish the result if a new leader was elected.
// It returns true if a new leader was elected, false if not.
func (p *Peer) doLeaderElectionAndPublish(ctx context.Context) (bool, error) {
	result, err := p.store.DoLeaderElection(ctx, p.Game, p.Lobby)
	if err != nil {
		return false, err
	}

	if result != nil {
		packet := LeaderPacket{
			Type:   "leader",
			Leader: result.Leader,
			Term:   result.Term,
		}
		data, err := json.Marshal(packet)
		if err != nil {
			return false, err
		}
		err = p.store.Publish(ctx, p.Game+p.Lobby, data)
		if err != nil {
			return false, err
		}
	}

	return result != nil, nil
}
