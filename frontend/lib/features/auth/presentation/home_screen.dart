import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../main.dart';
import '../../notice/providers/notice_provider.dart';
import '../../survey/providers/survey_provider.dart';
import 'auth_provider.dart';

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
                          radius: isEasy ? 30 : 26,
                          backgroundColor:
                              AppColors.primary.withOpacity(0.12),
                          child: Icon(Icons.person_rounded,
                              size: isEasy ? 34 : 30,
                              color: AppColors.primary),
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
            SizedBox(height: isEasy ? 16 : 12),

            // ─── メニュー：お知らせ（未読バッジ付き） ────────────
            _NoticeMenuCard(isEasy: isEasy),
            SizedBox(height: isEasy ? 12 : 10),

            // ─── メニュー：回覧物 ────────────────────────────────
            _MenuCard(
              icon: Icons.folder_rounded,
              title: '回覧物',
              subtitle: '回覧資料のアップロード、プレビュー、ダウンロード',
              onTap: () => context.push('/files'),
              isEasy: isEasy,
            ),
            SizedBox(height: isEasy ? 12 : 10),

            // ─── メニュー：アンケート（未回答バッジ付き） ─────────
            if (user.role != 'system_admin') ...[
              _SurveyMenuCard(isEasy: isEasy),
              SizedBox(height: isEasy ? 12 : 10),
            ],

            // ─── アカウント管理（association_admin 以上） ─────────
            if (_isAdmin(user.role)) ...[
              _MenuCard(
                icon: Icons.manage_accounts_rounded,
                title: 'アカウント管理',
                subtitle: '自治会メンバーのアカウントを登録・管理する',
                onTap: () => context.push('/accounts'),
                isEasy: isEasy,
                color: const Color(0xFF2D6A4F),
              ),
              SizedBox(height: isEasy ? 12 : 10),
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
            ],
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
    final iconSize = isEasy ? 56.0 : 46.0;
    final iconInner = isEasy ? 30.0 : 24.0;
    final titleSize = isEasy ? 18.0 : 15.0;
    final subtitleSize = isEasy ? 14.0 : 12.0;
    final padding = isEasy ? 22.0 : 18.0;

    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => context.push('/notices'),
        child: Padding(
          padding: EdgeInsets.all(padding),
          child: Row(
            children: [
              Stack(
                clipBehavior: Clip.none,
                children: [
                  Container(
                    width: iconSize,
                    height: iconSize,
                    decoration: BoxDecoration(
                      color: AppColors.primary.withOpacity(0.10),
                      borderRadius: BorderRadius.circular(isEasy ? 14 : 11),
                    ),
                    child: Icon(Icons.notifications_rounded,
                        color: AppColors.primary, size: iconInner),
                  ),
                  // 未読バッジ
                  unreadAsync.when(
                    data: (count) => count > 0
                        ? Positioned(
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
                                style: TextStyle(
                                  color: Colors.white,
                                  fontSize: isEasy ? 12 : 10,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                            ),
                          )
                        : const SizedBox.shrink(),
                    loading: () => const SizedBox.shrink(),
                    error: (_, __) => const SizedBox.shrink(),
                  ),
                ],
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
                            fontSize: titleSize,
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
                                    style: TextStyle(
                                      color: Colors.white,
                                      fontSize: isEasy ? 12 : 11,
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
                          fontSize: subtitleSize,
                          color: Colors.grey[600]),
                    ),
                  ],
                ),
              ),
              Icon(Icons.chevron_right_rounded,
                  color: AppColors.primary, size: isEasy ? 28 : 24),
            ],
          ),
        ),
      ),
    );
  }
}

// ─── アンケートメニューカード（未回答件数バッジ付き） ─────────────

class _SurveyMenuCard extends ConsumerWidget {
  final bool isEasy;
  const _SurveyMenuCard({required this.isEasy});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final unansweredAsync = ref.watch(unansweredSurveyCountProvider);
    final count = unansweredAsync.valueOrNull ?? 0;
    final iconSize = isEasy ? 56.0 : 46.0;
    final iconInner = isEasy ? 30.0 : 24.0;
    final titleSize = isEasy ? 18.0 : 15.0;
    final subtitleSize = isEasy ? 14.0 : 12.0;
    final padding = isEasy ? 22.0 : 18.0;

    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => context.push('/surveys'),
        child: Padding(
          padding: EdgeInsets.all(padding),
          child: Row(
            children: [
              Stack(
                clipBehavior: Clip.none,
                children: [
                  Container(
                    width: iconSize,
                    height: iconSize,
                    decoration: BoxDecoration(
                      color: Colors.teal.withOpacity(0.10),
                      borderRadius: BorderRadius.circular(isEasy ? 14 : 11),
                    ),
                    child: Icon(Icons.poll_rounded,
                        color: Colors.teal, size: iconInner),
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
                          style: TextStyle(
                            color: Colors.white,
                            fontSize: isEasy ? 12 : 10,
                            fontWeight: FontWeight.bold,
                          ),
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
                    Row(
                      children: [
                        Text(
                          'アンケート',
                          style: TextStyle(
                            fontWeight: FontWeight.bold,
                            fontSize: titleSize,
                          ),
                        ),
                        if (count > 0) ...[
                          const SizedBox(width: 8),
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 8, vertical: 2),
                            decoration: BoxDecoration(
                              color: Colors.orange,
                              borderRadius: BorderRadius.circular(10),
                            ),
                            child: Text(
                              '未回答 $count件',
                              style: TextStyle(
                                color: Colors.white,
                                fontSize: isEasy ? 12 : 11,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                          ),
                        ],
                      ],
                    ),
                    const SizedBox(height: 2),
                    Text(
                      count > 0
                          ? '未回答が $count 件あります'
                          : 'アンケートへの回答・確認',
                      style: TextStyle(
                        fontSize: subtitleSize,
                        color: count > 0
                            ? Colors.orange[700]
                            : Colors.grey[600],
                      ),
                    ),
                  ],
                ),
              ),
              Icon(Icons.chevron_right_rounded,
                  color: Colors.teal, size: isEasy ? 28 : 24),
            ],
          ),
        ),
      ),
    );
  }
}

// ─── メニューカード（汎用） ──────────────────────────────────────

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
    final iconSize = isEasy ? 56.0 : 46.0;
    final iconInner = isEasy ? 30.0 : 24.0;
    final titleSize = isEasy ? 18.0 : 15.0;
    final subtitleSize = isEasy ? 14.0 : 12.0;
    final padding = isEasy ? 22.0 : 18.0;

    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: EdgeInsets.all(padding),
          child: Row(
            children: [
              Container(
                width: iconSize,
                height: iconSize,
                decoration: BoxDecoration(
                  color: c.withOpacity(0.10),
                  borderRadius: BorderRadius.circular(isEasy ? 14 : 11),
                ),
                child: Icon(icon, color: c, size: iconInner),
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
                        fontSize: titleSize,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      subtitle,
                      style: TextStyle(
                          fontSize: subtitleSize,
                          color: Colors.grey[600]),
                    ),
                  ],
                ),
              ),
              Icon(Icons.chevron_right_rounded,
                  color: c, size: isEasy ? 28 : 24),
            ],
          ),
        ),
      ),
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
