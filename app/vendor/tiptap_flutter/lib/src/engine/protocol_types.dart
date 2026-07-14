// Protocol types for communication between the Flutter port and the Tiptap engine.
//
// These are simple data classes that mirror the TypeScript protocol types
// defined in the tiptap-engine package. They use hand-written fromJson
// factory constructors rather than code generation, keeping the project
// dependency-free and straightforward.
//
// The types here cover the full engine API surface as documented in the
// API reference, including schema introspection, editor state, selection,
// command states, marks, and error payloads.
//
// This file is a barrel. The type definitions are grouped by concern into
// four files, and this file re-exports them so every existing
// `import '.../protocol_types.dart'` keeps resolving to the same set of types
// with no consumer changes:
//
//   - marks_and_nodes.dart       — MarkData, AnnotatedNode
//   - selection_and_commands.dart — SelectionState, CommandState, ActiveNode
//   - schema_types.dart          — SchemaAttrInfo, SchemaNodeInfo,
//                                   SchemaMarkInfo, SchemaCommandInfo,
//                                   SchemaCommandArg, SchemaMetadata
//   - editor_state.dart          — EditorStatePayload, ErrorPayload,
//                                   ExtensionEvent

export 'marks_and_nodes.dart';
export 'selection_and_commands.dart';
export 'schema_types.dart';
export 'editor_state.dart';
