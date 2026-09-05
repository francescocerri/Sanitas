package turno

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Repository) List(ctx context.Context) ([]Turno, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, volontario_id, data, ora_inizio, ora_fine, stato
		FROM turni ORDER BY data, ora_inizio`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []Turno{}
	for rows.Next() {
		var t Turno
		if err := rows.Scan(&t.ID, &t.VolontarioID, &t.Data, &t.OraInizio, &t.OraFine, &t.Stato); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *Repository) Create(ctx context.Context, t Turno) (Turno, error) {
	var created Turno
	err := r.pool.QueryRow(ctx, `
		INSERT INTO turni (volontario_id, data, ora_inizio, ora_fine)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, volontario_id, data, ora_inizio, ora_fine, stato`,
		t.VolontarioID, t.Data, t.OraInizio, t.OraFine,
	).Scan(&created.ID, &created.VolontarioID, &created.Data, &created.OraInizio, &created.OraFine, &created.Stato)
	return created, err
}

func (r *Repository) Get(ctx context.Context, id string) (Turno, error) {
	var t Turno
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, volontario_id, data, ora_inizio, ora_fine, stato
		FROM turni WHERE id = $1`, id,
	).Scan(&t.ID, &t.VolontarioID, &t.Data, &t.OraInizio, &t.OraFine, &t.Stato)
	return t, err
}
