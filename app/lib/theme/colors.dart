import 'package:flutter/material.dart';

class AppColors {
  static Color _rgb(int r, int g, int b) => Color.fromARGB(255, r, g, b);

  // Light Mode
  static final primaryLight = _rgb(36, 39, 52);
  static final secondaryLight = _rgb(229, 231, 235);
  static final backgroundLight = _rgb(255, 255, 255);
  static final contentLight = _rgb(0, 0, 0);
  static final inputBorderLight = _rgb(209, 213, 219);

  // Dark Mode
  static final primaryDark = _rgb(36, 39, 52);
  static final secondaryDark = _rgb(45, 50, 65);
  static final backgroundDark = _rgb(18, 20, 28);
  static final contentDark = _rgb(255, 255, 255);
  static final inputBorderDark = _rgb(50, 54, 64);
}
