import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../main.dart';
import '../data/models/survey_model.dart';
import '../data/repositories/survey_repository.dart';
import '../providers/survey_provider.dart';

class SurveyAnswerScreen extends ConsumerStatefulWidget {
  final String surveyId;

  const SurveyAnswerScreen({super.key, required this.surveyId});

  @override
  ConsumerState<SurveyAnswerScreen> createState() => _SurveyAnswerScreenState();
}

class _SurveyAnswerScreenState extends ConsumerState<SurveyAnswerScreen> {
  SurveyModel? _survey;
  // questionId -> 選択済み choiceId(s)
  final Map<String, Set<String>> _selected = {};

  @override
  void initState() {
    super.initState();
    Future.microtask(_load);
  }

  Future<void> _load() async {
    final survey =
        await ref.read(surveyNotifierProvider).loadDetail(widget.surveyId);
    if (survey != null && mounted) {
      setState(() {
        _survey = survey;
        for (final q in survey.questions) {
          _selected[q.id] = {};
        }
      });
    }
  }

  Future<void> _submit() async {
    if (_survey == null) return;

    // 全質問に回答しているか確認
    for (final q in _survey!.questions) {
      if (_selected[q.id]?.isEmpty ?? true) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
          content: Text('すべての質問に回答してください'),
          backgroundColor: Colors.orange,
        ));
        return;
      }
    }

    final answers = _selected.entries
        .map((e) => {
              'question_id': e.key,
              'choice_ids': e.value.toList(),
            })
        .toList();

    final ok = await ref.read(surveyNotifierProvider).submitAnswer(
          surveyId: widget.surveyId,
          answers: answers,
        );

    if (!mounted) return;
    if (ok) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
        content: Text('回答しました'),
        backgroundColor: AppColors.primary,
      ));
      Navigator.of(context).pop(true);
    } else {
      final msg = ref.read(surveyNotifierProvider).errorMessage ?? 'エラーが発生しました';
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(msg),
        backgroundColor: Colors.red,
      ));
    }
  }

  @override
  Widget build(BuildContext context) {
    final notifier = ref.watch(surveyNotifierProvider);

    if (notifier.isLoading && _survey == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    if (_survey == null) {
      return Scaffold(
        appBar: AppBar(
          title: const Text('アンケート'),
          backgroundColor: AppColors.primary,
          foregroundColor: AppColors.onPrimary,
        ),
        body: Center(
          child: Text(notifier.errorMessage ?? 'アンケートが見つかりません'),
        ),
      );
    }

    final survey = _survey!;
    final fmt = DateFormat('yyyy/MM/dd HH:mm');
    final canAnswer = !survey.isExpired;

    return Scaffold(
      appBar: AppBar(
        title: Text(survey.title),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ─── ヘッダー情報 ────────────────────────────────────
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Icon(
                          survey.isExpired
                              ? Icons.schedule_rounded
                              : survey.isAnswered
                                  ? Icons.check_circle_rounded
                                  : Icons.circle_outlined,
                          color: survey.isExpired
                              ? Colors.grey
                              : survey.isAnswered
                                  ? AppColors.primary
                                  : Colors.orange[700],
                        ),
                        const SizedBox(width: 8),
                        Text(
                          survey.statusLabel,
                          style: TextStyle(
                            fontWeight: FontWeight.bold,
                            color: survey.isExpired
                                ? Colors.grey
                                : survey.isAnswered
                                    ? AppColors.primary
                                    : Colors.orange[700],
                          ),
                        ),
                      ],
                    ),
                    if (survey.description.isNotEmpty) ...[
                      const SizedBox(height: 10),
                      Text(survey.description,
                          style: TextStyle(color: Colors.grey[700])),
                    ],
                    const SizedBox(height: 8),
                    Text(
                      '回答期限: ${fmt.format(survey.expiresAt.toLocal())}',
                      style: TextStyle(fontSize: 12, color: Colors.grey[600]),
                    ),
                    if (survey.isExpired)
                      Container(
                        margin: const EdgeInsets.only(top: 10),
                        padding: const EdgeInsets.all(10),
                        decoration: BoxDecoration(
                          color: Colors.grey[100],
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: const Row(
                          children: [
                            Icon(Icons.info_outline_rounded,
                                color: Colors.grey, size: 16),
                            SizedBox(width: 6),
                            Text('このアンケートは期限切れのため回答できません',
                                style:
                                    TextStyle(fontSize: 12, color: Colors.grey)),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),

            // ─── 画像カルーセル ──────────────────────────────────
            if (survey.images.isNotEmpty)
              _SurveyImagesSection(
                surveyId: survey.id,
                images: survey.images,
              ),

            // ─── 質問リスト ──────────────────────────────────────
            ...survey.questions.asMap().entries.map((entry) {
              final idx = entry.key;
              final q = entry.value;
              return _QuestionCard(
                index: idx + 1,
                question: q,
                selected: _selected[q.id] ?? {},
                canAnswer: canAnswer,
                onChoiceToggled: (choiceId) {
                  if (!canAnswer) return;
                  setState(() {
                    final set = _selected[q.id] ??= {};
                    if (q.questionType == 'single') {
                      set.clear();
                      set.add(choiceId);
                    } else {
                      if (set.contains(choiceId)) {
                        set.remove(choiceId);
                      } else {
                        set.add(choiceId);
                      }
                    }
                  });
                },
              );
            }),

            const SizedBox(height: 24),

            // ─── 送信ボタン ──────────────────────────────────────
            if (canAnswer)
              SizedBox(
                width: double.infinity,
                height: 50,
                child: FilledButton.icon(
                  onPressed: notifier.isSubmitting ? null : _submit,
                  icon: notifier.isSubmitting
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(
                              strokeWidth: 2, color: Colors.white))
                      : const Icon(Icons.send_rounded),
                  label: Text(survey.isAnswered ? '回答を更新する' : '回答する'),
                  style: FilledButton.styleFrom(
                      backgroundColor: AppColors.primary),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

// ─── 画像カルーセルセクション ────────────────────────────────────────────

class _SurveyImagesSection extends StatefulWidget {
  final String surveyId;
  final List<SurveyImage> images;

  const _SurveyImagesSection({
    required this.surveyId,
    required this.images,
  });

  @override
  State<_SurveyImagesSection> createState() => _SurveyImagesSectionState();
}

class _SurveyImagesSectionState extends State<_SurveyImagesSection> {
  final _pageController = PageController();
  int _currentPage = 0;

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final images = widget.images;

    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Column(
        children: [
          SizedBox(
            height: 220,
            child: PageView.builder(
              controller: _pageController,
              itemCount: images.length,
              onPageChanged: (i) => setState(() => _currentPage = i),
              itemBuilder: (context, i) => _ImageTile(
                surveyId: widget.surveyId,
                imageId: images[i].id,
                allImages: images,
                initialIndex: i,
              ),
            ),
          ),
          if (images.length > 1) ...[
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: List.generate(
                images.length,
                (i) => AnimatedContainer(
                  duration: const Duration(milliseconds: 200),
                  margin: const EdgeInsets.symmetric(horizontal: 3),
                  width: _currentPage == i ? 16 : 6,
                  height: 6,
                  decoration: BoxDecoration(
                    color: _currentPage == i
                        ? AppColors.primary
                        : Colors.grey[300],
                    borderRadius: BorderRadius.circular(3),
                  ),
                ),
              ),
            ),
          ],
          const SizedBox(height: 8),
        ],
      ),
    );
  }
}

class _ImageTile extends ConsumerStatefulWidget {
  final String surveyId;
  final String imageId;
  final List<SurveyImage> allImages;
  final int initialIndex;

  const _ImageTile({
    required this.surveyId,
    required this.imageId,
    required this.allImages,
    required this.initialIndex,
  });

  @override
  ConsumerState<_ImageTile> createState() => _ImageTileState();
}

class _ImageTileState extends ConsumerState<_ImageTile> {
  late Future<Uint8List> _bytesFuture;

  @override
  void initState() {
    super.initState();
    _bytesFuture = ref
        .read(surveyRepositoryProvider)
        .getImageBytes(widget.surveyId, widget.imageId);
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () => _openFullScreen(context),
      child: FutureBuilder<Uint8List>(
        future: _bytesFuture,
        builder: (context, snap) {
          if (snap.connectionState == ConnectionState.waiting) {
            return Container(
              color: Colors.grey[100],
              child: const Center(child: CircularProgressIndicator()),
            );
          }
          if (snap.hasError || snap.data == null) {
            return Container(
              color: Colors.grey[100],
              child: const Center(
                child: Icon(Icons.broken_image_rounded,
                    color: Colors.grey, size: 48),
              ),
            );
          }
          return ClipRRect(
            borderRadius: BorderRadius.circular(10),
            child: Image.memory(
              snap.data!,
              fit: BoxFit.cover,
              width: double.infinity,
            ),
          );
        },
      ),
    );
  }

  void _openFullScreen(BuildContext context) {
    showDialog(
      context: context,
      builder: (_) => _FullScreenImagesDialog(
        surveyId: widget.surveyId,
        images: widget.allImages,
        initialIndex: widget.initialIndex,
      ),
    );
  }
}

class _FullScreenImagesDialog extends ConsumerStatefulWidget {
  final String surveyId;
  final List<SurveyImage> images;
  final int initialIndex;

  const _FullScreenImagesDialog({
    required this.surveyId,
    required this.images,
    required this.initialIndex,
  });

  @override
  ConsumerState<_FullScreenImagesDialog> createState() =>
      _FullScreenImagesDialogState();
}

class _FullScreenImagesDialogState
    extends ConsumerState<_FullScreenImagesDialog> {
  late final PageController _pageController;
  int _currentPage = 0;
  final Map<String, Future<Uint8List>> _futures = {};

  @override
  void initState() {
    super.initState();
    _currentPage = widget.initialIndex;
    _pageController = PageController(initialPage: widget.initialIndex);
    final repo = ref.read(surveyRepositoryProvider);
    for (final img in widget.images) {
      _futures[img.id] =
          repo.getImageBytes(widget.surveyId, img.id);
    }
  }

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Dialog.fullscreen(
      child: Scaffold(
        backgroundColor: Colors.black,
        appBar: AppBar(
          backgroundColor: Colors.black,
          foregroundColor: Colors.white,
          title: widget.images.length > 1
              ? Text('${_currentPage + 1} / ${widget.images.length}')
              : null,
          leading: IconButton(
            icon: const Icon(Icons.close_rounded),
            onPressed: () => Navigator.of(context).pop(),
          ),
        ),
        body: PageView.builder(
          controller: _pageController,
          itemCount: widget.images.length,
          onPageChanged: (i) => setState(() => _currentPage = i),
          itemBuilder: (context, i) {
            final imgId = widget.images[i].id;
            return FutureBuilder<Uint8List>(
              future: _futures[imgId],
              builder: (context, snap) {
                if (snap.connectionState == ConnectionState.waiting) {
                  return const Center(
                      child: CircularProgressIndicator(color: Colors.white));
                }
                if (snap.hasError || snap.data == null) {
                  return const Center(
                    child: Icon(Icons.broken_image_rounded,
                        color: Colors.grey, size: 64),
                  );
                }
                return InteractiveViewer(
                  child: Center(
                    child: Image.memory(snap.data!, fit: BoxFit.contain),
                  ),
                );
              },
            );
          },
        ),
      ),
    );
  }
}

// ─── 質問カード ──────────────────────────────────────────────────────────

class _QuestionCard extends StatelessWidget {
  final int index;
  final SurveyQuestion question;
  final Set<String> selected;
  final bool canAnswer;
  final void Function(String choiceId) onChoiceToggled;

  const _QuestionCard({
    required this.index,
    required this.question,
    required this.selected,
    required this.canAnswer,
    required this.onChoiceToggled,
  });

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
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: AppColors.primary,
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Text(
                    'Q$index',
                    style: const TextStyle(
                        color: Colors.white,
                        fontSize: 11,
                        fontWeight: FontWeight.bold),
                  ),
                ),
                const SizedBox(width: 8),
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: Colors.grey[100],
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    question.questionType == 'single' ? '単一選択' : '複数選択',
                    style:
                        TextStyle(fontSize: 10, color: Colors.grey[600]),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            Text(
              question.questionText,
              style: const TextStyle(
                  fontWeight: FontWeight.w600, fontSize: 15),
            ),
            const SizedBox(height: 12),
            ...question.choices.map((c) {
              final isSelected = selected.contains(c.id);
              return Padding(
                padding: const EdgeInsets.only(bottom: 6),
                child: InkWell(
                  onTap: canAnswer ? () => onChoiceToggled(c.id) : null,
                  borderRadius: BorderRadius.circular(8),
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 12, vertical: 10),
                    decoration: BoxDecoration(
                      color: isSelected
                          ? AppColors.primary.withOpacity(0.1)
                          : Colors.grey[50],
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(
                        color: isSelected
                            ? AppColors.primary
                            : Colors.grey[300]!,
                      ),
                    ),
                    child: Row(
                      children: [
                        Icon(
                          question.questionType == 'single'
                              ? (isSelected
                                  ? Icons.radio_button_checked_rounded
                                  : Icons.radio_button_unchecked_rounded)
                              : (isSelected
                                  ? Icons.check_box_rounded
                                  : Icons.check_box_outline_blank_rounded),
                          color: isSelected ? AppColors.primary : Colors.grey,
                          size: 20,
                        ),
                        const SizedBox(width: 10),
                        Expanded(
                          child: Text(c.choiceText,
                              style: TextStyle(
                                color: isSelected
                                    ? AppColors.primaryDark
                                    : Colors.black87,
                                fontWeight: isSelected
                                    ? FontWeight.w600
                                    : FontWeight.normal,
                              )),
                        ),
                      ],
                    ),
                  ),
                ),
              );
            }),
          ],
        ),
      ),
    );
  }
}
