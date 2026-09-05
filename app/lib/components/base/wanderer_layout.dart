import 'package:cached_network_image/cached_network_image.dart';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/router_provider.dart';
import 'package:wanderer/store/avatar_cache.dart';
import '/i18n/app_localizations.dart';

class WandererLayout extends ConsumerWidget {
  final Widget child;

  const WandererLayout({super.key, required this.child});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    final user = ref.watch(authProvider).value;
    final isOnline = ref.watch(onlineStatusProvider);

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
        // `extendBody: true` puts the page content directly behind the bar, and
        // Material alone doesn't absorb pointer events — without this, taps in
        // the gaps between nav items fall through to the page underneath.
        child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: () {},
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
                icon: _NavAvatar(user: user, isOnline: isOnline),
                label: AppLocalizations.of(context)!.profile,
                selected: selectedIndex == 3,
                selectedColor: selectedColor,
                unselectedColor: unselectedColor,
                onTap: () => onTap(3),
              ),
            ],
          ),
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

/// Bottom-nav profile avatar with a cached-file / network / glyph fallback
/// chain: a cached on-disk avatar (from a prior online session) always wins
/// when present, then a live network fetch while online, then a plain user
/// glyph while offline with nothing cached — never a blank grey circle.
///
/// `cachedAvatarFile` is async, so its result is held in a future that's
/// only recomputed when the user id or avatar filename actually changes
/// (`didUpdateWidget`), not on every rebuild/frame.
class _NavAvatar extends StatefulWidget {
  final UserEntity? user;
  final bool isOnline;

  const _NavAvatar({required this.user, required this.isOnline});

  @override
  State<_NavAvatar> createState() => _NavAvatarState();
}

class _NavAvatarState extends State<_NavAvatar> {
  late Future<File?> _avatarFuture;

  // Set when a NetworkImage load fails (e.g. connectivity drops mid-fetch) —
  // triggers the glyph fallback instead of leaving a blank grey circle
  // (the previous bug here: `onBackgroundImageError` returned a discarded
  // `FaIcon` from a void callback, which never rendered anything).
  bool _networkFailed = false;

  @override
  void initState() {
    super.initState();
    _avatarFuture = _resolveAvatarFuture();
  }

  @override
  void didUpdateWidget(covariant _NavAvatar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.user?.id != widget.user?.id ||
        oldWidget.user?.avatar != widget.user?.avatar) {
      setState(() {
        _networkFailed = false;
        _avatarFuture = _resolveAvatarFuture();
      });
    } else if (!oldWidget.isOnline && widget.isOnline && _networkFailed) {
      // Coming back online — give the network image another chance.
      setState(() => _networkFailed = false);
    }
  }

  Future<File?> _resolveAvatarFuture() {
    final user = widget.user;
    if (user == null) return Future.value(null);
    return cachedAvatarFile(user.id, user.avatar);
  }

  @override
  Widget build(BuildContext context) {
    final user = widget.user;
    final backgroundColor = Theme.of(
      context,
    ).colorScheme.surfaceContainerHighest;

    return FutureBuilder<File?>(
      future: _avatarFuture,
      builder: (context, snapshot) {
        final cachedFile = snapshot.data;
        if (cachedFile != null) {
          return CircleAvatar(
            radius: 12,
            backgroundColor: backgroundColor,
            backgroundImage: FileImage(cachedFile),
          );
        }

        if (widget.isOnline && !_networkFailed) {
          return CircleAvatar(
            radius: 12,
            backgroundColor: backgroundColor,
            backgroundImage: CachedNetworkImageProvider(
              user?.getFileUrl(user.serverUrl, user.avatar) ??
                  "https://api.dicebear.com/7.x/initials/png?seed=${user?.preferredUsername}&backgroundType=gradientLinear",
            ),
            onBackgroundImageError: (_, _) {
              if (mounted) setState(() => _networkFailed = true);
            },
          );
        }

        return CircleAvatar(
          radius: 12,
          backgroundColor: backgroundColor,
          child: const FaIcon(FontAwesomeIcons.user, size: 12),
        );
      },
    );
  }
}
