import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';

class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final username = ref
        .watch(authProvider)
        .maybeWhen(
          data: (user) => user?.preferredUsername ?? "Guest",
          orElse: () => "Loading...",
        );

    final baseURL = ref.watch(apiProvider).options.baseUrl;
    return Scaffold(body: Center(child: Text("Welcome, $username@$baseURL!")));
  }
}
