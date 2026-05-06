package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"circular-board/internal/config"
	"circular-board/internal/domain"
	"circular-board/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUserNotFound       = errors.New("user not found")
)

type Claims struct {
	UserID        string `json:"user_id"`
	AssociationID string `json:"association_id"`
	Role          string `json:"role"`
	jwt.RegisteredClaims
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         *repository.UserWithAssociation
}

type AuthService struct {
	userRepo    *repository.UserRepository
	tokenRepo   *repository.RefreshTokenRepository
	cfg         *config.Config
}

func NewAuthService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.RefreshTokenRepository,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		cfg:       cfg,
	}
}

func (s *AuthService) Login(ctx context.Context, associationCode, email, password string) (*LoginResult, error) {
	var (
		userWithAssoc *repository.UserWithAssociation
		err           error
	)

	// system_admin can log in with empty or any association code
	if associationCode == "" {
		userWithAssoc, err = s.userRepo.FindByEmailForSystemAdmin(ctx, email)
	} else {
		userWithAssoc, err = s.userRepo.FindByEmailAndAssociationCode(ctx, email, associationCode)
		if errors.Is(err, repository.ErrNotFound) && associationCode != "" {
			// Try system_admin fallback
			userWithAssoc, err = s.userRepo.FindByEmailForSystemAdmin(ctx, email)
		}
	}

	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userWithAssoc.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.generateAccessToken(userWithAssoc)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	rawRefreshToken, err := generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	tokenHash := hashToken(rawRefreshToken)
	expiresAt := time.Now().Add(s.cfg.RefreshTokenExpiry)

	if err := s.tokenRepo.Create(ctx, userWithAssoc.ID, tokenHash, expiresAt); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(s.cfg.AccessTokenExpiry.Seconds()),
		User:         userWithAssoc,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, rawRefreshToken string) (*LoginResult, error) {
	tokenHash := hashToken(rawRefreshToken)

	rt, err := s.tokenRepo.FindByTokenHash(ctx, tokenHash)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("find refresh token: %w", err)
	}

	userWithAssoc, err := s.userRepo.FindByID(ctx, rt.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	// Rotate refresh token
	if err := s.tokenRepo.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("delete old refresh token: %w", err)
	}

	accessToken, err := s.generateAccessToken(userWithAssoc)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	newRawRefreshToken, err := generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	newTokenHash := hashToken(newRawRefreshToken)
	expiresAt := time.Now().Add(s.cfg.RefreshTokenExpiry)

	if err := s.tokenRepo.Create(ctx, userWithAssoc.ID, newTokenHash, expiresAt); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: newRawRefreshToken,
		ExpiresIn:    int64(s.cfg.AccessTokenExpiry.Seconds()),
		User:         userWithAssoc,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	return s.tokenRepo.DeleteByUserID(ctx, userID)
}

func (s *AuthService) GetMe(ctx context.Context, userID uuid.UUID) (*repository.UserWithAssociation, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	return u, err
}

func (s *AuthService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (s *AuthService) generateAccessToken(u *repository.UserWithAssociation) (string, error) {
	assocID := ""
	if u.AssociationID != nil {
		assocID = u.AssociationID.String()
	}

	claims := Claims{
		UserID:        u.ID.String(),
		AssociationID: assocID,
		Role:          string(u.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   u.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func generateRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// HashPassword はパスワードのbcryptハッシュを生成する（シーダーから利用）
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// AssociationIDFromClaims はJWTクレームからassociation_idを取り出す
func AssociationIDFromClaims(claims *Claims) (*uuid.UUID, error) {
	if claims.AssociationID == "" {
		return nil, nil
	}
	id, err := uuid.Parse(claims.AssociationID)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// RequireRole はロールによるアクセス制御ヘルパー
func RequireRole(claims *Claims, allowedRoles ...domain.Role) bool {
	for _, r := range allowedRoles {
		if string(r) == claims.Role {
			return true
		}
	}
	return false
}

// ─── コンテキストキー（middleware と permission が共用） ────────────────

type claimsContextKey struct{}

// SetClaimsInContext はJWTクレームをコンテキストに保存する
func SetClaimsInContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// ClaimsFromContext はコンテキストからJWTクレームを取り出す
func ClaimsFromContext(ctx context.Context) *Claims {
	v := ctx.Value(claimsContextKey{})
	if v == nil {
		return nil
	}
	c, _ := v.(*Claims)
	return c
}
