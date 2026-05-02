// Android/iOS用PDFビューア
// flutter_pdfview を使用。PDFバイトをテンポラリファイルに保存してから表示する。
// 横スワイプでページ移動、ピンチズーム対応。
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_pdfview/flutter_pdfview.dart';
import 'package:path_provider/path_provider.dart';

class PdfViewer extends StatefulWidget {
  final Uint8List bytes;
  const PdfViewer({required this.bytes, super.key});

  @override
  State<PdfViewer> createState() => _PdfViewerIoState();
}

class _PdfViewerIoState extends State<PdfViewer> {
  String? _filePath;
  int _currentPage = 1;
  int _totalPages = 0;
  PDFViewController? _pdfController;

  @override
  void initState() {
    super.initState();
    _prepareFile();
  }

  Future<void> _prepareFile() async {
    final dir = await getTemporaryDirectory();
    final path =
        '${dir.path}/pdf_preview_${DateTime.now().millisecondsSinceEpoch}.pdf';
    await File(path).writeAsBytes(widget.bytes);
    if (mounted) setState(() => _filePath = path);
  }

  @override
  void dispose() {
    // テンポラリファイルを削除
    if (_filePath != null) {
      File(_filePath!).delete().catchError((_) {});
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_filePath == null) {
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
          PDFView(
            filePath: _filePath!,
            enableSwipe: true,
            swipeHorizontal: true, // 横スワイプでページ移動
            autoSpacing: true,
            pageFling: true, // ページ単位でスナップ
            pageSnap: true,
            defaultPage: 0,
            fitPolicy: FitPolicy.BOTH, // 1ページを画面に収める
            preventLinkNavigation: false,
            onViewCreated: (PDFViewController c) {
              _pdfController = c;
            },
            onRender: (pages) {
              if (mounted) setState(() => _totalPages = pages ?? 0);
            },
            onPageChanged: (page, total) {
              if (mounted) {
                setState(() {
                  _currentPage = (page ?? 0) + 1;
                  _totalPages = total ?? 0;
                });
              }
            },
            onError: (error) {
              if (mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('PDF読み込みエラー: $error')),
                );
              }
            },
          ),
          if (_totalPages > 0)
            _PageCounter(current: _currentPage, total: _totalPages),
        ],
      ),
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
