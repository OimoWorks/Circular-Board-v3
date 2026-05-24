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
  PermissionMatrix? _defaultMatrix;
  bool _isLoading = false;
  bool _isSubmitting = false;
  String? _errorMessage;
  String? _successMessage;

  /// 現在選択中の自治会ID。null の場合はデフォルト設定を表示する。
  String? _selectedAssociationId;

  /// ローカル編集用（保存前の一時状態）
  List<RolePermission> _editedPermissions = [];

  PermissionNotifier(this._repo);

  PermissionMatrix? get matrix => _matrix;
  PermissionMatrix? get defaultMatrix => _defaultMatrix;
  bool get isLoading => _isLoading;
  bool get isSubmitting => _isSubmitting;
  String? get errorMessage => _errorMessage;
  String? get successMessage => _successMessage;
  String? get selectedAssociationId => _selectedAssociationId;
  List<RolePermission> get editedPermissions => _editedPermissions;

  /// 自治会を切り替えてその権限設定をロードする。
  /// [associationId] が null の場合はデフォルト設定を表示する。
  Future<void> selectAssociation(String? associationId) async {
    _selectedAssociationId = associationId;
    notifyListeners();
    await load();
  }

  /// 現在選択中の自治会（または null = デフォルト）の権限設定をロードする。
  /// デフォルト設定は常にバックグラウンドでもロードして、
  /// セルの「カスタマイズ済み」判定に使用する。
  Future<void> load() async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      // デフォルト設定は常にロードする（カスタマイズ判定のため）
      final defaultFuture = _repo.getMatrix();

      if (_selectedAssociationId == null) {
        _defaultMatrix = await defaultFuture;
        _matrix = _defaultMatrix;
      } else {
        final results = await Future.wait([
          defaultFuture,
          _repo.getMatrix(associationId: _selectedAssociationId),
        ]);
        _defaultMatrix = results[0];
        _matrix = results[1];
      }
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

  /// デフォルト設定のパーミッションを返す（カスタマイズセルの強調表示に使用）。
  RolePermission? getDefault(String roleId, String featureId) {
    return _defaultMatrix?.permissionFor(roleId, featureId);
  }

  Future<bool> save() async {
    _isSubmitting = true;
    _errorMessage = null;
    _successMessage = null;
    notifyListeners();

    try {
      await _repo.updatePermissions(
        _editedPermissions,
        associationId: _selectedAssociationId,
      );
      _successMessage = _selectedAssociationId == null
          ? 'デフォルト権限を更新しました'
          : '自治会の権限設定を更新しました';
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
