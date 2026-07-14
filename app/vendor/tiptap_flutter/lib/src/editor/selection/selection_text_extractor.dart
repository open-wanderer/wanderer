// Extracts the plain text of a ProseMirror position range from the annotated
// document tree, for clipboard Copy and Cut.
//
// This is a pure transformation: document tree + [from, to] range in, plain
// text out. It runs entirely on the port side — the annotated tree already
// contains every text character with its positions, so no engine round-trip
// is needed for plain-text copy.
//
// Like block_text_extractor.dart, this file carries the +1 position
// compensation: the engine's serializer annotates text-node pos/end values
// 1 higher than the actual ProseMirror content positions, so each text
// node's annotated pos is shifted down by 1 before slicing. The expressions
// match the convention established in block_text_extractor exactly.
//
// Serialization choices:
//   - hardBreak nodes inside the range become '\n'.
//   - Block boundaries between consecutive text blocks in the range become
//     a single '\n'. (ProseMirror's own textBetween commonly uses a
//     configurable block separator; '\n' matches how the renderer and the
//     input layer already treat hard breaks, keeping the port's text view
//     of the document internally consistent.)
//   - Empty blocks inside the range contribute an empty line, so copying
//     across a blank paragraph preserves the blank line.

import '../../engine/protocol_types.dart';
import '../rendering/node_types.dart';

/// Extract the plain text content of the ProseMirror range [from, to)
/// from the annotated document tree.
///
/// Returns the empty string for collapsed or inverted ranges.
String extractTextInRange(AnnotatedNode doc, int from, int to) {
  if (doc.content == null || to <= from) return '';

  /// Each intersecting leaf text block contributes one entry (possibly
  /// empty, for blank paragraphs inside the range); entries are joined
  /// with a newline to represent the block boundary.
  final blockTexts = <String>[];
  _collectBlockTexts(doc.content!, from, to, blockTexts);
  return blockTexts.join('\n');
}

/// Recursively walk the tree in document order, slicing every leaf text
/// block that intersects the range. Container blocks (lists, blockquotes)
/// are descended into; their own tokens contribute no text.
void _collectBlockTexts(
  List<AnnotatedNode> nodes,
  int from,
  int to,
  List<String> out,
) {
  for (final node in nodes) {
    if (node.pos == null || node.end == null) continue;

    /// Skip nodes entirely outside the range.
    if (node.end! < from || node.pos! > to) continue;

    if (_isTextBlock(node)) {
      out.add(_sliceBlockText(node, from, to));
      continue;
    }

    if (node.content != null) {
      _collectBlockTexts(node.content!, from, to, out);
    }
  }
}

/// Check if a node is a text-containing block (has inline content).
/// Matches the set used by block_text_extractor.
bool _isTextBlock(AnnotatedNode node) {
  const textBlockTypes = {
    NodeType.paragraph,
    NodeType.heading,
    NodeType.codeBlock,
  };
  return textBlockTypes.contains(node.type);
}

/// Slice the portion of a text block's flattened text that falls within
/// the [from, to) range.
///
/// Text node positions are adjusted down by 1 to compensate for the
/// serializer's +1 annotation offset, matching the convention in
/// block_text_extractor. Each text node contributes the substring of its
/// text that overlaps the range; hard breaks inside the range contribute
/// a newline.
String _sliceBlockText(AnnotatedNode block, int from, int to) {
  if (block.content == null) return '';

  final buffer = StringBuffer();

  for (final inline in block.content!) {
    if (inline.type == NodeType.text && inline.text != null) {
      /// Adjust the serializer's pos down by 1 to the actual ProseMirror
      /// position of the text node's first character.
      final textPos = (inline.pos ?? 1) - 1;
      final length = inline.text!.length;

      /// Compute the overlap of [from, to) with this text node's character
      /// range and slice the corresponding substring.
      final sliceStart = (from - textPos).clamp(0, length);
      final sliceEnd = (to - textPos).clamp(0, length);
      if (sliceEnd > sliceStart) {
        buffer.write(inline.text!.substring(sliceStart, sliceEnd));
      }
    } else if (inline.type == NodeType.hardBreak) {
      /// A hard break occupies one position; include its newline when that
      /// position falls inside the range.
      final breakPos = (inline.pos ?? 1) - 1;
      if (breakPos >= from && breakPos < to) {
        buffer.write('\n');
      }
    }
  }

  return buffer.toString();
}
