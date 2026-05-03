class AccountModel {
  final String id;
  final String associationId;
  final String name;
  final String email;
  final String role;
  final bool isActive;
  final DateTime createdAt;
  final DateTime updatedAt;

  const AccountModel({
    required this.id,
    required this.associationId,
    required this.name,
    required this.email,
    required this.role,
    required this.isActive,
    required this.createdAt,
    required this.updatedAt,
  });

  factory AccountModel.fromJson(Map<String, dynamic> json) {
    return AccountModel(
      id: json['id'] as String,
      associationId: (json['association_id'] as String?) ?? '',
      name: json['name'] as String,
      email: json['email'] as String,
      role: json['role'] as String,
      isActive: json['is_active'] as bool,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  String get roleLabel {
    switch (role) {
      case 'system_admin':
        return 'システム管理者';
      case 'association_admin':
        return '自治会管理者';
      default:
        return '一般ユーザー';
    }
  }

  AccountModel copyWith({bool? isActive}) {
    return AccountModel(
      id: id,
      associationId: associationId,
      name: name,
      email: email,
      role: role,
      isActive: isActive ?? this.isActive,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }
}
