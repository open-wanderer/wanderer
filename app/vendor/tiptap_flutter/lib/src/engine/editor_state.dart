// Editor-state, error, and extension-event protocol types.
//
// [EditorStatePayload] is the full state the engine emits on every
// transaction — the primary structure flowing from engine to port, carrying
// the document tree, selection, active marks/nodes, and per-command states.
// [ErrorPayload] is the structured error shape used by both error responses
// and error events. [ExtensionEvent] is the passthrough envelope extensions
// use to talk to the port.
//
// This file is one of the concern-grouped pieces that together form the
// protocol-types surface; protocol_types.dart re-exports it so existing
// imports of that path keep resolving unchanged. It depends on the other
// type files for the nested shapes it parses (annotated nodes, selection,
// command state, active node, marks).
//
// State-payload, error, and extension-event fields are read through
// [ProtocolKey] constants.

import 'protocol_constants.dart';
import 'marks_and_nodes.dart';
import 'selection_and_commands.dart';

/// The full state payload emitted by the engine on every transaction.
///
/// This is the primary data structure that flows from the engine to the port
/// on every state change. It contains everything needed to render the document
/// and update the toolbar UI.
class EditorStatePayload {
  /// The annotated document tree with position offsets on every node.
  final AnnotatedNode? doc;

  /// The current selection state.
  final SelectionState? selection;

  /// List of mark type names that are active at the current selection.
  final List<String> activeMarks;

  /// List of node types that are active at the current selection,
  /// with their attributes. For example, when inside a level-2 heading,
  /// this contains ActiveNode(type: "heading", attrs: { "level": 2 }).
  final List<ActiveNode> activeNodes;

  /// Map of command names to their current states (canExec, isActive).
  final Map<String, CommandState> commandStates;

  /// Active decorations in the document. The structure depends on the
  /// extensions that provide decorations. Stored as raw JSON for now
  /// since decoration types are extension-specific.
  final List<dynamic> decorations;

  /// Marks that have been stored via storedMarks (e.g., when the user
  /// toggles bold with an empty selection, the mark is "stored" and will
  /// be applied to the next typed character).
  final List<MarkData> storedMarks;

  /// Whether the editor is currently in editable mode.
  final bool editable;

  const EditorStatePayload({
    this.doc,
    this.selection,
    this.activeMarks = const [],
    this.activeNodes = const [],
    this.commandStates = const {},
    this.decorations = const [],
    this.storedMarks = const [],
    this.editable = true,
  });

  factory EditorStatePayload.fromJson(Map<String, dynamic> json) {
    final docJson = json[ProtocolKey.doc] as Map<String, dynamic>?;
    final selectionJson = json[ProtocolKey.selection] as Map<String, dynamic>?;
    final activeMarksJson =
        json[ProtocolKey.activeMarks] as List<dynamic>? ?? [];
    final activeNodesJson =
        json[ProtocolKey.activeNodes] as List<dynamic>? ?? [];
    final commandStatesJson =
        json[ProtocolKey.commandStates] as Map<String, dynamic>? ?? {};
    final decorationsJson =
        json[ProtocolKey.decorations] as List<dynamic>? ?? [];
    final storedMarksJson =
        json[ProtocolKey.storedMarks] as List<dynamic>? ?? [];

    return EditorStatePayload(
      doc: docJson != null ? AnnotatedNode.fromJson(docJson) : null,
      selection: selectionJson != null
          ? SelectionState.fromJson(selectionJson)
          : null,
      activeMarks: activeMarksJson.cast<String>(),
      activeNodes: activeNodesJson
          .map((item) => ActiveNode.fromJson(item as Map<String, dynamic>))
          .toList(),
      commandStates: commandStatesJson.map(
        (key, value) =>
            MapEntry(key, CommandState.fromJson(value as Map<String, dynamic>)),
      ),
      decorations: decorationsJson,
      storedMarks: storedMarksJson
          .map((item) => MarkData.fromJson(item as Map<String, dynamic>))
          .toList(),
      editable: json[ProtocolKey.editable] as bool? ?? true,
    );
  }

  @override
  String toString() =>
      'EditorStatePayload(doc: ${doc?.type}, selection: $selection, '
      'activeMarks: $activeMarks, activeNodes: ${activeNodes.length}, '
      'commands: ${commandStates.length}, editable: $editable)';
}

/// Structured error information returned by the engine.
///
/// Used both in error responses (when a command fails) and in error events
/// (when the engine encounters an asynchronous error).
///
/// Error codes:
///   - NOT_INITIALIZED: command sent before init or after destroy
///   - ALREADY_INITIALIZED: init sent when engine is already running
///   - UNKNOWN_COMMAND: unrecognized command name
///   - UNKNOWN_EXEC_COMMAND: the command passed to exec doesn't exist
///   - INVALID_FORMAT: unknown format in getContent
///   - COMMAND_FAILED: the command threw an exception during execution
class ErrorPayload {
  /// A machine-readable error code (e.g., "NOT_INITIALIZED", "COMMAND_FAILED").
  final String code;

  /// A human-readable error description.
  final String message;

  /// The command ID that caused this error, if applicable.
  /// Present in error events but not in error responses (which are already
  /// correlated by the response ID).
  final String? commandId;

  const ErrorPayload({
    required this.code,
    required this.message,
    this.commandId,
  });

  factory ErrorPayload.fromJson(Map<String, dynamic> json) {
    return ErrorPayload(
      code: json[ProtocolKey.code] as String? ?? 'UNKNOWN',
      message: json[ProtocolKey.message] as String? ?? 'Unknown error',
      commandId: json[ProtocolKey.commandId] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      ProtocolKey.code: code,
      ProtocolKey.message: message,
      if (commandId != null) ProtocolKey.commandId: commandId,
    };
  }

  @override
  String toString() =>
      'ErrorPayload(code: $code, message: $message, commandId: $commandId)';
}

/// A generic event emitted by a Tiptap extension.
///
/// The engine does not interpret these — it passes them through as-is.
/// Extensions use this mechanism to communicate with the port for things
/// like mention suggestions, slash commands, or collaborative editing events.
class ExtensionEvent {
  /// The name of the extension that emitted this event.
  final String extensionName;

  /// The event name within the extension's namespace.
  final String eventName;

  /// The event-specific payload. Structure depends on the extension.
  final Map<String, dynamic> data;

  const ExtensionEvent({
    required this.extensionName,
    required this.eventName,
    this.data = const {},
  });

  factory ExtensionEvent.fromJson(Map<String, dynamic> json) {
    return ExtensionEvent(
      extensionName: json[ProtocolKey.extensionName] as String? ?? '',
      eventName: json[ProtocolKey.eventName] as String? ?? '',
      data: json[ProtocolKey.data] as Map<String, dynamic>? ?? const {},
    );
  }

  Map<String, dynamic> toJson() {
    return {
      ProtocolKey.extensionName: extensionName,
      ProtocolKey.eventName: eventName,
      ProtocolKey.data: data,
    };
  }

  @override
  String toString() =>
      'ExtensionEvent(extension: $extensionName, event: $eventName)';
}
