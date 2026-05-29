# 10. テストコードプロンプト集

テストは Go の `testify` ライブラリ（v1.9.0）を使用し、
実際の PostgreSQL に接続して行うインテグレーションテストです。

---

## 共通セットアップ

```
以下のテスト共通セットアップパターンに従ってテストを実装してください。

【テスト構成】
各 internal/<package>/ 配下に以下の4ファイルを作成:
- setup_test.go    : DB接続・テストデータ作成ヘルパー
- repository_test.go : リポジトリ層のテスト
- service_test.go  : サービス層のテスト
- handler_test.go  : ハンドラー層のHTTPテスト

【setup_test.go の共通構造】
package <package>

import (
    "context"
    "net/http/httptest"
    "os"
    "testing"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/stretchr/testify/require"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        dsn = "postgres://postgres:postgres@localhost:5432/circular_board_test?sslmode=disable"
    }
    var err error
    testDB, err = pgxpool.New(context.Background(), dsn)
    if err != nil {
        panic(err)
    }
    defer testDB.Close()
    os.Exit(m.Run())
}

// テスト用ヘルパー関数例
func createTestAssociation(t *testing.T, name, code string) uuid.UUID {
    t.Helper()
    id := uuid.New()
    _, err := testDB.Exec(context.Background(),
        `INSERT INTO associations (id, name, code) VALUES ($1, $2, $3)`,
        id, name, code)
    require.NoError(t, err)
    t.Cleanup(func() {
        testDB.Exec(context.Background(), `DELETE FROM associations WHERE id = $1`, id)
    })
    return id
}

func createTestUser(t *testing.T, assocID *uuid.UUID, role string) uuid.UUID {
    t.Helper()
    id := uuid.New()
    _, err := testDB.Exec(context.Background(),
        `INSERT INTO users (id, association_id, name, email, password_hash, role)
         VALUES ($1, $2, $3, $4, $5, $6)`,
        id, assocID, "テストユーザー", id.String()+"@test.com",
        "$2a$10$...", role)
    require.NoError(t, err)
    t.Cleanup(func() {
        testDB.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
    })
    return id
}
```

---

## 01. 認証テスト

```
以下の仕様に従って認証機能のテストを実装してください。

【テストファイル】
backend/internal/auth/

【repository_test.go のテストケース】
- TestUserRepository_FindByEmailAndAssociationCode
  - 正常系: 存在するユーザーが取得できる
  - 異常系: 存在しないメールアドレスで ErrNotFound
  - 異常系: 誤った自治会コードで ErrNotFound

- TestRefreshTokenRepository_Create_FindByTokenHash
  - 正常系: トークンを保存して取得できる
  - 異常系: 期限切れトークンは取得できない

- TestPasswordResetRepository_Create_FindValidByTokenHash
  - 正常系: 有効なトークンが取得できる
  - 異常系: 期限切れは ErrTokenExpiredOrUsed
  - 異常系: 使用済みは ErrTokenExpiredOrUsed

【service_test.go のテストケース】
- TestAuthService_Login_Success
  - 正常系: 正しい認証情報でLoginResult が返る
  - 異常系: 誤ったパスワードで ErrInvalidCredentials

- TestAuthService_RefreshToken
  - 正常系: リフレッシュで新しいトークンペアが得られる
  - ローテーション確認: 旧トークンは無効になる

【handler_test.go のテストケース】
- TestLogin_Success: POST /api/v1/auth/login で 200 + トークン返却
- TestLogin_InvalidCredentials: 誤認証で 401
- TestLogin_MissingEmail: email 空で 400
- TestMe_Success: GET /api/v1/auth/me で 200 + ユーザー情報
- TestMe_Unauthorized: 認証なしで 401
- TestRefresh_Success: POST /api/v1/auth/refresh で 200
- TestLogout_Success: POST /api/v1/auth/logout で 200
- TestForgotPassword_Success: POST /forgot-password で 200（メール未送信でも）
- TestResetPassword_Success: 有効なトークンで 200
- TestResetPassword_ExpiredToken: 期限切れで 400

【テストの JWT 生成ヘルパー】
func generateTestJWT(t *testing.T, userID, assocID, role string) string {
    t.Helper()
    claims := service.Claims{
        UserID:        userID,
        AssociationID: assocID,
        Role:          role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenStr, err := token.SignedString([]byte("test-secret"))
    require.NoError(t, err)
    return tokenStr
}
```

---

## 02. 回覧物テスト

```
以下の仕様に従って回覧物機能のテストを実装してください。

【テストファイル】
backend/internal/files/

【setup_test.go の追加ヘルパー】
- withPermission(t, roleID, featureID, perms): デフォルト権限設定（association_id IS NULL）
- withTestFile(t, assocID, year, month): テスト用ファイルレコード作成

【repository_test.go のテストケース】
- TestFileRepository_Create_List
  - 正常系: ファイル作成 → 一覧取得
  - 年月フィルタ: year=2024, month=5 で絞り込み
  - 論理削除後は一覧に表示されない

- TestFileRepository_SoftDelete
  - 正常系: deleted_at がセットされる
  - 存在しないIDで ErrNotFound

- TestFileRepository_AvailableYears
  - 複数年のファイルがある場合、全年が返る

【handler_test.go のテストケース】
- TestUpload_Success: POST /files でファイルアップロード 201
- TestUpload_TooLarge: 10MB超で 413
- TestUpload_UnsupportedMIME: 対応外形式で 400
- TestList_Success: GET /files で一覧取得 200
- TestList_SystemAdmin: system_admin が association_id 指定で取得
- TestDownload_Success: GET /files/{id}/download で 200
- TestDelete_Success: DELETE /files/{id} で 200
- TestDelete_Forbidden: 他自治会のファイル削除で 404（データ隠蔽）
```

---

## 03. お知らせテスト

```
以下の仕様に従ってお知らせ機能のテストを実装してください。

【テストファイル】
backend/internal/notice/

【repository_test.go のテストケース】
- TestNoticeRepository_Create_List
  - ピン留め優先ソート確認（is_pinned DESC, created_at DESC）
  - is_read フラグが正しく付与される

- TestNoticeRepository_MarkAsRead
  - 既読後に is_read=true
  - 同一ユーザーで2回呼んでもエラーにならない（ON CONFLICT DO NOTHING）

- TestNoticeRepository_UnreadCount
  - 未読件数が正確に返る

【handler_test.go のテストケース】
- TestCreate_Success: POST /notices で 201
- TestCreate_MissingTitle: title 空で 400
- TestCreate_Forbidden: 自治会なしユーザーで 403
- TestList_Success: GET /notices で 200 + is_read フラグ付き
- TestGet_AutoRead: GET /notices/{id} で既読になる
- TestUnreadCount_Success: GET /notices/unread-count で 200
- TestDelete_Success: DELETE /notices/{id} で 200
- TestDelete_NotFound: 存在しないIDで 404
```

---

## 04. アンケートテスト

```
以下の仕様に従ってアンケート機能のテストを実装してください。

【テストファイル】
backend/internal/survey/

【service_test.go のテストケース】
- TestSurveyService_Create_Success
- TestSurveyService_Create_EmptyTitle: title 空でエラー
- TestSurveyService_Create_PastExpiresAt: 過去日時でエラー
- TestSurveyService_Create_NoQuestions: 質問なしでエラー
- TestSurveyService_Create_InsufficientChoices: 選択肢1件でエラー
- TestSurveyService_Answer_Success: 単一選択・複数選択
- TestSurveyService_Answer_Expired: 期限切れアンケートで ErrExpired
- TestSurveyService_GetResults: 集計結果（件数・割合）

【handler_test.go のテストケース】
- TestCreate_Success: POST /surveys で 201
- TestCreate_ValidationError: バリデーションエラーで 400
- TestList_Success: GET /surveys で 200
- TestGet_Success: GET /surveys/{id} で 200
- TestAnswer_Success: POST /surveys/{id}/answer で 200
- TestAnswer_Expired: 期限切れで 400 SURVEY_EXPIRED
- TestResults_Success: GET /surveys/{id}/results で 200
- TestUnansweredCount: GET /surveys/unanswered-count で 200
- TestDelete_Success: DELETE /surveys/{id} で 200
```

---

## 05. アカウント管理テスト

```
以下の仕様に従ってアカウント管理機能のテストを実装してください。

【テストファイル】
backend/internal/account/

【service_test.go のテストケース】
- TestAccountService_Create_Success（各ロールパターン）
- TestAccountService_Create_EmailConflict: 重複メールで ErrEmailConflict
- TestAccountService_Create_WeakPassword: 7文字以下で ErrWeakPassword
- TestAccountService_Create_InvalidRole: 不正ロールで ErrInvalidRole
- TestAccountService_Create_Forbidden: user_admin が association_admin 作成で ErrForbidden
- TestAccountService_Deactivate_SelfDeactivation: 自分自身で ErrSelfDeactivation

【handler_test.go のテストケース】
- TestList_AssociationAdmin_Success: 自治会内のみ取得
- TestList_SystemAdmin_AllAssociations: system_admin が全件取得
- TestList_SystemAdmin_FilterByAssociation: association_id フィルタ
- TestCreate_Success: POST /accounts で 201
- TestCreate_EmailConflict: 重複メールで 409
- TestUpdate_Success: PUT /accounts/{id} で 200
- TestDelete_CannotDeleteSelf: 自分自身の削除で 400
- TestActivate_Success: PUT /accounts/{id}/activate で 200
- TestDeactivate_Success: PUT /accounts/{id}/deactivate で 200
```

---

## 06. 権限管理テスト

```
以下の仕様に従って権限管理機能のテストを実装してください。

【テストファイル】
backend/internal/permission/

【setup_test.go の注意点】
- withPermission ヘルパーは association_id IS NULL の行を操作すること
- UPSERT: ON CONFLICT (role_id, feature_id) WHERE association_id IS NULL DO UPDATE SET ...

【repository_test.go のテストケース】
- TestGetAllPermissions_Default: デフォルト設定取得（assocID=nil）
- TestGetAllPermissions_WithAssociation: 自治会別設定取得（フォールバック確認）
- TestGetPermissionByRoleAndFeature_Default: assocID=nil で検索
- TestGetPermissionByRoleAndFeature_Fallback: 専用設定なしでデフォルト返却
- TestUpdatePermissions_Default: association_id IS NULL で UPSERT
- TestUpdatePermissions_Association: 特定自治会の設定 UPSERT
- TestIsCustomized: 専用設定あり→true、フォールバック→false

【service_test.go のテストケース】
- TestGetMatrix_Default: assocID=nil でデフォルト matrix
- TestGetMatrix_WithAssociation: 自治会別 matrix
- TestCheckPermission_Allowed: 権限ありでエラーなし
- TestCheckPermission_Denied: can_view=false で Forbidden
- TestCheckPermission_Fallback: 専用設定なしでデフォルト参照

【handler_test.go のテストケース】
- TestGetMatrix_Success: GET /permissions で 200
- TestGetMatrix_WithAssociationID: ?association_id=xxx で 200
- TestUpdatePermissions_Success: PUT /permissions で 200
- TestUpdatePermissions_WithAssociation: 自治会別設定更新で 200
- TestEmergencyAppointment_Success: POST /permissions/emergency-appointment で 200
- TestEmergencyAppointment_InvalidRole: 不正ロールで 400
- TestGetLogs_Success: GET /permissions/logs で 200

【権限チェックミドルウェアのテスト】
テスト環境での CheckPermission 呼び出し:
  permRepo.CheckPermission(ctx, "vice_admin", "accounts", nil)  // 4引数
  permRepo.CheckPermission(ctx, "user_admin", "files", nil)     // assocID=nil
```

---

## テスト実行コマンド

```bash
# 全テスト実行
cd backend && go test ./...

# 特定パッケージのテスト
go test ./internal/auth/...
go test ./internal/files/...
go test ./internal/notice/...
go test ./internal/survey/...
go test ./internal/account/...
go test ./internal/association/...
go test ./internal/permission/...

# 詳細出力
go test -v ./internal/auth/...

# カバレッジ計測
go test -cover ./...

# テストDB の URL 指定
DATABASE_URL="postgres://..." go test ./...
```

---

## テストデータ管理のベストプラクティス

```go
// t.Cleanup でテスト後のクリーンアップを確実に実行
func createTestNotice(t *testing.T, assocID, userID uuid.UUID, title string) uuid.UUID {
    t.Helper()
    id := uuid.New()
    _, err := testDB.Exec(ctx,
        `INSERT INTO notices (id, association_id, created_by, title, body)
         VALUES ($1, $2, $3, $4, $5)`,
        id, assocID, userID, title, "テスト本文")
    require.NoError(t, err)
    t.Cleanup(func() {
        testDB.Exec(ctx, `DELETE FROM notices WHERE id = $1`, id)
    })
    return id
}

// permission の withPermission ヘルパー（association_id IS NULL 指定必須）
func withPermission(t *testing.T, roleID, featureID uuid.UUID, perms map[string]bool) {
    t.Helper()
    _, err := testDB.Exec(ctx, `
        INSERT INTO role_permissions
            (role_id, feature_id, association_id, can_view, can_create, can_edit, can_delete)
        VALUES ($1, $2, NULL, $3, $4, $5, $6)
        ON CONFLICT (role_id, feature_id) WHERE association_id IS NULL
        DO UPDATE SET
            can_view=$3, can_create=$4, can_edit=$5, can_delete=$6`,
        roleID, featureID,
        perms["view"], perms["create"], perms["edit"], perms["delete"])
    require.NoError(t, err)
}
```
