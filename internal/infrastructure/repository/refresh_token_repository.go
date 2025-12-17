package repository

import (
	"cinlok-be/internal/domain/entity"
	"context"
	"database/sql"
)

type RefreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		db: db,
	}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, is_revoked)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query, token.ID, token.UserID, token.Token, token.ExpiresAt, token.CreatedAt, token.IsRevoked)

	return err
}

func (r *RefreshTokenRepository) GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, is_revoked
		FROM refresh_tokens
		WHERE token = $1
	`

	refreshToken := &entity.RefreshToken{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&refreshToken.ID,
		&refreshToken.UserID,
		&refreshToken.Token,
		&refreshToken.ExpiresAt,
		&refreshToken.CreatedAt,
		&refreshToken.IsRevoked,
	)
	if err != nil {
		return nil, err
	}

	return refreshToken, nil
}

func (r *RefreshTokenRepository) RevokeToken(ctx context.Context, token string) error {
	query := `UPDATE refresh_tokens SET is_revoked = true WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)

	return err
}

func (r *RefreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userId string) error {
	query := `UPDATE refresh_tokens SET is_revoked = true WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userId)

	return err
}

func (r *RefreshTokenRepository) DeleteExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < NOW() OR is_revoked = true`
	_, err := r.db.ExecContext(ctx, query)

	return err
}
