package notification

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/diegoczajka/travelu-api/internal/platform/database"
)

// MySQLRepository implements Repository on top of a sqlx MySQL pool or, when
// built via WithTx, on top of an in-flight transaction.
type MySQLRepository struct {
	db database.SQLExecutor
}

// NewMySQLRepository builds a MySQLRepository backed by db.
func NewMySQLRepository(db *sqlx.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

// WithTx returns a MySQLRepository whose operations run against tx instead
// of the pool, so they participate in a transaction started by
// database.WithTx.
func (repo *MySQLRepository) WithTx(tx *sqlx.Tx) *MySQLRepository {
	return &MySQLRepository{db: tx}
}

// notificationRow mirrors the notifications table layout; mapping to/from
// the domain Notification happens entirely in this file.
type notificationRow struct {
	ID          string         `db:"id"`
	UserID      string         `db:"user_id"`
	Title       string         `db:"title"`
	Body        string         `db:"body"`
	Type        string         `db:"type"`
	ReferenceID sql.NullString `db:"reference_id"`
	IsRead      bool           `db:"is_read"`
	CreatedAt   time.Time      `db:"created_at"`
}

func (r notificationRow) toDomain() Notification {
	return Notification{
		ID:          r.ID,
		UserID:      r.UserID,
		Title:       r.Title,
		Body:        r.Body,
		Type:        r.Type,
		ReferenceID: r.ReferenceID.String,
		IsRead:      r.IsRead,
		CreatedAt:   r.CreatedAt,
	}
}

// nullable turns an empty string into a SQL NULL.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (repo *MySQLRepository) Create(ctx context.Context, n Notification) (Notification, error) {
	const query = `
		INSERT INTO notifications (id, user_id, title, body, type, reference_id, is_read, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	n.ID = uuid.NewString()

	_, err := repo.db.ExecContext(ctx, query,
		n.ID, n.UserID, n.Title, n.Body, n.Type, nullable(n.ReferenceID), n.IsRead, n.CreatedAt)
	if err != nil {
		return Notification{}, err
	}

	return n, nil
}

func (repo *MySQLRepository) GetByID(ctx context.Context, id string) (Notification, error) {
	const query = `
		SELECT id, user_id, title, body, type, reference_id, is_read, created_at
		FROM notifications
		WHERE id = ?`

	var row notificationRow
	if err := repo.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Notification{}, ErrNotFound
		}
		return Notification{}, err
	}

	return row.toDomain(), nil
}

func (repo *MySQLRepository) ListByUserID(ctx context.Context, userID string) ([]Notification, error) {
	const query = `
		SELECT id, user_id, title, body, type, reference_id, is_read, created_at
		FROM notifications
		WHERE user_id = ?
		ORDER BY created_at DESC`

	var rows []notificationRow
	if err := repo.db.SelectContext(ctx, &rows, query, userID); err != nil {
		return nil, err
	}

	notifications := make([]Notification, len(rows))
	for i, row := range rows {
		notifications[i] = row.toDomain()
	}
	return notifications, nil
}

func (repo *MySQLRepository) MarkRead(ctx context.Context, id string) error {
	const query = `UPDATE notifications SET is_read = TRUE WHERE id = ?`

	result, err := repo.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}
