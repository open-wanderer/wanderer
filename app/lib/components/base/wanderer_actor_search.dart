import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:skeletonizer/skeletonizer.dart';

class WandererActorSearch extends ConsumerStatefulWidget {
  final String? hintText;
  final ActorSearchResult? initialActor;
  final void Function(ActorSearchResult actor) onSelected;
  final VoidCallback onCleared;

  const WandererActorSearch({
    super.key,
    this.hintText,
    this.initialActor,
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
  final LayerLink _layerLink = LayerLink();
  Timer? _debounce;
  OverlayEntry? _overlayEntry;

  List<ActorSearchResult> _results = [];
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController();
    _focusNode.addListener(() {
      if (!_focusNode.hasFocus) _removeOverlay();
    });
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _removeOverlay();
    _controller.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  void _removeOverlay() {
    _overlayEntry?.remove();
    _overlayEntry = null;
  }

  void _showOverlay() {
    _removeOverlay();
    final overlay = Overlay.of(context);
    _overlayEntry = OverlayEntry(
      builder: (_) => CompositedTransformFollower(
        link: _layerLink,
        showWhenUnlinked: false,
        offset: const Offset(0, 52),
        child: Align(
          alignment: Alignment.topLeft,
          child: _DropdownPanel(
            results: _results,
            isLoading: _isLoading,
            onSelect: _onSelect,
          ),
        ),
      ),
    );
    overlay.insert(_overlayEntry!);
  }

  void _onChanged(String value) {
    if (_debounce?.isActive ?? false) _debounce!.cancel();

    if (value.isEmpty) {
      setState(() => _results = []);
      _removeOverlay();
      widget.onCleared();
      return;
    }

    _debounce = Timer(const Duration(milliseconds: 500), () => _search(value));
  }

  Future<void> _search(String query) async {
    setState(() => _isLoading = true);
    _showOverlay();

    List<ActorSearchResult> results = [];
    try {
      final api = ref.read(apiProvider);
      final response = await api.get(
        '/search/actor',
        queryParameters: {'q': query},
      );
      results = ((response.data['hits'] as List<dynamic>?) ?? [])
          .map((e) => ActorSearchResult.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (_) {
      results = [];
    }

    if (!mounted) return;
    setState(() {
      _isLoading = false;
      _results = results;
    });

    if (results.isEmpty) {
      _removeOverlay();
    } else {
      _showOverlay();
    }
  }

  void _onSelect(ActorSearchResult actor) {
    setState(() {
      _controller.clear();
      _results = [];
    });
    _removeOverlay();
    _focusNode.unfocus();
    widget.onSelected(actor);
  }

  void _onClear() {
    setState(() {
      _controller.clear();
      _results = [];
    });
    _removeOverlay();
    widget.onCleared();
  }

  @override
  Widget build(BuildContext context) {
    final showClear = _controller.text.isNotEmpty;

    return CompositedTransformTarget(
      link: _layerLink,
      child: TextField(
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
              ? IconButton(icon: const Icon(Icons.close), onPressed: _onClear)
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
    );
  }
}

class _DropdownPanel extends StatelessWidget {
  final List<ActorSearchResult> results;
  final bool isLoading;
  final void Function(ActorSearchResult) onSelect;

  const _DropdownPanel({
    required this.results,
    required this.isLoading,
    required this.onSelect,
  });

  @override
  Widget build(BuildContext context) {
    return Material(
      elevation: 4,
      borderRadius: BorderRadius.circular(12),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxHeight: 280, maxWidth: 400),
        child: ClipRRect(
          borderRadius: BorderRadius.circular(12),
          child: Skeletonizer(
            enabled: isLoading,
            child: ListView.builder(
              shrinkWrap: true,
              padding: EdgeInsets.zero,
              itemCount: results.length,
              itemBuilder: (context, index) {
                final actor = results[index];
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
                  onTap: () => onSelect(actor),
                );
              },
            ),
          ),
        ),
      ),
    );
  }
}
