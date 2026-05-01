import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../../../core/api_client.dart';
import '../../../core/constants.dart';
import '../domain/user_model.dart';

class AuthException implements Exception {
  final String message;
  final String? code;
  AuthException(this.message, {this.code});

  @override
  String toString() => message;
}

class AuthRepository {
  final ApiClient _client;
  final FlutterSecureStorage _storage;

  AuthRepository(this._client, this._storage);

  Future<AuthTokens> login({
    required String associationCode,
    required String email,
    required String password,
  }) async {
    try {
      final res = await _client.post('/auth/login', data: {
        'association_code': associationCode,
        'email': email,
        'password': password,
      });
      final data = (res.data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
      final tokens = AuthTokens.fromJson(data);
      await _saveTokens(tokens);
      return tokens;
    } on DioException catch (e) {
      throw _mapDioError(e);
    }
  }

  Future<User?> restoreSession() async {
    final refreshToken = await _storage.read(key: refreshTokenKey);
    if (refreshToken == null) return null;

    try {
      final res = await _client.post('/auth/refresh', data: {
        'refresh_token': refreshToken,
      });
      final data = (res.data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
      final tokens = AuthTokens.fromJson(data);
      await _saveTokens(tokens);
      return tokens.user;
    } on DioException {
      await _clearTokens();
      return null;
    }
  }

  Future<User> getMe() async {
    try {
      final res = await _client.get('/auth/me');
      final data = (res.data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
      return User.fromJson(data);
    } on DioException catch (e) {
      throw _mapDioError(e);
    }
  }

  Future<void> logout() async {
    try {
      await _client.post('/auth/logout');
    } on DioException {
      // エラーでもローカルは必ずクリアする
    } finally {
      await _clearTokens();
    }
  }

  Future<void> _saveTokens(AuthTokens tokens) async {
    await _storage.write(key: accessTokenKey, value: tokens.accessToken);
    await _storage.write(key: refreshTokenKey, value: tokens.refreshToken);
  }

  Future<void> _clearTokens() async {
    await _storage.delete(key: accessTokenKey);
    await _storage.delete(key: refreshTokenKey);
  }

  AuthException _mapDioError(DioException e) {
    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final error = data['error'] as Map<String, dynamic>?;
      if (error != null) {
        return AuthException(
          error['message'] as String? ?? 'エラーが発生しました',
          code: error['code'] as String?,
        );
      }
    }
    if (e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout) {
      return AuthException('接続タイムアウトです。通信環境を確認してください。');
    }
    return AuthException('通信エラーが発生しました。');
  }
}
