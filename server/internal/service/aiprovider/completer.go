package aiprovider

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Completer is the tenant-aware chat interface satisfied by *Resolver (AP.4 FR-2).
type Completer interface {
	Complete(ctx context.Context, orgID *uuid.UUID, modelOverride string, messages []Message, opts ...ChatOptions) (ChatResult, CallMeta, error)
}

// Streamer is the tenant-aware streaming interface satisfied by *Resolver.
type Streamer interface {
	CompleteStream(ctx context.Context, orgID *uuid.UUID, modelOverride string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, CallMeta, error)
}

// VisionCompleter is the tenant-aware vision interface satisfied by *Resolver.
type VisionCompleter interface {
	CompleteVision(ctx context.Context, orgID *uuid.UUID, modelOverride string, messages []Message, opts ...ChatOptions) (ChatResult, CallMeta, error)
}

// BoundCompleter fixes org scope so services need not thread orgID on every call.
type BoundCompleter struct {
	Resolver *Resolver
	OrgID    *uuid.UUID
}

func (b BoundCompleter) require() error {
	if b.Resolver == nil {
		return fmt.Errorf("aiprovider: nil resolver")
	}
	return nil
}

// Complete implements a scoped Completer without orgID on each call.
func (b BoundCompleter) Complete(ctx context.Context, modelOverride string, messages []Message, opts ...ChatOptions) (ChatResult, CallMeta, error) {
	if err := b.require(); err != nil {
		return ChatResult{}, CallMeta{}, err
	}
	return b.Resolver.Complete(ctx, b.OrgID, modelOverride, messages, opts...)
}

// CompleteStream streams with fixed org scope.
func (b BoundCompleter) CompleteStream(ctx context.Context, modelOverride string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, CallMeta, error) {
	if err := b.require(); err != nil {
		return ChatResult{}, CallMeta{}, err
	}
	return b.Resolver.CompleteStream(ctx, b.OrgID, modelOverride, messages, onChunk, opts...)
}

// CompleteVision runs vision with fixed org scope.
func (b BoundCompleter) CompleteVision(ctx context.Context, modelOverride string, messages []Message, opts ...ChatOptions) (ChatResult, CallMeta, error) {
	if err := b.require(); err != nil {
		return ChatResult{}, CallMeta{}, err
	}
	return b.Resolver.CompleteVision(ctx, b.OrgID, modelOverride, messages, opts...)
}

// ScopedCompleter is the org-bound chat surface used by feature services.
type ScopedCompleter interface {
	Complete(ctx context.Context, modelOverride string, messages []Message, opts ...ChatOptions) (ChatResult, CallMeta, error)
}

// ScopedVisionCompleter is the org-bound vision surface.
type ScopedVisionCompleter interface {
	CompleteVision(ctx context.Context, modelOverride string, messages []Message, opts ...ChatOptions) (ChatResult, CallMeta, error)
}
