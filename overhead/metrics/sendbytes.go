package metrics

import (
	"time"

	"github.com/relab/hotstuff"

	"github.com/relab/hotstuff/eventloop"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/metrics/types"
	"github.com/relab/hotstuff/modules"
)

func init() {
	RegisterReplicaMetric("proposalBytes", func() any {
		return &ProposalBytes{}
	})
}

// Throughput measures throughput in commits per second, and commands per second.
type ProposalBytes struct {
	metricsLogger Logger
	opts          *modules.Options

	bytesCount uint64
}

// InitModule gives the module access to the other modules.
func (t *ProposalBytes) InitModule(mods *modules.Core) {
	var (
		eventLoop *eventloop.EventLoop
		logger    logging.Logger
	)

	mods.Get(
		&t.metricsLogger,
		&t.opts,
		&eventLoop,
		&logger,
	)

	eventLoop.RegisterHandler(hotstuff.BlockBytesEvent{}, func(event any) {
		bytesSent := event.(hotstuff.BlockBytesEvent)
		t.recordBytes(bytesSent.NumberBytes)
	})

	eventLoop.RegisterObserver(types.TickEvent{}, func(event any) {
		t.tick(event.(types.TickEvent))
	})

	logger.Info("BlockBytes metric enabled")
}

func (t *ProposalBytes) recordBytes(bytesSent int) {
	t.bytesCount += uint64(bytesSent)
}

func (t *ProposalBytes) tick(tick types.TickEvent) {
	now := time.Now()
	event := &types.SentBytes{
		Event:       types.NewReplicaEvent(uint32(t.opts.ID()), now),
		SendEventrd: t.bytesCount,
	}
	t.metricsLogger.Log(event)
	// reset count for next tick
	t.bytesCount = 0
}
