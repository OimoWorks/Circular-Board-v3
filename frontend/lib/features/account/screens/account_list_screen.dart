import 'package:circular_board/features/account/data/models/account_model.dart';
import 'package:circular_board/features/account/providers/account_provider.dart';
import 'package:circular_board/features/association/data/models/association_model.dart';
import 'package:circular_board/features/association/providers/association_provider.dart';
import 'package:circular_board/features/auth/presentation/auth_provider.dart';
import 'package:circular_board/main.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class AccountListScreen extends ConsumerStatefulWidget {
  const AccountListScreen({super.key});

  @override
  ConsumerState<AccountListScreen> createState() => _AccountListScreenState();
}

class _AccountListScreenState extends ConsumerState<AccountListScreen> {
  String _searchQuery = '';
  String _roleFilter = '';
  String? _selectedAssociationId;

  @override
  void initState() {
    super.initState();
    Future.microtask(_loadAccounts);
  }

  String get _callerRole =>
      ref.read(currentUserProvider)?.role ?? '';

  bool get _isSystemAdmin => _callerRole == 'system_admin';

  void _loadAccounts() {
    ref
        .read(accountNotifierProvider)
        .load(associationId: _selectedAssociationId);
  }

  @override
  Widget build(BuildContext context) {
    final notifier = ref.watch(accountNotifierProvider);
    final theme = Theme.of(context);

    List<AccountModel> displayed = notifier.accounts.where((a) {
      final q = _searchQuery.toLowerCase();
      final matchQuery = q.isEmpty ||
          a.name.toLowerCase().contains(q) ||
          a.email.toLowerCase().contains(q);
      final matchRole =
          _roleFilter.isEmpty || a.role == _roleFilter;
      return matchQuery && matchRole;
    }).toList();

    return Scaffold(
      appBar: AppBar(
        title: const Text('アカウント管理'),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () async {
          final result = await context.push<bool>(
            '/accounts/create',
            extra: _isSystemAdmin ? _selectedAssociationId : null,
          );
          if (result == true && mounted) _loadAccounts();
        },
        icon: const Icon(Icons.person_add_rounded),
        label: const Text('新規登録'),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
      ),
      body: Column(
        children: [
          // ─── フィルタバー ─────────────────────────────────────
          Container(
            color: Colors.grey[50],
            padding: const EdgeInsets.fromLTRB(12, 10, 12, 8),
            child: Column(
              children: [
                // system_admin: 自治会フィルタ
                if (_isSystemAdmin) _buildAssociationFilter(),
                const SizedBox(height: 8),
                Row(
                  children: [
                    // 検索ボックス
                    Expanded(
                      child: TextField(
                        decoration: InputDecoration(
                          hintText: '名前・メールで検索',
                          prefixIcon: const Icon(Icons.search_rounded, size: 20),
                          isDense: true,
                          contentPadding:
                              const EdgeInsets.symmetric(vertical: 10),
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(8),
                            borderSide: BorderSide(color: Colors.grey[300]!),
                          ),
                        ),
                        onChanged: (v) => setState(() => _searchQuery = v),
                      ),
                    ),
                    const SizedBox(width: 8),
                    // ロールフィルタ
                    DropdownButtonHideUnderline(
                      child: Container(
                        padding: const EdgeInsets.symmetric(horizontal: 10),
                        decoration: BoxDecoration(
                          border: Border.all(color: Colors.grey[300]!),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: DropdownButton<String>(
                          value: _roleFilter,
                          isDense: true,
                          items: const [
                            DropdownMenuItem(value: '', child: Text('全ロール')),
                            DropdownMenuItem(
                                value: 'user', child: Text('一般ユーザー')),
                            DropdownMenuItem(
                                value: 'association_admin',
                                child: Text('自治会管理者')),
                            DropdownMenuItem(
                                value: 'system_admin',
                                child: Text('システム管理者')),
                          ],
                          onChanged: (v) =>
                              setState(() => _roleFilter = v ?? ''),
                        ),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),

          // ─── リスト ───────────────────────────────────────────
          Expanded(
            child: notifier.isLoading
                ? const Center(child: CircularProgressIndicator())
                : notifier.errorMessage != null
                    ? Center(
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(
                              notifier.errorMessage!,
                              style: TextStyle(color: Colors.red[700]),
                            ),
                            const SizedBox(height: 12),
                            FilledButton.icon(
                              onPressed: _loadAccounts,
                              icon: const Icon(Icons.refresh_rounded),
                              label: const Text('再読み込み'),
                            ),
                          ],
                        ),
                      )
                    : displayed.isEmpty
                        ? Center(
                            child: Text(
                              'アカウントがありません',
                              style: TextStyle(color: Colors.grey[500]),
                            ),
                          )
                        : ListView.separated(
                            padding: const EdgeInsets.symmetric(
                                vertical: 8, horizontal: 12),
                            itemCount: displayed.length,
                            separatorBuilder: (_, __) =>
                                const SizedBox(height: 4),
                            itemBuilder: (ctx, i) => _AccountCard(
                              account: displayed[i],
                              onEdit: () async {
                                final result = await context.push<bool>(
                                  '/accounts/${displayed[i].id}/edit',
                                  extra: displayed[i],
                                );
                                if (result == true && mounted) _loadAccounts();
                              },
                              onToggleActive: () =>
                                  _toggleActive(displayed[i], notifier),
                              onDelete: () =>
                                  _confirmDelete(context, displayed[i], notifier),
                            ),
                          ),
          ),
        ],
      ),
    );
  }

  Widget _buildAssociationFilter() {
    final assocAsync = ref.watch(allAssociationsProvider);
    return assocAsync.when(
      data: (assocs) => Row(
        children: [
          const Icon(Icons.home_work_rounded,
              size: 18, color: AppColors.primary),
          const SizedBox(width: 6),
          Expanded(
            child: DropdownButtonHideUnderline(
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 10),
                decoration: BoxDecoration(
                  border: Border.all(color: Colors.grey[300]!),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: DropdownButton<String?>(
                  value: _selectedAssociationId,
                  isDense: true,
                  isExpanded: true,
                  hint: const Text('全自治会'),
                  items: [
                    const DropdownMenuItem<String?>(
                        value: null, child: Text('全自治会')),
                    ...assocs.map((a) => DropdownMenuItem<String?>(
                          value: a.id,
                          child: Text('${a.name}（${a.code}）'),
                        )),
                  ],
                  onChanged: (v) {
                    setState(() => _selectedAssociationId = v);
                    _loadAccounts();
                  },
                ),
              ),
            ),
          ),
        ],
      ),
      loading: () => const LinearProgressIndicator(),
      error: (_, __) => const SizedBox.shrink(),
    );
  }

  Future<void> _toggleActive(
      AccountModel account, AccountNotifier notifier) async {
    final ok = account.isActive
        ? await notifier.deactivate(account.id)
        : await notifier.activate(account.id);
    if (!ok && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(notifier.errorMessage ?? 'エラーが発生しました'),
          backgroundColor: Colors.red,
        ),
      );
    }
  }

  Future<void> _confirmDelete(
    BuildContext context,
    AccountModel account,
    AccountNotifier notifier,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('アカウントの削除'),
        content: Text('「${account.name}」を無効化しますか？'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('キャンセル'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('無効化'),
          ),
        ],
      ),
    );
    if (confirmed == true && mounted) {
      final ok = await notifier.delete(account.id);
      if (!ok && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(notifier.errorMessage ?? 'エラーが発生しました'),
            backgroundColor: Colors.red,
          ),
        );
      }
    }
  }
}

// ─── アカウントカード ─────────────────────────────────────────────

class _AccountCard extends StatelessWidget {
  final AccountModel account;
  final VoidCallback onEdit;
  final VoidCallback onToggleActive;
  final VoidCallback onDelete;

  const _AccountCard({
    required this.account,
    required this.onEdit,
    required this.onToggleActive,
    required this.onDelete,
  });

  Color _roleColor() {
    switch (account.role) {
      case 'system_admin':
        return const Color(0xFFB91C1C);
      case 'association_admin':
        return AppColors.primaryDark;
      default:
        return AppColors.primary;
    }
  }

  Color _roleBg() {
    switch (account.role) {
      case 'system_admin':
        return const Color(0xFFFFE4E4);
      case 'association_admin':
        return const Color(0xFFD8F0E6);
      default:
        return AppColors.primary.withOpacity(0.1);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(
          color: account.isActive
              ? Colors.grey[200]!
              : Colors.orange[200]!,
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        child: Row(
          children: [
            // アバター
            CircleAvatar(
              radius: 22,
              backgroundColor: account.isActive
                  ? AppColors.primary.withOpacity(0.12)
                  : Colors.grey[200],
              child: Icon(
                Icons.person_rounded,
                color: account.isActive ? AppColors.primary : Colors.grey,
                size: 22,
              ),
            ),
            const SizedBox(width: 12),
            // 情報
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Flexible(
                        child: Text(
                          account.name,
                          style: TextStyle(
                            fontWeight: FontWeight.bold,
                            fontSize: 15,
                            color: account.isActive
                                ? Colors.black87
                                : Colors.grey,
                          ),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const SizedBox(width: 6),
                      Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 7, vertical: 2),
                        decoration: BoxDecoration(
                          color: _roleBg(),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Text(
                          account.roleLabel,
                          style: TextStyle(
                            fontSize: 10,
                            fontWeight: FontWeight.w600,
                            color: _roleColor(),
                          ),
                        ),
                      ),
                      if (!account.isActive) ...[
                        const SizedBox(width: 4),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 6, vertical: 2),
                          decoration: BoxDecoration(
                            color: Colors.orange[50],
                            borderRadius: BorderRadius.circular(8),
                            border: Border.all(color: Colors.orange[200]!),
                          ),
                          child: Text(
                            '無効',
                            style: TextStyle(
                                fontSize: 10, color: Colors.orange[800]),
                          ),
                        ),
                      ],
                    ],
                  ),
                  const SizedBox(height: 2),
                  Text(
                    account.email,
                    style: TextStyle(
                        fontSize: 12,
                        color: account.isActive
                            ? Colors.grey[600]
                            : Colors.grey[400]),
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            // 操作ボタン
            PopupMenuButton<String>(
              icon: const Icon(Icons.more_vert_rounded, size: 20),
              onSelected: (v) {
                switch (v) {
                  case 'edit':
                    onEdit();
                  case 'toggle':
                    onToggleActive();
                  case 'delete':
                    onDelete();
                }
              },
              itemBuilder: (_) => [
                const PopupMenuItem(
                  value: 'edit',
                  child: ListTile(
                    leading: Icon(Icons.edit_rounded),
                    title: Text('編集'),
                    dense: true,
                  ),
                ),
                PopupMenuItem(
                  value: 'toggle',
                  child: ListTile(
                    leading: Icon(account.isActive
                        ? Icons.toggle_off_rounded
                        : Icons.toggle_on_rounded),
                    title:
                        Text(account.isActive ? '無効化' : '有効化'),
                    dense: true,
                  ),
                ),
                const PopupMenuItem(
                  value: 'delete',
                  child: ListTile(
                    leading: Icon(Icons.person_off_rounded,
                        color: Colors.red),
                    title: Text('削除', style: TextStyle(color: Colors.red)),
                    dense: true,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
