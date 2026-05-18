import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_html/flutter_html.dart' as html;
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/trail/stat_chip.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/format_util.dart';

class TrailPanel extends StatelessWidget {
  const TrailPanel({
    super.key,
    required this.trail,
    required this.scrollController,
    this.actionMenu,
  });

  final Trail trail;
  final ScrollController scrollController;
  final Widget? actionMenu;

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      controller: scrollController,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(
              margin: const EdgeInsets.symmetric(vertical: 12),
              width: 40,
              height: 5,
              decoration: BoxDecoration(
                color: Colors.grey.withValues(alpha: 0.3),
                borderRadius: BorderRadius.circular(10),
              ),
            ),
          ),

          Padding(
            padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        trail.name,
                        style: Theme.of(context).textTheme.headlineMedium
                            ?.copyWith(fontWeight: FontWeight.bold),
                      ),
                    ),
                    ?actionMenu,
                  ],
                ),
                const SizedBox(height: 12),

                Wrap(
                  spacing: 10,
                  runSpacing: 10,
                  children: [
                    StatChip(
                      icon: FontAwesomeIcons.ruler,
                      label: formatDistance(trail.distance),
                    ),
                    StatChip(
                      icon: FontAwesomeIcons.clock,
                      label: Duration(seconds: trail.duration.toInt()).pretty(
                        abbreviated: true,
                        tersity: DurationTersity.minute,
                      ),
                    ),
                    StatChip(
                      icon: FontAwesomeIcons.arrowTrendUp,
                      label: formatElevation(trail.elevationGain),
                    ),
                    StatChip(
                      icon: FontAwesomeIcons.arrowTrendDown,
                      label: formatElevation(trail.elevationLoss),
                    ),
                    if (trail.expand?.category != null)
                      StatChip(
                        icon: FontAwesomeIcons.route,
                        label: trail.expand!.category!.name,
                      ),
                  ],
                ),

                const Divider(height: 32),

                Text(
                  AppLocalizations.of(context)!.description,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 8),
                if (trail.description.isNotEmpty)
                  html.Html(data: trail.description),
                if (trail.description.isEmpty)
                  Text(
                    AppLocalizations.of(context)!.no_description_for_now,
                    style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                      color: Colors.black87,
                      height: 1.5,
                    ),
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
