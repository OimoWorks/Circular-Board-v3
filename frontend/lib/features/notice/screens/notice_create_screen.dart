import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../main.dart';
import '../providers/notice_provider.dart';

class NoticeCreateScreen extends ConsumerStatefulWidget {
  const NoticeCreateScreen({super.key});

  @override
  ConsumerState<NoticeCreateScreen> createState() =>
      _NoticeCreateScreenState();
}

class _NoticeCreateScreenState extends ConsumerState<NoticeCreateScreen> {
  final _formKey = GlobalKey<FormState>();
  final _titleController = TextEditingController();
  final _bodyController = TextEditingController();
  bool _isPinned = false;

  @override
  void dispose() {
    _titleController.dispose();
    _bodyController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) return;

    final ok = await ref.read(noticeNotifierProvider).create(
          title: _titleController.text.trim(),
          body: _bodyController.text.trim(),
          isPinned: _isPinned,
        );

    if (ok && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('お知らせを登録しました'),
          backgroundColor: AppColors.primary,
        ),
      );
      Navigator.of(context).pop();
    } else if (!ok && mounted) {
      final msg = ref.read(noticeNotifierProvider).errorMessage ??
          'お知らせの登録に失敗しました';
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(msg), backgroundColor: Colors.red),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final isSubmitting =
        ref.watch(noticeNotifierProvider.select((n) => n.isSubmitting));

    return Scaffold(
      appBar: AppBar(
        title: const Text(
          'お知らせを登録',
          style: TextStyle(fontWeight: FontWeight.bold),
        ),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
        elevation: 0,
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // ─── タイトル ───────────────────────────────────
              const _FieldLabel(label: 'タイトル', required: true),
              const SizedBox(height: 6),
              TextFormField(
                controller: _titleController,
                decoration: const InputDecoration(
                  hintText: 'お知らせのタイトルを入力',
                  prefixIcon: Icon(Icons.title_rounded),
                ),
                textInputAction: TextInputAction.next,
                validator: (v) {
                  if (v == null || v.trim().isEmpty) {
                    return 'タイトルを入力してください';
                  }
                  if (v.trim().length > 255) {
                    return '255文字以内で入力してください';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 20),

              // ─── 本文 ────────────────────────────────────────
              const _FieldLabel(label: '本文', required: true),
              const SizedBox(height: 6),
              TextFormField(
                controller: _bodyController,
                decoration: const InputDecoration(
                  hintText: 'お知らせの内容を入力',
                  prefixIcon: Icon(Icons.notes_rounded),
                  alignLabelWithHint: true,
                ),
                maxLines: 8,
                textInputAction: TextInputAction.newline,
                validator: (v) {
                  if (v == null || v.trim().isEmpty) {
                    return '本文を入力してください';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 20),

              // ─── 重要フラグ ──────────────────────────────────
              Card(
                child: SwitchListTile(
                  title: const Text(
                    '重要なお知らせとしてピン留めする',
                    style: TextStyle(fontWeight: FontWeight.w600),
                  ),
                  subtitle: const Text(
                    'ONにすると一覧の最上部に固定表示されます',
                    style: TextStyle(fontSize: 12),
                  ),
                  secondary: Icon(
                    Icons.push_pin_rounded,
                    color: _isPinned ? AppColors.accent : Colors.grey,
                  ),
                  value: _isPinned,
                  activeColor: AppColors.primary,
                  onChanged: (v) => setState(() => _isPinned = v),
                ),
              ),
              const SizedBox(height: 32),

              // ─── 登録ボタン ──────────────────────────────────
              SizedBox(
                height: 50,
                child: FilledButton.icon(
                  onPressed: isSubmitting ? null : _submit,
                  style: FilledButton.styleFrom(
                    backgroundColor: AppColors.primary,
                    disabledBackgroundColor:
                        AppColors.primaryLight.withOpacity(0.5),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(12),
                    ),
                  ),
                  icon: isSubmitting
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: Colors.white,
                          ),
                        )
                      : const Icon(Icons.send_rounded, size: 18),
                  label: Text(
                    isSubmitting ? '登録中...' : '登録する',
                    style: const TextStyle(
                        fontSize: 16, fontWeight: FontWeight.bold),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _FieldLabel extends StatelessWidget {
  final String label;
  final bool required;
  const _FieldLabel({required this.label, required this.required});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Text(
          label,
          style: const TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w600,
            color: Color(0xFF374151),
          ),
        ),
        if (required) ...[
          const SizedBox(width: 4),
          const Text('*', style: TextStyle(color: Colors.red, fontSize: 13)),
        ],
      ],
    );
  }
}
