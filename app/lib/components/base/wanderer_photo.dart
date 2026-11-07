import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:vector_graphics/vector_graphics_compat.dart';
import 'package:wanderer/util/image_cache.dart';
import 'package:wanderer/util/route_placeholder.dart';

/// A trail or summit log photo, from the server or from disk.
///
/// Use this instead of [Image] wherever the photo comes from a `photos` field:
/// the first one may be a generated route image, which is an SVG.
class WandererPhoto extends StatelessWidget {
  final String? url;
  final String? path;
  final BoxFit fit;

  /// Shown when the photo can't be loaded or decoded.
  final WidgetBuilder? errorBuilder;

  /// Passed to [Image]. A route image is small enough to go without.
  final ImageLoadingBuilder? loadingBuilder;

  const WandererPhoto.network(
    String this.url, {
    super.key,
    this.fit = BoxFit.cover,
    this.errorBuilder,
    this.loadingBuilder,
  }) : path = null;

  const WandererPhoto.file(
    String this.path, {
    super.key,
    this.fit = BoxFit.cover,
    this.errorBuilder,
  }) : url = null,
       loadingBuilder = null;

  @override
  Widget build(BuildContext context) {
    final errorBuilder = this.errorBuilder;

    if (isSvgPhoto(url ?? path!)) {
      return _RouteImage(
        url: url,
        path: path,
        fit: fit,
        errorBuilder: errorBuilder,
      );
    }

    return Image(
      image: url != null ? wandererImageProvider(url!) : FileImage(File(path!)),
      fit: fit,
      loadingBuilder: loadingBuilder,
      errorBuilder: errorBuilder == null
          ? null
          : (context, _, _) => errorBuilder(context),
    );
  }
}

/// Draws a generated route image, which [Image] can't decode, in the style of
/// the current theme.
class _RouteImage extends StatefulWidget {
  final String? url;
  final String? path;
  final BoxFit fit;
  final WidgetBuilder? errorBuilder;

  const _RouteImage({
    required this.url,
    required this.path,
    required this.fit,
    required this.errorBuilder,
  });

  @override
  State<_RouteImage> createState() => _RouteImageState();
}

class _RouteImageState extends State<_RouteImage> {
  late Future<String> _svg = _read();

  /// Reads through the image cache, so a private instance gets the session
  /// cookie and a thumbnail seen once still shows offline.
  Future<String> _read() async {
    final url = widget.url;
    final file = url != null
        ? await WandererImageCacheManager().getSingleFile(url)
        : File(widget.path!);
    return utf8.decode(await file.readAsBytes(), allowMalformed: true);
  }

  @override
  void didUpdateWidget(_RouteImage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.url != oldWidget.url || widget.path != oldWidget.path) {
      _svg = _read();
    }
  }

  @override
  Widget build(BuildContext context) {
    final errorBuilder = widget.errorBuilder;

    return FutureBuilder(
      future: _svg,
      builder: (context, snapshot) {
        if (snapshot.hasError) {
          return errorBuilder?.call(context) ?? const SizedBox.shrink();
        }
        final svg = snapshot.data;
        if (svg == null) return const SizedBox.shrink();
        return _buildSvg(context, svg);
      },
    );
  }

  Widget _buildSvg(BuildContext context, String svg) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final errorBuilder = widget.errorBuilder;

    return LayoutBuilder(
      builder: (context, constraints) => SvgPicture(
        RoutePlaceholderLoader(
          svg,
          dark: dark,
          box: constraints.biggest,
          fit: widget.fit,
        ),
        width: constraints.hasBoundedWidth ? constraints.maxWidth : null,
        height: constraints.hasBoundedHeight ? constraints.maxHeight : null,
        fit: widget.fit,
        excludeFromSemantics: true,
        // Lists of these scroll: a raster is drawn once, where a picture
        // would tessellate every contour line on every frame.
        renderingStrategy: RenderingStrategy.raster,
        errorBuilder: errorBuilder == null
            ? null
            : (context, _, _) => errorBuilder(context),
      ),
    );
  }
}
