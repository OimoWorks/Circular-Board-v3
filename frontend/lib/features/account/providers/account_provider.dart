import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../features/auth/presentation/auth_provider.dart';
import '../data/models/account_model.dart';
import '../data/repositories/account_repository.dart';

final accountRepositoryProvider = Provider<AccountRepository>((ref) {
  final client = ref.watch(apiClientProvider);
  return AccountRepository(client);
});

final accountNotifierProvider =
    ChangeNotifierProvider<AccountNotifier>((ref) {
  final repo = ref.watch(accountRepositoryProvider);
  return AccountNotifier(repo);
});

class AccountNotifier extends ChangeNotifier {
  final AccountRepository _repo;

  List<AccountModel> _accounts = [];
  bool _isLoading = false;
  bool _isSubmitting = false;
  String? _errorMessage;

  AccountNotifier(this._repo);

  List<AccountModel> get accounts => _accounts;
  bool get isLoading => _isLoading;
  bool get isSubmitting => _isSubmitting;
  String? get errorMessage => _errorMessage;

  Future<void> load({String? associationId}) async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      _accounts = await _repo.list(associationId: associationId);
    } on AccountException catch (e) {
      _errorMessage = e.message;
    } catch (e) {
      _errorMessage = 'アカウント一覧の取得に失敗しました。';
      if (kDebugMode) print(e);
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<bool> create({
    required String name,
    required String email,
    required String password,
    required String role,
    String? associationId,
  }) async {
    _isSubmitting = true;
    _errorMessage = null;
    notifyListeners();

    try {
      final a = await _repo.create(
        name: name,
        email: email,
        password: password,
        role: role,
        associationId: associationId,
      );
      _accounts = [a, ..._accounts];
      return true;
    } on AccountException catch (e) {
      _errorMessage = e.message;
      return false;
    } catch (e) {
      _errorMessage = 'アカウントの作成に失敗しました。';
      if (kDebugMode) print(e);
      return false;
    } finally {
      _isSubmitting = false;
      notifyListeners();
    }
  }

  Future<bool> update({
    required String id,
    required String name,
    required String email,
    required String role,
    String? password,
  }) async {
    _isSubmitting = true;
    _errorMessage = null;
    notifyListeners();

    try {
      final a = await _repo.update(
        id: id,
        name: name,
        email: email,
        role: role,
        password: password,
      );
      _accounts = _accounts.map((x) => x.id == id ? a : x).toList();
      return true;
    } on AccountException catch (e) {
      _errorMessage = e.message;
      return false;
    } catch (e) {
      _errorMessage = '更新に失敗しました。';
      if (kDebugMode) print(e);
      return false;
    } finally {
      _isSubmitting = false;
      notifyListeners();
    }
  }

  Future<bool> delete(String id) async {
    _errorMessage = null;
    try {
      await _repo.delete(id);
      _accounts = _accounts
          .map((a) => a.id == id ? a.copyWith(isActive: false) : a)
          .toList();
      notifyListeners();
      return true;
    } on AccountException catch (e) {
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

  Future<bool> activate(String id) async {
    _errorMessage = null;
    try {
      await _repo.activate(id);
      _accounts = _accounts
          .map((a) => a.id == id ? a.copyWith(isActive: true) : a)
          .toList();
      notifyListeners();
      return true;
    } on AccountException catch (e) {
      _errorMessage = e.message;
      notifyListeners();
      return false;
    } catch (e) {
      _errorMessage = '有効化に失敗しました。';
      if (kDebugMode) print(e);
      notifyListeners();
      return false;
    }
  }

  Future<bool> deactivate(String id) async {
    _errorMessage = null;
    try {
      await _repo.deactivate(id);
      _accounts = _accounts
          .map((a) => a.id == id ? a.copyWith(isActive: false) : a)
          .toList();
      notifyListeners();
      return true;
    } on AccountException catch (e) {
      _errorMessage = e.message;
      notifyListeners();
      return false;
    } catch (e) {
      _errorMessage = '無効化に失敗しました。';
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
