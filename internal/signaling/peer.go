package signaling

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/util"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Peer struct {
	store Store
	conn  *websocket.Conn

	ID    string
	Game  string
	Lobby string
}

func (p *Peer) Close() {
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

	case "joinOrCreate":
		packet := JoinOrCreatePacket{}
		if err := json.Unmarshal(raw, &packet); err != nil {
			return fmt.Errorf("unable to unmarshal json: %w", err)
		}
		err = p.HandleJoinOrCreatePacket(ctx, packet)
		if err != nil {
			return fmt.Errorf("unable to handle packet: %w", err)
		}

	case "connected": // TODO: Handle
	case "disconnected": // TODO: Handle

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
		if err != nil {
			return fmt.Errorf("unable to publish packet to forward: %w", err)
		}

	default:
		logger.Warn("unknown packet type received", zap.String("type", typ))
	}

	return nil
}

func (p *Peer) HandleCreatePacket(ctx context.Context, packet CreatePacket) error {
	if p.Game != "" || p.Lobby != "" || p.ID != "" {
		// TODO: Maybe return an error to the client.
		return fmt.Errorf("already in a lobby %s:%s as %s", p.Game, p.Lobby, p.ID)
	}
	if packet.Game == "" { // TODO: Validate uuid
		return fmt.Errorf("no game id supplied")
	}
	p.Game = packet.Game
	p.Lobby = strconv.FormatInt(rand.Int63(), 36)
	p.ID = strconv.FormatInt(rand.Int63(), 36)

	err := p.store.CreateLobby(ctx, p.Game, p.Lobby, p.ID)
	if err != nil {
		return err
	}

	go p.store.Subscribe(ctx, p.Game+p.Lobby+p.ID, p.ForwardMessage)

	return p.Send(ctx, JoinedPacket{
		Type:  "joined",
		Lobby: p.Lobby,
		ID:    p.ID,
	})
}

func (p *Peer) HandleJoinPacket(ctx context.Context, packet JoinPacket) error {
	if p.Game != "" || p.Lobby != "" || p.ID != "" {
		// TODO: Maybe return an error to the client.
		return fmt.Errorf("already in a lobby %s:%s as %s", p.Game, p.Lobby, p.ID)
	}
	if packet.Game == "" { // TODO: Validate uuid
		return fmt.Errorf("no game id supplied")
	}
	if packet.Lobby == "" {
		return fmt.Errorf("no lobby code supplied")
	}

	p.Game = packet.Game
	p.Lobby = packet.Lobby
	p.ID = strconv.FormatInt(rand.Int63(), 36)

	others, err := p.store.JoinLobby(ctx, p.Game, p.Lobby, p.ID)
	if err != nil {
		return err
	}

	go p.store.Subscribe(ctx, p.Game+p.Lobby+p.ID, p.ForwardMessage)

	err = p.Send(ctx, JoinedPacket{
		Type:  "joined",
		Lobby: p.Lobby,
		ID:    p.ID,
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

	return nil
}

func (p *Peer) HandleJoinOrCreatePacket(ctx context.Context, packet JoinOrCreatePacket) error {
	if p.Game != "" || p.Lobby != "" || p.ID != "" {
		// TODO: Maybe return an error to the client.
		return fmt.Errorf("already in a lobby %s:%s as %s", p.Game, p.Lobby, p.ID)
	}
	if packet.Game == "" { // TODO: Validate uuid
		return fmt.Errorf("no game id supplied")
	}
	if packet.Prefix == "" {
		return fmt.Errorf("no prefix supplied")
	}

	p.Game = packet.Game
	p.ID = strconv.FormatInt(rand.Int63(), 36)

	lobby, others, err := p.store.JoinOrCreateLobby(ctx, p.Game, packet.Prefix, p.ID)
	if err != nil {
		return err
	}

	p.Lobby = lobby

	go p.store.Subscribe(ctx, p.Game+p.Lobby+p.ID, p.ForwardMessage)

	err = p.Send(ctx, JoinedPacket{
		Type:  "joined",
		Lobby: p.Lobby,
		ID:    p.ID,
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

	return nil
}
