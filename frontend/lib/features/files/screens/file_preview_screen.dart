import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/download_helper.dart';
import '../../../main.dart';
import '../data/models/file_model.dart';
import '../providers/file_provider.dart';
import '../widgets/pdf_viewer.dart';

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
          // PDFはプラットフォーム別ビューア（Web: pdfx、Android: flutter_pdfview）
          if (_isPdf) return PdfViewer(bytes: bytes);
          // 画像はInteractiveViewerでピンチズーム対応
          return _ImagePreview(bytes: bytes);
        },
      ),
    );
  }
}

// ─── 画像プレビュー（JPG/PNG）──────────────────────────────────

class _ImagePreview extends StatefulWidget {
  final Uint8List bytes;
  const _ImagePreview({required this.bytes});

  @override
  State<_ImagePreview> createState() => _ImagePreviewState();
}

class _ImagePreviewState extends State<_ImagePreview>
    with SingleTickerProviderStateMixin {
  late final AnimationController _animController;
  late final TransformationController _transformController;
  Animation<Matrix4>? _zoomAnimation;
  TapDownDetails? _doubleTapDetails;
  bool _isZoomed = false;

  @override
  void initState() {
    super.initState();
    _animController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 220),
    );
    _transformController = TransformationController();
    _transformController.addListener(_onTransformChanged);
  }

  void _onTransformChanged() {
    final scale = _transformController.value.getMaxScaleOnAxis();
    final zoomed = scale > 1.05;
    if (zoomed != _isZoomed) setState(() => _isZoomed = zoomed);
  }

  @override
  void dispose() {
    _zoomAnimation?.removeListener(_applyZoom);
    _animController.dispose();
    _transformController.removeListener(_onTransformChanged);
    _transformController.dispose();
    super.dispose();
  }

  void _applyZoom() {
    if (mounted && _zoomAnimation != null) {
      _transformController.value = _zoomAnimation!.value;
    }
  }

  void _handleDoubleTap() {
    Matrix4 target;
    if (_isZoomed) {
      target = Matrix4.identity();
    } else {
      if (_doubleTapDetails == null) return;
      final pos = _doubleTapDetails!.localPosition;
      const scale = 2.5;
      target = Matrix4.identity()
        ..translate(-pos.dx * (scale - 1), -pos.dy * (scale - 1))
        ..scale(scale);
    }

    _zoomAnimation?.removeListener(_applyZoom);
    _zoomAnimation = Matrix4Tween(
      begin: _transformController.value,
      end: target,
    ).animate(
      CurvedAnimation(parent: _animController, curve: Curves.easeInOut),
    );
    _zoomAnimation!.addListener(_applyZoom);
    _animController.forward(from: 0.0).then((_) {
      _zoomAnimation?.removeListener(_applyZoom);
    });
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      color: const Color(0xFF616161),
      child: GestureDetector(
        onDoubleTapDown: (d) => _doubleTapDetails = d,
        onDoubleTap: _handleDoubleTap,
        child: InteractiveViewer(
          transformationController: _transformController,
          panEnabled: _isZoomed,
          minScale: 0.5,
          maxScale: 5.0,
          child: Center(
            child: Image.memory(widget.bytes, fit: BoxFit.contain),
          ),
        ),
      ),
    );
  }
}
