import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../main.dart';
import '../../auth/presentation/auth_provider.dart';
import '../data/models/notice_model.dart';
import '../providers/notice_provider.dart';

class NoticeDetailScreen extends ConsumerStatefulWidget {
  final String noticeId;

  const NoticeDetailScreen({super.key, required this.noticeId});

  @override
  ConsumerState<NoticeDetailScreen> createState() =>
      _NoticeDetailScreenState();
}

class _NoticeDetailScreenState extends ConsumerState<NoticeDetailScreen> {
  @override
  void initState() {
    super.initState();
    // 既読処理（遷移直後に実施）
    Future.microtask(() {
      ref.read(noticeNotifierProvider).markAsRead(widget.noticeId);
    });
  }

  @override
  Widget build(BuildContext context) {
    final notifier = ref.watch(noticeNotifierProvider);
    final user = ref.watch(currentUserProvider);
    final isAdmin =
        user?.isAssociationAdmin == true || user?.isSystemAdmin == true;
    final isEasy = ref.watch(easyModeProvider);

    // 一覧から該当レコードを探す（なければロード中扱い）
    final NoticeModel? notice = notifier.notices
        .where((n) => n.id == widget.noticeId)
        .firstOrNull;

    return Scaffold(
      appBar: AppBar(
        title: Text(
          'お知らせ詳細',
          style: TextStyle(fontSize: isEasy ? 20 : 17),
        ),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
        elevation: 0,
        actions: [
          if (isAdmin && notice != null)
            IconButton(
              icon: const Icon(Icons.delete_outline_rounded),
              tooltip: '削除',
              onPressed: () => _confirmDelete(context, ref, notice),
            ),
        ],
      ),
      body: notice == null
          ? const Center(child: CircularProgressIndicator())
          : _DetailBody(notice: notice, isEasy: isEasy),
    );
  }

  Future<void> _confirmDelete(
      BuildContext context, WidgetRef ref, NoticeModel notice) async {
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
      if (context.mounted) {
        if (ok) {
          Navigator.of(context).pop();
        } else {
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
}

// ─── 詳細本文 ─────────────────────────────────────────────────────

class _DetailBody extends StatelessWidget {
  final NoticeModel notice;
  final bool isEasy;

  const _DetailBody({required this.notice, required this.isEasy});

  @override
  Widget build(BuildContext context) {
    final dateLabel = DateFormat('yyyy年MM月dd日 HH:mm')
        .format(notice.createdAt.toLocal());

    return SingleChildScrollView(
      padding: EdgeInsets.all(isEasy ? 24 : 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // ピン留めバッジ
          if (notice.isPinned)
            Container(
              margin: const EdgeInsets.only(bottom: 12),
              padding:
                  const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                color: AppColors.accent.withOpacity(0.15),
                borderRadius: BorderRadius.circular(20),
                border: Border.all(color: AppColors.accent),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.push_pin_rounded,
                      size: 14, color: AppColors.accent),
                  const SizedBox(width: 4),
                  Text(
                    '重要なお知らせ',
                    style: TextStyle(
                      fontSize: isEasy ? 14 : 12,
                      color: AppColors.primaryDark,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ],
              ),
            ),

          // タイトル
          Text(
            notice.title,
            style: TextStyle(
              fontSize: isEasy ? 24 : 20,
              fontWeight: FontWeight.bold,
              color: Colors.black87,
            ),
          ),
          const SizedBox(height: 8),

          // 投稿日時
          Text(
            dateLabel,
            style: TextStyle(
              fontSize: isEasy ? 14 : 12,
              color: Colors.grey[500],
            ),
          ),
          const SizedBox(height: 20),
          const Divider(),
          const SizedBox(height: 20),

          // 本文
          Text(
            notice.body,
            style: TextStyle(
              fontSize: isEasy ? 18 : 15,
              height: 1.7,
              color: Colors.black87,
            ),
          ),
          const SizedBox(height: 40),
        ],
      ),
    );
  }
}
