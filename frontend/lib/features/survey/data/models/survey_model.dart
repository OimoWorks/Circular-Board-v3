class SurveyModel {
  final String id;
  final String associationId;
  final String title;
  final String description;
  final DateTime expiresAt;
  final bool isAnswered;
  final bool isExpired;
  final String createdBy;
  final DateTime createdAt;
  final List<SurveyQuestion> questions;

  const SurveyModel({
    required this.id,
    required this.associationId,
    required this.title,
    required this.description,
    required this.expiresAt,
    required this.isAnswered,
    required this.isExpired,
    required this.createdBy,
    required this.createdAt,
    this.questions = const [],
  });

  factory SurveyModel.fromJson(Map<String, dynamic> json) {
    return SurveyModel(
      id: json['id'] as String,
      associationId: json['association_id'] as String,
      title: json['title'] as String,
      description: (json['description'] as String?) ?? '',
      expiresAt: DateTime.parse(json['expires_at'] as String),
      isAnswered: (json['is_answered'] as bool?) ?? false,
      isExpired: (json['is_expired'] as bool?) ?? false,
      createdBy: json['created_by'] as String,
      createdAt: DateTime.parse(json['created_at'] as String),
      questions: (json['questions'] as List<dynamic>?)
              ?.map((q) => SurveyQuestion.fromJson(q as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }

  String get statusLabel {
    if (isExpired) return '期限切れ';
    if (isAnswered) return '回答済み';
    return '未回答';
  }
}

class SurveyQuestion {
  final String id;
  final String questionText;
  final String questionType; // "single" | "multiple"
  final int sortOrder;
  final List<SurveyChoice> choices;

  const SurveyQuestion({
    required this.id,
    required this.questionText,
    required this.questionType,
    required this.sortOrder,
    required this.choices,
  });

  factory SurveyQuestion.fromJson(Map<String, dynamic> json) {
    return SurveyQuestion(
      id: json['id'] as String,
      questionText: json['question_text'] as String,
      questionType: json['question_type'] as String,
      sortOrder: (json['sort_order'] as int?) ?? 0,
      choices: (json['choices'] as List<dynamic>)
          .map((c) => SurveyChoice.fromJson(c as Map<String, dynamic>))
          .toList(),
    );
  }
}

class SurveyChoice {
  final String id;
  final String choiceText;
  final int sortOrder;

  const SurveyChoice({
    required this.id,
    required this.choiceText,
    required this.sortOrder,
  });

  factory SurveyChoice.fromJson(Map<String, dynamic> json) {
    return SurveyChoice(
      id: json['id'] as String,
      choiceText: json['choice_text'] as String,
      sortOrder: (json['sort_order'] as int?) ?? 0,
    );
  }
}

class SurveyResult {
  final String surveyId;
  final String title;
  final int totalAnswered;
  final List<QuestionResult> questions;

  const SurveyResult({
    required this.surveyId,
    required this.title,
    required this.totalAnswered,
    required this.questions,
  });

  factory SurveyResult.fromJson(Map<String, dynamic> json) {
    return SurveyResult(
      surveyId: json['survey_id'] as String,
      title: json['title'] as String,
      totalAnswered: (json['total_answered'] as int?) ?? 0,
      questions: (json['questions'] as List<dynamic>)
          .map((q) => QuestionResult.fromJson(q as Map<String, dynamic>))
          .toList(),
    );
  }
}

class QuestionResult {
  final String questionId;
  final String questionText;
  final String questionType;
  final int totalAnswers;
  final List<ChoiceResult> choices;

  const QuestionResult({
    required this.questionId,
    required this.questionText,
    required this.questionType,
    required this.totalAnswers,
    required this.choices,
  });

  factory QuestionResult.fromJson(Map<String, dynamic> json) {
    return QuestionResult(
      questionId: json['question_id'] as String,
      questionText: json['question_text'] as String,
      questionType: json['question_type'] as String,
      totalAnswers: (json['total_answers'] as int?) ?? 0,
      choices: (json['choices'] as List<dynamic>)
          .map((c) => ChoiceResult.fromJson(c as Map<String, dynamic>))
          .toList(),
    );
  }
}

class ChoiceResult {
  final String choiceId;
  final String choiceText;
  final int count;
  final double percentage;

  const ChoiceResult({
    required this.choiceId,
    required this.choiceText,
    required this.count,
    required this.percentage,
  });

  factory ChoiceResult.fromJson(Map<String, dynamic> json) {
    return ChoiceResult(
      choiceId: json['choice_id'] as String,
      choiceText: json['choice_text'] as String,
      count: (json['count'] as int?) ?? 0,
      percentage: ((json['percentage'] as num?) ?? 0.0).toDouble(),
    );
  }
}
