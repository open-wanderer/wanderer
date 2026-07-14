// Typed result classes for the engine's query-command responses.
//
// The bridge's query methods (getHTML, getText, getJSON, isActive, canExec,
// getAttributes, exec) return the raw engine response as a
// Map<String, dynamic> of the shape:
//
//   { type: "response", id: "cmd_N", success: true, payload: { ... } }
//
// The useful data lives under the `payload` key. Each class here owns that
// unwrapping in a `fromResponse` factory that reads through `payload` using
// ProtocolKey constants and applies sensible defaults when a key is absent
// (empty string, empty map, false).
//
// The payload shapes are taken from the engine's query handlers
// (handleGetContent, handleIsActive, handleCanExec, handleGetAttributes,
// handleExec) and are authoritative against that source.
//
// Note on getContent: the engine returns a single `content` key whose runtime
// type depends on the requested format — a String for html/text and a JSON
// object for json. The three content classes below deliberately do NOT share
// a cast: HtmlContent and TextContent cast `content` to String, while
// JsonContent casts it to Map. Collapsing them into one class would force a
// single cast that is wrong for one of the two shapes.

import 'protocol_constants.dart';

/// Result of a getContent command requested in HTML format.
///
/// Reads the response's `payload[content]` as a String. The engine returns
/// the serialized HTML document under the `content` key for the `html` format.
class HtmlContent {
  /// The document content serialized as an HTML string.
  final String html;

  const HtmlContent({required this.html});

  /// Build from a raw bridge response, unwrapping `payload[content]` as a
  /// String and defaulting to the empty string when the key is absent.
  factory HtmlContent.fromResponse(Map<String, dynamic> response) {
    final payload = response[ProtocolKey.payload] as Map<String, dynamic>?;
    return HtmlContent(html: payload?[ProtocolKey.content] as String? ?? '');
  }

  @override
  String toString() => 'HtmlContent(${html.length} chars)';
}

/// Result of a getContent command requested in plain-text format.
///
/// Reads the response's `payload[content]` as a String. The engine returns
/// the flattened text under the `content` key for the `text` format.
class TextContent {
  /// The document content as flattened plain text.
  final String text;

  const TextContent({required this.text});

  /// Build from a raw bridge response, unwrapping `payload[content]` as a
  /// String and defaulting to the empty string when the key is absent.
  factory TextContent.fromResponse(Map<String, dynamic> response) {
    final payload = response[ProtocolKey.payload] as Map<String, dynamic>?;
    return TextContent(text: payload?[ProtocolKey.content] as String? ?? '');
  }

  @override
  String toString() => 'TextContent(${text.length} chars)';
}

/// Result of a getContent command requested in JSON format.
///
/// Reads the response's `payload[content]` as a Map. The engine returns the
/// Tiptap JSON document object under the `content` key for the `json` format.
/// This is the one content shape where `content` is an object rather than a
/// String, which is why this class casts to Map and the HTML/text classes
/// cast to String.
class JsonContent {
  /// The document content as a Tiptap JSON document object.
  final Map<String, dynamic> json;

  const JsonContent({required this.json});

  /// Build from a raw bridge response, unwrapping `payload[content]` as a
  /// Map and defaulting to an empty map when the key is absent.
  factory JsonContent.fromResponse(Map<String, dynamic> response) {
    final payload = response[ProtocolKey.payload] as Map<String, dynamic>?;
    return JsonContent(
      json: payload?[ProtocolKey.content] as Map<String, dynamic>? ?? {},
    );
  }

  @override
  String toString() => 'JsonContent(${json.keys.length} keys)';
}

/// Result of an isActive query.
///
/// Reads the response's `payload[active]` as a bool. The engine reports
/// whether the queried mark or node type is active at the current selection.
class IsActiveResult {
  /// Whether the queried mark or node type is active at the selection.
  final bool active;

  const IsActiveResult({required this.active});

  /// Build from a raw bridge response, unwrapping `payload[active]` as a bool
  /// and defaulting to false when the key is absent.
  factory IsActiveResult.fromResponse(Map<String, dynamic> response) {
    final payload = response[ProtocolKey.payload] as Map<String, dynamic>?;
    return IsActiveResult(
      active: payload?[ProtocolKey.active] as bool? ?? false,
    );
  }

  @override
  String toString() => 'IsActiveResult(active: $active)';
}

/// Result of a canExec query.
///
/// Reads the response's `payload[canExec]` as a bool. The engine reports
/// whether the queried command can execute in the current document state.
class CanExecResult {
  /// Whether the queried command can execute in the current state.
  final bool canExec;

  const CanExecResult({required this.canExec});

  /// Build from a raw bridge response, unwrapping `payload[canExec]` as a bool
  /// and defaulting to false when the key is absent.
  factory CanExecResult.fromResponse(Map<String, dynamic> response) {
    final payload = response[ProtocolKey.payload] as Map<String, dynamic>?;
    return CanExecResult(
      canExec: payload?[ProtocolKey.canExec] as bool? ?? false,
    );
  }

  @override
  String toString() => 'CanExecResult(canExec: $canExec)';
}

/// Result of a getAttributes query.
///
/// Reads the response's `payload[attrs]` as a Map. The engine reports the
/// attributes of the queried mark or node type at the current selection.
class AttributesResult {
  /// The attributes of the queried mark or node type at the selection.
  final Map<String, dynamic> attrs;

  const AttributesResult({required this.attrs});

  /// Build from a raw bridge response, unwrapping `payload[attrs]` as a Map
  /// and defaulting to an empty map when the key is absent.
  factory AttributesResult.fromResponse(Map<String, dynamic> response) {
    final payload = response[ProtocolKey.payload] as Map<String, dynamic>?;
    return AttributesResult(
      attrs: payload?[ProtocolKey.attrs] as Map<String, dynamic>? ?? {},
    );
  }

  @override
  String toString() => 'AttributesResult(${attrs.keys.length} attrs)';
}

/// Result of an exec command.
///
/// Reads the response's `payload[executed]` as a bool. The engine reports
/// whether the command actually ran (false for a no-op command that could
/// not apply in the current state).
///
/// This class is provided for completeness and for advanced callers parsing
/// raw bridge responses directly. The controller's execCommand does not use
/// it: execCommand returns `Future<void>` and discards the executed flag,
/// preserving its existing public signature. Wire this in only if a caller
/// (such as the toolbar) needs to react to no-op commands.
class ExecResult {
  /// Whether the command actually executed (false for a no-op).
  final bool executed;

  const ExecResult({required this.executed});

  /// Build from a raw bridge response, unwrapping `payload[executed]` as a
  /// bool and defaulting to false when the key is absent.
  factory ExecResult.fromResponse(Map<String, dynamic> response) {
    final payload = response[ProtocolKey.payload] as Map<String, dynamic>?;
    return ExecResult(
      executed: payload?[ProtocolKey.executed] as bool? ?? false,
    );
  }

  @override
  String toString() => 'ExecResult(executed: $executed)';
}
