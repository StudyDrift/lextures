// Package library provides data access for the school library catalog and reading log (plan 13.8).
package library

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Book is a library catalog entry for an org (school).
type Book struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	Title       string
	Author      *string
	ISBN        *string
	CoverURL    *string
	LexileLevel *int
	FPBand      *string
	GradeBand   *string
	Summary     *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ReadingLogEntry is a single student reading log record.
type ReadingLogEntry struct {
	ID         uuid.UUID
	StudentID  uuid.UUID
	BookID     *uuid.UUID
	BookTitle  *string
	LogDate    time.Time
	PagesRead  *int
	Reflection *string
	LoggedAt   time.Time
}

// ListBooksFilter controls catalog search.
type ListBooksFilter struct {
	LexileMin *int
	LexileMax *int
	GradeBand *string
}

// ListBooks returns catalog books for an org, with optional filters.
func ListBooks(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, f ListBooksFilter) ([]Book, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, title, author, isbn, cover_url, lexile_level, fp_band, grade_band, summary, created_at, updated_at
FROM library.books
WHERE org_id = $1
  AND ($2::int IS NULL OR lexile_level >= $2)
  AND ($3::int IS NULL OR lexile_level <= $3)
  AND ($4::text IS NULL OR grade_band = $4)
ORDER BY title ASC
`, orgID, f.LexileMin, f.LexileMax, f.GradeBand)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBooks(rows)
}

// GetBook returns a single book by ID scoped to an org.
func GetBook(ctx context.Context, pool *pgxpool.Pool, orgID, bookID uuid.UUID) (*Book, error) {
	var b Book
	err := pool.QueryRow(ctx, `
SELECT id, org_id, title, author, isbn, cover_url, lexile_level, fp_band, grade_band, summary, created_at, updated_at
FROM library.books
WHERE id = $1 AND org_id = $2
`, bookID, orgID).Scan(
		&b.ID, &b.OrgID, &b.Title, &b.Author, &b.ISBN, &b.CoverURL,
		&b.LexileLevel, &b.FPBand, &b.GradeBand, &b.Summary,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// CreateBookParams holds fields for a new library book.
type CreateBookParams struct {
	OrgID       uuid.UUID
	Title       string
	Author      *string
	ISBN        *string
	CoverURL    *string
	LexileLevel *int
	FPBand      *string
	GradeBand   *string
	Summary     *string
}

// CreateBook inserts a new book into the library catalog.
func CreateBook(ctx context.Context, pool *pgxpool.Pool, p CreateBookParams) (*Book, error) {
	var b Book
	err := pool.QueryRow(ctx, `
INSERT INTO library.books (org_id, title, author, isbn, cover_url, lexile_level, fp_band, grade_band, summary)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, org_id, title, author, isbn, cover_url, lexile_level, fp_band, grade_band, summary, created_at, updated_at
`, p.OrgID, p.Title, p.Author, p.ISBN, p.CoverURL, p.LexileLevel, p.FPBand, p.GradeBand, p.Summary).Scan(
		&b.ID, &b.OrgID, &b.Title, &b.Author, &b.ISBN, &b.CoverURL,
		&b.LexileLevel, &b.FPBand, &b.GradeBand, &b.Summary,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// DeleteBook removes a book from the library catalog.
func DeleteBook(ctx context.Context, pool *pgxpool.Pool, orgID, bookID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM library.books WHERE id = $1 AND org_id = $2`, bookID, orgID)
	return err
}

// CreateReadingLogEntry inserts a new reading log entry for a student.
func CreateReadingLogEntry(ctx context.Context, pool *pgxpool.Pool,
	studentID uuid.UUID, bookID *uuid.UUID, bookTitle *string,
	logDate time.Time, pagesRead *int, reflection *string,
) (*ReadingLogEntry, error) {
	var e ReadingLogEntry
	err := pool.QueryRow(ctx, `
INSERT INTO library.reading_log_entries
    (student_id, book_id, book_title, log_date, pages_read, reflection)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, student_id, book_id, book_title, log_date, pages_read, reflection, logged_at
`, studentID, bookID, bookTitle, logDate, pagesRead, reflection).Scan(
		&e.ID, &e.StudentID, &e.BookID, &e.BookTitle, &e.LogDate,
		&e.PagesRead, &e.Reflection, &e.LoggedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListReadingLogEntries returns reading log entries for a student, most recent first.
func ListReadingLogEntries(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, limit int) ([]ReadingLogEntry, error) {
	rows, err := pool.Query(ctx, `
SELECT id, student_id, book_id, book_title, log_date, pages_read, reflection, logged_at
FROM library.reading_log_entries
WHERE student_id = $1
ORDER BY log_date DESC, logged_at DESC
LIMIT $2
`, studentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

// DashboardRow is an aggregated reading summary row for one student.
type DashboardRow struct {
	StudentID    uuid.UUID
	DisplayName  *string
	Email        string
	WeeklyPages  int
	TotalEntries int
	TotalPages   int
}

// ReadingDashboard returns an aggregated reading summary for all enrolled students in a course.
// Weekly totals cover the 7 days ending today (UTC).
func ReadingDashboard(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]DashboardRow, error) {
	rows, err := pool.Query(ctx, `
SELECT
    u.id            AS student_id,
    u.display_name,
    u.email,
    COALESCE(SUM(CASE WHEN e.log_date >= CURRENT_DATE - INTERVAL '6 days' THEN COALESCE(e.pages_read, 0) END), 0) AS weekly_pages,
    COUNT(e.id)     AS total_entries,
    COALESCE(SUM(COALESCE(e.pages_read, 0)), 0)                                                                   AS total_pages
FROM course.course_enrollments ce
JOIN "user".users u ON u.id = ce.user_id
LEFT JOIN library.reading_log_entries e ON e.student_id = u.id
WHERE ce.course_id = $1
  AND ce.role = 'student'
  AND ce.active
GROUP BY u.id, u.display_name, u.email
ORDER BY u.email ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DashboardRow
	for rows.Next() {
		var d DashboardRow
		if err := rows.Scan(&d.StudentID, &d.DisplayName, &d.Email,
			&d.WeeklyPages, &d.TotalEntries, &d.TotalPages); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func scanBooks(rows pgx.Rows) ([]Book, error) {
	var out []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(
			&b.ID, &b.OrgID, &b.Title, &b.Author, &b.ISBN, &b.CoverURL,
			&b.LexileLevel, &b.FPBand, &b.GradeBand, &b.Summary,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func scanEntries(rows pgx.Rows) ([]ReadingLogEntry, error) {
	var out []ReadingLogEntry
	for rows.Next() {
		var e ReadingLogEntry
		if err := rows.Scan(
			&e.ID, &e.StudentID, &e.BookID, &e.BookTitle, &e.LogDate,
			&e.PagesRead, &e.Reflection, &e.LoggedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
