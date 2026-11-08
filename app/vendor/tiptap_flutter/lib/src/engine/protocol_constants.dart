// Constants for the engine communication protocol.
//
// These mirror the wire contract defined in the engine's TypeScript protocol
// types. Every string here corresponds to a field name, command name, event
// name, message type, content format, or error code that crosses the bridge
// between the Flutter port and the Tiptap engine.
//
// The values in this file are authoritative against the engine source: they
// match the keys the engine reads off incoming commands and writes onto
// outgoing responses and events. When the engine protocol changes, this is
// the single file to reconcile.
//
// These classes are pure namespaces. They are declared abstract with a private
// constructor so they cannot be instantiated or extended — they exist only to
// group related string constants under a typed name.

/// Top-level message-class discriminant values, carried in the `type` field
/// of every message that flows across the bridge.
///
/// This is the coarse discriminant. The finer discriminant — which specific
/// command or event a message is — lives in the `name` field (see
/// [ProtocolKey.name]). A message with `type: "event"` and `name: "ready"`
/// is the ready event.
abstract class MessageType {
  MessageType._();

  /// A request from the port to the engine. Carries an `id` for correlation.
  static const String command = 'command';

  /// A reply from the engine correlated to a command by `id`.
  static const String response = 'response';

  /// An asynchronous push from the engine that is not a reply to any command.
  static const String event = 'event';
}

/// Internal bridge-handshake message types injected by the port's own WebView
/// bootstrap script — NOT part of the engine protocol.
///
/// The engine never sends these. They are emitted by the adapter/poll script
/// the bridge injects into the page to report when the JS adapter is wired up
/// and when (or whether) the `TiptapEngine` global becomes available. They are
/// deliberately separated from [MessageType] so a future port author reading
/// the protocol contract does not mistake them for engine-originated messages.
abstract class BridgeInternalMessage {
  BridgeInternalMessage._();

  /// The injected adapter script finished wiring up message forwarding.
  static const String bridgeAdapterReady = 'bridgeAdapterReady';

  /// The `TiptapEngine` global was found and `handleCommand` exists.
  static const String engineGlobalReady = 'engineGlobalReady';

  /// Polling for the `TiptapEngine` global exhausted its attempts.
  static const String engineGlobalTimeout = 'engineGlobalTimeout';
}

/// Command names sent in the `name` field of a command message.
///
/// These are the bridge-level commands the engine's central dispatcher
/// switches on. They are a fixed, engine-defined set — distinct from the
/// editor command names passed through [CommandName.exec] (see
/// [EditorCommand]), which are arbitrary Tiptap command strings the engine
/// forwards to the editor's command chain.
abstract class CommandName {
  CommandName._();

  // Lifecycle.
  static const String init = 'init';
  static const String destroy = 'destroy';
  static const String setEditable = 'setEditable';

  // Content.
  static const String setContent = 'setContent';
  static const String getContent = 'getContent';
  static const String insertContentAt = 'insertContentAt';

  // Text input.
  static const String insertText = 'insertText';
  static const String deleteRange = 'deleteRange';
  static const String backspace = 'backspace';
  static const String enter = 'enter';

  // Generic execution. The actual editor command travels in the payload's
  // `command` field; see [EditorCommand].
  static const String exec = 'exec';

  // Selection.
  static const String setTextSelection = 'setTextSelection';
  static const String setNodeSelection = 'setNodeSelection';
  static const String selectAll = 'selectAll';
  static const String focus = 'focus';
  static const String blur = 'blur';

  // Query.
  static const String getState = 'getState';
  static const String isActive = 'isActive';
  static const String canExec = 'canExec';
  static const String getAttributes = 'getAttributes';
}

/// Event names carried in the `name` field of an event message.
///
/// The engine pushes these asynchronously. [stateChanged] is the primary
/// full-state event; [contentChanged] and [selectionChanged] are the lighter
/// partial events the bridge merges into cached state.
abstract class EventName {
  EventName._();

  /// Full schema introspection, emitted once after init and before [ready].
  static const String schemaReady = 'schemaReady';

  /// Emitted once after [schemaReady]; the engine is operational.
  static const String ready = 'ready';

  /// Full editor state, emitted after every transaction.
  static const String stateChanged = 'stateChanged';

  /// Document-only change. Carries `doc`; omits selection and command state.
  static const String contentChanged = 'contentChanged';

  /// Selection-only change. Carries selection, active marks/nodes, and
  /// command states; omits the document tree and stored marks.
  static const String selectionChanged = 'selectionChanged';

  /// An engine error, optionally correlated to a command via `commandId`.
  static const String error = 'error';

  /// A passthrough event emitted by an extension; the engine does not
  /// interpret it.
  static const String extensionEvent = 'extensionEvent';
}

/// JSON field keys used across every message on the wire.
///
/// Grouped only by comment for readability; all are flat string constants so
/// a single key shared by multiple message shapes (for example [name], which
/// holds both a command name and an event name depending on [type], or
/// [content], which appears in both `setContent` payloads and `getContent`
/// results) is declared exactly once.
abstract class ProtocolKey {
  ProtocolKey._();

  // Message envelope.
  static const String type = 'type';
  static const String id = 'id';

  /// Dual-purpose: the command name on a command message and the event name
  /// on an event message. The [type] field disambiguates which.
  static const String name = 'name';
  static const String payload = 'payload';
  static const String success = 'success';
  static const String error = 'error';

  // Error body (also the shape of an error event's payload).
  static const String code = 'code';
  static const String message = 'message';
  static const String commandId = 'commandId';

  // Command payload fields.
  static const String content = 'content';
  static const String editable = 'editable';
  static const String emitUpdate = 'emitUpdate';
  static const String format = 'format';
  static const String position = 'position';
  static const String text = 'text';
  static const String range = 'range';
  static const String command = 'command';
  static const String args = 'args';
  static const String anchor = 'anchor';
  static const String head = 'head';
  static const String attrs = 'attrs';

  // Range sub-object keys (the {from, to} map built for ranges/selections).
  static const String from = 'from';
  static const String to = 'to';

  // Query-result payload fields.
  // `content` (reused above) holds getContent's result.
  static const String active = 'active';
  static const String canExec = 'canExec';
  // `attrs` (reused above) holds getAttributes' result.
  static const String executed = 'executed';

  // State payload fields.
  static const String doc = 'doc';
  static const String selection = 'selection';
  static const String activeMarks = 'activeMarks';
  static const String activeNodes = 'activeNodes';
  static const String commandStates = 'commandStates';
  static const String decorations = 'decorations';
  static const String storedMarks = 'storedMarks';

  // Performance instrumentation fields (optional, additive).
  // `timings` is a sibling of `payload` on both responses and the stateChanged
  // event, carrying the engine's per-phase duration breakdown. `causedBy` is a
  // sibling of `payload` on stateChanged, carrying the id of the command that
  // produced the state change so the port can pair a keystroke with its
  // resulting repaint exactly. Both are absent on messages where the engine
  // timed no phase (e.g. the initial-state emission) or did not attribute the
  // change to a single command. The phase keys inside the `timings` object are
  // named in [TimingPhase], which is kept separate for the same reason
  // [NodeKey] and [SchemaKey] are: those name a sub-object's shape, not the
  // message envelope.
  static const String timings = 'timings';
  static const String causedBy = 'causedBy';

  // Selection sub-object fields.
  static const String selectionType = 'type';
  static const String empty = 'empty';

  // Command-state sub-object fields.
  static const String isActive = 'isActive';
  static const String depth = 'depth';

  // Extension-event payload fields.
  static const String extensionName = 'extensionName';
  static const String eventName = 'eventName';
  static const String data = 'data';
}

/// Field keys on a serialized document-tree node or mark, as emitted by the
/// engine's state serializer.
///
/// Kept in its own namespace rather than folded into [ProtocolKey] because
/// these name the node/mark shape, not the message envelope. The distinction
/// matters most for `type`: the string `'type'` appears in three unrelated
/// roles across the wire —
///   - [ProtocolKey.type] — the message envelope discriminant
///     ("command"/"response"/"event"),
///   - [ProtocolKey.selectionType] — the selection object's kind
///     ("text"/"node"/"all"/"gapcursor"),
///   - [NodeKey.type] — a node's or mark's type identifier
///     ("paragraph"/"heading"/"bold"/...).
/// They share a wire string but mean different things; giving each its own
/// named constant keeps the parse layer from implying a contract the three
/// roles do not share. The same reasoning applies to [content], [text], and
/// [attrs], which also exist on [ProtocolKey] for unrelated payload fields:
/// the node-shape versions live here so a node parser reads all of one node's
/// fields through a single, consistent namespace.
abstract class NodeKey {
  NodeKey._();

  /// A node's or mark's type identifier (e.g., "paragraph", "heading",
  /// "text", "bold", "link").
  static const String type = 'type';

  /// The resolved start position of a node in the ProseMirror document.
  static const String pos = 'pos';

  /// The resolved end position of a node in the ProseMirror document.
  static const String end = 'end';

  /// A node's child-node array.
  static const String content = 'content';

  /// The literal text of a text node.
  static const String text = 'text';

  /// The marks applied to a text node.
  static const String marks = 'marks';

  /// A node's or mark's attribute map (e.g., heading level, image src,
  /// link href).
  static const String attrs = 'attrs';
}

/// Field keys on the schema-introspection objects emitted in the schemaReady
/// event: node-type info, mark-type info, command info, and their attribute
/// and argument descriptors.
///
/// Kept in its own namespace for the same reason as [NodeKey]: these name the
/// schema-descriptor shape, not the message envelope. In particular `name`
/// here is a node/mark/command's own name (e.g. "heading", "bold",
/// "toggleBold"), distinct from [ProtocolKey.name], which is the envelope's
/// command-or-event name. Giving the schema fields their own constants keeps
/// the parse layer from implying these descriptors share the envelope's `name`
/// contract. Fields with no [ProtocolKey] counterpart (`default`,
/// `contentExpression`, `group`, `isLeaf`, `isInline`, `isBlock`, `args`,
/// `required`, `associatedType`, `extensionName`) are named here for the first
/// time; `attrs` is named here so a descriptor reads all its fields through one
/// namespace.
abstract class SchemaKey {
  SchemaKey._();

  /// A node-type, mark-type, or command's own name.
  static const String name = 'name';

  /// The default value of an attribute descriptor.
  static const String defaultValue = 'default';

  /// A node type's ProseMirror content expression (e.g., "inline*", "block+").
  static const String contentExpression = 'contentExpression';

  /// The schema group a node type belongs to (e.g., "block", "inline").
  static const String group = 'group';

  /// A node type's attribute descriptors, or a mark type's attribute
  /// descriptors.
  static const String attrs = 'attrs';

  /// Whether a node type is a leaf (has no editable content).
  static const String isLeaf = 'isLeaf';

  /// Whether a node type is inline.
  static const String isInline = 'isInline';

  /// Whether a node type is block-level.
  static const String isBlock = 'isBlock';

  /// A command's type (e.g., "toggle-mark", "set-node", "action").
  static const String commandType = 'type';

  /// The node or mark type a command is associated with.
  static const String associatedType = 'associatedType';

  /// A command's argument descriptors.
  static const String args = 'args';

  /// Whether a command argument is required.
  static const String required = 'required';

  /// The name of the Tiptap extension that provides a command.
  static const String extensionName = 'extensionName';

  /// The top-level arrays of the schemaReady payload.
  static const String nodes = 'nodes';
  static const String marks = 'marks';
  static const String commands = 'commands';
}

/// Field keys on the `timings` sub-object emitted by the engine's performance
/// instrumentation. Each value is a duration in milliseconds describing how
/// long the engine spent in one internal phase of handling a command or
/// building a state payload.
///
/// These are durations, never timestamps: the engine's clock and the port's
/// clock are in different domains, so only elapsed deltas cross the wire. The
/// port measures the full send-to-response round-trip itself and uses these
/// phase durations to decompose the engine's share of that total.
///
/// Kept in its own namespace rather than folded into [ProtocolKey] because
/// these name the shape of the `timings` object, not the message envelope. As
/// with [NodeKey] and [SchemaKey], grouping the sub-shape's fields under one
/// typed name keeps the parse layer reading a single object through a single
/// namespace.
///
/// The object is sparse: only phases actually measured for a given message
/// appear. A response carries [handle]; a stateChanged carries the full build
/// breakdown ([serializeDoc], [commandStates], [active], [docDiff], [total]).
/// Every key is therefore optional on read.
///
/// As with the other namespace classes, this is declared abstract with a
/// private constructor so it groups string constants under a typed name
/// without being instantiable.
abstract class TimingPhase {
  TimingPhase._();

  /// Total time inside the command handler, from dispatch entry to just
  /// before the response is sent. For a mutating command this includes the
  /// synchronous state-build work, so it is the engine's full JavaScript-side
  /// cost for the command. Present on responses.
  static const String handle = 'handle';

  /// The recursive document-tree serialization walk.
  static const String serializeDoc = 'serializeDoc';

  /// The command-state sweep: canExec + isActive for every command. Expected
  /// to dominate per-keystroke engine cost on non-trivial documents.
  static const String commandStates = 'commandStates';

  /// The canExec half of the command-state sweep alone: the summed cost, over
  /// every command, of the editor.can()[name]() dry-run. Structurally the
  /// expensive half (it builds and discards a trial transaction per command)
  /// and the one a cached/derived isActive cannot avoid. A sub-phase of
  /// [commandStates]; the two halves sum to approximately its total.
  static const String commandStatesCan = 'commandStatesCan';

  /// The isActive half of the command-state sweep alone: the summed cost, over
  /// every command, of editor.isActive(name). Derivable in principle from the
  /// already-computed active marks/nodes, so its size bounds how much a
  /// derivation optimization can save. A sub-phase of [commandStates].
  static const String commandStatesActive = 'commandStatesActive';

  /// The combined active-marks, active-nodes, and stored-marks extraction.
  static const String active = 'active';

  /// The change-detection JSON.stringify of the document in onTransaction.
  static const String docDiff = 'docDiff';

  /// Total time inside onTransaction: the state build, the diff, and the
  /// adapter send calls. Present on stateChanged events.
  static const String total = 'total';
}

/// Values for the `format` field of a `getContent` command.
///
/// The engine returns the result in the `content` key of the response
/// payload, where the runtime shape is discriminated by this format: a String
/// for [html] and [text], a JSON object for [json].
abstract class ContentFormat {
  ContentFormat._();

  static const String json = 'json';
  static const String html = 'html';
  static const String text = 'text';
}

/// Editor command names passed through [CommandName.exec] in the payload's
/// `command` field.
///
/// Unlike [CommandName], this is NOT an exhaustive or gating set. The engine
/// forwards any string here to the editor's command chain, so ports and apps
/// may call commands not listed here. These constants cover the commands the
/// current StarterKit + Image build exposes and the toolbar uses; they exist
/// as a convenience to replace bare literals, not as a restriction on what
/// `exec` accepts.
abstract class EditorCommand {
  EditorCommand._();

  // Formatting marks.
  static const String toggleBold = 'toggleBold';
  static const String toggleItalic = 'toggleItalic';
  static const String toggleStrike = 'toggleStrike';
  static const String toggleCode = 'toggleCode';

  // Block types.
  static const String toggleHeading = 'toggleHeading';
  static const String toggleCodeBlock = 'toggleCodeBlock';
  static const String toggleBlockquote = 'toggleBlockquote';

  // Lists.
  static const String toggleBulletList = 'toggleBulletList';
  static const String toggleOrderedList = 'toggleOrderedList';

  // Inserts.
  static const String setHorizontalRule = 'setHorizontalRule';
  static const String setImage = 'setImage';

  // History.
  static const String undo = 'undo';
  static const String redo = 'redo';
}

/// Machine-readable error codes the engine emits in the `code` field of error
/// responses and error events.
///
/// This is the fixed set defined by the engine's command dispatcher and
/// guards. App code may branch on these instead of matching message strings.
abstract class ErrorCode {
  ErrorCode._();

  /// A command was sent before init or after destroy.
  static const String notInitialized = 'NOT_INITIALIZED';

  /// An init command arrived while the engine was already running.
  static const String alreadyInitialized = 'ALREADY_INITIALIZED';

  /// The top-level command name was not recognized by the dispatcher.
  static const String unknownCommand = 'UNKNOWN_COMMAND';

  /// The command passed to `exec` does not exist on the editor.
  static const String unknownExecCommand = 'UNKNOWN_EXEC_COMMAND';

  /// An unknown format was passed to `getContent`.
  static const String invalidFormat = 'INVALID_FORMAT';

  /// A command threw during execution.
  static const String commandFailed = 'COMMAND_FAILED';
}
