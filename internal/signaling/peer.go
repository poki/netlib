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
	store Store
	conn  *websocket.Conn

	retrievedIDCallback func(context.Context, *Peer) bool

	ID     string
	Secret string
	Game   string
	Lobby  string
}

func (p *Peer) Close() {
	if p.ID != "" && p.Game != "" && p.Lobby != "" {
		packet := DisconnectPacket{
			Type: "disconnect",
			ID:   p.ID,
		}
		data, err := json.Marshal(packet)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			others, err := p.store.LeaveLobby(ctx, p.Game, p.ID, p.Lobby)
			if err != nil {
				logging.GetLogger(ctx).Warn("failed to leave lobby", zap.Error(err))
			}
			for _, id := range others {
				if id != p.ID {
					_ = p.store.Publish(ctx, p.Game+p.Lobby+id, data)
				}
			}
		}
	}
	p.conn.Close(websocket.StatusInternalError, "error")
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

	case "leave":
		go metrics.Record(ctx, "lobby", "leave", p.Game, p.ID, p.Lobby)
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
	p.Game = packet.Game

	clientIsReconnecting := false
	if packet.ID != "" && packet.Secret != "" {
		clientIsReconnecting = true
		p.ID = packet.ID
		p.Secret = packet.Secret
		logger.Info("peer reconnecting", zap.String("game", p.Game), zap.String("peer", p.ID))
	} else {
		p.ID = util.GeneratePeerID(ctx)
		p.Secret = util.GenerateSecret(ctx)
		logger.Info("peer connecting", zap.String("game", p.Game), zap.String("peer", p.ID))
	}
	hasReconnected := p.retrievedIDCallback(ctx, p)
	if clientIsReconnecting && !hasReconnected {
		return fmt.Errorf("unable to reconnect")
	}

	if packet.Lobby != "" {
		inLobby, err := p.store.IsPeerInLobby(ctx, p.Game, p.Lobby, p.ID)
		if err != nil {
			return err
		}
		if hasReconnected && inLobby {
			logger.Debug("peer rejoining lobby", zap.String("game", p.Game), zap.String("peer", p.ID), zap.String("lobby", p.Lobby))
			p.Lobby = packet.Lobby
			go p.store.Subscribe(ctx, p.Game+p.Lobby+p.ID, p.ForwardMessage)
			go metrics.Record(ctx, "lobby", "reconnected", p.Game, p.ID, p.Lobby)
		} else {
			fakeJoinPacket := JoinPacket{
				Type:  "join",
				Lobby: p.Lobby,
			}
			err := p.HandleJoinPacket(ctx, fakeJoinPacket)
			if err != nil {
				return err
			}
			go metrics.Record(ctx, "lobby", "joined", p.Game, p.ID, p.Lobby)
		}
	}

	return p.Send(ctx, WelcomePacket{
		Type:   "welcome",
		ID:     p.ID,
		Secret: p.Secret,
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
	p.Lobby = util.GenerateLobbyCode(ctx)

	go p.store.Subscribe(ctx, p.Game+p.Lobby+p.ID, p.ForwardMessage)

	err := p.store.CreateLobby(ctx, p.Game, p.Lobby, p.ID)
	if err != nil {
		return err
	}

	_, err = p.store.JoinLobby(ctx, p.Game, p.Lobby, p.ID)
	if err != nil {
		return err
	}

	logger.Info("created lobby", zap.String("game", p.Game), zap.String("lobby", p.Lobby))
	go metrics.Record(ctx, "lobby", "created", p.Game, p.ID, p.Lobby)

	return p.Send(ctx, JoinedPacket{
		Type:  "joined",
		Lobby: p.Lobby,
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

	p.Lobby = packet.Lobby

	go p.store.Subscribe(ctx, p.Game+p.Lobby+p.ID, p.ForwardMessage)

	others, err := p.store.JoinLobby(ctx, p.Game, p.Lobby, p.ID)
	if err != nil {
		return err
	}

	err = p.Send(ctx, JoinedPacket{
		Type:  "joined",
		Lobby: p.Lobby,
	})
	if err != nil {
		return err
	}

	for _, otherID := range others {
		err := p.RequestConnection(ctx, otherID)
		if err != nil {
			return err
		}
	}

	logger.Info("joined lobby", zap.String("game", p.Game), zap.String("lobby", p.Lobby))
	go metrics.Record(ctx, "lobby", "joined", p.Game, p.ID, p.Lobby)

	return nil
}
