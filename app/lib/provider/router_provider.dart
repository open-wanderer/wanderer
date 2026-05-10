import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/components/base/wanderer_layout.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/routes/home_screen.dart';
import 'package:wanderer/routes/login_screen.dart';
import 'package:wanderer/routes/map_screen.dart';
import 'package:wanderer/routes/profile_screen.dart';
import 'package:wanderer/routes/register_screen.dart';
import 'package:wanderer/routes/server_selection_screen.dart';
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

        final unprotectedRoutes = [
          '/',
          '/welcome',
          '/login',
          '/register',
          '/select-server',
        ];

        if (!loggedIn && !unprotectedRoutes.contains(state.matchedLocation)) {
          return '/welcome';
        }

        if (loggedIn && unprotectedRoutes.contains(state.matchedLocation)) {
          return '/trail';
        }

        return null;
      },
      routes: [
        ShellRoute(
          builder: (BuildContext context, GoRouterState state, Widget child) {
            return WandererLayout(child: child);
          },
          routes: [
            GoRoute(path: '/', builder: (context, state) => const HomeScreen()),
            GoRoute(
              path: '/trail',
              builder: (context, state) => const TrailScreen(),
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
            ),
            GoRoute(path: '/map', builder: (context, state) => MapScreen()),
            GoRoute(
              path: '/profile',
              builder: (context, state) => ProfileScreen(),
            ),
          ],
        ),
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
      ],
    );
  }
}
