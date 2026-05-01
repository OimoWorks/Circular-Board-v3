import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

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
  int _selectedYear = 0;
  int _selectedMonth = 0;

  static const _months = [
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

  Future<void> _pickAndUpload() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['pdf', 'jpg', 'jpeg', 'png'],
      withData: true,
    );
    if (result == null || result.files.isEmpty) return;

    final picked = result.files.first;
    if (picked.bytes == null) return;

    final mimeType = _mimeTypeFromExtension(picked.extension ?? '');
    if (mimeType == null) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('PDF・JPG・PNGのみアップロード可能です')),
        );
      }
      return;
    }

    final now = DateTime.now();
    final year = _selectedYear != 0 ? _selectedYear : now.year;
    final month = _selectedMonth != 0 ? _selectedMonth : now.month;

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
          content: Text(ok ? 'アップロードしました' : (notifier.errorMessage ?? 'アップロードに失敗しました')),
          backgroundColor: ok ? AppColors.primary : Colors.red,
        ),
      );
    }
  }

  String? _mimeTypeFromExtension(String ext) {
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
          content: Text(ok ? '削除しました' : (notifier.errorMessage ?? '削除に失敗しました')),
          backgroundColor: ok ? AppColors.primary : Colors.red,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final user = ref.watch(currentUserProvider);
    final notifier = ref.watch(fileNotifierProvider);
    final yearsAsync = ref.watch(availableYearsProvider);
    final canAdmin = user?.role == 'association_admin' || user?.role == 'system_admin';

    return Scaffold(
      appBar: AppBar(
        title: const Row(
          children: [
            Icon(Icons.folder_rounded, size: 22, color: Colors.white),
            SizedBox(width: 8),
            Text('ファイル管理', style: TextStyle(fontWeight: FontWeight.bold)),
          ],
        ),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
        elevation: 0,
      ),
      floatingActionButton: canAdmin
          ? FloatingActionButton.extended(
              onPressed: notifier.isUploading ? null : _pickAndUpload,
              backgroundColor: AppColors.primary,
              foregroundColor: Colors.white,
              icon: notifier.isUploading
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                    )
                  : const Icon(Icons.upload_file_rounded),
              label: const Text('アップロード'),
            )
          : null,
      body: Column(
        children: [
          _FilterBar(
            selectedYear: _selectedYear,
            selectedMonth: _selectedMonth,
            yearsAsync: yearsAsync,
            months: _months,
            onYearChanged: (y) {
              setState(() {
                _selectedYear = y;
                _selectedMonth = 0;
              });
              ref.read(fileNotifierProvider).loadFiles(year: y, month: 0);
            },
            onMonthChanged: (m) {
              setState(() => _selectedMonth = m);
              ref.read(fileNotifierProvider).loadFiles(year: _selectedYear, month: m);
            },
          ),
          const Divider(height: 1),
          Expanded(
            child: _FileListBody(
              notifier: notifier,
              canAdmin: canAdmin,
              months: _months,
              onDownload: (f) => ref.read(fileNotifierProvider).download(f),
              onDelete: _confirmDelete,
            ),
          ),
        ],
      ),
    );
  }
}

class _FilterBar extends StatelessWidget {
  final int selectedYear;
  final int selectedMonth;
  final AsyncValue<List<int>> yearsAsync;
  final List<String> months;
  final ValueChanged<int> onYearChanged;
  final ValueChanged<int> onMonthChanged;

  const _FilterBar({
    required this.selectedYear,
    required this.selectedMonth,
    required this.yearsAsync,
    required this.months,
    required this.onYearChanged,
    required this.onMonthChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      color: AppColors.primary.withOpacity(0.04),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          const Icon(Icons.filter_list_rounded, size: 18, color: AppColors.primary),
          const SizedBox(width: 8),
          yearsAsync.when(
            data: (years) => DropdownButton<int>(
              value: selectedYear,
              isDense: true,
              underline: const SizedBox(),
              items: [
                const DropdownMenuItem(value: 0, child: Text('すべての年')),
                ...years.map((y) => DropdownMenuItem(value: y, child: Text('$y年'))),
              ],
              onChanged: (v) => onYearChanged(v ?? 0),
            ),
            loading: () => const SizedBox(width: 80, child: LinearProgressIndicator()),
            error: (_, __) => const Text('年の取得失敗'),
          ),
          const SizedBox(width: 16),
          DropdownButton<int>(
            value: selectedMonth,
            isDense: true,
            underline: const SizedBox(),
            items: [
              const DropdownMenuItem(value: 0, child: Text('すべての月')),
              ...List.generate(12, (i) => i + 1).map(
                (m) => DropdownMenuItem(value: m, child: Text(months[m - 1])),
              ),
            ],
            onChanged: selectedYear == 0 ? null : (v) => onMonthChanged(v ?? 0),
          ),
        ],
      ),
    );
  }
}

class _FileListBody extends StatelessWidget {
  final FileNotifier notifier;
  final bool canAdmin;
  final List<String> months;
  final Future<void> Function(FileModel) onDownload;
  final Future<void> Function(FileModel) onDelete;

  const _FileListBody({
    required this.notifier,
    required this.canAdmin,
    required this.months,
    required this.onDownload,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    if (notifier.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (notifier.errorMessage != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline_rounded, size: 48, color: Colors.red),
            const SizedBox(height: 12),
            Text(notifier.errorMessage!, style: const TextStyle(color: Colors.red)),
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
            Text('ファイルがありません', style: TextStyle(color: Colors.grey[500])),
          ],
        ),
      );
    }

    return ListView.separated(
      padding: const EdgeInsets.symmetric(vertical: 8),
      itemCount: notifier.files.length,
      separatorBuilder: (_, __) => const Divider(height: 1, indent: 56),
      itemBuilder: (ctx, i) {
        final file = notifier.files[i];
        return _FileListTile(
          file: file,
          canAdmin: canAdmin,
          months: months,
          isDownloading: notifier.isDownloading,
          onDownload: () => onDownload(file),
          onDelete: () => onDelete(file),
        );
      },
    );
  }
}

class _FileListTile extends StatelessWidget {
  final FileModel file;
  final bool canAdmin;
  final List<String> months;
  final bool isDownloading;
  final VoidCallback onDownload;
  final VoidCallback onDelete;

  const _FileListTile({
    required this.file,
    required this.canAdmin,
    required this.months,
    required this.isDownloading,
    required this.onDownload,
    required this.onDelete,
  });

  IconData get _fileIcon {
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

  Color get _fileIconColor {
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
    final theme = Theme.of(context);
    final monthLabel = months[file.month - 1];

    return ListTile(
      leading: Container(
        width: 40,
        height: 40,
        decoration: BoxDecoration(
          color: _fileIconColor.withOpacity(0.1),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Icon(_fileIcon, color: _fileIconColor, size: 22),
      ),
      title: Text(
        file.originalFilename,
        style: const TextStyle(fontWeight: FontWeight.w500),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Text(
        '${file.year}年$monthLabel ・ ${file.fileSizeLabel}',
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.onSurfaceVariant,
        ),
      ),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            icon: isDownloading
                ? const SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.download_rounded),
            tooltip: 'ダウンロード',
            color: AppColors.primary,
            onPressed: isDownloading ? null : onDownload,
          ),
          if (canAdmin)
            IconButton(
              icon: const Icon(Icons.delete_outline_rounded),
              tooltip: '削除',
              color: Colors.red[700],
              onPressed: onDelete,
            ),
        ],
      ),
    );
  }
}
