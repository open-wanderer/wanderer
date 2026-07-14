// High-level controller for the Tiptap editor.
//
// This class wraps [TiptapBridge] and provides a clean, typed API for the
// editor UI layer. It manages the editor lifecycle (initialization, destruction),
// exposes reactive streams for state changes, and provides convenience methods
// for every command the engine supports.
//
// The controller is the single point of contact between the UI and the engine.
// UI widgets never touch the bridge directly — they call methods on the
// controller and listen to its streams.
//
// Usage:
//   final controller = EditorController();
//   await controller.initialize(content: '<p>Hello</p>');
//   controller.editorStateStream.listen((state) { /* rebuild UI */ });
//   await controller.execCommand('toggleBold');
//   controller.dispose();

import 'dart:async';

import 'package:flutter/widgets.dart';

import '../engine/metrics.dart';
import '../engine/protocol_types.dart';
import '../engine/query_results.dart';
import '../engine/tiptap_bridge.dart';

/// The editor controller manages the bridge lifecycle and provides a typed
/// API for the editor UI.
///
/// Create one controller per editor instance. The controller creates and owns
/// the [TiptapBridge], which in turn owns the headless WebView.
class EditorController {
  /// The underlying bridge that communicates with the headless WebView.
  /// Exposed for advanced use cases (e.g., accessing the WebView widget
  /// for placement in the widget tree, or listening to raw events).
  final TiptapBridge bridge = TiptapBridge();

  /// Subscriptions to bridge streams, cancelled on dispose.
  final List<StreamSubscription> _subscriptions = [];

  /// Whether the editor has been initialized and is ready for commands.
  bool _isReady = false;
  bool get isReady => _isReady;

  /// Guard to prevent sending init more than once.
  bool _initSent = false;

  /// Completer that resolves when the editor reaches the ready state.
  /// This allows callers to await initialization completion.
  Completer<void>? _readyCompleter;

  // ---------------------------------------------------------------------------
  // Cached state
  // ---------------------------------------------------------------------------

  /// The current engine lifecycle state.
  EngineState _engineState = EngineState.uninitialized;
  EngineState get engineState => _engineState;

  /// Schema metadata received from the engine during initialization.
  SchemaMetadata? _schema;
  SchemaMetadata? get schema => _schema;

  /// The latest editor state from the engine.
  EditorStatePayload? _editorState;
  EditorStatePayload? get editorState => _editorState;

  /// The latest error message, if any.
  String? get errorMessage => bridge.errorMessage;

  // ---------------------------------------------------------------------------
  // Streams (re-exported from bridge with controller-level transformations)
  // ---------------------------------------------------------------------------

  /// Stream of engine lifecycle state transitions.
  Stream<EngineState> get engineStateStream => bridge.engineStateStream;

  /// Stream of editor state updates, emitted on every transaction.
  /// This is the primary stream that UI widgets listen to for rebuilds.
  Stream<EditorStatePayload> get editorStateStream => bridge.stateChangedStream;

  /// Stream of schema metadata, emitted once during initialization.
  Stream<SchemaMetadata> get schemaStream => bridge.schemaReadyStream;

  /// Stream of engine error events.
  Stream<ErrorPayload> get errorStream => bridge.errorEventStream;

  /// Stream of extension-specific events. Listen to this for custom
  /// extension communication (e.g., mention suggestions, slash commands).
  Stream<ExtensionEvent> get extensionEventStream =>
      bridge.extensionEventStream;

  /// Stream of debug log entries from the bridge.
  Stream<BridgeLogEntry> get logStream => bridge.logStream;

  // ---------------------------------------------------------------------------
  // Performance metrics
  // ---------------------------------------------------------------------------

  /// Performance metrics collected by the bridge (command round-trips,
  /// engine load phases) and the editor (typing latency). Read by the
  /// performance overlay.
  BridgeMetrics get metrics => bridge.metrics;

  /// Stream that emits whenever a new metric sample is recorded, so the
  /// performance overlay can rebuild live.
  Stream<void> get metricsStream => bridge.metricsStream;

  /// Report an end-to-end typing-latency sample measured by the editor.
  ///
  /// [operation] is the input kind ("insert", "delete", "newline").
  /// [milliseconds] is the time from keystroke to completed repaint.
  /// [exact] is whether the measurement came from an exact correlation token
  /// (false until the engine emits one — see the editor's timing seam).
  void recordTypingSample(
    String operation,
    double milliseconds, {
    required bool exact,
  }) {
    bridge.metrics.recordTypingSample(operation, milliseconds, exact: exact);
  }

  /// Report that a keystroke's latency could not be measured because the
  /// in-order approximation was ambiguous.
  void recordDroppedTypingSample() {
    bridge.metrics.recordDroppedTypingSample();
  }

  // ---------------------------------------------------------------------------
  // The WebView widget
  // ---------------------------------------------------------------------------

  /// The invisible WebView widget that must be placed in the widget tree.
  /// Wrap this in an Offstage or zero-size container. The WebView serves
  /// purely as a computation engine — no pixels from it reach the screen.
  Widget get webViewWidget => bridge.webViewWidget;

  // ---------------------------------------------------------------------------
  // Initialization
  // ---------------------------------------------------------------------------

  /// Initialize the editor with optional initial content.
  ///
  /// This method:
  ///   1. Initializes the bridge (loads engine assets into the WebView)
  ///   2. Waits for the engine JS global to become available
  ///   3. Sends the init command with the provided content
  ///   4. Waits for the engine to reach the ready state
  ///
  /// The returned Future completes when the editor is fully ready for commands.
  ///
  /// [content] is the initial HTML content to load into the editor.
  /// [editable] controls whether the editor starts in editable mode.
  Future<void> initialize({String? content, bool editable = true}) async {
    if (_initSent) return;

    _readyCompleter = Completer<void>();

    /// Subscribe to bridge streams to track state and cache values.
    _subscriptions.add(bridge.engineStateStream.listen(_onEngineStateChanged));

    _subscriptions.add(
      bridge.schemaReadyStream.listen((schema) {
        _schema = schema;
      }),
    );

    _subscriptions.add(
      bridge.stateChangedStream.listen((state) {
        _editorState = state;
      }),
    );

    /// Store the content and editable flag for use when the engine is ready.
    _pendingContent = content;
    _pendingEditable = editable;

    /// Start the bridge initialization (loads WebView, injects adapter, polls
    /// for engine global).
    await bridge.initialize();

    /// Wait for the full initialization sequence to complete:
    /// page load → engine global ready → init command → schema ready → ready.
    return _readyCompleter!.future;
  }

  /// Content and editable flag stored during initialize(), sent when the
  /// engine global becomes available.
  String? _pendingContent;
  bool _pendingEditable = true;

  /// Called on every engine state transition. Handles the initialization
  /// sequence by sending the init command when the engine global is ready,
  /// and completing the ready completer when the engine is fully operational.
  void _onEngineStateChanged(EngineState state) {
    _engineState = state;

    switch (state) {
      case EngineState.engineGlobalReady:

        /// The engine JS global is available. Send the init command.
        _sendInit();
        break;

      case EngineState.ready:

        /// The engine is fully operational. Complete the ready future.
        _isReady = true;
        if (_readyCompleter != null && !_readyCompleter!.isCompleted) {
          _readyCompleter!.complete();
        }
        break;

      case EngineState.error:

        /// Initialization failed. Complete the ready future with an error.
        _isReady = false;
        if (_readyCompleter != null && !_readyCompleter!.isCompleted) {
          _readyCompleter!.completeError(
            Exception(bridge.errorMessage ?? 'Unknown initialization error'),
          );
        }
        break;

      default:
        break;
    }
  }

  /// Send the init command to the engine.
  Future<void> _sendInit() async {
    if (_initSent) return;
    _initSent = true;

    try {
      await bridge.initEditor(
        content: _pendingContent,
        editable: _pendingEditable,
      );
    } catch (e) {
      _initSent = false;
      _isReady = false;
      if (_readyCompleter != null && !_readyCompleter!.isCompleted) {
        _readyCompleter!.completeError(e);
      }
    }
  }

  // ---------------------------------------------------------------------------
  // Lifecycle commands
  // ---------------------------------------------------------------------------

  /// Destroy the editor instance in the engine.
  /// After this, a new [initialize] call is required to use the editor again.
  Future<void> destroy() async {
    await bridge.destroyEditor();
    _isReady = false;
    _initSent = false;
    _editorState = null;
    _schema = null;
  }

  /// Toggle the editor's editable/read-only mode.
  Future<void> setEditable(bool editable) async {
    _ensureReady();
    await bridge.setEditable(editable);
  }

  // ---------------------------------------------------------------------------
  // Content commands
  // ---------------------------------------------------------------------------

  /// Replace the entire document content with new HTML.
  Future<void> setContent(String htmlContent, {bool emitUpdate = true}) async {
    _ensureReady();
    await bridge.setContent(htmlContent, emitUpdate: emitUpdate);
  }

  /// Get the current document content as HTML.
  Future<String> getHTML() async {
    _ensureReady();
    final result = await bridge.getHTML();
    return HtmlContent.fromResponse(result).html;
  }

  /// Get the current document content as plain text.
  Future<String> getText() async {
    _ensureReady();
    final result = await bridge.getText();
    return TextContent.fromResponse(result).text;
  }

  /// Get the current document content as a Tiptap JSON object.
  Future<Map<String, dynamic>> getJSON() async {
    _ensureReady();
    final result = await bridge.getJSON();
    return JsonContent.fromResponse(result).json;
  }

  /// Insert content at a specific position or range.
  Future<void> insertContentAt(dynamic position, String content) async {
    _ensureReady();
    await bridge.insertContentAt(position, content);
  }

  // ---------------------------------------------------------------------------
  // Text input commands
  // ---------------------------------------------------------------------------

  /// Insert text at the current selection or a given range.
  Future<void> insertText(String text, {Map<String, int>? range}) async {
    _ensureReady();
    await bridge.insertText(text, range: range);
  }

  /// Delete content in a range or perform simple backspace at cursor.
  Future<void> deleteRange({Map<String, int>? range}) async {
    _ensureReady();
    await bridge.deleteRange(range: range);
  }

  /// Perform a full backspace operation at the current cursor position.
  /// Handles structural operations like joining blocks, lifting list items,
  /// and deleting atomic nodes.
  Future<void> backspace() async {
    _ensureReady();
    await bridge.backspace();
  }

  /// Perform a full Enter operation at the current cursor position.
  /// Handles context-specific behavior like splitting paragraphs, creating
  /// new list items, exiting code blocks, etc.
  Future<void> enter() async {
    _ensureReady();
    await bridge.enter();
  }

  // ---------------------------------------------------------------------------
  // Generic execution
  // ---------------------------------------------------------------------------

  /// Execute any Tiptap command by name.
  ///
  /// This is the primary method for formatting and structural commands.
  /// Examples:
  ///   execCommand('toggleBold')
  ///   execCommand('setHeading', {'level': 2})
  ///   execCommand('insertTable', {'rows': 3, 'cols': 3})
  ///
  /// The engine's exec response carries an `executed` flag indicating whether
  /// the command actually ran (false for a no-op that could not apply in the
  /// current state). This method does not surface that flag; callers needing
  /// it can parse the raw bridge response with [ExecResult.fromResponse].
  Future<void> execCommand(
    String commandName, [
    Map<String, dynamic>? args,
  ]) async {
    _ensureReady();
    await bridge.execCommand(commandName, args);
  }

  // ---------------------------------------------------------------------------
  // Selection commands
  // ---------------------------------------------------------------------------

  /// Set a cursor or text range selection.
  ///
  /// If only [anchor] is provided, places a collapsed cursor at that position.
  /// If [head] is also provided, creates a range selection from anchor to head.
  Future<void> setTextSelection(int anchor, {int? head}) async {
    _ensureReady();
    await bridge.setTextSelection(anchor, head: head);
  }

  /// Select an entire node at a position.
  Future<void> setNodeSelection(int position) async {
    _ensureReady();
    await bridge.setNodeSelection(position);
  }

  /// Select the entire document.
  Future<void> selectAll() async {
    _ensureReady();
    await bridge.selectAll();
  }

  /// Set logical focus on the editor.
  Future<void> focus({dynamic position}) async {
    _ensureReady();
    await bridge.focus(position: position);
  }

  /// Remove logical focus from the editor.
  Future<void> blur() async {
    _ensureReady();
    await bridge.blur();
  }

  // ---------------------------------------------------------------------------
  // Query commands
  // ---------------------------------------------------------------------------

  /// Request a full state snapshot from the engine.
  Future<EditorStatePayload> getState() async {
    _ensureReady();
    final result = await bridge.getState();
    final payload = result['payload'] as Map<String, dynamic>? ?? {};
    return EditorStatePayload.fromJson(payload);
  }

  /// Check if a mark or node type is active at the current selection.
  Future<bool> isActive(String name, {Map<String, dynamic>? attrs}) async {
    _ensureReady();
    final result = await bridge.isActive(name, attrs: attrs);
    return IsActiveResult.fromResponse(result).active;
  }

  /// Check if a command can execute in the current state.
  Future<bool> canExec(String command, {Map<String, dynamic>? args}) async {
    _ensureReady();
    final result = await bridge.canExec(command, args: args);
    return CanExecResult.fromResponse(result).canExec;
  }

  /// Get attributes of a mark or node type at the current selection.
  Future<Map<String, dynamic>> getAttributes(String name) async {
    _ensureReady();
    final result = await bridge.getAttributes(name);
    return AttributesResult.fromResponse(result).attrs;
  }

  // ---------------------------------------------------------------------------
  // Convenience queries on cached state
  // ---------------------------------------------------------------------------

  /// Check if a command is currently active based on cached state.
  /// This is synchronous and uses the last known state — no engine round-trip.
  bool isCommandActive(String commandName) {
    return _editorState?.commandStates[commandName]?.isActive ?? false;
  }

  /// Check if a command can currently execute based on cached state.
  /// This is synchronous and uses the last known state — no engine round-trip.
  bool canCommandExec(String commandName) {
    return _editorState?.commandStates[commandName]?.canExec ?? false;
  }

  /// Get the current selection from cached state.
  SelectionState? get selection => _editorState?.selection;

  /// Get the active marks at the current selection from cached state.
  List<String> get activeMarks => _editorState?.activeMarks ?? [];

  /// Get the active nodes at the current selection from cached state.
  List<ActiveNode> get activeNodes => _editorState?.activeNodes ?? [];

  /// Get the current document tree from cached state.
  AnnotatedNode? get document => _editorState?.doc;

  // ---------------------------------------------------------------------------
  // Internal utilities
  // ---------------------------------------------------------------------------

  /// Throws if the editor is not in the ready state.
  void _ensureReady() {
    if (!_isReady) {
      throw StateError(
        'Editor is not ready. Current state: $_engineState. '
        'Call initialize() and await its completion before sending commands.',
      );
    }
  }

  /// Clean up all resources.
  void dispose() {
    for (final sub in _subscriptions) {
      sub.cancel();
    }
    _subscriptions.clear();
    bridge.dispose();
  }
}
