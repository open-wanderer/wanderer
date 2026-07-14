// Registry that maps ProseMirror node type names to Flutter widget builders.
//
// This is the extension point for custom node rendering. To support a new
// node type, register a builder function that takes an [AnnotatedNode],
// a recursive [childBuilder] callback, and an optional [PositionRegistry],
// and returns a Widget.
//
// The registry is pre-populated with builders for all standard Tiptap node
// types. Extension developers can add or override builders at runtime.

import 'package:flutter/widgets.dart';

import '../../engine/protocol_types.dart';
import '../selection/position_registry.dart';

/// Signature for a function that builds a widget for a given node.
///
/// [node] is the annotated node to render.
/// [childBuilder] is a callback that renders a child node — use it for
/// recursive descent into the document tree.
/// [registry] is the position registry for registering text blocks. It may
/// be null if position tracking is disabled.
typedef NodeWidgetBuilder =
    Widget Function(
      AnnotatedNode node,
      Widget Function(AnnotatedNode child) childBuilder,
      PositionRegistry? registry,
    );

/// A registry mapping node type names to their widget builders.
///
/// The [DocumentRenderer] consults this registry for every node in the tree.
/// Unknown node types are rendered as debug placeholders.
class NodeRendererRegistry {
  /// The singleton default registry, pre-populated with standard builders.
  static final NodeRendererRegistry defaultRegistry = NodeRendererRegistry();

  /// The internal map of node type names to their builders.
  final Map<String, NodeWidgetBuilder> _builders = {};

  /// Register a builder for a node type. Overwrites any existing builder
  /// for the same type name.
  void register(String nodeType, NodeWidgetBuilder builder) {
    _builders[nodeType] = builder;
  }

  /// Remove a builder for a node type.
  void unregister(String nodeType) {
    _builders.remove(nodeType);
  }

  /// Look up the builder for a node type. Returns null if no builder is
  /// registered for this type.
  NodeWidgetBuilder? builderFor(String nodeType) {
    return _builders[nodeType];
  }

  /// Whether a builder is registered for a node type.
  bool hasBuilder(String nodeType) {
    return _builders.containsKey(nodeType);
  }

  /// All registered node type names.
  Iterable<String> get registeredTypes => _builders.keys;
}
