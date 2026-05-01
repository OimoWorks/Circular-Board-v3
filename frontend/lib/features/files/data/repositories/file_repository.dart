import 'package:dio/dio.dart';

import '../../../../core/api_client.dart';
import '../models/file_model.dart';

class FileException implements Exception {
  final String message;
  final String? code;
  FileException(this.message, {this.code});

  @override
  String toString() => message;
}

class FileRepository {
  final ApiClient _client;

  FileRepository(this._client);

  Future<List<FileModel>> list({int year = 0, int month = 0}) async {
    try {
      final res = await _client.get<Map<String, dynamic>>(
        '/files',
        queryParameters: {
          if (year != 0) 'year': year,
          if (month != 0) 'month': month,
        },
      );
      final data = (res.data!['data'] as Map<String, dynamic>);
      final items = data['files'] as List<dynamic>;
      return items
          .map((e) => FileModel.fromJson(e as Map<String, dynamic>))
          .toList();
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<List<int>> listAvailableYears() async {
    try {
      final res = await _client.get<Map<String, dynamic>>('/files/years');
      final data = (res.data!['data'] as Map<String, dynamic>);
      final years = data['years'] as List<dynamic>;
      return years.map((e) => (e as num).toInt()).toList();
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<FileModel> upload({
    required String filePath,
    required String filename,
    required List<int> bytes,
    required String mimeType,
    required int year,
    required int month,
  }) async {
    try {
      final formData = FormData.fromMap({
        'file': MultipartFile.fromBytes(bytes, filename: filename, contentType: DioMediaType.parse(mimeType)),
        'year': year.toString(),
        'month': month.toString(),
      });
      final res = await _client.postFormData<Map<String, dynamic>>('/files', formData);
      final data = res.data!['data'] as Map<String, dynamic>;
      return FileModel.fromJson(data);
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<List<int>> download(String fileId) async {
    try {
      final res = await _client.getBytes('/files/$fileId/download');
      return res.data ?? [];
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  Future<void> delete(String fileId) async {
    try {
      await _client.delete('/files/$fileId');
    } on DioException catch (e) {
      throw _mapError(e);
    }
  }

  FileException _mapError(DioException e) {
    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final error = data['error'] as Map<String, dynamic>?;
      if (error != null) {
        return FileException(
          error['message'] as String? ?? 'エラーが発生しました',
          code: error['code'] as String?,
        );
      }
    }
    if (e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout) {
      return FileException('接続タイムアウトです。通信環境を確認してください。');
    }
    return FileException('通信エラーが発生しました。');
  }
}
