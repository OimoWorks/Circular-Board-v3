import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../main.dart';
import '../../auth/presentation/auth_provider.dart';
import '../providers/survey_provider.dart';
import '../data/models/survey_model.dart';

bool _isAdmin(String role) =>
    role == 'association_admin' || role == 'system_admin';

class SurveyListScreen extends ConsumerStatefulWidget {
  const SurveyListScreen({super.key});

  @override
  ConsumerState<SurveyListScreen> createState() => _SurveyListScreenState();
}

class _SurveyListScreenState extends ConsumerState<SurveyListScreen> {
  @override
  void initState() {
    super.initState();
    Future.microtask(_load);
  }

  void _load() {
    ref.read(surveyNotifierProvider).load();
  }

  @override
  Widget build(BuildContext context) {
    final user = ref.watch(currentUserProvider);
    final notifier = ref.watch(surveyNotifierProvider);
    final isAdmin = user != null && _isAdmin(user.role);

    return Scaffold(
      appBar: AppBar(
        title: const Text('アンケート'),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            onPressed: _load,
          ),
        ],
      ),
      floatingActionButton: isAdmin
          ? FloatingActionButton.extended(
              onPressed: () async {
                final created = await context.push<bool>('/surveys/create');
                if (created == true && mounted) _load();
              },
              icon: const Icon(Icons.add_rounded),
              label: const Text('作成'),
              backgroundColor: AppColors.primary,
              foregroundColor: AppColors.onPrimary,
            )
          : null,
      body: notifier.isLoading
          ? const Center(child: CircularProgressIndicator())
          : notifier.errorMessage != null
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(notifier.errorMessage!,
                          style: TextStyle(color: Colors.red[700])),
                      const SizedBox(height: 12),
                      FilledButton.icon(
                        onPressed: _load,
                        icon: const Icon(Icons.refresh_rounded),
                        label: const Text('再読み込み'),
                      ),
                    ],
                  ),
                )
              : notifier.surveys.isEmpty
                  ? const Center(child: Text('アンケートはありません'))
                  : RefreshIndicator(
                      onRefresh: () async => _load(),
                      child: ListView.separated(
                        padding: const EdgeInsets.all(12),
                        itemCount: notifier.surveys.length,
                        separatorBuilder: (_, __) =>
                            const SizedBox(height: 8),
                        itemBuilder: (ctx, i) => _SurveyCard(
                          survey: notifier.surveys[i],
                          isAdmin: isAdmin,
                          onTap: () => context.push(
                            '/surveys/${notifier.surveys[i].id}',
                          ),
                          onDelete: isAdmin
                              ? () => _confirmDelete(
                                  context, notifier.surveys[i], notifier)
                              : null,
                          onResults: isAdmin
                              ? () => context.push(
                                  '/surveys/${notifier.surveys[i].id}/results')
                              : null,
                        ),
                      ),
                    ),
    );
  }

  Future<void> _confirmDelete(
      BuildContext context, SurveyModel survey, SurveyNotifier notifier) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('アンケートの削除'),
        content: Text('「${survey.title}」を削除しますか？'),
        actions: [
          TextButton(
              onPressed: () => Navigator.of(ctx).pop(false),
              child: const Text('キャンセル')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('削除'),
          ),
        ],
      ),
    );
    if (ok == true && mounted) {
      await notifier.delete(survey.id);
      if (notifier.errorMessage != null && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text(notifier.errorMessage!),
          backgroundColor: Colors.red,
        ));
      }
    }
  }
}

class _SurveyCard extends StatelessWidget {
  final SurveyModel survey;
  final bool isAdmin;
  final VoidCallback onTap;
  final VoidCallback? onDelete;
  final VoidCallback? onResults;

  const _SurveyCard({
    required this.survey,
    required this.isAdmin,
    required this.onTap,
    this.onDelete,
    this.onResults,
  });

  Color _statusColor() {
    if (survey.isExpired) return Colors.grey;
    if (survey.isAnswered) return AppColors.primary;
    return Colors.orange[700]!;
  }

  IconData _statusIcon() {
    if (survey.isExpired) return Icons.schedule_rounded;
    if (survey.isAnswered) return Icons.check_circle_rounded;
    return Icons.circle_outlined;
  }

  @override
  Widget build(BuildContext context) {
    final fmt = DateFormat('yyyy/MM/dd HH:mm');
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(
          color: survey.isExpired
              ? Colors.grey[300]!
              : survey.isAnswered
                  ? AppColors.primary.withOpacity(0.3)
                  : Colors.orange[200]!,
        ),
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(10),
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Row(
            children: [
              Icon(_statusIcon(), color: _statusColor(), size: 28),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      survey.title,
                      style: TextStyle(
                        fontWeight: FontWeight.bold,
                        fontSize: 15,
                        color: survey.isExpired ? Colors.grey : Colors.black87,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Row(
                      children: [
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 6, vertical: 2),
                          decoration: BoxDecoration(
                            color: _statusColor().withOpacity(0.1),
                            borderRadius: BorderRadius.circular(6),
                            border: Border.all(
                                color: _statusColor().withOpacity(0.4)),
                          ),
                          child: Text(
                            survey.statusLabel,
                            style: TextStyle(
                                fontSize: 10,
                                fontWeight: FontWeight.w600,
                                color: _statusColor()),
                          ),
                        ),
                        const SizedBox(width: 8),
                        Text(
                          '期限: ${fmt.format(survey.expiresAt.toLocal())}',
                          style: TextStyle(
                              fontSize: 11, color: Colors.grey[500]),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              if (isAdmin)
                PopupMenuButton<String>(
                  icon:
                      const Icon(Icons.more_vert_rounded, size: 20),
                  onSelected: (v) {
                    if (v == 'results') onResults?.call();
                    if (v == 'delete') onDelete?.call();
                  },
                  itemBuilder: (_) => [
                    const PopupMenuItem(
                      value: 'results',
                      child: ListTile(
                        leading: Icon(Icons.bar_chart_rounded),
                        title: Text('集計結果'),
                        dense: true,
                      ),
                    ),
                    const PopupMenuItem(
                      value: 'delete',
                      child: ListTile(
                        leading:
                            Icon(Icons.delete_rounded, color: Colors.red),
                        title: Text('削除',
                            style: TextStyle(color: Colors.red)),
                        dense: true,
                      ),
                    ),
                  ],
                )
              else
                const Icon(Icons.chevron_right_rounded,
                    color: Colors.grey),
            ],
          ),
        ),
      ),
    );
  }
}
