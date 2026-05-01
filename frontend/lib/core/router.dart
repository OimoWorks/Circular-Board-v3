import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../features/auth/presentation/auth_provider.dart';
import '../features/auth/presentation/home_screen.dart';
import '../features/auth/presentation/login_screen.dart';
import '../features/files/screens/file_list_screen.dart';

// GoRouterはAuthNotifierをlistenable登録して認証状態変化で再評価する
final routerProvider = Provider<GoRouter>((ref) {
  final authNotifier = ref.watch(authNotifierProvider);

  return GoRouter(
    refreshListenable: authNotifier,
    initialLocation: '/',
    redirect: (context, state) {
      if (!authNotifier.initialized) return null;

      final isAuthenticated = authNotifier.isAuthenticated;
      final isOnLogin = state.matchedLocation == '/login';

      if (!isAuthenticated && !isOnLogin) return '/login';
      if (isAuthenticated && isOnLogin) return '/';
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
      GoRoute(
        path: '/files',
        builder: (context, state) => const FileListScreen(),
      ),
    ],
    errorBuilder: (context, state) => Scaffold(
      body: Center(child: Text('ページが見つかりません: ${state.error}')),
    ),
  );
});
