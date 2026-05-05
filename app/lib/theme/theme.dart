import 'package:flutter/material.dart';
import 'colors.dart';

class AppTheme {
  static ThemeData createTheme(Brightness brightness) {
    final isDark = brightness == Brightness.dark;

    return ThemeData(
      useMaterial3: true,
      brightness: brightness,
      fontFamily: 'IBMPlexSans',

      // Mapping your CSS variables to Flutter's ColorScheme
      colorScheme: ColorScheme(
        brightness: brightness,
        primary: isDark ? AppColors.primaryDark : AppColors.primaryLight,
        secondary: isDark ? AppColors.secondaryDark : AppColors.secondaryLight,
        onPrimary: AppColors.contentDark,
        onSecondary: isDark ? AppColors.contentLight : AppColors.contentDark,
        surface: isDark ? AppColors.backgroundDark : AppColors.backgroundLight,
        onSurface: isDark ? AppColors.contentDark : AppColors.contentLight,
        error: const Color(0xFFFEF2F2), // From your --input-background-error
        onError: const Color(0xFF4C1111),
        outline: isDark
            ? AppColors.inputBorderDark
            : AppColors.inputBorderLight,
      ),

      // Input Decoration (Tailwind's --color-input-border)
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: isDark
            ? const Color.fromRGBO(38, 40, 49, 1)
            : const Color.fromRGBO(249, 250, 251, 1),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(
            color: isDark
                ? AppColors.inputBorderDark
                : AppColors.inputBorderLight,
          ),
        ),
      ),

      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: isDark
              ? AppColors.primaryDark
              : AppColors.primaryLight,
          foregroundColor: Colors.white,
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
        ),
      ),

      dividerTheme: DividerThemeData(
        color: isDark
            ? const Color.fromRGBO(55, 65, 81, 1)
            : const Color.fromRGBO(209, 213, 219, 1),
        thickness: 1,
      ),

      sliderTheme: SliderThemeData(
        showValueIndicator: ShowValueIndicator.onDrag,
      ),
    );
  }
}
