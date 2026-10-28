import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// Shown in place of a result list when a non-empty search query matched
/// nothing.
class WandererNoResults extends StatelessWidget {
  final String query;

  const WandererNoResults({super.key, required this.query});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          SvgPicture.asset(
            "assets/svgs/empty_state_search_${Theme.of(context).brightness.name}.svg",
            semanticsLabel: 'wanderer search empty state',
            height: 120,
          ),
          const SizedBox(height: 12),
          Text(
            AppLocalizations.of(context)!.no_results_for_query(query),
            style: TextStyle(color: Colors.grey.shade500),
          ),
        ],
      ),
    );
  }
}
