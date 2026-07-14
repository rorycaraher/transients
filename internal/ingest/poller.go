// Package ingest polls the Cloudflare Queue fed by R2 object-create event
// notifications, and turns newly discovered objects into ready tracks. It is
// the single ingestion path for both presigned browser uploads and files
// dropped in via rclone.
package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"time"

	"github.com/rorycaraher/transients/internal/idgen"
	"github.com/rorycaraher/transients/internal/r2"
	"github.com/rorycaraher/transients/internal/store"
)

const (
	batchSize           = 10
	visibilityTimeoutMS = 60_000
)

type Poller struct {
	queue    *queueClient
	store    *store.Store
	r2       *r2.Client
	interval time.Duration
	log      *slog.Logger
}

func NewPoller(accountID, apiToken, queueID string, st *store.Store, r2c *r2.Client, interval time.Duration, log *slog.Logger) *Poller {
	return &Poller{
		queue: &queueClient{
			http:      &http.Client{Timeout: 30 * time.Second},
			apiToken:  apiToken,
			accountID: accountID,
			queueID:   queueID,
		},
		store:    st,
		r2:       r2c,
		interval: interval,
		log:      log,
	}
}

// Run blocks, polling on a ticker until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.pollOnce(ctx); err != nil {
				p.log.Error("ingest poll failed", "err", err)
			}
		}
	}
}

func (p *Poller) pollOnce(ctx context.Context) error {
	messages, err := p.queue.pull(ctx, batchSize, visibilityTimeoutMS)
	if err != nil {
		return fmt.Errorf("pull messages: %w", err)
	}

	var acks []string
	for _, msg := range messages {
		if err := p.handleMessage(ctx, msg); err != nil {
			p.log.Error("failed to process queue message, leaving for retry", "id", msg.ID, "err", err)
			continue
		}
		acks = append(acks, msg.LeaseID)
	}

	if len(acks) > 0 {
		if err := p.queue.ack(ctx, acks); err != nil {
			return fmt.Errorf("ack messages: %w", err)
		}
	}
	return nil
}

func (p *Poller) handleMessage(ctx context.Context, msg queueMessage) error {
	var event r2Event
	if err := json.Unmarshal([]byte(msg.Body), &event); err != nil {
		return fmt.Errorf("unmarshal r2 event: %w", err)
	}

	if !event.isCreate() {
		return nil // not an object-create event, nothing to ingest
	}

	return p.ingestObject(ctx, event.Object.Key)
}

func (p *Poller) ingestObject(ctx context.Context, key string) error {
	track, err := p.store.GetByObjectKey(key)
	if err != nil && err != store.ErrNotFound {
		return fmt.Errorf("lookup track by object key: %w", err)
	}

	if err == store.ErrNotFound {
		// Discovered with no pre-existing pending row: an rclone drop.
		slug := idgen.New()
		title := path.Base(key)
		if err := p.store.CreateFromDiscovery(slug, key, title); err != nil {
			return fmt.Errorf("create discovered track: %w", err)
		}
		track = &store.Track{Slug: slug, ObjectKey: key, Title: title}
	}

	meta, err := p.r2.Head(ctx, key)
	if errors.Is(err, r2.ErrNotFound) {
		// Object was deleted (e.g. via the admin dashboard) after this
		// event was already enqueued. Nothing to ingest; ack and move on.
		p.log.Warn("ingest: object no longer exists, dropping event", "object_key", key)
		_ = p.store.MarkFailed(track.Slug)
		return nil
	}
	if err != nil {
		_ = p.store.MarkFailed(track.Slug)
		return fmt.Errorf("head object %s: %w", key, err)
	}

	if err := p.store.MarkReady(track.Slug, meta.ContentType, meta.SizeBytes); err != nil {
		return fmt.Errorf("mark track ready: %w", err)
	}

	p.log.Info("ingested track", "slug", track.Slug, "object_key", key, "size_bytes", meta.SizeBytes)
	return nil
}
