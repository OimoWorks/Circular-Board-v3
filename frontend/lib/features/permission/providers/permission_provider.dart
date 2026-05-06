import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../features/auth/presentation/auth_provider.dart';
import '../data/models/permission_model.dart';
import '../data/repositories/permission_repository.dart';

final permissionRepositoryProvider = Provider<PermissionRepository>((ref) {
  final client = ref.watch(apiClientProvider);
  return PermissionRepository(client);
});

final permissionNotifierProvider =
    ChangeNotifierProvider<PermissionNotifier>((ref) {
  final repo = ref.watch(permissionRepositoryProvider);
  return PermissionNotifier(repo);
});

class PermissionNotifier extends ChangeNotifier {
  final PermissionRepository _repo;

  PermissionMatrix? _matrix;
  bool _isLoading = false;
  bool _isSubmitting = false;
  String? _errorMessage;
  String? _successMessage;

  // ローカル編集用（保存前の一時状態）
  List<RolePermission> _editedPermissions = [];

  PermissionNotifier(this._repo);

  PermissionMatrix? get matrix => _matrix;
  bool get isLoading => _isLoading;
  bool get isSubmitting => _isSubmitting;
  String? get errorMessage => _errorMessage;
  String? get successMessage => _successMessage;
  List<RolePermission> get editedPermissions => _editedPermissions;

  Future<void> load() async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      _matrix = await _repo.getMatrix();
      _editedPermissions = List.from(_matrix!.permissions);
    } on PermissionException catch (e) {
      _errorMessage = e.message;
    } catch (e) {
      _errorMessage = '権限情報の取得に失敗しました。';
      if (kDebugMode) print(e);
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  void updatePermission(RolePermission updated) {
    final idx = _editedPermissions.indexWhere(
      (p) => p.roleId == updated.roleId && p.featureId == updated.featureId,
    );
    if (idx >= 0) {
      _editedPermissions[idx] = updated;
    } else {
      _editedPermissions.add(updated);
    }
    notifyListeners();
  }

  RolePermission? getEdited(String roleId, String featureId) {
    try {
      return _editedPermissions.firstWhere(
        (p) => p.roleId == roleId && p.featureId == featureId,
      );
    } catch (_) {
      return _matrix?.permissionFor(roleId, featureId);
    }
  }

  Future<bool> save() async {
    _isSubmitting = true;
    _errorMessage = null;
    _successMessage = null;
    notifyListeners();

    try {
      await _repo.updatePermissions(_editedPermissions);
      _successMessage = '権限を更新しました';
      await load();
      return true;
    } on PermissionException catch (e) {
      _errorMessage = e.message;
      return false;
    } catch (e) {
      _errorMessage = '権限の更新に失敗しました。';
      if (kDebugMode) print(e);
      return false;
    } finally {
      _isSubmitting = false;
      notifyListeners();
    }
  }

  void clearMessages() {
    _errorMessage = null;
    _successMessage = null;
    notifyListeners();
  }
}

// ─── 緊急任命 ────────────────────────────────────────────────────────

class EmergencyAppointmentNotifier extends ChangeNotifier {
  final PermissionRepository _repo;

  bool _isSubmitting = false;
  String? _errorMessage;
  String? _successMessage;

  EmergencyAppointmentNotifier(this._repo);

  bool get isSubmitting => _isSubmitting;
  String? get errorMessage => _errorMessage;
  String? get successMessage => _successMessage;

  Future<bool> appoint({
    required String userId,
    required String newRole,
  }) async {
    _isSubmitting = true;
    _errorMessage = null;
    _successMessage = null;
    notifyListeners();

    try {
      await _repo.emergencyAppointment(userId: userId, newRole: newRole);
      _successMessage = '緊急任命を実行しました';
      return true;
    } on PermissionException catch (e) {
      _errorMessage = e.message;
      return false;
    } catch (e) {
      _errorMessage = '緊急任命に失敗しました。';
      if (kDebugMode) print(e);
      return false;
    } finally {
      _isSubmitting = false;
      notifyListeners();
    }
  }

  void clearMessages() {
    _errorMessage = null;
    _successMessage = null;
    notifyListeners();
  }
}

final emergencyAppointmentNotifierProvider =
    ChangeNotifierProvider<EmergencyAppointmentNotifier>((ref) {
  final repo = ref.watch(permissionRepositoryProvider);
  return EmergencyAppointmentNotifier(repo);
});
