package repository

import (
	"cinlok-be/internal/domain/entity"
	"context"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetAll(ctx context.Context) ([]*entity.User, error)
	GetById(ctx context.Context, id string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id string) error
}
