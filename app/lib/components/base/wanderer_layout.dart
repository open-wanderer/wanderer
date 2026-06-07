import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/router_provider.dart';
import '/i18n/app_localizations.dart';

class WandererLayout extends ConsumerWidget {
  final Widget child;

  const WandererLayout({super.key, required this.child});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    final user = ref.watch(authProvider).value;

    final int currentIndex = _calculateSelectedIndex(router.state.uri.path);

    return Scaffold(
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed,
        currentIndex: currentIndex,
        selectedItemColor: Theme.of(context).primaryColor,
        unselectedItemColor: Colors.grey,
        items: [
          BottomNavigationBarItem(
            icon: const FaIcon(FontAwesomeIcons.mapLocationDot),
            label: AppLocalizations.of(context)!.trail(2),
          ),
          BottomNavigationBarItem(
            icon: const FaIcon(FontAwesomeIcons.list),
            label: AppLocalizations.of(context)!.list(2),
          ),
          BottomNavigationBarItem(
            icon: const FaIcon(FontAwesomeIcons.bookAtlas),
            label: "Library",
          ),
          BottomNavigationBarItem(
            icon: CircleAvatar(
              radius: 12,
              backgroundColor: Colors.grey.shade300,
              backgroundImage: NetworkImage(
                user?.getFileUrl(user.serverUrl, user.avatar) ??
                    "https://api.dicebear.com/7.x/initials/png?seed=${user?.preferredUsername}&backgroundType=gradientLinear",
              ),
              onBackgroundImageError: (_, _) => FaIcon(FontAwesomeIcons.user),
            ),

            label: AppLocalizations.of(context)!.profile,
          ),
        ],
        onTap: (index) {
          switch (index) {
            case 0:
              router.go('/map');
              break;
            case 1:
              router.go('/list');
              break;
            case 2:
              router.go('/library');
              break;
            case 3:
              router.go('/profile');
              break;
          }
        },
      ),
      body:
          child, // Removed Center() to let the child page handle its own layout
    );
  }

  int _calculateSelectedIndex(String path) {
    if (path.startsWith('/map')) return 0;
    if (path.startsWith('/list')) return 1;
    if (path.startsWith('/library')) return 2;
    if (path.startsWith('/profile')) return 3;
    return 0;
  }
}
