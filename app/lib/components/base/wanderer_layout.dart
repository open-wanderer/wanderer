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

    final selectedColor = selectedIndex < 0
        ? unselectedColor
        : Theme.of(context).brightness == Brightness.dark
        ? Theme.of(context).colorScheme.onSurface
        : Theme.of(context).primaryColor;

    void onTap(int index) {
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
    }

    return Scaffold(
      extendBody: true,
      floatingActionButtonLocation: FloatingActionButtonLocation.centerDocked,
      floatingActionButton: FloatingActionButton(
        heroTag: 'new_trail',
        shape: const StadiumBorder(),
        elevation: 2,
        backgroundColor: Theme.of(context).brightness == Brightness.dark
            ? Theme.of(context).colorScheme.secondaryContainer
            : Theme.of(context).colorScheme.surface,
        foregroundColor: Theme.of(context).colorScheme.onSurface,
        onPressed: () => router.pushReplacement('/trail/create'),
        child: const FaIcon(FontAwesomeIcons.plus, size: 18),
      ),
      bottomNavigationBar: BottomAppBar(
        shape: const CircularNotchedRectangle(),
        notchMargin: 8,
        color: Theme.of(context).colorScheme.surface,
        height: kBottomNavigationBarHeight,
        padding: EdgeInsets.zero,
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceAround,
          children: [
            _NavItem(
              icon: const FaIcon(FontAwesomeIcons.mapLocationDot),
              label: AppLocalizations.of(context)!.trail(2),
              selected: selectedIndex == 0,
              selectedColor: selectedColor,
              unselectedColor: unselectedColor,
              onTap: () => onTap(0),
            ),
            _NavItem(
              icon: const FaIcon(FontAwesomeIcons.list),
              label: AppLocalizations.of(context)!.list(2),
              selected: selectedIndex == 1,
              selectedColor: selectedColor,
              unselectedColor: unselectedColor,
              onTap: () => onTap(1),
            ),
            const SizedBox(width: kBottomNavigationBarHeight - 16),
            _NavItem(
              icon: const FaIcon(FontAwesomeIcons.bookAtlas),
              label: AppLocalizations.of(context)!.library,
              selected: selectedIndex == 2,
              selectedColor: selectedColor,
              unselectedColor: unselectedColor,
              onTap: () => onTap(2),
            ),
            _NavItem(
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
              selected: selectedIndex == 3,
              selectedColor: selectedColor,
              unselectedColor: unselectedColor,
              onTap: () => onTap(3),
            ),
          ],
        ),
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

class _NavItem extends StatelessWidget {
  final Widget icon;
  final String label;
  final bool selected;
  final Color selectedColor;
  final Color unselectedColor;
  final VoidCallback onTap;

  const _NavItem({
    required this.icon,
    required this.label,
    required this.selected,
    required this.selectedColor,
    required this.unselectedColor,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final color = selected ? selectedColor : unselectedColor;

    return InkWell(
      onTap: onTap,
      customBorder: const StadiumBorder(),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconTheme(
              data: IconThemeData(color: color, size: 22),
              child: icon,
            ),
            const SizedBox(height: 2),
            Text(
              label,
              style: Theme.of(
                context,
              ).textTheme.labelSmall?.copyWith(color: color),
            ),
          ],
        ),
      ),
    );
  }
}
