// The bridge between Flutter and the headless Tiptap engine running in a WebView.
//
// This class owns the invisible WebView instance and manages all communication
// with the Tiptap engine via JSON messages. The WebView is purely a computation
// engine — no pixels from it ever reach the user's screen.
//
// Communication flow:
//   Commands:  Dart → JSON → runJavaScript → Engine
//   Responses: Engine → JavaScript channel → JSON → Dart (matched by command ID)
//   Events:    Engine → JavaScript channel → JSON → Dart (broadcast via streams)

import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/services.dart' show rootBundle;
import 'package:flutter/widgets.dart';
import 'package:path_provider/path_provider.dart';
import 'package:webview_flutter/webview_flutter.dart';

import 'metrics.dart';
import 'protocol_constants.dart';
import 'protocol_types.dart';

/// Debug-log direction labels for [BridgeLogEntry].
///
/// Port-internal, not part of the engine protocol: these classify log entries
/// for the debug panel and console output, describing whether a logged message
/// was sent, received, an internal lifecycle note, an error, or a warning.
abstract class LogDirection {
  LogDirection._();

  /// A command the bridge sent to the engine.
  static const String sent = 'sent';

  /// A response or event the bridge received from the engine.
  static const String received = 'received';

  /// An internal lifecycle or diagnostic note.
  static const String system = 'system';

  /// A failure.
  static const String error = 'error';

  /// An unexpected but non-fatal condition.
  static const String warning = 'warning';
}

/// Possible states of the engine lifecycle.
enum EngineState {
  /// The engine has not been initialized yet.
  uninitialized,

  /// The engine assets are being loaded and the WebView is starting up.
  loading,

  /// The WebView page has finished loading.
  pageLoaded,

  /// The TiptapEngine global is available and handleCommand exists.
  engineGlobalReady,

  /// The engine has emitted the schemaReady event and is preparing.
  schemaReady,

  /// The engine has emitted the ready event and is fully operational.
  ready,

  /// An error occurred during initialization or communication.
  error,

  /// The engine has been explicitly destroyed.
  destroyed,
}

/// A log entry for debugging bridge communication.
class BridgeLogEntry {
  final DateTime timestamp;

  /// The direction of the message: "sent" for commands, "received" for
  /// responses/events, "system" for internal lifecycle messages,
  /// "error" for failures, "warning" for unexpected but non-fatal issues.
  final String direction;

  /// The raw JSON string or descriptive text of the message.
  final String message;

  const BridgeLogEntry({
    required this.timestamp,
    required this.direction,
    required this.message,
  });
}

/// Manages the headless WebView running the Tiptap engine and provides
/// a typed Dart API for sending commands and receiving events.
class TiptapBridge {
  /// The WebView controller — created eagerly in the constructor so that
  /// the widget is available for the widget tree immediately, avoiding
  /// late initialization errors.
  final WebViewController _controller = WebViewController();

  /// The WebViewWidget that must be placed in the widget tree.
  /// It should be wrapped in a zero-size container or Offstage so it remains
  /// invisible to the user. The webview_flutter package requires the widget
  /// to be in the tree for the controller to function.
  ///
  /// Created in the constructor so it's available before initialize() completes.
  late final Widget webViewWidget = WebViewWidget(controller: _controller);

  /// Whether initialize() has been called and completed (successfully or not).
  bool _initialized = false;
  bool get isInitialized => _initialized;

  /// Auto-incrementing counter for generating unique command IDs.
  int _nextId = 0;

  /// Pending command futures waiting for a response from the engine.
  /// Keyed by the command ID string (e.g., "cmd_0", "cmd_1").
  final Map<String, Completer<Map<String, dynamic>>> _pendingCommands = {};

  /// The current state of the engine lifecycle.
  EngineState _engineState = EngineState.uninitialized;
  EngineState get engineState => _engineState;

  // ---------------------------------------------------------------------------
  // Streams
  // ---------------------------------------------------------------------------

  /// Stream controller for engine state transitions.
  final _engineStateController = StreamController<EngineState>.broadcast();
  Stream<EngineState> get engineStateStream => _engineStateController.stream;

  /// Stream controller for schema metadata, emitted once during init.
  final _schemaReadyController = StreamController<SchemaMetadata>.broadcast();
  Stream<SchemaMetadata> get schemaReadyStream => _schemaReadyController.stream;

  /// Stream controller for editor state updates, emitted on every transaction.
  final _stateChangedController =
      StreamController<EditorStatePayload>.broadcast();
  Stream<EditorStatePayload> get stateChangedStream =>
      _stateChangedController.stream;

  /// Stream controller for raw event data, useful for debugging.
  final _rawEventController =
      StreamController<Map<String, dynamic>>.broadcast();
  Stream<Map<String, dynamic>> get rawEventStream => _rawEventController.stream;

  /// Stream controller for engine error events.
  final _errorEventController = StreamController<ErrorPayload>.broadcast();
  Stream<ErrorPayload> get errorEventStream => _errorEventController.stream;

  /// Stream controller for extension-specific events.
  final _extensionEventController =
      StreamController<ExtensionEvent>.broadcast();
  Stream<ExtensionEvent> get extensionEventStream =>
      _extensionEventController.stream;

  /// Log of all messages sent and received, for the debug panel in the PoC UI.
  final List<BridgeLogEntry> _log = [];
  List<BridgeLogEntry> get log => List.unmodifiable(_log);

  /// Stream controller that emits whenever a new log entry is added.
  final _logController = StreamController<BridgeLogEntry>.broadcast();
  Stream<BridgeLogEntry> get logStream => _logController.stream;

  // ---------------------------------------------------------------------------
  // Performance metrics
  // ---------------------------------------------------------------------------

  /// Performance metrics collected by the bridge: command round-trips,
  /// engine load phases, typing-latency samples reported by the editor, and
  /// engine-reported internal phase durations. The editor pushes typing
  /// samples through the controller, which forwards them here, so the overlay
  /// has a single source for all metrics.
  final BridgeMetrics metrics = BridgeMetrics();

  /// Stream controller that emits whenever a new metric sample is recorded,
  /// so the performance overlay can rebuild live.
  final _metricsController = StreamController<void>.broadcast();
  Stream<void> get metricsStream => _metricsController.stream;

  /// Map of command id to the time the command was sent, used to compute
  /// round-trip durations when the response arrives. Entries are removed on
  /// response or timeout so the map does not grow unbounded.
  final Map<String, DateTime> _commandSentAt = {};

  /// Map of command id to command name, used to label round-trip samples
  /// when the response (which omits the name) arrives.
  final Map<String, String> _commandNames = {};

  /// Timestamp of the previous lifecycle transition, used to compute the
  /// duration of each load phase. Set on the first transition into loading.
  DateTime? _lastTransitionAt;

  /// Timestamp when initialization started (transition into loading), used
  /// to compute total cold-start time once the engine is ready.
  DateTime? _loadStartedAt;

  /// The id of the command that caused the most recent stateChanged event,
  /// as reported by the engine's `causedBy` field, or null when the engine
  /// did not attribute the state change to a single command (initial state,
  /// async plugin transactions). Exposed so the editor's typing-latency
  /// tracker can move from in-order approximation to exact keystroke→repaint
  /// pairing: the tracker can match the keystroke whose command id equals
  /// this value, rather than popping the oldest pending keystroke.
  ///
  /// This is the read seam for that future change; the bridge records and
  /// exposes the token now, and the tracker rewrite consumes it separately.
  String? _lastCausedBy;
  String? get lastCausedBy => _lastCausedBy;

  // ---------------------------------------------------------------------------
  // Cached state
  // ---------------------------------------------------------------------------

  /// The most recently received schema metadata.
  SchemaMetadata? _schemaMetadata;
  SchemaMetadata? get schemaMetadata => _schemaMetadata;

  /// The most recently received editor state.
  EditorStatePayload? _lastState;
  EditorStatePayload? get lastState => _lastState;

  /// Error message if the engine is in the error state.
  String? _errorMessage;
  String? get errorMessage => _errorMessage;

  // ---------------------------------------------------------------------------
  // Initialization
  // ---------------------------------------------------------------------------

  /// Initialize the bridge by configuring the WebView controller,
  /// setting up the JavaScript communication channel, and loading
  /// the engine assets.
  ///
  /// The WebView controller and widget are created eagerly in the constructor,
  /// so they can be placed in the widget tree before this method is called.
  /// This method configures the controller and starts the asset loading process.
  ///
  /// After this method returns, the WebView is loading. The actual readiness
  /// sequence happens asynchronously through the message channel:
  ///   page loads → adapter injected → polls for TiptapEngine global →
  ///   engineGlobalReady message → state transitions
  Future<void> initialize() async {
    if (_initialized) {
      _addLog(
        LogDirection.warning,
        'initialize() called more than once, ignoring',
      );
      return;
    }

    _addLog(LogDirection.system, 'Bridge initialization starting');

    /// Wire the metrics change callback so any recorded sample emits a tick
    /// on the metrics stream for the performance overlay.
    metrics.onChange = () => _metricsController.add(null);

    _updateState(EngineState.loading);

    try {
      /// Enable JavaScript execution — the engine is entirely JS-based.
      await _controller.setJavaScriptMode(JavaScriptMode.unrestricted);
      _addLog(LogDirection.system, 'JavaScript mode set to unrestricted');

      /// Register the JavaScript channel that receives messages from the engine.
      /// The engine's platform adapter posts messages to this channel.
      /// On Android, the engine calls: window.TiptapBridge.postMessage(json)
      /// On iOS with webview_flutter, the channel is also registered under
      /// the same name via the platform's JavaScript channel mechanism.
      await _controller.addJavaScriptChannel(
        'TiptapBridge',
        onMessageReceived: (JavaScriptMessage message) {
          _handleIncomingMessage(message.message);
        },
      );
      _addLog(
        LogDirection.system,
        'JavaScript channel "TiptapBridge" registered',
      );

      /// Set up a navigation delegate to detect when the page finishes loading.
      /// Once loaded, the engine JS is executing and we can inject the bridge
      /// adapter and start polling for the engine global.
      ///
      /// IMPORTANT: onPageFinished is a callback — we must not await anything
      /// that depends on the message channel inside it, because that creates
      /// a deadlock. Instead, we fire-and-forget the adapter injection and
      /// polling, and let the message channel handle the results reactively.
      await _controller.setNavigationDelegate(
        NavigationDelegate(
          onPageStarted: (String url) {
            _addLog(LogDirection.system, 'Page started loading: $url');
          },
          onProgress: (int progress) {
            if (progress % 25 == 0) {
              _addLog(LogDirection.system, 'Page load progress: $progress%');
            }
          },
          onPageFinished: (String url) {
            _addLog(LogDirection.system, 'Page finished loading: $url');
            _updateState(EngineState.pageLoaded);

            /// Fire-and-forget: inject the bridge adapter and start polling.
            /// Results come back through the message channel.
            _injectBridgeAndPollForEngine();
          },
          onWebResourceError: (WebResourceError error) {
            _updateState(EngineState.error);
            _errorMessage =
                'WebView resource error: ${error.description} '
                '(code: ${error.errorCode}, '
                'type: ${error.errorType}, '
                'url: ${error.url})';
            _addLog(LogDirection.error, _errorMessage!);
          },
          onNavigationRequest: (NavigationRequest request) {
            _addLog(
              LogDirection.system,
              'Navigation request: ${request.url} '
              '(isMainFrame: ${request.isMainFrame})',
            );
            return NavigationDecision.navigate;
          },
        ),
      );
      _addLog(LogDirection.system, 'Navigation delegate configured');

      /// Load the engine HTML into the WebView from a temp directory.
      await _loadEngineAssets();

      _initialized = true;
      _addLog(
        LogDirection.system,
        'Bridge initialization completed — '
        'waiting for page load and engine readiness',
      );
    } catch (e, stackTrace) {
      _updateState(EngineState.error);
      _errorMessage = 'Initialization failed: $e';
      _addLog(LogDirection.error, '$_errorMessage\nStack trace:\n$stackTrace');
      _initialized = true;
      rethrow;
    }
  }

  /// Copy the engine HTML and JS assets from the Flutter asset bundle to a
  /// temporary directory, then load the HTML file into the WebView.
  ///
  /// WebView can't reliably load Flutter assets via file:// across all platforms,
  /// so we write them to disk first and load from there.
  ///
  /// Assets are loaded from the package's own asset bundle using the
  /// 'packages/tiptap_flutter/' prefix, which is how Flutter resolves
  /// assets declared in a package's pubspec.yaml.
  Future<void> _loadEngineAssets() async {
    _addLog(
      LogDirection.system,
      'Loading engine assets from Flutter asset bundle',
    );

    final tempDir = await getTemporaryDirectory();
    final engineDir = Directory('${tempDir.path}/tiptap_engine');
    _addLog(LogDirection.system, 'Temp directory: ${tempDir.path}');

    /// Create the engine directory if it doesn't already exist.
    if (!await engineDir.exists()) {
      await engineDir.create(recursive: true);
      _addLog(
        LogDirection.system,
        'Created engine directory: ${engineDir.path}',
      );
    } else {
      _addLog(
        LogDirection.system,
        'Engine directory already exists: ${engineDir.path}',
      );
    }

    /// Read the HTML shell and JS bundle from the package's assets.
    /// The 'packages/tiptap_flutter/' prefix tells Flutter to look in
    /// this package's asset bundle rather than the host app's assets.
    _addLog(
      LogDirection.system,
      'Reading tiptap-engine.html from package assets',
    );
    final htmlContent = await rootBundle.loadString(
      'packages/tiptap_flutter/assets/engine/tiptap-engine.html',
    );
    _addLog(
      LogDirection.system,
      'Read tiptap-engine.html (${htmlContent.length} chars)',
    );

    _addLog(
      LogDirection.system,
      'Reading tiptap-engine.js from package assets',
    );
    final jsContent = await rootBundle.loadString(
      'packages/tiptap_flutter/assets/engine/tiptap-engine.js',
    );
    _addLog(
      LogDirection.system,
      'Read tiptap-engine.js (${jsContent.length} chars)',
    );

    /// Write both files to the temp directory.
    final htmlFile = File('${engineDir.path}/tiptap-engine.html');
    final jsFile = File('${engineDir.path}/tiptap-engine.js');

    await htmlFile.writeAsString(htmlContent);
    _addLog(LogDirection.system, 'Wrote HTML to ${htmlFile.path}');

    await jsFile.writeAsString(jsContent);
    _addLog(LogDirection.system, 'Wrote JS to ${jsFile.path}');

    /// Verify the files were written successfully.
    final htmlExists = await htmlFile.exists();
    final jsExists = await jsFile.exists();
    final htmlSize = await htmlFile.length();
    final jsSize = await jsFile.length();
    _addLog(
      LogDirection.system,
      'File verification — HTML exists: $htmlExists ($htmlSize bytes), '
      'JS exists: $jsExists ($jsSize bytes)',
    );

    /// Load the HTML file into the WebView using a file:// URI.
    _addLog(LogDirection.system, 'Loading HTML into WebView: ${htmlFile.path}');
    await _controller.loadFile(htmlFile.path);
  }

  /// Inject the bridge adapter and start polling for the TiptapEngine global.
  ///
  /// This is called from onPageFinished as fire-and-forget. It injects a
  /// single script that does two things:
  ///   1. Sets up the bridge adapter (webkit messageHandler forwarding)
  ///   2. Starts polling for TiptapEngine.handleCommand to exist
  ///
  /// Results are communicated back through the TiptapBridge message channel
  /// rather than through return values, avoiding the Android WebView issue
  /// where runJavaScriptReturningResult doesn't properly await Promises.
  Future<void> _injectBridgeAndPollForEngine() async {
    _addLog(
      LogDirection.system,
      'Injecting bridge adapter and starting engine poll',
    );

    /// Combined script: adapter setup + engine polling.
    /// Everything communicates results via TiptapBridge.postMessage.
    const script = '''
      (function() {
        // === STEP 1: Bridge adapter setup ===

        // Ensure webkit messageHandler path exists for iOS compatibility.
        if (!window.webkit) { window.webkit = {}; }
        if (!window.webkit.messageHandlers) { window.webkit.messageHandlers = {}; }
        if (!window.webkit.messageHandlers.TiptapEngine) {
          window.webkit.messageHandlers.TiptapEngine = {
            postMessage: function(msg) {
              var jsonStr = (typeof msg === 'string') ? msg : JSON.stringify(msg);
              window.TiptapBridge.postMessage(jsonStr);
            }
          };
        }

        // Wrap TiptapBridge.postMessage to ensure messages are always strings.
        var originalPostMessage = window.TiptapBridge.postMessage.bind(window.TiptapBridge);
        window.TiptapBridge.postMessage = function(msg) {
          var jsonStr = (typeof msg === 'string') ? msg : JSON.stringify(msg);
          originalPostMessage(jsonStr);
        };

        console.log('[TiptapBridge] Adapter injected successfully');

        // Notify Dart that the adapter is ready.
        window.TiptapBridge.postMessage(JSON.stringify({
          type: 'bridgeAdapterReady'
        }));

        // === STEP 2: Poll for TiptapEngine global ===

        var attempts = 0;
        var maxAttempts = 200;
        var intervalMs = 50;

        function checkEngine() {
          attempts++;

          if (typeof TiptapEngine !== 'undefined' && typeof TiptapEngine.handleCommand === 'function') {
            window.TiptapBridge.postMessage(JSON.stringify({
              type: 'engineGlobalReady',
              attempts: attempts,
              elapsed: attempts * intervalMs,
              engineKeys: Object.keys(TiptapEngine)
            }));
            return;
          }

          if (attempts >= maxAttempts) {
            var diag = {
              hasTiptapEngine: typeof TiptapEngine !== 'undefined',
              tiptapEngineType: typeof TiptapEngine,
              tiptapGlobals: Object.keys(window).filter(function(k) {
                return k.toLowerCase().indexOf('tiptap') !== -1;
              }),
              allGlobals: Object.keys(window).filter(function(k) {
                // Filter out built-in browser globals to find custom ones.
                return ['TiptapEngine','TiptapBridge','tiptap','editor'].indexOf(k) !== -1
                  || k.toLowerCase().indexOf('tiptap') !== -1
                  || k.toLowerCase().indexOf('prosemirror') !== -1
                  || k.toLowerCase().indexOf('engine') !== -1;
              })
            };
            window.TiptapBridge.postMessage(JSON.stringify({
              type: 'engineGlobalTimeout',
              attempts: attempts,
              elapsed: attempts * intervalMs,
              diagnostics: diag
            }));
            return;
          }

          setTimeout(checkEngine, intervalMs);
        }

        checkEngine();
      })();
    ''';

    try {
      await _controller.runJavaScript(script);
      _addLog(
        LogDirection.system,
        'Bridge + poll script injected (fire-and-forget)',
      );
    } catch (e, stackTrace) {
      _addLog(
        LogDirection.error,
        'Failed to inject bridge + poll script: $e\n'
        'Stack trace:\n$stackTrace',
      );
      _updateState(EngineState.error);
      _errorMessage = 'Failed to inject bridge script: $e';
    }
  }

  // ---------------------------------------------------------------------------
  // Command transport
  // ---------------------------------------------------------------------------

  /// Send a command to the engine and return a Future that completes
  /// when the engine sends back a response with the matching ID.
  ///
  /// The command protocol is:
  ///   { type: "command", id: "cmd_N", name: "commandName", payload: { ... } }
  ///
  /// The engine responds with:
  ///   { type: "response", id: "cmd_N", success: true/false, payload: { ... } }
  Future<Map<String, dynamic>> sendCommand(
    String name, [
    Map<String, dynamic>? payload,
  ]) async {
    final id = 'cmd_${_nextId++}';
    final command = {
      ProtocolKey.type: MessageType.command,
      ProtocolKey.id: id,
      ProtocolKey.name: name,
      ProtocolKey.payload: payload ?? {},
    };

    final completer = Completer<Map<String, dynamic>>();
    _pendingCommands[id] = completer;

    /// Record the send time and command name so the round-trip can be
    /// measured and labeled when the correlated response arrives.
    _commandSentAt[id] = DateTime.now();
    _commandNames[id] = name;

    final jsonStr = jsonEncode(command);
    _addLog(LogDirection.sent, jsonStr);

    /// Escape the JSON string for safe embedding in a JavaScript string literal.
    /// Single quotes in the JSON could break the JS string delimiter.
    final escapedJson = _escapeForJavaScript(jsonStr);

    try {
      await _controller.runJavaScript(
        "TiptapEngine.handleCommand('$escapedJson')",
      );
    } catch (e, stackTrace) {
      _pendingCommands.remove(id);
      _commandSentAt.remove(id);
      _commandNames.remove(id);
      _addLog(
        LogDirection.error,
        'Failed to send command "$name" (id: $id): $e\n'
        'Stack trace:\n$stackTrace',
      );
      rethrow;
    }

    /// Set a timeout so commands don't hang forever if the engine doesn't respond.
    return completer.future.timeout(
      const Duration(seconds: 10),
      onTimeout: () {
        _pendingCommands.remove(id);
        _commandSentAt.remove(id);
        _commandNames.remove(id);
        _addLog(
          LogDirection.error,
          'Command "$name" (id: $id) timed out after 10 seconds',
        );
        throw TimeoutException(
          'Command "$name" (id: $id) timed out',
          const Duration(seconds: 10),
        );
      },
    );
  }

  // ---------------------------------------------------------------------------
  // Lifecycle commands
  // ---------------------------------------------------------------------------

  /// Send the init command to create a Tiptap editor instance in the engine.
  ///
  /// This should be called after the engineGlobalReady message is received,
  /// confirming that TiptapEngine.handleCommand exists in the WebView.
  /// The engine will respond with schemaReady, ready, and stateChanged events.
  ///
  /// [content] can be an HTML string or a Tiptap JSON document object.
  /// [editable] controls whether the editor starts in editable mode.
  Future<Map<String, dynamic>> initEditor({
    String? content,
    bool editable = true,
  }) async {
    _addLog(
      LogDirection.system,
      'Sending init command '
      '(content length: ${content?.length ?? 0}, editable: $editable)',
    );
    return sendCommand(CommandName.init, {
      if (content != null) ProtocolKey.content: content,
      ProtocolKey.editable: editable,
    });
  }

  /// Tear down the editor instance and clean up all resources inside the engine.
  /// After this, the engine accepts a new init command.
  Future<Map<String, dynamic>> destroyEditor() async {
    _addLog(LogDirection.system, 'Sending destroy command');
    return sendCommand(CommandName.destroy);
  }

  /// Toggle the editor's read-only mode.
  Future<Map<String, dynamic>> setEditable(bool editable) async {
    _addLog(LogDirection.system, 'Setting editable: $editable');
    return sendCommand(CommandName.setEditable, {
      ProtocolKey.editable: editable,
    });
  }

  // ---------------------------------------------------------------------------
  // Content commands
  // ---------------------------------------------------------------------------

  /// Replace the entire document content with new HTML or JSON.
  ///
  /// [content] can be an HTML string or a Tiptap JSON document object.
  /// [emitUpdate] controls whether a stateChanged event is emitted (default true).
  Future<Map<String, dynamic>> setContent(
    String content, {
    bool emitUpdate = true,
  }) async {
    _addLog(LogDirection.system, 'Setting content (${content.length} chars)');
    return sendCommand(CommandName.setContent, {
      ProtocolKey.content: content,
      ProtocolKey.emitUpdate: emitUpdate,
    });
  }

  /// Retrieve the current document content as HTML.
  Future<Map<String, dynamic>> getHTML() async {
    _addLog(LogDirection.system, 'Requesting HTML content');
    return sendCommand(CommandName.getContent, {
      ProtocolKey.format: ContentFormat.html,
    });
  }

  /// Retrieve the current document content as plain text.
  Future<Map<String, dynamic>> getText() async {
    _addLog(LogDirection.system, 'Requesting plain text content');
    return sendCommand(CommandName.getContent, {
      ProtocolKey.format: ContentFormat.text,
    });
  }

  /// Retrieve the current document content as JSON.
  Future<Map<String, dynamic>> getJSON() async {
    _addLog(LogDirection.system, 'Requesting JSON content');
    return sendCommand(CommandName.getContent, {
      ProtocolKey.format: ContentFormat.json,
    });
  }

  /// Insert content at a specific position or range.
  ///
  /// [position] is either an integer position or a map with "from" and "to" keys
  /// defining a range to replace.
  /// [content] can be an HTML string, plain text, or a JSON document fragment.
  Future<Map<String, dynamic>> insertContentAt(
    dynamic position,
    String content,
  ) async {
    _addLog(LogDirection.system, 'Inserting content at position: $position');
    return sendCommand(CommandName.insertContentAt, {
      ProtocolKey.position: position,
      ProtocolKey.content: content,
    });
  }

  // ---------------------------------------------------------------------------
  // Text input commands
  // ---------------------------------------------------------------------------

  /// Insert text at the current selection or a given range.
  ///
  /// This is the primary command for committed keystrokes from the native
  /// input system. When [range] is provided, the text replaces that range
  /// (useful after IME composition commit).
  Future<Map<String, dynamic>> insertText(
    String text, {
    Map<String, int>? range,
  }) async {
    _addLog(
      LogDirection.system,
      'Inserting text: "${text.length > 50 ? '${text.substring(0, 50)}...' : text}"'
      '${range != null ? ' at range $range' : ''}',
    );
    return sendCommand(CommandName.insertText, {
      ProtocolKey.text: text,
      if (range != null) ProtocolKey.range: range,
    });
  }

  /// Delete content in a range or at the cursor (backspace behavior).
  ///
  /// When [range] is omitted, performs a backspace operation at the current
  /// cursor position. When provided, deletes the content in the specified range.
  Future<Map<String, dynamic>> deleteRange({Map<String, int>? range}) async {
    _addLog(
      LogDirection.system,
      'Deleting${range != null ? ' range $range' : ' at cursor (backspace)'}',
    );
    return sendCommand(CommandName.deleteRange, {
      if (range != null) ProtocolKey.range: range,
    });
  }

  /// Perform a backspace operation at the current cursor position.
  ///
  /// This delegates to ProseMirror's full backspace keybinding chain, which
  /// handles structural operations like joining blocks, lifting list items,
  /// and deleting atomic nodes — not just single character deletion.
  Future<Map<String, dynamic>> backspace() async {
    _addLog(LogDirection.system, 'Backspace');
    return sendCommand(CommandName.backspace);
  }

  /// Perform an Enter/newline operation at the current cursor position.
  ///
  /// This delegates to Tiptap's full Enter keybinding chain, which handles
  /// context-specific behavior like splitting paragraphs, creating new list
  /// items, exiting code blocks, and splitting blockquotes.
  Future<Map<String, dynamic>> enter() async {
    _addLog(LogDirection.system, 'Enter');
    return sendCommand(CommandName.enter);
  }

  // ---------------------------------------------------------------------------
  // Generic execution
  // ---------------------------------------------------------------------------

  /// Execute a named command on the editor (e.g., toggleBold, setHeading).
  ///
  /// This is the gateway for all formatting, structural, and utility commands.
  /// The engine calls editor.chain().focus()[commandName](args).run().
  Future<Map<String, dynamic>> execCommand(
    String commandName, [
    Map<String, dynamic>? args,
  ]) async {
    _addLog(
      LogDirection.system,
      'Executing command: $commandName'
      '${args != null ? " with args: $args" : ""}',
    );
    return sendCommand(CommandName.exec, {
      ProtocolKey.command: commandName,
      if (args != null) ProtocolKey.args: args,
    });
  }

  // ---------------------------------------------------------------------------
  // Selection commands
  // ---------------------------------------------------------------------------

  /// Set a cursor or text range selection.
  ///
  /// [anchor] is the fixed side of the selection. [head] is the moving side.
  /// If [head] is omitted, a collapsed cursor is placed at [anchor].
  Future<Map<String, dynamic>> setTextSelection(int anchor, {int? head}) async {
    _addLog(
      LogDirection.system,
      'Setting text selection: anchor=$anchor'
      '${head != null ? ', head=$head' : ''}',
    );
    return sendCommand(CommandName.setTextSelection, {
      ProtocolKey.anchor: anchor,
      if (head != null) ProtocolKey.head: head,
    });
  }

  /// Select an entire node at a position (e.g., an image or horizontal rule).
  Future<Map<String, dynamic>> setNodeSelection(int position) async {
    _addLog(
      LogDirection.system,
      'Setting node selection at position: $position',
    );
    return sendCommand(CommandName.setNodeSelection, {
      ProtocolKey.position: position,
    });
  }

  /// Select the entire document.
  Future<Map<String, dynamic>> selectAll() async {
    _addLog(LogDirection.system, 'Selecting all');
    return sendCommand(CommandName.selectAll);
  }

  /// Set logical focus on the editor.
  ///
  /// [position] can be "start", "end", "all", or an integer position.
  /// If omitted, focus is set at the current cursor position.
  Future<Map<String, dynamic>> focus({dynamic position}) async {
    _addLog(
      LogDirection.system,
      'Focusing${position != null ? ' at position: $position' : ''}',
    );
    return sendCommand(CommandName.focus, {
      if (position != null) ProtocolKey.position: position,
    });
  }

  /// Remove logical focus from the editor.
  Future<Map<String, dynamic>> blur() async {
    _addLog(LogDirection.system, 'Blurring');
    return sendCommand(CommandName.blur);
  }

  // ---------------------------------------------------------------------------
  // Query commands
  // ---------------------------------------------------------------------------

  /// Request a full state snapshot. Returns the same payload shape as the
  /// stateChanged event.
  Future<Map<String, dynamic>> getState() async {
    _addLog(LogDirection.system, 'Requesting state snapshot');
    return sendCommand(CommandName.getState);
  }

  /// Check if a mark or node type is active at the current selection.
  ///
  /// [name] is the mark or node type name (e.g., "bold", "heading").
  /// [attrs] is an optional attribute map for matching (e.g., { "level": 2 }).
  Future<Map<String, dynamic>> isActive(
    String name, {
    Map<String, dynamic>? attrs,
  }) async {
    _addLog(
      LogDirection.system,
      'Checking isActive: $name'
      '${attrs != null ? ' with attrs: $attrs' : ''}',
    );
    return sendCommand(CommandName.isActive, {
      ProtocolKey.name: name,
      if (attrs != null) ProtocolKey.attrs: attrs,
    });
  }

  /// Check if a command can execute in the current state.
  ///
  /// [command] is the command name (e.g., "toggleBold").
  /// [args] is optional arguments for the check.
  Future<Map<String, dynamic>> canExec(
    String command, {
    Map<String, dynamic>? args,
  }) async {
    _addLog(
      LogDirection.system,
      'Checking canExec: $command'
      '${args != null ? ' with args: $args' : ''}',
    );
    return sendCommand(CommandName.canExec, {
      ProtocolKey.command: command,
      if (args != null) ProtocolKey.args: args,
    });
  }

  /// Get attributes of a mark or node type at the current selection.
  ///
  /// [name] is the mark or node type name (e.g., "heading", "link").
  Future<Map<String, dynamic>> getAttributes(String name) async {
    _addLog(LogDirection.system, 'Getting attributes for: $name');
    return sendCommand(CommandName.getAttributes, {ProtocolKey.name: name});
  }

  // ---------------------------------------------------------------------------
  // Message handling
  // ---------------------------------------------------------------------------

  /// Handle an incoming message from the engine (response or event).
  ///
  /// Messages arrive as JSON strings through the JavaScript channel.
  /// The type field determines how they're routed:
  ///   - "response": matched to a pending command by ID
  ///   - "event": dispatched to the appropriate stream
  ///   - "bridgeAdapterReady": confirms the adapter script ran
  ///   - "engineGlobalReady": confirms TiptapEngine global exists
  ///   - "engineGlobalTimeout": TiptapEngine was not found after polling
  void _handleIncomingMessage(String rawMessage) {
    _addLog(LogDirection.received, rawMessage);

    Map<String, dynamic> data;
    try {
      data = jsonDecode(rawMessage) as Map<String, dynamic>;
    } catch (e) {
      _addLog(
        LogDirection.error,
        'Failed to parse incoming message as JSON: $e\n'
        'Raw message: $rawMessage',
      );
      return;
    }

    /// Broadcast the raw event data for the debug panel.
    _rawEventController.add(data);

    final type = data[ProtocolKey.type] as String?;

    switch (type) {
      case MessageType.response:
        _handleResponse(data);
        break;
      case MessageType.event:
        _handleEvent(data);
        break;
      case BridgeInternalMessage.bridgeAdapterReady:
        _addLog(LogDirection.system, 'Bridge adapter confirmed ready by JS');
        break;
      case BridgeInternalMessage.engineGlobalReady:
        final attempts = data['attempts'] ?? '?';
        final elapsed = data['elapsed'] ?? '?';
        final engineKeys = data['engineKeys'] ?? [];
        _addLog(
          LogDirection.system,
          'TiptapEngine global found after $attempts attempts '
          '(${elapsed}ms). Keys: $engineKeys',
        );
        _updateState(EngineState.engineGlobalReady);
        break;
      case BridgeInternalMessage.engineGlobalTimeout:
        final diagnostics = data['diagnostics'] ?? {};
        _addLog(
          LogDirection.error,
          'TiptapEngine global NOT found after polling. '
          'Diagnostics: ${jsonEncode(diagnostics)}',
        );
        _updateState(EngineState.error);
        _errorMessage =
            'Engine JS did not register TiptapEngine global. '
            'Diagnostics: ${jsonEncode(diagnostics)}';
        break;
      default:
        _addLog(
          LogDirection.warning,
          'Received message with unknown type: "$type". '
          'Full message: ${jsonEncode(data)}',
        );
    }
  }

  /// Route a response message to its matching pending command.
  void _handleResponse(Map<String, dynamic> data) {
    final id = data[ProtocolKey.id] as String?;
    if (id == null) {
      _addLog(
        LogDirection.warning,
        'Received response without an id field: ${jsonEncode(data)}',
      );
      return;
    }

    /// Record the round-trip duration if we have a recorded send time.
    /// This runs for both success and failure responses, so failed commands
    /// are measured too. The command name isn't carried on the response, so
    /// we recover it from the name recorded at send time.
    final sentAt = _commandSentAt.remove(id);
    final name = _commandNames.remove(id) ?? 'unknown';
    if (sentAt != null) {
      final ms = DateTime.now().difference(sentAt).inMicroseconds / 1000.0;
      metrics.recordRoundTrip(name, ms);
    }

    /// Record the engine-reported phase timings, if present. The `timings`
    /// field is a sibling of `payload` on the response, carrying at least the
    /// handle phase (the engine's total JavaScript-side time for the command).
    /// Absent when engine metrics are disabled, in which case nothing is
    /// recorded.
    _recordEngineTimings(data);

    final completer = _pendingCommands.remove(id);
    if (completer == null) {
      _addLog(
        LogDirection.warning,
        'Received response for unknown command id: $id '
        '(may have already timed out)',
      );
      return;
    }

    final success = data[ProtocolKey.success] as bool? ?? false;
    if (success) {
      _addLog(LogDirection.system, 'Command $id completed successfully');
      completer.complete(data);
    } else {
      /// Parse the structured error if present, otherwise fall back to
      /// the raw error/payload data.
      final errorJson = data[ProtocolKey.error] as Map<String, dynamic>?;
      final errorPayload = errorJson != null
          ? ErrorPayload.fromJson(errorJson)
          : ErrorPayload(
              code: ErrorCode.commandFailed,
              message: data[ProtocolKey.payload]?.toString() ?? 'Unknown error',
            );
      _addLog(LogDirection.error, 'Command $id failed: $errorPayload');
      completer.completeError(errorPayload);
    }
  }

  /// Dispatch an event message to the appropriate stream.
  void _handleEvent(Map<String, dynamic> data) {
    /// The engine uses "name" for the event name and "payload" for event data.
    final eventName = data[ProtocolKey.name] as String?;
    final eventData = data[ProtocolKey.payload] as Map<String, dynamic>? ?? {};

    _addLog(
      LogDirection.system,
      'Received event: "$eventName" '
      '(data keys: ${eventData.keys.join(", ")})',
    );

    switch (eventName) {
      case EventName.schemaReady:
        _schemaMetadata = SchemaMetadata.fromJson(eventData);
        _updateState(EngineState.schemaReady);
        _schemaReadyController.add(_schemaMetadata!);
        _addLog(
          LogDirection.system,
          'Schema ready — '
          '${_schemaMetadata!.nodes.length} nodes, '
          '${_schemaMetadata!.marks.length} marks, '
          '${_schemaMetadata!.commands.length} commands',
        );
        break;

      case EventName.ready:
        _updateState(EngineState.ready);
        _addLog(LogDirection.system, 'Engine is fully ready and operational');
        break;

      case EventName.stateChanged:

        /// Capture the causedBy correlation token (sibling of payload) before
        /// parsing the state, so the editor's typing-latency tracker can read
        /// it to pair a keystroke with the repaint this state produces. Null
        /// when the engine did not attribute this change to a single command.
        _lastCausedBy = data[ProtocolKey.causedBy] as String?;

        /// Record the engine-reported phase timings (sibling of payload),
        /// which on a stateChanged carry the full build breakdown
        /// (serializeDoc, commandStates, active, docDiff, total). This is the
        /// per-keystroke decomposition the instrumentation is for.
        _recordEngineTimings(data);

        _lastState = EditorStatePayload.fromJson(eventData);
        _stateChangedController.add(_lastState!);
        _addLog(
          LogDirection.system,
          'State changed — '
          'doc type: ${_lastState!.doc?.type ?? "null"}, '
          'selection: ${_lastState!.selection}, '
          'active marks: ${_lastState!.activeMarks}, '
          'active nodes: ${_lastState!.activeNodes.length}, '
          'command states: ${_lastState!.commandStates.length}'
          '${_lastCausedBy != null ? ', causedBy: $_lastCausedBy' : ''}',
        );
        break;

      /// contentChanged carries only the document tree, without selection
      /// or command state data. Merge with the existing state so we don't
      /// lose selection and command state information.
      case EventName.contentChanged:
        final partial = EditorStatePayload.fromJson(eventData);
        if (_lastState != null) {
          _lastState = EditorStatePayload(
            doc: partial.doc ?? _lastState!.doc,
            selection: _lastState!.selection,
            activeMarks: _lastState!.activeMarks,
            activeNodes: _lastState!.activeNodes,
            commandStates: _lastState!.commandStates,
            decorations: _lastState!.decorations,
            storedMarks: _lastState!.storedMarks,
            editable: _lastState!.editable,
          );
        } else {
          _lastState = partial;
        }
        _stateChangedController.add(_lastState!);
        _addLog(
          LogDirection.system,
          'Content changed event received (merged with existing state)',
        );
        break;

      /// selectionChanged carries selection, active marks/nodes, and command
      /// states, but omits the document tree. Merge with the existing state
      /// so we don't lose the document.
      case EventName.selectionChanged:
        final partial = EditorStatePayload.fromJson(eventData);
        if (_lastState != null) {
          _lastState = EditorStatePayload(
            doc: _lastState!.doc,
            selection: partial.selection ?? _lastState!.selection,
            activeMarks: partial.activeMarks.isNotEmpty
                ? partial.activeMarks
                : _lastState!.activeMarks,
            activeNodes: partial.activeNodes.isNotEmpty
                ? partial.activeNodes
                : _lastState!.activeNodes,
            commandStates: partial.commandStates.isNotEmpty
                ? partial.commandStates
                : _lastState!.commandStates,
            decorations: _lastState!.decorations,
            storedMarks: partial.storedMarks.isNotEmpty
                ? partial.storedMarks
                : _lastState!.storedMarks,
            editable: _lastState!.editable,
          );
        } else {
          _lastState = partial;
        }
        _stateChangedController.add(_lastState!);
        _addLog(
          LogDirection.system,
          'Selection changed event received (merged with existing state)',
        );
        break;

      case EventName.error:
        final errorPayload = ErrorPayload.fromJson(eventData);
        _errorEventController.add(errorPayload);
        _addLog(
          LogDirection.error,
          'Engine error event: ${errorPayload.code} — ${errorPayload.message}'
          '${errorPayload.commandId != null ? ' (command: ${errorPayload.commandId})' : ''}',
        );
        break;

      case EventName.extensionEvent:
        final extensionEvent = ExtensionEvent.fromJson(eventData);
        _extensionEventController.add(extensionEvent);
        _addLog(
          LogDirection.system,
          'Extension event from "${extensionEvent.extensionName}": '
          '"${extensionEvent.eventName}"',
        );
        break;

      default:
        _addLog(
          LogDirection.warning,
          'Received unhandled event type: "$eventName". '
          'Data: ${jsonEncode(eventData)}',
        );
    }
  }

  /// Parse the optional engine `timings` field from a response or stateChanged
  /// message and fold it into the per-phase metrics. The field is a sibling of
  /// `payload` (not nested inside it), so it is read from the top-level message
  /// map. Absent or empty timings record nothing — the engine omits the field
  /// for messages where no phase was timed (the initial-state emission, or any
  /// internal build path), so this is a no-op for those.
  void _recordEngineTimings(Map<String, dynamic> data) {
    final timingsJson = data[ProtocolKey.timings] as Map<String, dynamic>?;
    if (timingsJson == null) return;
    final timings = EngineTimings.fromJson(timingsJson);
    metrics.recordEngineTimings(timings);
  }

  // ---------------------------------------------------------------------------
  // Internal utilities
  // ---------------------------------------------------------------------------

  /// Update the engine state and notify listeners.
  ///
  /// Also records load-phase timings during cold start: the first transition
  /// into loading starts the clock, and each subsequent transition records
  /// the gap since the previous one as a named phase. Once the engine reaches
  /// ready, the total cold-start time is recorded. Phase recording stops after
  /// ready (totalLoadMs set), so post-startup transitions are not measured.
  void _updateState(EngineState newState) {
    final now = DateTime.now();

    /// Mark the start of cold-start timing on the first transition into
    /// loading, and record each subsequent phase as the gap between
    /// consecutive transitions until the engine is ready.
    if (newState == EngineState.loading && _loadStartedAt == null) {
      _loadStartedAt = now;
      _lastTransitionAt = now;
    } else if (_lastTransitionAt != null &&
        newState != _engineState &&
        _loadStartedAt != null &&
        metrics.totalLoadMs == null) {
      final phaseMs =
          now.difference(_lastTransitionAt!).inMicroseconds / 1000.0;
      metrics.recordLoadPhase('$_engineState -> $newState', phaseMs);
      _lastTransitionAt = now;

      /// Once the engine reaches ready, record total cold-start time.
      if (newState == EngineState.ready) {
        metrics.totalLoadMs =
            now.difference(_loadStartedAt!).inMicroseconds / 1000.0;
      }
    }

    final previousState = _engineState;
    _engineState = newState;
    _engineStateController.add(newState);
    _addLog(
      LogDirection.system,
      'Engine state transition: $previousState -> $newState',
    );
  }

  /// Add an entry to the debug log and print to console for easy sharing.
  void _addLog(String direction, String message) {
    final entry = BridgeLogEntry(
      timestamp: DateTime.now(),
      direction: direction,
      message: message,
    );
    _log.add(entry);
    _logController.add(entry);

    /// Print to console (visible in `flutter run` terminal / adb logcat)
    /// so logs can be easily copied and shared.
    final timeStr =
        '${entry.timestamp.hour.toString().padLeft(2, '0')}:'
        '${entry.timestamp.minute.toString().padLeft(2, '0')}:'
        '${entry.timestamp.second.toString().padLeft(2, '0')}.'
        '${entry.timestamp.millisecond.toString().padLeft(3, '0')}';
    // ignore: avoid_print
    print('[TiptapEngine][$timeStr][$direction] $message');
  }

  /// Escape a string for safe inclusion inside a JavaScript single-quoted
  /// string literal. Handles backslashes, single quotes, newlines, and
  /// carriage returns.
  String _escapeForJavaScript(String input) {
    return input
        .replaceAll('\\', '\\\\')
        .replaceAll("'", "\\'")
        .replaceAll('\n', '\\n')
        .replaceAll('\r', '\\r');
  }

  /// Clean up all resources. Cancels pending commands, closes streams,
  /// and marks the engine as destroyed.
  void dispose() {
    _addLog(
      LogDirection.system,
      'Bridge dispose() called — cleaning up resources',
    );

    /// Fail any pending commands that haven't received responses.
    final pendingCount = _pendingCommands.length;
    if (pendingCount > 0) {
      _addLog(
        LogDirection.warning,
        'Disposing with $pendingCount pending commands — '
        'they will be completed with errors',
      );
    }
    for (final entry in _pendingCommands.entries) {
      if (!entry.value.isCompleted) {
        entry.value.completeError(
          Exception('Bridge disposed while command ${entry.key} was pending'),
        );
      }
    }
    _pendingCommands.clear();
    _commandSentAt.clear();
    _commandNames.clear();

    _updateState(EngineState.destroyed);

    // Must run before the controllers below close — _addLog writes to
    // _logController, which throws "Cannot add new events after calling
    // close" if called afterward.
    _addLog(LogDirection.system, 'Bridge disposed successfully');

    _engineStateController.close();
    _schemaReadyController.close();
    _stateChangedController.close();
    _rawEventController.close();
    _errorEventController.close();
    _extensionEventController.close();
    _logController.close();
    _metricsController.close();
  }
}
