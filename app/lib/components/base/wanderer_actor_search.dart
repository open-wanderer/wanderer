import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/api_provider.dart';

class WandererActorSearch extends ConsumerStatefulWidget {
  final String? hintText;
  final String? initialAuthorId;
  final String? initialAuthorLabel;
  final void Function(ActorSearchResult actor) onSelected;
  final VoidCallback onCleared;

  const WandererActorSearch({
    super.key,
    this.hintText,
    this.initialAuthorId,
    this.initialAuthorLabel,
    required this.onSelected,
    required this.onCleared,
  });

  @override
  ConsumerState<WandererActorSearch> createState() =>
      _WandererActorSearchState();
}

class _WandererActorSearchState extends ConsumerState<WandererActorSearch> {
  late final TextEditingController _controller;
  final FocusNode _focusNode = FocusNode();
  Timer? _debounce;

  List<ActorSearchResult> _results = [];
  bool _isLoading = false;
  bool _showDropdown = false;
  ActorSearchResult? _selected;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController(
      text: (widget.initialAuthorLabel != null &&
              widget.initialAuthorLabel!.isNotEmpty)
          ? widget.initialAuthorLabel
          : '',
    );
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _controller.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  void _onChanged(String value) {
    if (_debounce?.isActive ?? false) {
      _debounce!.cancel();
    }

    if (value.isEmpty) {
      setState(() {
        _results = [];
        _showDropdown = false;
        _selected = null;
      });
      widget.onCleared();
      return;
    }

    _debounce = Timer(const Duration(milliseconds: 500), () {
      _search(value);
    });
  }

  Future<void> _search(String query) async {
    setState(() {
      _isLoading = true;
    });

    List<ActorSearchResult> results = [];
    try {
      final api = ref.read(apiProvider);
      final response =
          await api.get('/search/actor', queryParameters: {'q': query});
      results = ((response.data['items'] as List<dynamic>?) ?? [])
          .map((e) => ActorSearchResult.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (_) {
      results = [];
    }

    if (!mounted) return;
    setState(() {
      _isLoading = false;
      _results = results;
      _showDropdown = true;
    });
  }

  void _onSelect(ActorSearchResult actor) {
    final displayName =
        actor.username.isNotEmpty ? actor.username : actor.preferredUsername;
    setState(() {
      _selected = actor;
      _controller.text = displayName;
      _showDropdown = false;
      _results = [];
    });
    _focusNode.unfocus();
    widget.onSelected(actor);
  }

  void _onClear() {
    setState(() {
      _controller.clear();
      _selected = null;
      _results = [];
      _showDropdown = false;
    });
    widget.onCleared();
  }

  @override
  Widget build(BuildContext context) {
    final showClear = _selected != null || _controller.text.isNotEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextField(
          controller: _controller,
          focusNode: _focusNode,
          onChanged: _onChanged,
          decoration: InputDecoration(
            hintText: widget.hintText,
            hintStyle: const TextStyle(color: Colors.grey),
            filled: true,
            fillColor: Theme.of(context).inputDecorationTheme.fillColor,
            contentPadding: const EdgeInsets.all(12),
            prefixIcon: Icon(Icons.search, color: Colors.grey.shade300),
            suffixIcon: showClear
                ? IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: _onClear,
                  )
                : null,
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(56),
              borderSide: BorderSide(
                color: Theme.of(context).colorScheme.outline,
              ),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(56),
              borderSide: BorderSide(
                color: Theme.of(context).colorScheme.onSurface,
                width: 1.5,
              ),
            ),
          ),
        ),
        if (_isLoading)
          const Padding(
            padding: EdgeInsets.symmetric(vertical: 8.0),
            child: LinearProgressIndicator(),
          ),
        if (_showDropdown && _results.isNotEmpty)
          ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 280),
            child: ListView.builder(
              shrinkWrap: true,
              itemCount: _results.length,
              itemBuilder: (context, index) {
                final actor = _results[index];
                final displayName = actor.username.isNotEmpty
                    ? actor.username
                    : actor.preferredUsername;
                final subtitle =
                    '@${actor.preferredUsername}${actor.isLocal ? "" : "@${actor.domain}"}';

                return ListTile(
                  leading: CircleAvatar(
                    radius: 20,
                    backgroundColor: Colors.grey.shade300,
                    backgroundImage:
                        actor.icon != null && actor.icon!.isNotEmpty
                            ? NetworkImage(actor.icon!)
                            : NetworkImage(
                                'https://api.dicebear.com/7.x/initials/png?seed=$displayName',
                              ),
                  ),
                  title: Text(
                    displayName,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  subtitle: Text(subtitle),
                  onTap: () => _onSelect(actor),
                );
              },
            ),
          ),
      ],
    );
  }
}
