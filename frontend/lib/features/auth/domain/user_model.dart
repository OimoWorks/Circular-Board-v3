class User {
  final String id;
  final String name;
  final String email;
  final String role;
  final String? associationId;
  final String? associationName;
  final bool isActive;

  const User({
    required this.id,
    required this.name,
    required this.email,
    required this.role,
    this.associationId,
    this.associationName,
    required this.isActive,
  });

  factory User.fromJson(Map<String, dynamic> json) => User(
        id: json['id'] as String,
        name: json['name'] as String,
        email: json['email'] as String,
        role: json['role'] as String,
        associationId: json['association_id'] as String?,
        associationName: json['association_name'] as String?,
        isActive: json['is_active'] as bool? ?? true,
      );

  String get roleLabel {
    switch (role) {
      case 'system_admin':
        return 'システム管理者';
      case 'association_admin':
        return '自治会管理者';
      case 'vice_admin':
        return '副会長';
      case 'user_admin':
        return 'ユーザー管理者';
      case 'user':
        return '一般ユーザー';
      default:
        return role;
    }
  }

  bool get isSystemAdmin => role == 'system_admin';
  bool get isAssociationAdmin => role == 'association_admin';
  bool get isViceAdmin => role == 'vice_admin';
  bool get isUserAdmin => role == 'user_admin';
}

class AuthTokens {
  final String accessToken;
  final String refreshToken;
  final int expiresIn;
  final User user;

  const AuthTokens({
    required this.accessToken,
    required this.refreshToken,
    required this.expiresIn,
    required this.user,
  });

  factory AuthTokens.fromJson(Map<String, dynamic> json) => AuthTokens(
        accessToken: json['access_token'] as String,
        refreshToken: json['refresh_token'] as String,
        expiresIn: json['expires_in'] as int,
        user: User.fromJson(json['user'] as Map<String, dynamic>),
      );
}
