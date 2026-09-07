package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrDuplicateUser = errors.New("email or username already in use")
	ErrInvalidToken  = errors.New("invalid, expired, or already used token")
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("user: ping: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

func (r *Repository) CountUsers(ctx context.Context) (int, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&User{}).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("user: count: %w", err)
	}
	return int(n), nil
}

// isUniqueViolation reports whether err is a unique-constraint violation
// (email/username already taken). Relies on gorm.Config.TranslateError
// (set in cmd/server/main.go) mapping the underlying Postgres error to
// GORM's own portable error — no driver-specific type here.
func isUniqueViolation(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}

// CreateActiveUser creates a user with a password already set (used only
// for the bootstrap admin — every other account starts pending, see
// CreatePendingUser). No admin flag: the bootstrap admin's ability to
// manage accounts comes entirely from the technical role Bootstrap assigns
// it afterwards, same mechanism as any other user — see docs/adr/0018.
func (r *Repository) CreateActiveUser(ctx context.Context, email, username, passwordHash string) (User, error) {
	u := User{Email: email, Username: username, PasswordHash: &passwordHash}
	if err := r.db.WithContext(ctx).Create(&u).Error; err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrDuplicateUser
		}
		return User{}, fmt.Errorf("user: create active: %w", err)
	}
	return u, nil
}

// CreatePendingUser creates a user with no password yet: it's set when the
// invite token generated alongside it is redeemed (see CreateToken,
// ConsumeToken).
func (r *Repository) CreatePendingUser(ctx context.Context, email, username string) (User, error) {
	u := User{Email: email, Username: username}
	if err := r.db.WithContext(ctx).Create(&u).Error; err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrDuplicateUser
		}
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
	err := r.db.WithContext(ctx).
		Where("email = ? OR username = ?", identifier, identifier).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, "", ErrNotFound
	}
	if err != nil {
		return User{}, "", fmt.Errorf("user: get by login: %w", err)
	}
	if u.PasswordHash == nil {
		// Account created but the invite was never redeemed yet.
		return User{}, "", ErrNotFound
	}
	if err := r.populateRolesAndPermissions(ctx, &u); err != nil {
		return User{}, "", err
	}
	return u, *u.PasswordHash, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("user: get by id: %w", err)
	}
	if err := r.populateRolesAndPermissions(ctx, &u); err != nil {
		return User{}, err
	}
	return u, nil
}

func (r *Repository) populateRolesAndPermissions(ctx context.Context, u *User) error {
	roles, err := r.GetRolesForUser(ctx, u.ID)
	if err != nil {
		return err
	}
	u.Roles = roles
	permissions, err := r.GetPermissionsForUser(ctx, u.ID)
	if err != nil {
		return err
	}
	u.Permissions = permissions
	return nil
}

func (r *Repository) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	var u User
	err := r.db.WithContext(ctx).Select("password_hash").Where("id = ?", userID).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || u.PasswordHash == nil {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("user: get password hash: %w", err)
	}
	return *u.PasswordHash, nil
}

func (r *Repository) SetPassword(ctx context.Context, userID, passwordHash string) error {
	result := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("password_hash", passwordHash)
	if result.Error != nil {
		return fmt.Errorf("user: set password: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// UpsertRole is called at startup for every role in the seed file (see
// docs/adr/0012) — idempotent, so restarting the service after a fork edits
// its roles config applies the change without a full schema reset. Returns
// the role's id: Bootstrap needs it to assign the technical admin role
// (see docs/adr/0018), nobody else currently does. Raw SQL (not GORM's
// OnConflict clause builder): an explicit, easy to audit atomic upsert
// mirroring exactly what ran under pgx before — see docs/adr/0019.
func (r *Repository) UpsertRole(ctx context.Context, slug, displayName string, permissions []string) (string, error) {
	if permissions == nil {
		// A nil Go slice would otherwise become SQL NULL — the column is
		// NOT NULL (no "permissions" in a seed entry means "grants
		// nothing", not "unset").
		permissions = []string{}
	}
	var id string
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO roles (slug, display_name, permissions) VALUES (?, ?, ?)
		ON CONFLICT (slug) DO UPDATE SET display_name = EXCLUDED.display_name, permissions = EXCLUDED.permissions
		RETURNING id`,
		slug, displayName, StringArray(permissions)).Scan(&id).Error
	if err != nil {
		return "", fmt.Errorf("user: upsert role: %w", err)
	}
	return id, nil
}

// RoleIDsBySlug resolves role slugs to ids, so the caller can catch an
// unknown slug (e.g. a typo in an admin's request) before assigning it.
func (r *Repository) RoleIDsBySlug(ctx context.Context, slugs []string) (map[string]string, error) {
	var roles []Role
	if err := r.db.WithContext(ctx).Select("id", "slug").Where("slug IN ?", slugs).Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("user: role ids by slug: %w", err)
	}
	result := make(map[string]string, len(roles))
	for _, role := range roles {
		result[role.Slug] = role.ID
	}
	return result, nil
}

// ListRoles returns the database catalog in a stable order for role selectors.
func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	roles := make([]Role, 0)
	if err := r.db.WithContext(ctx).Order("display_name").Order("slug").Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("user: list roles: %w", err)
	}
	return roles, nil
}

// ListUsers returns every user with Roles/Permissions populated — no
// pagination, stessa scelta già fatta per ListRoles/shifts.List: la scala
// di un comitato non lo giustifica ancora. Roles/Permissions non sono
// colonne GORM (vedi User), quindi vanno popolate riga per riga come già fa
// GetByID — un round-trip in più per utente, accettabile per un elenco
// pensato per un pannello di amministrazione, non per un hot path.
func (r *Repository) ListUsers(ctx context.Context) ([]User, error) {
	users := make([]User, 0)
	if err := r.db.WithContext(ctx).Order("username").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("user: list users: %w", err)
	}
	for i := range users {
		if err := r.populateRolesAndPermissions(ctx, &users[i]); err != nil {
			return nil, err
		}
	}
	return users, nil
}

func (r *Repository) AssignRoles(ctx context.Context, userID string, roleIDs []string) error {
	for _, roleID := range roleIDs {
		err := r.db.WithContext(ctx).Exec(`
			INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)
			ON CONFLICT DO NOTHING`, userID, roleID).Error
		if err != nil {
			return fmt.Errorf("user: assign role: %w", err)
		}
	}
	return nil
}

// ReplaceRoles sostituisce l'intero insieme di ruoli di un utente — a
// differenza di AssignRoles, solo additiva (mai una DELETE), usata alla
// creazione dove non c'è nulla da rimuovere. In una transazione perché uno
// stato intermedio "nessun ruolo" visibile da un'altra richiesta (es. un
// controllo di permesso in corso) sarebbe peggio di un errore che annulla
// tutto.
func (r *Repository) ReplaceRoles(ctx context.Context, userID string, roleIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM user_roles WHERE user_id = ?`, userID).Error; err != nil {
			return fmt.Errorf("user: replace roles: delete: %w", err)
		}
		for _, roleID := range roleIDs {
			if err := tx.Exec(`
				INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)
				ON CONFLICT DO NOTHING`, userID, roleID).Error; err != nil {
				return fmt.Errorf("user: replace roles: insert: %w", err)
			}
		}
		return nil
	})
}

func (r *Repository) GetRolesForUser(ctx context.Context, userID string) ([]string, error) {
	slugs := []string{}
	err := r.db.WithContext(ctx).
		Table("roles").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Order("roles.slug").
		Pluck("roles.slug", &slugs).Error
	if err != nil {
		return nil, fmt.Errorf("user: roles for user: %w", err)
	}
	return slugs, nil
}

// GetPermissionsForUser unions the permissions of every role assigned to
// the user, deduplicated — a permission granted by more than one of a
// user's roles still shows up once. See docs/adr/0018.
func (r *Repository) GetPermissionsForUser(ctx context.Context, userID string) ([]string, error) {
	permissions := []string{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT permission
		FROM user_roles
		JOIN roles ON roles.id = user_roles.role_id
		CROSS JOIN LATERAL unnest(roles.permissions) AS permission
		WHERE user_roles.user_id = ?
		ORDER BY permission`, userID).Scan(&permissions).Error
	if err != nil {
		return nil, fmt.Errorf("user: permissions for user: %w", err)
	}
	return permissions, nil
}

// CreateToken creates a single-use token for the given purpose ("invite",
// "refresh", ... — see the tokens table, docs/adr/0016) and returns the raw
// value; only its hash is persisted.
func (r *Repository) CreateToken(ctx context.Context, userID, purpose string, ttl time.Duration) (string, error) {
	raw, hash, err := GenerateToken()
	if err != nil {
		return "", err
	}
	token := Token{UserID: userID, Purpose: purpose, TokenHash: hash, ExpiresAt: time.Now().Add(ttl)}
	if err := r.db.WithContext(ctx).Create(&token).Error; err != nil {
		return "", fmt.Errorf("user: create token: %w", err)
	}
	return raw, nil
}

// ConsumeToken validates raw against a stored, unused, unexpired token for
// the given purpose and marks it used in the same statement — a token can
// only ever be redeemed once, even under concurrent requests. Raw SQL: this
// atomicity (check + mark-used in one round trip) is exactly what a plain
// UPDATE...RETURNING already gives for free.
func (r *Repository) ConsumeToken(ctx context.Context, raw, purpose string) (string, error) {
	var userID string
	result := r.db.WithContext(ctx).Raw(`
		UPDATE tokens SET used_at = now()
		WHERE token_hash = ? AND purpose = ? AND used_at IS NULL AND expires_at > now()
		RETURNING user_id`,
		HashToken(raw), purpose).Scan(&userID)
	if result.Error != nil {
		return "", fmt.Errorf("user: consume token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return "", ErrInvalidToken
	}
	return userID, nil
}
