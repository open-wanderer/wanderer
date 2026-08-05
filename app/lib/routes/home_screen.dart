import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/router_provider.dart';

/// Rendered size of the logo, and the source SVG's viewBox. Both layers are
/// pinned to this box rather than left to size themselves, so the painter's
/// viewBox→pixel scale is a fact rather than an assumption about how
/// `SvgPicture` resolves a height-only constraint inside a `Stack`.
const double _logoHeight = 256;
const double _viewBoxWidth = 143;
const double _viewBoxHeight = 175;
const double _logoWidth = _logoHeight * _viewBoxWidth / _viewBoxHeight;

/// The two colours the mark is drawn in, which the reveal has to match exactly
/// or the trail shows a seam against the disc. Taken from the source SVG's
/// literal fills, not the theme — `splash_logo_*.svg` hard-codes them.
const Color _ink = Color(0xFF242734);
const Color _paper = Colors.white;

/// The disc, in viewBox units: the logo's circle is centred (72, 64) with
/// radius 64.
const Offset _discCentre = Offset(72, 64);
const double _discRadius = 64;

/// Where the reveal grows from — the midpoint of the trail's two feet on the
/// rim — and the radius at which it has covered the whole ribbon.
const Offset _revealOrigin = Offset(59.7516, 123.824);
const double _revealRadius = 56;

/// How long the ascent takes when nothing interrupts it.
const Duration _revealDuration = Duration(milliseconds: 900);

/// Upper bound on the catch-up once auth has already settled. Scaled by the
/// fraction still left to draw, so a nearly-finished reveal snaps shut and a
/// barely-started one still reads as motion rather than a jump cut.
const Duration _revealCatchUp = Duration(milliseconds: 300);

/// Wall-clock release for the router hold, independent of the animation.
///
/// Sits just past `auth_provider.dart`'s own 3s validation timeout. `/` is the
/// initial route and has no navigation affordance, so a hold that never lifts
/// strands the user with no way out but force-quit. Treat the animation as
/// something that can only *shorten* the hold — never as the sole thing that
/// ends it.
const Duration _revealDeadline = Duration(milliseconds: 3500);

class HomeScreen extends ConsumerStatefulWidget {
  const HomeScreen({super.key});

  @override
  ConsumerState<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends ConsumerState<HomeScreen>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  Timer? _failsafe;
  bool _started = false;
  bool _caughtUp = false;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(vsync: this, duration: _revealDuration)
      ..addStatusListener((AnimationStatus status) {
        if (status == AnimationStatus.completed) _releaseHold();
      });
    _failsafe = Timer(_revealDeadline, _releaseHold);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_started) return;
    _started = true;

    // Reduced motion: show the finished trail and get out of the way. Holding
    // the route for an animation that will not play would be a pure delay.
    if (MediaQuery.disableAnimationsOf(context)) {
      _controller.value = 1;
      _releaseHold();
      return;
    }

    _controller.forward();
  }

  @override
  void dispose() {
    _failsafe?.cancel();
    _controller.dispose();
    super.dispose();
  }

  /// Lets the router leave `/`. Safe to call repeatedly and from any of the
  /// three paths that can trigger it (completion, reduced motion, failsafe).
  void _releaseHold() {
    _failsafe?.cancel();
    _failsafe = null;
    if (!mounted) return;
    ref.read(splashRevealProvider.notifier).complete();
  }

  /// Auth has settled with the reveal still in flight — run it to the summit at
  /// speed rather than letting the screen sit there fully loaded.
  void _catchUp() {
    if (_caughtUp || !_controller.isAnimating) return;
    _caughtUp = true;

    final int remaining =
        (_revealCatchUp.inMilliseconds * (1 - _controller.value)).round().clamp(
          80,
          _revealCatchUp.inMilliseconds,
        );
    _controller.animateTo(
      1,
      duration: Duration(milliseconds: remaining),
      curve: Curves.easeOut,
    );
  }

  @override
  Widget build(BuildContext context) {
    ref.listen<AsyncValue<dynamic>>(authProvider, (
      AsyncValue<dynamic>? previous,
      AsyncValue<dynamic> next,
    ) {
      if (!next.isLoading) _catchUp();
    });

    // Auth may already have settled before the first frame.
    if (!ref.read(authProvider).isLoading) {
      WidgetsBinding.instance.addPostFrameCallback((_) => _catchUp());
    }

    final bool isDark = Theme.of(context).brightness == Brightness.dark;

    return Scaffold(
      body: Center(
        child: Semantics(
          label: 'wanderer',
          child: ExcludeSemantics(
            child: SizedBox(
              width: _logoWidth,
              height: _logoHeight,
              child: Stack(
                fit: StackFit.expand,
                children: <Widget>[
                  // Painted behind the mark, showing through the trail-shaped
                  // hole in it. Order matters: this must come first.
                  AnimatedBuilder(
                    animation: _controller,
                    builder: (BuildContext context, Widget? child) {
                      return CustomPaint(
                        painter: _TrailRevealPainter(
                          progress: _controller.value,
                          disc: isDark ? _paper : _ink,
                          trail: isDark ? _ink : _paper,
                        ),
                      );
                    },
                  ),
                  SvgPicture.asset(
                    'assets/svgs/splash_logo_${isDark ? 'dark' : 'light'}.svg',
                    fit: BoxFit.fill,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Paints the trail reveal *behind* the mark.
///
/// The shipped logo draws its disc with the trail already cut out of it as a
/// hole, and paints the ribbon back in on top. `splash_logo_*.svg` drops that
/// top layer, so the hole is open and whatever is painted here shows through it
/// — already in the shape of the trail.
///
/// That is what makes this cheap: the reveal never has to know the trail's
/// geometry. A plain circle growing from the trail's feet on the rim is clipped
/// into a trail by the hole itself, and because the ribbon's distance from
/// those feet increases monotonically along its length, the circle uncovers it
/// end to end — up past the switchback, summit last — with no detached islands
/// on the way.
class _TrailRevealPainter extends CustomPainter {
  const _TrailRevealPainter({
    required this.progress,
    required this.disc,
    required this.trail,
  });

  final double progress;
  final Color disc;
  final Color trail;

  @override
  void paint(Canvas canvas, Size size) {
    final double scale = size.width / _viewBoxWidth;
    final Rect discRect = Rect.fromCircle(
      center: _discCentre * scale,
      radius: _discRadius * scale,
    );

    // The hole reaches the rim, so an unclipped reveal would spill out past the
    // mark's silhouette onto the scaffold.
    canvas.clipPath(Path()..addOval(discRect));

    // Backing the hole in the disc's own colour is what makes an unrevealed
    // trail invisible rather than a window onto the scaffold.
    canvas.drawRect(discRect, Paint()..color = disc);
    canvas.drawCircle(
      _revealOrigin * scale,
      _revealRadius * progress * scale,
      Paint()..color = trail,
    );
  }

  @override
  bool shouldRepaint(_TrailRevealPainter oldDelegate) =>
      oldDelegate.progress != progress ||
      oldDelegate.disc != disc ||
      oldDelegate.trail != trail;
}
