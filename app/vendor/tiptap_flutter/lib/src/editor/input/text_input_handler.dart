// Manages the connection to the platform's text input system using the
// delta-based input model.
//
// This class implements [DeltaTextInputClient] to receive structured editing
// deltas from the platform's IME rather than full-state snapshots. The delta
// model eliminates the need for manual diffing and echo detection — each delta
// explicitly describes an insertion, deletion, replacement, or non-text update.
//
// The handler translates deltas into engine commands via three callbacks:
//   - onInsertText: for character insertions and the insertion part of replacements
//   - onDelete: for deletions and the deletion part of replacements
//   - onNewline: for Enter/Return actions and newline character insertions
//
// Replacements (from IME composition commits, autocorrect, etc.) are decomposed
// into a delete followed by an insert on the handler side, keeping the callback
// surface minimal and the editor widget free of delta-specific logic.

import 'package:flutter/services.dart';

/// Callback signatures for translating input events into engine commands.
typedef OnInsertText = void Function(String text);
typedef OnDeleteCount = void Function(int count);
typedef OnNewline = void Function();

/// Manages the platform text input connection for the editor using the
/// delta-based input model.
///
/// DeltaTextInputClient provides explicit deltas (insertion, deletion,
/// replacement, non-text update) instead of full editing state snapshots.
/// This eliminates the diffing and echo detection problems that plagued
/// the original TextInputClient implementation.
class TextInputHandler implements DeltaTextInputClient {
  final OnInsertText onInsertText;
  final OnDeleteCount onDelete;
  final OnNewline onNewline;

  TextInputHandler({
    required this.onInsertText,
    required this.onDelete,
    required this.onNewline,
  });

  TextInputConnection? _connection;
  bool get isAttached => _connection != null && _connection!.attached;

  /// The current editing value as known by the platform. Updated when we
  /// receive deltas or when we push state to the platform via syncState.
  /// This serves as our reference for the platform's current state, used
  /// only for syncState — not for diffing or echo detection.
  TextEditingValue _currentValue = TextEditingValue.empty;

  /// Attach to the platform text input system and show the keyboard.
  ///
  /// Configures the connection with delta model enabled, multiline input,
  /// and suggestions/autocorrect disabled to avoid unexpected replacements
  /// during structured editing.
  void attach() {
    if (isAttached) {
      _connection!.close();
    }

    _currentValue = TextEditingValue.empty;

    _connection = TextInput.attach(
      this,
      const TextInputConfiguration(
        inputType: TextInputType.multiline,
        inputAction: TextInputAction.newline,
        enableDeltaModel: true,
        autocorrect: false,
        enableSuggestions: false,
      ),
    );

    _connection!.setEditingState(_currentValue);
    _connection!.show();
  }

  /// Detach from the platform text input system and hide the keyboard.
  void detach() {
    _connection?.close();
    _connection = null;
    _currentValue = TextEditingValue.empty;
  }

  /// Sync the platform's editing state with the engine's current state.
  ///
  /// Called when the cursor moves due to a tap, drag-to-select, or an
  /// engine-initiated selection change — NOT during active typing. The
  /// delta model means we don't need to re-sync after every keystroke.
  ///
  /// [text] is the flattened text content of the block containing the cursor.
  /// [cursorOffset] is the cursor's position within that block text.
  /// [extentOffset], when provided, is the other end of a range selection
  /// within the same block text. The platform is then given a range
  /// selection (base at [cursorOffset], extent at [extentOffset]) instead
  /// of a collapsed cursor, so soft-keyboard delete/replace operates on the
  /// selected text. Cross-block selections cannot be represented in a single
  /// block's text — callers pass null for those and the platform falls back
  /// to a collapsed cursor at the base.
  void syncState(String text, int cursorOffset, {int? extentOffset}) {
    if (!isAttached) return;

    final clampedBase = cursorOffset.clamp(0, text.length);

    final TextSelection selection;
    if (extentOffset != null) {
      final clampedExtent = extentOffset.clamp(0, text.length);
      selection = TextSelection(
        baseOffset: clampedBase,
        extentOffset: clampedExtent,
      );
    } else {
      selection = TextSelection.collapsed(offset: clampedBase);
    }

    final value = TextEditingValue(text: text, selection: selection);

    _currentValue = value;
    _connection!.setEditingState(value);

    _log(
      'syncState: text="${_truncate(text)}" base=$clampedBase'
      '${extentOffset != null ? ' extent=$extentOffset' : ''}',
    );
  }

  // ---------------------------------------------------------------------------
  // DeltaTextInputClient implementation
  // ---------------------------------------------------------------------------

  /// Process editing deltas from the platform.
  ///
  /// Each delta explicitly describes what changed, eliminating the need for
  /// manual diffing or echo detection. Deltas are processed in order since
  /// the platform may batch multiple deltas in a single callback.
  @override
  void updateEditingValueWithDeltas(List<TextEditingDelta> deltas) {
    for (final delta in deltas) {
      if (delta is TextEditingDeltaInsertion) {
        _handleInsertion(delta);
      } else if (delta is TextEditingDeltaDeletion) {
        _handleDeletion(delta);
      } else if (delta is TextEditingDeltaReplacement) {
        _handleReplacement(delta);
      } else if (delta is TextEditingDeltaNonTextUpdate) {
        _handleNonTextUpdate(delta);
      }

      /// Update our reference to match the platform's state after each delta.
      _currentValue = delta.apply(_currentValue);
    }
  }

  /// Handle a text insertion delta.
  ///
  /// If the inserted text contains newlines, split it into text segments
  /// and newline commands. This handles paste operations that include
  /// line breaks and keyboards that send newlines as insertions rather
  /// than as performAction(TextInputAction.newline).
  void _handleInsertion(TextEditingDeltaInsertion delta) {
    final inserted = delta.textInserted;
    _log('delta insertion: "$inserted" at offset ${delta.insertionOffset}');

    if (inserted.contains('\n')) {
      _processTextWithNewlines(inserted);
    } else {
      onInsertText(inserted);
    }
  }

  /// Handle a text deletion delta.
  ///
  /// Computes the number of characters deleted from the delta's deleted
  /// range and forwards it to the onDelete callback.
  void _handleDeletion(TextEditingDeltaDeletion delta) {
    final count = delta.deletedRange.end - delta.deletedRange.start;
    _log('delta deletion: $count chars at range ${delta.deletedRange}');

    if (count > 0) {
      onDelete(count);
    }
  }

  /// Handle a replacement delta by decomposing it into delete + insert.
  ///
  /// Replacements come from IME composition commits, autocorrect, and
  /// predictive text. The engine doesn't have a replace command, so we
  /// decompose into two sequential operations: delete the old text, then
  /// insert the new text. Each becomes its own engine transaction.
  void _handleReplacement(TextEditingDeltaReplacement delta) {
    final deletedCount = delta.replacedRange.end - delta.replacedRange.start;
    final inserted = delta.replacementText;

    _log(
      'delta replacement: $deletedCount chars at ${delta.replacedRange} '
      '→ "$inserted"',
    );

    /// Delete the old text first, then insert the replacement.
    /// Both callbacks fire synchronously, but the engine processes them
    /// as separate async commands. The editor widget is responsible for
    /// sequencing these correctly (the first command's response arrives
    /// before the second command is sent, since the callbacks are wired
    /// to async engine methods).
    if (deletedCount > 0) {
      onDelete(deletedCount);
    }
    if (inserted.isNotEmpty) {
      if (inserted.contains('\n')) {
        _processTextWithNewlines(inserted);
      } else {
        onInsertText(inserted);
      }
    }
  }

  /// Handle a non-text update delta (cursor/selection movement only).
  ///
  /// The platform moved the cursor or changed the selection without
  /// modifying any text. No engine command is needed — we just update
  /// our internal reference. The editor widget handles cursor positioning
  /// through taps and the engine's selection commands.
  void _handleNonTextUpdate(TextEditingDeltaNonTextUpdate delta) {
    _log('delta non-text update: selection=${delta.selection}');
  }

  /// Split text containing newlines into alternating insert and newline
  /// operations. For example, "line1\nline2\n" becomes:
  /// onInsertText("line1"), onNewline(), onInsertText("line2"), onNewline()
  void _processTextWithNewlines(String text) {
    final parts = text.split('\n');
    for (var i = 0; i < parts.length; i++) {
      if (parts[i].isNotEmpty) {
        onInsertText(parts[i]);
      }
      if (i < parts.length - 1) {
        onNewline();
      }
    }
  }

  /// Called when the user presses a soft keyboard action button (e.g., Enter).
  ///
  /// Some keyboards send Enter as a TextInputAction rather than as an
  /// insertion delta. This ensures Enter works regardless of how the
  /// platform delivers it.
  @override
  void performAction(TextInputAction action) {
    _log('performAction: $action');
    if (action == TextInputAction.newline) {
      onNewline();
    }
  }

  // ---------------------------------------------------------------------------
  // Required DeltaTextInputClient / TextInputClient interface methods
  // ---------------------------------------------------------------------------

  @override
  TextEditingValue get currentTextEditingValue => _currentValue;

  @override
  AutofillScope? get currentAutofillScope => null;

  /// Legacy full-state update callback. With enableDeltaModel: true, the
  /// platform should use updateEditingValueWithDeltas instead. This is
  /// implemented as a no-op fallback in case the platform falls back to
  /// the legacy path on older devices.
  @override
  void updateEditingValue(TextEditingValue value) {
    _log(
      'updateEditingValue (legacy fallback): '
      'text="${_truncate(value.text)}" cursor=${value.selection.baseOffset}',
    );
    _currentValue = value;
  }

  @override
  void performPrivateCommand(String action, Map<String, dynamic> data) {}

  @override
  void showAutocorrectionPromptRect(int start, int end) {}

  @override
  void updateFloatingCursor(RawFloatingCursorPoint point) {}

  @override
  void connectionClosed() {
    _connection = null;
  }

  @override
  void didChangeInputControl(
    TextInputControl? oldControl,
    TextInputControl? newControl,
  ) {}

  @override
  void insertContent(KeyboardInsertedContent content) {}

  @override
  void insertTextPlaceholder(Size size) {}

  @override
  void removeTextPlaceholder() {}

  @override
  void performSelector(String selectorName) {}

  @override
  void showToolbar() {}

  @override
  bool onFocusReceived() {
    return true;
  }

  void dispose() {
    detach();
  }

  // ---------------------------------------------------------------------------
  // Debug logging
  // ---------------------------------------------------------------------------

  void _log(String message) {
    // ignore: avoid_print
    print('[TextInputHandler] $message');
  }

  String _truncate(String s) {
    if (s.length <= 40) return s;
    return '${s.substring(0, 40)}...';
  }
}
