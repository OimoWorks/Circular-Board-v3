import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../../../core/api_client.dart';
import '../data/auth_repository.dart';
import '../domain/user_model.dart';

// ストレージのProvider
final secureStorageProvider = Provider<FlutterSecureStorage>((ref) {
  return const FlutterSecureStorage(
    webOptions: WebOptions(dbName: 'circular_board', publicKey: 'cb_tokens'),
  );
});

// APIクライアントのProvider
final apiClientProvider = Provider<ApiClient>((ref) {
  final storage = ref.watch(secureStorageProvider);
  return ApiClient(storage);
});

// AuthRepositoryのProvider
final authRepositoryProvider = Provider<AuthRepository>((ref) {
  final client = ref.watch(apiClientProvider);
  final storage = ref.watch(secureStorageProvider);
  return AuthRepository(client, storage);
});

// 認証状態のProvider
final authNotifierProvider = ChangeNotifierProvider<AuthNotifier>((ref) {
  final repo = ref.watch(authRepositoryProvider);
  return AuthNotifier(repo);
});

// ログインユーザーへの簡易アクセス
final currentUserProvider = Provider<User?>((ref) {
  return ref.watch(authNotifierProvider).user;
});

class AuthNotifier extends ChangeNotifier {
  final AuthRepository _repo;

  User? _user;
  bool _isLoading = false;
  String? _errorMessage;
  bool _initialized = false;

  AuthNotifier(this._repo);

  User? get user => _user;
  bool get isLoading => _isLoading;
  bool get isAuthenticated => _user != null;
  String? get errorMessage => _errorMessage;
  bool get initialized => _initialized;

  Future<void> initialize() async {
    if (_initialized) return;
    _isLoading = true;
    notifyListeners();

    try {
      _user = await _repo.restoreSession();
    } catch (_) {
      _user = null;
    } finally {
      _isLoading = false;
      _initialized = true;
      notifyListeners();
    }
  }

  Future<bool> login({
    required String associationCode,
    required String email,
    required String password,
  }) async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      final tokens = await _repo.login(
        associationCode: associationCode,
        email: email,
        password: password,
      );
      _user = tokens.user;
      return true;
    } on AuthException catch (e) {
      _errorMessage = e.message;
      return false;
    } catch (e) {
      _errorMessage = '予期しないエラーが発生しました。';
      if (kDebugMode) print(e);
      return false;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> logout() async {
    _isLoading = true;
    notifyListeners();

    try {
      await _repo.logout();
    } finally {
      _user = null;
      _isLoading = false;
      _errorMessage = null;
      notifyListeners();
    }
  }

  void clearError() {
    _errorMessage = null;
    notifyListeners();
  }
}
