import 'package:dio/dio.dart';

import '../../../../core/api_client.dart';
import '../models/association_model.dart';

class AssociationException implements Exception {
  final String message;
  final String? code;
  AssociationException(this.message, {this.code});

  @override
  String toString() => message;
}

class AssociationManagementRepository {
  final ApiClient _client;

  AssociationManagementRepository(this._client);

  Future<List<AssociationDetail>> list() async {
    try {
      final res =
          await _client.get<Map<String, dynamic>>('/associations');
      final data = res.data!['data'] as Map<String, dynamic>;
      final items = data['associations'] as List<dynamic>;
      return items
          .map((e) =>
              AssociationDetail.fromJson(e as Map<String, dynamic>))
          .toList();
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<AssociationDetail> create({
    required String name,
    required String code,
  }) async {
    try {
      final res = await _client
          .post<Map<String, dynamic>>('/associations', data: {
        'name': name,
        'code': code,
      });
      final data = res.data!['data'] as Map<String, dynamic>;
      return AssociationDetail.fromJson(data);
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<AssociationDetail> update({
    required String id,
    required String name,
    required String code,
  }) async {
    try {
      final res = await _client
          .put<Map<String, dynamic>>('/associations/$id', data: {
        'name': name,
        'code': code,
      });
      final data = res.data!['data'] as Map<String, dynamic>;
      return AssociationDetail.fromJson(data);
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _client.delete('/associations/$id');
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> activate(String id) async {
    try {
      await _client.put('/associations/$id/activate');
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> deactivate(String id) async {
    try {
      await _client.put('/associations/$id/deactivate');
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  AssociationException _mapError(DioException e) {
    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final error = data['error'] as Map<String, dynamic>?;
      if (error != null) {
        return AssociationException(
          error['message'] as String? ?? 'エラーが発生しました',
          code: error['code'] as String?,
        );
      }
    }
    if (e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout) {
      return AssociationException('接続タイムアウトです。通信環境を確認してください。');
    }
    return AssociationException('通信エラーが発生しました。');
  }
}
