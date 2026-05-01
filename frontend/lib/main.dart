import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/router.dart';
import 'features/auth/presentation/auth_provider.dart';

void main() {
  runApp(const ProviderScope(child: CircularBoardApp()));
}

class CircularBoardApp extends ConsumerStatefulWidget {
  const CircularBoardApp({super.key});

  @override
  ConsumerState<CircularBoardApp> createState() => _CircularBoardAppState();
}

class _CircularBoardAppState extends ConsumerState<CircularBoardApp> {
  @override
  void initState() {
    super.initState();
    // アプリ起動時にセッション復元を試みる
    Future.microtask(() => ref.read(authNotifierProvider).initialize());
  }

  @override
  Widget build(BuildContext context) {
    final router = ref.watch(routerProvider);

    return MaterialApp.router(
      title: '回覧板',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF2E7D32),
          brightness: Brightness.light,
        ),
        useMaterial3: true,
        inputDecorationTheme: const InputDecorationTheme(
          border: OutlineInputBorder(),
          contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        ),
      ),
      routerConfig: router,
    );
  }
}
