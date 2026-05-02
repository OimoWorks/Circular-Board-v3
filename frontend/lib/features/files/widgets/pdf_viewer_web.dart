// Web用PDFビューア
// pdfx でページごとに画像レンダリングし Flutter PageView に配置する。
// PdfViewPinch を使わないため PDF.js 由来のスクロールガタつきが発生しない。
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:pdfx/pdfx.dart';

class PdfViewer extends StatefulWidget {
  final Uint8List bytes;
  const PdfViewer({required this.bytes, super.key});

  @override
  State<PdfViewer> createState() => _PdfViewerWebState();
}

class _PdfViewerWebState extends State<PdfViewer> {
  PdfDocument? _document;
  int _totalPages = 0;
  int _currentPage = 1;
  final PageController _pageController = PageController();

  // ページ番号 → レンダリング済みFuture のキャッシュ
  final Map<int, Future<Uint8List?>> _cache = {};

  @override
  void initState() {
    super.initState();
    _loadDocument();
  }

  Future<void> _loadDocument() async {
    final doc = await PdfDocument.openData(widget.bytes);
    if (mounted) {
      setState(() {
        _document = doc;
        _totalPages = doc.pagesCount;
      });
    }
  }

  @override
  void dispose() {
    _document?.close();
    _pageController.dispose();
    super.dispose();
  }

  // 同一ページのFutureを使い回してレンダリングの重複を防ぐ
  Future<Uint8List?> _renderPage(int pageNumber) {
    return _cache.putIfAbsent(pageNumber, () => _doRender(pageNumber));
  }

  Future<Uint8List?> _doRender(int pageNumber) async {
    final doc = _document;
    if (doc == null) return null;
    final page = await doc.getPage(pageNumber);
    // 2倍解像度でレンダリングして拡大時も鮮明に保つ
    const pixelRatio = 2.0;
    final image = await page.render(
      width: page.width * pixelRatio,
      height: page.height * pixelRatio,
      format: PdfPageImageFormat.png,
    );
    await page.close();
    return image?.bytes;
  }

  @override
  Widget build(BuildContext context) {
    if (_document == null) {
      return Container(
        color: const Color(0xFF616161),
        child: const Center(
          child: CircularProgressIndicator(color: Colors.white),
        ),
      );
    }

    return Container(
      color: const Color(0xFF616161),
      child: Stack(
        children: [
          // 横スワイプでページ移動する PageView
          PageView.builder(
            controller: _pageController,
            itemCount: _totalPages,
            onPageChanged: (i) => setState(() => _currentPage = i + 1),
            itemBuilder: (_, i) => _PdfPageView(
              pageNumber: i + 1,
              renderPage: _renderPage,
            ),
          ),
          // ページカウンター
          _PageCounter(current: _currentPage, total: _totalPages),
        ],
      ),
    );
  }
}

// ─── 1ページ分のビュー ──────────────────────────────────────────
// InteractiveViewer でピンチズーム・ダブルタップズームに対応。
// panEnabled を zoom 状態と連動させることで、非ズーム時は PageView の
// 横スワイプを妨げない。

class _PdfPageView extends StatefulWidget {
  final int pageNumber;
  final Future<Uint8List?> Function(int) renderPage;
  const _PdfPageView({required this.pageNumber, required this.renderPage});

  @override
  State<_PdfPageView> createState() => _PdfPageViewState();
}

class _PdfPageViewState extends State<_PdfPageView>
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
      // 元に戻す
      target = Matrix4.identity();
    } else {
      // タップ位置を中心に 2.5x ズーム
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
    return FutureBuilder<Uint8List?>(
      future: widget.renderPage(widget.pageNumber),
      builder: (_, snap) {
        if (snap.connectionState != ConnectionState.done) {
          return const Center(
            child: CircularProgressIndicator(color: Colors.white),
          );
        }
        if (snap.data == null) {
          return const Center(
            child: Text(
              'ページの読み込みに失敗しました',
              style: TextStyle(color: Colors.white70),
            ),
          );
        }
        return GestureDetector(
          onDoubleTapDown: (d) => _doubleTapDetails = d,
          onDoubleTap: _handleDoubleTap,
          child: InteractiveViewer(
            transformationController: _transformController,
            // ズーム中のみパンを有効にして非ズーム時は PageView のスワイプを通す
            panEnabled: _isZoomed,
            minScale: 0.5,
            maxScale: 5.0,
            child: Center(
              child: Padding(
                padding:
                    const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
                child: Image.memory(
                  snap.data!,
                  fit: BoxFit.contain,
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

// ─── ページカウンター ───────────────────────────────────────────

class _PageCounter extends StatelessWidget {
  final int current;
  final int total;
  const _PageCounter({required this.current, required this.total});

  @override
  Widget build(BuildContext context) {
    return Positioned(
      bottom: 20,
      left: 0,
      right: 0,
      child: IgnorePointer(
        child: Center(
          child: Container(
            padding:
                const EdgeInsets.symmetric(horizontal: 18, vertical: 7),
            decoration: BoxDecoration(
              color: Colors.black.withOpacity(0.60),
              borderRadius: BorderRadius.circular(20),
            ),
            child: Text(
              '$current / $total',
              style: const TextStyle(
                color: Colors.white,
                fontSize: 14,
                fontWeight: FontWeight.w500,
                letterSpacing: 1.2,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
