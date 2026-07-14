// Document-tree protocol types: marks and annotated nodes.
//
// These are the building blocks of the annotated document tree the engine
// emits. A [MarkData] is an inline annotation on a text node; an
// [AnnotatedNode] is any node in the tree, carrying the position offsets that
// let the port map between pixel coordinates and ProseMirror positions.
//
// This file is one of the concern-grouped pieces that together form the
// protocol-types surface; protocol_types.dart re-exports it so existing
// imports of that path keep resolving unchanged.
//
// JSON keys for the node/mark shape are read and written through [NodeKey]
// constants rather than [ProtocolKey]. The two namespaces share some wire
// strings (notably `type`, and also `content`/`text`/`attrs`) but mean
// different things; [NodeKey] is the set that names the serialized node and
// mark fields, so every field of a node is read through one consistent
// namespace.

import 'protocol_constants.dart';

/// Represents a mark applied to a text node (e.g., bold, italic, link).
///
/// Marks carry a [type] identifier and an optional [attrs] map for
/// parameterized marks like links (which have an href attribute).
class MarkData {
  final String type;
  final Map<String, dynamic>? attrs;

  const MarkData({required this.type, this.attrs});

  factory MarkData.fromJson(Map<String, dynamic> json) {
    return MarkData(
      type: json[NodeKey.type] as String,
      attrs: json[NodeKey.attrs] as Map<String, dynamic>?,
    );
  }

  Map<String, dynamic> toJson() {
    return {NodeKey.type: type, if (attrs != null) NodeKey.attrs: attrs};
  }

  @override
  String toString() => 'MarkData(type: $type, attrs: $attrs)';
}

/// Represents a node in the annotated document tree emitted by the engine.
///
/// Every node carries [pos] and [end] position offsets that map the node's
/// location in the ProseMirror document model. These offsets are critical
/// for mapping pixel coordinates (from tap events) back to document positions
/// for selection and editing.
///
/// Position rules (from the engine API):
///   - For block nodes: [pos] is after the opening token, [end] is after the
///     closing token.
///   - For text nodes: [pos] is the first character, [end] - [pos] equals the
///     text length.
///   - The document node starts at pos: 0.
///   - A child's [pos] equals its parent's pos + 1 (for the first child of a
///     block node).
///
/// The [content] list holds child nodes, forming a tree. Leaf text nodes
/// have a [text] field instead of children. Marks on text nodes appear in
/// the [marks] list.
class AnnotatedNode {
  /// The ProseMirror node type identifier (e.g., "paragraph", "heading", "text").
  final String type;

  /// The resolved start position of this node in the document.
  final int? pos;

  /// The resolved end position of this node in the document.
  final int? end;

  /// Child nodes. Empty for leaf nodes like text.
  final List<AnnotatedNode>? content;

  /// The text content, present only on text nodes.
  final String? text;

  /// Marks applied to this node (bold, italic, link, etc.).
  final List<MarkData>? marks;

  /// Node-specific attributes (e.g., level for headings, src for images).
  final Map<String, dynamic>? attrs;

  const AnnotatedNode({
    required this.type,
    this.pos,
    this.end,
    this.content,
    this.text,
    this.marks,
    this.attrs,
  });

  factory AnnotatedNode.fromJson(Map<String, dynamic> json) {
    return AnnotatedNode(
      type: json[NodeKey.type] as String,
      pos: json[NodeKey.pos] as int?,
      end: json[NodeKey.end] as int?,
      content: (json[NodeKey.content] as List<dynamic>?)
          ?.map((item) => AnnotatedNode.fromJson(item as Map<String, dynamic>))
          .toList(),
      text: json[NodeKey.text] as String?,
      marks: (json[NodeKey.marks] as List<dynamic>?)
          ?.map((item) => MarkData.fromJson(item as Map<String, dynamic>))
          .toList(),
      attrs: json[NodeKey.attrs] as Map<String, dynamic>?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      NodeKey.type: type,
      if (pos != null) NodeKey.pos: pos,
      if (end != null) NodeKey.end: end,
      if (content != null)
        NodeKey.content: content!.map((node) => node.toJson()).toList(),
      if (text != null) NodeKey.text: text,
      if (marks != null)
        NodeKey.marks: marks!.map((mark) => mark.toJson()).toList(),
      if (attrs != null) NodeKey.attrs: attrs,
    };
  }

  @override
  String toString() => 'AnnotatedNode(type: $type, pos: $pos, end: $end)';
}
