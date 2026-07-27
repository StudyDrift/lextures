package context

import (
	stdctx "context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

// IngestPolicy controls whether a host may be fetched (FR-16).
type IngestPolicy struct {
	Mode      string   // off | allowlist | public
	Allowlist []string // hostnames when allowlist
}

// Allow reports whether url may be ingested under policy.
func (p IngestPolicy) Allow(rawURL string) error {
	if KillSwitchActive() {
		return ErrKillSwitch
	}
	mode := p.Mode
	if mode == "" {
		mode = IngestPublic
	}
	switch mode {
	case IngestOff:
		return ErrIngestDisabled
	case IngestAllowlist:
		host := HostOf(rawURL)
		for _, h := range p.Allowlist {
			if strings.EqualFold(strings.TrimSpace(h), host) {
				return nil
			}
		}
		return ErrHostNotAllowlisted
	default:
		return nil
	}
}

// EnsureSource returns a cached or newly created pending/ready source for url.
func EnsureSource(ctx stdctx.Context, pool *pgxpool.Pool, orgID uuid.UUID, rawURL string, policy IngestPolicy) (*ctrepo.LinkSourceRow, error) {
	norm, err := NormalizeURL(rawURL)
	if err != nil {
		return saveBlocked(ctx, pool, orgID, rawURL, err)
	}
	if err := policy.Allow(norm); err != nil {
		return saveBlocked(ctx, pool, orgID, norm, err)
	}
	if err := ValidateFetchURL(norm); err != nil {
		return saveBlocked(ctx, pool, orgID, norm, err)
	}
	hash := HashURL(norm)
	existing, err := ctrepo.GetLinkSourceByHash(ctx, pool, orgID, hash, ExtractionVersion)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.Status == StatusReady && existing.ExpiresAt != nil && existing.ExpiresAt.After(time.Now()) {
			observeCacheHit()
			return existing, nil
		}
		return existing, nil
	}
	org := orgID
	pending := ctrepo.LinkSourceRow{
		OrgID:             &org,
		URLHash:           hash,
		URL:               norm,
		ExtractionVersion: ExtractionVersion,
		Status:            StatusPending,
	}
	return ctrepo.UpsertLinkSource(ctx, pool, pending)
}

func saveBlocked(ctx stdctx.Context, pool *pgxpool.Pool, orgID uuid.UUID, rawURL string, cause error) (*ctrepo.LinkSourceRow, error) {
	norm := rawURL
	if n, err := NormalizeURL(rawURL); err == nil {
		norm = n
	}
	hash := HashURL(norm)
	status, reason := ReasonForFetchError(cause)
	org := orgID
	row := ctrepo.LinkSourceRow{
		OrgID:             &org,
		URLHash:           hash,
		URL:               norm,
		ExtractionVersion: ExtractionVersion,
		Status:            status,
		Error:             &reason,
	}
	out, err := ctrepo.UpsertLinkSource(ctx, pool, row)
	if err != nil {
		return nil, err
	}
	observeFetch(status)
	return out, nil
}

// IngestSource fetches, extracts, chunks, and caches a source (FR-4–FR-8).
func IngestSource(ctx stdctx.Context, pool *pgxpool.Pool, sourceID uuid.UUID, policy IngestPolicy) (*ctrepo.LinkSourceRow, error) {
	src, err := ctrepo.GetLinkSource(ctx, pool, sourceID)
	if err != nil || src == nil {
		return src, err
	}
	if err := policy.Allow(src.URL); err != nil {
		return persistOutcome(ctx, pool, src, nil, err)
	}
	host := HostOf(src.URL)
	if HostBreakerOpen(host) {
		return persistOutcome(ctx, pool, src, nil, ErrHostBreakerOpen)
	}
	etag, lm := "", ""
	if src.ETag != nil {
		etag = *src.ETag
	}
	if src.LastModified != nil {
		lm = *src.LastModified
	}
	out, err := FetchURL(ctx, src.URL, etag, lm)
	if err != nil {
		return persistOutcome(ctx, pool, src, out, err)
	}
	if out.NotModified {
		now := time.Now().UTC()
		exp := now.Add(DefaultCacheTTL)
		src.Status = StatusReady
		src.FetchedAt = &now
		src.ExpiresAt = &exp
		src.Error = nil
		observeFetch("revalidated")
		return ctrepo.UpsertLinkSource(ctx, pool, *src)
	}
	extracted, err := ExtractMainContent(out.ContentType, out.Body)
	if err != nil {
		return persistOutcome(ctx, pool, src, out, err)
	}
	now := time.Now().UTC()
	exp := now.Add(DefaultCacheTTL)
	size := len(out.Body)
	final := out.FinalURL
	ct := out.ContentType
	title := extracted.Title
	lang := extracted.Lang
	text := extracted.Text
	etagOut := out.ETag
	lmOut := out.LastModified
	src.FinalURL = &final
	src.ContentType = &ct
	src.Title = &title
	src.Lang = &lang
	src.ExtractedText = &text
	src.ByteSize = &size
	if etagOut != "" {
		src.ETag = &etagOut
	}
	if lmOut != "" {
		src.LastModified = &lmOut
	}
	src.Status = StatusReady
	src.Error = nil
	src.FetchedAt = &now
	src.ExpiresAt = &exp
	saved, err := ctrepo.UpsertLinkSource(ctx, pool, *src)
	if err != nil {
		return nil, err
	}
	chunks := ChunkText(text)
	rows := make([]ctrepo.LinkChunkRow, 0, len(chunks))
	for i, ch := range chunks {
		rows = append(rows, ctrepo.LinkChunkRow{
			SourceID:   saved.ID,
			Ordinal:    i,
			Text:       ch,
			TokenCount: EstimateTokens(ch),
		})
	}
	if err := ctrepo.ReplaceLinkChunks(ctx, pool, saved.ID, rows); err != nil {
		return nil, err
	}
	observeFetch("ready")
	return saved, nil
}

func persistOutcome(ctx stdctx.Context, pool *pgxpool.Pool, src *ctrepo.LinkSourceRow, out *FetchOutcome, err error) (*ctrepo.LinkSourceRow, error) {
	status, reason := ReasonForFetchError(err)
	src.Status = status
	src.Error = &reason
	now := time.Now().UTC()
	src.FetchedAt = &now
	if out != nil {
		if out.FinalURL != "" {
			fu := out.FinalURL
			src.FinalURL = &fu
		}
		if out.ContentType != "" {
			ct := out.ContentType
			src.ContentType = &ct
		}
	}
	if errors.Is(err, ErrTooLarge) {
		observeFetch("too_large")
	} else {
		observeFetch(status)
	}
	return ctrepo.UpsertLinkSource(ctx, pool, *src)
}

// IngestAsync runs IngestSource in a background goroutine (cold path NFR).
func IngestAsync(pool *pgxpool.Pool, sourceID uuid.UUID, policy IngestPolicy) {
	if pool == nil {
		return
	}
	go func() {
		ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 30*time.Second)
		defer cancel()
		_, _ = IngestSource(ctx, pool, sourceID, policy)
	}()
}
