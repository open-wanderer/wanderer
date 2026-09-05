import 'dart:math' as math;

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';

const kTopographyGradientDark = LinearGradient(
  begin: Alignment.topCenter,
  end: Alignment.bottomCenter,
  colors: [
    Color(0xFF0B0D16),
    Color(0xFF12141C),
    Color(0xFF1B1E2C),
    Color(0xFF242734),
  ],
  stops: [0.0, 0.30, 0.62, 1.0],
);

const kTopographyGradientLight = LinearGradient(
  begin: Alignment.topCenter,
  end: Alignment.bottomCenter,
  colors: [
    Color(0xFFF8F9FC),
    Color(0xFFEFF1F6),
    Color(0xFFE2E5EE),
    Color(0xFFD5D9E4),
  ],
  stops: [0.0, 0.30, 0.62, 1.0],
);

/// Contour-field constants, matching the hero canvas in `docs/src/pages/index.astro`.
const int _kLevels = 24;
const double _kNoiseScale = 0.0028;
const double _kAmplitude = 0.095;

/// The JS loop advances `t` by 0.003 per animation frame at ~60fps.
const double _kTimePerSecond = 0.003 * 60;

/// Starting offset — "mid-noise for an interesting initial frame".
const double _kTimeOrigin = 2.4;

/// Horizontal sampling resolution, in logical pixels per segment. The web
/// version uses 2.5; 4 keeps the curve smooth at a third of the noise cost.
const double _kStepPx = 4.0;

class TopographyBackground extends StatefulWidget {
  const TopographyBackground({super.key});

  @override
  State<TopographyBackground> createState() => _TopographyBackgroundState();
}

class _TopographyBackgroundState extends State<TopographyBackground>
    with SingleTickerProviderStateMixin {
  /// Free-running clock. Long enough that it never wraps in a session, so the
  /// noise field never snaps back on repeat.
  late final AnimationController _controller = AnimationController(
    vsync: this,
    duration: const Duration(days: 1),
  );

  /// Smoothed pointer position, eased toward [_targetPointerX] each frame.
  final ValueNotifier<double> _pointerX = ValueNotifier<double>(0.5);
  double _targetPointerX = 0.5;

  /// The painter's clock, republished from [_controller] at ~30 Hz instead
  /// of every display frame. Each repaint runs the full contour-field noise
  /// pass (24 levels × ~w/4 steps × 12 hashed sines — tens of thousands of
  /// `sin()` calls), and driving a subtle ambient background at 60–120 Hz
  /// doubled-to-quadrupled that for no perceptible smoothness gain.
  final ValueNotifier<double> _time = ValueNotifier<double>(0);
  Duration _lastTimePush = Duration.zero;

  bool _reducedMotion = false;

  @override
  void initState() {
    super.initState();
    _controller.addListener(_onTick);
  }

  void _onTick() {
    _easePointer();
    final elapsed = _controller.lastElapsedDuration ?? Duration.zero;
    if ((elapsed - _lastTimePush).inMilliseconds >= 33) {
      _lastTimePush = elapsed;
      _time.value = _controller.value;
    }
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final reducedMotion = MediaQuery.disableAnimationsOf(context);
    if (reducedMotion == _reducedMotion && _controller.isAnimating) return;
    _reducedMotion = reducedMotion;
    if (reducedMotion) {
      _controller
        ..stop()
        ..value = 0;
    } else {
      _controller.repeat();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    _pointerX.dispose();
    _time.dispose();
    super.dispose();
  }

  void _easePointer() {
    final next = _pointerX.value + (_targetPointerX - _pointerX.value) * 0.04;
    if ((next - _pointerX.value).abs() > 0.0001) _pointerX.value = next;
  }

  void _onHover(PointerHoverEvent event) {
    final width = context.size?.width ?? 0;
    if (width <= 0) return;
    _targetPointerX = (event.localPosition.dx / width).clamp(0.0, 1.0);
  }

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return MouseRegion(
      opaque: false,
      onHover: _onHover,
      child: DecoratedBox(
        decoration: BoxDecoration(
          gradient: isDark ? kTopographyGradientDark : kTopographyGradientLight,
        ),
        child: RepaintBoundary(
          child: CustomPaint(
            painter: _TopographyPainter(
              time: _time,
              durationMs: _controller.duration!.inMilliseconds,
              pointerX: _pointerX,
              lineColor: isDark ? Colors.white : Colors.black,
              // Dark strokes on a light field need more weight to read.
              opacityScale: isDark ? 1.0 : 1.6,
            ),
            size: Size.infinite,
          ),
        ),
      ),
    );
  }
}

class _TopographyPainter extends CustomPainter {
  _TopographyPainter({
    required this.time,
    required this.durationMs,
    required this.pointerX,
    required this.lineColor,
    required this.opacityScale,
  }) : super(repaint: Listenable.merge([time, pointerX]));

  /// Throttled clock (see `_TopographyBackgroundState._time`) — the painter
  /// deliberately does NOT listen to the raw controller.
  final ValueNotifier<double> time;
  final int durationMs;
  final ValueNotifier<double> pointerX;
  final Color lineColor;
  final double opacityScale;

  static double _hash(double ix, double iy) {
    final n = math.sin(ix * 127.1 + iy * 311.7) * 43758.5453;
    return n - n.floorToDouble();
  }

  static double _smoothstep(double x) => x * x * (3 - 2 * x);

  static double _valueNoise(double x, double y) {
    final xi = x.floorToDouble(), yi = y.floorToDouble();
    final u = _smoothstep(x - xi), v = _smoothstep(y - yi);
    final a = _hash(xi, yi), b = _hash(xi + 1, yi);
    final c = _hash(xi, yi + 1), d = _hash(xi + 1, yi + 1);
    return a + (b - a) * u + (c - a) * v + (a - b - c + d) * u * v;
  }

  /// Fractional Brownian motion — 3 octaves of value noise, normalised to [-1, 1].
  static double _fbm(double x, double y) {
    final v =
        _valueNoise(x, y) * 0.5 +
        _valueNoise(x * 2.1 + 1.7, y * 2.1 + 9.2) * 0.25 +
        _valueNoise(x * 4.3 + 4.3, y * 4.3 + 2.8) * 0.125;
    return (v / 0.875) * 2 - 1;
  }

  @override
  void paint(Canvas canvas, Size size) {
    final w = size.width, h = size.height;
    if (w <= 0 || h <= 0) return;

    final t = _kTimeOrigin + time.value * durationMs / 1000 * _kTimePerSecond;
    final pointer = pointerX.value;
    final steps = math.max(2, (w / _kStepPx).ceil());

    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.round
      ..isAntiAlias = true;

    for (var i = 0; i < _kLevels; i++) {
      final progress = i / (_kLevels - 1);
      final baseY = h * (0.05 + 0.92 * progress);
      final parallax = (pointer - 0.5) * w * 0.03 * (0.2 + progress * 0.8);
      final opacity = (0.018 + progress * 0.062) * opacityScale;

      paint
        ..color = lineColor.withValues(alpha: opacity.clamp(0.0, 1.0))
        ..strokeWidth = 0.3 + progress * 0.8;

      final path = Path();
      for (var j = 0; j <= steps; j++) {
        final x = (j / steps) * w;
        final nx = (x + parallax) * _kNoiseScale + t * 0.07;
        final ny = baseY * _kNoiseScale * 0.55 + i * 0.18;
        final y = baseY + _fbm(nx, ny) * h * _kAmplitude;
        if (j == 0) {
          path.moveTo(x, y);
        } else {
          path.lineTo(x, y);
        }
      }
      canvas.drawPath(path, paint);
    }
  }

  @override
  bool shouldRepaint(_TopographyPainter oldDelegate) =>
      oldDelegate.lineColor != lineColor ||
      oldDelegate.opacityScale != opacityScale;
}
