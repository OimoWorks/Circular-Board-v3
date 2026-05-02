import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../../main.dart';
import '../../../auth/presentation/auth_provider.dart';
import '../data/models/notice_model.dart';
import '../providers/notice_provider.dart';

class NoticeListScreen extends ConsumerStatefulWidget {
  const NoticeListScreen({super.key});

  @override
  ConsumerState<NoticeListScreen> createState() => _NoticeListScreenState();
}

class _NoticeListScreenState extends ConsumerState<NoticeListScreen> {
  @override
  void initState() {
    super.initState();
    Future.microtask(() => ref.read(noticeNotifierProvider).load());
  }

  @override
  Widget build(BuildContext context) {
    final notifier = ref.watch(noticeNotifierProvider);
    final user = ref.watch(currentUserProvider);
    final isAdmin =
        user?.isAssociationAdmin == true || user?.isSystemAdmin == true;
    final isEasy = ref.watch(easyModeProvider);

    return Scaffold(
      appBar: AppBar(
        title: Text(
          'お知らせ',
          style: TextStyle(
            fontWeight: FontWeight.bold,
            fontSize: isEasy ? 20 : 17,
          ),
        ),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
        elevation: 0,
        actions: [
          // かんたんモード切替
          IconButton(
            icon: Icon(
              isEasy ? Icons.text_decrease_rounded : Icons.text_increase_rounded,
            ),
            tooltip: isEasy ? '通常モード' : 'かんたんモード',
            onPressed: () =>
                ref.read(easyModeProvider.notifier).state = !isEasy,
          ),
          if (isAdmin)
            IconButton(
              icon: const Icon(Icons.add_rounded),
              tooltip: 'お知らせを追加',
              onPressed: () => context.push('/notices/create'),
            ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () => ref.read(noticeNotifierProvider).load(),
        color: AppColors.primary,
        child: notifier.isLoading
            ? const Center(child: CircularProgressIndicator())
            : notifier.errorMessage != null
                ? _ErrorView(
                    message: notifier.errorMessage!,
                    onRetry: () => ref.read(noticeNotifierProvider).load(),
                  )
                : notifier.notices.isEmpty
                    ? _EmptyView(isEasy: isEasy)
                    : _NoticeListView(
                        notices: notifier.notices,
                        isAdmin: isAdmin,
                        isEasy: isEasy,
                      ),
      ),
    );
  }
}

// ─── リストビュー ──────────────────────────────────────────────────

class _NoticeListView extends StatelessWidget {
  final List<NoticeModel> notices;
  final bool isAdmin;
  final bool isEasy;

  const _NoticeListView({
    required this.notices,
    required this.isAdmin,
    required this.isEasy,
  });

  @override
  Widget build(BuildContext context) {
    final pinned = notices.where((n) => n.isPinned).toList();
    final normal = notices.where((n) => !n.isPinned).toList();

    return ListView(
      padding: EdgeInsets.all(isEasy ? 16 : 12),
      children: [
        if (pinned.isNotEmpty) ...[
          _SectionLabel(label: '重要なお知らせ', isEasy: isEasy),
          const SizedBox(height: 8),
          ...pinned.map((n) => _NoticeTile(
                notice: n,
                isAdmin: isAdmin,
                isEasy: isEasy,
              )),
          const SizedBox(height: 16),
        ],
        if (normal.isNotEmpty) ...[
          if (pinned.isNotEmpty)
            _SectionLabel(label: 'お知らせ', isEasy: isEasy),
          const SizedBox(height: 8),
          ...normal.map((n) => _NoticeTile(
                notice: n,
                isAdmin: isAdmin,
                isEasy: isEasy,
              )),
        ],
      ],
    );
  }
}

// ─── セクションラベル ──────────────────────────────────────────────

class _SectionLabel extends StatelessWidget {
  final String label;
  final bool isEasy;
  const _SectionLabel({required this.label, required this.isEasy});

  @override
  Widget build(BuildContext context) {
    return Text(
      label,
      style: TextStyle(
        fontSize: isEasy ? 16 : 13,
        fontWeight: FontWeight.bold,
        color: AppColors.primaryDark,
      ),
    );
  }
}

// ─── お知らせタイル ────────────────────────────────────────────────

class _NoticeTile extends ConsumerWidget {
  final NoticeModel notice;
  final bool isAdmin;
  final bool isEasy;

  const _NoticeTile({
    required this.notice,
    required this.isAdmin,
    required this.isEasy,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final dateLabel = DateFormat('yyyy/MM/dd').format(notice.createdAt.toLocal());

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      elevation: notice.isPinned ? 2 : 1,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: notice.isPinned
            ? const BorderSide(color: AppColors.accent, width: 1.5)
            : BorderSide.none,
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => context.push('/notices/${notice.id}'),
        child: Padding(
          padding: EdgeInsets.symmetric(
            horizontal: isEasy ? 16 : 14,
            vertical: isEasy ? 14 : 12,
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // ピン留めアイコン / 未読ドット
              Padding(
                padding: const EdgeInsets.only(top: 2, right: 10),
                child: notice.isPinned
                    ? Icon(Icons.push_pin_rounded,
                        size: isEasy ? 20 : 16, color: AppColors.accent)
                    : notice.isRead
                        ? SizedBox(width: isEasy ? 20 : 16)
                        : Container(
                            width: isEasy ? 10 : 8,
                            height: isEasy ? 10 : 8,
                            margin: EdgeInsets.only(
                                top: isEasy ? 5 : 4,
                                right: isEasy ? 5 : 4),
                            decoration: const BoxDecoration(
                              color: AppColors.primary,
                              shape: BoxShape.circle,
                            ),
                          ),
              ),
              // テキスト
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      notice.title,
                      style: TextStyle(
                        fontSize: isEasy ? 17 : 14,
                        fontWeight: notice.isRead
                            ? FontWeight.normal
                            : FontWeight.bold,
                        color: notice.isRead
                            ? Colors.grey[600]
                            : Colors.black87,
                      ),
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      dateLabel,
                      style: TextStyle(
                        fontSize: isEasy ? 13 : 11,
                        color: Colors.grey[500],
                      ),
                    ),
                  ],
                ),
              ),
              // 削除ボタン（管理者のみ）
              if (isAdmin)
                IconButton(
                  icon: const Icon(Icons.delete_outline_rounded,
                      color: Colors.red, size: 20),
                  tooltip: '削除',
                  onPressed: () => _confirmDelete(context, ref),
                ),
              const Icon(Icons.chevron_right_rounded,
                  color: AppColors.primary, size: 20),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _confirmDelete(BuildContext context, WidgetRef ref) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('お知らせを削除'),
        content: Text('「${notice.title}」を削除してよいですか？'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('キャンセル'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('削除'),
          ),
        ],
      ),
    );
    if (confirmed == true && context.mounted) {
      final ok = await ref.read(noticeNotifierProvider).delete(notice.id);
      if (!ok && context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
                ref.read(noticeNotifierProvider).errorMessage ?? '削除に失敗しました'),
            backgroundColor: Colors.red,
          ),
        );
      }
    }
  }
}

// ─── 空状態 ────────────────────────────────────────────────────────

class _EmptyView extends StatelessWidget {
  final bool isEasy;
  const _EmptyView({required this.isEasy});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.notifications_none_rounded,
              size: isEasy ? 72 : 56, color: Colors.grey[300]),
          const SizedBox(height: 16),
          Text(
            'お知らせはありません',
            style: TextStyle(
              fontSize: isEasy ? 18 : 15,
              color: Colors.grey[500],
            ),
          ),
        ],
      ),
    );
  }
}

// ─── エラー状態 ────────────────────────────────────────────────────

class _ErrorView extends StatelessWidget {
  final String message;
  final VoidCallback onRetry;
  const _ErrorView({required this.message, required this.onRetry});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.error_outline_rounded,
              size: 48, color: Colors.red[300]),
          const SizedBox(height: 12),
          Text(message,
              style: const TextStyle(color: Colors.black54, fontSize: 14)),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: onRetry,
            icon: const Icon(Icons.refresh_rounded),
            label: const Text('再試行'),
            style: FilledButton.styleFrom(
                backgroundColor: AppColors.primary),
          ),
        ],
      ),
    );
  }
}
