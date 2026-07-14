// Selection and command-state protocol types.
//
// These types describe the engine's view of the current selection and the
// per-command state it reports on every transaction: [SelectionState] (the
// cursor or range), [CommandState] (whether each command can run and is
// active), and [ActiveNode] (the node types under the selection, with attrs).
//
// This file is one of the concern-grouped pieces that together form the
// protocol-types surface; protocol_types.dart re-exports it so existing
// imports of that path keep resolving unchanged.
//
// Selection and command-state fields are read through [ProtocolKey] constants.
// [ActiveNode], being a node shape, reads its `type` and `attrs` through
// [NodeKey] for the same reason the document-tree types do.

import 'protocol_constants.dart';

/// Represents the current selection state in the ProseMirror document.
///
/// ProseMirror selections have an [anchor] (where the user started selecting)
/// and a [head] (where the selection ends). The [from] and [to] fields are
/// the normalized (min/max) versions — [from] is always <= [to].
/// [empty] is true when anchor == head (i.e., a cursor with no range).
///
/// Selection types from the engine:
///   - "text": a standard text selection (cursor or range)
///   - "node": an entire node is selected (e.g., an image)
///   - "all": the entire document is selected
///   - "gapcursor": a cursor in a position between nodes where text can't exist
class SelectionState {
  /// The selection type (e.g., "text", "node", "all", "gapcursor").
  final String? type;

  /// The anchor position — where the selection was initiated.
  final int anchor;

  /// The head position — where the selection currently ends.
  final int head;

  /// The lower bound of the selection range (min of anchor, head).
  final int from;

  /// The upper bound of the selection range (max of anchor, head).
  final int to;

  /// Whether the selection is collapsed (cursor only, no range).
  final bool empty;

  const SelectionState({
    this.type,
    required this.anchor,
    required this.head,
    required this.from,
    required this.to,
    required this.empty,
  });

  factory SelectionState.fromJson(Map<String, dynamic> json) {
    return SelectionState(
      type: json[ProtocolKey.selectionType] as String?,
      anchor: json[ProtocolKey.anchor] as int,
      head: json[ProtocolKey.head] as int,
      from: json[ProtocolKey.from] as int,
      to: json[ProtocolKey.to] as int,
      empty: json[ProtocolKey.empty] as bool,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      if (type != null) ProtocolKey.selectionType: type,
      ProtocolKey.anchor: anchor,
      ProtocolKey.head: head,
      ProtocolKey.from: from,
      ProtocolKey.to: to,
      ProtocolKey.empty: empty,
    };
  }

  @override
  String toString() =>
      'SelectionState(type: $type, anchor: $anchor, head: $head, '
      'from: $from, to: $to, empty: $empty)';
}

/// Represents the state of a single editor command.
///
/// The engine reports this for every registered command on each state change,
/// allowing the UI to enable/disable toolbar buttons and show active states
/// (e.g., the bold button appears pressed when the cursor is inside bold text).
class CommandState {
  /// Whether the command can currently be executed given the document state.
  final bool canExec;

  /// Whether the command's associated mark or node is active at the current selection.
  final bool isActive;

  /// The nesting depth, relevant for undo/redo commands. Indicates how many
  /// steps are available in the history stack.
  final int? depth;

  const CommandState({
    required this.canExec,
    required this.isActive,
    this.depth,
  });

  factory CommandState.fromJson(Map<String, dynamic> json) {
    return CommandState(
      canExec: json[ProtocolKey.canExec] as bool? ?? false,
      isActive: json[ProtocolKey.isActive] as bool? ?? false,
      depth: json[ProtocolKey.depth] as int?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      ProtocolKey.canExec: canExec,
      ProtocolKey.isActive: isActive,
      if (depth != null) ProtocolKey.depth: depth,
    };
  }

  @override
  String toString() =>
      'CommandState(canExec: $canExec, isActive: $isActive, depth: $depth)';
}

/// Represents a node type that is active at the current selection.
///
/// The engine reports this as part of the stateChanged event. For example,
/// when the cursor is inside a heading, `activeNodes` will contain an entry
/// with type "heading" and attrs { "level": 2 }. This enables the toolbar
/// to show which block type is currently active.
class ActiveNode {
  /// The node type name (e.g., "paragraph", "heading", "blockquote").
  final String type;

  /// The node's attributes at this position (e.g., { "level": 2 } for headings).
  final Map<String, dynamic> attrs;

  const ActiveNode({required this.type, this.attrs = const {}});

  factory ActiveNode.fromJson(Map<String, dynamic> json) {
    return ActiveNode(
      type: json[NodeKey.type] as String,
      attrs: json[NodeKey.attrs] as Map<String, dynamic>? ?? const {},
    );
  }

  Map<String, dynamic> toJson() {
    return {NodeKey.type: type, NodeKey.attrs: attrs};
  }

  @override
  String toString() => 'ActiveNode(type: $type, attrs: $attrs)';
}
