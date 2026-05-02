import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_fonts/google_fonts.dart';

import 'core/router.dart';
import 'features/auth/presentation/auth_provider.dart';

// ブランドカラー定数
class AppColors {
  static const primary = Color(0xFF2D6A4F);
  static const primaryLight = Color(0xFF40916C);
  static const primaryDark = Color(0xFF1B4332);
  static const accent = Color(0xFF52B788);
  static const onPrimary = Colors.white;
}

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
    Future.microtask(() => ref.read(authNotifierProvider).initialize());
  }

  @override
  Widget build(BuildContext context) {
    final router = ref.watch(routerProvider);

    // Noto Sans JP をベースに全テキストテーマを構築
    final textTheme = GoogleFonts.notoSansJpTextTheme();

    return MaterialApp.router(
      title: '回覧板',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: AppColors.primary,
          brightness: Brightness.light,
        ).copyWith(
          primary: AppColors.primary,
          onPrimary: AppColors.onPrimary,
          secondary: AppColors.primaryLight,
          tertiary: AppColors.accent,
        ),
        textTheme: textTheme,
        primaryTextTheme: textTheme,
        useMaterial3: true,
        appBarTheme: const AppBarTheme(
          backgroundColor: AppColors.primary,
          foregroundColor: AppColors.onPrimary,
          elevation: 0,
        ),
        filledButtonTheme: FilledButtonThemeData(
          style: FilledButton.styleFrom(
            backgroundColor: AppColors.primary,
            foregroundColor: AppColors.onPrimary,
          ),
        ),
        inputDecorationTheme: InputDecorationTheme(
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(10),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(10),
            borderSide: const BorderSide(color: AppColors.primary, width: 2),
          ),
          prefixIconColor: AppColors.primaryLight,
          contentPadding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        ),
      ),
      routerConfig: router,
    );
  }
}
