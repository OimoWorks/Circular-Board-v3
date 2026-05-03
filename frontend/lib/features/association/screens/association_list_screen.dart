import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../../main.dart';
import '../../data/models/association_model.dart';
import '../../providers/association_provider.dart';

class AssociationListScreen extends ConsumerStatefulWidget {
  const AssociationListScreen({super.key});

  @override
  ConsumerState<AssociationListScreen> createState() =>
      _AssociationListScreenState();
}

class _AssociationListScreenState
    extends ConsumerState<AssociationListScreen> {
  @override
  void initState() {
    super.initState();
    Future.microtask(
        () => ref.read(associationManagementNotifierProvider).load());
  }

  @override
  Widget build(BuildContext context) {
    final notifier = ref.watch(associationManagementNotifierProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('自治会管理'),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () async {
          final result =
              await context.push<bool>('/associations/create');
          if (result == true && mounted) {
            ref.read(associationManagementNotifierProvider).load();
          }
        },
        icon: const Icon(Icons.add_rounded),
        label: const Text('自治会登録'),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
      ),
      body: notifier.isLoading
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
                        onPressed: () => ref
                            .read(associationManagementNotifierProvider)
                            .load(),
                        icon: const Icon(Icons.refresh_rounded),
                        label: const Text('再読み込み'),
                      ),
                    ],
                  ),
                )
              : notifier.associations.isEmpty
                  ? Center(
                      child: Text(
                        '自治会がありません',
                        style: TextStyle(color: Colors.grey[500]),
                      ),
                    )
                  : ListView.separated(
                      padding: const EdgeInsets.symmetric(
                          vertical: 8, horizontal: 12),
                      itemCount: notifier.associations.length,
                      separatorBuilder: (_, __) => const SizedBox(height: 4),
                      itemBuilder: (ctx, i) => _AssociationCard(
                        association: notifier.associations[i],
                        onEdit: () async {
                          final result = await context.push<bool>(
                            '/associations/${notifier.associations[i].id}/edit',
                            extra: notifier.associations[i],
                          );
                          if (result == true && mounted) {
                            ref
                                .read(associationManagementNotifierProvider)
                                .load();
                          }
                        },
                        onToggleActive: () =>
                            _toggleActive(notifier.associations[i], notifier),
                        onDelete: () => _confirmDelete(
                            context, notifier.associations[i], notifier),
                      ),
                    ),
    );
  }

  Future<void> _toggleActive(
      AssociationDetail assoc, AssociationManagementNotifier notifier) async {
    final ok = assoc.isActive
        ? await notifier.deactivate(assoc.id)
        : await notifier.activate(assoc.id);
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
    AssociationDetail assoc,
    AssociationManagementNotifier notifier,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('自治会の削除'),
        content: Text(
            '「${assoc.name}」を無効化しますか？\n関連するアカウントはログインできなくなります。'),
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
      final ok = await notifier.delete(assoc.id);
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

// ─── 自治会カード ─────────────────────────────────────────────────

class _AssociationCard extends StatelessWidget {
  final AssociationDetail association;
  final VoidCallback onEdit;
  final VoidCallback onToggleActive;
  final VoidCallback onDelete;

  const _AssociationCard({
    required this.association,
    required this.onEdit,
    required this.onToggleActive,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final dateLabel =
        DateFormat('yyyy/MM/dd').format(association.createdAt.toLocal());

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(
          color: association.isActive
              ? Colors.grey[200]!
              : Colors.orange[200]!,
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        child: Row(
          children: [
            // アイコン
            Container(
              width: 44,
              height: 44,
              decoration: BoxDecoration(
                color: association.isActive
                    ? AppColors.primary.withOpacity(0.10)
                    : Colors.grey[100],
                borderRadius: BorderRadius.circular(10),
              ),
              child: Icon(
                Icons.home_work_rounded,
                color: association.isActive ? AppColors.primary : Colors.grey,
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
                          association.name,
                          style: TextStyle(
                            fontWeight: FontWeight.bold,
                            fontSize: 15,
                            color: association.isActive
                                ? Colors.black87
                                : Colors.grey,
                          ),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const SizedBox(width: 6),
                      // コードバッジ
                      Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 7, vertical: 2),
                        decoration: BoxDecoration(
                          color: association.isActive
                              ? AppColors.primary.withOpacity(0.08)
                              : Colors.grey[100],
                          borderRadius: BorderRadius.circular(6),
                        ),
                        child: Text(
                          association.code,
                          style: TextStyle(
                            fontFamily: 'monospace',
                            fontSize: 11,
                            fontWeight: FontWeight.w600,
                            color: association.isActive
                                ? AppColors.primaryDark
                                : Colors.grey,
                          ),
                        ),
                      ),
                      if (!association.isActive) ...[
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
                    '登録日: $dateLabel',
                    style: TextStyle(
                        fontSize: 11,
                        color: association.isActive
                            ? Colors.grey[500]
                            : Colors.grey[400]),
                  ),
                ],
              ),
            ),
            // 操作
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
                    leading: Icon(association.isActive
                        ? Icons.toggle_off_rounded
                        : Icons.toggle_on_rounded),
                    title:
                        Text(association.isActive ? '無効化' : '有効化'),
                    dense: true,
                  ),
                ),
                const PopupMenuItem(
                  value: 'delete',
                  child: ListTile(
                    leading:
                        Icon(Icons.delete_rounded, color: Colors.red),
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
