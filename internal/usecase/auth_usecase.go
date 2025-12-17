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
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtSecret        []byte
	logger           *zap.Logger
}

func NewAuthUseCase(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtSecret []byte,
	logger *zap.Logger,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtSecret:        []byte(jwtSecret),
		logger:           logger,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	User         *entity.User `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
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

	// Generate refresh token
	refreshToken, err := uc.generateRefreshToken(ctx, user)
	if err != nil {
		uc.logger.Error("Failed to generate refresh token", zap.Error(err), zap.String("user_id", user.ID))
		return nil, err
	}

	uc.logger.Info("User registered successfully", zap.String("user_id", user.ID), zap.String("email", user.Email))

	return &AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
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

	refreshToken, err := uc.generateRefreshToken(ctx, user)
	if err != nil {
		uc.logger.Error("Failed to generate refresh token", zap.Error(err), zap.String("user_id", user.ID))
		return nil, err
	}

	uc.logger.Info("Login success", zap.String("email", req.Email))

	return &AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (uc *AuthUseCase) generateToken(user *entity.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"type":    "access",
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(uc.jwtSecret)
}

func (uc *AuthUseCase) generateRefreshToken(ctx context.Context, user *entity.User) (string, error) {
	tokenStr := uuid.New().String()

	refreshToken := &entity.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 30),
		CreatedAt: time.Now(),
		IsRevoked: false,
	}

	if err := uc.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return "", err
	}

	return tokenStr, nil
}

func (uc *AuthUseCase) RefreshAccessToken(ctx context.Context, refreshTokenStr string) (*AuthResponse, error) {
	uc.logger.Info("Refresh token attempt")

	refreshToken, err := uc.refreshTokenRepo.GetByToken(ctx, refreshTokenStr)
	if err != nil {
		uc.logger.Warn("Refresh token is revoked", zap.String("token_id", refreshToken.ID))
		return nil, errors.New("refresh token has been revoked")
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		uc.logger.Warn("Refresh tplem expired", zap.String("token_id", refreshToken.ID))
		return nil, errors.New("refresh token has expired")
	}

	user, err := uc.userRepo.GetById(ctx, refreshToken.UserID)
	if err != nil {
		uc.logger.Error("User not found for refresh token", zap.Error(err))
		return nil, errors.New("user not found")
	}

	accessToken, err := uc.generateToken(user)
	if err != nil {
		uc.logger.Error("Failed to generate new access token", zap.Error(err))
		return nil, err
	}

	newRefreshToken, err := uc.generateRefreshToken(ctx, user)
	if err != nil {
		uc.logger.Error("Failed to generate new refresh token", zap.Error(err))
		return nil, err
	}

	// Revoke
	if err := uc.refreshTokenRepo.RevokeToken(ctx, refreshTokenStr); err != nil {
		uc.logger.Warn("Failed to revoke old refresh token", zap.Error(err))
		return nil, err
	}

	return &AuthResponse{
		Token:        accessToken,
		RefreshToken: newRefreshToken,
		User:         user,
	}, nil
}

func (uc *AuthUseCase) Logout(ctx context.Context, refreshTokenStr string) error {
	uc.logger.Info("Logout attempt")

	if err := uc.refreshTokenRepo.RevokeToken(ctx, refreshTokenStr); err != nil {
		uc.logger.Error("Failed to revoke refresh token", zap.Error(err))
		return err
	}

	uc.logger.Info("Logout successfully")

	return nil
}

func (uc *AuthUseCase) LogoutAllDevices(ctx context.Context, userId string) error {
	uc.logger.Info("Logout from all devices", zap.String("user_id", userId))

	if err := uc.refreshTokenRepo.RevokeAllUserTokens(ctx, userId); err != nil {
		uc.logger.Error("Failed to revoke all tokens", zap.Error(err))
		return err
	}

	uc.logger.Info("All devices logged out successfully", zap.String("user_id", userId))

	return nil
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
