package eventbus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/ddd/event/outbox"
	outboxxorm "github.com/go-jimu/contrib/event/outbox/xorm"
	jimuxorm "github.com/go-jimu/contrib/transaction/xorm"
	"github.com/go-jimu/template/internal/pkg/telemetry"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/propagation"
)

// OutboxConfig contains the Outbox's Relay policy. Durations use Go syntax.
type OutboxConfig struct {
	// Owner names this instance in record leases, at most 191 bytes. Empty
	// means the hostname, the pod name in Kubernetes.
	Owner        string `json:"owner" toml:"owner" yaml:"owner"`
	Lease        string `json:"lease" toml:"lease" yaml:"lease"`
	PollInterval string `json:"poll-interval" toml:"poll-interval" yaml:"poll-interval"`
	Workers      int    `json:"workers" toml:"workers" yaml:"workers"`
	BatchSize    int    `json:"batch-size" toml:"batch-size" yaml:"batch-size"`
	MinBackoff   string `json:"min-backoff" toml:"min-backoff" yaml:"min-backoff"`
	MaxBackoff   string `json:"max-backoff" toml:"max-backoff" yaml:"max-backoff"`
	Retention    string `json:"retention" toml:"retention" yaml:"retention"`
}

// maxOwnerLength is the width of outbox_records.lease_owner; a longer owner
// fails every insert.
const maxOwnerLength = 191

// outboxPolicy is OutboxConfig parsed.
type outboxPolicy struct {
	lease, pollInterval, minBackoff, maxBackoff, retention time.Duration
}

// Validate checks the Outbox policy.
func (c OutboxConfig) Validate() error {
	_, err := c.parse()
	return err
}

func (c OutboxConfig) parse() (outboxPolicy, error) {
	var p outboxPolicy
	for _, d := range []struct {
		name  string
		value string
		into  *time.Duration
	}{
		{"lease", c.Lease, &p.lease},
		{"poll-interval", c.PollInterval, &p.pollInterval},
		{"min-backoff", c.MinBackoff, &p.minBackoff},
		{"max-backoff", c.MaxBackoff, &p.maxBackoff},
		{"retention", c.Retention, &p.retention},
	} {
		parsed, err := time.ParseDuration(d.value)
		if err != nil {
			return p, oops.With("option", "outbox."+d.name).With("value", d.value).Wrap(err)
		}
		if parsed <= 0 {
			return p, fmt.Errorf("eventbus outbox.%s must be positive", d.name)
		}
		*d.into = parsed
	}
	if len(c.Owner) > maxOwnerLength {
		return p, fmt.Errorf("eventbus outbox.owner is longer than %d bytes", maxOwnerLength)
	}
	if c.Workers < 1 || c.BatchSize < 1 {
		return p, errors.New("eventbus outbox.workers and outbox.batch-size must be positive")
	}
	if p.maxBackoff < p.minBackoff {
		return p, errors.New("eventbus outbox.max-backoff cannot be below outbox.min-backoff")
	}
	return p, nil
}

// NewOutbox creates the Outbox on the business database: the Domain Event
// dispatcher of use cases, whose Relay pkg.Lifecycle runs. Bounded contexts
// register their event types with the Registry and subscribe their must-run
// handlers to the Outbox.
func NewOutbox(
	config Config,
	db *jimuxorm.DB,
	registry *outbox.Registry,
	bestEffort *event.InMemoryDispatcher,
	logger *slog.Logger,
) (*outbox.Outbox, event.Dispatcher, error) {
	policy, err := config.Outbox.parse()
	if err != nil {
		return nil, nil, err
	}
	store, err := outboxxorm.New(db)
	if err != nil {
		return nil, nil, err
	}
	owner := config.Outbox.Owner
	if owner == "" {
		owner, _ = os.Hostname()
	}
	o, err := outbox.New(store, registry,
		outbox.WithLogger(logger.With(slog.String("component", "outbox"))),
		outbox.WithOwner(owner),
		outbox.WithLease(policy.lease),
		outbox.WithPollInterval(policy.pollInterval),
		outbox.WithWorkers(config.Outbox.Workers),
		outbox.WithBatchSize(config.Outbox.BatchSize),
		outbox.WithBackoff(policy.minBackoff, policy.maxBackoff),
		outbox.WithRetention(policy.retention),
		outbox.WithBestEffort(bestEffort),
		// A must-run handler continues the trace of the request that raised
		// the event, however long after it runs.
		outbox.WithTrace(captureTrace, restoreTrace),
	)
	if err != nil {
		return nil, nil, err
	}
	return o, o, nil
}

func captureTrace(ctx context.Context) map[string]string {
	carrier := propagation.MapCarrier{}
	telemetry.Propagator.Inject(ctx, carrier)
	return carrier
}

func restoreTrace(ctx context.Context, carrier map[string]string) context.Context {
	return telemetry.Propagator.Extract(ctx, propagation.MapCarrier(carrier))
}
