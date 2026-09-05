package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("utente non trovato")
	ErrDuplicateUser = errors.New("email o username già in uso")
	ErrInvalidToken  = errors.New("token non valido, scaduto o già usato")
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

func (r *Repository) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&n); err != nil {
		return 0, fmt.Errorf("user: count: %w", err)
	}
	return n, nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (email/username already taken) — translated to ErrDuplicateUser
// so callers don't need to know about pgx/Postgres error codes.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// CreateActiveUser creates a user with a password already set (used only
// for the bootstrap admin — every other account starts pending, see
// CreatePendingUser).
func (r *Repository) CreateActiveUser(ctx context.Context, email, username, passwordHash string, isAdmin bool) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, is_admin)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, email, username, is_admin, created_at`,
		email, username, passwordHash, isAdmin,
	).Scan(&u.ID, &u.Email, &u.Username, &u.IsAdmin, &u.CreatedAt)
	if isUniqueViolation(err) {
		return User{}, ErrDuplicateUser
	}
	if err != nil {
		return User{}, fmt.Errorf("user: create active: %w", err)
	}
	return u, nil
}

// CreatePendingUser creates a user with no password yet: it's set when the
// invite token generated alongside it is redeemed (see CreateInviteToken,
// ConsumeInviteToken).
func (r *Repository) CreatePendingUser(ctx context.Context, email, username string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, username)
		VALUES ($1, $2)
		RETURNING id::text, email, username, is_admin, created_at`,
		email, username,
	).Scan(&u.ID, &u.Email, &u.Username, &u.IsAdmin, &u.CreatedAt)
	if isUniqueViolation(err) {
		return User{}, ErrDuplicateUser
	}
	if err != nil {
		return User{}, fmt.Errorf("user: create pending: %w", err)
	}
	return u, nil
}

// GetByLogin looks a user up by email or username (either is accepted at
// the login form) and returns their password hash for verification —
// separate from User itself so the hash never accidentally ends up in a
// JSON response.
func (r *Repository) GetByLogin(ctx context.Context, identifier string) (User, string, error) {
	var u User
	var passwordHash *string
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, email, username, is_admin, created_at, password_hash
		FROM users WHERE email = $1 OR username = $1`, identifier,
	).Scan(&u.ID, &u.Email, &u.Username, &u.IsAdmin, &u.CreatedAt, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	if err != nil {
		return User{}, "", fmt.Errorf("user: get by login: %w", err)
	}
	if passwordHash == nil {
		// Account created but the invite was never redeemed yet.
		return User{}, "", ErrNotFound
	}
	roles, err := r.GetRolesForUser(ctx, u.ID)
	if err != nil {
		return User{}, "", err
	}
	u.Roles = roles
	return u, *passwordHash, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, email, username, is_admin, created_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Username, &u.IsAdmin, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("user: get by id: %w", err)
	}
	roles, err := r.GetRolesForUser(ctx, u.ID)
	if err != nil {
		return User{}, err
	}
	u.Roles = roles
	return u, nil
}

func (r *Repository) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	var hash *string
	err := r.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) || hash == nil {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("user: get password hash: %w", err)
	}
	return *hash, nil
}

func (r *Repository) SetPassword(ctx context.Context, userID, passwordHash string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("user: set password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpsertRole is called at startup for every role in the seed file (see
// docs/adr/0012) — idempotent, so restarting the service after a fork edits
// its roles config applies the change without a full schema reset.
func (r *Repository) UpsertRole(ctx context.Context, slug, displayName string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO roles (slug, display_name) VALUES ($1, $2)
		ON CONFLICT (slug) DO UPDATE SET display_name = EXCLUDED.display_name`,
		slug, displayName)
	if err != nil {
		return fmt.Errorf("user: upsert role: %w", err)
	}
	return nil
}

// RoleIDsBySlug resolves role slugs to ids, so the caller can catch an
// unknown slug (e.g. a typo in an admin's request) before assigning it.
func (r *Repository) RoleIDsBySlug(ctx context.Context, slugs []string) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text, slug FROM roles WHERE slug = ANY($1)`, slugs)
	if err != nil {
		return nil, fmt.Errorf("user: role ids by slug: %w", err)
	}
	defer rows.Close()

	result := map[string]string{}
	for rows.Next() {
		var id, slug string
		if err := rows.Scan(&id, &slug); err != nil {
			return nil, fmt.Errorf("user: role ids by slug: scan: %w", err)
		}
		result[slug] = id
	}
	return result, rows.Err()
}

func (r *Repository) AssignRoles(ctx context.Context, userID string, roleIDs []string) error {
	for _, roleID := range roleIDs {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, userID, roleID)
		if err != nil {
			return fmt.Errorf("user: assign role: %w", err)
		}
	}
	return nil
}

func (r *Repository) GetRolesForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT roles.slug FROM roles
		JOIN user_roles ON user_roles.role_id = roles.id
		WHERE user_roles.user_id = $1
		ORDER BY roles.slug`, userID)
	if err != nil {
		return nil, fmt.Errorf("user: roles for user: %w", err)
	}
	defer rows.Close()

	slugs := []string{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("user: roles for user: scan: %w", err)
		}
		slugs = append(slugs, slug)
	}
	return slugs, rows.Err()
}

func (r *Repository) CreateInviteToken(ctx context.Context, userID, purpose string, ttl time.Duration) (string, error) {
	raw, hash, err := GenerateToken()
	if err != nil {
		return "", err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO invite_tokens (user_id, purpose, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`,
		userID, purpose, hash, time.Now().Add(ttl))
	if err != nil {
		return "", fmt.Errorf("user: create invite token: %w", err)
	}
	return raw, nil
}

// ConsumeInviteToken validates raw against a stored, unused, unexpired
// token and marks it used in the same statement — a token can only ever be
// redeemed once, even under concurrent requests.
func (r *Repository) ConsumeInviteToken(ctx context.Context, raw, purpose string) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx, `
		UPDATE invite_tokens SET used_at = now()
		WHERE token_hash = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
		RETURNING user_id::text`,
		HashToken(raw), purpose,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidToken
	}
	if err != nil {
		return "", fmt.Errorf("user: consume invite token: %w", err)
	}
	return userID, nil
}
