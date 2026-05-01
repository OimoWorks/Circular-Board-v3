class FileModel {
  final String id;
  final String associationId;
  final int year;
  final int month;
  final String originalFilename;
  final int fileSize;
  final String mimeType;
  final String uploadedBy;
  final DateTime createdAt;

  const FileModel({
    required this.id,
    required this.associationId,
    required this.year,
    required this.month,
    required this.originalFilename,
    required this.fileSize,
    required this.mimeType,
    required this.uploadedBy,
    required this.createdAt,
  });

  factory FileModel.fromJson(Map<String, dynamic> json) {
    return FileModel(
      id: json['id'] as String,
      associationId: json['association_id'] as String,
      year: (json['year'] as num).toInt(),
      month: (json['month'] as num).toInt(),
      originalFilename: json['original_filename'] as String,
      fileSize: (json['file_size'] as num).toInt(),
      mimeType: json['mime_type'] as String,
      uploadedBy: json['uploaded_by'] as String,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }

  String get fileSizeLabel {
    if (fileSize < 1024) return '${fileSize}B';
    if (fileSize < 1024 * 1024) return '${(fileSize / 1024).toStringAsFixed(1)}KB';
    return '${(fileSize / (1024 * 1024)).toStringAsFixed(1)}MB';
  }
}
