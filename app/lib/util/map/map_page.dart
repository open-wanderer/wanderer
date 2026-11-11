import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// [GoRoute.pageBuilder] for full-screen map routes: wraps [builder]'s screen
/// in a [MapPage], keyed and named the way go_router's default page is.
GoRouterPageBuilder mapPageBuilder(GoRouterWidgetBuilder builder) =>
    (context, state) => MapPage<void>(
      key: state.pageKey,
      name: state.name ?? state.path,
      arguments: <String, String>{
        ...state.pathParameters,
        ...state.uri.queryParameters,
      },
      restorationId: state.pageKey.value,
      child: builder(context, state),
    );

/// A [MaterialPage] that skips Android's predictive-back transition.
///
/// The default Android builder scales the outgoing screen live while the
/// back gesture is held, re-rasterizing it every frame — on a platform-view
/// map that visibly jitters. Here the gesture only pops on release, and the
/// pop plays the same [FadeForwardsPageTransitionsBuilder] slide predictive
/// back falls back to for button presses. Other platforms are unchanged.
class MapPage<T> extends MaterialPage<T> {
  const MapPage({
    required super.child,
    super.key,
    super.name,
    super.arguments,
    super.restorationId,
  });

  @override
  Route<T> createRoute(BuildContext context) => _MapPageRoute<T>(page: this);
}

class _MapPageRoute<T> extends PageRoute<T>
    with MaterialRouteTransitionMixin<T> {
  _MapPageRoute({required MapPage<T> page}) : super(settings: page);

  MapPage<T> get _page => settings as MapPage<T>;

  @override
  Widget buildContent(BuildContext context) => _page.child;

  @override
  bool get maintainState => _page.maintainState;

  @override
  bool get fullscreenDialog => _page.fullscreenDialog;

  @override
  String get debugLabel => '${super.debugLabel}(${_page.name})';

  @override
  Widget buildTransitions(
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) {
    if (Theme.of(context).platform != TargetPlatform.android) {
      return super.buildTransitions(
        context,
        animation,
        secondaryAnimation,
        child,
      );
    }
    return const FadeForwardsPageTransitionsBuilder().buildTransitions(
      this,
      context,
      animation,
      secondaryAnimation,
      child,
    );
  }
}
