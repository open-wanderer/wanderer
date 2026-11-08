import 'dart:math' as math;

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/util/format.dart';
import 'package:wanderer/util/geo/polyline.dart';
import 'package:wanderer/util/image_cache.dart';

// One tile per search result type. Shared by the search screens' live
// results and their search history, so a remembered result looks exactly
// like it did when it was tapped. What a tap does is up to the screen.

class SearchTrailTile extends ConsumerWidget {
  final TrailSearchResult trail;
  final String unit;
  final VoidCallback onTap;

  const SearchTrailTile({
    super.key,
    required this.trail,
    required this.unit,
    required this.onTap,
  });

  String _difficultyLabel(BuildContext context, int difficulty) {
    final l10n = AppLocalizations.of(context)!;
    return difficulty == 0
        ? l10n.easy
        : difficulty == 1
        ? l10n.moderate
        : l10n.difficult;
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final locale = Localizations.localeOf(context);
    final categories = ref.watch(categoryProvider).value ?? [];
    final subcategories = ref.watch(subcategoryProvider);

    final Category? category = trail.categoryId != null
        ? categories.firstWhereOrNull((c) => c.id == trail.categoryId)
        : null;
    final subcategory = trail.subcategoryId != null
        ? subcategories.firstWhereOrNull((s) => s.id == trail.subcategoryId)
        : null;
    final categoryLabel = (category != null && subcategory != null)
        ? '${category.displayName(locale)} / ${subcategory.displayName(locale)}'
        : category?.displayName(locale);

    return ListTile(
      onTap: onTap,
      leading: _PolylinePreview(
        polyline: trail.polyline,
        color: Theme.of(context).colorScheme.primary,
        backgroundColor: Theme.of(context).colorScheme.primaryContainer,
      ),
      title: Text(trail.name, maxLines: 1, overflow: TextOverflow.ellipsis),
      subtitle: Text(
        [
          ?categoryLabel,
          _difficultyLabel(context, trail.difficulty),
          formatDistance(trail.distance, unit: unit),
        ].join(' · '),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      trailing: trail.authorName.isNotEmpty
          ? CircleAvatar(
              radius: 14,
              backgroundColor: Colors.grey.shade300,
              backgroundImage: wandererImageProvider(
                trail.authorAvatar.isNotEmpty
                    ? trail.authorAvatar
                    : 'https://api.dicebear.com/7.x/initials/png?seed=${trail.authorName}',
              ),
            )
          : null,
    );
  }
}

class SearchListTile extends StatelessWidget {
  final ListSearchResult list;
  final VoidCallback onTap;

  const SearchListTile({super.key, required this.list, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return ListTile(
      leading: Container(
        width: 40,
        height: 40,
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.secondaryContainer,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Center(child: FaIcon(FontAwesomeIcons.layerGroup, size: 18)),
      ),
      onTap: onTap,
      title: Text(list.name, maxLines: 1, overflow: TextOverflow.ellipsis),
      subtitle: Text(
        '${list.trails} ${l10n.trail(list.trails)}${list.authorName.isNotEmpty ? ' · ${list.authorName}' : ''}',
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
    );
  }
}

class SearchLocationTile extends StatelessWidget {
  final LocationSearchResult location;
  final VoidCallback onTap;

  const SearchLocationTile({
    super.key,
    required this.location,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      onTap: onTap,
      leading: Container(
        width: 40,
        height: 40,
        decoration: BoxDecoration(
          color: Colors.orange.withValues(alpha: 0.15),
          borderRadius: BorderRadius.circular(8),
        ),
        child: const FaIcon(
          FontAwesomeIcons.locationDot,
          size: 18,
          color: Colors.orange,
        ),
      ),
      title: Text(location.name, maxLines: 1, overflow: TextOverflow.ellipsis),
      subtitle: location.description.isNotEmpty
          ? Text(
              location.description,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            )
          : null,
    );
  }
}

class SearchActorTile extends StatelessWidget {
  final ActorSearchResult actor;
  final VoidCallback onTap;

  const SearchActorTile({super.key, required this.actor, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final displayName = actor.username.isNotEmpty
        ? actor.username
        : actor.preferredUsername;
    final subtitle =
        '@${actor.preferredUsername}${actor.isLocal ? "" : "@${actor.domain}"}';

    return ListTile(
      leading: CircleAvatar(
        radius: 20,
        backgroundColor: Colors.grey.shade300,
        backgroundImage: actor.icon != null && actor.icon!.isNotEmpty
            ? wandererImageProvider(actor.icon!)
            : wandererImageProvider(
                'https://api.dicebear.com/7.x/initials/png?seed=$displayName',
              ),
      ),
      title: Text(displayName, maxLines: 1, overflow: TextOverflow.ellipsis),
      subtitle: Text(subtitle),
      onTap: onTap,
    );
  }
}

class _PolylinePreview extends StatelessWidget {
  final String? polyline;
  final Color color;
  final Color backgroundColor;

  const _PolylinePreview({
    required this.polyline,
    required this.color,
    required this.backgroundColor,
  });

  @override
  Widget build(BuildContext context) {
    List<Geographic>? points;
    if (polyline != null && polyline!.isNotEmpty) {
      try {
        points = PolylineUtil.decode(polyline!);
      } catch (_) {}
    }

    return Container(
      width: 40,
      height: 40,
      decoration: BoxDecoration(
        color: backgroundColor,
        borderRadius: BorderRadius.circular(8),
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(8),
        child: points != null && points.length >= 2
            ? CustomPaint(
                painter: _PolylinePainter(points: points, color: color),
              )
            : FaIcon(FontAwesomeIcons.route, size: 20, color: color),
      ),
    );
  }
}

class _PolylinePainter extends CustomPainter {
  final List<Geographic> points;
  final Color color;

  const _PolylinePainter({required this.points, required this.color});

  @override
  void paint(Canvas canvas, Size size) {
    double minLat = double.infinity, maxLat = -double.infinity;
    double minLng = double.infinity, maxLng = -double.infinity;

    for (final p in points) {
      if (p.lat < minLat) minLat = p.lat;
      if (p.lat > maxLat) maxLat = p.lat;
      if (p.lon < minLng) minLng = p.lon;
      if (p.lon > maxLng) maxLng = p.lon;
    }

    final latRange = maxLat - minLat;
    final lngRange = maxLng - minLng;
    if (latRange == 0 && lngRange == 0) return;

    const padding = 5.0;
    final w = size.width - padding * 2;
    final h = size.height - padding * 2;

    final scale = latRange == 0
        ? w / lngRange
        : lngRange == 0
        ? h / latRange
        : math.min(w / lngRange, h / latRange);

    final dx = padding + (w - lngRange * scale) / 2;
    final dy = padding + (h - latRange * scale) / 2;

    final path = Path();
    for (int i = 0; i < points.length; i++) {
      final x = dx + (points[i].lon - minLng) * scale;
      final y = dy + (maxLat - points[i].lat) * scale;
      i == 0 ? path.moveTo(x, y) : path.lineTo(x, y);
    }

    canvas.drawPath(
      path,
      Paint()
        ..color = Colors.white.withValues(alpha: 0.7)
        ..strokeWidth = 2.5
        ..style = PaintingStyle.stroke
        ..strokeCap = StrokeCap.round
        ..strokeJoin = StrokeJoin.round,
    );
    canvas.drawPath(
      path,
      Paint()
        ..color = color
        ..strokeWidth = 1.5
        ..style = PaintingStyle.stroke
        ..strokeCap = StrokeCap.round
        ..strokeJoin = StrokeJoin.round,
    );
  }

  @override
  bool shouldRepaint(_PolylinePainter old) =>
      old.points != points || old.color != color;
}
