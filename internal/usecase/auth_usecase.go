package usecase

import (
	"cinlok-be/internal/domain/entity"
	"cinlok-be/internal/domain/repository"
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthUseCase struct {
	userRepo  repository.UserRepository
	jwtSecret []byte
	logger    *zap.Logger
}

func NewAuthUseCase(
	userRepo repository.UserRepository,
	jwtSecret []byte,
	logger *zap.Logger,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		logger:    logger,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *entity.User `json:"user"`
}

func (uc *AuthUseCase) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	uc.logger.Info("User registration attemp", zap.String("email", req.Email))

	// Validate
	if req.Email == "" || req.Password == "" || req.Name == "" {
		uc.logger.Warn("Registration validation failed", zap.String("email", req.Email))
		return nil, errors.New("email, password, and name are required")
	}
	if len(req.Password) < 8 {
		uc.logger.Warn("Password too short", zap.String("email", req.Email))
		return nil, errors.New("password must be at least 8 characters")
	}

	existingUser, _ := uc.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		uc.logger.Warn("User already exist", zap.String("email", req.Email))
		return nil, errors.New("user with this email already exist")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		uc.logger.Error("Failed to hash password", zap.Error(err))
		return nil, err
	}

	// Create user
	user := &entity.User{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Password:  string(hashedPassword),
		Name:      req.Name,
		Phone:     req.Phone,
		Role:      string(entity.RoleBuyer),
		Avatar:    req.Avatar,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := uc.userRepo.Create(ctx, user); err != nil {
		uc.logger.Error("Failed to create user", zap.Error(err))
	}

	// Generate token
	token, err := uc.generateToken(user)
	if err != nil {
		uc.logger.Error("Failed to generate token", zap.Error(err), zap.String("user_id", user.ID))
		return nil, err
	}

	uc.logger.Info("User registered successfully", zap.String("user_id", user.ID), zap.String("email", user.Email))

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	uc.logger.Info("Login attempt", zap.String("email", req.Email))

	if req.Email == "" || req.Password == "" {
		uc.logger.Warn("Login validation failed", zap.String("email", req.Email))
		return nil, errors.New("email and password are required")
	}

	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		uc.logger.Warn("User not found", zap.String("email", req.Email))
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		uc.logger.Warn("Login failed, wrong email or password", zap.String("email", req.Email))
		return nil, errors.New("invalid credentials")
	}

	token, err := uc.generateToken(user)
	if err != nil {
		uc.logger.Error("Failed to generate token", zap.String("email", req.Email))
		return nil, errors.New("invalid credential")
	}

	uc.logger.Info("Login success", zap.String("email", req.Email))

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (uc *AuthUseCase) generateToken(user *entity.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(uc.jwtSecret)
}

func (uc *AuthUseCase) ValidateToken(tokenString string) (*entity.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return uc.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return &entity.User{
		ID:    claims["user_id"].(string),
		Email: claims["email"].(string),
		Role:  claims["role"].(string),
	}, nil
}
