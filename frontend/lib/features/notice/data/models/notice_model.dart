class NoticeModel {
  final String id;
  final String associationId;
  final String title;
  final String body;
  final bool isPinned;
  final bool isRead;
  final String createdBy;
  final DateTime createdAt;
  final DateTime updatedAt;

  const NoticeModel({
    required this.id,
    required this.associationId,
    required this.title,
    required this.body,
    required this.isPinned,
    required this.isRead,
    required this.createdBy,
    required this.createdAt,
    required this.updatedAt,
  });

  factory NoticeModel.fromJson(Map<String, dynamic> json) => NoticeModel(
        id: json['id'] as String,
        associationId: json['association_id'] as String,
        title: json['title'] as String,
        body: json['body'] as String,
        isPinned: json['is_pinned'] as bool? ?? false,
        isRead: json['is_read'] as bool? ?? false,
        createdBy: json['created_by'] as String,
        createdAt: DateTime.parse(json['created_at'] as String),
        updatedAt: DateTime.parse(json['updated_at'] as String),
      );

  NoticeModel copyWith({bool? isRead}) => NoticeModel(
        id: id,
        associationId: associationId,
        title: title,
        body: body,
        isPinned: isPinned,
        isRead: isRead ?? this.isRead,
        createdBy: createdBy,
        createdAt: createdAt,
        updatedAt: updatedAt,
      );
}
