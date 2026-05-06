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
import '../domain/user_model.dart';
import 'auth_provider.dart';

bool _isAdmin(String role) =>
    role == 'association_admin' || role == 'system_admin';

// ═══════════════════════════════════════════════════════════
// HomeScreen
// ═══════════════════════════════════════════════════════════

class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(currentUserProvider);
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
        ],
      ),
      drawer: _AppDrawer(isEasy: isEasy),
      body: SingleChildScrollView(
        padding: EdgeInsets.all(isEasy ? 20 : 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ─── ユーザー情報カード ──────────────────────────────
            _UserInfoCard(user: user, isEasy: isEasy),
            SizedBox(height: isEasy ? 20 : 16),

            // ─── 最新お知らせ ────────────────────────────────────
            _RecentNoticesSection(isEasy: isEasy),
            SizedBox(height: isEasy ? 20 : 16),

            // ─── 最新回覧物（system_admin 以外） ─────────────────
            if (user.role != 'system_admin') ...[
              _RecentFilesSection(isEasy: isEasy),
              SizedBox(height: isEasy ? 20 : 16),
            ],

            // ─── 未回答アンケート（system_admin 以外） ─────────
            if (user.role != 'system_admin')
              _RecentSurveysSection(isEasy: isEasy),
          ],
        ),
      ),
    );
  }
}

// ═══════════════════════════════════════════════════════════
// _AppDrawer（ハンバーガーメニュー）
// ═══════════════════════════════════════════════════════════

class _AppDrawer extends ConsumerWidget {
  final bool isEasy;
  const _AppDrawer({required this.isEasy});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(currentUserProvider);
    final auth = ref.watch(authNotifierProvider);
    final unreadAsync = ref.watch(unreadCountProvider);
    final unansweredAsync = ref.watch(unansweredSurveyCountProvider);

    if (user == null) return const Drawer();

    final unread = unreadAsync.valueOrNull ?? 0;
    final unanswered = unansweredAsync.valueOrNull ?? 0;

    final titleSize = isEasy ? 16.0 : 14.0;

    return Drawer(
      child: Column(
        children: [
          // ─── ドロワーヘッダー ──────────────────────────────
          DrawerHeader(
            decoration: const BoxDecoration(color: AppColors.primary),
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    const Icon(Icons.article_rounded,
                        size: 20, color: Colors.white70),
                    const SizedBox(width: 6),
                    const Text(
                      '回覧板',
                      style: TextStyle(
                        color: Colors.white,
                        fontWeight: FontWeight.bold,
                        fontSize: 16,
                        letterSpacing: 2,
                      ),
                    ),
                  ],
                ),
                const Spacer(),
                Row(
                  children: [
                    CircleAvatar(
                      radius: isEasy ? 24 : 20,
                      backgroundColor: Colors.white.withOpacity(0.20),
                      child: Icon(Icons.person_rounded,
                          size: isEasy ? 26 : 22, color: Colors.white),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            user.name,
                            style: TextStyle(
                              color: Colors.white,
                              fontWeight: FontWeight.bold,
                              fontSize: isEasy ? 16 : 14,
                            ),
                            overflow: TextOverflow.ellipsis,
                          ),
                          const SizedBox(height: 3),
                          Text(
                            user.roleLabel,
                            style: TextStyle(
                              color: Colors.white70,
                              fontSize: isEasy ? 13 : 11,
                            ),
                          ),
                          if (user.associationName != null) ...[
                            const SizedBox(height: 2),
                            Text(
                              user.associationName!,
                              style: TextStyle(
                                color: Colors.white60,
                                fontSize: isEasy ? 12 : 10,
                              ),
                              overflow: TextOverflow.ellipsis,
                            ),
                          ],
                        ],
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),

          // ─── メニュー項目 ──────────────────────────────────
          Expanded(
            child: ListView(
              padding: EdgeInsets.zero,
              children: [
                // お知らせ
                _DrawerNavItem(
                  icon: Icons.notifications_rounded,
                  title: 'お知らせ',
                  badge: unread,
                  isEasy: isEasy,
                  titleSize: titleSize,
                  onTap: () {
                    Navigator.of(context).pop();
                    context.push('/notices');
                  },
                ),

                // 回覧物
                _DrawerNavItem(
                  icon: Icons.folder_rounded,
                  title: '回覧物',
                  badge: 0,
                  isEasy: isEasy,
                  titleSize: titleSize,
                  onTap: () {
                    Navigator.of(context).pop();
                    context.push('/files');
                  },
                ),

                // アンケート（system_admin 以外）
                if (user.role != 'system_admin')
                  _DrawerNavItem(
                    icon: Icons.poll_rounded,
                    title: 'アンケート',
                    badge: unanswered,
                    badgeColor: Colors.orange,
                    isEasy: isEasy,
                    titleSize: titleSize,
                    onTap: () {
                      Navigator.of(context).pop();
                      context.push('/surveys');
                    },
                  ),

                // アカウント管理（association_admin 以上）
                if (_isAdmin(user.role))
                  _DrawerNavItem(
                    icon: Icons.manage_accounts_rounded,
                    title: 'アカウント管理',
                    badge: 0,
                    isEasy: isEasy,
                    titleSize: titleSize,
                    onTap: () {
                      Navigator.of(context).pop();
                      context.push('/accounts');
                    },
                  ),

                // 自治会管理（system_admin のみ）
                if (user.role == 'system_admin')
                  _DrawerNavItem(
                    icon: Icons.home_work_rounded,
                    title: '自治会管理',
                    badge: 0,
                    isEasy: isEasy,
                    titleSize: titleSize,
                    onTap: () {
                      Navigator.of(context).pop();
                      context.push('/associations');
                    },
                  ),

                // 権限管理（system_admin のみ）
                if (user.role == 'system_admin')
                  _DrawerNavItem(
                    icon: Icons.admin_panel_settings_rounded,
                    title: '権限管理',
                    badge: 0,
                    isEasy: isEasy,
                    titleSize: titleSize,
                    onTap: () {
                      Navigator.of(context).pop();
                      context.push('/permissions');
                    },
                  ),

                const Divider(height: 1),

                // ログアウト
                ListTile(
                  leading: Icon(Icons.logout_rounded,
                      color: Colors.red[400],
                      size: isEasy ? 26 : 22),
                  title: Text(
                    'ログアウト',
                    style: TextStyle(
                      fontSize: titleSize,
                      color: Colors.red[600],
                    ),
                  ),
                  onTap: auth.isLoading
                      ? null
                      : () async {
                          final confirmed =
                              await _showLogoutDialog(context);
                          if (confirmed && context.mounted) {
                            await ref.read(authNotifierProvider).logout();
                          }
                        },
                ),
              ],
            ),
          ),
        ],
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

// ─── ドロワーメニュー項目 ────────────────────────────────────────────

class _DrawerNavItem extends StatelessWidget {
  final IconData icon;
  final String title;
  final int badge;
  final Color? badgeColor;
  final bool isEasy;
  final double titleSize;
  final VoidCallback onTap;

  const _DrawerNavItem({
    required this.icon,
    required this.title,
    required this.badge,
    required this.isEasy,
    required this.titleSize,
    required this.onTap,
    this.badgeColor,
  });

  @override
  Widget build(BuildContext context) {
    final bColor = badgeColor ?? Colors.red;

    return ListTile(
      leading: Stack(
        clipBehavior: Clip.none,
        children: [
          Icon(icon, color: AppColors.primary, size: isEasy ? 26 : 22),
          if (badge > 0)
            Positioned(
              top: -4,
              right: -6,
              child: Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
                decoration: BoxDecoration(
                  color: bColor,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  badge > 99 ? '99+' : '$badge',
                  style: const TextStyle(
                      color: Colors.white,
                      fontSize: 9,
                      fontWeight: FontWeight.bold),
                ),
              ),
            ),
        ],
      ),
      title: Row(
        children: [
          Text(title,
              style: TextStyle(
                  fontSize: titleSize, fontWeight: FontWeight.w500)),
          if (badge > 0) ...[
            const SizedBox(width: 8),
            Container(
              padding:
                  const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
              decoration: BoxDecoration(
                color: bColor,
                borderRadius: BorderRadius.circular(10),
              ),
              child: Text(
                badge > 99 ? '99+' : '$badge',
                style: TextStyle(
                    color: Colors.white,
                    fontSize: isEasy ? 11 : 10,
                    fontWeight: FontWeight.bold),
              ),
            ),
          ],
        ],
      ),
      onTap: onTap,
    );
  }
}

// ═══════════════════════════════════════════════════════════
// _UserInfoCard
// ═══════════════════════════════════════════════════════════

class _UserInfoCard extends StatelessWidget {
  final User user;
  final bool isEasy;
  const _UserInfoCard({required this.user, required this.isEasy});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                CircleAvatar(
                  radius: isEasy ? 30 : 26,
                  backgroundColor: AppColors.primary.withOpacity(0.12),
                  child: Icon(Icons.person_rounded,
                      size: isEasy ? 34 : 30, color: AppColors.primary),
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
                          fontSize: isEasy ? 20 : null,
                        ),
                      ),
                      const SizedBox(height: 4),
                      _RoleBadge(role: user.role, label: user.roleLabel),
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
    );
  }
}

// ═══════════════════════════════════════════════════════════
// セクションヘッダー
// ═══════════════════════════════════════════════════════════

class _SectionHeader extends StatelessWidget {
  final String title;
  final IconData icon;
  final Color color;
  final VoidCallback onViewAll;
  final bool isEasy;

  const _SectionHeader({
    required this.title,
    required this.icon,
    required this.color,
    required this.onViewAll,
    required this.isEasy,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          Icon(icon, size: isEasy ? 20 : 18, color: color),
          const SizedBox(width: 6),
          Text(
            title,
            style: TextStyle(
              fontWeight: FontWeight.bold,
              fontSize: isEasy ? 16 : 14,
              color: color,
            ),
          ),
          const Spacer(),
          TextButton(
            onPressed: onViewAll,
            style: TextButton.styleFrom(
              foregroundColor: color,
              padding:
                  const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            ),
            child: Text(
              'すべて見る',
              style: TextStyle(fontSize: isEasy ? 13 : 12),
            ),
          ),
        ],
      ),
    );
  }
}

// ═══════════════════════════════════════════════════════════
// 最新お知らせセクション
// ═══════════════════════════════════════════════════════════

class _RecentNoticesSection extends ConsumerWidget {
  final bool isEasy;
  const _RecentNoticesSection({required this.isEasy});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final noticesAsync = ref.watch(recentNoticesProvider);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionHeader(
          title: '最新お知らせ',
          icon: Icons.notifications_rounded,
          color: AppColors.primary,
          isEasy: isEasy,
          onViewAll: () => context.push('/notices'),
        ),
        Card(
          child: noticesAsync.when(
            loading: () => const Padding(
              padding: EdgeInsets.all(24),
              child: Center(child: CircularProgressIndicator()),
            ),
            error: (_, __) => _EmptyState(
              icon: Icons.notifications_off_outlined,
              message: '取得できませんでした',
              isEasy: isEasy,
            ),
            data: (notices) => notices.isEmpty
                ? _EmptyState(
                    icon: Icons.notifications_none_rounded,
                    message: 'お知らせはありません',
                    isEasy: isEasy,
                  )
                : Column(
                    children: notices
                        .asMap()
                        .entries
                        .map((e) => _NoticeTile(
                              notice: e.value,
                              isEasy: isEasy,
                              showDivider: e.key < notices.length - 1,
                            ))
                        .toList(),
                  ),
          ),
        ),
      ],
    );
  }
}

class _NoticeTile extends StatelessWidget {
  final NoticeModel notice;
  final bool isEasy;
  final bool showDivider;

  const _NoticeTile({
    required this.notice,
    required this.isEasy,
    required this.showDivider,
  });

  @override
  Widget build(BuildContext context) {
    final fmt = DateFormat('MM/dd');
    return Column(
      children: [
        ListTile(
          leading: Icon(
            notice.isPinned
                ? Icons.push_pin_rounded
                : Icons.notifications_rounded,
            color: notice.isPinned
                ? Colors.orange[700]
                : notice.isRead
                    ? Colors.grey[400]
                    : AppColors.primary,
            size: isEasy ? 22 : 20,
          ),
          title: Text(
            notice.title,
            style: TextStyle(
              fontWeight:
                  notice.isRead ? FontWeight.normal : FontWeight.bold,
              fontSize: isEasy ? 15 : 13,
              color: notice.isRead ? Colors.grey[700] : Colors.black87,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          subtitle: Text(
            fmt.format(notice.createdAt.toLocal()),
            style: TextStyle(
                fontSize: isEasy ? 12 : 11, color: Colors.grey[500]),
          ),
          trailing: notice.isRead
              ? null
              : Container(
                  width: 8,
                  height: 8,
                  decoration: const BoxDecoration(
                    color: Colors.red,
                    shape: BoxShape.circle,
                  ),
                ),
          dense: !isEasy,
          onTap: () => context.push('/notices/${notice.id}'),
        ),
        if (showDivider)
          const Divider(height: 1, indent: 16, endIndent: 16),
      ],
    );
  }
}

// ═══════════════════════════════════════════════════════════
// 最新回覧物セクション
// ═══════════════════════════════════════════════════════════

class _RecentFilesSection extends ConsumerWidget {
  final bool isEasy;
  const _RecentFilesSection({required this.isEasy});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final filesAsync = ref.watch(recentFilesProvider);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionHeader(
          title: '最新回覧物',
          icon: Icons.folder_rounded,
          color: const Color(0xFF2D6A4F),
          isEasy: isEasy,
          onViewAll: () => context.push('/files'),
        ),
        Card(
          child: filesAsync.when(
            loading: () => const Padding(
              padding: EdgeInsets.all(24),
              child: Center(child: CircularProgressIndicator()),
            ),
            error: (_, __) => _EmptyState(
              icon: Icons.folder_off_outlined,
              message: '取得できませんでした',
              isEasy: isEasy,
            ),
            data: (files) {
              final recent = files.take(3).toList();
              return recent.isEmpty
                  ? _EmptyState(
                      icon: Icons.folder_open_rounded,
                      message: '回覧物はありません',
                      isEasy: isEasy,
                    )
                  : Column(
                      children: recent
                          .asMap()
                          .entries
                          .map((e) => _FileTile(
                                file: e.value,
                                isEasy: isEasy,
                                showDivider: e.key < recent.length - 1,
                              ))
                          .toList(),
                    );
            },
          ),
        ),
      ],
    );
  }
}

class _FileTile extends StatelessWidget {
  final FileModel file;
  final bool isEasy;
  final bool showDivider;

  const _FileTile({
    required this.file,
    required this.isEasy,
    required this.showDivider,
  });

  IconData get _fileIcon {
    final mime = file.mimeType.toLowerCase();
    if (mime.contains('pdf')) return Icons.picture_as_pdf_rounded;
    if (mime.contains('image')) return Icons.image_rounded;
    if (mime.contains('word') || mime.contains('document')) {
      return Icons.description_rounded;
    }
    if (mime.contains('excel') || mime.contains('sheet')) {
      return Icons.table_chart_rounded;
    }
    return Icons.insert_drive_file_rounded;
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        ListTile(
          leading: Icon(
            _fileIcon,
            color: AppColors.primary,
            size: isEasy ? 22 : 20,
          ),
          title: Text(
            file.originalFilename,
            style: TextStyle(
              fontSize: isEasy ? 14 : 13,
              fontWeight: FontWeight.w500,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          subtitle: Text(
            '${file.year}年${file.month}月  ${file.fileSizeLabel}',
            style: TextStyle(
                fontSize: isEasy ? 12 : 11, color: Colors.grey[500]),
          ),
          trailing: Icon(Icons.chevron_right_rounded,
              size: isEasy ? 22 : 18, color: Colors.grey[400]),
          dense: !isEasy,
          onTap: () => context.push('/files/preview', extra: file),
        ),
        if (showDivider)
          const Divider(height: 1, indent: 16, endIndent: 16),
      ],
    );
  }
}

// ═══════════════════════════════════════════════════════════
// 未回答アンケートセクション
// ═══════════════════════════════════════════════════════════

class _RecentSurveysSection extends ConsumerWidget {
  final bool isEasy;
  const _RecentSurveysSection({required this.isEasy});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final surveysAsync = ref.watch(recentSurveysProvider);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionHeader(
          title: '未回答アンケート',
          icon: Icons.poll_rounded,
          color: Colors.teal[700]!,
          isEasy: isEasy,
          onViewAll: () => context.push('/surveys'),
        ),
        Card(
          child: surveysAsync.when(
            loading: () => const Padding(
              padding: EdgeInsets.all(24),
              child: Center(child: CircularProgressIndicator()),
            ),
            error: (_, __) => _EmptyState(
              icon: Icons.poll_outlined,
              message: '取得できませんでした',
              isEasy: isEasy,
            ),
            data: (surveys) => surveys.isEmpty
                ? _EmptyState(
                    icon: Icons.check_circle_outline_rounded,
                    message: '未回答のアンケートはありません',
                    isEasy: isEasy,
                  )
                : Column(
                    children: surveys
                        .asMap()
                        .entries
                        .map((e) => _SurveyTile(
                              survey: e.value,
                              isEasy: isEasy,
                              showDivider:
                                  e.key < surveys.length - 1,
                            ))
                        .toList(),
                  ),
          ),
        ),
      ],
    );
  }
}

class _SurveyTile extends StatelessWidget {
  final SurveyModel survey;
  final bool isEasy;
  final bool showDivider;

  const _SurveyTile({
    required this.survey,
    required this.isEasy,
    required this.showDivider,
  });

  @override
  Widget build(BuildContext context) {
    final now = DateTime.now();
    final diff = survey.expiresAt.difference(now);
    final isUrgent = diff.inHours < 48;
    final fmt = DateFormat('MM/dd HH:mm');

    return Column(
      children: [
        ListTile(
          leading: Icon(
            Icons.poll_rounded,
            color: isUrgent ? Colors.orange[700] : Colors.teal[600],
            size: isEasy ? 22 : 20,
          ),
          title: Text(
            survey.title,
            style: TextStyle(
              fontSize: isEasy ? 14 : 13,
              fontWeight: FontWeight.w600,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          subtitle: Text(
            '期限: ${fmt.format(survey.expiresAt.toLocal())}',
            style: TextStyle(
              fontSize: isEasy ? 12 : 11,
              color: isUrgent ? Colors.orange[700] : Colors.grey[500],
              fontWeight:
                  isUrgent ? FontWeight.w600 : FontWeight.normal,
            ),
          ),
          trailing: isUrgent
              ? Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: Colors.orange[50],
                    borderRadius: BorderRadius.circular(6),
                    border: Border.all(color: Colors.orange[300]!),
                  ),
                  child: Text(
                    '期限近い',
                    style: TextStyle(
                      fontSize: isEasy ? 11 : 10,
                      color: Colors.orange[700],
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                )
              : Icon(Icons.chevron_right_rounded,
                  size: isEasy ? 22 : 18, color: Colors.grey[400]),
          dense: !isEasy,
          onTap: () => context.push('/surveys/${survey.id}'),
        ),
        if (showDivider)
          const Divider(height: 1, indent: 16, endIndent: 16),
      ],
    );
  }
}

// ═══════════════════════════════════════════════════════════
// 空状態
// ═══════════════════════════════════════════════════════════

class _EmptyState extends StatelessWidget {
  final IconData icon;
  final String message;
  final bool isEasy;

  const _EmptyState({
    required this.icon,
    required this.message,
    required this.isEasy,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 20, horizontal: 16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(icon, size: isEasy ? 20 : 18, color: Colors.grey[400]),
          const SizedBox(width: 8),
          Text(
            message,
            style: TextStyle(
                fontSize: isEasy ? 14 : 13, color: Colors.grey[500]),
          ),
        ],
      ),
    );
  }
}

// ═══════════════════════════════════════════════════════════
// 共通ヘルパーウィジェット
// ═══════════════════════════════════════════════════════════

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
        Icon(icon, size: isEasy ? 20 : 18, color: AppColors.primary),
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
                fontSize: isEasy ? 16 : null,
              ),
            ),
          ],
        ),
      ],
    );
  }
}

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
