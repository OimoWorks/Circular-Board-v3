import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../main.dart';
import '../../account/data/models/account_model.dart';
import '../../account/data/repositories/account_repository.dart';
import '../../association/data/models/association_model.dart' show AssociationDetail;
import '../../association/data/repositories/association_repository.dart';
import '../../auth/presentation/auth_provider.dart';
import '../providers/permission_provider.dart';

class EmergencyAppointmentScreen extends ConsumerStatefulWidget {
  const EmergencyAppointmentScreen({super.key});

  @override
  ConsumerState<EmergencyAppointmentScreen> createState() =>
      _EmergencyAppointmentScreenState();
}

class _EmergencyAppointmentScreenState
    extends ConsumerState<EmergencyAppointmentScreen> {
  List<AssociationDetail> _associations = [];
  List<AccountModel> _accounts = [];
  bool _loadingAssoc = true;
  bool _loadingAccounts = false;

  String? _selectedAssocId;
  String? _selectedUserId;
  String? _selectedRole;

  static const _roles = [
    ('association_admin', '自治会管理者'),
    ('vice_admin', '副会長'),
    ('user_admin', 'ユーザー管理者'),
    ('user', '一般ユーザー'),
  ];

  @override
  void initState() {
    super.initState();
    _loadAssociations();
  }

  Future<void> _loadAssociations() async {
    setState(() => _loadingAssoc = true);
    try {
      final client = ref.read(apiClientProvider);
      final repo = AssociationManagementRepository(client);
      _associations = await repo.list();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('自治会一覧の取得に失敗しました')),
        );
      }
    } finally {
      if (mounted) setState(() => _loadingAssoc = false);
    }
  }

  Future<void> _loadAccounts(String assocId) async {
    setState(() {
      _loadingAccounts = true;
      _accounts = [];
      _selectedUserId = null;
    });
    try {
      final client = ref.read(apiClientProvider);
      final repo = AccountRepository(client);
      _accounts = await repo.list(associationId: assocId);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('ユーザー一覧の取得に失敗しました')),
        );
      }
    } finally {
      if (mounted) setState(() => _loadingAccounts = false);
    }
  }

  Future<void> _appoint() async {
    if (_selectedUserId == null || _selectedRole == null) return;

    final confirmed = await _showConfirmDialog();
    if (!confirmed) return;

    final notifier = ref.read(emergencyAppointmentNotifierProvider);
    final success = await notifier.appoint(
      userId: _selectedUserId!,
      newRole: _selectedRole!,
    );

    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('緊急任命を実行しました'),
          backgroundColor: Colors.green,
        ),
      );
      Navigator.of(context).pop();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(notifier.errorMessage ?? '緊急任命に失敗しました'),
          backgroundColor: Colors.red,
        ),
      );
    }
  }

  Future<bool> _showConfirmDialog() async {
    final user = _accounts.firstWhere((a) => a.id == _selectedUserId);
    final roleLabel = _roles
        .firstWhere((r) => r.$1 == _selectedRole,
            orElse: () => (_selectedRole!, _selectedRole!))
        .$2;

    return await showDialog<bool>(
          context: context,
          builder: (ctx) => AlertDialog(
            title: const Text('緊急任命の確認'),
            content: Text(
              '${user.name} さんのロールを\n「$roleLabel」に変更します。\n\n'
              'この操作はログに記録され、自治会の全メンバーに通知されます。',
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(ctx).pop(false),
                child: const Text('キャンセル'),
              ),
              ElevatedButton(
                style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.orange[700]),
                onPressed: () => Navigator.of(ctx).pop(true),
                child: const Text('実行',
                    style: TextStyle(color: Colors.white)),
              ),
            ],
          ),
        ) ??
        false;
  }

  @override
  Widget build(BuildContext context) {
    final notifier = ref.watch(emergencyAppointmentNotifierProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('緊急任命'),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
      ),
      body: _loadingAssoc
          ? const Center(child: CircularProgressIndicator())
          : SingleChildScrollView(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // 注意書き
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: Colors.orange[50],
                      border: Border.all(color: Colors.orange[300]!),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Icon(Icons.warning_amber_rounded,
                            color: Colors.orange[700], size: 20),
                        const SizedBox(width: 8),
                        const Expanded(
                          child: Text(
                            '緊急任命はロールを強制変更します。'
                            '操作ログに記録され、対象自治会の全メンバーに通知されます。',
                            style: TextStyle(fontSize: 13),
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 24),

                  // 自治会選択
                  _SectionLabel(label: '① 自治会を選択'),
                  const SizedBox(height: 8),
                  DropdownButtonFormField<String>(
                    decoration: const InputDecoration(
                      border: OutlineInputBorder(),
                      contentPadding:
                          EdgeInsets.symmetric(horizontal: 12, vertical: 12),
                    ),
                    hint: const Text('自治会を選択してください'),
                    value: _selectedAssocId,
                    items: _associations
                        .map((a) => DropdownMenuItem(
                              value: a.id,
                              child: Text(a.name),
                            ))
                        .toList(),
                    onChanged: (v) {
                      setState(() {
                        _selectedAssocId = v;
                        _selectedRole = null;
                      });
                      if (v != null) _loadAccounts(v);
                    },
                  ),
                  const SizedBox(height: 24),

                  // ユーザー選択
                  _SectionLabel(label: '② 対象ユーザーを選択'),
                  const SizedBox(height: 8),
                  if (_loadingAccounts)
                    const Center(child: CircularProgressIndicator())
                  else
                    DropdownButtonFormField<String>(
                      decoration: const InputDecoration(
                        border: OutlineInputBorder(),
                        contentPadding:
                            EdgeInsets.symmetric(horizontal: 12, vertical: 12),
                      ),
                      hint: const Text('ユーザーを選択してください'),
                      value: _selectedUserId,
                      items: _accounts
                          .map((a) => DropdownMenuItem(
                                value: a.id,
                                child: Text('${a.name}（${_roleName(a.role)}）'),
                              ))
                          .toList(),
                      onChanged: _selectedAssocId == null
                          ? null
                          : (v) => setState(() => _selectedUserId = v),
                    ),
                  const SizedBox(height: 24),

                  // ロール選択
                  _SectionLabel(label: '③ 移譲するロールを選択'),
                  const SizedBox(height: 8),
                  DropdownButtonFormField<String>(
                    decoration: const InputDecoration(
                      border: OutlineInputBorder(),
                      contentPadding:
                          EdgeInsets.symmetric(horizontal: 12, vertical: 12),
                    ),
                    hint: const Text('ロールを選択してください'),
                    value: _selectedRole,
                    items: _roles
                        .map((r) => DropdownMenuItem(
                              value: r.$1,
                              child: Text(r.$2),
                            ))
                        .toList(),
                    onChanged: (v) => setState(() => _selectedRole = v),
                  ),
                  const SizedBox(height: 32),

                  // 実行ボタン
                  SizedBox(
                    width: double.infinity,
                    child: ElevatedButton.icon(
                      style: ElevatedButton.styleFrom(
                        backgroundColor: Colors.orange[700],
                        foregroundColor: Colors.white,
                        padding: const EdgeInsets.symmetric(vertical: 14),
                      ),
                      onPressed: (_selectedUserId != null &&
                              _selectedRole != null &&
                              !notifier.isSubmitting)
                          ? _appoint
                          : null,
                      icon: notifier.isSubmitting
                          ? const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(
                                  strokeWidth: 2, color: Colors.white))
                          : const Icon(Icons.assignment_ind_rounded),
                      label: const Text('緊急任命を実行',
                          style: TextStyle(fontSize: 16)),
                    ),
                  ),
                ],
              ),
            ),
    );
  }

  String _roleName(String role) {
    switch (role) {
      case 'system_admin':
        return 'システム管理者';
      case 'association_admin':
        return '自治会管理者';
      case 'vice_admin':
        return '副会長';
      case 'user_admin':
        return 'ユーザー管理者';
      case 'user':
        return '一般ユーザー';
      default:
        return role;
    }
  }
}

class _SectionLabel extends StatelessWidget {
  final String label;
  const _SectionLabel({required this.label});

  @override
  Widget build(BuildContext context) => Text(
        label,
        style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 15),
      );
}
