import 'package:cached_network_image/cached_network_image.dart';
import 'dart:math' as math;

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/search/global_search_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/util/format.dart';
import 'package:wanderer/util/geo/polyline.dart';

class GlobalSearchScreen extends ConsumerStatefulWidget {
  const GlobalSearchScreen({super.key});

  @override
  ConsumerState<GlobalSearchScreen> createState() => _GlobalSearchScreenState();
}

class _GlobalSearchScreenState extends ConsumerState<GlobalSearchScreen> {
  late final TextEditingController _controller;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController();
    _controller.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final state = ref.watch(globalSearchProvider);
    final notifier = ref.read(globalSearchProvider.notifier);
    final unit = ref.watch(unitProvider);

    return Scaffold(
      appBar: AppBar(
        title: Builder(
          builder: (context) {
            final colorScheme = Theme.of(context).colorScheme;
            return TextField(
              controller: _controller,
              autofocus: true,
              onChanged: notifier.setQuery,
              cursorColor: colorScheme.onSurface,
              decoration: InputDecoration(
                hintText: l10n.search_for_trails_places,
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(56),
                  borderSide: BorderSide(color: colorScheme.outline, width: 1),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(56),
                  borderSide: BorderSide(
                    color: colorScheme.onSurface,
                    width: 1,
                  ),
                ),
                hintStyle: TextStyle(color: Colors.grey),
              ),
              style: const TextStyle(fontSize: 16),
            );
          },
        ),
        actions: [
          if (_controller.text.isNotEmpty)
            IconButton(
              icon: const Icon(Icons.close),
              onPressed: () {
                _controller.clear();
                notifier.setQuery('');
              },
            ),
        ],
      ),
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _CategoryChips(state: state, notifier: notifier, l10n: l10n),
          const Divider(height: 1),
          Expanded(
            child: _ResultsList(state: state, l10n: l10n, unit: unit),
          ),
        ],
      ),
    );
  }
}

class _CategoryChips extends StatelessWidget {
  final GlobalSearchState state;
  final GlobalSearchNotifier notifier;
  final AppLocalizations l10n;

  const _CategoryChips({
    required this.state,
    required this.notifier,
    required this.l10n,
  });

  String _label(GlobalSearchCategory cat) => switch (cat) {
    GlobalSearchCategory.all => l10n.all,
    GlobalSearchCategory.trails => l10n.trail(2),
    GlobalSearchCategory.lists => l10n.list(2),
    GlobalSearchCategory.locations => l10n.locations,
    GlobalSearchCategory.actors => l10n.users,
  };

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: GlobalSearchCategory.values.map((cat) {
          final selected = state.category == cat;
          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: ChoiceChip(
              label: Text(_label(cat)),
              selected: selected,
              onSelected: (_) => notifier.setCategory(cat),
              backgroundColor: colorScheme.surfaceContainerHighest,
              selectedColor: colorScheme.primaryContainer,
              checkmarkColor: colorScheme.onPrimaryContainer,
              shape: StadiumBorder(
                side: BorderSide(
                  color: selected
                      ? colorScheme.outlineVariant
                      : colorScheme.outline,
                ),
              ),
              labelStyle: TextStyle(
                color: selected
                    ? colorScheme.onPrimaryContainer
                    : colorScheme.onSurface,
                fontWeight: selected ? FontWeight.w600 : FontWeight.normal,
              ),
            ),
          );
        }).toList(),
      ),
    );
  }
}

class _ResultsList extends StatelessWidget {
  final GlobalSearchState state;
  final AppLocalizations l10n;
  final String unit;

  const _ResultsList({
    required this.state,
    required this.l10n,
    required this.unit,
  });

  @override
  Widget build(BuildContext context) {
    if (state.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.query.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.search, size: 64, color: Colors.grey.shade300),
            const SizedBox(height: 12),
            Text(
              l10n.search_for_trails_places,
              style: TextStyle(color: Colors.grey.shade500),
            ),
          ],
        ),
      );
    }

    if (!state.hasResults) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            SvgPicture.asset(
              "assets/svgs/empty_state_search_${Theme.of(context).brightness.name}.svg",
              semanticsLabel: 'wanderer comment empty state',
              height: 120,
            ),
            const SizedBox(height: 12),
            Text(
              l10n.no_results_for_query(state.query),
              style: TextStyle(color: Colors.grey.shade500),
            ),
          ],
        ),
      );
    }

    return ListView(
      children: [
        ...state.visibleTrails.map((t) => _TrailTile(trail: t, unit: unit)),
        ...state.visibleLists.map((l) => _ListTile(list: l)),
        ...state.visibleLocations.map((l) => _LocationTile(location: l)),
        ...state.visibleActors.map((a) => _ActorTile(actor: a)),
      ],
    );
  }
}

class _TrailTile extends ConsumerWidget {
  final TrailSearchResult trail;
  final String unit;

  const _TrailTile({required this.trail, required this.unit});

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
      onTap: () => context.push('/trail/${trail.id}'),
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
              backgroundImage: CachedNetworkImageProvider(
                trail.authorAvatar.isNotEmpty
                    ? trail.authorAvatar
                    : 'https://api.dicebear.com/7.x/initials/png?seed=${trail.authorName}',
              ),
            )
          : null,
    );
  }
}

class _ListTile extends StatelessWidget {
  final ListSearchResult list;

  const _ListTile({required this.list});

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
      onTap: () => context.push('/list/${list.id}'),
      title: Text(list.name, maxLines: 1, overflow: TextOverflow.ellipsis),
      subtitle: Text(
        '${list.trails} ${l10n.trail(list.trails)}${list.authorName.isNotEmpty ? ' · ${list.authorName}' : ''}',
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
    );
  }
}

class _LocationTile extends StatelessWidget {
  final LocationSearchResult location;

  const _LocationTile({required this.location});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      onTap: () => context.go(
        '/map',
        extra: {'lat': location.lat, 'lon': location.lon, 'zoom': 13.0},
      ),
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

class _ActorTile extends StatelessWidget {
  final ActorSearchResult actor;

  const _ActorTile({required this.actor});

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
            ? CachedNetworkImageProvider(actor.icon!)
            : CachedNetworkImageProvider(
                'https://api.dicebear.com/7.x/initials/png?seed=$displayName',
              ),
      ),
      title: Text(displayName, maxLines: 1, overflow: TextOverflow.ellipsis),
      subtitle: Text(subtitle),
      onTap: () {
        final route = actor.isLocal
            ? '/profile/@${actor.preferredUsername}'
            : '/profile/@${actor.preferredUsername}@${actor.domain}';
        context.push(route);
      },
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
