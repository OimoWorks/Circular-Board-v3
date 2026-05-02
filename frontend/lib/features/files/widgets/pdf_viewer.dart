// プラットフォーム別PDFビューアの条件付きexport
// Web  → pdfx でページを画像レンダリング（PDF.js使用）
// その他 → flutter_pdfview でネイティブレンダリング
export 'pdf_viewer_stub.dart'
    if (dart.library.html) 'pdf_viewer_web.dart'
    if (dart.library.io) 'pdf_viewer_io.dart';
