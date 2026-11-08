import 'dart:async';

import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:tiptap_flutter/tiptap_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// A limited WYSIWYG editor that produces HTML, mirroring the web client's
/// TipTap editor (`web/src/lib/components/base/editor.svelte`).
///
/// Powered by `tiptap_flutter`, which runs the same TipTap engine as the web
/// (headless, off-screen) and renders natively — so [getHTML] output shares the
/// web editor's HTML semantics. The third-party dependency is deliberately
/// confined to this file so it can be swapped without touching callers.
///
/// Data flow is one-way (mirrors the web editor's guard): the document is seeded
/// from [initialValue] once, then edits flow out via [onChanged]. The current
/// [FormBuilderField] value is intentionally NOT fed back into the editor, which
/// would reset the cursor / cause update loops.
class WandererRichTextEditor extends StatefulWidget {
  final String initialValue;
  final ValueChanged<String> onChanged;
  final String? label;

  /// Height of the editing surface. [TiptapEditor] needs a bounded height (it
  /// expands internally), so it lives in a fixed-height box inside scrolling
  /// forms.
  final double editorHeight;

  const WandererRichTextEditor({
    super.key,
    required this.initialValue,
    required this.onChanged,
    this.label,
    this.editorHeight = 180,
  });

  @override
  State<WandererRichTextEditor> createState() => _WandererRichTextEditorState();
}

class _WandererRichTextEditorState extends State<WandererRichTextEditor> {
  // Raw command names not exposed as EditorCommand constants but registered in
  // the bundled engine (TipTap v3 StarterKit + custom Link).
  static const String _toggleUnderline = 'toggleUnderline';
  static const String _setLink = 'setLink';
  static const String _unsetLink = 'unsetLink';

  late final EditorController _controller;
  StreamSubscription<EditorStatePayload>? _stateSub;
  Timer? _debounce;

  bool _ready = false;
  Object? _error;
  String _lastEmitted = '';

  @override
  void initState() {
    super.initState();
    _lastEmitted = widget.initialValue;
    _controller = EditorController();
    // stateChangedStream is a broadcast stream, so listening alongside the
    // controller's own internal subscription is safe.
    _stateSub = _controller.editorStateStream.listen(_onEditorStateChanged);

    _controller
        .initialize(content: widget.initialValue)
        .then((_) {
          if (mounted) setState(() => _ready = true);
        })
        .catchError((Object e) {
          if (mounted) setState(() => _error = e);
        });
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _stateSub?.cancel();
    _controller.dispose();
    super.dispose();
  }

  void _onEditorStateChanged(EditorStatePayload _) {
    // Refresh toolbar active-state highlighting on every transaction.
    if (mounted) setState(() {});

    // The stream fires per transaction and getHTML() is async, so debounce the
    // HTML read/emit to keep typing smooth.
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 300), () async {
      if (!mounted) return;
      final html = await _controller.getHTML();
      if (!mounted || html == _lastEmitted) return;
      _lastEmitted = html;
      widget.onChanged(html);
    });
  }

  bool _markActive(String mark) => _controller.activeMarks.contains(mark);

  // tiptap_flutter is vendored (app/vendor/tiptap_flutter) specifically so its
  // rendering colors can be driven by TiptapPalette instead of the upstream
  // hardcoded literals — see the vendored copy's tiptap_palette.dart for why
  // (no theming hook, no exported registry/position types to override
  // cleanly). Assigning these here, synchronously before returning the
  // widget tree, is safe: TiptapEditor's subtree builds later in the same
  // frame, so the node builders read the values set on this build.
  void _applyPalette(ThemeData theme) {
    final onSurface = theme.colorScheme.onSurface;
    TiptapPalette.text = onSurface;
    TiptapPalette.muted = onSurface.withValues(alpha: 0.6);
    TiptapPalette.link = theme.colorScheme.primary;
    TiptapPalette.cursor = theme.colorScheme.primary;
    TiptapPalette.selectionHighlight = theme.colorScheme.primary.withValues(
      alpha: 0.25,
    );
    TiptapPalette.codeBackground = onSurface.withValues(alpha: 0.08);
    TiptapPalette.codeBlockBackground = onSurface.withValues(alpha: 0.06);
    TiptapPalette.codeBlockText = onSurface;
    TiptapPalette.blockquoteBorder = theme.colorScheme.outline;
    TiptapPalette.divider = theme.colorScheme.outline;
    TiptapPalette.imagePlaceholderBackground = onSurface.withValues(
      alpha: 0.06,
    );
    TiptapPalette.imagePlaceholderText = TiptapPalette.muted;
    TiptapPalette.unknownNodeBackground = onSurface.withValues(alpha: 0.06);
    TiptapPalette.unknownNodeBorder = theme.colorScheme.outline;
    TiptapPalette.unknownNodeText = TiptapPalette.muted;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    _applyPalette(theme);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (widget.label != null && widget.label!.isNotEmpty)
          Padding(
            padding: const EdgeInsets.only(bottom: 4),
            child: Text(
              widget.label!,
              style: theme.textTheme.bodySmall?.copyWith(
                fontWeight: FontWeight.w600,
                color: theme.colorScheme.onSurface,
              ),
            ),
          ),
        Container(
          decoration: BoxDecoration(
            color: theme.inputDecorationTheme.fillColor,
            borderRadius: BorderRadius.circular(6),
            border: Border.all(color: theme.colorScheme.outline),
          ),
          clipBehavior: Clip.antiAlias,
          child: _buildBody(theme),
        ),
      ],
    );
  }

  Widget _buildBody(ThemeData theme) {
    if (_error != null) {
      return SizedBox(
        height: widget.editorHeight,
        child: Center(
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Text(
              _error.toString(),
              style: TextStyle(color: Colors.red.shade400, fontSize: 12),
              textAlign: TextAlign.center,
            ),
          ),
        ),
      );
    }

    if (!_ready) {
      return SizedBox(
        height: widget.editorHeight,
        child: const Center(
          child: SizedBox(
            width: 20,
            height: 20,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
        ),
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _buildToolbar(theme),
        Divider(height: 1, color: theme.colorScheme.outline),
        SizedBox(
          height: widget.editorHeight,
          child: TiptapEditor(controller: _controller),
        ),
      ],
    );
  }

  Widget _buildToolbar(ThemeData theme) {
    final l10n = AppLocalizations.of(context)!;
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        children: [
          _toolButton(
            theme,
            icon: FontAwesomeIcons.bold,
            tooltip: l10n.bold,
            active: _markActive('bold'),
            onPressed: () => _controller.execCommand(EditorCommand.toggleBold),
          ),
          _toolButton(
            theme,
            icon: FontAwesomeIcons.italic,
            tooltip: l10n.italic,
            active: _markActive('italic'),
            onPressed: () =>
                _controller.execCommand(EditorCommand.toggleItalic),
          ),
          _toolButton(
            theme,
            icon: FontAwesomeIcons.underline,
            tooltip: l10n.underline,
            active: _markActive('underline'),
            onPressed: () => _controller.execCommand(_toggleUnderline),
          ),
          _toolButton(
            theme,
            icon: FontAwesomeIcons.listUl,
            tooltip: l10n.bullet_list,
            active: _controller.isCommandActive(EditorCommand.toggleBulletList),
            onPressed: () =>
                _controller.execCommand(EditorCommand.toggleBulletList),
          ),
          _toolButton(
            theme,
            icon: FontAwesomeIcons.listOl,
            tooltip: l10n.ordered_list,
            active: _controller.isCommandActive(
              EditorCommand.toggleOrderedList,
            ),
            onPressed: () =>
                _controller.execCommand(EditorCommand.toggleOrderedList),
          ),
          _toolButton(
            theme,
            icon: FontAwesomeIcons.quoteLeft,
            tooltip: l10n.blockquote,
            active: _controller.isCommandActive(EditorCommand.toggleBlockquote),
            onPressed: () =>
                _controller.execCommand(EditorCommand.toggleBlockquote),
          ),
          _toolButton(
            theme,
            icon: FontAwesomeIcons.link,
            tooltip: l10n.link,
            active: _markActive('link'),
            onPressed: _openLinkDialog,
          ),
        ],
      ),
    );
  }

  Widget _toolButton(
    ThemeData theme, {
    required FaIconData icon,
    required String tooltip,
    required bool active,
    required VoidCallback onPressed,
  }) {
    return IconButton(
      tooltip: tooltip,
      onPressed: onPressed,
      iconSize: 16,
      visualDensity: VisualDensity.compact,
      color: active
          ? theme.colorScheme.primary
          : theme.colorScheme.onSurface.withValues(alpha: 0.7),
      icon: FaIcon(icon),
    );
  }

  Future<void> _openLinkDialog() async {
    final bool linkActive = _markActive('link');
    final attrs = linkActive
        ? await _controller.getAttributes('link')
        : const <String, dynamic>{};
    if (!mounted) return;

    final result = await showDialog<_LinkDialogResult>(
      context: context,
      builder: (_) => _LinkDialog(
        initialHref: (attrs['href'] as String?) ?? '',
        initialNewTab: (attrs['target'] as String?) == '_blank',
        showRemove: linkActive,
      ),
    );

    if (result == null) return;

    if (result.remove) {
      await _controller.execCommand(_unsetLink);
      return;
    }

    if (result.href.isEmpty) return;
    await _controller.execCommand(_setLink, {
      'href': result.href,
      if (result.newTab) 'target': '_blank',
    });
  }
}

class _LinkDialogResult {
  final String href;
  final bool newTab;
  final bool remove;

  const _LinkDialogResult({
    required this.href,
    required this.newTab,
    this.remove = false,
  });
}

class _LinkDialog extends StatefulWidget {
  final String initialHref;
  final bool initialNewTab;
  final bool showRemove;

  const _LinkDialog({
    required this.initialHref,
    required this.initialNewTab,
    required this.showRemove,
  });

  @override
  State<_LinkDialog> createState() => _LinkDialogState();
}

class _LinkDialogState extends State<_LinkDialog> {
  late final TextEditingController _urlController;
  late bool _newTab;

  @override
  void initState() {
    super.initState();
    _urlController = TextEditingController(text: widget.initialHref);
    _newTab = widget.initialNewTab;
  }

  @override
  void dispose() {
    _urlController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final onSurface = theme.colorScheme.onSurface;
    final l10n = AppLocalizations.of(context)!;

    return AlertDialog(
      title: Text(l10n.link, style: TextStyle(color: onSurface)),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.url,
            style: theme.textTheme.bodySmall?.copyWith(
              fontWeight: FontWeight.w600,
              color: onSurface,
            ),
          ),
          const SizedBox(height: 4),
          TextField(
            controller: _urlController,
            autofocus: true,
            keyboardType: TextInputType.url,
            style: TextStyle(color: onSurface),
            decoration: InputDecoration(
              hintText: 'https://example.com',
              hintStyle: TextStyle(color: onSurface.withValues(alpha: 0.5)),
              filled: true,
              fillColor: theme.inputDecorationTheme.fillColor,
              contentPadding: const EdgeInsets.all(12),
              enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(6),
                borderSide: BorderSide(color: theme.colorScheme.outline),
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(6),
                borderSide: BorderSide(
                  color: theme.colorScheme.primary,
                  width: 1.5,
                ),
              ),
            ),
            onSubmitted: (_) => _apply(),
          ),
          CheckboxListTile(
            contentPadding: EdgeInsets.zero,
            controlAffinity: ListTileControlAffinity.leading,
            value: _newTab,
            onChanged: (v) => setState(() => _newTab = v ?? false),
            title: Text(
              l10n.open_in_new_tab,
              style: TextStyle(color: onSurface),
            ),
          ),
        ],
      ),
      actions: [
        if (widget.showRemove)
          TextButton(
            onPressed: () => Navigator.of(context).pop(
              const _LinkDialogResult(href: '', newTab: false, remove: true),
            ),
            child: Text(
              l10n.remove,
              style: TextStyle(color: Colors.red.shade400),
            ),
          ),
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(l10n.cancel),
        ),
        TextButton(onPressed: _apply, child: Text(l10n.apply)),
      ],
    );
  }

  void _apply() {
    Navigator.of(
      context,
    ).pop(_LinkDialogResult(href: _urlController.text.trim(), newTab: _newTab));
  }
}
