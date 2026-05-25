import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/models/server_instance.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart'; // Your custom button

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

  void _selectAndGoBack(ServerInstance server) {
    var url = server.url.trim();
    if (url.isEmpty) return;

    // Clean up spaces which are often typed instead of dots on mobile keyboards
    url = url.replaceAll(' ', '.').replaceAll(RegExp(r'\.+'), '.');

    // If we corrupted the http:// or https:// prefix, strip it so we can rebuild it cleanly
    url = url.replaceFirst(RegExp(r'^https?:?/?/?\.?'), '');
    url = url.replaceFirst(RegExp(r'^http?:?/?/?\.?'), '');

    // Now determine the correct protocol
    final bool isHttp = server.url.trim().startsWith('http://');
    url = isHttp ? 'http://$url' : 'https://$url';

    final selectedServer = server.copyWith(url: url);
    ref
        .read(serverSelectionProvider.notifier)
        .setSelectedServer(selectedServer);
    ref.read(apiProvider.notifier).updateBaseUrl(selectedServer.url);

    context.pop();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final severSelection = ref.watch(serverSelectionProvider);

    return Scaffold(
      appBar: AppBar(title: const Text("Select Instance")),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: TextField(
              controller: _urlController,
              decoration: InputDecoration(
                hintText: "Enter server URL (e.g. wanderer.to)",
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
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
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
                        Text("No servers match '$_searchQuery'"),
                        TextButton(
                          onPressed: () => _selectAndGoBack(
                            ServerInstance(url: _urlController.text.trim()),
                          ),
                          child: const Text("Use custom URL instead"),
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
                        child: Image.network(
                          "https://wanderer.to/${server.image}",
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
        color: Theme.of(context).colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        text,
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
          color: Theme.of(context).colorScheme.onSecondaryContainer,
          fontSize: 10,
        ),
      ),
    );
  }
}
