import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/server_instance.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/welcome/recent_servers_provider.dart';
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

  bool _prefilled = false;

  @override
  void initState() {
    super.initState();

    _prefillFromSelection(
      ref.read(serverSelectionProvider).value?.selectedServer,
      notify: false,
    );
  }

  @override
  void dispose() {
    _urlController.dispose();
    super.dispose();
  }

  void _prefillFromSelection(ServerInstance? selected, {required bool notify}) {
    if (_prefilled || selected == null) return;
    final url = selected.url;
    if (url.isEmpty) return;

    _prefilled = true;
    _urlController.value = TextEditingValue(
      text: url,
      selection: TextSelection.collapsed(offset: url.length),
    );
    _searchQuery = url;
    if (notify) setState(() {});
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
  ///
  /// Recording as "last used" happens only on a successful login/OAuth/register
  /// (see `auth_provider.dart`'s `_recordLastUsedServer`) — merely selecting a
  /// server here must NOT record it.
  void _selectAndGoBack(ServerInstance server) {
    final url = normalizeServerUrl(server.url);
    if (url == null) return;

    ref
        .read(serverSelectionProvider.notifier)
        .setSelectedServer(server.copyWith(url: url));
    ref.read(apiProvider.notifier).updateBaseUrl(url);

    context.pop();
  }

  /// Strips a leading `https?://` for display, same regex as
  /// `ServerSelector._displayUrl`-equivalent logic in `server_selctor.dart`.
  String _displayUrl(String url) {
    return url.replaceFirst(RegExp(r'https?://'), '');
  }

  /// The 48x48 fallback icon shown when a server has no image, or its image
  /// fails to load.
  Widget _buildServerIconFallback(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      color: theme.colorScheme.surfaceContainerHighest,
      width: 48,
      height: 48,
      child: const Center(child: FaIcon(FontAwesomeIcons.server)),
    );
  }

  /// A single server row, shared by the "Last used" section and the remote
  /// list. Title falls back to the URL (scheme stripped) when there is no
  /// name, and the leading image falls back to the server icon when there is
  /// no image — so a nameless, image-less entry (a typed custom URL) renders
  /// safely, with no request to `https://wanderer.to/null`.
  Widget _buildServerTile(BuildContext context, ServerInstance server) {
    final theme = Theme.of(context);
    final hasImage = server.image != null && server.image!.isNotEmpty;

    return ListTile(
      contentPadding: const EdgeInsets.symmetric(
        horizontal: 16,
        vertical: 8,
      ),
      leading: hasImage
          ? ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: Image(
                image: CachedNetworkImageProvider(
                  "https://wanderer.to/${server.image}",
                ),
                width: 48,
                height: 48,
                fit: BoxFit.cover,
                errorBuilder: (_, _, _) => _buildServerIconFallback(context),
              ),
            )
          : _buildServerIconFallback(context),
      title: Text(
        server.name ?? _displayUrl(server.url),
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
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final severSelection = ref.watch(serverSelectionProvider);
    final recentServers = ref.watch(recentServersProvider);
    final l10n = AppLocalizations.of(context)!;

    // Late-resolution fallback for the initState prefill. Listener callbacks
    // run after the frame, so touching the controller here is safe.
    ref.listen(serverSelectionProvider, (_, next) {
      _prefillFromSelection(next.value?.selectedServer, notify: true);
    });

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
            child: CustomScrollView(
              slivers: [
                // The "Last used" section is intentionally NOT filtered by
                // _searchQuery (the field is prefilled with the current
                // selection's URL, which would routinely hide it) and does
                // not depend on the servers.json AsyncValue, so it renders
                // even while that fetch is loading or has failed.
                if (recentServers.isNotEmpty) ...[
                  _buildSectionHeader(context, l10n.last_used),
                  SliverList.separated(
                    itemCount: recentServers.length,
                    separatorBuilder: (context, index) =>
                        const Divider(height: 1),
                    itemBuilder: (context, index) =>
                        _buildServerTile(context, recentServers[index]),
                  ),
                  // A filled band rather than a plain Divider: a hairline
                  // reads as just another tile separator.
                  SliverToBoxAdapter(
                    child: Container(
                      height: 8,
                      margin: const EdgeInsets.only(top: 8),
                      decoration: BoxDecoration(
                        color: theme.colorScheme.surfaceContainerHighest,
                        border: Border.symmetric(
                          horizontal: BorderSide(
                            color: theme.colorScheme.outlineVariant,
                          ),
                        ),
                      ),
                    ),
                  ),
                  _buildSectionHeader(context, l10n.all_instances),
                ],

                ...severSelection.when(
                  data: (serverState) {
                    final filteredServers = serverState.availableServers
                        .where(
                          (s) =>
                              (s.name ?? '').toLowerCase().contains(
                                _searchQuery.toLowerCase(),
                              ) ||
                              s.url.toLowerCase().contains(
                                _searchQuery.toLowerCase(),
                              ),
                        )
                        .toList();

                    if (filteredServers.isEmpty && _searchQuery.isNotEmpty) {
                      return [
                        SliverFillRemaining(
                          hasScrollBody: false,
                          child: Center(
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
                                    ServerInstance(
                                      url: _urlController.text.trim(),
                                    ),
                                  ),
                                  child: Text(l10n.use_custom_url_instead),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ];
                    }

                    return [
                      SliverList.separated(
                        itemCount: filteredServers.length,
                        separatorBuilder: (context, index) =>
                            const Divider(height: 1),
                        itemBuilder: (context, index) =>
                            _buildServerTile(context, filteredServers[index]),
                      ),
                    ];
                  },
                  loading: () => [
                    const SliverFillRemaining(
                      hasScrollBody: false,
                      child: Center(child: CircularProgressIndicator()),
                    ),
                  ],
                  error: (err, stack) => [
                    SliverFillRemaining(
                      hasScrollBody: false,
                      child: WandererError(err: err, stack: stack),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSectionHeader(BuildContext context, String label) {
    final theme = Theme.of(context);
    return SliverToBoxAdapter(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
        child: Text(
          label,
          style: theme.textTheme.labelLarge?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
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
