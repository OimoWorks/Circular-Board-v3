import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pdfx/pdfx.dart';

import '../../../core/download_helper.dart';
import '../../../main.dart';
import '../data/models/file_model.dart';
import '../providers/file_provider.dart';

class FilePreviewScreen extends ConsumerStatefulWidget {
  final FileModel file;
  const FilePreviewScreen({required this.file, super.key});

  @override
  ConsumerState<FilePreviewScreen> createState() => _FilePreviewScreenState();
}

class _FilePreviewScreenState extends ConsumerState<FilePreviewScreen> {
  late final Future<Uint8List> _bytesFuture;
  bool _isDownloading = false;

  bool get _isPdf => widget.file.mimeType == 'application/pdf';

  @override
  void initState() {
    super.initState();
    _bytesFuture = ref
        .read(fileRepositoryProvider)
        .download(widget.file.id)
        .then(Uint8List.fromList);
  }

  Future<void> _download(Uint8List bytes) async {
    setState(() => _isDownloading = true);
    try {
      await triggerDownload(bytes, widget.file.originalFilename);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('ダウンロードに失敗しました')),
        );
      }
    } finally {
      if (mounted) setState(() => _isDownloading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(
          widget.file.originalFilename,
          style: const TextStyle(fontSize: 15),
          overflow: TextOverflow.ellipsis,
        ),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
        elevation: 0,
        actions: [
          FutureBuilder<Uint8List>(
            future: _bytesFuture,
            builder: (_, snap) => IconButton(
              icon: _isDownloading
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white),
                    )
                  : const Icon(Icons.download_rounded),
              tooltip: 'ダウンロード',
              onPressed: snap.hasData && !_isDownloading
                  ? () => _download(snap.data!)
                  : null,
            ),
          ),
        ],
      ),
      body: FutureBuilder<Uint8List>(
        future: _bytesFuture,
        builder: (_, snap) {
          if (snap.connectionState == ConnectionState.waiting) {
            return const Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  CircularProgressIndicator(),
                  SizedBox(height: 16),
                  Text('読み込み中...'),
                ],
              ),
            );
          }
          if (snap.hasError) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.error_outline_rounded,
                      size: 48, color: Colors.red),
                  const SizedBox(height: 12),
                  Text(
                    'ファイルの読み込みに失敗しました',
                    style: TextStyle(color: Colors.red[700]),
                  ),
                ],
              ),
            );
          }
          final bytes = snap.data!;
          if (_isPdf) return _PdfPreview(bytes: bytes);
          return _ImagePreview(bytes: bytes);
        },
      ),
    );
  }
}

// ─── PDFプレビュー ──────────────────────────────────────────────

class _PdfPreview extends StatefulWidget {
  final Uint8List bytes;
  const _PdfPreview({required this.bytes});

  @override
  State<_PdfPreview> createState() => _PdfPreviewState();
}

class _PdfPreviewState extends State<_PdfPreview> {
  late final PdfControllerPinch _controller;

  @override
  void initState() {
    super.initState();
    _controller = PdfControllerPinch(
      document: PdfDocument.openData(widget.bytes),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return PdfViewPinch(
      controller: _controller,
      builders: PdfViewPinchBuilders<DefaultBuilderOptions>(
        options: const DefaultBuilderOptions(),
        documentLoaderBuilder: (_) => const Center(
          child: CircularProgressIndicator(),
        ),
        pageLoaderBuilder: (_) => const Center(
          child: CircularProgressIndicator(),
        ),
        errorBuilder: (_, error) => Center(
          child: Text('PDFの読み込みエラー: $error'),
        ),
      ),
    );
  }
}

// ─── 画像プレビュー ─────────────────────────────────────────────

class _ImagePreview extends StatelessWidget {
  final Uint8List bytes;
  const _ImagePreview({required this.bytes});

  @override
  Widget build(BuildContext context) {
    return InteractiveViewer(
      minScale: 0.5,
      maxScale: 4.0,
      child: Center(
        child: Image.memory(bytes, fit: BoxFit.contain),
      ),
    );
  }
}
