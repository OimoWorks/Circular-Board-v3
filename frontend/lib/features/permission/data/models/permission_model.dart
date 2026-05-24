class RoleModel {
  final String id;
  final String name;
  final String displayName;
  final bool isSystem;

  const RoleModel({
    required this.id,
    required this.name,
    required this.displayName,
    required this.isSystem,
  });

  factory RoleModel.fromJson(Map<String, dynamic> json) => RoleModel(
        id: json['id'] as String,
        name: json['name'] as String,
        displayName: json['display_name'] as String,
        isSystem: json['is_system'] as bool? ?? false,
      );
}

class FeatureModel {
  final String id;
  final String name;
  final String displayName;
  final int sortOrder;

  const FeatureModel({
    required this.id,
    required this.name,
    required this.displayName,
    required this.sortOrder,
  });

  factory FeatureModel.fromJson(Map<String, dynamic> json) => FeatureModel(
        id: json['id'] as String,
        name: json['name'] as String,
        displayName: json['display_name'] as String,
        sortOrder: json['sort_order'] as int? ?? 0,
      );
}

class RolePermission {
  final String id;
  final String roleId;
  final String featureId;
  /// null の場合はデフォルト設定（全自治会共通）を表す。
  /// 非 null の場合はその自治会専用の設定を表す。
  final String? associationId;
  final bool canView;
  final bool canCreate;
  final bool canEdit;
  final bool canDelete;
  final String scope;
  /// true の場合は自治会専用の設定が存在することを示す。
  /// false の場合はデフォルト設定にフォールバックしていることを示す。
  final bool isCustomized;

  const RolePermission({
    required this.id,
    required this.roleId,
    required this.featureId,
    this.associationId,
    required this.canView,
    required this.canCreate,
    required this.canEdit,
    required this.canDelete,
    required this.scope,
    this.isCustomized = false,
  });

  factory RolePermission.fromJson(Map<String, dynamic> json) => RolePermission(
        id: json['id'] as String,
        roleId: json['role_id'] as String,
        featureId: json['feature_id'] as String,
        associationId: json['association_id'] as String?,
        canView: json['can_view'] as bool? ?? false,
        canCreate: json['can_create'] as bool? ?? false,
        canEdit: json['can_edit'] as bool? ?? false,
        canDelete: json['can_delete'] as bool? ?? false,
        scope: json['scope'] as String? ?? 'own_association',
        isCustomized: json['is_customized'] as bool? ?? false,
      );

  RolePermission copyWith({
    bool? canView,
    bool? canCreate,
    bool? canEdit,
    bool? canDelete,
    String? scope,
    bool? isCustomized,
  }) =>
      RolePermission(
        id: id,
        roleId: roleId,
        featureId: featureId,
        associationId: associationId,
        canView: canView ?? this.canView,
        canCreate: canCreate ?? this.canCreate,
        canEdit: canEdit ?? this.canEdit,
        canDelete: canDelete ?? this.canDelete,
        scope: scope ?? this.scope,
        isCustomized: isCustomized ?? this.isCustomized,
      );

  Map<String, dynamic> toJson() => {
        'role_id': roleId,
        'feature_id': featureId,
        'can_view': canView,
        'can_create': canCreate,
        'can_edit': canEdit,
        'can_delete': canDelete,
        'scope': scope,
      };
}

class PermissionMatrix {
  final List<RoleModel> roles;
  final List<FeatureModel> features;
  final List<RolePermission> permissions;

  const PermissionMatrix({
    required this.roles,
    required this.features,
    required this.permissions,
  });

  factory PermissionMatrix.fromJson(Map<String, dynamic> json) =>
      PermissionMatrix(
        roles: (json['roles'] as List<dynamic>? ?? [])
            .map((e) => RoleModel.fromJson(e as Map<String, dynamic>))
            .toList(),
        features: (json['features'] as List<dynamic>? ?? [])
            .map((e) => FeatureModel.fromJson(e as Map<String, dynamic>))
            .toList(),
        permissions: (json['permissions'] as List<dynamic>? ?? [])
            .map((e) => RolePermission.fromJson(e as Map<String, dynamic>))
            .toList(),
      );

  RolePermission? permissionFor(String roleId, String featureId) {
    try {
      return permissions.firstWhere(
        (p) => p.roleId == roleId && p.featureId == featureId,
      );
    } catch (_) {
      return null;
    }
  }
}

class OperationLog {
  final String id;
  final String operatorId;
  final String operationType;
  final String targetType;
  final String? targetId;
  final String? beforeValue;
  final String? afterValue;
  final String? associationId;
  final DateTime createdAt;

  const OperationLog({
    required this.id,
    required this.operatorId,
    required this.operationType,
    required this.targetType,
    this.targetId,
    this.beforeValue,
    this.afterValue,
    this.associationId,
    required this.createdAt,
  });

  factory OperationLog.fromJson(Map<String, dynamic> json) => OperationLog(
        id: json['id'] as String,
        operatorId: json['operator_id'] as String,
        operationType: json['operation_type'] as String,
        targetType: json['target_type'] as String,
        targetId: json['target_id'] as String?,
        beforeValue: json['before_value'] as String?,
        afterValue: json['after_value'] as String?,
        associationId: json['association_id'] as String?,
        createdAt: DateTime.parse(json['created_at'] as String),
      );
}
