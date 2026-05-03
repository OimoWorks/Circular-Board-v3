import 'package:dio/dio.dart';

import '../../../../core/api_client.dart';
import '../models/account_model.dart';

class AccountException implements Exception {
  final String message;
  final String? code;
  AccountException(this.message, {this.code});

  @override
  String toString() => message;
}

class AccountRepository {
  final ApiClient _client;

  AccountRepository(this._client);

  Future<List<AccountModel>> list({String? associationId}) async {
    try {
      final Map<String, dynamic> params = {};
      if (associationId != null && associationId.isNotEmpty) {
        params['association_id'] = associationId;
      }
      final res = await _client.get<Map<String, dynamic>>(
        '/accounts',
        queryParameters: params.isEmpty ? null : params,
      );
      final data = res.data!['data'] as Map<String, dynamic>;
      final items = data['accounts'] as List<dynamic>;
      return items
          .map((e) => AccountModel.fromJson(e as Map<String, dynamic>))
          .toList();
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<AccountModel> create({
    required String name,
    required String email,
    required String password,
    required String role,
    String? associationId,
  }) async {
    try {
      final body = <String, dynamic>{
        'name': name,
        'email': email,
        'password': password,
        'role': role,
      };
      if (associationId != null && associationId.isNotEmpty) {
        body['association_id'] = associationId;
      }
      final res = await _client.post<Map<String, dynamic>>('/accounts', data: body);
      final data = res.data!['data'] as Map<String, dynamic>;
      return AccountModel.fromJson(data);
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<AccountModel> update({
    required String id,
    required String name,
    required String email,
    required String role,
    String? password,
  }) async {
    try {
      final body = <String, dynamic>{
        'name': name,
        'email': email,
        'role': role,
      };
      if (password != null && password.isNotEmpty) {
        body['password'] = password;
      }
      final res = await _client.put<Map<String, dynamic>>('/accounts/$id', data: body);
      final data = res.data!['data'] as Map<String, dynamic>;
      return AccountModel.fromJson(data);
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _client.delete('/accounts/$id');
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> activate(String id) async {
    try {
      await _client.put('/accounts/$id/activate');
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> deactivate(String id) async {
    try {
      await _client.put('/accounts/$id/deactivate');
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  AccountException _mapError(DioException e) {
    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final error = data['error'] as Map<String, dynamic>?;
      if (error != null) {
        return AccountException(
          error['message'] as String? ?? 'エラーが発生しました',
          code: error['code'] as String?,
        );
      }
    }
    if (e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout) {
      return AccountException('接続タイムアウトです。通信環境を確認してください。');
    }
    return AccountException('通信エラーが発生しました。');
  }
}
