import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/provider/router_provider.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';

class ServerSelector extends ConsumerWidget {
  final FaIconData? icon;
  const ServerSelector({super.key, this.icon = FontAwesomeIcons.chevronRight});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final serverSelection = ref.watch(serverSelectionProvider);

    final router = ref.watch(routerProvider);

    final theme = Theme.of(context);
    final bool hasServer =
        serverSelection.value != null &&
        serverSelection.value?.selectedServer != null;

    String serverTitle = "Select a Server";
    String serverSubtitle = "Required to login or register";

    if (hasServer && serverSelection.value?.selectedServer!.name != null) {
      serverTitle = serverSelection.value!.selectedServer!.name!;
      serverSubtitle = serverSelection.value!.selectedServer!.url.replaceFirst(
        RegExp(r'https?://'),
        '',
      );
    } else if (hasServer) {
      serverTitle = serverSelection.value!.selectedServer!.url.replaceFirst(
        RegExp(r'https?://'),
        '',
      );
      serverSubtitle = "Tap to change instance";
    }

    Widget serverIcon = FaIcon(
      FontAwesomeIcons.fediverse,
      color: theme.colorScheme.onPrimaryContainer,
      size: 20,
    );
    if (hasServer && serverSelection.value?.selectedServer!.image != null) {
      serverIcon = ClipOval(
        child: Image(
          image: CachedNetworkImageProvider(
            "https://wanderer.to/${serverSelection.value?.selectedServer!.image!}",
          ),
          width: 48,
          height: 48,
          fit: BoxFit.cover,
          errorBuilder: (_, _, _) => FaIcon(FontAwesomeIcons.server),
        ),
      );
    }

    return Material(
      type: MaterialType.card,
      color: theme.colorScheme.surfaceContainerLow,
      borderRadius: BorderRadius.circular(16),
      clipBehavior: Clip.antiAlias,
      child: ListTile(
        onTap: () {
          router.push('/select-server');
        },
        leading: CircleAvatar(
          backgroundColor: theme.colorScheme.primaryContainer,
          child: serverIcon,
        ),
        title: Text(
          serverTitle,
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.bold,
            color: hasServer ? null : theme.colorScheme.onSurfaceVariant,
          ),
        ),
        subtitle: Text(serverSubtitle, style: theme.textTheme.bodySmall),
        trailing: FaIcon(icon, size: 14),

        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      ),
    );
  }
}
