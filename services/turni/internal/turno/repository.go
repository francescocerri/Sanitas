package turno

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is the domain error for "no matching row" — pgx.ErrNoRows is
// translated here, so callers above this layer (HTTP) don't need to know
// about the Postgres driver to tell a 404 apart from a real error.
var ErrNotFound = errors.New("turno non trovato")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// id::text everywhere below: pgx's default type mapping doesn't scan a uuid
// column straight into a Go string, so we let Postgres cast it to text
// instead of teaching this layer about pgx's uuid/pgtype machinery.
func (r *Repository) List(ctx context.Context) ([]Turno, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, volontario_id, data, ora_inizio, ora_fine, stato
		FROM turni ORDER BY data, ora_inizio`)
	if err != nil {
		return nil, fmt.Errorf("turno: list: %w", err)
	}
	defer rows.Close()

	// []Turno{}, not a nil slice: encodes to `[]` in JSON, not `null`.
	result := []Turno{}
	for rows.Next() {
		var t Turno
		if err := rows.Scan(&t.ID, &t.VolontarioID, &t.Data, &t.OraInizio, &t.OraFine, &t.Stato); err != nil {
			return nil, fmt.Errorf("turno: list: scan: %w", err)
		}
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("turno: list: %w", err)
	}
	return result, nil
}

func (r *Repository) Create(ctx context.Context, t Turno) (Turno, error) {
	var created Turno
	err := r.pool.QueryRow(ctx, `
		INSERT INTO turni (volontario_id, data, ora_inizio, ora_fine)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, volontario_id, data, ora_inizio, ora_fine, stato`,
		t.VolontarioID, t.Data, t.OraInizio, t.OraFine,
	).Scan(&created.ID, &created.VolontarioID, &created.Data, &created.OraInizio, &created.OraFine, &created.Stato)
	if err != nil {
		return Turno{}, fmt.Errorf("turno: create: %w", err)
	}
	return created, nil
}

func (r *Repository) Get(ctx context.Context, id string) (Turno, error) {
	var t Turno
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, volontario_id, data, ora_inizio, ora_fine, stato
		FROM turni WHERE id = $1`, id,
	).Scan(&t.ID, &t.VolontarioID, &t.Data, &t.OraInizio, &t.OraFine, &t.Stato)
	if errors.Is(err, pgx.ErrNoRows) {
		return Turno{}, ErrNotFound
	}
	if err != nil {
		return Turno{}, fmt.Errorf("turno: get: %w", err)
	}
	return t, nil
}
