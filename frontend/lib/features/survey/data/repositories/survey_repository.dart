import 'dart:typed_data';

import 'package:dio/dio.dart';

import '../../../../core/api_client.dart';
import '../models/survey_model.dart';

class SurveyRepository {
  final ApiClient _client;

  SurveyRepository(this._client);

  Future<List<SurveyModel>> list({String? associationId}) async {
    final params = associationId != null
        ? {'association_id': associationId}
        : null;
    final res = await _client.get('/surveys', queryParameters: params);
    final data = (res.data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
    final surveys = data['surveys'] as List<dynamic>;
    return surveys
        .map((s) => SurveyModel.fromJson(s as Map<String, dynamic>))
        .toList();
  }

  Future<SurveyModel> get(String id, {String? associationId}) async {
    final params = associationId != null
        ? {'association_id': associationId}
        : null;
    final res = await _client.get('/surveys/$id', queryParameters: params);
    final data = (res.data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
    return SurveyModel.fromJson(data);
  }

  Future<SurveyModel> create({
    required String title,
    required String description,
    required DateTime expiresAt,
    required List<Map<String, dynamic>> questions,
    String? associationId,
  }) async {
    final body = {
      'title': title,
      'description': description,
      'expires_at': expiresAt.toUtc().toIso8601String(),
      'questions': questions,
      if (associationId != null) 'association_id': associationId,
    };
    final res = await _client.post('/surveys', data: body);
    final data = (res.data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
    return SurveyModel.fromJson(data);
  }

  Future<void> delete(String id) async {
    await _client.delete('/surveys/$id');
  }

  Future<void> answer({
    required String surveyId,
    required List<Map<String, dynamic>> answers,
  }) async {
    await _client.post('/surveys/$surveyId/answer', data: {'answers': answers});
  }

  Future<SurveyResult> getResults(String id, {String? associationId}) async {
    final params = associationId != null
        ? {'association_id': associationId}
        : null;
    final res = await _client.get('/surveys/$id/results', queryParameters: params);
    final data = (res.data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
    return SurveyResult.fromJson(data);
  }

  Future<SurveyImage> uploadImage(
    String surveyId,
    Uint8List bytes,
    String filename,
    String mimeType, {
    int sortOrder = 0,
  }) async {
    final formData = FormData.fromMap({
      'image': MultipartFile.fromBytes(
        bytes,
        filename: filename,
        contentType: DioMediaType.parse(mimeType),
      ),
      'sort_order': sortOrder.toString(),
    });
    final res = await _client.postFormData('/surveys/$surveyId/images', formData);
    final data = (res.data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
    return SurveyImage.fromJson(data);
  }

  Future<void> deleteImage(String surveyId, String imageId) async {
    await _client.delete('/surveys/$surveyId/images/$imageId');
  }

  Future<Uint8List> getImageBytes(String surveyId, String imageId) async {
    final res = await _client.getBytes('/surveys/$surveyId/images/$imageId');
    return Uint8List.fromList(res.data!);
  }

  Future<int> getUnansweredCount({String? associationId}) async {
    final params = associationId != null
        ? {'association_id': associationId}
        : null;
    final res = await _client.get('/surveys/unanswered-count', queryParameters: params);
    final data = (res.data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
    return (data['unanswered_count'] as int?) ?? 0;
  }
}

class SurveyException implements Exception {
  final String message;
  final String? code;
  SurveyException(this.message, {this.code});

  @override
  String toString() => message;
}

SurveyException mapDioError(DioException e) {
  final data = e.response?.data;
  if (data is Map<String, dynamic>) {
    final error = data['error'] as Map<String, dynamic>?;
    if (error != null) {
      return SurveyException(
        error['message'] as String? ?? 'エラーが発生しました',
        code: error['code'] as String?,
      );
    }
  }
  return SurveyException('通信エラーが発生しました');
}
