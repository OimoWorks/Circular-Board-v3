import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../features/auth/presentation/auth_provider.dart';
import '../data/models/notice_model.dart';
import '../data/repositories/notice_repository.dart';

// ─── かんたんモード ───────────────────────────────────────────────
// アプリ全体で共有するかんたんモードフラグ
final easyModeProvider = StateProvider<bool>((ref) => false);

// ─── Repository / Notifier ────────────────────────────────────────

final noticeRepositoryProvider = Provider<NoticeRepository>((ref) {
  final client = ref.watch(apiClientProvider);
  return NoticeRepository(client);
});

final noticeNotifierProvider =
    ChangeNotifierProvider<NoticeNotifier>((ref) {
  final repo = ref.watch(noticeRepositoryProvider);
  return NoticeNotifier(repo);
});

// ホーム画面用: 最新3件（未読優先・ピン留め優先）
final recentNoticesProvider =
    FutureProvider.autoDispose<List<NoticeModel>>((ref) async {
  final repo = ref.watch(noticeRepositoryProvider);
  final items = await repo.list();
  return items.take(3).toList();
});

// ホーム画面バッジ用: 未読件数
final unreadCountProvider = FutureProvider.autoDispose<int>((ref) async {
  final repo = ref.watch(noticeRepositoryProvider);
  return repo.unreadCount();
});

// ─── NoticeNotifier ────────────────────────────────────────────────

class NoticeNotifier extends ChangeNotifier {
  final NoticeRepository _repo;

  List<NoticeModel> _notices = [];
  bool _isLoading = false;
  bool _isSubmitting = false;
  String? _errorMessage;

  NoticeNotifier(this._repo);

  List<NoticeModel> get notices => _notices;
  bool get isLoading => _isLoading;
  bool get isSubmitting => _isSubmitting;
  String? get errorMessage => _errorMessage;

  int get unreadCount =>
      _notices.where((n) => !n.isRead).length;

  Future<void> load() async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      _notices = await _repo.list();
    } on NoticeException catch (e) {
      _errorMessage = e.message;
    } catch (e) {
      _errorMessage = 'お知らせ一覧の取得に失敗しました。';
      if (kDebugMode) print(e);
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<bool> create({
    required String title,
    required String body,
    required bool isPinned,
  }) async {
    _isSubmitting = true;
    _errorMessage = null;
    notifyListeners();

    try {
      final n = await _repo.create(title: title, body: body, isPinned: isPinned);
      // ピン留めは先頭、それ以外は先頭（最新順）
      if (n.isPinned) {
        _notices = [n, ..._notices];
      } else {
        final firstUnpinned = _notices.indexWhere((x) => !x.isPinned);
        if (firstUnpinned == -1) {
          _notices = [..._notices, n];
        } else {
          _notices = [
            ..._notices.sublist(0, firstUnpinned),
            n,
            ..._notices.sublist(firstUnpinned),
          ];
        }
      }
      return true;
    } on NoticeException catch (e) {
      _errorMessage = e.message;
      return false;
    } catch (e) {
      _errorMessage = 'お知らせの作成に失敗しました。';
      if (kDebugMode) print(e);
      return false;
    } finally {
      _isSubmitting = false;
      notifyListeners();
    }
  }

  Future<bool> delete(String noticeId) async {
    _errorMessage = null;

    try {
      await _repo.delete(noticeId);
      _notices = _notices.where((n) => n.id != noticeId).toList();
      notifyListeners();
      return true;
    } on NoticeException catch (e) {
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

  Future<void> markAsRead(String noticeId) async {
    try {
      await _repo.markAsRead(noticeId);
      _notices = _notices
          .map((n) => n.id == noticeId ? n.copyWith(isRead: true) : n)
          .toList();
      notifyListeners();
    } catch (_) {
      // 既読失敗はサイレントに無視（UXを壊さないため）
    }
  }

  void clearError() {
    _errorMessage = null;
    notifyListeners();
  }
}
