import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/router_provider.dart';

class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authProvider.notifier);
    final router = ref.watch(routerProvider);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        spacing: 12,
        children: [
          WandererButton(
            primary: true,
            child: Text("Logout"),
            onPressed: () => auth.logout(),
          ),
          WandererButton(
            primary: true,
            child: Text("Downloads"),
            onPressed: () => router.push('/library'),
          ),
        ],
      ),
    );
  }
}
