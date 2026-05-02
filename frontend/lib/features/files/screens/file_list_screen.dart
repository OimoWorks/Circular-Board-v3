import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../main.dart';
import '../../auth/presentation/auth_provider.dart';
import '../data/models/file_model.dart';
import '../providers/file_provider.dart';

class FileListScreen extends ConsumerStatefulWidget {
  const FileListScreen({super.key});

  @override
  ConsumerState<FileListScreen> createState() => _FileListScreenState();
}

class _FileListScreenState extends ConsumerState<FileListScreen> {
  static const _monthLabels = [
    '1月', '2月', '3月', '4月', '5月', '6月',
    '7月', '8月', '9月', '10月', '11月', '12月',
  ];

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(fileNotifierProvider).loadFiles();
    });
  }

  // ─── アップロード ──────────────────────────────────────────────

  Future<void> _showUploadDialog() async {
    final now = DateTime.now();
    int selectedYear = now.year;
    int selectedMonth = now.month;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDlgState) => AlertDialog(
          title: const Text('アップロード先を選択'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              DropdownButtonFormField<int>(
                decoration: const InputDecoration(labelText: '年'),
                value: selectedYear,
                items: List.generate(5, (i) => now.year - i)
                    .map((y) => DropdownMenuItem(
                          value: y,
                          child: Text('$y年'),
                        ))
                    .toList(),
                onChanged: (v) =>
                    setDlgState(() => selectedYear = v ?? selectedYear),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<int>(
                decoration: const InputDecoration(labelText: '月'),
                value: selectedMonth,
                items: List.generate(12, (i) => i + 1)
                    .map((m) => DropdownMenuItem(
                          value: m,
                          child: Text(_monthLabels[m - 1]),
                        ))
                    .toList(),
                onChanged: (v) =>
                    setDlgState(() => selectedMonth = v ?? selectedMonth),
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(false),
              child: const Text('キャンセル'),
            ),
            FilledButton(
              onPressed: () => Navigator.of(ctx).pop(true),
              child: const Text('ファイルを選択'),
            ),
          ],
        ),
      ),
    );

    if (confirmed != true || !mounted) return;
    await _pickAndUpload(year: selectedYear, month: selectedMonth);
  }

  Future<void> _pickAndUpload({required int year, required int month}) async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['pdf', 'jpg', 'jpeg', 'png'],
      withData: true,
    );
    if (result == null || result.files.isEmpty) return;

    final picked = result.files.first;
    if (picked.bytes == null) return;

    final mimeType = _mimeFromExtension(picked.extension ?? '');
    if (mimeType == null) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('PDF・JPG・PNGのみアップロード可能です')),
        );
      }
      return;
    }

    final notifier = ref.read(fileNotifierProvider);
    final ok = await notifier.upload(
      filename: picked.name,
      bytes: picked.bytes!,
      mimeType: mimeType,
      year: year,
      month: month,
    );
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(ok
              ? 'アップロードしました'
              : (notifier.errorMessage ?? 'アップロードに失敗しました')),
          backgroundColor: ok ? AppColors.primary : Colors.red,
        ),
      );
    }
  }

  String? _mimeFromExtension(String ext) {
    switch (ext.toLowerCase()) {
      case 'pdf':
        return 'application/pdf';
      case 'jpg':
      case 'jpeg':
        return 'image/jpeg';
      case 'png':
        return 'image/png';
      default:
        return null;
    }
  }

  // ─── 削除 ──────────────────────────────────────────────────────

  Future<void> _confirmDelete(FileModel file) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('ファイルを削除'),
        content: Text('「${file.originalFilename}」を削除してよいですか？'),
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
    if (confirmed != true) return;

    final notifier = ref.read(fileNotifierProvider);
    final ok = await notifier.delete(file.id);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(ok
              ? '削除しました'
              : (notifier.errorMessage ?? '削除に失敗しました')),
          backgroundColor: ok ? AppColors.primary : Colors.red,
        ),
      );
    }
  }

  // ─── ビルド ────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final user = ref.watch(currentUserProvider);
    final notifier = ref.watch(fileNotifierProvider);
    final canAdmin =
        user?.role == 'association_admin' || user?.role == 'system_admin';

    return Scaffold(
      appBar: AppBar(
        title: const Row(
          children: [
            Icon(Icons.folder_rounded, size: 22, color: Colors.white),
            SizedBox(width: 8),
            Text('回覧物', style: TextStyle(fontWeight: FontWeight.bold)),
          ],
        ),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
        elevation: 0,
      ),
      floatingActionButton: canAdmin
          ? FloatingActionButton.extended(
              onPressed: notifier.isUploading ? null : _showUploadDialog,
              backgroundColor: AppColors.primary,
              foregroundColor: Colors.white,
              icon: notifier.isUploading
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white),
                    )
                  : const Icon(Icons.upload_file_rounded),
              label: const Text('アップロード'),
            )
          : null,
      body: _buildBody(notifier, canAdmin),
    );
  }

  Widget _buildBody(FileNotifier notifier, bool canAdmin) {
    if (notifier.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (notifier.errorMessage != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline_rounded,
                size: 48, color: Colors.red),
            const SizedBox(height: 12),
            Text(notifier.errorMessage!,
                style: const TextStyle(color: Colors.red)),
          ],
        ),
      );
    }
    if (notifier.files.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.folder_open_rounded, size: 64, color: Colors.grey[300]),
            const SizedBox(height: 16),
            Text('回覧物がありません',
                style: TextStyle(color: Colors.grey[500])),
          ],
        ),
      );
    }

    final grouped = notifier.filesGrouped;
    final sortedYears = notifier.sortedYears;
    final latestYear = sortedYears.first;
    final latestMonth = notifier.sortedMonths(latestYear).first;

    return ListView.builder(
      padding: const EdgeInsets.only(top: 8, bottom: 88),
      itemCount: sortedYears.length,
      itemBuilder: (ctx, yi) {
        final year = sortedYears[yi];
        final isLatestYear = year == latestYear;
        final monthsInYear = notifier.sortedMonths(year);

        return _YearGroup(
          year: year,
          initiallyExpanded: isLatestYear,
          children: monthsInYear.map((month) {
            final files = grouped[year]![month]!;
            final isLatestMonth = isLatestYear && month == latestMonth;

            return _MonthGroup(
              monthLabel: _monthLabels[month - 1],
              fileCount: files.length,
              initiallyExpanded: isLatestMonth,
              children: files
                  .map((f) => _FileTile(
                        file: f,
                        canAdmin: canAdmin,
                        onPreview: () =>
                            context.push('/files/preview', extra: f),
                        onDownload: () =>
                            ref.read(fileNotifierProvider).download(f),
                        onDelete: () => _confirmDelete(f),
                      ))
                  .toList(),
            );
          }).toList(),
        );
      },
    );
  }
}

// ─── 年グループ ────────────────────────────────────────────────

class _YearGroup extends StatelessWidget {
  final int year;
  final bool initiallyExpanded;
  final List<Widget> children;

  const _YearGroup({
    required this.year,
    required this.initiallyExpanded,
    required this.children,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      elevation: 1,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
      child: Theme(
        data: Theme.of(context).copyWith(dividerColor: Colors.transparent),
        child: ExpansionTile(
          leading: const Icon(Icons.calendar_today_rounded,
              color: AppColors.primary, size: 20),
          title: Text(
            '$year年',
            style: const TextStyle(
                fontWeight: FontWeight.bold,
                fontSize: 16,
                color: AppColors.primaryDark),
          ),
          initiallyExpanded: initiallyExpanded,
          iconColor: AppColors.primary,
          collapsedIconColor: AppColors.primary,
          childrenPadding:
              const EdgeInsets.only(left: 8, right: 8, bottom: 8),
          children: children,
        ),
      ),
    );
  }
}

// ─── 月グループ ────────────────────────────────────────────────

class _MonthGroup extends StatelessWidget {
  final String monthLabel;
  final int fileCount;
  final bool initiallyExpanded;
  final List<Widget> children;

  const _MonthGroup({
    required this.monthLabel,
    required this.fileCount,
    required this.initiallyExpanded,
    required this.children,
  });

  @override
  Widget build(BuildContext context) {
    return Theme(
      data: Theme.of(context).copyWith(dividerColor: Colors.transparent),
      child: ExpansionTile(
        leading: Container(
          width: 28,
          height: 28,
          decoration: BoxDecoration(
            color: AppColors.primary.withOpacity(0.08),
            borderRadius: BorderRadius.circular(6),
          ),
          child: const Icon(Icons.event_note_rounded,
              color: AppColors.primaryLight, size: 16),
        ),
        title: Text(
          '$monthLabel　$fileCount件',
          style: const TextStyle(fontSize: 14, color: AppColors.primaryDark),
        ),
        initiallyExpanded: initiallyExpanded,
        iconColor: AppColors.primaryLight,
        collapsedIconColor: AppColors.primaryLight,
        childrenPadding: const EdgeInsets.only(left: 12, bottom: 4),
        children: children,
      ),
    );
  }
}

// ─── ファイルタイル ────────────────────────────────────────────

class _FileTile extends StatelessWidget {
  final FileModel file;
  final bool canAdmin;
  final VoidCallback onPreview;
  final VoidCallback onDownload;
  final VoidCallback onDelete;

  const _FileTile({
    required this.file,
    required this.canAdmin,
    required this.onPreview,
    required this.onDownload,
    required this.onDelete,
  });

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
    return InkWell(
      onTap: onPreview,
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        child: Row(
          children: [
            Container(
              width: 34,
              height: 34,
              decoration: BoxDecoration(
                color: _iconColor.withOpacity(0.10),
                borderRadius: BorderRadius.circular(7),
              ),
              child: Icon(_icon, color: _iconColor, size: 18),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    file.originalFilename,
                    style: const TextStyle(fontSize: 13),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  Text(
                    file.fileSizeLabel,
                    style: TextStyle(fontSize: 11, color: Colors.grey[500]),
                  ),
                ],
              ),
            ),
            // プレビューボタン
            IconButton(
              icon: const Icon(Icons.visibility_rounded, size: 19),
              tooltip: 'プレビュー',
              color: AppColors.primaryLight,
              onPressed: onPreview,
            ),
            // ダウンロードボタン
            IconButton(
              icon: const Icon(Icons.download_rounded, size: 19),
              tooltip: 'ダウンロード',
              color: AppColors.primary,
              onPressed: onDownload,
            ),
            // 削除ボタン（管理者のみ）
            if (canAdmin)
              IconButton(
                icon: const Icon(Icons.delete_outline_rounded, size: 19),
                tooltip: '削除',
                color: Colors.red[600],
                onPressed: onDelete,
              ),
          ],
        ),
      ),
    );
  }
}
