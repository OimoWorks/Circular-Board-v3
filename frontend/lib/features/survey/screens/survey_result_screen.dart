import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../main.dart';
import '../data/models/survey_model.dart';
import '../providers/survey_provider.dart';

class SurveyResultScreen extends ConsumerStatefulWidget {
  final String surveyId;

  const SurveyResultScreen({super.key, required this.surveyId});

  @override
  ConsumerState<SurveyResultScreen> createState() => _SurveyResultScreenState();
}

class _SurveyResultScreenState extends ConsumerState<SurveyResultScreen> {
  @override
  void initState() {
    super.initState();
    Future.microtask(() =>
        ref.read(surveyNotifierProvider).loadResults(widget.surveyId));
  }

  @override
  Widget build(BuildContext context) {
    final notifier = ref.watch(surveyNotifierProvider);

    if (notifier.isLoading) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    if (notifier.currentResult == null) {
      return Scaffold(
        appBar: AppBar(
          title: const Text('集計結果'),
          backgroundColor: AppColors.primary,
          foregroundColor: AppColors.onPrimary,
        ),
        body: Center(
          child: Text(notifier.errorMessage ?? 'データが見つかりません'),
        ),
      );
    }

    final result = notifier.currentResult!;

    return Scaffold(
      appBar: AppBar(
        title: const Text('集計結果'),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ─── サマリカード ────────────────────────────────────
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      result.title,
                      style: const TextStyle(
                          fontWeight: FontWeight.bold, fontSize: 16),
                    ),
                    const SizedBox(height: 12),
                    Row(
                      children: [
                        const Icon(Icons.people_rounded,
                            color: AppColors.primary, size: 20),
                        const SizedBox(width: 8),
                        Text(
                          '回答者数: ${result.totalAnswered}人',
                          style: const TextStyle(
                              fontWeight: FontWeight.w600, fontSize: 14),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),

            // ─── 質問ごとの集計 ──────────────────────────────────
            ...result.questions.asMap().entries.map((e) =>
                _QuestionResultCard(index: e.key + 1, result: e.value)),
          ],
        ),
      ),
    );
  }
}

class _QuestionResultCard extends StatelessWidget {
  final int index;
  final QuestionResult result;

  const _QuestionResultCard({required this.index, required this.result});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: AppColors.primary,
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Text('Q$index',
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 11,
                          fontWeight: FontWeight.bold)),
                ),
                const SizedBox(width: 8),
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: Colors.grey[100],
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    result.questionType == 'single' ? '単一選択' : '複数選択',
                    style: TextStyle(fontSize: 10, color: Colors.grey[600]),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            Text(
              result.questionText,
              style: const TextStyle(
                  fontWeight: FontWeight.w600, fontSize: 15),
            ),
            const SizedBox(height: 4),
            Text(
              '回答数: ${result.totalAnswers}',
              style: TextStyle(fontSize: 12, color: Colors.grey[500]),
            ),
            const SizedBox(height: 12),
            ...result.choices.map((c) => _ChoiceBar(choice: c)),
          ],
        ),
      ),
    );
  }
}

class _ChoiceBar extends StatelessWidget {
  final ChoiceResult choice;

  const _ChoiceBar({required this.choice});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(choice.choiceText,
                    style: const TextStyle(fontSize: 13)),
              ),
              Text(
                '${choice.count}票 (${choice.percentage.toStringAsFixed(1)}%)',
                style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: AppColors.primary),
              ),
            ],
          ),
          const SizedBox(height: 4),
          ClipRRect(
            borderRadius: BorderRadius.circular(4),
            child: LinearProgressIndicator(
              value: choice.percentage / 100,
              backgroundColor: Colors.grey[200],
              valueColor:
                  const AlwaysStoppedAnimation<Color>(AppColors.primary),
              minHeight: 8,
            ),
          ),
        ],
      ),
    );
  }
}
