package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"circular-board/internal/domain"
	"circular-board/internal/repository"
	"circular-board/internal/service"
)

// ─── テスト用サービスの初期化ヘルパー ────────────────────────
func newAuthService() *service.AuthService {
	userRepo := repository.NewUserRepository(testPool)
	tokenRepo := repository.NewRefreshTokenRepository(testPool)
	return service.NewAuthService(userRepo, tokenRepo, testConfig())
}

// ═══════════════════════════════════════════════════════════
// Login
// ═══════════════════════════════════════════════════════════

// TestAuthService_Login_Success_User
// 一般ユーザーが自治会コード＋メール＋パスワードでログインできる
func TestAuthService_Login_Success_User(t *testing.T) {
	assocID := insertTestAssociation(t, "ログインテスト自治会", "SVC_LOGIN_1")
	_ = insertTestUser(t, &assocID, "ログインユーザー", "login@svc-1.test", "secure!pass", "user")

	svc := newAuthService()
	result, err := svc.Login(context.Background(), "SVC_LOGIN_1", "login@svc-1.test", "secure!pass")

	require.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
	assert.Equal(t, int64(900), result.ExpiresIn) // 15 * 60 = 900秒
	assert.Equal(t, "login@svc-1.test", result.User.Email)
	assert.Equal(t, "user", string(result.User.Role))
}

// TestAuthService_Login_Success_AssociationAdmin
// 自治会管理者がログインできる
func TestAuthService_Login_Success_AssociationAdmin(t *testing.T) {
	assocID := insertTestAssociation(t, "管理者ログイン自治会", "SVC_LOGIN_ADM")
	_ = insertTestUser(t, &assocID, "管理者", "admin@svc-adm.test", "adminPass!", "association_admin")

	svc := newAuthService()
	result, err := svc.Login(context.Background(), "SVC_LOGIN_ADM", "admin@svc-adm.test", "adminPass!")

	require.NoError(t, err)
	assert.Equal(t, "association_admin", string(result.User.Role))
}

// TestAuthService_Login_Success_SystemAdmin_EmptyCode
// system_admin は自治会コードを空にしてログインできる
func TestAuthService_Login_Success_SystemAdmin_EmptyCode(t *testing.T) {
	_ = insertTestUser(t, nil, "システム管理者", "sysadmin@svc.test", "sysPass!", "system_admin")

	svc := newAuthService()
	result, err := svc.Login(context.Background(), "", "sysadmin@svc.test", "sysPass!")

	require.NoError(t, err)
	assert.Equal(t, "system_admin", string(result.User.Role))
	assert.Nil(t, result.User.AssociationID, "system_admin はassociation_idがnil")
}

// TestAuthService_Login_Failure_WrongPassword
// パスワード不一致では ErrInvalidCredentials が返る
func TestAuthService_Login_Failure_WrongPassword(t *testing.T) {
	assocID := insertTestAssociation(t, "パスワード不一致自治会", "SVC_WRONGPW")
	_ = insertTestUser(t, &assocID, "パスワードユーザー", "pw@svc-wrongpw.test", "correctPass", "user")

	svc := newAuthService()
	_, err := svc.Login(context.Background(), "SVC_WRONGPW", "pw@svc-wrongpw.test", "wrongPass")

	assert.True(t, errors.Is(err, service.ErrInvalidCredentials))
}

// TestAuthService_Login_Failure_UserNotFound
// 存在しないユーザーでは ErrInvalidCredentials が返る（ErrNotFound を公開しない）
func TestAuthService_Login_Failure_UserNotFound(t *testing.T) {
	assocID := insertTestAssociation(t, "存在しない自治会", "SVC_NOUSER")
	_ = assocID

	svc := newAuthService()
	_, err := svc.Login(context.Background(), "SVC_NOUSER", "nobody@svc-nouser.test", "anyPass")

	assert.True(t, errors.Is(err, service.ErrInvalidCredentials),
		"存在しないユーザーでも ErrInvalidCredentials を返しユーザー存在の有無を漏らさない")
}

// TestAuthService_Login_Failure_WrongAssociationCode
// ユーザーは存在するが自治会コードが違う場合は ErrInvalidCredentials
func TestAuthService_Login_Failure_WrongAssociationCode(t *testing.T) {
	assocID := insertTestAssociation(t, "正しい自治会", "SVC_RIGHTASSOC")
	_ = insertTestAssociation(t, "別の自治会", "SVC_WRONGASSOC")
	_ = insertTestUser(t, &assocID, "テナントユーザー", "tenant@svc-right.test", "pass123", "user")

	svc := newAuthService()
	_, err := svc.Login(context.Background(), "SVC_WRONGASSOC", "tenant@svc-right.test", "pass123")

	assert.True(t, errors.Is(err, service.ErrInvalidCredentials),
		"別の自治会コードでのログインは拒否される")
}

// ═══════════════════════════════════════════════════════════
// JWT トークン発行・検証
// ═══════════════════════════════════════════════════════════

// TestAuthService_ValidateAccessToken_Valid
// ログインで発行したトークンが正しく検証できる
func TestAuthService_ValidateAccessToken_Valid(t *testing.T) {
	assocID := insertTestAssociation(t, "JWT検証自治会", "SVC_JWT_1")
	userID := insertTestUser(t, &assocID, "JWTユーザー", "jwt@svc-jwt.test", "pass123", "user")

	svc := newAuthService()
	result, err := svc.Login(context.Background(), "SVC_JWT_1", "jwt@svc-jwt.test", "pass123")
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(result.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, assocID.String(), claims.AssociationID)
	assert.Equal(t, "user", claims.Role)
}

// TestAuthService_ValidateAccessToken_Expired
// 期限切れのトークンは ErrInvalidToken が返る
func TestAuthService_ValidateAccessToken_Expired(t *testing.T) {
	cfg := testConfig()
	svc := newAuthService()

	expiredToken := createExpiredAccessToken(t, cfg, uuid.New().String(), uuid.New().String(), "user")
	_, err := svc.ValidateAccessToken(expiredToken)

	assert.True(t, errors.Is(err, service.ErrInvalidToken),
		"期限切れトークンは ErrInvalidToken が返るべき")
}

// TestAuthService_ValidateAccessToken_Tampered
// 署名を改ざんしたトークンは ErrInvalidToken が返る
func TestAuthService_ValidateAccessToken_Tampered(t *testing.T) {
	svc := newAuthService()
	tamperedToken := "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoiZmFrZSJ9.invalid-signature"
	_, err := svc.ValidateAccessToken(tamperedToken)

	assert.True(t, errors.Is(err, service.ErrInvalidToken))
}

// TestAuthService_ValidateAccessToken_WrongSecret
// 異なるシークレットで署名されたトークンは ErrInvalidToken が返る
func TestAuthService_ValidateAccessToken_WrongSecret(t *testing.T) {
	// 別シークレットで署名
	claims := service.Claims{
		UserID: uuid.New().String(),
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	wrongToken, err := token.SignedString([]byte("completely-different-secret"))
	require.NoError(t, err)

	svc := newAuthService()
	_, validateErr := svc.ValidateAccessToken(wrongToken)

	assert.True(t, errors.Is(validateErr, service.ErrInvalidToken))
}

// ═══════════════════════════════════════════════════════════
// RefreshToken
// ═══════════════════════════════════════════════════════════

// TestAuthService_RefreshToken_Success
// 有効なリフレッシュトークンで新しいトークンペアを取得できる
func TestAuthService_RefreshToken_Success(t *testing.T) {
	assocID := insertTestAssociation(t, "リフレッシュ自治会", "SVC_REFRESH_1")
	_ = insertTestUser(t, &assocID, "リフレッシュユーザー", "refresh@svc-refresh.test", "pass123", "user")

	svc := newAuthService()
	first, err := svc.Login(context.Background(), "SVC_REFRESH_1", "refresh@svc-refresh.test", "pass123")
	require.NoError(t, err)

	second, err := svc.RefreshToken(context.Background(), first.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, second.AccessToken)
	assert.NotEmpty(t, second.RefreshToken)
	// リフレッシュトークンのローテーション確認（新旧で異なる値）
	assert.NotEqual(t, first.RefreshToken, second.RefreshToken,
		"リフレッシュ後は新しいトークンが発行される（ローテーション）")
	// 古いリフレッシュトークンは使えない
	_, err = svc.RefreshToken(context.Background(), first.RefreshToken)
	assert.True(t, errors.Is(err, service.ErrInvalidToken),
		"使用済みリフレッシュトークンは無効になる")
}

// TestAuthService_RefreshToken_InvalidToken
// 存在しないリフレッシュトークンでは ErrInvalidToken が返る
func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	svc := newAuthService()
	_, err := svc.RefreshToken(context.Background(), "nonexistent-refresh-token")

	assert.True(t, errors.Is(err, service.ErrInvalidToken))
}

// TestAuthService_RefreshToken_ExpiredToken
// 期限切れのリフレッシュトークンでは ErrInvalidToken が返る
func TestAuthService_RefreshToken_ExpiredToken(t *testing.T) {
	assocID := insertTestAssociation(t, "期限切れリフレッシュ自治会", "SVC_REFEXP")
	userID := insertTestUser(t, &assocID, "期限切れユーザー", "exp@svc-refexp.test", "pass123", "user")

	rawToken := "expired-refresh-" + uuid.New().String()
	insertExpiredRefreshToken(t, userID, rawToken)

	svc := newAuthService()
	_, err := svc.RefreshToken(context.Background(), rawToken)

	assert.True(t, errors.Is(err, service.ErrInvalidToken),
		"期限切れリフレッシュトークンは ErrInvalidToken が返るべき")
}

// ═══════════════════════════════════════════════════════════
// Logout
// ═══════════════════════════════════════════════════════════

// TestAuthService_Logout_ClearsAllTokens
// ログアウトするとそのユーザーの全リフレッシュトークンが無効になる
func TestAuthService_Logout_ClearsAllTokens(t *testing.T) {
	assocID := insertTestAssociation(t, "ログアウト自治会", "SVC_LOGOUT")
	_ = insertTestUser(t, &assocID, "ログアウトユーザー", "logout@svc-logout.test", "pass123", "user")

	svc := newAuthService()

	// 2回ログインして2つのリフレッシュトークンを作成
	r1, err := svc.Login(context.Background(), "SVC_LOGOUT", "logout@svc-logout.test", "pass123")
	require.NoError(t, err)
	r2, err := svc.Login(context.Background(), "SVC_LOGOUT", "logout@svc-logout.test", "pass123")
	require.NoError(t, err)

	// JWTクレームからユーザーIDを取得してログアウト
	claims, err := svc.ValidateAccessToken(r1.AccessToken)
	require.NoError(t, err)
	userID, err := uuid.Parse(claims.UserID)
	require.NoError(t, err)

	err = svc.Logout(context.Background(), userID)
	require.NoError(t, err)

	// 両方のリフレッシュトークンが無効になる
	_, errR1 := svc.RefreshToken(context.Background(), r1.RefreshToken)
	_, errR2 := svc.RefreshToken(context.Background(), r2.RefreshToken)
	assert.True(t, errors.Is(errR1, service.ErrInvalidToken))
	assert.True(t, errors.Is(errR2, service.ErrInvalidToken))
}

// ═══════════════════════════════════════════════════════════
// ロールによるアクセス制御
// ═══════════════════════════════════════════════════════════

// TestRequireRole_SystemAdmin_AllowsAll
// system_admin はすべてのロール要件を通過する
func TestRequireRole_SystemAdmin_AllowsAll(t *testing.T) {
	claims := &service.Claims{UserID: uuid.New().String(), Role: "system_admin"}

	assert.True(t, service.RequireRole(claims, domain.RoleUser))
	assert.True(t, service.RequireRole(claims, domain.RoleAssociationAdmin))
	assert.True(t, service.RequireRole(claims, domain.RoleSystemAdmin))
	assert.True(t, service.RequireRole(claims, domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
}

// TestRequireRole_AssociationAdmin
// association_admin は association_admin 以上が必要なルートを通過できるが user 単独での要件は通過できない
func TestRequireRole_AssociationAdmin(t *testing.T) {
	claims := &service.Claims{UserID: uuid.New().String(), Role: "association_admin"}

	assert.True(t, service.RequireRole(claims, domain.RoleAssociationAdmin))
	assert.True(t, service.RequireRole(claims, domain.RoleUser, domain.RoleAssociationAdmin))
	assert.True(t, service.RequireRole(claims, domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
	assert.False(t, service.RequireRole(claims, domain.RoleSystemAdmin),
		"association_admin は system_admin 専用ルートを通過できない")
}

// TestRequireRole_User
// user は user のみが許可されたルートを通過できるが管理者専用ルートは通過できない
func TestRequireRole_User(t *testing.T) {
	claims := &service.Claims{UserID: uuid.New().String(), Role: "user"}

	assert.True(t, service.RequireRole(claims, domain.RoleUser))
	assert.False(t, service.RequireRole(claims, domain.RoleAssociationAdmin))
	assert.False(t, service.RequireRole(claims, domain.RoleSystemAdmin))
	assert.False(t, service.RequireRole(claims, domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
}

// TestRequireRole_EmptyAllowed
// 許可リストが空の場合は全ロールが拒否される
func TestRequireRole_EmptyAllowed(t *testing.T) {
	claims := &service.Claims{UserID: uuid.New().String(), Role: "system_admin"}
	assert.False(t, service.RequireRole(claims))
}
