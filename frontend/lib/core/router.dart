import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../features/auth/presentation/auth_provider.dart';
import '../features/auth/presentation/forgot_password_screen.dart';
import '../features/auth/presentation/home_screen.dart';
import '../features/auth/presentation/login_screen.dart';
import '../features/auth/presentation/reset_password_screen.dart';
import '../features/files/data/models/file_model.dart';
import '../features/files/screens/file_list_screen.dart';
import '../features/files/screens/file_preview_screen.dart';
import '../features/notice/screens/notice_create_screen.dart';
import '../features/notice/screens/notice_detail_screen.dart';
import '../features/notice/screens/notice_list_screen.dart';
import '../features/account/data/models/account_model.dart';
import '../features/account/screens/account_list_screen.dart';
import '../features/account/screens/account_form_screen.dart';
import '../features/association/data/models/association_model.dart';
import '../features/association/screens/association_list_screen.dart';
import '../features/association/screens/association_form_screen.dart';
import '../features/survey/screens/survey_answer_screen.dart';
import '../features/survey/screens/survey_create_screen.dart';
import '../features/survey/screens/survey_list_screen.dart';
import '../features/survey/screens/survey_result_screen.dart';

// GoRouterはAuthNotifierをlistenable登録して認証状態変化で再評価する
final routerProvider = Provider<GoRouter>((ref) {
  final authNotifier = ref.watch(authNotifierProvider);

  return GoRouter(
    refreshListenable: authNotifier,
    initialLocation: '/',
    redirect: (context, state) {
      if (!authNotifier.initialized) return null;

      final isAuthenticated = authNotifier.isAuthenticated;
      final loc = state.matchedLocation;

      // 未認証でもアクセス可能なページ
      final publicPaths = ['/login', '/forgot-password', '/reset-password'];
      final isPublic = publicPaths.any((p) => loc.startsWith(p));

      if (!isAuthenticated && !isPublic) return '/login';
      if (isAuthenticated && loc == '/login') return '/';
      return null;
    },
    routes: [
      GoRoute(
        path: '/',
        builder: (context, state) => const HomeScreen(),
      ),
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
      ),
      // ─── パスワードリセット ───────────────────────────────────
      GoRoute(
        path: '/forgot-password',
        builder: (context, state) => const ForgotPasswordScreen(),
      ),
      GoRoute(
        path: '/reset-password',
        builder: (context, state) {
          final token = state.uri.queryParameters['token'] ?? '';
          return ResetPasswordScreen(token: token);
        },
      ),
      // ─── 回覧物 ───────────────────────────────────────────────
      GoRoute(
        path: '/files',
        builder: (context, state) => const FileListScreen(),
      ),
      GoRoute(
        path: '/files/preview',
        builder: (context, state) {
          final file = state.extra as FileModel;
          return FilePreviewScreen(file: file);
        },
      ),
      // ─── お知らせ ─────────────────────────────────────────────
      GoRoute(
        path: '/notices',
        builder: (context, state) => const NoticeListScreen(),
      ),
      GoRoute(
        path: '/notices/create',
        builder: (context, state) => const NoticeCreateScreen(),
      ),
      GoRoute(
        path: '/notices/:id',
        builder: (context, state) {
          final id = state.pathParameters['id']!;
          return NoticeDetailScreen(noticeId: id);
        },
      ),
      // ─── アカウント管理 ───────────────────────────────────────
      GoRoute(
        path: '/accounts',
        builder: (context, state) => const AccountListScreen(),
      ),
      GoRoute(
        path: '/accounts/create',
        builder: (context, state) {
          final preselectedAssocId = state.extra as String?;
          return AccountFormScreen(preselectedAssociationId: preselectedAssocId);
        },
      ),
      GoRoute(
        path: '/accounts/:id/edit',
        builder: (context, state) {
          final account = state.extra as AccountModel;
          return AccountFormScreen(account: account);
        },
      ),
      // ─── 自治会管理 ───────────────────────────────────────────
      GoRoute(
        path: '/associations',
        builder: (context, state) => const AssociationListScreen(),
      ),
      GoRoute(
        path: '/associations/create',
        builder: (context, state) => const AssociationFormScreen(),
      ),
      GoRoute(
        path: '/associations/:id/edit',
        builder: (context, state) {
          final assoc = state.extra as AssociationDetail;
          return AssociationFormScreen(association: assoc);
        },
      ),
      // ─── アンケート ───────────────────────────────────────────
      GoRoute(
        path: '/surveys',
        builder: (context, state) => const SurveyListScreen(),
      ),
      GoRoute(
        path: '/surveys/create',
        builder: (context, state) => const SurveyCreateScreen(),
      ),
      GoRoute(
        path: '/surveys/:id',
        builder: (context, state) {
          final id = state.pathParameters['id']!;
          return SurveyAnswerScreen(surveyId: id);
        },
      ),
      GoRoute(
        path: '/surveys/:id/results',
        builder: (context, state) {
          final id = state.pathParameters['id']!;
          return SurveyResultScreen(surveyId: id);
        },
      ),
    ],
    errorBuilder: (context, state) => Scaffold(
      body: Center(child: Text('ページが見つかりません: ${state.error}')),
    ),
  );
});
