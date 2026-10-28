import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/foreground_position_stream_provider.dart';

/// Round floating "center on my location" button. Disabled until there is
/// a fix; what centering means (zoom, follow-up searches) is up to
/// [onPressed].
class CenterOnLocationButton extends ConsumerWidget {
  final ValueChanged<ml.Geographic> onPressed;

  const CenterOnLocationButton({super.key, required this.onPressed});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return StreamBuilder<LocationMarkerPosition?>(
      stream: ref.watch(liveLocationProvider),
      builder: (context, snapshot) {
        final position = snapshot.data;
        return Material(
          color: Theme.of(context).colorScheme.surface,
          elevation: 4,
          shadowColor: Colors.black26,
          shape: const CircleBorder(),
          child: IconButton(
            icon: const FaIcon(FontAwesomeIcons.locationCrosshairs, size: 18),
            tooltip: AppLocalizations.of(context)!.center_on_my_location,
            onPressed: position == null
                ? null
                : () => onPressed(
                    ml.Geographic(
                      lat: position.latitude,
                      lon: position.longitude,
                    ),
                  ),
          ),
        );
      },
    );
  }
}
