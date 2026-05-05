import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../main.dart';
import '../providers/survey_provider.dart';

class SurveyCreateScreen extends ConsumerStatefulWidget {
  const SurveyCreateScreen({super.key});

  @override
  ConsumerState<SurveyCreateScreen> createState() => _SurveyCreateScreenState();
}

class _SurveyCreateScreenState extends ConsumerState<SurveyCreateScreen> {
  final _formKey = GlobalKey<FormState>();
  final _titleCtrl = TextEditingController();
  final _descCtrl = TextEditingController();
  DateTime? _expiresAt;

  final List<_QuestionDraft> _questions = [];

  @override
  void dispose() {
    _titleCtrl.dispose();
    _descCtrl.dispose();
    super.dispose();
  }

  Future<void> _pickExpiry() async {
    final now = DateTime.now();
    final date = await showDatePicker(
      context: context,
      initialDate: now.add(const Duration(days: 7)),
      firstDate: now,
      lastDate: now.add(const Duration(days: 365)),
    );
    if (date == null || !mounted) return;
    final time = await showTimePicker(
      context: context,
      initialTime: const TimeOfDay(hour: 23, minute: 59),
    );
    if (time == null || !mounted) return;
    setState(() {
      _expiresAt = DateTime(
          date.year, date.month, date.day, time.hour, time.minute);
    });
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    if (_expiresAt == null) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
        content: Text('回答期限を設定してください'),
        backgroundColor: Colors.orange,
      ));
      return;
    }
    if (_questions.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
        content: Text('質問を1つ以上追加してください'),
        backgroundColor: Colors.orange,
      ));
      return;
    }
    for (final q in _questions) {
      if (q.choices.length < 2) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text('「${q.text}」の選択肢を2つ以上追加してください'),
          backgroundColor: Colors.orange,
        ));
        return;
      }
    }

    final questions = _questions.asMap().entries.map((e) {
      final q = e.value;
      return {
        'question_text': q.text,
        'question_type': q.type,
        'sort_order': e.key + 1,
        'choices': q.choices.asMap().entries.map((ce) => {
              'choice_text': ce.value,
              'sort_order': ce.key + 1,
            }).toList(),
      };
    }).toList();

    final ok = await ref.read(surveyNotifierProvider).create(
          title: _titleCtrl.text.trim(),
          description: _descCtrl.text.trim(),
          expiresAt: _expiresAt!,
          questions: questions,
        );

    if (!mounted) return;
    if (ok) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
        content: Text('アンケートを作成しました'),
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

    return Scaffold(
      appBar: AppBar(
        title: const Text('アンケート作成'),
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // エラー表示
              if (notifier.errorMessage != null)
                Container(
                  margin: const EdgeInsets.only(bottom: 12),
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.red[50],
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.red[200]!),
                  ),
                  child: Text(notifier.errorMessage!,
                      style: TextStyle(color: Colors.red[700])),
                ),

              // タイトル
              TextFormField(
                controller: _titleCtrl,
                decoration: const InputDecoration(
                  labelText: 'タイトル *',
                  prefixIcon: Icon(Icons.title_rounded),
                ),
                validator: (v) =>
                    (v == null || v.trim().isEmpty) ? 'タイトルを入力してください' : null,
              ),
              const SizedBox(height: 14),

              // 説明
              TextFormField(
                controller: _descCtrl,
                maxLines: 3,
                decoration: const InputDecoration(
                  labelText: '説明（任意）',
                  prefixIcon: Icon(Icons.description_rounded),
                  alignLabelWithHint: true,
                ),
              ),
              const SizedBox(height: 14),

              // 回答期限
              InkWell(
                onTap: _pickExpiry,
                borderRadius: BorderRadius.circular(8),
                child: InputDecorator(
                  decoration: const InputDecoration(
                    labelText: '回答期限 *',
                    prefixIcon: Icon(Icons.calendar_today_rounded),
                  ),
                  child: Text(
                    _expiresAt != null
                        ? '${_expiresAt!.year}/${_expiresAt!.month.toString().padLeft(2, '0')}/${_expiresAt!.day.toString().padLeft(2, '0')} '
                          '${_expiresAt!.hour.toString().padLeft(2, '0')}:${_expiresAt!.minute.toString().padLeft(2, '0')}'
                        : '日時を選択',
                    style: TextStyle(
                      color: _expiresAt != null ? null : Colors.grey[500],
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 20),

              // 質問セクション
              Row(
                children: [
                  const Text('質問',
                      style: TextStyle(
                          fontSize: 15, fontWeight: FontWeight.bold)),
                  const Spacer(),
                  TextButton.icon(
                    onPressed: () => setState(() => _questions.add(_QuestionDraft())),
                    icon: const Icon(Icons.add_rounded, size: 18),
                    label: const Text('追加'),
                  ),
                ],
              ),
              const Divider(),

              if (_questions.isEmpty)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 12),
                  child: Text('質問を追加してください',
                      style: TextStyle(color: Colors.grey[500])),
                ),

              ..._questions.asMap().entries.map((e) => _QuestionEditor(
                    index: e.key + 1,
                    draft: e.value,
                    onRemove: () =>
                        setState(() => _questions.removeAt(e.key)),
                    onChanged: () => setState(() {}),
                  )),

              const SizedBox(height: 24),

              // 作成ボタン
              SizedBox(
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
                  label: const Text('作成する'),
                  style: FilledButton.styleFrom(
                      backgroundColor: AppColors.primary),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _QuestionDraft {
  String text = '';
  String type = 'single';
  List<String> choices = ['', ''];
}

class _QuestionEditor extends StatefulWidget {
  final int index;
  final _QuestionDraft draft;
  final VoidCallback onRemove;
  final VoidCallback onChanged;

  const _QuestionEditor({
    required this.index,
    required this.draft,
    required this.onRemove,
    required this.onChanged,
  });

  @override
  State<_QuestionEditor> createState() => _QuestionEditorState();
}

class _QuestionEditorState extends State<_QuestionEditor> {
  late final TextEditingController _textCtrl;

  @override
  void initState() {
    super.initState();
    _textCtrl = TextEditingController(text: widget.draft.text);
  }

  @override
  void dispose() {
    _textCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(color: Colors.grey[300]!),
      ),
      elevation: 0,
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ヘッダー
            Row(
              children: [
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: AppColors.primary,
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Text('Q${widget.index}',
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 11,
                          fontWeight: FontWeight.bold)),
                ),
                const Spacer(),
                IconButton(
                  icon: const Icon(Icons.delete_outline_rounded,
                      color: Colors.red, size: 20),
                  onPressed: widget.onRemove,
                ),
              ],
            ),
            const SizedBox(height: 8),

            // 質問テキスト
            TextField(
              controller: _textCtrl,
              decoration: const InputDecoration(
                hintText: '質問を入力 *',
                isDense: true,
              ),
              onChanged: (v) {
                widget.draft.text = v;
                widget.onChanged();
              },
            ),
            const SizedBox(height: 8),

            // 選択タイプ
            Row(
              children: [
                const Text('回答タイプ: ',
                    style: TextStyle(fontSize: 12)),
                ChoiceChip(
                  label: const Text('単一選択', style: TextStyle(fontSize: 12)),
                  selected: widget.draft.type == 'single',
                  onSelected: (_) {
                    setState(() => widget.draft.type = 'single');
                    widget.onChanged();
                  },
                ),
                const SizedBox(width: 6),
                ChoiceChip(
                  label: const Text('複数選択', style: TextStyle(fontSize: 12)),
                  selected: widget.draft.type == 'multiple',
                  onSelected: (_) {
                    setState(() => widget.draft.type = 'multiple');
                    widget.onChanged();
                  },
                ),
              ],
            ),
            const SizedBox(height: 8),

            // 選択肢
            ...widget.draft.choices.asMap().entries.map((e) {
              final ctrl = TextEditingController(text: e.value);
              return Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Row(
                  children: [
                    const Icon(Icons.circle_outlined,
                        size: 14, color: Colors.grey),
                    const SizedBox(width: 6),
                    Expanded(
                      child: TextField(
                        controller: ctrl,
                        decoration: InputDecoration(
                          hintText: '選択肢 ${e.key + 1}',
                          isDense: true,
                        ),
                        onChanged: (v) {
                          widget.draft.choices[e.key] = v;
                          widget.onChanged();
                        },
                      ),
                    ),
                    if (widget.draft.choices.length > 2)
                      IconButton(
                        icon: const Icon(Icons.close_rounded,
                            size: 16, color: Colors.grey),
                        onPressed: () {
                          setState(() => widget.draft.choices.removeAt(e.key));
                          widget.onChanged();
                        },
                      ),
                  ],
                ),
              );
            }),
            TextButton.icon(
              onPressed: () {
                setState(() => widget.draft.choices.add(''));
                widget.onChanged();
              },
              icon: const Icon(Icons.add_rounded, size: 16),
              label: const Text('選択肢を追加', style: TextStyle(fontSize: 12)),
            ),
          ],
        ),
      ),
    );
  }
}
