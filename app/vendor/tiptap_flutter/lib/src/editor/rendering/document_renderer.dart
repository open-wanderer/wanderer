// Recursive document renderer that walks the [AnnotatedNode] tree and
// produces Flutter widgets for each node.
//
// This is the entry point for document rendering. It dispatches each node
// to its registered builder in the [NodeRendererRegistry], falling back to
// a debug placeholder for unknown node types.
//
// The renderer registers all standard Tiptap node builders on first use.
// Extension developers can add custom builders to the registry before
// the renderer is instantiated, or override the default ones.
//
// Each text-rendering block (paragraph, heading) registers itself with
// the [PositionRegistry] so that taps can be mapped to document positions
// and cursors can be painted at the correct pixel locations.
//
// This file is split across three parts that together form the
// `document_renderer` library:
//
//   - This file: the DocumentRenderer widget, node dispatch, the unknown-node
//     placeholder, the default-builder registration, and the shared base text
//     style and link-tap handler used by the builders.
//   - node_builders.dart: the block-node builders (paragraph, heading, lists,
//     blockquote, code block, horizontal rule) and the shared
//     _buildRichTextBlock helper and _ListItemWrapper widget.
//   - image_builders.dart: the image node builder and its helpers for network
//     and base64 sources, plus the error placeholder.
//
// The builders are top-level library-private functions (prefixed with `_`).
// They are split out via `part` rather than into a standalone class so they
// keep their library privacy: the registry seam means the default builders are
// an internal implementation detail, not public API. Using `part` lets them
// live in separate files while remaining visible to this file's registration
// code and invisible outside the library.

import 'dart:convert';

import 'package:flutter/material.dart';

import '../../engine/protocol_types.dart';
import '../selection/position_registry.dart';
import 'node_renderer_registry.dart';
import 'node_types.dart';
import 'tiptap_palette.dart';
import 'text_span_builder.dart';

part 'node_builders.dart';
part 'image_builders.dart';

/// Widget that renders an entire annotated document tree.
///
/// Takes the root [AnnotatedNode] (always type "doc") and recursively
/// builds the widget tree for all descendants.
///
/// The [positionRegistry] is populated during build with entries for each
/// text-rendering block, enabling tap-to-cursor and cursor painting.
class DocumentRenderer extends StatefulWidget {
  /// The root document node from the engine's stateChanged event.
  final AnnotatedNode doc;

  /// The position registry to populate with block entries.
  /// If null, position tracking is disabled (read-only mode without cursor).
  final PositionRegistry? positionRegistry;

  /// The renderer registry to use. Defaults to the global default registry,
  /// which includes all standard node type builders.
  final NodeRendererRegistry? registry;

  const DocumentRenderer({
    super.key,
    required this.doc,
    this.positionRegistry,
    this.registry,
  });

  @override
  State<DocumentRenderer> createState() => _DocumentRendererState();
}

class _DocumentRendererState extends State<DocumentRenderer> {
  @override
  Widget build(BuildContext context) {
    final reg = widget.registry ?? NodeRendererRegistry.defaultRegistry;

    /// Register default builders if the registry is empty.
    if (!reg.hasBuilder(NodeType.paragraph)) {
      _registerDefaultBuilders(reg);
    }

    /// Clear the position registry before rebuilding so stale entries
    /// from previous renders don't persist.
    widget.positionRegistry?.clear();

    /// The doc node's children are the top-level block nodes.
    final children = widget.doc.content ?? [];
    if (children.isEmpty) {
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [for (final child in children) _buildNode(context, child, reg)],
    );
  }

  /// Build a widget for a single node, dispatching to the registry.
  Widget _buildNode(
    BuildContext context,
    AnnotatedNode node,
    NodeRendererRegistry reg,
  ) {
    final builder = reg.builderFor(node.type);
    if (builder != null) {
      return builder(
        node,
        (child) => _buildNode(context, child, reg),
        widget.positionRegistry,
      );
    }

    /// Unknown node type — render a debug placeholder.
    return _UnknownNodePlaceholder(node: node);
  }
}

/// Debug placeholder widget for unrecognized node types.
class _UnknownNodePlaceholder extends StatelessWidget {
  final AnnotatedNode node;

  const _UnknownNodePlaceholder({required this.node});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 4),
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: TiptapPalette.unknownNodeBackground,
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: TiptapPalette.unknownNodeBorder),
      ),
      child: Text(
        'Unknown node: ${node.type}',
        style: TextStyle(
          color: TiptapPalette.unknownNodeText,
          fontStyle: FontStyle.italic,
          fontSize: 12,
        ),
      ),
    );
  }
}

// =============================================================================
// Default node builders
// =============================================================================

/// Register all standard Tiptap node type builders with the registry.
///
/// The set registered here matches the engine's fixed extension set:
/// StarterKit plus the Image node. Each block node that can appear in the
/// document tree has a hand-written builder. The builders are registered
/// through the [NodeRendererRegistry] rather than collapsed into a single
/// switch, preserving an extension seam: if the engine ever regains dynamic
/// extension loading, app-supplied custom builders can be added to the
/// registry without restructuring this code.
void _registerDefaultBuilders(NodeRendererRegistry registry) {
  registry.register(NodeType.paragraph, _buildParagraph);
  registry.register(NodeType.heading, _buildHeading);
  registry.register(NodeType.bulletList, _buildBulletList);
  registry.register(NodeType.orderedList, _buildOrderedList);
  registry.register(NodeType.listItem, _buildListItem);
  registry.register(NodeType.blockquote, _buildBlockquote);
  registry.register(NodeType.codeBlock, _buildCodeBlock);
  registry.register(NodeType.horizontalRule, _buildHorizontalRule);
  registry.register(NodeType.image, _buildImage);
}

/// The default base text style used for body text.
TextStyle get _baseTextStyle =>
    TextStyle(fontSize: 16, height: 1.6, color: TiptapPalette.text);

/// Handle link taps. For now, prints the URL to console.
/// In a production app, this would use url_launcher or a custom callback.
void _onLinkTap(String url) {
  // ignore: avoid_print
  print('[TiptapEditor] Link tapped: $url');
}
