package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/prenansantana/waid/internal/model"
	_ "modernc.org/sqlite"
)

// SQLiteStore is a Store backed by a SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLite opens (or creates) the SQLite database at dbPath, enables WAL
// and foreign keys, runs migrations, and returns a ready SQLiteStore.
func NewSQLite(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: open sqlite %q: %w", dbPath, err)
	}

	// Enable WAL mode and foreign keys.
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("store: set pragma %q: %w", p, err)
		}
	}

	if err := RunMigrations(db, "sqlite"); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// Ping verifies the database connection is alive.
func (s *SQLiteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Create inserts a new Contact. If c.ID is empty a new UUID is generated.
func (s *SQLiteStore) Create(ctx context.Context, c *model.Contact) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	meta := metaJSON(c.Metadata)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO contacts
		 (id, phone, bsuid, external_id, name, metadata, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Phone, c.BSUID, c.ExternalID, c.Name, meta, c.Status,
		c.CreatedAt.Format(time.RFC3339Nano), c.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("store: create contact: %w", err)
	}
	return nil
}

// Update overwrites all mutable fields of an existing Contact.
func (s *SQLiteStore) Update(ctx context.Context, c *model.Contact) error {
	c.UpdatedAt = time.Now().UTC()
	meta := metaJSON(c.Metadata)

	res, err := s.db.ExecContext(ctx,
		`UPDATE contacts
		 SET phone=?, bsuid=?, external_id=?, name=?, metadata=?, status=?, updated_at=?
		 WHERE id=? AND deleted_at IS NULL`,
		c.Phone, c.BSUID, c.ExternalID, c.Name, meta, c.Status,
		c.UpdatedAt.Format(time.RFC3339Nano), c.ID,
	)
	if err != nil {
		return fmt.Errorf("store: update contact: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: update contact %q: not found", c.ID)
	}
	return nil
}

// Delete soft-deletes a Contact by setting deleted_at.
func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx,
		`UPDATE contacts SET deleted_at=? WHERE id=? AND deleted_at IS NULL`, now, id,
	)
	if err != nil {
		return fmt.Errorf("store: delete contact: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: delete contact %q: not found", id)
	}
	return nil
}

// FindByPhone returns the active Contact with the given phone number.
func (s *SQLiteStore) FindByPhone(ctx context.Context, phone string) (*model.Contact, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+contactColumns+` FROM contacts WHERE phone=? AND deleted_at IS NULL LIMIT 1`, phone,
	)
	return scanContact(row)
}

// FindByBSUID returns the active Contact with the given BSUID.
func (s *SQLiteStore) FindByBSUID(ctx context.Context, bsuid string) (*model.Contact, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+contactColumns+` FROM contacts WHERE bsuid=? AND deleted_at IS NULL LIMIT 1`, bsuid,
	)
	return scanContact(row)
}

// FindByExternalID returns the active Contact with the given external_id.
func (s *SQLiteStore) FindByExternalID(ctx context.Context, extID string) (*model.Contact, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+contactColumns+` FROM contacts WHERE external_id=? AND deleted_at IS NULL LIMIT 1`, extID,
	)
	return scanContact(row)
}

// FindByID returns the active Contact with the given primary key.
func (s *SQLiteStore) FindByID(ctx context.Context, id string) (*model.Contact, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+contactColumns+` FROM contacts WHERE id=? AND deleted_at IS NULL LIMIT 1`, id,
	)
	return scanContact(row)
}

// List returns a page of active Contacts optionally filtered by a LIKE query
// on phone, name, or external_id. It also returns the total matching count.
func (s *SQLiteStore) List(ctx context.Context, opts ListOpts) ([]model.Contact, int, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PerPage < 1 {
		opts.PerPage = 20
	}
	offset := (opts.Page - 1) * opts.PerPage

	var (
		where string
		args  []any
	)
	if opts.Query != "" {
		like := "%" + escapeLike(opts.Query) + "%"
		where = ` AND (phone LIKE ? ESCAPE '\' OR name LIKE ? ESCAPE '\' OR external_id LIKE ? ESCAPE '\')`
		args = append(args, like, like, like)
	}

	// Total count.
	var total int
	countArgs := append([]any{}, args...)
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM contacts WHERE deleted_at IS NULL`+where,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: list count: %w", err)
	}

	// Paginated rows.
	listArgs := append(args, opts.PerPage, offset)
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+contactColumns+` FROM contacts WHERE deleted_at IS NULL`+where+
			` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		listArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list query: %w", err)
	}
	defer rows.Close()

	var contacts []model.Contact
	for rows.Next() {
		c, err := scanContactRow(rows)
		if err != nil {
			return nil, 0, err
		}
		contacts = append(contacts, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: list rows: %w", err)
	}
	return contacts, total, nil
}

// BulkUpsert inserts or updates a slice of Contacts in a single transaction.
// Conflicts on phone are resolved by updating the existing row.
func (s *SQLiteStore) BulkUpsert(ctx context.Context, contacts []model.Contact) (*model.ImportReport, error) {
	report := &model.ImportReport{Total: len(contacts)}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: bulk upsert begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	existsStmt, err := tx.PrepareContext(ctx,
		`SELECT COUNT(*) FROM contacts WHERE phone=? AND deleted_at IS NULL`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: bulk upsert prepare exists: %w", err)
	}
	defer existsStmt.Close()

	upsertStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO contacts (id, phone, bsuid, external_id, name, metadata, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(phone) DO UPDATE SET
		   bsuid       = excluded.bsuid,
		   external_id = excluded.external_id,
		   name        = excluded.name,
		   metadata    = excluded.metadata,
		   status      = excluded.status,
		   updated_at  = excluded.updated_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: bulk upsert prepare: %w", err)
	}
	defer upsertStmt.Close()

	now := time.Now().UTC()
	for i := range contacts {
		c := &contacts[i]
		if c.ID == "" {
			c.ID = uuid.New().String()
		}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = now
		}
		c.UpdatedAt = now
		meta := metaJSON(c.Metadata)

		// Check whether the row already exists to distinguish insert from update.
		var existing int
		if err := existsStmt.QueryRowContext(ctx, c.Phone).Scan(&existing); err != nil {
			report.Errors++
			report.Details = append(report.Details, model.ImportError{Row: i, Phone: c.Phone, Reason: err.Error()})
			continue
		}

		_, err := upsertStmt.ExecContext(ctx,
			c.ID, c.Phone, c.BSUID, c.ExternalID, c.Name, meta, c.Status,
			c.CreatedAt.Format(time.RFC3339Nano), c.UpdatedAt.Format(time.RFC3339Nano),
		)
		if err != nil {
			report.Errors++
			report.Details = append(report.Details, model.ImportError{Row: i, Phone: c.Phone, Reason: err.Error()})
			continue
		}
		if existing > 0 {
			report.Updated++
		} else {
			report.Created++
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: bulk upsert commit: %w", err)
	}
	return report, nil
}

// ---- helpers ----------------------------------------------------------------

const contactColumns = `id, phone, bsuid, external_id, name, metadata, status,
	created_at, updated_at, deleted_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanContact(row *sql.Row) (*model.Contact, error) {
	c, err := scanContactRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func scanContactRow(row rowScanner) (*model.Contact, error) {
	var (
		c         model.Contact
		bsuid     sql.NullString
		extID     sql.NullString
		meta      sql.NullString
		createdAt string
		updatedAt string
		deletedAt sql.NullString
	)
	if err := row.Scan(
		&c.ID, &c.Phone, &bsuid, &extID, &c.Name, &meta, &c.Status,
		&createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}
	if bsuid.Valid {
		c.BSUID = &bsuid.String
	}
	if extID.Valid {
		c.ExternalID = &extID.String
	}
	if meta.Valid && meta.String != "" {
		c.Metadata = json.RawMessage(meta.String)
	}
	var err error
	if c.CreatedAt, err = ParseTime(createdAt); err != nil {
		return nil, fmt.Errorf("store: parse created_at: %w", err)
	}
	if c.UpdatedAt, err = ParseTime(updatedAt); err != nil {
		return nil, fmt.Errorf("store: parse updated_at: %w", err)
	}
	if deletedAt.Valid && deletedAt.String != "" {
		t, err := ParseTime(deletedAt.String)
		if err != nil {
			return nil, fmt.Errorf("store: parse deleted_at: %w", err)
		}
		c.DeletedAt = &t
	}
	return &c, nil
}

// metaJSON returns the JSON representation of meta, defaulting to "{}".
func metaJSON(meta json.RawMessage) string {
	if len(meta) == 0 {
		return "{}"
	}
	return string(meta)
}

// escapeLike escapes LIKE pattern special characters (%, _, \) with a backslash.
func escapeLike(s string) string {
	var buf []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '%', '_':
			buf = append(buf, '\\', s[i])
		default:
			buf = append(buf, s[i])
		}
	}
	if buf == nil {
		return s
	}
	return string(buf)
}
