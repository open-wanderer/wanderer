// Extracts the flattened text of the block containing the cursor, and the
// cursor's local offset within that text, from the annotated document tree.
//
// This is a pure transformation: document tree + ProseMirror cursor position
// in, block text + local cursor offset out. It reads nothing from any widget
// State — no controller, no setState, no fields — which is why it lives here
// as standalone top-level functions rather than as methods on the editor.
// The editor calls [extractBlockText] when it needs to tell the platform's
// text-input system what text surrounds the cursor (see the editor's
// syncState path).
//
// [extractBlockTextRange] is the range-selection variant: it extracts the
// same block text but computes local offsets for both endpoints of a
// selection, so the platform can be given a real range when the selection
// fits within one block. Its per-endpoint offset computation delegates to
// the same walker the cursor path uses.
//
// This file carries the +1 position compensation. The engine's serializer
// annotates text-node pos/end values 1 higher than the actual ProseMirror
// content positions; [extractTextFromBlock] shifts them down by 1 before
// computing the local offset. Those expressions are load-bearing and were
// arrived at to fix specific cursor-placement bugs — they are reproduced here
// exactly as written, unchanged.

import '../../engine/protocol_types.dart';
import '../rendering/node_types.dart';

/// Result of extracting the text content from the block containing the cursor.
class BlockTextResult {
  /// The flattened text content of the block.
  final String text;

  /// The cursor's offset within [text].
  final int cursorOffset;

  const BlockTextResult({required this.text, required this.cursorOffset});
}

/// Result of extracting the text content of the block containing a range
/// selection's start, with the local offsets of both selection endpoints.
///
/// [extentOffset] is null when the selection's end falls outside the block
/// (a cross-block selection), which a single block's text cannot represent —
/// the caller then syncs a collapsed cursor to the platform instead.
class BlockTextRangeResult {
  /// The flattened text content of the block containing the selection start.
  final String text;

  /// The selection start's offset within [text].
  final int baseOffset;

  /// The selection end's offset within [text], or null if the selection
  /// extends beyond this block.
  final int? extentOffset;

  const BlockTextRangeResult({
    required this.text,
    required this.baseOffset,
    this.extentOffset,
  });
}

/// Walk the document tree to find the block node containing the given
/// ProseMirror position, extract its flattened text content, and compute
/// the cursor's local offset within that text.
BlockTextResult? extractBlockText(AnnotatedNode doc, int cursorPos) {
  if (doc.content == null) return null;

  /// Search top-level blocks and their nested content for the block
  /// that contains the cursor position.
  return _searchBlock(doc.content!, cursorPos);
}

/// Walk the document tree to find the block containing a range selection's
/// start position, extract its flattened text, and compute the local offsets
/// of both selection endpoints within that text.
///
/// Used to sync a range selection to the platform's text input system so
/// that soft-keyboard delete/replace operates on the selected text. The
/// per-endpoint offset computation reuses [_extractTextFromBlock], so the
/// serializer's +1 position compensation lives in exactly one place.
///
/// When [to] falls outside the block containing [from] (a cross-block
/// selection), [BlockTextRangeResult.extentOffset] is null — a single
/// block's text cannot represent the range, and the caller falls back to
/// a collapsed platform cursor.
BlockTextRangeResult? extractBlockTextRange(
  AnnotatedNode doc,
  int from,
  int to,
) {
  if (doc.content == null) return null;

  final block = _findTextBlock(doc.content!, from);
  if (block == null) return null;

  final base = _extractTextFromBlock(block, from);

  int? extentOffset;
  if (to != from && block.end != null && to <= block.end!) {
    extentOffset = _extractTextFromBlock(block, to).cursorOffset;
  }

  return BlockTextRangeResult(
    text: base.text,
    baseOffset: base.cursorOffset,
    extentOffset: extentOffset,
  );
}

/// Recursively search for the leaf block (paragraph, heading, codeBlock)
/// containing the cursor. Container blocks like blockquote and listItem
/// contain child blocks, so we recurse into them.
BlockTextResult? _searchBlock(List<AnnotatedNode> nodes, int cursorPos) {
  for (final node in nodes) {
    if (node.pos == null || node.end == null) continue;
    if (cursorPos < node.pos! || cursorPos > node.end!) continue;

    /// If this node has inline content (text nodes), it's a leaf block.
    /// Extract its text.
    if (_isTextBlock(node)) {
      return _extractTextFromBlock(node, cursorPos);
    }

    /// Otherwise it's a container block (list, blockquote, etc.).
    /// Recurse into its children.
    if (node.content != null) {
      final result = _searchBlock(node.content!, cursorPos);
      if (result != null) return result;
    }
  }
  return null;
}

/// Recursively find the leaf text block (paragraph, heading, codeBlock)
/// containing the given position, returning the node itself rather than an
/// extraction result. Mirrors the descent logic of [_searchBlock]; kept
/// separate because [extractBlockTextRange] needs the block node to run
/// two endpoint extractions against it.
AnnotatedNode? _findTextBlock(List<AnnotatedNode> nodes, int pos) {
  for (final node in nodes) {
    if (node.pos == null || node.end == null) continue;
    if (pos < node.pos! || pos > node.end!) continue;

    if (_isTextBlock(node)) return node;

    if (node.content != null) {
      final found = _findTextBlock(node.content!, pos);
      if (found != null) return found;
    }
  }
  return null;
}

/// Check if a node is a text-containing block (has inline content).
bool _isTextBlock(AnnotatedNode node) {
  const textBlockTypes = {
    NodeType.paragraph,
    NodeType.heading,
    NodeType.codeBlock,
  };
  return textBlockTypes.contains(node.type);
}

/// Extract the flattened text content from a text block and compute
/// the cursor's local offset.
///
/// The engine's serializer annotates text nodes with pos/end values that
/// are 1 higher than the actual ProseMirror content positions. To correctly
/// map a ProseMirror cursor position to a local text offset, we shift the
/// text node's pos down by 1 before computing the offset. This makes
/// the mapping consistent with ProseMirror's parentOffset semantics.
BlockTextResult _extractTextFromBlock(AnnotatedNode block, int cursorPos) {
  final buffer = StringBuffer();
  var cursorOffset = 0;
  var foundCursor = false;

  if (block.content != null) {
    for (final inline in block.content!) {
      if (inline.type == NodeType.text && inline.text != null) {
        /// Adjust the serializer's pos/end down by 1 to match actual
        /// ProseMirror positions. The serializer's pos is 1 higher than
        /// the content start position in ProseMirror's position space.
        final textPos = (inline.pos ?? 1) - 1;
        final textEnd = (inline.end ?? (textPos + inline.text!.length + 1)) - 1;

        /// Check if the cursor falls within this text node.
        if (!foundCursor && cursorPos >= textPos && cursorPos <= textEnd) {
          cursorOffset = buffer.length + (cursorPos - textPos);
          foundCursor = true;
        }

        buffer.write(inline.text);
      } else if (inline.type == NodeType.hardBreak) {
        final breakPos = (inline.pos ?? 1) - 1;
        if (!foundCursor && cursorPos <= breakPos) {
          cursorOffset = buffer.length;
          foundCursor = true;
        }
        buffer.write('\n');
      }
    }
  }

  if (!foundCursor) {
    cursorOffset = buffer.length;
  }

  final text = buffer.toString();
  return BlockTextResult(
    text: text,
    cursorOffset: cursorOffset.clamp(0, text.length),
  );
}
