import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/services/server_capabilities.dart';

final serverCapabilitiesProvider = Provider<ServerCapabilities>((ref) {
  final capabilities = ServerCapabilities(
    ref.watch(apiProvider),
    initiallyActive: false,
  );
  final notifier = ref.read(apiProvider.notifier);
  void changed() {
    capabilities.reset();
    capabilities.discover();
  }

  notifier.onServerChanged = changed;
  ref.onDispose(() {
    if (identical(notifier.onServerChanged, changed)) {
      notifier.onServerChanged = null;
    }
    capabilities.dispose();
  });
  return capabilities;
});
