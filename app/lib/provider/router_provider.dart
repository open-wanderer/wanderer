import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:latlong2/latlong.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/components/base/wanderer_layout.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/routes/library_detail_screen.dart';
import 'package:wanderer/routes/library_screen.dart';
import 'package:wanderer/routes/home_screen.dart';
import 'package:wanderer/routes/login_screen.dart';
import 'package:wanderer/routes/map_screen.dart';
import 'package:wanderer/routes/profile_screen.dart';
import 'package:wanderer/routes/register_screen.dart';
import 'package:wanderer/routes/server_selection_screen.dart';
import 'package:wanderer/routes/global_search_screen.dart';
import 'package:wanderer/routes/trail_detail_map_screen.dart';
import 'package:wanderer/routes/trail_detail_screen.dart';
import 'package:wanderer/routes/trail_filter_screen.dart';
import 'package:wanderer/routes/trail_screen.dart';
import 'package:wanderer/routes/welcome_screen.dart';

part 'router_provider.g.dart';

final navigatorKey = GlobalKey<NavigatorState>();

class RouterNotifier extends ChangeNotifier {
  void notify() => notifyListeners();
}

@riverpod
Listenable routerListenable(Ref ref) {
  final notifier = RouterNotifier();

  ref.listen(authProvider, (previous, next) {
    if (!next.isLoading) {
      notifier.notify();
    }
  });

  return notifier;
}

@riverpod
class Router extends _$Router {
  @override
  GoRouter build() {
    final refreshListenable = ref.watch(routerListenableProvider);

    return GoRouter(
      initialLocation: '/',
      navigatorKey: navigatorKey,
      refreshListenable: refreshListenable,
      redirect: (context, state) {
        final authState = ref.read(authProvider);

        if (authState.isLoading && !authState.hasValue) {
          return null;
        }

        final user = authState.value;
        final bool loggedIn = user != null;
        final String location = state.matchedLocation;

        final authRoutes = [
          '/login',
          '/register',
          '/welcome',
          '/select-server',
        ];
        final isAtSplash = location == '/';
        final isAtAuthRoute = authRoutes.contains(location);

        if (!loggedIn) {
          if (isAtSplash || !isAtAuthRoute) {
            return '/welcome';
          }
          return null; // Stay on login/register/etc.
        }

        if (loggedIn && (isAtSplash || isAtAuthRoute)) {
          return '/trail';
        }

        return null;
      },
      routes: [
        GoRoute(path: '/', builder: (context, state) => const HomeScreen()),

        GoRoute(path: '/welcome', builder: (context, state) => WelcomeScreen()),
        GoRoute(
          path: '/select-server',
          builder: (context, state) => ServerSelectionScreen(),
        ),
        GoRoute(path: '/login', builder: (context, state) => LoginScreen()),
        GoRoute(
          path: '/register',
          builder: (context, state) => RegisterScreen(),
        ),

        ShellRoute(
          builder: (BuildContext context, GoRouterState state, Widget child) {
            return WandererLayout(child: child);
          },
          routes: [
            GoRoute(
              path: '/trail',
              builder: (context, state) => const TrailScreen(),
              routes: [
                // Sub-routes keep the /trail prefix but stay inside the Shell
              ],
            ),
            GoRoute(
              path: '/map',
              builder: (context, state) {
                LatLng? initialCenter;
                double? initialZoom;
                if (state.extra is Map<String, dynamic>) {
                  final extra = state.extra as Map<String, dynamic>;
                  final lat = extra['lat'] as double?;
                  final lon = extra['lon'] as double?;
                  if (lat != null && lon != null) {
                    initialCenter = LatLng(lat, lon);
                  }
                  initialZoom = extra['zoom'] as double?;
                }
                return MapScreen(
                  initialCenter: initialCenter,
                  initialZoom: initialZoom,
                );
              },
            ),
            GoRoute(
              path: '/profile',
              builder: (context, state) => ProfileScreen(),
            ),
            GoRoute(
              path: '/library',
              builder: (context, state) => LibraryScreen(),
              routes: [
                GoRoute(
                  path: ':id',
                  builder: (context, state) {
                    final trailId = state.pathParameters['id']!;
                    return LibraryDetailScreen(id: trailId);
                  },
                ),
              ],
            ),
          ],
        ),
        GoRoute(
          path: '/search',
          builder: (context, state) => const GlobalSearchScreen(),
        ),
        GoRoute(
          path: '/trail/filter',
          builder: (context, state) => const TrailFilterScreen(),
        ),
        GoRoute(
          path: '/trail/:id',
          builder: (context, state) {
            final trailId = state.pathParameters['id']!;
            return TrailDetailScreen(id: trailId);
          },
          routes: [
            GoRoute(
              path: 'map',
              builder: (context, state) {
                final trailId = state.pathParameters['id']!;
                return TrailDetailMapScreen(id: trailId);
              },
            ),
          ],
        ),
      ],
    );
  }
}
