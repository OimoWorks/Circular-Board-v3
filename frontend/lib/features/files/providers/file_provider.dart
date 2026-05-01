import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/download_helper.dart';
import '../../auth/presentation/auth_provider.dart';
import '../data/models/file_model.dart';
import '../data/repositories/file_repository.dart';

final fileRepositoryProvider = Provider<FileRepository>((ref) {
  final client = ref.watch(apiClientProvider);
  return FileRepository(client);
});

final fileNotifierProvider = ChangeNotifierProvider<FileNotifier>((ref) {
  final repo = ref.watch(fileRepositoryProvider);
  return FileNotifier(repo);
});

final availableYearsProvider = FutureProvider<List<int>>((ref) async {
  final repo = ref.watch(fileRepositoryProvider);
  return repo.listAvailableYears();
});

class FileNotifier extends ChangeNotifier {
  final FileRepository _repo;

  List<FileModel> _files = [];
  bool _isLoading = false;
  bool _isUploading = false;
  bool _isDownloading = false;
  String? _errorMessage;
  int _selectedYear = 0;
  int _selectedMonth = 0;

  FileNotifier(this._repo);

  List<FileModel> get files => _files;
  bool get isLoading => _isLoading;
  bool get isUploading => _isUploading;
  bool get isDownloading => _isDownloading;
  String? get errorMessage => _errorMessage;
  int get selectedYear => _selectedYear;
  int get selectedMonth => _selectedMonth;

  Future<void> loadFiles({int year = 0, int month = 0}) async {
    _isLoading = true;
    _errorMessage = null;
    _selectedYear = year;
    _selectedMonth = month;
    notifyListeners();

    try {
      _files = await _repo.list(year: year, month: month);
    } on FileException catch (e) {
      _errorMessage = e.message;
    } catch (e) {
      _errorMessage = 'ファイル一覧の取得に失敗しました。';
      if (kDebugMode) print(e);
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<bool> upload({
    required String filename,
    required List<int> bytes,
    required String mimeType,
    required int year,
    required int month,
  }) async {
    _isUploading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      await _repo.upload(
        filePath: filename,
        filename: filename,
        bytes: bytes,
        mimeType: mimeType,
        year: year,
        month: month,
      );
      await loadFiles(year: _selectedYear, month: _selectedMonth);
      return true;
    } on FileException catch (e) {
      _errorMessage = e.message;
      notifyListeners();
      return false;
    } catch (e) {
      _errorMessage = 'アップロードに失敗しました。';
      if (kDebugMode) print(e);
      notifyListeners();
      return false;
    } finally {
      _isUploading = false;
      notifyListeners();
    }
  }

  Future<void> download(FileModel file) async {
    _isDownloading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      final bytes = await _repo.download(file.id);
      await triggerDownload(bytes, file.originalFilename);
    } on FileException catch (e) {
      _errorMessage = e.message;
    } catch (e) {
      _errorMessage = 'ダウンロードに失敗しました。';
      if (kDebugMode) print(e);
    } finally {
      _isDownloading = false;
      notifyListeners();
    }
  }

  Future<bool> delete(String fileId) async {
    _errorMessage = null;
    notifyListeners();

    try {
      await _repo.delete(fileId);
      _files = _files.where((f) => f.id != fileId).toList();
      notifyListeners();
      return true;
    } on FileException catch (e) {
      _errorMessage = e.message;
      notifyListeners();
      return false;
    } catch (e) {
      _errorMessage = '削除に失敗しました。';
      if (kDebugMode) print(e);
      notifyListeners();
      return false;
    }
  }

  void clearError() {
    _errorMessage = null;
    notifyListeners();
  }
}
