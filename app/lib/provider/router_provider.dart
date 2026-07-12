import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/components/base/wanderer_layout.dart';
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/routes/global_search_screen.dart';
import 'package:wanderer/routes/home_screen.dart';
import 'package:wanderer/routes/library_detail_screen.dart';
import 'package:wanderer/routes/library_screen.dart';
import 'package:wanderer/routes/list_detail_map_screen.dart';
import 'package:wanderer/routes/list_detail_screen.dart';
import 'package:wanderer/routes/list_screen.dart';
import 'package:wanderer/routes/login_screen.dart';
import 'package:wanderer/routes/map_screen.dart';
import 'package:wanderer/routes/navigation_screen.dart';
import 'package:wanderer/routes/profile_follow_screen.dart';
import 'package:wanderer/routes/profile_list_screen.dart';
import 'package:wanderer/routes/profile_screen.dart';
import 'package:wanderer/routes/profile_share_screen.dart';
import 'package:wanderer/routes/profile_trail_screen.dart';
import 'package:wanderer/routes/register_screen.dart';
import 'package:wanderer/routes/server_selection_screen.dart';
import 'package:wanderer/routes/settings_account_screen.dart';
import 'package:wanderer/routes/settings_appearance_screen.dart';
import 'package:wanderer/routes/settings_categories_screen.dart';
import 'package:wanderer/routes/settings_language_screen.dart';
import 'package:wanderer/routes/settings_notifications_screen.dart';
import 'package:wanderer/routes/settings_privacy_screen.dart';
import 'package:wanderer/routes/settings_screen.dart';
import 'package:wanderer/routes/settings_subcategories_screen.dart';
import 'package:wanderer/routes/trail_detail_map_screen.dart';
import 'package:wanderer/routes/trail_detail_screen.dart';
import 'package:wanderer/routes/trail_filter_screen.dart';
import 'package:wanderer/routes/trail_sort_screen.dart';
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
          return '/map';
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
              path: '/map',
              builder: (context, state) {
                Geographic? initialCenter;
                double? initialZoom;
                if (state.extra is Map<String, dynamic>) {
                  final extra = state.extra as Map<String, dynamic>;
                  final lat = extra['lat'] as double?;
                  final lon = extra['lon'] as double?;
                  if (lat != null && lon != null) {
                    initialCenter = Geographic(lat: lat, lon: lon);
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
              path: '/list',
              builder: (context, state) => const ListScreen(),
            ),
            GoRoute(
              path: '/profile',
              builder: (context, state) => const ProfileScreen(handle: null),
              routes: [
                GoRoute(
                  path: 'share',
                  builder: (context, state) => const ProfileShareScreen(),
                ),
              ],
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
          path: '/settings',
          builder: (context, state) => const SettingsScreen(),
          routes: [
            GoRoute(
              path: 'account',
              builder: (context, state) => const SettingsAccountScreen(),
            ),
            GoRoute(
              path: 'privacy',
              builder: (context, state) => const SettingsPrivacyScreen(),
            ),
            GoRoute(
              path: 'language',
              builder: (context, state) => const SettingsLanguageScreen(),
            ),
            GoRoute(
              path: 'notifications',
              builder: (context, state) => const SettingsNotificationsScreen(),
            ),
            GoRoute(
              path: 'appearance',
              builder: (context, state) => const SettingsAppearanceScreen(),
            ),
            GoRoute(
              path: 'categories',
              builder: (context, state) => const SettingsCategoriesScreen(),
              routes: [
                GoRoute(
                  path: 'subcategories',
                  builder: (context, state) {
                    final extra = state.extra;
                    if (extra is! Category) {
                      // extra is lost across process restart / deep-link —
                      // fall back so the user isn't left on a crashed
                      // screen.
                      return const SettingsCategoriesScreen();
                    }
                    return SettingsSubcategoriesScreen(category: extra);
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
          path: '/trail/sort',
          builder: (context, state) => const TrailSortScreen(),
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
            GoRoute(
              path: 'navigate',
              builder: (context, state) {
                final trailId = state.pathParameters['id']!;
                final extra = state.extra;
                if (extra
                    is! (NavigateResponse, bool, ActiveNavigationEntity?)) {
                  // extra is lost across process restart / deep-link — fall back
                  // to trail detail so the user isn't left on a blank screen.
                  return TrailDetailScreen(id: trailId);
                }
                final (response, isOffline, resumeSession) = extra;
                return NavigationScreen(
                  id: trailId,
                  response: response,
                  isOffline: isOffline,
                  resumeSession: resumeSession,
                );
              },
            ),
          ],
        ),
        GoRoute(
          path: '/list/:id',
          builder: (context, state) {
            final listId = state.pathParameters['id']!;
            return ListDetailScreen(id: listId);
          },
          routes: [
            GoRoute(
              path: 'map',
              builder: (context, state) {
                final listId = state.pathParameters['id']!;
                return ListDetailMapScreen(id: listId);
              },
            ),
          ],
        ),
        GoRoute(
          path: '/profile/:handle',
          builder: (context, state) {
            final handle = state.pathParameters['handle']!;
            return ProfileScreen(handle: handle);
          },
          routes: [
            GoRoute(
              path: 'trails',
              builder: (context, state) {
                final handle = state.pathParameters['handle']!;
                return ProfileTrailScreen(handle: handle);
              },
            ),
            GoRoute(
              path: 'lists',
              builder: (context, state) {
                final handle = state.pathParameters['handle']!;
                return ProfileListScreen(handle: handle);
              },
            ),
            GoRoute(
              path: 'followers',
              builder: (context, state) {
                final handle = state.pathParameters['handle']!;
                return ProfileFollowScreen(handle: handle, type: 'followers');
              },
            ),
            GoRoute(
              path: 'following',
              builder: (context, state) {
                final handle = state.pathParameters['handle']!;
                return ProfileFollowScreen(handle: handle, type: 'following');
              },
            ),
          ],
        ),
      ],
    );
  }
}
