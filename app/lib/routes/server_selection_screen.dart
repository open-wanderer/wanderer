import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/server_instance.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';
import 'package:wanderer/util/server_url.dart';

class ServerSelectionScreen extends ConsumerStatefulWidget {
  const ServerSelectionScreen({super.key});

  @override
  ConsumerState<ServerSelectionScreen> createState() =>
      _ServerSelectionScreenState();
}

class _ServerSelectionScreenState extends ConsumerState<ServerSelectionScreen> {
  final _urlController = TextEditingController();
  String _searchQuery = "";

  @override
  void dispose() {
    _urlController.dispose();
    super.dispose();
  }

  /// Applies [server] as the selected instance and closes the picker.
  ///
  /// The normalised URL is what gets stored AND what the api client is pointed
  /// at — passing the raw text to `updateBaseUrl` while normalising only a
  /// local copy meant a bare host ("wanderer.to", exactly what this screen's
  /// own hint suggests) reached Dio as the hostless "wanderer.to/api/v1". Dio's
  /// `baseUrl` setter throws on that, so the screen never reached `pop()` and
  /// simply appeared to ignore the input.
  ///
  /// Unusable input (empty, or still hostless after normalisation) leaves the
  /// picker open rather than selecting something the client cannot talk to.
  void _selectAndGoBack(ServerInstance server) {
    final url = normalizeServerUrl(server.url);
    if (url == null) return;

    ref
        .read(serverSelectionProvider.notifier)
        .setSelectedServer(server.copyWith(url: url));
    ref.read(apiProvider.notifier).updateBaseUrl(url);

    context.pop();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final severSelection = ref.watch(serverSelectionProvider);
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.select_instance)),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: TextField(
              controller: _urlController,
              cursorColor: Theme.of(context).colorScheme.onSurface,
              decoration: InputDecoration(
                hintText: l10n.enter_server_url_hint,
                hintStyle: TextStyle(color: Colors.grey),
                prefixIcon: const Icon(Icons.link),

                suffixIcon: IconButton(
                  icon: const FaIcon(FontAwesomeIcons.chevronRight, size: 16),
                  onPressed: () {
                    if (_urlController.text.isNotEmpty) {
                      _selectAndGoBack(
                        ServerInstance(url: _urlController.text.trim()),
                      );
                    }
                  },
                ),
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                  borderSide: BorderSide(
                    color: Theme.of(context).colorScheme.outline,
                  ),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                  borderSide: BorderSide(
                    color: Theme.of(context).colorScheme.outlineVariant,
                  ),
                ),
              ),
              onChanged: (value) => setState(() => _searchQuery = value),
              onSubmitted: (value) =>
                  _selectAndGoBack(ServerInstance(url: value)),
            ),
          ),

          const Divider(),

          Expanded(
            child: severSelection.when(
              data: (serverState) {
                final filteredServers = serverState.availableServers
                    .where(
                      (s) =>
                          s.name!.toLowerCase().contains(
                            _searchQuery.toLowerCase(),
                          ) ||
                          s.url.toLowerCase().contains(
                            _searchQuery.toLowerCase(),
                          ),
                    )
                    .toList();

                if (filteredServers.isEmpty && _searchQuery.isNotEmpty) {
                  return Center(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        const FaIcon(
                          FontAwesomeIcons.magnifyingGlass,
                          size: 48,
                        ),
                        const SizedBox(height: 16),
                        Text(l10n.no_servers_match_query(_searchQuery)),
                        TextButton(
                          onPressed: () => _selectAndGoBack(
                            ServerInstance(url: _urlController.text.trim()),
                          ),
                          child: Text(l10n.use_custom_url_instead),
                        ),
                      ],
                    ),
                  );
                }

                return ListView.separated(
                  itemCount: filteredServers.length,
                  separatorBuilder: (context, index) =>
                      const Divider(height: 1),
                  itemBuilder: (context, index) {
                    final server = filteredServers[index];
                    return ListTile(
                      contentPadding: const EdgeInsets.symmetric(
                        horizontal: 16,
                        vertical: 8,
                      ),
                      leading: ClipRRect(
                        borderRadius: BorderRadius.circular(8),
                        child: Image(
                          image: CachedNetworkImageProvider(
                            "https://wanderer.to/${server.image}",
                          ),
                          width: 48,
                          height: 48,
                          fit: BoxFit.cover,
                          errorBuilder: (_, _, _) => Container(
                            color: theme.colorScheme.surfaceContainerHighest,
                            width: 48,
                            height: 48,
                            child: Center(
                              child: const FaIcon(FontAwesomeIcons.server),
                            ),
                          ),
                        ),
                      ),
                      title: Text(
                        server.name!,
                        style: const TextStyle(fontWeight: FontWeight.bold),
                      ),
                      subtitle: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(server.url, style: theme.textTheme.bodySmall),
                          const SizedBox(height: 4),
                          Wrap(
                            spacing: 4,
                            children: server.category
                                .map((c) => _buildTinyTag(context, c))
                                .toList(),
                          ),
                        ],
                      ),
                      onTap: () => _selectAndGoBack(server),
                    );
                  },
                );
              },
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (err, stack) => WandererError(err: err, stack: stack),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildTinyTag(BuildContext context, String text) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.secondaryContainer,
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        text,
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
          color: Theme.of(context).colorScheme.onSurface,
          fontSize: 10,
        ),
      ),
    );
  }
}
