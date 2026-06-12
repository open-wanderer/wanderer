import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/models/navigate_response.dart';

// Stub — full implementation in Task 2
class NavigationScreen extends ConsumerWidget {
  final String id;
  final NavigateResponse response;

  const NavigationScreen({super.key, required this.id, required this.response});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return const Scaffold(body: Center(child: CircularProgressIndicator()));
  }
}
