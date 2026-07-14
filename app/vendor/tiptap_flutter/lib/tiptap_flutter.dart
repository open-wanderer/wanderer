// Tiptap Flutter — A composable rich-text editor for Flutter powered by the
// Tiptap engine running inside a headless WebView.
//
// This library exposes a controller-based API inspired by Tiptap React:
//   - [EditorController] is the core — create one, initialize it, send commands,
//     listen to state streams.
//   - [TiptapEditor] is the content area widget that renders the document,
//     handles gestures, selection painting, and keyboard input.
//   - [TiptapToolbar] is a standalone formatting toolbar you can place anywhere.
//   - [TiptapPerformanceOverlay] is an opt-in performance metrics panel for
//     development.
//   - [NodeRendererRegistry] lets you register custom node type renderers.
//
// Minimal usage:
//   final controller = EditorController();
//   await controller.initialize(content: '<p>Hello</p>');
//
//   // In your widget tree:
//   Column(
//     children: [
//       TiptapToolbar(controller: controller),
//       Expanded(child: TiptapEditor(controller: controller)),
//     ],
//   )

// Protocol constants — wire-contract field names, command names, event names,
// content formats, and error codes shared with the engine, plus the
// rendering-layer node/mark/attribute vocabulary. Exported whole so app code
// can reference EditorCommand, ContentFormat, ErrorCode, MessageType, and the
// like without reaching into src/. The port-internal classes in this file
// (BridgeInternalMessage, LogDirection) are exposed as a harmless side effect
// of exporting the file whole; hiding them would require splitting the file
// or a show/hide clause, which costs more than it is worth.
export 'src/engine/protocol_constants.dart';

// Engine layer — protocol types and lifecycle state
export 'src/engine/protocol_types.dart';
export 'src/engine/tiptap_bridge.dart' show EngineState, BridgeLogEntry;
export 'src/engine/metrics.dart';

// Editor controller — the primary public API
export 'src/editor/editor_controller.dart';

// Composable widgets
export 'src/editor/tiptap_editor.dart';
export 'src/editor/tiptap_toolbar.dart' show TiptapToolbar, ImageInsertResult;
export 'src/editor/performance_overlay.dart';

// Rendering extensibility
export 'src/editor/rendering/node_renderer_registry.dart';

// Vendored patch (not upstream) — lets a consuming app drive rendering colors
// from its own theme. See tiptap_palette.dart for rationale.
export 'src/editor/rendering/tiptap_palette.dart';

// Rendering vocabulary — node type, mark type, and attribute key constants
// the renderer dispatches on (NodeType, MarkType, NodeAttr, MarkAttr).
// Exported so app code registering custom node builders can name the standard
// types without reaching into src/.
export 'src/editor/rendering/node_types.dart';
