package contenttools

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

// ReconcileMarkdownFences validates ```lex-tool fences against course instances,
// strips unknown/cross-course fences, and archives active instances for the host
// item that are no longer referenced (plan CT.2 FR-12).
func ReconcileMarkdownFences(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	structureItemID *uuid.UUID,
	markdown string,
) (cleaned string, err error) {
	cleanedBodies, err := reconcileMarkdownBodies(ctx, pool, courseID, structureItemID, []string{markdown})
	if err != nil {
		return markdown, err
	}
	if len(cleanedBodies) == 0 {
		return "", nil
	}
	return cleanedBodies[0], nil
}

// ReconcileMarkdownBodies reconciles multiple markdown bodies that share one
// structure item (or syllabus when structureItemID is nil). Archive runs once
// against the union of referenced instance ids across all bodies.
func ReconcileMarkdownBodies(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	structureItemID *uuid.UUID,
	bodies []string,
) ([]string, error) {
	return reconcileMarkdownBodies(ctx, pool, courseID, structureItemID, bodies)
}

func reconcileMarkdownBodies(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	structureItemID *uuid.UUID,
	bodies []string,
) ([]string, error) {
	if pool == nil {
		return bodies, nil
	}
	out := make([]string, len(bodies))
	copy(out, bodies)

	var fenceIDs []uuid.UUID
	seen := map[uuid.UUID]struct{}{}
	for _, md := range bodies {
		for _, p := range ParseLexToolFences(md) {
			id, err := uuid.Parse(strings.TrimSpace(p.InstanceID))
			if err != nil {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			fenceIDs = append(fenceIDs, id)
		}
	}

	valid := map[string]bool{}
	var referenced []uuid.UUID
	if len(fenceIDs) > 0 {
		rows, err := ctrepo.GetInstancesByIDs(ctx, pool, courseID, fenceIDs)
		if err != nil {
			return bodies, err
		}
		for _, row := range rows {
			valid[row.ID.String()] = true
			referenced = append(referenced, row.ID)
		}
	}

	for i, md := range bodies {
		out[i] = StripInvalidLexToolFences(md, valid)
	}

	if err := ctrepo.ArchiveUnreferencedForItem(ctx, pool, courseID, structureItemID, referenced); err != nil {
		return out, err
	}
	return out, nil
}
