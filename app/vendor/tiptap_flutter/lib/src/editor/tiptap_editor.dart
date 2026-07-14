// The core editor content widget that renders the document, handles gestures,
// paints selections, and manages keyboard input.
//
// This widget is the Flutter equivalent of Tiptap React's <EditorContent />.
// It is a composable building block — it does NOT include a toolbar, scaffold,
// app bar, or debug overlay. Developers compose those separately:
//
//   Column(
//     children: [
//       TiptapToolbar(controller: controller),
//       Expanded(child: TiptapEditor(controller: controller)),
//     ],
//   )
//
// The widget manages the position registry for tap-to-cursor, the selection
// overlay for cursor/highlight painting, and the text input handler for
// keyboard input. It also places the invisible WebView in the widget tree
// (required by webview_flutter for the controller to function).
//
// Two pieces of input-side logic live in their own files and are owned by this
// widget rather than implemented inline:
//   - block_text_extractor.dart: the pure document-tree walk that produces the
//     text and cursor offset for the block under the cursor (carries the +1
//     serializer compensation). Called from the syncState path.
//   - typing_latency_tracker.dart: the keystroke-to-repaint pairing that
//     measures end-to-end typing latency. The widget records a keystroke on
//     each input callback and asks the tracker to pair it on each repaint.
//
// Native text selection adds three more owned pieces:
//   - selection_text_extractor.dart: the pure document-tree slice that
//     produces the plain text of a selected range, for clipboard Copy/Cut.
//   - selection_overlay_controls.dart: the platform-native drag handles and
//     drag magnifier, rendered in this widget's overlay Stack.
//   - The local preview-selection pattern, implemented here: during a
//     selection gesture (long-press word select, handle drag) the widget
//     holds a transient SelectionState computed entirely from the position
//     registry and feeds it to the painter and handles, committing to the
//     engine only when the gesture ends. This is a deliberate, bounded
//     exception to "the engine is the only state holder" — scoped to the
//     lifetime of one gesture — so the highlight tracks the finger without
//     a per-frame engine round-trip. The engine remains authoritative: its
//     next stateChanged supersedes the preview.
//
// Selection chrome geometry timing:
// Handle and toolbar positions come from RenderParagraph text-layout queries
// through the position registry. Those queries are only valid AFTER the
// frame's layout pass — and because the document renderer creates fresh
// GlobalKeys (and therefore fresh RenderParagraphs) on every rebuild, they
// are NEVER valid during the build phase (querying then throws the
// !debugNeedsLayout assertion). The widget therefore computes a cached
// SelectionChromeGeometry in a post-frame callback and renders the chrome
// from the cache, calling setState only when the geometry actually changed
// so the update loop settles after one extra frame. The painter is
// unaffected (it reads layout at paint time, which is always after layout),
// and gesture callbacks are unaffected (they run between frames, after the
// previous frame's layout). The visible cost is the chrome lagging content
// by one frame.

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../engine/protocol_constants.dart';
import '../engine/protocol_types.dart';
import '../engine/tiptap_bridge.dart';
import 'editor_controller.dart';
import 'input/block_text_extractor.dart';
import 'input/text_input_handler.dart';
import 'input/typing_latency_tracker.dart';
import 'rendering/document_renderer.dart';
import 'rendering/tiptap_palette.dart';
import 'selection/position_registry.dart';
import 'selection/selection_overlay_controls.dart';
import 'selection/selection_painter.dart';
import 'selection/selection_text_extractor.dart';

/// The core editor content area that renders the document, handles gestures,
/// paints selections, and manages keyboard input.
///
/// Requires an [EditorController] that has been (or is being) initialized.
/// The widget handles all state transitions gracefully, showing appropriate
/// loading and error states.
///
/// This widget does not include a toolbar, scaffold, or debug overlay.
/// Compose those separately using [TiptapToolbar] and [PerformanceOverlay].
class TiptapEditor extends StatefulWidget {
  /// The editor controller that manages the bridge and editor state.
  final EditorController controller;

  /// Padding around the document content area.
  final EdgeInsets padding;

  /// Builder for a custom loading indicator. If null, a default
  /// [CircularProgressIndicator] is shown while the engine initializes.
  final WidgetBuilder? loadingBuilder;

  /// Builder for a custom error display. If null, a default error message
  /// with an icon is shown when the engine encounters an error.
  /// The builder receives the error message string.
  final Widget Function(BuildContext context, String? errorMessage)?
  errorBuilder;

  const TiptapEditor({
    super.key,
    required this.controller,
    this.padding = const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
    this.loadingBuilder,
    this.errorBuilder,
  });

  @override
  State<TiptapEditor> createState() => _TiptapEditorState();
}

class _TiptapEditorState extends State<TiptapEditor> {
  /// Subscriptions to controller streams, cancelled on dispose.
  final List<StreamSubscription> _subscriptions = [];

  /// Current engine state for determining what to render.
  EngineState _engineState = EngineState.uninitialized;

  /// Latest editor state for rendering the document.
  EditorStatePayload? _editorState;

  /// The position registry shared between the document renderer and the
  /// selection overlay. Populated on each document render, consumed by the
  /// selection painter for cursor/highlight positioning.
  final PositionRegistry _positionRegistry = PositionRegistry();

  /// Whether the editor currently has logical focus for cursor blinking
  /// and keyboard input.
  bool _hasFocus = false;

  /// The text input handler that manages the platform keyboard connection
  /// using the delta-based input model.
  late final TextInputHandler _inputHandler;

  /// Tracks end-to-end typing latency by pairing each keystroke with the
  /// repaint it produces. The widget records a keystroke on every input
  /// callback and asks the tracker to pair it on every state-driven repaint;
  /// the tracker forwards measured and dropped samples to the controller.
  late final TypingLatencyTracker _latencyTracker;

  /// Focus node for managing keyboard focus within Flutter's focus system.
  final FocusNode _focusNode = FocusNode();

  /// Flag indicating that a sync of the platform's editing state is needed.
  /// Set to true when the user taps to place the cursor, so that
  /// the next stateChanged event from the engine triggers a syncState call.
  /// This prevents syncState from firing after every keystroke, which would
  /// disrupt the platform's input state and cause off-by-one insertion bugs.
  ///
  /// The key insight: syncState (which calls setEditingState on the platform)
  /// should only happen when the cursor moved due to a user gesture (tap)
  /// or a non-typing engine action — never as a side-effect of typing. During
  /// typing, the platform tracks its own cursor position via deltas, and the
  /// engine tracks its own via the insertText command. Calling setEditingState
  /// between keystrokes resets the platform's internal state and causes the
  /// next delta to reference stale offsets.
  ///
  /// Selection gestures (long-press, handle drags) and clipboard operations
  /// also set this flag when they commit, for the same reason taps do: the
  /// platform needs to learn the new cursor/selection context.
  bool _syncNeeded = false;

  /// Flag set to true when the delta-based input handler processes a deletion
  /// or newline in the current frame. The hardware key event handler checks
  /// this flag to avoid double-processing the same keystroke. Reset at the
  /// end of each frame via a post-frame callback.
  bool _deltaHandledBackspace = false;
  bool _deltaHandledEnter = false;

  // ---------------------------------------------------------------------------
  // Selection gesture state
  // ---------------------------------------------------------------------------

  /// Local preview of an in-progress selection gesture. While non-null, this
  /// takes precedence over the engine's selection for painting and handle
  /// placement (see [_effectiveSelection]). It is built entirely from the
  /// position registry — no engine round-trips — and is discarded when the
  /// engine's stateChanged arrives after the gesture's commit.
  SelectionState? _previewSelection;

  /// Whether the Copy/Cut/Paste/Select All context toolbar is showing.
  bool _toolbarVisible = false;

  /// Which handle is being dragged: true for the start handle, false for
  /// the end handle, null when no handle drag is in progress.
  bool? _draggingStartHandle;

  /// The fixed (non-dragged) selection endpoint during a handle drag.
  int? _dragFixedPos;

  /// The moving (dragged) selection endpoint during a handle drag. Retains
  /// its last resolvable value when the finger passes over a point the
  /// registry cannot map to a position.
  int? _dragMovingPos;

  /// The magnifier's focal point in the overlay Stack's local coordinates,
  /// or null when no magnifier should be shown. Set during long-press and
  /// handle drags.
  Offset? _magnifierFocalPoint;

  /// Whether a long-press word-selection gesture is in progress.
  bool _longPressActive = false;

  /// The word selected at long-press start, kept as the gesture's anchor:
  /// dragging extends the selection word-by-word away from this word.
  WordRange? _longPressAnchorWord;

  /// Key on the overlay Stack, used to convert the registry's global pixel
  /// coordinates into the Stack's local space for positioning handles, the
  /// toolbar, and the magnifier.
  final GlobalKey _overlayStackKey = GlobalKey();

  /// Cached endpoint geometry for the selection chrome (handles, toolbar),
  /// computed post-frame because the position registry's text-layout queries
  /// are invalid during the build phase (see the file header). Null when
  /// there is no range selection or the geometry has not been computed yet.
  SelectionChromeGeometry? _chromeGeometry;

  /// Guard so at most one chrome-geometry post-frame callback is pending.
  bool _chromeGeometryUpdateScheduled = false;

  /// The selection to paint and decorate: the gesture preview while one is
  /// active, otherwise the engine's authoritative selection.
  SelectionState? get _effectiveSelection =>
      _previewSelection ?? _editorState?.selection;

  @override
  void initState() {
    super.initState();

    _inputHandler = TextInputHandler(
      onInsertText: _handleInsertText,
      onDelete: _handleDelete,
      onNewline: _handleNewline,
    );

    /// Construct the typing-latency tracker, wiring its sample and dropped
    /// callbacks to the controller's metric-recording methods. The tracker
    /// owns the pending-keystroke queue and the pairing rule; it schedules
    /// its own post-frame callback for the T1 timestamp, so nothing about it
    /// needs teardown in dispose().
    _latencyTracker = TypingLatencyTracker(
      onSample: (operation, ms, {required exact}) {
        widget.controller.recordTypingSample(operation, ms, exact: exact);
      },
      onDropped: () {
        widget.controller.recordDroppedTypingSample();
      },
    );

    _focusNode.addListener(_onFocusChanged);

    _subscribe();
  }

  @override
  void didUpdateWidget(TiptapEditor oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      _unsubscribe();
      _subscribe();
    }
  }

  void _subscribe() {
    /// Sync cached state from the controller in case it was initialized
    /// before this widget was built.
    _engineState = widget.controller.engineState;
    _editorState = widget.controller.editorState;

    _subscriptions.add(
      widget.controller.engineStateStream.listen((state) {
        setState(() {
          _engineState = state;
        });
      }),
    );

    _subscriptions.add(
      widget.controller.editorStateStream.listen((state) {
        setState(() {
          _editorState = state;

          /// The engine's state is authoritative. Any local preview
          /// selection left over from a committed gesture is superseded
          /// the moment the engine reports its post-commit state. The
          /// preview is kept alive only while a gesture is still in
          /// progress (handle drag or long-press), since mid-gesture
          /// engine updates must not snap the preview out from under
          /// the finger.
          if (_draggingStartHandle == null && !_longPressActive) {
            _previewSelection = null;
          }

          /// Hide the context toolbar and drop the cached chrome geometry
          /// when the selection collapses — typing, cut, paste, and tapping
          /// elsewhere all collapse it.
          if (state.selection == null || state.selection!.empty) {
            _toolbarVisible = false;
            _chromeGeometry = null;
          }
        });

        /// Pair this state-driven repaint with the oldest pending keystroke
        /// to measure end-to-end typing latency. The tracker schedules the
        /// post-frame callback so T1 is taken after the rebuild this state
        /// produced has actually painted.
        _latencyTracker.pairWithRepaint();

        /// Only sync the platform's text input state when a tap
        /// gesture initiated the cursor move. During typing, the platform
        /// and engine each track the cursor independently — calling
        /// setEditingState between keystrokes disrupts the platform's
        /// internal state and causes off-by-one errors and missed backspaces.
        if (_syncNeeded &&
            _hasFocus &&
            _inputHandler.isAttached &&
            state.selection != null) {
          _syncNeeded = false;
          _syncInputState(state);
        }
      }),
    );
  }

  void _unsubscribe() {
    for (final sub in _subscriptions) {
      sub.cancel();
    }
    _subscriptions.clear();
  }

  @override
  void dispose() {
    _unsubscribe();
    _inputHandler.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  // ---------------------------------------------------------------------------
  // Focus management
  // ---------------------------------------------------------------------------

  /// Called when the Flutter focus node's focus state changes.
  void _onFocusChanged() {
    if (!_focusNode.hasFocus && _hasFocus) {
      _blur();
    }
  }

  /// Give focus to the editor: show cursor, attach to keyboard.
  ///
  /// Always detaches and re-attaches the platform input connection on every
  /// focus request. This handles the case where the system dismissed the
  /// keyboard (swipe down, back button) without going through our [_blur]
  /// method — the old connection may be in a stale state where
  /// [_connection.show()] no longer brings up the keyboard. Creating a
  /// fresh connection on every tap is cheap and guarantees the keyboard
  /// appears reliably.
  void _gainFocus() {
    if (!_hasFocus) {
      setState(() {
        _hasFocus = true;
      });
    }

    _focusNode.requestFocus();

    /// Force a fresh connection to the platform input system. Detach first
    /// to close any stale connection, then attach to create a new one and
    /// show the keyboard.
    _inputHandler.detach();
    _inputHandler.attach();
  }

  /// Remove focus from the editor: hide cursor, detach from keyboard.
  void _blur() {
    if (!_hasFocus) return;

    setState(() {
      _hasFocus = false;
    });

    _inputHandler.detach();
  }

  // ---------------------------------------------------------------------------
  // Text input callbacks
  // ---------------------------------------------------------------------------

  /// Called when the user types text on the keyboard. With the delta-based
  /// input model, this receives the exact inserted text from the platform
  /// without any diffing or echo detection.
  void _handleInsertText(String text) {
    if (_engineState != EngineState.ready) return;

    /// Record the keystroke start time (T0) for typing-latency measurement.
    _latencyTracker.recordKeystroke('insert');

    /// Type-over-selection: when a range selection is active, pass the
    /// range explicitly so the engine replaces it. This avoids relying on
    /// the engine's default insertText-at-selection behavior — the ranged
    /// form is documented as replacing the given range, so it is the safe
    /// path regardless of how the engine treats a bare insertText with a
    /// non-empty selection.
    final sel = widget.controller.selection;
    if (sel != null && !sel.empty) {
      _syncNeeded = true;
      widget.controller.insertText(
        text,
        range: {ProtocolKey.from: sel.from, ProtocolKey.to: sel.to},
      );
    } else {
      widget.controller.insertText(text);
    }
  }

  /// Called when the user presses backspace. [count] is the number of
  /// characters deleted, derived from the deletion delta's range length.
  /// For single-character backspace, count is 1. For word-delete or
  /// selection-delete, count may be larger. Each deletion is sent as a
  /// separate backspace command to the engine, which runs the full
  /// ProseMirror backspace chain (handling structural operations like
  /// joining blocks and lifting list items).
  void _handleDelete(int count) {
    if (_engineState != EngineState.ready) return;

    /// Record the keystroke start time (T0) for typing-latency measurement.
    _latencyTracker.recordKeystroke('delete');

    /// Mark that the delta handler processed a backspace this frame,
    /// so the hardware key event handler won't double-process it.
    _markDeltaHandledBackspace();

    /// Selection delete: when a range selection is active, delete it with
    /// a single ranged deleteRange rather than per-character backspaces.
    /// This avoids relying on the backspace chain's
    /// delete-selection-on-backspace behavior, and is one round-trip
    /// instead of [count].
    final sel = widget.controller.selection;
    if (sel != null && !sel.empty) {
      _syncNeeded = true;
      widget.controller
          .deleteRange(
            range: {ProtocolKey.from: sel.from, ProtocolKey.to: sel.to},
          )
          .catchError((_) {});
      return;
    }

    Future<void> deleteSequence() async {
      for (var i = 0; i < count; i++) {
        await widget.controller.backspace();
      }
    }

    deleteSequence();
  }

  /// Called when the user presses Enter. With the delta-based model, this
  /// can arrive either as a newline insertion delta or as a
  /// performAction(TextInputAction.newline) — the handler normalizes both
  /// paths into this single callback.
  void _handleNewline() {
    if (_engineState != EngineState.ready) return;

    /// Record the keystroke start time (T0) for typing-latency measurement.
    _latencyTracker.recordKeystroke('newline');

    /// Mark that the delta handler processed an enter this frame,
    /// so the hardware key event handler won't double-process it.
    _markDeltaHandledEnter();

    widget.controller.enter();
  }

  /// Set the backspace-handled flag and schedule a reset at the end of
  /// the current frame. This prevents the hardware key event handler from
  /// sending a duplicate backspace command for the same keystroke.
  void _markDeltaHandledBackspace() {
    _deltaHandledBackspace = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _deltaHandledBackspace = false;
    });
  }

  /// Set the enter-handled flag and schedule a reset at the end of
  /// the current frame.
  void _markDeltaHandledEnter() {
    _deltaHandledEnter = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _deltaHandledEnter = false;
    });
  }

  // ---------------------------------------------------------------------------
  // Hardware keyboard fallback
  // ---------------------------------------------------------------------------

  /// Handle hardware key events as a fallback for backspace and enter.
  ///
  /// Some platforms and soft keyboards don't produce a deletion delta for
  /// backspace (e.g., when the platform buffer is empty at cursor position 0)
  /// or send Enter as a key event rather than a newline delta. This handler
  /// catches those cases.
  ///
  /// The delta-handled flags prevent double-processing when both a delta and
  /// a key event arrive for the same keystroke.
  KeyEventResult _handleKeyEvent(FocusNode node, KeyEvent event) {
    if (_engineState != EngineState.ready) return KeyEventResult.ignored;

    /// Only handle key-down and repeat events to avoid processing the same
    /// key twice (once on down, once on up).
    if (event is! KeyDownEvent && event is! KeyRepeatEvent) {
      return KeyEventResult.ignored;
    }

    if (event.logicalKey == LogicalKeyboardKey.backspace) {
      if (!_deltaHandledBackspace) {
        /// Same selection-delete logic as the delta path: a range selection
        /// is deleted with one ranged command rather than relying on the
        /// backspace chain.
        final sel = widget.controller.selection;
        if (sel != null && !sel.empty) {
          _syncNeeded = true;
          widget.controller
              .deleteRange(
                range: {ProtocolKey.from: sel.from, ProtocolKey.to: sel.to},
              )
              .catchError((e) {});
        } else {
          widget.controller.backspace().catchError((e) {});
        }
        return KeyEventResult.handled;
      }
      return KeyEventResult.ignored;
    }

    if (event.logicalKey == LogicalKeyboardKey.enter ||
        event.logicalKey == LogicalKeyboardKey.numpadEnter) {
      if (!_deltaHandledEnter) {
        widget.controller.enter().catchError((e) {});
        return KeyEventResult.handled;
      }
      return KeyEventResult.ignored;
    }

    return KeyEventResult.ignored;
  }

  // ---------------------------------------------------------------------------
  // Platform input state sync
  // ---------------------------------------------------------------------------

  /// Sync the platform's text input state with the engine's current state.
  ///
  /// Extracts the text of the block containing the cursor and maps the
  /// ProseMirror cursor position to a local offset within that block text.
  /// The platform then knows the real text around the cursor for word
  /// boundaries, autocorrect context, and deletion behavior.
  ///
  /// IMPORTANT: This is only called after tap gestures, never after
  /// typing. During typing, the platform tracks its own cursor via deltas
  /// and calling setEditingState would disrupt it.
  ///
  /// For a non-empty selection contained in a single block, both endpoints
  /// are synced so the platform sees the real range. Cross-block selections
  /// cannot be represented in one block's text; the platform gets a
  /// collapsed cursor at the selection start, and the editor's own input
  /// callbacks handle range replacement/deletion defensively regardless.
  void _syncInputState(EditorStatePayload state) {
    if (state.doc == null || state.selection == null) return;

    final selection = state.selection!;

    if (!selection.empty) {
      final rangeResult = extractBlockTextRange(
        state.doc!,
        selection.from,
        selection.to,
      );
      if (rangeResult != null) {
        _inputHandler.syncState(
          rangeResult.text,
          rangeResult.baseOffset,
          extentOffset: rangeResult.extentOffset,
        );
        return;
      }
    }

    final cursorPos = selection.from;
    final result = extractBlockText(state.doc!, cursorPos);
    if (result != null) {
      _inputHandler.syncState(result.text, result.cursorOffset);
    }
  }

  // ---------------------------------------------------------------------------
  // Gesture handling
  // ---------------------------------------------------------------------------

  /// Handle a single tap on the document area. Converts the tap position
  /// to a ProseMirror document position and places a collapsed cursor.
  void _onTapUp(TapUpDetails details) {
    if (_engineState != EngineState.ready) return;

    _gainFocus();

    /// A tap dismisses any active selection chrome: the context toolbar
    /// hides and a stale gesture preview is dropped. The engine's selection
    /// collapses through the setTextSelection below.
    if (_previewSelection != null || _toolbarVisible) {
      setState(() {
        _previewSelection = null;
        _toolbarVisible = false;
        _chromeGeometry = null;
      });
    }

    /// Use a post-frame callback to ensure the position registry has been
    /// populated by the current frame's layout pass before we query it.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final docPos = _positionRegistry.positionFromGlobalOffset(
        details.globalPosition,
      );

      if (docPos != null) {
        /// Empty blocks (e.g., an empty paragraph created by pressing Enter
        /// twice) use a zero-width space for rendering, which causes
        /// localOffsetToPos to compute a position between the block's opening
        /// and closing tokens (e.g., 148 for a block at pos:147, end:149).
        /// ProseMirror does not recognize that position as inside the
        /// paragraph's content — it reports activeNodes:[] and insertText
        /// at that position creates a new paragraph instead of filling the
        /// empty one. The correct cursor position for an empty block is
        /// the block's own pos value, which ProseMirror resolves as the
        /// content start of that paragraph. We detect empty blocks by
        /// finding a registered block near the computed position that has
        /// exactly one span mapping with zero length.
        final tappedBlock = _positionRegistry.blocks.where(
          (b) =>
              docPos >= b.pos - 1 &&
              docPos <= b.end + 1 &&
              b.spanMappings.length == 1 &&
              b.spanMappings.first.length == 0,
        );

        int positionToSend = docPos;
        if (tappedBlock.isNotEmpty) {
          positionToSend = tappedBlock.first.pos;
        }

        /// Mark that the next stateChanged event should trigger a sync
        /// of the platform's editing state. This tells the platform what
        /// text surrounds the new cursor position so it can handle
        /// backspace, word boundaries, and autocorrect correctly.
        _syncNeeded = true;
        widget.controller.setTextSelection(positionToSend);
      }
    });
  }

  // ---------------------------------------------------------------------------
  // Long-press word selection
  // ---------------------------------------------------------------------------

  /// Begin a long-press word selection: select the word under the finger
  /// as a local preview, show the magnifier, and keep the keyboard up.
  void _onLongPressStart(LongPressStartDetails details) {
    if (_engineState != EngineState.ready) return;

    _gainFocus();

    final word = _positionRegistry.wordRangeAtGlobalOffset(
      details.globalPosition,
    );
    if (word == null) return;

    setState(() {
      _longPressActive = true;
      _longPressAnchorWord = word;
      _previewSelection = _buildPreviewSelection(word.from, word.to);
      _magnifierFocalPoint = _globalToOverlayLocal(details.globalPosition);
      _toolbarVisible = false;
    });
  }

  /// Extend the long-press selection word-by-word as the finger drags.
  /// The word selected at gesture start stays anchored; the selection grows
  /// to cover the word currently under the finger, with the head placed on
  /// the finger's side so handle semantics stay natural after commit.
  void _onLongPressMoveUpdate(LongPressMoveUpdateDetails details) {
    if (!_longPressActive) return;

    final word = _positionRegistry.wordRangeAtGlobalOffset(
      details.globalPosition,
    );

    setState(() {
      _magnifierFocalPoint = _globalToOverlayLocal(details.globalPosition);

      if (word != null && _longPressAnchorWord != null) {
        final anchorWord = _longPressAnchorWord!;
        if (word.from < anchorWord.from) {
          /// Dragging backward: anchor at the end of the start word,
          /// head at the start of the word under the finger.
          _previewSelection = _buildPreviewSelection(anchorWord.to, word.from);
        } else {
          /// Dragging forward (or within the start word): anchor at the
          /// start of the start word, head at the end of the farther word.
          final headPos = word.to > anchorWord.to ? word.to : anchorWord.to;
          _previewSelection = _buildPreviewSelection(anchorWord.from, headPos);
        }
      }
    });
  }

  /// End the long-press gesture: hide the magnifier, commit the previewed
  /// selection to the engine, and show the context toolbar for a non-empty
  /// result. A collapsed result (long-press on an empty block) commits a
  /// collapsed cursor with no toolbar.
  void _onLongPressEnd(LongPressEndDetails details) {
    if (!_longPressActive) return;

    final sel = _previewSelection;

    setState(() {
      _longPressActive = false;
      _longPressAnchorWord = null;
      _magnifierFocalPoint = null;
      _toolbarVisible = sel != null && !sel.empty;
    });

    if (sel != null) {
      _commitSelection(sel.anchor, sel.head);
    }
  }

  // ---------------------------------------------------------------------------
  // Handle dragging
  // ---------------------------------------------------------------------------

  /// Begin dragging a selection handle. The opposite endpoint becomes the
  /// fixed anchor for the duration of the drag; the context toolbar hides
  /// and the magnifier appears.
  void _onHandleDragStart(bool isStartHandle, Offset globalPosition) {
    final sel = _effectiveSelection;
    if (sel == null || sel.empty) return;

    setState(() {
      _draggingStartHandle = isStartHandle;
      _dragFixedPos = isStartHandle ? sel.to : sel.from;
      _dragMovingPos = isStartHandle ? sel.from : sel.to;
      _toolbarVisible = false;
      _magnifierFocalPoint = _globalToOverlayLocal(globalPosition);
    });
  }

  /// Update the dragged handle's endpoint from the finger position. The
  /// registry resolves the finger to a document position; points it cannot
  /// resolve (gaps between blocks mid-drag) keep the last good position
  /// rather than jumping, so the preview never lands on a position that
  /// did not come from a span mapping.
  void _onHandleDragUpdate(bool isStartHandle, Offset globalPosition) {
    if (_draggingStartHandle == null) return;

    final pos = _positionRegistry.positionFromGlobalOffset(globalPosition);

    setState(() {
      _magnifierFocalPoint = _globalToOverlayLocal(globalPosition);
      if (pos != null && _dragFixedPos != null) {
        _dragMovingPos = pos;
        _previewSelection = _buildPreviewSelection(_dragFixedPos!, pos);
      }
    });
  }

  /// End the handle drag: hide the magnifier, commit the previewed range to
  /// the engine, and re-show the context toolbar at the new selection.
  void _onHandleDragEnd(bool isStartHandle) {
    if (_draggingStartHandle == null) return;

    final fixed = _dragFixedPos;
    final moving = _dragMovingPos;

    setState(() {
      _draggingStartHandle = null;
      _dragFixedPos = null;
      _dragMovingPos = null;
      _magnifierFocalPoint = null;
      _toolbarVisible = fixed != null && moving != null && fixed != moving;
    });

    if (fixed != null && moving != null) {
      _commitSelection(fixed, moving);
    }
  }

  // ---------------------------------------------------------------------------
  // Selection helpers
  // ---------------------------------------------------------------------------

  /// Build a local preview SelectionState from an anchor and head position.
  /// Mirrors the engine's selection shape: from/to are the normalized
  /// min/max, anchor/head preserve gesture direction.
  SelectionState _buildPreviewSelection(int anchor, int head) {
    final from = anchor < head ? anchor : head;
    final to = anchor < head ? head : anchor;
    return SelectionState(
      type: 'text',
      anchor: anchor,
      head: head,
      from: from,
      to: to,
      empty: from == to,
    );
  }

  /// Commit a selection to the engine and request a platform input sync on
  /// the resulting stateChanged. Errors are swallowed: a rejected selection
  /// leaves the engine's previous selection in place, and the preview is
  /// discarded when its stateChanged-equivalent arrives or on the next
  /// gesture.
  void _commitSelection(int anchor, int head) {
    _syncNeeded = true;
    if (anchor == head) {
      widget.controller.setTextSelection(anchor).catchError((_) {});
    } else {
      widget.controller.setTextSelection(anchor, head: head).catchError((_) {});
    }
  }

  /// Convert a global pixel offset into the overlay Stack's local space.
  /// Falls back to the global offset if the Stack has not been laid out.
  Offset _globalToOverlayLocal(Offset global) {
    final renderObject = _overlayStackKey.currentContext?.findRenderObject();
    if (renderObject is RenderBox && renderObject.attached) {
      return renderObject.globalToLocal(global);
    }
    return global;
  }

  // ---------------------------------------------------------------------------
  // Selection chrome geometry
  // ---------------------------------------------------------------------------

  /// Schedule a post-frame recomputation of the selection chrome geometry.
  ///
  /// The geometry must be computed after layout: the registry's pixel
  /// lookups go through RenderParagraph text-layout queries, and the
  /// document renderer's per-build GlobalKeys mean those RenderParagraphs
  /// are freshly created — and thus not yet laid out — during every build
  /// phase. Querying them from build throws !debugNeedsLayout.
  ///
  /// setState is only called when the recomputed geometry differs from the
  /// cached one, so the build → post-frame → setState cycle converges after
  /// one extra frame rather than rebuilding forever.
  void _scheduleChromeGeometryUpdate() {
    if (_chromeGeometryUpdateScheduled) return;
    _chromeGeometryUpdateScheduled = true;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      _chromeGeometryUpdateScheduled = false;
      if (!mounted) return;

      final updated = _computeChromeGeometry();
      if (updated != _chromeGeometry) {
        setState(() {
          _chromeGeometry = updated;
        });
      }
    });
  }

  /// Compute the current selection's chrome geometry from the registry.
  /// Only safe to call after the frame's layout pass (post-frame callback
  /// or gesture handler). Returns null when there is no range selection or
  /// the endpoints cannot be resolved to laid-out blocks.
  SelectionChromeGeometry? _computeChromeGeometry() {
    final sel = _effectiveSelection;
    if (sel == null || sel.empty) return null;

    final startTopGlobal = _positionRegistry.globalOffsetFromPosition(sel.from);
    final endTopGlobal = _positionRegistry.globalOffsetFromPosition(sel.to);
    if (startTopGlobal == null || endTopGlobal == null) return null;

    final startHeight =
        _positionRegistry.caretHeightAtPosition(sel.from) ?? 20.0;
    final endHeight = _positionRegistry.caretHeightAtPosition(sel.to) ?? 20.0;

    return SelectionChromeGeometry(
      startTop: _globalToOverlayLocal(startTopGlobal),
      startCaretHeight: startHeight,
      endTop: _globalToOverlayLocal(endTopGlobal),
      endCaretHeight: endHeight,
    );
  }

  // ---------------------------------------------------------------------------
  // Clipboard actions
  // ---------------------------------------------------------------------------

  /// Copy the selected text to the system clipboard. Plain text only: the
  /// text is extracted locally from the annotated document tree, so no
  /// engine round-trip is needed. Rich (HTML/JSON) clipboard is a follow-up
  /// that requires an engine-defined ranged content-extraction message.
  Future<void> _copySelection() async {
    final sel = _effectiveSelection;
    final doc = _editorState?.doc;
    if (sel == null || sel.empty || doc == null) return;

    final text = extractTextInRange(doc, sel.from, sel.to);
    await Clipboard.setData(ClipboardData(text: text));

    /// Copy keeps the selection but dismisses the toolbar, matching
    /// platform behavior.
    if (mounted) {
      setState(() {
        _toolbarVisible = false;
      });
    }
  }

  /// Cut: copy the selected text, then delete the range with a single
  /// ranged deleteRange command.
  Future<void> _cutSelection() async {
    final sel = _effectiveSelection;
    final doc = _editorState?.doc;
    if (sel == null || sel.empty || doc == null) return;

    final text = extractTextInRange(doc, sel.from, sel.to);
    await Clipboard.setData(ClipboardData(text: text));

    if (mounted) {
      setState(() {
        _toolbarVisible = false;
      });
    }

    _syncNeeded = true;
    await widget.controller.deleteRange(
      range: {ProtocolKey.from: sel.from, ProtocolKey.to: sel.to},
    );
  }

  /// Paste the system clipboard's plain text at the selection.
  ///
  /// Single-line text with an active range selection uses insertText's
  /// ranged form, which replaces the range in one command. Multi-line text
  /// replays the keyboard path — alternating insertText and enter commands,
  /// the same decomposition the input handler uses for typed/pasted
  /// newlines — after clearing any selected range, so paste exercises only
  /// command semantics the engine already defines.
  Future<void> _pasteClipboard() async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    final text = data?.text;
    if (text == null || text.isEmpty) return;

    final sel = _effectiveSelection;

    if (mounted) {
      setState(() {
        _toolbarVisible = false;
      });
    }

    _syncNeeded = true;

    if (!text.contains('\n')) {
      if (sel != null && !sel.empty) {
        await widget.controller.insertText(
          text,
          range: {ProtocolKey.from: sel.from, ProtocolKey.to: sel.to},
        );
      } else {
        await widget.controller.insertText(text);
      }
      return;
    }

    /// Multi-line paste: clear the selected range first, then insert each
    /// line followed by an enter, sequenced with awaits so the engine
    /// processes them in order.
    if (sel != null && !sel.empty) {
      await widget.controller.deleteRange(
        range: {ProtocolKey.from: sel.from, ProtocolKey.to: sel.to},
      );
    }

    final parts = text.split('\n');
    for (var i = 0; i < parts.length; i++) {
      if (parts[i].isNotEmpty) {
        await widget.controller.insertText(parts[i]);
      }
      if (i < parts.length - 1) {
        await widget.controller.enter();
      }
    }
  }

  /// Select the entire document via the engine's selectAll command and
  /// keep the toolbar up at the new selection.
  Future<void> _selectAllInEditor() async {
    _syncNeeded = true;
    await widget.controller.selectAll();
    if (mounted) {
      setState(() {
        _toolbarVisible = true;
      });
    }
  }

  // ---------------------------------------------------------------------------
  // Build
  // ---------------------------------------------------------------------------

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        /// The invisible WebView — must be in the widget tree for the
        /// webview_flutter controller to function. Wrapped in a zero-height
        /// Offstage so it takes no space in the layout.
        Offstage(
          offstage: true,
          child: SizedBox(
            width: 1,
            height: 1,
            child: widget.controller.webViewWidget,
          ),
        ),

        /// The editor content area with gesture detection and selection overlay.
        Expanded(
          child: Focus(
            focusNode: _focusNode,
            onKeyEvent: _handleKeyEvent,
            child: _buildContent(),
          ),
        ),
      ],
    );
  }

  /// Build the main content area based on the current engine state.
  Widget _buildContent() {
    if (_engineState == EngineState.error) {
      if (widget.errorBuilder != null) {
        return widget.errorBuilder!(context, widget.controller.errorMessage);
      }
      return _buildDefaultError();
    }

    if (_engineState != EngineState.ready || _editorState?.doc == null) {
      if (widget.loadingBuilder != null) {
        return widget.loadingBuilder!(context);
      }
      return const Center(child: CircularProgressIndicator());
    }

    final effectiveSelection = _effectiveSelection;
    final hasRangeSelection =
        effectiveSelection != null && !effectiveSelection.empty;

    /// Keep the cached chrome geometry in sync with the selection. The
    /// computation itself happens post-frame (after layout); rendering
    /// below uses only the cached value, never live registry queries.
    if (hasRangeSelection) {
      _scheduleChromeGeometryUpdate();
    }

    return GestureDetector(
      onTapUp: _onTapUp,
      onLongPressStart: _onLongPressStart,
      onLongPressMoveUpdate: _onLongPressMoveUpdate,
      onLongPressEnd: _onLongPressEnd,
      behavior: HitTestBehavior.translucent,
      child: NotificationListener<ScrollNotification>(
        /// Reposition the selection chrome (handles, toolbar) while the
        /// document scrolls under it: schedule a post-frame geometry
        /// recomputation, which setStates only if positions actually moved.
        onNotification: (_) {
          if (hasRangeSelection) {
            _scheduleChromeGeometryUpdate();
          }
          return false;
        },
        child: Stack(
          key: _overlayStackKey,
          children: [
            /// The rendered document.
            SingleChildScrollView(
              padding: widget.padding,
              child: DocumentRenderer(
                doc: _editorState!.doc!,
                positionRegistry: _positionRegistry,
              ),
            ),

            /// The selection overlay (cursor and highlight painting). It
            /// paints the effective selection, so an in-progress gesture's
            /// local preview is highlighted without an engine round-trip.
            Positioned.fill(
              child: IgnorePointer(
                child: EditorSelectionOverlay(
                  selection: effectiveSelection,
                  registry: _positionRegistry,
                  hasFocus: _hasFocus,
                  cursorColor: TiptapPalette.cursor,
                  selectionColor: TiptapPalette.selectionHighlight,
                ),
              ),
            ),

            /// Native drag handles at the endpoints of a range selection,
            /// rendered from the post-frame cached geometry.
            if (hasRangeSelection && _chromeGeometry != null)
              EditorSelectionHandles(
                geometry: _chromeGeometry!,
                onDragStart: _onHandleDragStart,
                onDragUpdate: _onHandleDragUpdate,
                onDragEnd: _onHandleDragEnd,
              ),

            /// The Copy/Cut/Paste/Select All context toolbar, anchored to
            /// the post-frame cached geometry.
            if (_toolbarVisible && hasRangeSelection && _chromeGeometry != null)
              _buildContextToolbar(_chromeGeometry!),

            /// The drag magnifier, shown during long-press and handle drags.
            if (_magnifierFocalPoint != null)
              EditorDragMagnifier(focalPoint: _magnifierFocalPoint!),
          ],
        ),
      ),
    );
  }

  /// Build the platform-native context toolbar anchored to the selection's
  /// cached chrome geometry.
  ///
  /// Uses AdaptiveTextSelectionToolbar so the visuals, labels, and overflow
  /// behavior match the platform. Anchors are the top of the selection's
  /// first caret (primary) and the bottom of its last caret (secondary), in
  /// the overlay Stack's local space — the toolbar positions itself above
  /// the selection when there is room, below otherwise.
  Widget _buildContextToolbar(SelectionChromeGeometry geometry) {
    final endBottom = geometry.endEndpoint;
    final midX = (geometry.startTop.dx + endBottom.dx) / 2;

    final anchors = TextSelectionToolbarAnchors(
      primaryAnchor: Offset(midX, geometry.startTop.dy),
      secondaryAnchor: Offset(midX, endBottom.dy),
    );

    return Positioned.fill(
      child: AdaptiveTextSelectionToolbar.buttonItems(
        anchors: anchors,
        buttonItems: [
          ContextMenuButtonItem(
            type: ContextMenuButtonType.cut,
            onPressed: () {
              _cutSelection();
            },
          ),
          ContextMenuButtonItem(
            type: ContextMenuButtonType.copy,
            onPressed: () {
              _copySelection();
            },
          ),
          ContextMenuButtonItem(
            type: ContextMenuButtonType.paste,
            onPressed: () {
              _pasteClipboard();
            },
          ),
          ContextMenuButtonItem(
            type: ContextMenuButtonType.selectAll,
            onPressed: () {
              _selectAllInEditor();
            },
          ),
        ],
      ),
    );
  }

  /// Default error display when no custom [errorBuilder] is provided.
  Widget _buildDefaultError() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 16),
            Text(
              'Engine Error',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              widget.controller.errorMessage ?? 'Unknown error',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
