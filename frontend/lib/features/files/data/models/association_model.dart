class AssociationModel {
  final String id;
  final String name;
  final String code;

  const AssociationModel({
    required this.id,
    required this.name,
    required this.code,
  });

  factory AssociationModel.fromJson(Map<String, dynamic> json) {
    return AssociationModel(
      id: json['id'] as String,
      name: json['name'] as String,
      code: json['code'] as String,
    );
  }
}
