import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/download_helper.dart';
import '../../auth/presentation/auth_provider.dart';
import '../data/models/file_model.dart';
import '../data/repositories/file_repository.dart';

export '../data/models/file_model.dart';

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

// ホーム画面用: 最新5件（system_adminには空リストを返す）
final recentFilesProvider = FutureProvider.autoDispose<List<FileModel>>((ref) async {
  final user = ref.watch(currentUserProvider);
  if (user?.role == 'system_admin') return [];
  final repo = ref.watch(fileRepositoryProvider);
  final files = await repo.list();
  return files.take(5).toList();
});

class FileNotifier extends ChangeNotifier {
  final FileRepository _repo;

  List<FileModel> _files = [];
  bool _isLoading = false;
  bool _isUploading = false;
  bool _isDownloading = false;
  String? _errorMessage;
  String? _currentAssociationId;

  FileNotifier(this._repo);

  List<FileModel> get files => _files;
  bool get isLoading => _isLoading;
  bool get isUploading => _isUploading;
  bool get isDownloading => _isDownloading;
  String? get errorMessage => _errorMessage;

  // ─── ツリー表示用 ───────────────────────────────────────────────

  /// year → month → files のマップ（キーは降順ソート済み）
  Map<int, Map<int, List<FileModel>>> get filesGrouped {
    final result = <int, Map<int, List<FileModel>>>{};
    for (final f in _files) {
      result.putIfAbsent(f.year, () => {});
      result[f.year]!.putIfAbsent(f.month, () => []);
      result[f.year]![f.month]!.add(f);
    }
    return result;
  }

  /// 年リスト（降順）
  List<int> get sortedYears {
    return filesGrouped.keys.toList()..sort((a, b) => b.compareTo(a));
  }

  /// 指定年の月リスト（降順）
  List<int> sortedMonths(int year) {
    final months = filesGrouped[year]?.keys.toList() ?? [];
    return months..sort((a, b) => b.compareTo(a));
  }

  // ─── 操作 ──────────────────────────────────────────────────────

  Future<void> loadFiles({int year = 0, int month = 0, String? associationId}) async {
    if (associationId != null) _currentAssociationId = associationId;
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      _files = await _repo.list(year: year, month: month, associationId: _currentAssociationId);
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
    String? associationId,
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
        associationId: associationId ?? _currentAssociationId,
      );
      await loadFiles();
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
