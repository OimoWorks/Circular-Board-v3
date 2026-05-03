import 'package:circular_board/features/association/data/models/association_model.dart';
import 'package:circular_board/features/association/providers/association_provider.dart';
import 'package:circular_board/main.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AssociationFormScreen extends ConsumerStatefulWidget {
  /// null = 新規作成、非null = 編集
  final AssociationDetail? association;

  const AssociationFormScreen({super.key, this.association});

  @override
  ConsumerState<AssociationFormScreen> createState() =>
      _AssociationFormScreenState();
}

class _AssociationFormScreenState
    extends ConsumerState<AssociationFormScreen> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _nameCtrl;
  late final TextEditingController _codeCtrl;

  bool get _isEdit => widget.association != null;

  @override
  void initState() {
    super.initState();
    _nameCtrl =
        TextEditingController(text: widget.association?.name ?? '');
    _codeCtrl =
        TextEditingController(text: widget.association?.code ?? '');
  }

  @override
  void dispose() {
    _nameCtrl.dispose();
    _codeCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final notifier = ref.watch(associationManagementNotifierProvider);

    return Scaffold(
      appBar: AppBar(
        title: Text(_isEdit ? '自治会編集' : '自治会登録'),
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

              // ─── 自治会名 ─────────────────────────────────────
              TextFormField(
                controller: _nameCtrl,
                decoration: const InputDecoration(
                  labelText: '自治会名 *',
                  prefixIcon: Icon(Icons.home_work_rounded),
                  hintText: '例: 〇〇町内会',
                ),
                validator: (v) =>
                    (v == null || v.trim().isEmpty) ? '自治会名を入力してください' : null,
              ),
              const SizedBox(height: 16),

              // ─── 自治会コード ──────────────────────────────────
              TextFormField(
                controller: _codeCtrl,
                textCapitalization: TextCapitalization.characters,
                inputFormatters: [
                  FilteringTextInputFormatter.allow(RegExp(r'[A-Z0-9_]')),
                  LengthLimitingTextInputFormatter(50),
                ],
                decoration: const InputDecoration(
                  labelText: '自治会コード *',
                  prefixIcon: Icon(Icons.tag_rounded),
                  hintText: '例: EXAMPLE_01',
                  helperText: '半角英大文字・数字・アンダースコアのみ（最大50文字）',
                ),
                onChanged: (v) {
                  final upper = v.toUpperCase();
                  if (upper != v) {
                    _codeCtrl.value = _codeCtrl.value.copyWith(
                      text: upper,
                      selection: TextSelection.collapsed(
                          offset: upper.length),
                    );
                  }
                },
                validator: (v) {
                  if (v == null || v.trim().isEmpty) {
                    return 'コードを入力してください';
                  }
                  if (!RegExp(r'^[A-Z0-9_]+$').hasMatch(v)) {
                    return '半角英大文字・数字・アンダースコアのみ使用できます';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 32),

              // ─── 送信ボタン ───────────────────────────────────
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

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    ref.read(associationManagementNotifierProvider).clearError();
    final notifier = ref.read(associationManagementNotifierProvider);

    bool success;
    if (_isEdit) {
      success = await notifier.update(
        id: widget.association!.id,
        name: _nameCtrl.text.trim(),
        code: _codeCtrl.text.trim().toUpperCase(),
      );
    } else {
      success = await notifier.create(
        name: _nameCtrl.text.trim(),
        code: _codeCtrl.text.trim().toUpperCase(),
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
