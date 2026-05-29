# 08. パスワードリセット機能

## 実装対象

- パスワードリセットメール送信（SendGrid）
- パスワードリセット実行
- リセットトークン管理

---

## プロンプト

```
以下の仕様に従って、Go + Chi v5 + pgx v5 でパスワードリセット機能を実装してください。

【技術スタック】
- Go 1.24 / chi v5.0.11 / pgx v5.5.1 / uuid v1.6.0
- bcrypt: golang.org/x/crypto/bcrypt
- メール送信: SendGrid v3 API（net/http で直接呼び出し、SDK不使用）

【DBテーブル】

CREATE TABLE password_reset_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);

【APIエンドポイント（/api/v1/auth）】

POST /forgot-password
  Request:  { "email": string }
  レスポンス: 常に成功を返す（ブルートフォース対策）
  {
    "data": {
      "message": "入力されたメールアドレスにパスワードリセット用のメールを送信しました（登録済みの場合）"
    }
  }
  
  処理フロー（ゴルーチンで非同期）:
  1. system_admin のメールで検索 → なければ通常ユーザーで検索
  2. 見つからない場合は何もしない（ゴルーチン終了）
  3. 32バイトのランダムトークン生成 → hex エンコード
  4. SHA-256 でハッシュ化して password_reset_tokens に保存
  5. 有効期限: 現在時刻 + 1時間
  6. SendGrid API でリセットメール送信

POST /reset-password
  Request:  { "token": string, "password": string }
  バリデーション:
  - token, password は必須
  - password は8文字以上
  処理フロー:
  1. token を SHA-256 でハッシュ化
  2. password_reset_tokens から有効なレコード検索
     （token_hash 一致 AND used_at IS NULL AND expires_at > NOW()）
  3. 見つからない場合 400 INVALID_TOKEN
  4. bcrypt でパスワードハッシュ生成
  5. users テーブルのパスワード更新
  6. password_reset_tokens の used_at に NOW() をセット
  Response: 200 { "data": { "message": "パスワードを更新しました" } }

【SendGrid メール仕様】
エンドポイント: https://api.sendgrid.com/v3/mail/send
Headers:
  Authorization: Bearer {SENDGRID_API_KEY}
  Content-Type: application/json

メール構成:
- from: { "email": SENDGRID_FROM_EMAIL, "name": "回覧板システム" }
- subject: 「【回覧板】パスワードリセットのご案内」
- content-type: text/html
- to: リクエストのメールアドレス

HTML メール本文:
- 宛名表示（ユーザー名 + 様）
- パスワードリセットボタン（リンク）
- 有効期限: 1時間
- リンクが機能しない場合はURL直接コピー案内
- 心当たりない場合は無視するよう案内

リセットURL: {APP_BASE_URL}/reset-password?token={rawToken}

【エラーコード】
INVALID_REQUEST  400 リクエスト形式不正
INVALID_TOKEN    400 トークン不正・期限切れ・使用済み
WEAK_PASSWORD    400 パスワード8文字未満
INTERNAL_ERROR   500

【トークン生成・ハッシュ化】
// 生成
b := make([]byte, 32)
rand.Read(b)
rawToken := hex.EncodeToString(b)  // 64文字のhex文字列

// ハッシュ化
h := sha256.Sum256([]byte(rawToken))
tokenHash := hex.EncodeToString(h[:])  // 64文字

// 保存するのは tokenHash のみ
// メールに含めるのは rawToken

【フロントエンド（Flutter）】

lib/features/auth/presentation/ 配下:
- forgot_password_screen.dart
  - メールアドレス入力フォーム
  - 送信後「メールを確認してください」メッセージ表示
  - ログイン画面へ戻るリンク
  
- reset_password_screen.dart
  - URLクエリパラメータ ?token=<token> からトークン取得
  - 新パスワード入力（確認用も含む）
  - パスワード一致バリデーション
  - 送信後ログイン画面へリダイレクト
  
GoRouter での設定:
GoRoute(
  path: '/reset-password',
  builder: (context, state) {
    final token = state.uri.queryParameters['token'] ?? '';
    return ResetPasswordScreen(token: token);
  },
),

auth_repository.dart に追加するメソッド:
- forgotPassword(String email) → void
- resetPassword(String token, String password) → void

エラーハンドリング:
- 400 INVALID_TOKEN: 「このリンクは無効または期限切れです」
- その他エラー: 「エラーが発生しました。再度お試しください」
```

---

## ファイル配置

### バックエンド

```
backend/internal/
├── handler/auth_handler.go                 # ForgotPassword / ResetPassword
├── repository/password_reset_repository.go # トークンCRUD
└── service/auth_service.go                 # HashPassword等のユーティリティ
```

### フロントエンド

```
frontend/lib/features/auth/
├── data/auth_repository.dart
└── presentation/
    ├── forgot_password_screen.dart
    └── reset_password_screen.dart
```

---

## 実装済みの詳細仕様

### 非同期処理（セキュリティ）

```go
// ForgotPassword は常に即座に成功レスポンスを返す
// 実際のメール送信はゴルーチンで非同期実行
go func() {
    ctx := context.Background()
    
    // system_admin 優先で検索
    user, err := h.userRepo.FindByEmailForSystemAdmin(ctx, req.Email)
    if errors.Is(err, repository.ErrNotFound) {
        user, err = h.userRepo.FindActiveByEmail(ctx, req.Email)
    }
    if err != nil {
        return  // ユーザーが存在しない場合は何もしない（ログも残さない）
    }
    
    // トークン生成・DB保存・メール送信
}()
```

### パスワードリセットトークンのライフサイクル

1. **生成**: `POST /forgot-password` でランダム生成・DB保存
2. **使用**: `POST /reset-password` でトークン検証・パスワード更新
3. **無効化**: `used_at = NOW()` で使用済みマーク
4. **期限切れ**: `expires_at < NOW()` の場合はエラー

### HTMLメールテンプレート

```html
<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #2D6A4F;">パスワードリセットのご案内</h2>
  <p>{name} 様</p>
  <p>以下のボタンをクリックしてパスワードを再設定してください。</p>
  <p>リンクの有効期限は <strong>1時間</strong> です。</p>
  <a href="{resetURL}" style="display:inline-block;padding:12px 24px;background:#2D6A4F;
     color:#fff;border-radius:8px;text-decoration:none;font-weight:bold;">
    パスワードを再設定する
  </a>
  <p style="margin-top:24px;font-size:12px;color:#666;">
    ボタンが機能しない場合は以下のURLをブラウザにコピーしてください：<br/>
    <a href="{resetURL}">{resetURL}</a>
  </p>
  <p style="font-size:12px;color:#666;">
    このメールに心当たりのない場合は無視してください。
  </p>
</body>
</html>
```

### SENDGRID_API_KEY が未設定の場合

```go
if cfg.SendGridAPIKey == "" {
    log.Printf("SENDGRID_API_KEY not set; reset URL: %s", resetURL)
    return nil  // 開発環境ではログのみ出力してエラーにしない
}
```
