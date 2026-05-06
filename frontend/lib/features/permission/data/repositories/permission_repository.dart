import 'package:dio/dio.dart';

import '../../../../core/api_client.dart';
import '../models/permission_model.dart';

class PermissionException implements Exception {
  final String message;
  final String? code;
  PermissionException(this.message, {this.code});

  @override
  String toString() => message;
}

class PermissionRepository {
  final ApiClient _client;

  PermissionRepository(this._client);

  Future<PermissionMatrix> getMatrix() async {
    try {
      final res = await _client.get<Map<String, dynamic>>('/permissions');
      final data = res.data!['data'] as Map<String, dynamic>;
      return PermissionMatrix.fromJson(data);
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> updatePermissions(List<RolePermission> permissions) async {
    try {
      await _client.put<void>('/permissions', data: {
        'permissions': permissions.map((p) => p.toJson()).toList(),
      });
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<List<RoleModel>> getRoles() async {
    try {
      final res = await _client.get<Map<String, dynamic>>('/permissions/roles');
      final data = res.data!['data'] as Map<String, dynamic>;
      final items = data['roles'] as List<dynamic>? ?? [];
      return items
          .map((e) => RoleModel.fromJson(e as Map<String, dynamic>))
          .toList();
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<List<FeatureModel>> getFeatures() async {
    try {
      final res =
          await _client.get<Map<String, dynamic>>('/permissions/features');
      final data = res.data!['data'] as Map<String, dynamic>;
      final items = data['features'] as List<dynamic>? ?? [];
      return items
          .map((e) => FeatureModel.fromJson(e as Map<String, dynamic>))
          .toList();
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> emergencyAppointment({
    required String userId,
    required String newRole,
  }) async {
    try {
      await _client.post<void>('/permissions/emergency-appointment', data: {
        'user_id': userId,
        'new_role': newRole,
      });
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<List<OperationLog>> getLogs() async {
    try {
      final res = await _client.get<Map<String, dynamic>>('/permissions/logs');
      final data = res.data!['data'] as Map<String, dynamic>;
      final items = data['logs'] as List<dynamic>? ?? [];
      return items
          .map((e) => OperationLog.fromJson(e as Map<String, dynamic>))
          .toList();
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  PermissionException _mapError(DioException e) {
    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final err = data['error'] as Map<String, dynamic>?;
      if (err != null) {
        return PermissionException(
          err['message'] as String? ?? '権限管理エラーが発生しました',
          code: err['code'] as String?,
        );
      }
    }
    return PermissionException('権限管理エラーが発生しました');
  }
}
