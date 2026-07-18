import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/components/base/wanderer_layout.dart';
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/routes/global_search_screen.dart';
import 'package:wanderer/routes/home_screen.dart';
import 'package:wanderer/routes/library_detail_screen.dart';
import 'package:wanderer/routes/library_screen.dart';
import 'package:wanderer/routes/list_detail_map_screen.dart';
import 'package:wanderer/routes/list_detail_screen.dart';
import 'package:wanderer/routes/list_screen.dart';
import 'package:wanderer/routes/location_search_screen.dart';
import 'package:wanderer/routes/login_screen.dart';
import 'package:wanderer/routes/map_screen.dart';
import 'package:wanderer/routes/navigation_screen.dart';
import 'package:wanderer/routes/profile_follow_screen.dart';
import 'package:wanderer/routes/profile_list_screen.dart';
import 'package:wanderer/routes/profile_screen.dart';
import 'package:wanderer/routes/profile_share_screen.dart';
import 'package:wanderer/routes/profile_trail_screen.dart';
import 'package:wanderer/routes/register_screen.dart';
import 'package:wanderer/routes/route_planner_screen.dart';
import 'package:wanderer/routes/server_selection_screen.dart';
import 'package:wanderer/routes/settings_account_screen.dart';
import 'package:wanderer/routes/settings_appearance_screen.dart';
import 'package:wanderer/routes/settings_categories_screen.dart';
import 'package:wanderer/routes/settings_language_screen.dart';
import 'package:wanderer/routes/settings_notifications_screen.dart';
import 'package:wanderer/routes/settings_privacy_screen.dart';
import 'package:wanderer/routes/settings_screen.dart';
import 'package:wanderer/routes/settings_subcategories_screen.dart';
import 'package:wanderer/routes/trail_create_screen.dart';
import 'package:wanderer/routes/trail_detail_map_screen.dart';
import 'package:wanderer/routes/trail_detail_screen.dart';
import 'package:wanderer/routes/trail_filter_screen.dart';
import 'package:wanderer/routes/trail_source_select_screen.dart';
import 'package:wanderer/routes/trail_sort_screen.dart';
import 'package:wanderer/routes/waypoint_create_screen.dart';
import 'package:wanderer/routes/welcome_screen.dart';
import 'package:wanderer/util/trail_import_util.dart';

part 'router_provider.g.dart';

final navigatorKey = GlobalKey<NavigatorState>();

class RouterNotifier extends ChangeNotifier {
  void notify() => notifyListeners();
}

@riverpod
Listenable routerListenable(Ref ref) {
  final notifier = RouterNotifier();

  ref.listen<AsyncValue<UserEntity?>>(authProvider, (
    AsyncValue<UserEntity?>? previous,
    AsyncValue<UserEntity?> next,
  ) {
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
            GoRoute(
              path: '/trail/create',
              builder: (context, state) => const TrailSourceSelectScreen(),
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
                      // extra is lost across process restart / deep-link — fall back.
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
          path: '/location-search',
          builder: (context, state) => const LocationSearchScreen(),
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
          path: '/route-planner',
          builder: (context, state) {
            final extra = state.extra as Map<String, dynamic>?;
            final profile = extra?['travelProfile'] as String? ?? 'pedestrian';
            final costingOptions =
                extra?['costingOptions'] as Map<String, dynamic>?;
            final lat = extra?['lat'] as double?;
            final lon = extra?['lon'] as double?;
            final center = (lat != null && lon != null)
                ? Geographic(lat: lat, lon: lon)
                : const Geographic(lat: 0, lon: 0);
            // Edit-mode seed anchors; edit mode is inferred from this list being
            // non-empty, not from a separate flag.
            final seedAnchors = extra?['seedAnchors'] as List<Geographic>?;
            // Full-resolution per-segment polylines aligned with seedAnchors —
            // preserves the original recorded route until the user edits it.
            final seedSegmentPolylines =
                extra?['seedSegmentPolylines'] as List<List<Geographic>>?;
            return RoutePlannerScreen(
              travelProfile: profile,
              initialCostingOptions: costingOptions,
              initialCenter: center,
              seedAnchors: seedAnchors,
              seedSegmentPolylines: seedSegmentPolylines,
            );
          },
        ),
        GoRoute(
          path: '/trail/create/edit',
          builder: (context, state) {
            final extra = state.extra;
            if (extra is Trail) {
              pendingImportedTrail = null;
              return TrailCreateScreen(trail: extra);
            }
            // extra is lost across process restart / deep-link, or a same-process
            // router refresh mid-navigation — recover a pending imported trail if
            // any, else fall back to the source selector.
            final pending = pendingImportedTrail;
            if (pending != null) return TrailCreateScreen(trail: pending);
            return const TrailSourceSelectScreen();
          },
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
                  // extra is lost across process restart / deep-link — fall back.
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
          path: '/waypoint/create',
          builder: (context, state) {
            final extra = state.extra;
            if (extra is! Waypoint) return const MapScreen();
            return WaypointCreateScreen(waypoint: extra);
          },
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
