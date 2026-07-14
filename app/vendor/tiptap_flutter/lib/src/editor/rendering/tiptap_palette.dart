// Vendored patch — not part of the upstream `tiptap_flutter` package
// (pub.dev, v0.1.1, MIT license, https://github.com/blackcoffee2/tiptap-flutter).
//
// Upstream hardcodes every rendering color as a literal `Color(0x...)` and
// exposes no theming hook: `TiptapEditor` never accepts a custom
// `NodeRendererRegistry`, and `PositionRegistry`/`RegisteredBlock` (the types a
// custom node builder would need) are not exported from the package, so even
// registering builders through the public registry API would require
// reaching into private `src/` internals. This file introduces a single,
// mutable color bridge that the app can set from its own Theme before each
// build, letting the vendored builders below stay theme-aware without a
// BuildContext threaded through their signatures.
//
// Defaults below match the original hardcoded literals, so behavior is
// unchanged until a consumer app assigns these fields (see
// `WandererRichTextEditor` in the wanderer app, which sets them from
// `Theme.of(context)` on every build).
//
// Re-vendoring a future upstream release: diff the new release's rendering
// files against this vendored copy and re-apply the same literal → field swaps
// documented at each call site.

import 'package:flutter/material.dart';

class TiptapPalette {
  TiptapPalette._();

  /// Body text (paragraphs, headings, list items).
  static Color text = const Color(0xFF1F1F1F);

  /// Secondary/caption text (image captions, code block language label).
  static Color muted = const Color(0xFF9E9E9E);

  /// Link mark color (text + underline).
  static Color link = const Color(0xFF1A73E8);

  /// Text cursor (caret) and selection highlight. `EditorSelectionOverlay`
  /// defaults these to the same fixed blue as [link] internally; the app
  /// passes these explicitly instead of relying on that default.
  static Color cursor = const Color(0xFF1A73E8);
  static Color selectionHighlight = const Color(0x401A73E8);

  /// Inline code mark background.
  static Color codeBackground = const Color(0x1A000000);

  /// Code block background and text.
  static Color codeBlockBackground = const Color(0xFFF5F5F5);
  static Color codeBlockText = const Color(0xFF37474F);

  /// Blockquote left border.
  static Color blockquoteBorder = const Color(0xFFBDBDBD);

  /// Horizontal rule.
  static Color divider = const Color(0xFFE0E0E0);

  /// Image placeholder (missing/failed src) background and caption text.
  static Color imagePlaceholderBackground = const Color(0xFFF5F5F5);
  static Color imagePlaceholderText = const Color(0xFF9E9E9E);

  /// Debug placeholder shown for a node type the engine sent but this
  /// renderer doesn't recognize — not expected in normal use.
  static Color unknownNodeBackground = Colors.grey.shade200;
  static Color unknownNodeBorder = Colors.grey.shade400;
  static Color unknownNodeText = Colors.grey.shade600;
}
