// Package h5ppackages provides DB access for content.h5p_packages (plan 8.12).
package h5ppackages

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Package is a row from content.h5p_packages.
type Package struct {
	ID              uuid.UUID
	StorageObjectID uuid.UUID
	StructureItemID *uuid.UUID
	CourseID        uuid.UUID
	Title           string
	ContentType     string
	H5PVersion      *string
	ManifestJSON    json.RawMessage
	AssetsPrefix    string
	ExtractStatus   string
	ExtractError    *string
	CreatedAt       time.Time
}

// Insert creates a new H5P package row with the given id.
func Insert(
	ctx context.Context,
	pool *pgxpool.Pool,
	id, storageObjectID, courseID uuid.UUID,
	structureItemID *uuid.UUID,
	title, contentType string,
	h5pVersion *string,
	manifest json.RawMessage,
	assetsPrefix string,
) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO content.h5p_packages
		  (id, storage_object_id, structure_item_id, course_id, title, content_type, h5p_version, manifest_json, assets_prefix)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, storageObjectID, structureItemID, courseID, title, contentType, h5pVersion, manifest, assetsPrefix,
	)
	return err
}

// LoadByID fetches a package by id scoped to course.
func LoadByID(ctx context.Context, pool *pgxpool.Pool, courseID, packageID uuid.UUID) (*Package, error) {
	return scanPackage(pool.QueryRow(ctx, `
		SELECT id, storage_object_id, structure_item_id, course_id, title, content_type, h5p_version,
		       manifest_json, assets_prefix, extract_status, extract_error, created_at
		FROM content.h5p_packages
		WHERE id = $1 AND course_id = $2`, packageID, courseID))
}

// LoadByStructureItem fetches the package for a module item.
func LoadByStructureItem(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (*Package, error) {
	return scanPackage(pool.QueryRow(ctx, `
		SELECT id, storage_object_id, structure_item_id, course_id, title, content_type, h5p_version,
		       manifest_json, assets_prefix, extract_status, extract_error, created_at
		FROM content.h5p_packages
		WHERE structure_item_id = $1 AND course_id = $2`, itemID, courseID))
}

// ClaimPendingForExtract returns one pending package id for background extraction.
func ClaimPendingForExtract(ctx context.Context, pool *pgxpool.Pool) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT id FROM content.h5p_packages
		WHERE extract_status = 'pending'
		ORDER BY created_at
		LIMIT 1
		FOR UPDATE SKIP LOCKED`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// LoadByIDGlobal loads by package id only (worker).
func LoadByIDGlobal(ctx context.Context, pool *pgxpool.Pool, packageID uuid.UUID) (*Package, error) {
	return scanPackage(pool.QueryRow(ctx, `
		SELECT id, storage_object_id, structure_item_id, course_id, title, content_type, h5p_version,
		       manifest_json, assets_prefix, extract_status, extract_error, created_at
		FROM content.h5p_packages WHERE id = $1`, packageID))
}

// MarkExtractReady sets extract_status to ready.
func MarkExtractReady(ctx context.Context, pool *pgxpool.Pool, packageID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE content.h5p_packages SET extract_status = 'ready', extract_error = NULL WHERE id = $1`, packageID)
	return err
}

// MarkExtractFailed records extraction failure.
func MarkExtractFailed(ctx context.Context, pool *pgxpool.Pool, packageID uuid.UUID, msg string) error {
	_, err := pool.Exec(ctx, `
		UPDATE content.h5p_packages SET extract_status = 'failed', extract_error = $2 WHERE id = $1`, packageID, msg)
	return err
}

// LinkStructureItem associates a package with a structure item after insert.
func LinkStructureItem(ctx context.Context, pool *pgxpool.Pool, packageID, itemID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE content.h5p_packages SET structure_item_id = $2 WHERE id = $1`, packageID, itemID)
	return err
}

func scanPackage(row pgx.Row) (*Package, error) {
	var p Package
	err := row.Scan(
		&p.ID, &p.StorageObjectID, &p.StructureItemID, &p.CourseID, &p.Title, &p.ContentType,
		&p.H5PVersion, &p.ManifestJSON, &p.AssetsPrefix, &p.ExtractStatus, &p.ExtractError, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
