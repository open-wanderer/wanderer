import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:intl/intl.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';

class TrailCard extends ConsumerWidget {
  final TrailSearchResult trail;
  final bool fullWidth;
  final bool selected;
  final VoidCallback? onTrailSelect;

  const TrailCard({
    super.key,
    required this.trail,
    this.fullWidth = false,
    this.selected = false,
    this.onTrailSelect,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final trailIsShared = trail.shares?.isNotEmpty ?? false;
    final user = ref.watch(authProvider).value!;

    final thumbnail = trail.getFileUrl(
      user.serverUrl,
      trail.thumbnail,
      thumb: "600x0",
    );

    final String locale = Localizations.localeOf(context).toString();

    return InkWell(
      onTap: onTrailSelect,
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 8),
        width: fullWidth ? double.infinity : 288,
        decoration: BoxDecoration(
          color: Theme.of(context).cardColor,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(
            color: selected
                ? Theme.of(context).primaryColor
                : Colors.grey.withValues(alpha: 0.2),
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
                    child: thumbnail != null
                        ? Image.network(thumbnail, fit: BoxFit.cover)
                        : SvgPicture.asset(
                            "assets/svgs/empty_state_trail_${Brightness.light.name}.svg",
                            semanticsLabel: 'wanderer logo',
                            height: 80,
                          ),
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
                          child: FaIcon(FontAwesomeIcons.shareNodes, size: 16),
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

                  Text(
                    DateFormat.yMMMMd(locale).format(
                      DateTime.fromMillisecondsSinceEpoch(trail.date * 1000),
                    ),
                    style: TextStyle(color: Colors.grey[600], fontSize: 12),
                  ),

                  const SizedBox(height: 4),

                  Column(
                    children: [
                      Row(
                        children: [
                          Text(
                            AppLocalizations.of(context)?.by ?? "by",
                            style: TextStyle(
                              color: Colors.grey[600],
                              fontSize: 12,
                            ),
                          ),
                          const SizedBox(width: 6),
                          CircleAvatar(
                            radius: 12,
                            backgroundColor: Colors.grey.shade300,
                            backgroundImage: NetworkImage(
                              trail.authorAvatar.isNotEmpty
                                  ? trail.authorAvatar
                                  : "https://api.dicebear.com/7.x/initials/png?seed=${trail.authorName}&backgroundType=gradientLinear",
                            ),
                            onBackgroundImageError: (_, _) =>
                                FaIcon(FontAwesomeIcons.user),
                          ),
                          const SizedBox(width: 6),
                          Text(
                            "${trail.authorName}${(trail.domain?.isNotEmpty ?? false) ? "@${trail.domain}" : ""}",
                            style: TextStyle(
                              color: Colors.grey[600],
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
                      if (trail.category.isNotEmpty)
                        _CategoryIcon(
                          icon: FontAwesomeIcons.shapes,
                          value: trail.category,
                        ),

                      if (trail.location.isNotEmpty)
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
                      _CategoryIcon(
                        icon: FontAwesomeIcons.gauge,
                        value: _getDifficultyLabel(context, trail.difficulty),
                      ),
                    ],
                  ),

                  if (trail.tags != null && trail.tags!.isNotEmpty)
                    Column(
                      children: [
                        const SizedBox(height: 16),
                        Wrap(
                          spacing: 6,
                          children: trail.tags!
                              .take(3)
                              .map((tag) => _Chip(text: tag))
                              .toList(),
                        ),
                        const SizedBox(height: 6),
                      ],
                    ),

                  _StatsGrid(trail: trail),
                ],
              ),
            ),
          ],
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

class _BadgeWrapper extends StatelessWidget {
  final Widget child;
  const _BadgeWrapper({required this.child});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(6),
      decoration: const BoxDecoration(
        color: Colors.white,
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
        color: Colors.grey[100],
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        text,
        style: const TextStyle(fontSize: 11, color: Colors.black87),
      ),
    );
  }
}

class _StatsGrid extends StatelessWidget {
  final TrailSearchResult trail;
  const _StatsGrid({required this.trail});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.only(top: 12),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          _StatIcon(
            icon: FontAwesomeIcons.leftRight,
            value: "${(trail.distance / 1000).toStringAsFixed(2)} km",
          ),
          _StatIcon(
            icon: FontAwesomeIcons.clock,
            value: Duration(
              seconds: trail.duration.toInt(),
            ).pretty(abbreviated: true, tersity: DurationTersity.minute),
          ),
          _StatIcon(
            icon: FontAwesomeIcons.arrowTrendUp,
            value: "${trail.elevationGain.toInt()} m",
          ),
          _StatIcon(
            icon: FontAwesomeIcons.arrowTrendDown,
            value: "${trail.elevationLoss.toInt()} m",
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

class _CategoryIcon extends StatelessWidget {
  final FaIconData icon;
  final String value;
  const _CategoryIcon({required this.icon, required this.value});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        FaIcon(icon, size: 16),
        const SizedBox(width: 6),
        Text(value, style: TextStyle(fontSize: 16)),
      ],
    );
  }
}

class _StatIcon extends StatelessWidget {
  final FaIconData icon;
  final String value;
  const _StatIcon({required this.icon, required this.value});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        FaIcon(icon, size: 12, color: Colors.grey[600]),
        const SizedBox(width: 6),
        Text(value, style: TextStyle(fontSize: 12, color: Colors.grey[800])),
      ],
    );
  }
}
