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

    final int selectedIndex = _calculateSelectedIndex(router.state.uri.path);
    final unselectedColor = Theme.of(
      context,
    ).colorScheme.onSurface.withValues(alpha: 0.5);

    return Scaffold(
      floatingActionButtonLocation: FloatingActionButtonLocation.centerDocked,
      floatingActionButton: FloatingActionButton(
        heroTag: 'new_trail',
        shape: const StadiumBorder(),
        elevation: 2,
        backgroundColor: Theme.of(context).brightness == Brightness.dark
            ? Theme.of(context).colorScheme.secondaryContainer
            : Theme.of(context).colorScheme.surface,
        foregroundColor: Theme.of(context).colorScheme.onSurface,
        onPressed: () => router.push('/trail/create'),
        child: const FaIcon(FontAwesomeIcons.plus),
      ),
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed,
        // BottomNavigationBar has no built-in "nothing selected" state — when
        // no nav destination matches the current route (e.g. /trail/create),
        // clamp to a valid index but make selected/unselected colors equal so
        // no item visually appears active.
        currentIndex: selectedIndex < 0 ? 0 : selectedIndex,
        backgroundColor: Theme.of(context).colorScheme.surface,
        selectedItemColor: selectedIndex < 0
            ? unselectedColor
            : Theme.of(context).brightness == Brightness.dark
            ? Theme.of(context).colorScheme.onSurface
            : Theme.of(context).primaryColor,
        unselectedItemColor: unselectedColor,
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
              backgroundColor: Theme.of(
                context,
              ).colorScheme.surfaceContainerHighest,
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
    return -1;
  }
}
