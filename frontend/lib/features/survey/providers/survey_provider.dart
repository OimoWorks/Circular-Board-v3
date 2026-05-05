import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../auth/presentation/auth_provider.dart';
import '../data/models/survey_model.dart';
import '../data/repositories/survey_repository.dart';

final surveyRepositoryProvider = Provider<SurveyRepository>((ref) {
  final client = ref.watch(apiClientProvider);
  return SurveyRepository(client);
});

final surveyNotifierProvider =
    ChangeNotifierProvider<SurveyNotifier>((ref) {
  final repo = ref.watch(surveyRepositoryProvider);
  return SurveyNotifier(repo);
});

final unansweredSurveyCountProvider = FutureProvider.autoDispose<int>((ref) async {
  final user = ref.watch(currentUserProvider);
  if (user == null) return 0;
  final repo = ref.watch(surveyRepositoryProvider);
  try {
    final assocId = user.role == 'system_admin' ? null : user.associationId;
    if (user.role == 'system_admin') return 0; // system_admin には所属自治会なし
    return await repo.getUnansweredCount(associationId: assocId);
  } catch (_) {
    return 0;
  }
});

final recentSurveysProvider = FutureProvider.autoDispose<List<SurveyModel>>((ref) async {
  final user = ref.watch(currentUserProvider);
  if (user == null || user.role == 'system_admin') return [];
  final repo = ref.watch(surveyRepositoryProvider);
  try {
    final surveys = await repo.list();
    return surveys.take(3).toList();
  } catch (_) {
    return [];
  }
});

class SurveyNotifier extends ChangeNotifier {
  final SurveyRepository _repo;

  List<SurveyModel> _surveys = [];
  SurveyModel? _currentSurvey;
  SurveyResult? _currentResult;
  bool _isLoading = false;
  bool _isSubmitting = false;
  String? _errorMessage;

  SurveyNotifier(this._repo);

  List<SurveyModel> get surveys => _surveys;
  SurveyModel? get currentSurvey => _currentSurvey;
  SurveyResult? get currentResult => _currentResult;
  bool get isLoading => _isLoading;
  bool get isSubmitting => _isSubmitting;
  String? get errorMessage => _errorMessage;

  void clearError() {
    _errorMessage = null;
    notifyListeners();
  }

  Future<void> load({String? associationId}) async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      _surveys = await _repo.list(associationId: associationId);
    } on SurveyException catch (e) {
      _errorMessage = e.message;
    } catch (e) {
      _errorMessage = '予期しないエラーが発生しました';
      if (kDebugMode) print(e);
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<SurveyModel?> loadDetail(String id, {String? associationId}) async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      _currentSurvey = await _repo.get(id, associationId: associationId);
      return _currentSurvey;
    } on SurveyException catch (e) {
      _errorMessage = e.message;
      return null;
    } catch (e) {
      _errorMessage = '予期しないエラーが発生しました';
      if (kDebugMode) print(e);
      return null;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<SurveyModel?> create({
    required String title,
    required String description,
    required DateTime expiresAt,
    required List<Map<String, dynamic>> questions,
    String? associationId,
  }) async {
    _isSubmitting = true;
    _errorMessage = null;
    notifyListeners();

    try {
      final survey = await _repo.create(
        title: title,
        description: description,
        expiresAt: expiresAt,
        questions: questions,
        associationId: associationId,
      );
      _surveys.insert(0, survey);
      return survey;
    } on SurveyException catch (e) {
      _errorMessage = e.message;
      return null;
    } catch (e) {
      _errorMessage = '予期しないエラーが発生しました';
      if (kDebugMode) print(e);
      return null;
    } finally {
      _isSubmitting = false;
      notifyListeners();
    }
  }

  Future<bool> delete(String id) async {
    try {
      await _repo.delete(id);
      _surveys.removeWhere((s) => s.id == id);
      notifyListeners();
      return true;
    } on SurveyException catch (e) {
      _errorMessage = e.message;
      notifyListeners();
      return false;
    } catch (_) {
      _errorMessage = '削除に失敗しました';
      notifyListeners();
      return false;
    }
  }

  Future<bool> submitAnswer({
    required String surveyId,
    required List<Map<String, dynamic>> answers,
  }) async {
    _isSubmitting = true;
    _errorMessage = null;
    notifyListeners();

    try {
      await _repo.answer(surveyId: surveyId, answers: answers);
      // 一覧の該当アンケートを回答済みに更新
      final idx = _surveys.indexWhere((s) => s.id == surveyId);
      if (idx >= 0) {
        _surveys[idx] = SurveyModel(
          id: _surveys[idx].id,
          associationId: _surveys[idx].associationId,
          title: _surveys[idx].title,
          description: _surveys[idx].description,
          expiresAt: _surveys[idx].expiresAt,
          isAnswered: true,
          isExpired: _surveys[idx].isExpired,
          createdBy: _surveys[idx].createdBy,
          createdAt: _surveys[idx].createdAt,
          questions: _surveys[idx].questions,
          images: _surveys[idx].images,
        );
      }
      return true;
    } on SurveyException catch (e) {
      _errorMessage = e.message;
      return false;
    } catch (e) {
      _errorMessage = '回答の送信に失敗しました';
      if (kDebugMode) print(e);
      return false;
    } finally {
      _isSubmitting = false;
      notifyListeners();
    }
  }

  Future<SurveyResult?> loadResults(String id, {String? associationId}) async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      _currentResult = await _repo.getResults(id, associationId: associationId);
      return _currentResult;
    } on SurveyException catch (e) {
      _errorMessage = e.message;
      return null;
    } catch (e) {
      _errorMessage = '集計結果の取得に失敗しました';
      if (kDebugMode) print(e);
      return null;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
}
