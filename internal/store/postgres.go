package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx database/sql driver for migrations
	"github.com/prenansantana/waid/internal/model"
)

// PostgresStore is a Store backed by a PostgreSQL database pool.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgres connects to the PostgreSQL instance at databaseURL, runs
// migrations, and returns a ready PostgresStore.
func NewPostgres(databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: connect postgres: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping postgres: %w", err)
	}

	sqlDB, err := pgxpoolToSQLDB(databaseURL)
	if err != nil {
		pool.Close()
		return nil, err
	}
	defer sqlDB.Close()

	if err := RunMigrations(sqlDB); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresStore{pool: pool}, nil
}

// pgxpoolToSQLDB opens a standard database/sql handle for running migrations.
func pgxpoolToSQLDB(url string) (*sql.DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("store: open pgx stdlib db: %w", err)
	}
	return db, nil
}

// Ping verifies the database connection is alive.
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Close closes the connection pool.
func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

// Create inserts a new Contact.
func (s *PostgresStore) Create(ctx context.Context, c *model.Contact) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	meta := metaJSON(c.Metadata)

	_, err := s.pool.Exec(ctx,
		`INSERT INTO contacts
		 (id, phone, bsuid, external_id, name, metadata, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		c.ID, c.Phone, c.BSUID, c.ExternalID, c.Name, meta, c.Status, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("store: create contact: %w", err)
	}
	return nil
}

// Update overwrites all mutable fields of an existing Contact.
func (s *PostgresStore) Update(ctx context.Context, c *model.Contact) error {
	c.UpdatedAt = time.Now().UTC()
	meta := metaJSON(c.Metadata)

	tag, err := s.pool.Exec(ctx,
		`UPDATE contacts
		 SET phone=$1, bsuid=$2, external_id=$3, name=$4, metadata=$5, status=$6, updated_at=$7
		 WHERE id=$8 AND deleted_at IS NULL`,
		c.Phone, c.BSUID, c.ExternalID, c.Name, meta, c.Status, c.UpdatedAt, c.ID,
	)
	if err != nil {
		return fmt.Errorf("store: update contact: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store: update contact %q: not found", c.ID)
	}
	return nil
}

// Delete soft-deletes a Contact.
func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE contacts SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("store: delete contact: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store: delete contact %q: not found", id)
	}
	return nil
}

// FindByPhone returns the active Contact with the given phone number.
func (s *PostgresStore) FindByPhone(ctx context.Context, phone string) (*model.Contact, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+pgContactColumns+` FROM contacts WHERE phone=$1 AND deleted_at IS NULL LIMIT 1`, phone,
	)
	return pgScanContact(row)
}

// FindByBSUID returns the active Contact with the given BSUID.
func (s *PostgresStore) FindByBSUID(ctx context.Context, bsuid string) (*model.Contact, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+pgContactColumns+` FROM contacts WHERE bsuid=$1 AND deleted_at IS NULL LIMIT 1`, bsuid,
	)
	return pgScanContact(row)
}

// FindByExternalID returns the active Contact with the given external_id.
func (s *PostgresStore) FindByExternalID(ctx context.Context, extID string) (*model.Contact, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+pgContactColumns+` FROM contacts WHERE external_id=$1 AND deleted_at IS NULL LIMIT 1`, extID,
	)
	return pgScanContact(row)
}

// FindByID returns the active Contact with the given primary key.
func (s *PostgresStore) FindByID(ctx context.Context, id string) (*model.Contact, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+pgContactColumns+` FROM contacts WHERE id=$1 AND deleted_at IS NULL LIMIT 1`, id,
	)
	return pgScanContact(row)
}

// List returns a page of active Contacts with optional search and total count.
func (s *PostgresStore) List(ctx context.Context, opts ListOpts) ([]model.Contact, int, error) {
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
		like := "%" + opts.Query + "%"
		where = " AND (phone LIKE $1 OR name LIKE $1 OR external_id LIKE $1)"
		args = append(args, like)
	}

	var total int
	countArgs := append([]any{}, args...)
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contacts WHERE deleted_at IS NULL`+where,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: list count: %w", err)
	}

	nextPlaceholder := len(args) + 1
	listArgs := append(args, opts.PerPage, offset)
	rows, err := s.pool.Query(ctx,
		fmt.Sprintf(`SELECT `+pgContactColumns+` FROM contacts WHERE deleted_at IS NULL`+where+
			` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, nextPlaceholder, nextPlaceholder+1),
		listArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list query: %w", err)
	}
	defer rows.Close()

	var contacts []model.Contact
	for rows.Next() {
		c, err := pgScanContactRow(rows)
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
func (s *PostgresStore) BulkUpsert(ctx context.Context, contacts []model.Contact) (*model.ImportReport, error) {
	report := &model.ImportReport{Total: len(contacts)}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: bulk upsert begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

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

		tag, err := tx.Exec(ctx,
			`INSERT INTO contacts (id, phone, bsuid, external_id, name, metadata, status, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			 ON CONFLICT(phone) DO UPDATE SET
			   bsuid       = EXCLUDED.bsuid,
			   external_id = EXCLUDED.external_id,
			   name        = EXCLUDED.name,
			   metadata    = EXCLUDED.metadata,
			   status      = EXCLUDED.status,
			   updated_at  = EXCLUDED.updated_at`,
			c.ID, c.Phone, c.BSUID, c.ExternalID, c.Name, meta, c.Status, c.CreatedAt, c.UpdatedAt,
		)
		if err != nil {
			report.Errors++
			report.Details = append(report.Details, model.ImportError{Row: i, Phone: c.Phone, Reason: err.Error()})
			continue
		}
		if tag.RowsAffected() == 1 {
			report.Created++
		} else {
			report.Updated++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("store: bulk upsert commit: %w", err)
	}
	return report, nil
}

// ---- helpers ----------------------------------------------------------------

const pgContactColumns = `id, phone, bsuid, external_id, name, metadata, status,
	created_at, updated_at, deleted_at`

func pgScanContact(row pgx.Row) (*model.Contact, error) {
	c, err := pgScanContactRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("store: not found: %w", err)
	}
	return c, err
}

type pgxScanner interface {
	Scan(dest ...any) error
}

func pgScanContactRow(row pgxScanner) (*model.Contact, error) {
	var (
		c         model.Contact
		bsuid     *string
		extID     *string
		meta      *string
		deletedAt *time.Time
	)
	if err := row.Scan(
		&c.ID, &c.Phone, &bsuid, &extID, &c.Name, &meta, &c.Status,
		&c.CreatedAt, &c.UpdatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}
	c.BSUID = bsuid
	c.ExternalID = extID
	c.DeletedAt = deletedAt
	if meta != nil && *meta != "" {
		c.Metadata = json.RawMessage(*meta)
	}
	return &c, nil
}

