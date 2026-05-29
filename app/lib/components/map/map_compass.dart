import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'dart:math';

import 'package:font_awesome_flutter/font_awesome_flutter.dart';

class MapCompass extends StatefulWidget {
  /// Use this constructor if you want to customize your compass.
  ///
  /// Use [MapCompass.cupertino] to use the default.
  const MapCompass({
    super.key,
    this.rotationOffset = 0,
    this.rotationDuration = const Duration(seconds: 1),
    this.animationCurve = Curves.fastOutSlowIn,
    this.onPressed,
    this.onPressedOverridesDefault = true,
    this.hideIfRotatedNorth = false,
    this.padding = const EdgeInsets.all(10),
  });

  /// Sometimes icons are rotated itself. Use this to fix the rotation offset.
  final double rotationOffset;

  /// Overrides the default behaviour for a tap or click event
  ///
  /// This will override the default behaviour.
  final VoidCallback? onPressed;

  /// Set to true to hide the compass while the map is not rotated.
  ///
  /// Defaults to false (always visible).
  final bool hideIfRotatedNorth;

  /// The padding of the compass widget.
  ///
  /// Defaults to 10px on all sides.
  final EdgeInsets padding;

  /// The duration of the rotation animation.
  ///
  /// Default to 1 second.
  final Duration rotationDuration;

  /// The curve of the rotation animation.
  final Curve animationCurve;

  /// When [onPressedOverridesDefault] is true, [onPressed] overrides
  /// the default rotation.
  final bool onPressedOverridesDefault;

  @override
  State<MapCompass> createState() => _MapCompassState();
}

class _MapCompassState extends State<MapCompass> with TickerProviderStateMixin {
  AnimationController? _animationController;
  late Animation<double> _rotateAnimation;

  late Tween<double> _rotationTween;

  @override
  Widget build(BuildContext context) {
    final camera = MapCamera.of(context);

    if (widget.hideIfRotatedNorth && camera.rotation % 360 == 0) {
      return const SizedBox(width: 68);
    }

    return Padding(
      padding: widget.padding,
      child: Transform.rotate(
        angle: (camera.rotation + widget.rotationOffset) * (pi / 180.0),
        child: IconButton(
          alignment: Alignment.center,
          padding: EdgeInsets.zero,
          icon: Column(
            children: [
              FaIcon(FontAwesomeIcons.caretUp, size: 12, color: Colors.orange),
              Text(
                "N",
                style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold),
              ),
              FaIcon(FontAwesomeIcons.caretDown, size: 12),
            ],
          ),
          onPressed: () {
            if (widget.onPressedOverridesDefault) {
              if (widget.onPressed case final VoidCallback onPressed) {
                onPressed();
              } else {
                _resetRotation(context, camera);
              }
            } else {
              _resetRotation(context, camera);
              widget.onPressed?.call();
            }
          },
          style: IconButton.styleFrom(
            backgroundColor: Theme.of(context).canvasColor,
          ),
        ),
      ),
    );
  }

  void _resetRotation(BuildContext context, MapCamera camera) {
    // current rotation of the map
    final rotation = camera.rotation;
    // nearest north (0°, 360°, -360°, ...)
    final endRotation = (rotation / 360).round() * 360.0;
    // don't start animation if rotation doesn't need to change
    if (rotation == endRotation) return;

    _animationController = AnimationController(
      duration: widget.rotationDuration,
      vsync: this,
    )..addListener(_handleAnimation);
    _rotateAnimation = CurvedAnimation(
      parent: _animationController!,
      curve: widget.animationCurve,
    );

    _rotationTween = Tween<double>(begin: rotation, end: endRotation);
    _animationController!.forward(from: 0);
  }

  void _handleAnimation() {
    final controller = MapController.of(context);
    controller.rotate(_rotationTween.evaluate(_rotateAnimation));
  }

  @override
  void dispose() {
    _animationController?.dispose();
    super.dispose();
  }
}
