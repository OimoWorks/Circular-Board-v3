# 09. UI修正・バグ修正・スマホ対応

このファイルは、実際に発生したバグ修正・UI改善の依頼プロンプト集です。
同種の問題が発生した際のテンプレートとして活用してください。

※ importパスはプロジェクトのディレクトリ構造に合わせて変更してください

---

## 01. コンパイルエラー修正（import 不足）

```
以下のコンパイルエラーを修正してください。

【エラー】
lib/features/permission/screens/permission_screen.dart:
  The getter 'apiClientProvider' isn't defined for the class '_PermissionScreenState'.

【原因】
apiClientProvider の import が不足している。

【対応方針】
1. 他の画面ファイル（home_screen.dart 等）で apiClientProvider をどのように
   import しているか確認する
2. 同じ import パスを permission_screen.dart に追加する
3. emergency_appointment_screen.dart にも同様の問題があれば合わせて修正する
4. 修正後にコンパイルが通ることを確認すること

【正解の import パス】
import '../../auth/presentation/auth_provider.dart';
```

---

## 02. バリデーションエラーメッセージ改善

```
アンケート登録画面のバリデーションを修正してください。

【現状】
必須項目が未入力の場合に「予期せぬエラーが発生しました」と表示される。

【修正内容】
フロントエンドのバリデーション（登録ボタン押下時、API呼び出し前）:
  - タイトルが空: 「タイトルを入力してください」（Form の validator で実装済み）
  - 回答期限が未設定: 「回答期限を設定してください」（SnackBar）
  - 質問が0件: 「質問を1つ以上追加してください」（SnackBar）
  - 質問テキストが空: 「質問内容を入力してください」（SnackBar）← 追加
  - 選択肢が1件以下: 「選択肢を2つ以上追加してください」（SnackBar）

APIエラーのマッピング（survey_repository.dart の mapDioError）:
  - API の error.message がある場合はそれを優先表示
  - error.message がない場合のフォールバック:
    - 400: 「入力内容を確認してください」
    - 401: 「再度ログインしてください」
    - 403: 「この操作の権限がありません」
    - 500: 「サーバーエラーが発生しました。しばらく待ってから再試行してください」

create() に DioException → SurveyException の変換を追加:
  try { ... } on DioException catch (e) { throw mapDioError(e); }
```

---

## 03. 成功メッセージに実名を表示

```
権限更新時のメッセージを改善してください。

【現状】
権限を更新すると「自治会の権限設定を更新しました」と表示される。

【修正内容】
- 更新した対象の自治会名をメッセージに含める
- 例: 自治会Aの権限を更新したら「自治会Aの権限を更新しました」

【実装箇所】
frontend/lib/features/permission/providers/permission_provider.dart
  PermissionNotifier に _selectedAssociationName フィールドを追加:
    String? _selectedAssociationName
  
  selectAssociation() メソッドのシグネチャ変更:
    Future<void> selectAssociation(String? associationId, {String? associationName})
  
  save() の成功メッセージ:
    if (_selectedAssociationId == null) {
      _successMessage = 'デフォルト権限を更新しました';
    } else if (_selectedAssociationName != null) {
      _successMessage = '${_selectedAssociationName!}の権限を更新しました';
    } else {
      _successMessage = '権限を更新しました';
    }

frontend/lib/features/permission/screens/permission_screen.dart
  _AssociationSelector の onChanged:
    onChanged: (id) {
      final name = id == null
          ? null
          : _associations.firstWhere((a) => a.id == id).name;
      notifier.selectAssociation(id, associationName: name);
    },
```

---

## 04. 特定ロールのホーム画面から特定セクションを非表示

```
system_admin のTOP画面から「最新のお知らせ」セクションを削除してください。

【現状】
home_screen.dart の _buildContent で全ロールに最新お知らせセクションが表示される。

【修正内容】
_RecentNoticesSection を system_admin 以外のみ表示するように変更:

// 修正前
_RecentNoticesSection(isEasy: isEasy),
SizedBox(height: isEasy ? 20 : 16),

// 修正後
if (user.role != 'system_admin') ...[
  _RecentNoticesSection(isEasy: isEasy),
  SizedBox(height: isEasy ? 20 : 16),
],

同様に、files と surveys のセクションも system_admin には表示不要な場合は
同じパターンで条件分岐を追加する。
```

---

## 05. スマホ対応（レスポンシブ対応）

```
以下の画面をスマホ対応（レスポンシブ）にしてください。

【対象画面】
- アカウント管理一覧画面（account_list_screen.dart）
- アンケート一覧画面（survey_list_screen.dart）

【修正内容】
1. LayoutBuilder または MediaQuery.of(context).size.width で画面幅を取得
2. 幅 600px 未満をスマホ、それ以上をデスクトップとして切り替え
3. スマホ: ListView（縦スクロール、カード1列）
   デスクトップ: GridView または DataTable

【Flutter での実装パターン】
LayoutBuilder(
  builder: (context, constraints) {
    final isSmall = constraints.maxWidth < 600;
    return isSmall ? _buildMobileLayout() : _buildDesktopLayout();
  },
)

【注意事項】
- kIsWeb は使わない（ブラウザのウィンドウサイズで判定）
- デスクトップとスマホで同じデータソース・状態管理を使う
```

---

## 06. 日付フォーマット統一

```
アプリ内の日付表示を統一してください。

【現状】
一部画面で ISO 8601 形式（2024-05-01T00:00:00Z）がそのまま表示されている。

【修正内容】
intl パッケージ（^0.19.0）を使用して日本語形式に統一:

// pubspec.yaml 依存済み
import 'package:intl/intl.dart';

// 日付のみ
DateFormat('yyyy/MM/dd').format(dateTime)  // → 2024/05/01

// 日時
DateFormat('yyyy/MM/dd HH:mm').format(dateTime)  // → 2024/05/01 23:59

// 使用箇所
- アンケートの回答期限（expires_at）
- ファイルのアップロード日時（created_at）
- お知らせの投稿日時（created_at）
- アカウントの作成日時（created_at）
```

---

## 07. エラーバナーの改善

```
エラーメッセージの表示方法を改善してください。

【現状】
エラーが SnackBar で表示されるが、長文の場合に見切れる。

【修正内容】
- 短いエラー: SnackBar（現状維持）
- APIエラー等の重要なエラー: 画面上部のバナー表示

バナーコンポーネントの実装例（permission_screen.dart の _Banner を参考）:
Container(
  width: double.infinity,
  color: isError ? Colors.red[100] : Colors.green[100],
  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
  child: Row(
    children: [
      Icon(isError ? Icons.error_outline : Icons.check_circle_outline, ...),
      Expanded(child: Text(message, ...)),
      IconButton(icon: Icon(Icons.close), onPressed: onClose),
    ],
  ),
)
```

---

## 08. ローディング状態の改善

```
ボタン押下中のローディング表示を統一してください。

【修正内容】
送信ボタンのローディング状態パターン:

// 標準パターン（アンケート作成ボタンを参考）
SizedBox(
  height: 50,
  child: FilledButton.icon(
    onPressed: isBusy ? null : _submit,
    icon: isBusy
        ? const SizedBox(
            width: 18, height: 18,
            child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
        : const Icon(Icons.send_rounded),
    label: Text(isBusy ? '処理中...' : '送信する'),
    style: FilledButton.styleFrom(backgroundColor: AppColors.primary),
  ),
)

// AppColors.primary = Color(0xFF2D6A4F)（メインカラー）
```
