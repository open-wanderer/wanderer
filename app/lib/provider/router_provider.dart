import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/components/base/wanderer_layout.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/routes/home_screen.dart';
import 'package:wanderer/routes/login_screen.dart';
import 'package:wanderer/routes/server_selection_screen.dart';
import 'package:wanderer/routes/welcome_screen.dart';

part 'router_provider.g.dart';

@riverpod
class Router extends _$Router {
  @override
  GoRouter build() {
    final authState = ref.watch(authProvider);

    return GoRouter(
      initialLocation: '/',
      redirect: (context, state) {
        final bool loggedIn = authState.asData?.value != null;

        final unprotectedRoutes = ['/welcome', '/login', '/select-server'];

        if (!loggedIn && !unprotectedRoutes.contains(state.matchedLocation)) {
          return '/welcome';
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
          ],
        ),
        GoRoute(path: '/welcome', builder: (context, state) => WelcomeScreen()),
        GoRoute(
          path: '/select-server',
          builder: (context, state) => ServerSelectionScreen(),
        ),
        GoRoute(path: '/login', builder: (context, state) => LoginScreen()),
      ],
    );
  }
}
