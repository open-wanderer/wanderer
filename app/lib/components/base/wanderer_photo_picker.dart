import 'dart:io';

import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:image_picker/image_picker.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// A reusable photo picker for attaching images to a record.
///
/// Follows the "wrapped externally in a [FormBuilderField]" pattern (like
/// [WandererRichTextEditor]) because it drives `ImagePicker` side effects and
/// owns the local file list. Emits newly-picked local file paths via
/// [onChanged]; already-uploaded [initialWebPhotos] are removable too, when a
/// caller supplies [onWebPhotosChanged].
class WandererPhotoPicker extends StatefulWidget {
  final String? label;

  /// Local file paths of images picked in this session (editable).
  final List<String> initialLocalPhotos;

  /// Filenames of already-uploaded photos, resolved to a displayable URL via
  /// [resolveWebPhotoUrl]. Removable, unlike v1's read-only display.
  final List<String> initialWebPhotos;

  /// Resolves a web photo filename (from [initialWebPhotos]) to a displayable
  /// URL. Required whenever [initialWebPhotos] is non-empty.
  final String Function(String filename)? resolveWebPhotoUrl;

  /// Emits the current list of local file paths whenever it changes.
  final ValueChanged<List<String>> onChanged;

  /// Emits the remaining (non-removed) web photo filenames whenever the user
  /// removes one. Left null by callers that don't support server-photo
  /// removal (no remove action is shown on web photo thumbnails).
  final ValueChanged<List<String>>? onWebPhotosChanged;

  /// When set, each local photo shows an action (e.g. extract EXIF GPS) that
  /// invokes this callback with the photo's local file path. Left null by
  /// callers that don't need it (no button is shown).
  final void Function(String path)? onExifRequested;

  const WandererPhotoPicker({
    super.key,
    this.label,
    this.initialLocalPhotos = const [],
    this.initialWebPhotos = const [],
    this.resolveWebPhotoUrl,
    required this.onChanged,
    this.onWebPhotosChanged,
    this.onExifRequested,
  });

  @override
  State<WandererPhotoPicker> createState() => _WandererPhotoPickerState();
}

class _WandererPhotoPickerState extends State<WandererPhotoPicker> {
  late List<String> _localPhotos = List.of(widget.initialLocalPhotos);
  late List<String> _webPhotos = List.of(widget.initialWebPhotos);

  void _removeWeb(int index) {
    setState(() => _webPhotos = [..._webPhotos]..removeAt(index));
    widget.onWebPhotosChanged?.call(_webPhotos);
  }

  Future<void> _pick() async {
    final picked = await ImagePicker().pickMultiImage();
    if (picked.isEmpty) return;
    setState(
      () => _localPhotos = [..._localPhotos, ...picked.map((x) => x.path)],
    );
    widget.onChanged(_localPhotos);
  }

  void _removeLocal(int index) {
    setState(() => _localPhotos = [..._localPhotos]..removeAt(index));
    widget.onChanged(_localPhotos);
  }

  void _reorderLocal(int oldIndex, int newIndex) {
    if (oldIndex == newIndex) return;
    setState(() {
      final updated = [..._localPhotos];
      final moved = updated.removeAt(oldIndex);
      updated.insert(newIndex, moved);
      _localPhotos = updated;
    });
    widget.onChanged(_localPhotos);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (widget.label != null && widget.label!.isNotEmpty)
          Padding(
            padding: const EdgeInsets.only(bottom: 4),
            child: Text(
              widget.label!,
              style: theme.textTheme.bodySmall?.copyWith(
                fontWeight: FontWeight.w600,
                color: theme.colorScheme.onSurface,
              ),
            ),
          ),
        if (_localPhotos.length > 1)
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: Text(
              AppLocalizations.of(context)!.reorder_photos_hint,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
              ),
            ),
          ),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (var i = 0; i < _webPhotos.length; i++)
              _Thumbnail(
                image: Image.network(
                  widget.resolveWebPhotoUrl?.call(_webPhotos[i]) ??
                      _webPhotos[i],
                  fit: BoxFit.cover,
                ),
                onRemove: widget.onWebPhotosChanged == null
                    ? null
                    : () => _removeWeb(i),
              ),
            for (var i = 0; i < _localPhotos.length; i++)
              _ReorderablePhoto(
                key: ValueKey(_localPhotos[i]),
                index: i,
                onReorder: _reorderLocal,
                child: _Thumbnail(
                  image: Image.file(File(_localPhotos[i]), fit: BoxFit.cover),
                  onRemove: () => _removeLocal(i),
                  onExif: widget.onExifRequested == null
                      ? null
                      : () => widget.onExifRequested!(_localPhotos[i]),
                ),
              ),
            _AddTile(onTap: _pick),
          ],
        ),
      ],
    );
  }
}

/// Makes [child] both draggable (long-press) and a drop target, so photos in
/// the [Wrap] grid can be reordered by dragging one onto another — a
/// `ReorderableListView` only supports a linear list, not a wrapping grid.
class _ReorderablePhoto extends StatelessWidget {
  static const double _size = 96;
  final int index;
  final void Function(int oldIndex, int newIndex) onReorder;
  final Widget child;

  const _ReorderablePhoto({
    super.key,
    required this.index,
    required this.onReorder,
    required this.child,
  });

  @override
  Widget build(BuildContext context) {
    return DragTarget<int>(
      onWillAcceptWithDetails: (details) => details.data != index,
      onAcceptWithDetails: (details) => onReorder(details.data, index),
      builder: (context, candidateData, rejectedData) {
        final isTarget = candidateData.isNotEmpty;
        return AnimatedScale(
          scale: isTarget ? 1.08 : 1.0,
          duration: const Duration(milliseconds: 150),
          child: LongPressDraggable<int>(
            data: index,
            feedback: Opacity(
              opacity: 0.8,
              child: SizedBox(width: _size, height: _size, child: child),
            ),
            childWhenDragging: Opacity(opacity: 0.3, child: child),
            child: child,
          ),
        );
      },
    );
  }
}

class _Thumbnail extends StatelessWidget {
  static const double _size = 96;
  final Widget image;
  final VoidCallback? onRemove;
  final VoidCallback? onExif;

  const _Thumbnail({required this.image, this.onRemove, this.onExif});

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: _size,
      height: _size,
      child: Stack(
        fit: StackFit.expand,
        children: [
          ClipRRect(borderRadius: BorderRadius.circular(6), child: image),
          if (onRemove != null)
            Positioned(
              top: 4,
              right: 4,
              child: _OverlayButton(
                icon: FontAwesomeIcons.xmark,
                onTap: onRemove!,
              ),
            ),
          if (onExif != null)
            Positioned(
              top: 4,
              left: 4,
              child: _OverlayButton(
                icon: FontAwesomeIcons.locationDot,
                onTap: onExif!,
              ),
            ),
        ],
      ),
    );
  }
}

class _OverlayButton extends StatelessWidget {
  final FaIconData icon;
  final VoidCallback onTap;

  const _OverlayButton({required this.icon, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        width: 24,
        height: 24,
        padding: const EdgeInsets.all(4),
        decoration: BoxDecoration(
          color: Colors.black54,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Center(child: FaIcon(icon, size: 16, color: Colors.white)),
      ),
    );
  }
}

class _AddTile extends StatelessWidget {
  static const double _size = 96;
  final VoidCallback onTap;

  const _AddTile({required this.onTap});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(6),
      child: Container(
        width: _size,
        height: _size,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(6),
          border: Border.all(color: theme.colorScheme.outline),
          color: theme.inputDecorationTheme.fillColor,
        ),
        child: Center(
          child: FaIcon(
            FontAwesomeIcons.plus,
            size: 28,
            color: theme.colorScheme.onSurface.withValues(alpha: 0.7),
          ),
        ),
      ),
    );
  }
}
