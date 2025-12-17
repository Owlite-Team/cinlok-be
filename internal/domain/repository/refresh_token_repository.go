package repository

import (
	"cinlok-be/internal/domain/entity"
	"context"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *entity.RefreshToken) error
	GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error)
	RevokeToken(ctx context.Context, token string) error
	RevokeAllUserTokens(ctx context.Context, userId string) error
	DeleteExpiredTokens(ctx context.Context) error
}
