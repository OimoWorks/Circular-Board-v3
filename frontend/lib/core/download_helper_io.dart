import 'dart:io';
import 'package:path_provider/path_provider.dart';

Future<void> triggerDownload(List<int> bytes, String filename) async {
  final dir = await getTemporaryDirectory();
  final file = File('${dir.path}/$filename');
  await file.writeAsBytes(bytes);
}
