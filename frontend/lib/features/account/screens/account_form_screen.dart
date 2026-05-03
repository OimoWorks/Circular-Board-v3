import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../main.dart';
import '../../../auth/presentation/auth_provider.dart';
import '../../data/models/account_model.dart';
import '../../providers/account_provider.dart';
import '../../../files/providers/association_provider.dart';

class AccountFormScreen extends ConsumerStatefulWidget {
  /// null = 新規作成、非null = 編集
  final AccountModel? account;

  /// system_admin が一覧画面で選択中の自治会 ID（新規作成時に使用）
  final String? preselectedAssociationId;

  const AccountFormScreen({
    super.key,
    this.account,
    this.preselectedAssociationId,
  });

  @override
  ConsumerState<AccountFormScreen> createState() => _AccountFormScreenState();
}

class _AccountFormScreenState extends ConsumerState<AccountFormScreen> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _nameCtrl;
  late final TextEditingController _emailCtrl;
  final TextEditingController _passwordCtrl = TextEditingController();
  String _role = 'user';
  String? _selectedAssociationId;
  bool _obscurePassword = true;

  bool get _isEdit => widget.account != null;
  bool get _isSystemAdmin =>
      ref.read(currentUserProvider)?.role == 'system_admin';

  @override
  void initState() {
    super.initState();
    final a = widget.account;
    _nameCtrl = TextEditingController(text: a?.name ?? '');
    _emailCtrl = TextEditingController(text: a?.email ?? '');
    _role = a?.role ?? 'user';
    _selectedAssociationId =
        a?.associationId.isNotEmpty == true
            ? a!.associationId
            : widget.preselectedAssociationId;
  }

  @override
  void dispose() {
    _nameCtrl.dispose();
    _emailCtrl.dispose();
    _passwordCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final notifier = ref.watch(accountNotifierProvider);
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(_isEdit ? 'アカウント編集' : 'アカウント登録'),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // エラー表示
              if (notifier.errorMessage != null)
                Container(
                  margin: const EdgeInsets.only(bottom: 16),
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.red[50],
                    borderRadius: BorderRadius.circular(10),
                    border: Border.all(color: Colors.red[200]!),
                  ),
                  child: Row(
                    children: [
                      Icon(Icons.error_outline_rounded,
                          color: Colors.red[700], size: 18),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          notifier.errorMessage!,
                          style: TextStyle(
                              color: Colors.red[700], fontSize: 13),
                        ),
                      ),
                    ],
                  ),
                ),

              // ─── 自治会選択（system_admin のみ表示） ─────────────
              if (_isSystemAdmin && !_isEdit) _buildAssociationPicker(theme),

              const SizedBox(height: 16),

              // ─── 名前 ────────────────────────────────────────────
              TextFormField(
                controller: _nameCtrl,
                decoration: const InputDecoration(
                  labelText: '名前 *',
                  prefixIcon: Icon(Icons.person_rounded),
                ),
                validator: (v) =>
                    (v == null || v.trim().isEmpty) ? '名前を入力してください' : null,
              ),
              const SizedBox(height: 16),

              // ─── メールアドレス ───────────────────────────────────
              TextFormField(
                controller: _emailCtrl,
                keyboardType: TextInputType.emailAddress,
                decoration: const InputDecoration(
                  labelText: 'メールアドレス *',
                  prefixIcon: Icon(Icons.email_rounded),
                ),
                validator: (v) {
                  if (v == null || v.trim().isEmpty) {
                    return 'メールアドレスを入力してください';
                  }
                  if (!v.contains('@')) {
                    return '正しいメールアドレスを入力してください';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 16),

              // ─── パスワード ───────────────────────────────────────
              TextFormField(
                controller: _passwordCtrl,
                obscureText: _obscurePassword,
                decoration: InputDecoration(
                  labelText: _isEdit
                      ? 'パスワード（変更する場合のみ入力）'
                      : 'パスワード *',
                  prefixIcon: const Icon(Icons.lock_rounded),
                  suffixIcon: IconButton(
                    icon: Icon(_obscurePassword
                        ? Icons.visibility_off_rounded
                        : Icons.visibility_rounded),
                    onPressed: () =>
                        setState(() => _obscurePassword = !_obscurePassword),
                  ),
                ),
                validator: (v) {
                  if (!_isEdit && (v == null || v.isEmpty)) {
                    return 'パスワードを入力してください';
                  }
                  if (v != null && v.isNotEmpty && v.length < 8) {
                    return 'パスワードは8文字以上で入力してください';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 16),

              // ─── ロール ───────────────────────────────────────────
              DropdownButtonFormField<String>(
                value: _role,
                decoration: const InputDecoration(
                  labelText: 'ロール *',
                  prefixIcon: Icon(Icons.admin_panel_settings_rounded),
                ),
                items: [
                  const DropdownMenuItem(
                      value: 'user', child: Text('一般ユーザー')),
                  const DropdownMenuItem(
                      value: 'association_admin', child: Text('自治会管理者')),
                  if (_isSystemAdmin)
                    const DropdownMenuItem(
                        value: 'system_admin', child: Text('システム管理者')),
                ],
                onChanged: (v) => setState(() => _role = v ?? 'user'),
              ),
              const SizedBox(height: 32),

              // ─── 送信ボタン ───────────────────────────────────────
              FilledButton(
                onPressed: notifier.isSubmitting ? null : _submit,
                child: notifier.isSubmitting
                    ? const SizedBox(
                        height: 20,
                        width: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : Text(_isEdit ? '更新する' : '登録する'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildAssociationPicker(ThemeData theme) {
    final assocAsync = ref.watch(allAssociationsProvider);
    return assocAsync.when(
      data: (assocs) => DropdownButtonFormField<String>(
        value: _selectedAssociationId,
        decoration: const InputDecoration(
          labelText: '所属自治会 *',
          prefixIcon: Icon(Icons.home_work_rounded),
        ),
        hint: const Text('自治会を選択'),
        items: assocs
            .map((a) => DropdownMenuItem(
                  value: a.id,
                  child: Text('${a.name}（${a.code}）'),
                ))
            .toList(),
        onChanged: (v) => setState(() => _selectedAssociationId = v),
        validator: (v) =>
            (v == null || v.isEmpty) ? '自治会を選択してください' : null,
      ),
      loading: () => const LinearProgressIndicator(),
      error: (_, __) => const Text('自治会の取得に失敗しました'),
    );
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    ref.read(accountNotifierProvider).clearError();
    final notifier = ref.read(accountNotifierProvider);

    bool success;
    if (_isEdit) {
      success = await notifier.update(
        id: widget.account!.id,
        name: _nameCtrl.text.trim(),
        email: _emailCtrl.text.trim(),
        role: _role,
        password: _passwordCtrl.text.isEmpty ? null : _passwordCtrl.text,
      );
    } else {
      success = await notifier.create(
        name: _nameCtrl.text.trim(),
        email: _emailCtrl.text.trim(),
        password: _passwordCtrl.text,
        role: _role,
        associationId: _isSystemAdmin ? _selectedAssociationId : null,
      );
    }

    if (success && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(_isEdit ? '更新しました' : '登録しました'),
          backgroundColor: AppColors.primary,
        ),
      );
      Navigator.of(context).pop(true);
    }
  }
}
