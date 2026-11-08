// Schema-introspection protocol types.
//
// The engine emits these once during initialization, in the schemaReady
// event, to describe the editor's node types, mark types, and available
// commands. They let the port discover what the editor supports without
// hardcoding extension knowledge: [SchemaAttrInfo] (an attribute descriptor),
// [SchemaNodeInfo] / [SchemaMarkInfo] (node and mark type descriptors),
// [SchemaCommandInfo] / [SchemaCommandArg] (command and argument descriptors),
// and [SchemaMetadata] (the whole payload, with lookup helpers).
//
// This file is one of the concern-grouped pieces that together form the
// protocol-types surface; protocol_types.dart re-exports it so existing
// imports of that path keep resolving unchanged.
//
// JSON keys for the schema-descriptor shape are read through [SchemaKey]
// constants rather than [ProtocolKey]: the descriptors' `name` field is a
// node/mark/command name, not the message envelope's command-or-event name,
// and most descriptor fields have no [ProtocolKey] counterpart at all.

import 'protocol_constants.dart';

/// Metadata about a single attribute on a node or mark type.
///
/// Emitted as part of schemaReady to describe the configurable attributes
/// of each node and mark type in the schema.
class SchemaAttrInfo {
  /// The attribute name (e.g., "level", "href", "src").
  final String name;

  /// The default value for this attribute, or null if no default is specified.
  final dynamic defaultValue;

  const SchemaAttrInfo({required this.name, this.defaultValue});

  factory SchemaAttrInfo.fromJson(Map<String, dynamic> json) {
    return SchemaAttrInfo(
      name: json[SchemaKey.name] as String,
      defaultValue: json[SchemaKey.defaultValue],
    );
  }

  Map<String, dynamic> toJson() {
    return {SchemaKey.name: name, SchemaKey.defaultValue: defaultValue};
  }

  @override
  String toString() => 'SchemaAttrInfo(name: $name, default: $defaultValue)';
}

/// Metadata about a node type in the ProseMirror schema.
///
/// Emitted once during initialization as part of the schemaReady event.
/// Describes the structural properties of each node type, enabling the
/// port to dynamically discover what the editor supports.
class SchemaNodeInfo {
  /// The node type name (e.g., "paragraph", "heading", "image").
  final String name;

  /// The ProseMirror content expression defining what children this node
  /// can contain (e.g., "inline*", "block+", "text*").
  final String? contentExpression;

  /// The group this node type belongs to (e.g., "block", "inline").
  final String? group;

  /// The attribute definitions for this node type.
  final List<SchemaAttrInfo> attrs;

  /// Whether this is a leaf node (has no editable content).
  final bool isLeaf;

  /// Whether this is an inline node.
  final bool isInline;

  /// Whether this is a block-level node.
  final bool isBlock;

  const SchemaNodeInfo({
    required this.name,
    this.contentExpression,
    this.group,
    this.attrs = const [],
    this.isLeaf = false,
    this.isInline = false,
    this.isBlock = false,
  });

  factory SchemaNodeInfo.fromJson(Map<String, dynamic> json) {
    return SchemaNodeInfo(
      name: json[SchemaKey.name] as String,
      contentExpression: json[SchemaKey.contentExpression] as String?,
      group: json[SchemaKey.group] as String?,
      attrs:
          (json[SchemaKey.attrs] as List<dynamic>?)
              ?.map(
                (item) => SchemaAttrInfo.fromJson(item as Map<String, dynamic>),
              )
              .toList() ??
          const [],
      isLeaf: json[SchemaKey.isLeaf] as bool? ?? false,
      isInline: json[SchemaKey.isInline] as bool? ?? false,
      isBlock: json[SchemaKey.isBlock] as bool? ?? false,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      SchemaKey.name: name,
      if (contentExpression != null)
        SchemaKey.contentExpression: contentExpression,
      if (group != null) SchemaKey.group: group,
      SchemaKey.attrs: attrs.map((a) => a.toJson()).toList(),
      SchemaKey.isLeaf: isLeaf,
      SchemaKey.isInline: isInline,
      SchemaKey.isBlock: isBlock,
    };
  }

  @override
  String toString() => 'SchemaNodeInfo(name: $name, group: $group)';
}

/// Metadata about a mark type in the ProseMirror schema.
///
/// Emitted as part of schemaReady. Marks are inline annotations applied
/// to text (bold, italic, link, etc.).
class SchemaMarkInfo {
  /// The mark type name (e.g., "bold", "italic", "link").
  final String name;

  /// The attribute definitions for this mark type.
  /// Simple marks like bold have no attributes.
  /// Parameterized marks like link have attributes such as href and target.
  final List<SchemaAttrInfo> attrs;

  const SchemaMarkInfo({required this.name, this.attrs = const []});

  factory SchemaMarkInfo.fromJson(Map<String, dynamic> json) {
    return SchemaMarkInfo(
      name: json[SchemaKey.name] as String,
      attrs:
          (json[SchemaKey.attrs] as List<dynamic>?)
              ?.map(
                (item) => SchemaAttrInfo.fromJson(item as Map<String, dynamic>),
              )
              .toList() ??
          const [],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      SchemaKey.name: name,
      SchemaKey.attrs: attrs.map((a) => a.toJson()).toList(),
    };
  }

  @override
  String toString() => 'SchemaMarkInfo(name: $name, attrs: ${attrs.length})';
}

/// Metadata about a command available on the editor.
///
/// Emitted as part of schemaReady. Each command entry describes the command's
/// purpose, what type of action it performs, and what arguments it accepts.
/// This enables the port to dynamically build toolbars and menus without
/// hardcoding knowledge of specific extensions.
///
/// Command types:
///   - "toggle-mark": toggles an inline mark on/off (e.g., toggleBold)
///   - "toggle-node": toggles a block node type (e.g., toggleBlockquote)
///   - "set-node": sets a block node type without toggle (e.g., setHeading)
///   - "wrap": wraps selection in a node (e.g., wrapInBlockquote)
///   - "lift": lifts content out of a wrapping node
///   - "action": one-shot action (e.g., undo, insertTable)
class SchemaCommandInfo {
  /// The command name as used in exec (e.g., "toggleBold", "setHeading").
  final String name;

  /// The command type indicating what kind of action this performs.
  final String? type;

  /// The associated node or mark type name (e.g., "bold" for toggleBold,
  /// "heading" for setHeading).
  final String? associatedType;

  /// The arguments this command accepts, with their names and whether
  /// they are required.
  final List<SchemaCommandArg> args;

  /// The logical group this command belongs to (e.g., "formatting", "blocks",
  /// "lists", "tables", "history").
  final String? group;

  /// The name of the Tiptap extension that provides this command.
  final String? extensionName;

  const SchemaCommandInfo({
    required this.name,
    this.type,
    this.associatedType,
    this.args = const [],
    this.group,
    this.extensionName,
  });

  factory SchemaCommandInfo.fromJson(Map<String, dynamic> json) {
    return SchemaCommandInfo(
      name: json[SchemaKey.name] as String,
      type: json[SchemaKey.commandType] as String?,
      associatedType: json[SchemaKey.associatedType] as String?,
      args:
          (json[SchemaKey.args] as List<dynamic>?)
              ?.map(
                (item) =>
                    SchemaCommandArg.fromJson(item as Map<String, dynamic>),
              )
              .toList() ??
          const [],
      group: json[SchemaKey.group] as String?,
      extensionName: json[SchemaKey.extensionName] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      SchemaKey.name: name,
      if (type != null) SchemaKey.commandType: type,
      if (associatedType != null) SchemaKey.associatedType: associatedType,
      SchemaKey.args: args.map((a) => a.toJson()).toList(),
      if (group != null) SchemaKey.group: group,
      if (extensionName != null) SchemaKey.extensionName: extensionName,
    };
  }

  @override
  String toString() =>
      'SchemaCommandInfo(name: $name, type: $type, group: $group)';
}

/// Describes a single argument accepted by a command.
class SchemaCommandArg {
  /// The argument name (e.g., "level", "rows", "cols").
  final String name;

  /// Whether this argument is required for the command to execute.
  final bool required;

  const SchemaCommandArg({required this.name, this.required = false});

  factory SchemaCommandArg.fromJson(Map<String, dynamic> json) {
    return SchemaCommandArg(
      name: json[SchemaKey.name] as String,
      required: json[SchemaKey.required] as bool? ?? false,
    );
  }

  Map<String, dynamic> toJson() {
    return {SchemaKey.name: name, SchemaKey.required: required};
  }

  @override
  String toString() => 'SchemaCommandArg(name: $name, required: $required)';
}

/// Full schema metadata emitted by the engine during initialization.
///
/// Contains information about all registered node types, mark types,
/// and available commands. This is used to build dynamic toolbars and
/// to understand the document structure without hardcoded assumptions.
///
/// The engine sends arrays for nodes, marks, and commands (not maps).
/// Each entry is a self-describing object with a "name" field.
class SchemaMetadata {
  /// All registered node types (paragraph, heading, image, etc.).
  final List<SchemaNodeInfo> nodes;

  /// All registered mark types (bold, italic, link, etc.).
  final List<SchemaMarkInfo> marks;

  /// All available commands discovered from the editor instance.
  final List<SchemaCommandInfo> commands;

  const SchemaMetadata({
    required this.nodes,
    required this.marks,
    required this.commands,
  });

  factory SchemaMetadata.fromJson(Map<String, dynamic> json) {
    final nodesJson = json[SchemaKey.nodes] as List<dynamic>? ?? [];
    final marksJson = json[SchemaKey.marks] as List<dynamic>? ?? [];
    final commandsJson = json[SchemaKey.commands] as List<dynamic>? ?? [];

    return SchemaMetadata(
      nodes: nodesJson
          .map((item) => SchemaNodeInfo.fromJson(item as Map<String, dynamic>))
          .toList(),
      marks: marksJson
          .map((item) => SchemaMarkInfo.fromJson(item as Map<String, dynamic>))
          .toList(),
      commands: commandsJson
          .map(
            (item) => SchemaCommandInfo.fromJson(item as Map<String, dynamic>),
          )
          .toList(),
    );
  }

  /// Look up a node type by name. Returns null if not found.
  SchemaNodeInfo? findNode(String name) {
    for (final node in nodes) {
      if (node.name == name) return node;
    }
    return null;
  }

  /// Look up a mark type by name. Returns null if not found.
  SchemaMarkInfo? findMark(String name) {
    for (final mark in marks) {
      if (mark.name == name) return mark;
    }
    return null;
  }

  /// Look up a command by name. Returns null if not found.
  SchemaCommandInfo? findCommand(String name) {
    for (final command in commands) {
      if (command.name == name) return command;
    }
    return null;
  }

  @override
  String toString() =>
      'SchemaMetadata(nodes: ${nodes.length}, marks: ${marks.length}, '
      'commands: ${commands.length})';
}
