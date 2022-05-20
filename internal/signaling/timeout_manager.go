package signaling

import (
	"context"
	"sync"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal/util"
	"go.uber.org/zap"
)

type timedPeer struct {
	peer *Peer
	time time.Time
}

type TimeoutManager struct {
	sync.Mutex

	DisconnectThreshold time.Duration

	peers map[string]timedPeer
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

	i.Lock()
	defer i.Unlock()

	now := util.Now(ctx)
	for _, tp := range i.peers {
		p := tp.peer
		t := tp.time
		if now.Sub(t) > i.DisconnectThreshold {
			logger.Debug("peer timed out closing peer", zap.String("id", p.ID))
			delete(i.peers, p.ID)
			go p.Close()
		}
	}
}

func (i *TimeoutManager) Disconnected(ctx context.Context, p *Peer) {
	logger := logging.GetLogger(ctx)

	i.Lock()
	defer i.Unlock()

	if i.peers == nil {
		i.peers = make(map[string]timedPeer)
	}

	logger.Debug("peer marked as disconnected", zap.String("id", p.ID))
	i.peers[p.ID] = timedPeer{
		peer: p,
		time: util.Now(ctx),
	}
}

func (i *TimeoutManager) Reconnected(ctx context.Context, p *Peer) bool {
	logger := logging.GetLogger(ctx)

	i.Lock()
	defer i.Unlock()

	if i.peers == nil {
		return false
	}

	logger.Debug("peer marked as reconnected", zap.String("id", p.ID))
	_, seen := i.peers[p.ID]
	delete(i.peers, p.ID)
	return seen
}
