import 'dart:io';

import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:intl/intl.dart';
import 'package:wanderer/components/trail/stat_chip.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/util/icon_util.dart';

class TrailListItem extends ConsumerWidget {
  final TrailSummary trail;
  final VoidCallback? onTrailSelect;

  const TrailListItem({super.key, required this.trail, this.onTrailSelect});

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

    if (trail.isOffline && localPath != null) {
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
        imageProvider = NetworkImage(networkUrl);
      }
    }

    final String locale = Localizations.localeOf(context).toString();

    return Container(
      margin: const EdgeInsets.symmetric(vertical: 5),
      child: Material(
        color: Theme.of(context).cardColor,
        borderRadius: BorderRadius.circular(16),
        child: Ink(
          decoration: BoxDecoration(
            color: Theme.of(context).cardColor,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: Colors.grey.withValues(alpha: 0.2),
              width: 1,
            ),
          ),
          child: InkWell(
            onTap: onTrailSelect,
            borderRadius: BorderRadius.circular(16),
            child: Padding(
              padding: const EdgeInsets.all(10),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // ── Thumbnail ──────────────────────────────────────────
                  _Thumbnail(imageProvider: imageProvider, trail: trail),

                  const SizedBox(width: 12),

                  // ── Content ────────────────────────────────────────────
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        // Title row + badges
                        Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Expanded(
                              child: Text(
                                trail.name,
                                style: const TextStyle(
                                  fontWeight: FontWeight.bold,
                                  fontSize: 15,
                                ),
                                maxLines: 2,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                            const SizedBox(width: 6),
                            _BadgeRow(
                              isPublic: trail.public == true,
                              isShared: trailIsShared,
                            ),
                          ],
                        ),

                        const SizedBox(height: 3),

                        // Date + author
                        Row(
                          children: [
                            if (trail.summaryDate != null) ...[
                              Text(
                                DateFormat.yMMMd(
                                  locale,
                                ).format(trail.summaryDate!),
                                style: TextStyle(
                                  color: Colors.grey[600],
                                  fontSize: 11,
                                ),
                              ),
                              const SizedBox(width: 8),
                            ],
                            CircleAvatar(
                              radius: 9,
                              backgroundColor: Colors.grey.shade300,
                              backgroundImage: NetworkImage(
                                trail.summaryAuthorAvatar.isNotEmpty
                                    ? trail.summaryAuthorAvatar
                                    : "https://api.dicebear.com/7.x/initials/png?seed=${trail.summaryAuthorName}&backgroundType=gradientLinear",
                              ),
                              onBackgroundImageError: (_, _) =>
                                  const FaIcon(FontAwesomeIcons.user, size: 9),
                            ),
                            const SizedBox(width: 4),
                            Expanded(
                              child: Text(
                                "${trail.summaryAuthorName}${(trail.domain?.isNotEmpty ?? false) ? "@${trail.domain}" : ""}",
                                style: TextStyle(
                                  color: Colors.grey[600],
                                  fontSize: 11,
                                ),
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                          ],
                        ),

                        const SizedBox(height: 6),

                        // Category / location / difficulty
                        Wrap(
                          spacing: 10,
                          runSpacing: 4,
                          children: [
                            if (trail.summaryCategory.isNotEmpty)
                              _CategoryIcon(value: trail.summaryCategory),
                            if (trail.location != null &&
                                trail.location!.isNotEmpty)
                              _InlineIcon(
                                icon: FontAwesomeIcons.locationDot,
                                label: trail.location!,
                              ),
                            _InlineIcon(
                              icon: FontAwesomeIcons.gauge,
                              label: _getDifficultyLabel(
                                context,
                                trail.summaryDifficulty,
                              ),
                            ),
                          ],
                        ),

                        const SizedBox(height: 6),

                        // Tags
                        if (trail.summaryTags != null &&
                            trail.summaryTags!.isNotEmpty) ...[
                          Wrap(
                            spacing: 4,
                            runSpacing: 4,
                            children: trail.summaryTags!
                                .take(3)
                                .map((tag) => _Chip(text: tag))
                                .toList(),
                          ),
                          const SizedBox(height: 6),
                        ],

                        // Stats row
                        _StatsRow(trail: trail, unit: unit),
                      ],
                    ),
                  ),
                ],
              ),
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
}

// ── Private helpers ────────────────────────────────────────────────────────────

class _Thumbnail extends StatelessWidget {
  final ImageProvider? imageProvider;
  final TrailSummary trail;

  const _Thumbnail({required this.imageProvider, required this.trail});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(12),
      child: SizedBox(
        width: 90,
        height: 90,
        child: imageProvider != null
            ? Image(
                image: imageProvider!,
                fit: BoxFit.cover,
                errorBuilder: (context, _, _) => _placeholder(context),
              )
            : _placeholder(context),
      ),
    );
  }

  Widget _placeholder(BuildContext context) {
    return Container(
      color: Colors.grey[200],
      child: Center(
        child: SvgPicture.asset(
          "assets/svgs/empty_state_trail_${Theme.of(context).brightness.name}.svg",
          semanticsLabel: 'wanderer logo',
          height: 40,
        ),
      ),
    );
  }
}

class _BadgeRow extends StatelessWidget {
  final bool isPublic;
  final bool isShared;

  const _BadgeRow({required this.isPublic, required this.isShared});

  @override
  Widget build(BuildContext context) {
    if (!isPublic && !isShared) return const SizedBox.shrink();
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (isPublic)
          _BadgeIcon(child: const FaIcon(FontAwesomeIcons.globe, size: 10)),
        if (isShared) ...[
          if (isPublic) const SizedBox(width: 3),
          _BadgeIcon(
            child: const FaIcon(FontAwesomeIcons.shareNodes, size: 10),
          ),
        ],
      ],
    );
  }
}

class _BadgeIcon extends StatelessWidget {
  final Widget child;
  const _BadgeIcon({required this.child});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainer,
        shape: BoxShape.circle,
      ),
      child: child,
    );
  }
}

class _CategoryIcon extends StatelessWidget {
  final String value;
  const _CategoryIcon({required this.value});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        getTrailIcon(value, color: Colors.grey[700]),
        const SizedBox(width: 4),
        Text(value, style: TextStyle(fontSize: 12, color: Colors.grey[800])),
      ],
    );
  }
}

class _InlineIcon extends StatelessWidget {
  final FaIconData icon;
  final String label;

  const _InlineIcon({required this.icon, required this.label});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        FaIcon(icon, size: 12, color: Colors.grey[700]),
        const SizedBox(width: 4),
        Flexible(
          child: Text(
            label,
            style: TextStyle(fontSize: 12, color: Colors.grey[800]),
          ),
        ),
      ],
    );
  }
}

class _Chip extends StatelessWidget {
  final String text;
  const _Chip({required this.text});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: Colors.grey[100],
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        text,
        style: const TextStyle(fontSize: 10, color: Colors.black87),
      ),
    );
  }
}

class _StatsRow extends StatelessWidget {
  final TrailSummary trail;
  final String unit;
  const _StatsRow({required this.trail, required this.unit});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 6,
      runSpacing: 6,
      children: [
        StatChip(
          icon: FontAwesomeIcons.ruler,
          label: formatDistance(trail.distance, unit: unit),
        ),
        StatChip(
          icon: FontAwesomeIcons.clock,
          label: Duration(
            seconds: trail.duration.toInt(),
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
      ],
    );
  }
}
