import 'dart:typed_data';
import 'package:flutter/material.dart';

class PdfViewer extends StatelessWidget {
  final Uint8List bytes;
  const PdfViewer({required this.bytes, super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      color: const Color(0xFF616161),
      child: const Center(
        child: Text(
          'PDFビューアはこのプラットフォームでは利用できません',
          style: TextStyle(color: Colors.white),
        ),
      ),
    );
  }
}
