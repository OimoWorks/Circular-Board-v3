import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../features/auth/presentation/auth_provider.dart';
import '../data/models/association_model.dart';
import '../data/repositories/association_repository.dart';

final associationManagementRepositoryProvider =
    Provider<AssociationManagementRepository>((ref) {
  final client = ref.watch(apiClientProvider);
  return AssociationManagementRepository(client);
});

final associationManagementNotifierProvider =
    ChangeNotifierProvider<AssociationManagementNotifier>((ref) {
  final repo = ref.watch(associationManagementRepositoryProvider);
  return AssociationManagementNotifier(repo);
});

class AssociationManagementNotifier extends ChangeNotifier {
  final AssociationManagementRepository _repo;

  List<AssociationDetail> _associations = [];
  bool _isLoading = false;
  bool _isSubmitting = false;
  String? _errorMessage;

  AssociationManagementNotifier(this._repo);

  List<AssociationDetail> get associations => _associations;
  bool get isLoading => _isLoading;
  bool get isSubmitting => _isSubmitting;
  String? get errorMessage => _errorMessage;

  Future<void> load() async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      _associations = await _repo.list();
    } on AssociationException catch (e) {
      _errorMessage = e.message;
    } catch (e) {
      _errorMessage = '自治会一覧の取得に失敗しました。';
      if (kDebugMode) print(e);
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<bool> create({required String name, required String code}) async {
    _isSubmitting = true;
    _errorMessage = null;
    notifyListeners();

    try {
      final a = await _repo.create(name: name, code: code);
      _associations = [a, ..._associations];
      return true;
    } on AssociationException catch (e) {
      _errorMessage = e.message;
      return false;
    } catch (e) {
      _errorMessage = '自治会の作成に失敗しました。';
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
    required String code,
  }) async {
    _isSubmitting = true;
    _errorMessage = null;
    notifyListeners();

    try {
      final a = await _repo.update(id: id, name: name, code: code);
      _associations =
          _associations.map((x) => x.id == id ? a : x).toList();
      return true;
    } on AssociationException catch (e) {
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
      _associations = _associations
          .map((a) => a.id == id ? a.copyWith(isActive: false) : a)
          .toList();
      notifyListeners();
      return true;
    } on AssociationException catch (e) {
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
      _associations = _associations
          .map((a) => a.id == id ? a.copyWith(isActive: true) : a)
          .toList();
      notifyListeners();
      return true;
    } on AssociationException catch (e) {
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
      _associations = _associations
          .map((a) => a.id == id ? a.copyWith(isActive: false) : a)
          .toList();
      notifyListeners();
      return true;
    } on AssociationException catch (e) {
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
