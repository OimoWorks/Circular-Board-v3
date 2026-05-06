import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../main.dart';
import '../data/models/permission_model.dart';
import '../providers/permission_provider.dart';

class PermissionScreen extends ConsumerStatefulWidget {
  const PermissionScreen({super.key});

  @override
  ConsumerState<PermissionScreen> createState() => _PermissionScreenState();
}

class _PermissionScreenState extends ConsumerState<PermissionScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(permissionNotifierProvider).load();
    });
  }

  @override
  Widget build(BuildContext context) {
    final notifier = ref.watch(permissionNotifierProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('権限管理'),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
        actions: [
          TextButton.icon(
            onPressed: notifier.isSubmitting ? null : () => _save(context),
            icon: notifier.isSubmitting
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(
                        strokeWidth: 2, color: Colors.white))
                : const Icon(Icons.save_rounded, color: Colors.white),
            label: const Text('保存', style: TextStyle(color: Colors.white)),
          ),
          IconButton(
            icon: const Icon(Icons.assignment_rounded),
            tooltip: '緊急任命',
            onPressed: () => context.push('/permissions/emergency-appointment'),
          ),
        ],
      ),
      body: _buildBody(notifier),
    );
  }

  Widget _buildBody(PermissionNotifier notifier) {
    if (notifier.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (notifier.errorMessage != null && notifier.matrix == null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(notifier.errorMessage!,
                style: const TextStyle(color: Colors.red)),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: () => notifier.load(),
              child: const Text('再読み込み'),
            ),
          ],
        ),
      );
    }

    final matrix = notifier.matrix;
    if (matrix == null) return const SizedBox.shrink();

    // system_admin を除いた編集可能ロール
    final editableRoles =
        matrix.roles.where((r) => r.name != 'system_admin').toList();

    return Column(
      children: [
        if (notifier.errorMessage != null)
          _Banner(message: notifier.errorMessage!, isError: true,
              onClose: notifier.clearMessages),
        if (notifier.successMessage != null)
          _Banner(message: notifier.successMessage!, isError: false,
              onClose: notifier.clearMessages),
        Expanded(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: SingleChildScrollView(
              child: _PermissionTable(
                roles: editableRoles,
                features: matrix.features,
                notifier: notifier,
              ),
            ),
          ),
        ),
      ],
    );
  }

  Future<void> _save(BuildContext context) async {
    final success = await ref.read(permissionNotifierProvider).save();
    if (!success && context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
              ref.read(permissionNotifierProvider).errorMessage ?? '更新に失敗しました'),
          backgroundColor: Colors.red,
        ),
      );
    }
  }
}

// ─── 権限マトリクステーブル ────────────────────────────────────────────

class _PermissionTable extends StatelessWidget {
  final List<RoleModel> roles;
  final List<FeatureModel> features;
  final PermissionNotifier notifier;

  const _PermissionTable({
    required this.roles,
    required this.features,
    required this.notifier,
  });

  @override
  Widget build(BuildContext context) {
    const headerColor = Color(0xFF455A64);
    const borderColor = Color(0xFFCFD8DC);

    return Table(
      border: TableBorder.all(color: borderColor, width: 0.5),
      defaultColumnWidth: const IntrinsicColumnWidth(),
      children: [
        // ─── ヘッダー行1: 機能名 ────────────────────────────────
        TableRow(
          decoration: const BoxDecoration(color: headerColor),
          children: [
            // 左上の空セル
            _HeaderCell(text: '機能 \\ ロール', width: 120),
            for (final role in roles)
              _HeaderCell(text: role.displayName, width: 160),
          ],
        ),
        // ─── 行: 各機能 ─────────────────────────────────────────
        for (final feature in features)
          TableRow(
            children: [
              // 機能名セル
              Container(
                width: 120,
                padding: const EdgeInsets.symmetric(
                    horizontal: 10, vertical: 8),
                color: const Color(0xFFECEFF1),
                child: Text(
                  feature.displayName,
                  style: const TextStyle(
                      fontWeight: FontWeight.w600, fontSize: 13),
                ),
              ),
              // ロールごとの権限チェックボックス群
              for (final role in roles)
                _PermissionCell(
                  role: role,
                  feature: feature,
                  notifier: notifier,
                ),
            ],
          ),
      ],
    );
  }
}

class _HeaderCell extends StatelessWidget {
  final String text;
  final double width;
  const _HeaderCell({required this.text, required this.width});

  @override
  Widget build(BuildContext context) => Container(
        width: width,
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
        child: Text(
          text,
          style: const TextStyle(
              color: Colors.white, fontWeight: FontWeight.bold, fontSize: 12),
          textAlign: TextAlign.center,
        ),
      );
}

class _PermissionCell extends StatelessWidget {
  final RoleModel role;
  final FeatureModel feature;
  final PermissionNotifier notifier;

  const _PermissionCell({
    required this.role,
    required this.feature,
    required this.notifier,
  });

  @override
  Widget build(BuildContext context) {
    final perm = notifier.getEdited(role.id, feature.id);

    // 権限がない場合はデフォルト（全false）の一時エントリを使う
    final canView = perm?.canView ?? false;
    final canCreate = perm?.canCreate ?? false;
    final canEdit = perm?.canEdit ?? false;
    final canDelete = perm?.canDelete ?? false;
    final scope = perm?.scope ?? 'own_association';

    void update(RolePermission updated) => notifier.updatePermission(updated);

    RolePermission current() => perm ??
        RolePermission(
          id: '',
          roleId: role.id,
          featureId: feature.id,
          canView: false,
          canCreate: false,
          canEdit: false,
          canDelete: false,
          scope: 'own_association',
        );

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _CheckRow(
            label: '閲覧',
            value: canView,
            onChanged: (v) => update(current().copyWith(canView: v)),
          ),
          _CheckRow(
            label: '作成',
            value: canCreate,
            onChanged: (v) => update(current().copyWith(canCreate: v)),
          ),
          _CheckRow(
            label: '編集',
            value: canEdit,
            onChanged: (v) => update(current().copyWith(canEdit: v)),
          ),
          _CheckRow(
            label: '削除',
            value: canDelete,
            onChanged: (v) => update(current().copyWith(canDelete: v)),
          ),
          const SizedBox(height: 4),
          // スコープ選択
          _ScopeDropdown(
            value: scope,
            onChanged: (v) => update(current().copyWith(scope: v)),
          ),
        ],
      ),
    );
  }
}

class _CheckRow extends StatelessWidget {
  final String label;
  final bool value;
  final ValueChanged<bool> onChanged;

  const _CheckRow({
    required this.label,
    required this.value,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) => SizedBox(
        height: 28,
        child: Row(
          children: [
            SizedBox(
              width: 24,
              height: 24,
              child: Checkbox(
                value: value,
                onChanged: (v) => onChanged(v ?? false),
                materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                visualDensity: VisualDensity.compact,
              ),
            ),
            const SizedBox(width: 4),
            Text(label, style: const TextStyle(fontSize: 12)),
          ],
        ),
      );
}

class _ScopeDropdown extends StatelessWidget {
  final String value;
  final ValueChanged<String> onChanged;

  const _ScopeDropdown({required this.value, required this.onChanged});

  @override
  Widget build(BuildContext context) => DropdownButton<String>(
        value: value,
        isDense: true,
        style: const TextStyle(fontSize: 11, color: Colors.black87),
        items: const [
          DropdownMenuItem(value: 'own_association', child: Text('自自治体')),
          DropdownMenuItem(value: 'all', child: Text('全自治体')),
        ],
        onChanged: (v) => onChanged(v ?? 'own_association'),
      );
}

// ─── バナー ──────────────────────────────────────────────────────────

class _Banner extends StatelessWidget {
  final String message;
  final bool isError;
  final VoidCallback onClose;

  const _Banner({
    required this.message,
    required this.isError,
    required this.onClose,
  });

  @override
  Widget build(BuildContext context) => Container(
        width: double.infinity,
        color: isError ? Colors.red[100] : Colors.green[100],
        padding:
            const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        child: Row(
          children: [
            Icon(
              isError ? Icons.error_outline : Icons.check_circle_outline,
              color: isError ? Colors.red[700] : Colors.green[700],
              size: 18,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                message,
                style: TextStyle(
                  color: isError ? Colors.red[900] : Colors.green[900],
                  fontSize: 13,
                ),
              ),
            ),
            IconButton(
              icon: const Icon(Icons.close, size: 18),
              onPressed: onClose,
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(),
            ),
          ],
        ),
      );
}
