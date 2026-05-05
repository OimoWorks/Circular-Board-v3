import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../main.dart';
import '../../files/data/models/file_model.dart';
import '../../files/providers/file_provider.dart';
import '../../notice/data/models/notice_model.dart';
import '../../notice/providers/notice_provider.dart';
import '../../survey/data/models/survey_model.dart';
import '../../survey/providers/survey_provider.dart';
import 'auth_provider.dart';

// 管理メニューを表示するロールかどうか
bool _isAdmin(String role) =>
    role == 'association_admin' || role == 'system_admin';

class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(currentUserProvider);
    final auth = ref.watch(authNotifierProvider);
    final theme = Theme.of(context);
    final isEasy = ref.watch(easyModeProvider);

    if (user == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    return Scaffold(
      appBar: AppBar(
        title: Row(
          children: [
            const Icon(Icons.article_rounded, size: 22, color: Colors.white),
            const SizedBox(width: 8),
            Text(
              '回覧板',
              style: TextStyle(
                fontWeight: FontWeight.bold,
                letterSpacing: 2,
                fontSize: isEasy ? 20 : 17,
              ),
            ),
          ],
        ),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
        elevation: 0,
        actions: [
          // かんたんモード切替
          IconButton(
            icon: Icon(
              isEasy
                  ? Icons.text_decrease_rounded
                  : Icons.text_increase_rounded,
            ),
            tooltip: isEasy ? '通常モード' : 'かんたんモード',
            onPressed: () =>
                ref.read(easyModeProvider.notifier).state = !isEasy,
          ),
          IconButton(
            icon: const Icon(Icons.logout_rounded),
            tooltip: 'ログアウト',
            onPressed: auth.isLoading
                ? null
                : () async {
                    final confirmed = await _showLogoutDialog(context);
                    if (confirmed && context.mounted) {
                      await ref.read(authNotifierProvider).logout();
                    }
                  },
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: EdgeInsets.all(isEasy ? 20 : 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ─── ユーザー情報カード ──────────────────────────────
            Card(
              child: Padding(
                padding: const EdgeInsets.all(20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        CircleAvatar(
                          radius: 26,
                          backgroundColor:
                              AppColors.primary.withOpacity(0.12),
                          child: const Icon(Icons.person_rounded,
                              size: 30, color: AppColors.primary),
                        ),
                        const SizedBox(width: 14),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                user.name,
                                style: theme.textTheme.titleMedium?.copyWith(
                                  fontWeight: FontWeight.bold,
                                  fontSize: isEasy ? 18 : null,
                                ),
                              ),
                              const SizedBox(height: 4),
                              _RoleBadge(
                                  role: user.role, label: user.roleLabel),
                            ],
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 16),
                    const Divider(),
                    const SizedBox(height: 10),
                    _InfoRow(
                      icon: Icons.email_rounded,
                      label: 'メールアドレス',
                      value: user.email,
                      isEasy: isEasy,
                    ),
                    const SizedBox(height: 10),
                    _InfoRow(
                      icon: Icons.home_work_rounded,
                      label: '所属自治会',
                      value: user.associationName ?? '（全自治会）',
                      isEasy: isEasy,
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),

            // ─── メニュー：お知らせ ──────────────────────────────
            _NoticeMenuCard(isEasy: isEasy),
            const SizedBox(height: 10),

            // ─── メニュー：アンケート ────────────────────────────
            if (user.role != 'system_admin')
              _SurveyMenuCard(isEasy: isEasy),
            if (user.role != 'system_admin') const SizedBox(height: 10),

            // ─── メニュー：回覧物 ────────────────────────────────
            _MenuCard(
              icon: Icons.folder_rounded,
              title: '回覧物',
              subtitle: '回覧資料のアップロード、プレビュー、ダウンロード',
              onTap: () => context.push('/files'),
              isEasy: isEasy,
            ),
            const SizedBox(height: 10),

            // ─── 管理メニュー（association_admin 以上） ──────────
            if (_isAdmin(user.role)) ...[
              _MenuCard(
                icon: Icons.manage_accounts_rounded,
                title: 'アカウント管理',
                subtitle: '自治会メンバーのアカウントを登録・管理する',
                onTap: () => context.push('/accounts'),
                isEasy: isEasy,
                color: const Color(0xFF2D6A4F),
              ),
              const SizedBox(height: 10),
            ],

            // ─── 自治会管理（system_admin のみ） ─────────────────
            if (user.role == 'system_admin') ...[
              _MenuCard(
                icon: Icons.home_work_rounded,
                title: '自治会管理',
                subtitle: '自治会の登録・編集・有効無効の管理',
                onTap: () => context.push('/associations'),
                isEasy: isEasy,
                color: const Color(0xFF1B4332),
              ),
              const SizedBox(height: 10),
            ],

            const SizedBox(height: 10),

            // ─── 最新のお知らせ ──────────────────────────────────
            _RecentNoticesSection(theme: theme, isEasy: isEasy),
            const SizedBox(height: 20),

            // ─── 最新のアンケート（system_admin 以外） ───────────
            if (user.role != 'system_admin') ...[
              _RecentSurveysSection(theme: theme, isEasy: isEasy),
              const SizedBox(height: 20),
            ],

            // ─── 最新の回覧物 ────────────────────────────────────
            _RecentFilesSection(theme: theme, isEasy: isEasy),
          ],
        ),
      ),
    );
  }

  Future<bool> _showLogoutDialog(BuildContext context) async {
    return await showDialog<bool>(
          context: context,
          builder: (ctx) => AlertDialog(
            title: const Text('ログアウト'),
            content: const Text('ログアウトしてよいですか？'),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(ctx).pop(false),
                child: const Text('キャンセル'),
              ),
              FilledButton(
                onPressed: () => Navigator.of(ctx).pop(true),
                child: const Text('ログアウト'),
              ),
            ],
          ),
        ) ??
        false;
  }
}

// ─── お知らせメニューカード（未読バッジ付き） ─────────────────────

class _NoticeMenuCard extends ConsumerWidget {
  final bool isEasy;
  const _NoticeMenuCard({required this.isEasy});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final unreadAsync = ref.watch(unreadCountProvider);

    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => context.push('/notices'),
        child: Padding(
          padding: const EdgeInsets.all(18),
          child: Row(
            children: [
              Container(
                width: 46,
                height: 46,
                decoration: BoxDecoration(
                  color: AppColors.primary.withOpacity(0.10),
                  borderRadius: BorderRadius.circular(11),
                ),
                child: Stack(
                  children: [
                    const Center(
                      child: Icon(Icons.notifications_rounded,
                          color: AppColors.primary, size: 24),
                    ),
                    // 未読バッジ
                    unreadAsync.when(
                      data: (count) => count > 0
                          ? Positioned(
                              top: 4,
                              right: 4,
                              child: Container(
                                padding: const EdgeInsets.all(2),
                                constraints: const BoxConstraints(
                                    minWidth: 16, minHeight: 16),
                                decoration: const BoxDecoration(
                                  color: Colors.red,
                                  shape: BoxShape.circle,
                                ),
                                child: Text(
                                  count > 99 ? '99+' : '$count',
                                  style: const TextStyle(
                                    color: Colors.white,
                                    fontSize: 9,
                                    fontWeight: FontWeight.bold,
                                  ),
                                  textAlign: TextAlign.center,
                                ),
                              ),
                            )
                          : const SizedBox.shrink(),
                      loading: () => const SizedBox.shrink(),
                      error: (_, __) => const SizedBox.shrink(),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Text(
                          'お知らせ',
                          style: TextStyle(
                            fontWeight: FontWeight.bold,
                            fontSize: isEasy ? 16 : 15,
                          ),
                        ),
                        const SizedBox(width: 8),
                        unreadAsync.when(
                          data: (count) => count > 0
                              ? Container(
                                  padding: const EdgeInsets.symmetric(
                                      horizontal: 8, vertical: 2),
                                  decoration: BoxDecoration(
                                    color: Colors.red,
                                    borderRadius: BorderRadius.circular(10),
                                  ),
                                  child: Text(
                                    '未読 $count件',
                                    style: const TextStyle(
                                      color: Colors.white,
                                      fontSize: 11,
                                      fontWeight: FontWeight.bold,
                                    ),
                                  ),
                                )
                              : const SizedBox.shrink(),
                          loading: () => const SizedBox.shrink(),
                          error: (_, __) => const SizedBox.shrink(),
                        ),
                      ],
                    ),
                    const SizedBox(height: 2),
                    Text(
                      '自治会からのお知らせを確認する',
                      style: TextStyle(
                          fontSize: isEasy ? 13 : 12,
                          color: Colors.grey[600]),
                    ),
                  ],
                ),
              ),
              const Icon(Icons.chevron_right_rounded,
                  color: AppColors.primary),
            ],
          ),
        ),
      ),
    );
  }
}

// ─── 最新のお知らせセクション ────────────────────────────────────

class _RecentNoticesSection extends ConsumerWidget {
  final ThemeData theme;
  final bool isEasy;
  const _RecentNoticesSection({required this.theme, required this.isEasy});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final recentAsync = ref.watch(recentNoticesProvider);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              '最新のお知らせ',
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.bold,
                color: AppColors.primaryDark,
                fontSize: isEasy ? 16 : null,
              ),
            ),
            TextButton.icon(
              onPressed: () => context.push('/notices'),
              icon: const Icon(Icons.arrow_forward_rounded, size: 16),
              label: const Text('すべて見る'),
              style: TextButton.styleFrom(
                foregroundColor: AppColors.primary,
                padding:
                    const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        recentAsync.when(
          data: (notices) {
            if (notices.isEmpty) {
              return Padding(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: Center(
                  child: Text(
                    '最新のお知らせはありません',
                    style: TextStyle(
                        color: Colors.grey[500],
                        fontSize: isEasy ? 15 : 13),
                  ),
                ),
              );
            }
            return Card(
              child: ListView.separated(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: notices.length,
                separatorBuilder: (_, __) =>
                    const Divider(height: 1, indent: 52),
                itemBuilder: (ctx, i) =>
                    _RecentNoticeTile(notice: notices[i], isEasy: isEasy),
              ),
            );
          },
          loading: () => const Center(
            child: Padding(
              padding: EdgeInsets.symmetric(vertical: 20),
              child: CircularProgressIndicator(),
            ),
          ),
          error: (_, __) => Padding(
            padding: const EdgeInsets.symmetric(vertical: 12),
            child: Text(
              '取得に失敗しました',
              style: TextStyle(color: Colors.red[400], fontSize: 13),
            ),
          ),
        ),
      ],
    );
  }
}

// ─── 最新お知らせタイル ───────────────────────────────────────────

class _RecentNoticeTile extends StatelessWidget {
  final NoticeModel notice;
  final bool isEasy;
  const _RecentNoticeTile({required this.notice, required this.isEasy});

  @override
  Widget build(BuildContext context) {
    final dateLabel =
        DateFormat('MM/dd').format(notice.createdAt.toLocal());

    return ListTile(
      leading: Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          color: notice.isPinned
              ? AppColors.accent.withOpacity(0.15)
              : AppColors.primary.withOpacity(0.08),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Icon(
          notice.isPinned
              ? Icons.push_pin_rounded
              : Icons.notifications_rounded,
          color: notice.isPinned ? AppColors.accent : AppColors.primary,
          size: 18,
        ),
      ),
      title: Text(
        notice.title,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: TextStyle(
          fontSize: isEasy ? 15 : 13,
          fontWeight:
              notice.isRead ? FontWeight.normal : FontWeight.bold,
          color: notice.isRead ? Colors.grey[600] : Colors.black87,
        ),
      ),
      subtitle: Text(
        dateLabel,
        style: TextStyle(
            fontSize: isEasy ? 13 : 11, color: Colors.grey[500]),
      ),
      trailing: const Icon(Icons.chevron_right_rounded,
          color: AppColors.primary, size: 20),
      onTap: () => context.push('/notices/${notice.id}'),
    );
  }
}

// ─── 最新の回覧物セクション ─────────────────────────────────────

class _RecentFilesSection extends ConsumerWidget {
  final ThemeData theme;
  final bool isEasy;
  const _RecentFilesSection({required this.theme, required this.isEasy});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final recentAsync = ref.watch(recentFilesProvider);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              '最新の回覧物',
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.bold,
                color: AppColors.primaryDark,
                fontSize: isEasy ? 16 : null,
              ),
            ),
            TextButton.icon(
              onPressed: () => context.push('/files'),
              icon: const Icon(Icons.arrow_forward_rounded, size: 16),
              label: const Text('すべて見る'),
              style: TextButton.styleFrom(
                foregroundColor: AppColors.primary,
                padding:
                    const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        recentAsync.when(
          data: (files) {
            if (files.isEmpty) {
              return Padding(
                padding: const EdgeInsets.symmetric(vertical: 16),
                child: Center(
                  child: Text(
                    '最新の回覧物はありません',
                    style: TextStyle(
                        color: Colors.grey[500],
                        fontSize: isEasy ? 15 : 13),
                  ),
                ),
              );
            }
            return Card(
              child: ListView.separated(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: files.length,
                separatorBuilder: (_, __) =>
                    const Divider(height: 1, indent: 52),
                itemBuilder: (ctx, i) =>
                    _RecentFileTile(file: files[i], isEasy: isEasy),
              ),
            );
          },
          loading: () => const Center(
            child: Padding(
              padding: EdgeInsets.symmetric(vertical: 20),
              child: CircularProgressIndicator(),
            ),
          ),
          error: (_, __) => Padding(
            padding: const EdgeInsets.symmetric(vertical: 12),
            child: Text(
              '取得に失敗しました',
              style: TextStyle(color: Colors.red[400], fontSize: 13),
            ),
          ),
        ),
      ],
    );
  }
}

// ─── 最新回覧物タイル ───────────────────────────────────────────

class _RecentFileTile extends StatelessWidget {
  final FileModel file;
  final bool isEasy;
  const _RecentFileTile({required this.file, required this.isEasy});

  static const _monthLabels = [
    '1月', '2月', '3月', '4月', '5月', '6月',
    '7月', '8月', '9月', '10月', '11月', '12月',
  ];

  IconData get _icon {
    switch (file.mimeType) {
      case 'application/pdf':
        return Icons.picture_as_pdf_rounded;
      case 'image/jpeg':
      case 'image/png':
        return Icons.image_rounded;
      default:
        return Icons.insert_drive_file_rounded;
    }
  }

  Color get _iconColor {
    switch (file.mimeType) {
      case 'application/pdf':
        return Colors.red[700]!;
      case 'image/jpeg':
      case 'image/png':
        return Colors.blue[600]!;
      default:
        return Colors.grey;
    }
  }

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          color: _iconColor.withOpacity(0.10),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Icon(_icon, color: _iconColor, size: 20),
      ),
      title: Text(
        file.originalFilename,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: TextStyle(fontSize: isEasy ? 15 : 13),
      ),
      subtitle: Text(
        '${file.year}年${_monthLabels[file.month - 1]}　${file.fileSizeLabel}',
        style: TextStyle(
            fontSize: isEasy ? 13 : 11, color: Colors.grey[500]),
      ),
      trailing: const Icon(Icons.chevron_right_rounded,
          color: AppColors.primary, size: 20),
      onTap: () => context.push('/files/preview', extra: file),
    );
  }
}

// ─── ユーザー情報行 ─────────────────────────────────────────────

class _InfoRow extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final bool isEasy;

  const _InfoRow({
    required this.icon,
    required this.label,
    required this.value,
    required this.isEasy,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icon, size: 18, color: AppColors.primary),
        const SizedBox(width: 10),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              label,
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
                fontSize: isEasy ? 13 : null,
              ),
            ),
            Text(
              value,
              style: theme.textTheme.bodyMedium?.copyWith(
                fontSize: isEasy ? 15 : null,
              ),
            ),
          ],
        ),
      ],
    );
  }
}

// ─── メニューカード ─────────────────────────────────────────────

class _MenuCard extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;
  final bool isEasy;
  final Color? color;

  const _MenuCard({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
    required this.isEasy,
    this.color,
  });

  @override
  Widget build(BuildContext context) {
    final c = color ?? AppColors.primary;
    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(18),
          child: Row(
            children: [
              Container(
                width: 46,
                height: 46,
                decoration: BoxDecoration(
                  color: c.withOpacity(0.10),
                  borderRadius: BorderRadius.circular(11),
                ),
                child: Icon(icon, color: c, size: 24),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: TextStyle(
                        fontWeight: FontWeight.bold,
                        fontSize: isEasy ? 16 : 15,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      subtitle,
                      style: TextStyle(
                          fontSize: isEasy ? 13 : 12,
                          color: Colors.grey[600]),
                    ),
                  ],
                ),
              ),
              Icon(Icons.chevron_right_rounded, color: c),
            ],
          ),
        ),
      ),
    );
  }
}

// ─── ロールバッジ ───────────────────────────────────────────────

class _RoleBadge extends StatelessWidget {
  final String role;
  final String label;
  const _RoleBadge({required this.role, required this.label});

  Color _bgColor() {
    switch (role) {
      case 'system_admin':
        return const Color(0xFFFFE4E4);
      case 'association_admin':
        return const Color(0xFFD8F0E6);
      default:
        return AppColors.primary.withOpacity(0.1);
    }
  }

  Color _fgColor() {
    switch (role) {
      case 'system_admin':
        return const Color(0xFFB91C1C);
      case 'association_admin':
        return AppColors.primaryDark;
      default:
        return AppColors.primary;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: _bgColor(),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        label,
        style: TextStyle(
            fontSize: 12, fontWeight: FontWeight.w600, color: _fgColor()),
      ),
    );
  }
}

// ─────────────────────────────────────────────────────────────
// アンケートメニューカード（未回答件数バッジ付き）
// ─────────────────────────────────────────────────────────────
class _SurveyMenuCard extends ConsumerWidget {
  final bool isEasy;
  const _SurveyMenuCard({required this.isEasy});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final unansweredAsync = ref.watch(unansweredSurveyCountProvider);
    final count = unansweredAsync.valueOrNull ?? 0;

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(14),
        side: BorderSide(color: Colors.grey[200]!),
      ),
      child: InkWell(
        onTap: () => context.push('/surveys'),
        borderRadius: BorderRadius.circular(14),
        child: Padding(
          padding:
              EdgeInsets.symmetric(horizontal: 16, vertical: isEasy ? 18 : 14),
          child: Row(
            children: [
              Stack(
                clipBehavior: Clip.none,
                children: [
                  Container(
                    width: 44,
                    height: 44,
                    decoration: BoxDecoration(
                      color: Colors.teal.withOpacity(0.12),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: const Icon(Icons.poll_rounded,
                        color: Colors.teal, size: 24),
                  ),
                  if (count > 0)
                    Positioned(
                      top: -4,
                      right: -4,
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 5, vertical: 1),
                        decoration: BoxDecoration(
                          color: Colors.red,
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: Text(
                          count > 99 ? '99+' : '$count',
                          style: const TextStyle(
                              color: Colors.white,
                              fontSize: 10,
                              fontWeight: FontWeight.bold),
                        ),
                      ),
                    ),
                ],
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'アンケート',
                      style: TextStyle(
                        fontWeight: FontWeight.bold,
                        fontSize: isEasy ? 17 : 15,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      count > 0 ? '未回答が $count 件あります' : 'アンケートへの回答・確認',
                      style: TextStyle(
                        fontSize: isEasy ? 13 : 12,
                        color: count > 0 ? Colors.orange[700] : Colors.grey[600],
                      ),
                    ),
                  ],
                ),
              ),
              const Icon(Icons.chevron_right_rounded, color: Colors.grey),
            ],
          ),
        ),
      ),
    );
  }
}

// ─────────────────────────────────────────────────────────────
// 最新のアンケートセクション
// ─────────────────────────────────────────────────────────────
class _RecentSurveysSection extends ConsumerWidget {
  final ThemeData theme;
  final bool isEasy;

  const _RecentSurveysSection({required this.theme, required this.isEasy});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final surveysAsync = ref.watch(recentSurveysProvider);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            const Icon(Icons.poll_rounded, size: 18, color: Colors.teal),
            const SizedBox(width: 6),
            Text('最新のアンケート',
                style: theme.textTheme.titleSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                  fontSize: isEasy ? 15 : null,
                )),
            const Spacer(),
            TextButton(
              onPressed: () => context.push('/surveys'),
              child: const Text('すべて見る',
                  style: TextStyle(fontSize: 12)),
            ),
          ],
        ),
        const SizedBox(height: 6),
        surveysAsync.when(
          data: (surveys) {
            if (surveys.isEmpty) {
              return Text('アンケートはありません',
                  style: TextStyle(
                      color: Colors.grey[500], fontSize: isEasy ? 14 : 12));
            }
            return Column(
              children: surveys
                  .map((s) => _SurveyListTile(survey: s, isEasy: isEasy))
                  .toList(),
            );
          },
          loading: () => const LinearProgressIndicator(),
          error: (_, __) => Text('取得に失敗しました',
              style: TextStyle(
                  color: Colors.grey[500], fontSize: isEasy ? 14 : 12)),
        ),
      ],
    );
  }
}

class _SurveyListTile extends StatelessWidget {
  final SurveyModel survey;
  final bool isEasy;

  const _SurveyListTile({required this.survey, required this.isEasy});

  @override
  Widget build(BuildContext context) {
    final statusColor = survey.isExpired
        ? Colors.grey
        : survey.isAnswered
            ? AppColors.primary
            : Colors.orange[700]!;
    final statusIcon = survey.isExpired
        ? Icons.schedule_rounded
        : survey.isAnswered
            ? Icons.check_circle_rounded
            : Icons.circle_outlined;

    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 6),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(color: Colors.grey[200]!),
      ),
      child: InkWell(
        onTap: () => context.push('/surveys/${survey.id}'),
        borderRadius: BorderRadius.circular(10),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          child: Row(
            children: [
              Icon(statusIcon, color: statusColor, size: 22),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  survey.title,
                  style: TextStyle(
                    fontSize: isEasy ? 14 : 13,
                    fontWeight: FontWeight.w500,
                    color: survey.isExpired ? Colors.grey : Colors.black87,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                  color: statusColor.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Text(
                  survey.statusLabel,
                  style: TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.w600,
                      color: statusColor),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
