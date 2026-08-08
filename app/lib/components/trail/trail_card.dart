import 'package:cached_network_image/cached_network_image.dart';
import 'dart:io';

import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:intl/intl.dart';
import 'package:wanderer/components/base/actor_avatar.dart';
import 'package:wanderer/components/trail/stat_chip.dart';
import 'package:wanderer/components/trail/sync_status_chip.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/components/trail/trail_category_label.dart';
import 'package:wanderer/util/format.dart';

class TrailCard extends ConsumerWidget {
  final TrailSummary trail;
  final bool fullWidth;
  final bool selected;
  final VoidCallback? onTrailSelect;
  final VoidCallback? onLongPress;

  const TrailCard({
    super.key,
    required this.trail,
    this.fullWidth = false,
    this.selected = false,
    this.onTrailSelect,
    this.onLongPress,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final trailIsShared = trail.summaryShares?.isNotEmpty ?? false;
    final user = ref.watch(authProvider).value;
    if (user == null) return const SizedBox.shrink();
    final unit = ref.watch(unitProvider);

    final String? localPath = trail.localPhotos.isNotEmpty
        ? trail.localPhotos.first
        : null;

    ImageProvider? imageProvider;

    if (trail.isLocal && localPath != null) {
      final file = File(localPath);
      if (file.existsSync()) {
        imageProvider = FileImage(file);
      }
    }

    if (imageProvider == null && trail.summaryThumbnail.isNotEmpty) {
      final networkUrl = trail.getFileUrl(
        user.serverUrl,
        trail.summaryThumbnail,
        thumb: "600x0",
      );
      if (networkUrl != null) {
        imageProvider = CachedNetworkImageProvider(networkUrl);
      }
    }

    final String locale = Localizations.localeOf(context).toString();

    return Container(
      margin: const EdgeInsets.symmetric(vertical: 8),
      width: fullWidth ? double.infinity : 288,
      child: Material(
        color: Theme.of(context).cardColor,
        borderRadius: BorderRadius.circular(20),
        elevation: selected ? 4 : 0,
        child: Ink(
          decoration: BoxDecoration(
            color: Theme.of(context).cardColor,
            borderRadius: BorderRadius.circular(20),
            border: Border.all(
              color: selected
                  ? Theme.of(context).primaryColor
                  : Theme.of(
                      context,
                    ).colorScheme.outlineVariant.withValues(alpha: 0.1),
              width: selected ? 2 : 1,
            ),
            boxShadow: [
              if (selected)
                BoxShadow(
                  color: Theme.of(context).primaryColor.withValues(alpha: 0.1),
                  blurRadius: 8,
                  spreadRadius: 2,
                ),
            ],
          ),
          child: InkWell(
            onTap: onTrailSelect,
            onLongPress: onLongPress,
            borderRadius: BorderRadius.circular(20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Stack(
                  children: [
                    ClipRRect(
                      borderRadius: const BorderRadius.vertical(
                        top: Radius.circular(20),
                      ),
                      child: AspectRatio(
                        aspectRatio: 16 / 9,
                        child: imageProvider != null
                            ? Image(
                                image: imageProvider,
                                fit: BoxFit.cover,
                                errorBuilder: (context, error, stackTrace) =>
                                    _buildPlaceholder(context),
                              )
                            : _buildPlaceholder(context),
                      ),
                    ),

                    if (selected)
                      Positioned(
                        top: 12,
                        left: 12,
                        child: _BadgeWrapper(
                          child: Icon(
                            Icons.check_circle,
                            color: Theme.of(context).primaryColor,
                            size: 24,
                          ),
                        ),
                      ),

                    Positioned(
                      top: 12,
                      right: 12,
                      child: Row(
                        children: [
                          if (trail.public == true)
                            const _BadgeWrapper(
                              child: FaIcon(FontAwesomeIcons.globe, size: 16),
                            ),
                          if (trailIsShared == true) ...[
                            const SizedBox(width: 4),
                            const _BadgeWrapper(
                              child: FaIcon(
                                FontAwesomeIcons.shareNodes,
                                size: 16,
                              ),
                            ),
                          ],
                        ],
                      ),
                    ),
                  ],
                ),

                Padding(
                  padding: const EdgeInsets.all(16.0),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        trail.name,
                        style: const TextStyle(
                          fontWeight: FontWeight.bold,
                          fontSize: 17,
                        ),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                      const SizedBox(height: 4),
                      Align(
                        alignment: Alignment.centerLeft,
                        child: SyncStatusChip(trail: trail),
                      ),
                      const SizedBox(height: 4),
                      if (trail.summaryDate != null)
                        Text(
                          DateFormat.yMMMMd(locale).format(trail.summaryDate!),
                          style: TextStyle(
                            color: Theme.of(
                              context,
                            ).colorScheme.onSurface.withValues(alpha: 0.6),
                            fontSize: 12,
                          ),
                        ),

                      const SizedBox(height: 4),

                      Column(
                        children: [
                          Row(
                            children: [
                              Text(
                                AppLocalizations.of(context)?.by ?? "by",
                                style: TextStyle(
                                  color: Theme.of(context).colorScheme.onSurface
                                      .withValues(alpha: 0.6),
                                  fontSize: 12,
                                ),
                              ),
                              const SizedBox(width: 6),
                              ActorAvatar(
                                actorId: trail.summaryAuthorActorId,
                                imageUrl: trail.summaryAuthorAvatar,
                                nameSeed: trail.summaryAuthorName,
                                radius: 12,
                              ),
                              const SizedBox(width: 6),
                              Text(
                                "${trail.summaryAuthorName}${(trail.domain?.isNotEmpty ?? false) ? "@${trail.domain}" : ""}",
                                style: TextStyle(
                                  color: Theme.of(context).colorScheme.onSurface
                                      .withValues(alpha: 0.6),
                                  fontSize: 12,
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 12),
                        ],
                      ),

                      Wrap(
                        spacing: 16,
                        runSpacing: 8,
                        children: [
                          if (trail.categoryId != null)
                            TrailCategoryLabel(
                              categoryId: trail.categoryId!,
                              subcategoryId: trail.subcategoryId,
                              iconSize: 16,
                              fontSize: 16,
                              color: Theme.of(context).colorScheme.onSurface,
                            ),

                          if (trail.location != null &&
                              trail.location!.isNotEmpty)
                            Text.rich(
                              TextSpan(
                                children: [
                                  WidgetSpan(
                                    child: FaIcon(
                                      FontAwesomeIcons.locationDot,
                                      size: 16,
                                    ),
                                    alignment: PlaceholderAlignment.middle,
                                  ),
                                  const TextSpan(text: "   "),
                                  TextSpan(
                                    text: trail.location,
                                    style: const TextStyle(fontSize: 16),
                                  ),
                                ],
                              ),
                            ),
                          Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              FaIcon(FontAwesomeIcons.gauge, size: 14),
                              const SizedBox(width: 6),
                              Text(
                                _getDifficultyLabel(
                                  context,
                                  trail.summaryDifficulty,
                                ),
                                style: TextStyle(fontSize: 16),
                              ),
                            ],
                          ),
                        ],
                      ),

                      if (trail.summaryTags != null &&
                          trail.summaryTags!.isNotEmpty)
                        Column(
                          children: [
                            const SizedBox(height: 16),
                            Wrap(
                              spacing: 6,
                              children: trail.summaryTags!
                                  .take(3)
                                  .map((tag) => _Chip(text: tag))
                                  .toList(),
                            ),
                            const SizedBox(height: 6),
                          ],
                        ),

                      _StatsGrid(trail: trail, unit: unit),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  String _getDifficultyLabel(BuildContext context, int difficulty) {
    final l10n = AppLocalizations.of(context)!;

    return difficulty == 0
        ? l10n.easy
        : difficulty == 1
        ? l10n.moderate
        : l10n.difficult;
  }

  Widget _buildPlaceholder(BuildContext context) {
    return SvgPicture.asset(
      "assets/svgs/empty_state_trail_${Theme.of(context).brightness.name}.svg",
      semanticsLabel: 'wanderer logo',
      height: 80,
    );
  }
}

class _BadgeWrapper extends StatelessWidget {
  final Widget child;
  const _BadgeWrapper({required this.child});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainer,
        shape: BoxShape.circle,
      ),
      child: child,
    );
  }
}

class _Chip extends StatelessWidget {
  final String text;
  const _Chip({required this.text});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        text,
        style: TextStyle(
          fontSize: 11,
          color: Theme.of(context).colorScheme.onSurface,
        ),
      ),
    );
  }
}

class _StatsGrid extends StatelessWidget {
  final TrailSummary trail;
  final String unit;
  const _StatsGrid({required this.trail, required this.unit});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.only(top: 12),
      child: Wrap(
        spacing: 10,
        runSpacing: 10,
        children: [
          StatChip(
            icon: FontAwesomeIcons.ruler,
            label: formatDistance(trail.distance, unit: unit),
          ),
          StatChip(
            icon: FontAwesomeIcons.clock,
            label: Duration(
              seconds: (trailDisplayDuration(trail) ?? 0).toInt(),
            ).pretty(abbreviated: true, tersity: DurationTersity.minute),
          ),
          StatChip(
            icon: FontAwesomeIcons.arrowTrendUp,
            label: formatElevation(trail.elevationGain, unit: unit),
          ),
          StatChip(
            icon: FontAwesomeIcons.arrowTrendDown,
            label: formatElevation(trail.elevationLoss, unit: unit),
          ),
          // _StatIcon(
          //   icon: FontAwesomeIcons.gaugeHigh,
          //   value: _getDifficultyLabel(context, trail.difficulty),
          // ),
        ],
      ),
    );
  }
}
