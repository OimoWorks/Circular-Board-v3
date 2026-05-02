import 'package:dio/dio.dart';

import '../../../../core/api_client.dart';
import '../models/notice_model.dart';

class NoticeException implements Exception {
  final String message;
  final String? code;
  NoticeException(this.message, {this.code});

  @override
  String toString() => message;
}

class NoticeRepository {
  final ApiClient _client;

  NoticeRepository(this._client);

  Future<List<NoticeModel>> list() async {
    try {
      final res = await _client.get<Map<String, dynamic>>('/notices');
      final data = res.data!['data'] as Map<String, dynamic>;
      final items = data['notices'] as List<dynamic>;
      return items
          .map((e) => NoticeModel.fromJson(e as Map<String, dynamic>))
          .toList();
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<NoticeModel> getById(String id) async {
    try {
      final res = await _client.get<Map<String, dynamic>>('/notices/$id');
      final data = res.data!['data'] as Map<String, dynamic>;
      return NoticeModel.fromJson(data);
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<NoticeModel> create({
    required String title,
    required String body,
    required bool isPinned,
  }) async {
    try {
      final res = await _client.post<Map<String, dynamic>>('/notices', data: {
        'title': title,
        'body': body,
        'is_pinned': isPinned,
      });
      final data = res.data!['data'] as Map<String, dynamic>;
      return NoticeModel.fromJson(data);
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _client.delete('/notices/$id');
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> markAsRead(String id) async {
    try {
      await _client.post('/notices/$id/read');
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<int> unreadCount() async {
    try {
      final res =
          await _client.get<Map<String, dynamic>>('/notices/unread-count');
      final data = res.data!['data'] as Map<String, dynamic>;
      return (data['unread_count'] as num?)?.toInt() ?? 0;
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  NoticeException _mapError(DioException e) {
    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final error = data['error'] as Map<String, dynamic>?;
      if (error != null) {
        return NoticeException(
          error['message'] as String? ?? 'エラーが発生しました',
          code: error['code'] as String?,
        );
      }
    }
    if (e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout) {
      return NoticeException('接続タイムアウトです。通信環境を確認してください。');
    }
    return NoticeException('通信エラーが発生しました。');
  }
}
