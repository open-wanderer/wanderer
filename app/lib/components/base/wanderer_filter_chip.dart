import 'package:flutter/material.dart';

class WandererFilterChip<T> extends StatelessWidget {
  final List<T> options;
  final List<T> selectedValues;
  final String Function(T) labelBuilder;
  final Function(List<T>) onChanged;
  final bool multiple;
  // When true, tapping an already-selected chip keeps it selected (no deselect
  // on tap). Deselection is handled via onLongPress instead.
  final bool keepSelectedOnTap;
  final void Function(T item)? onItemTap;
  final void Function(T item)? onLongPress;
  final Widget? Function(T item)? avatarBuilder;
  final int Function(T item)? badgeCountBuilder;

  const WandererFilterChip({
    super.key,
    required this.options,
    required this.selectedValues,
    required this.labelBuilder,
    required this.onChanged,
    this.multiple = false,
    this.keepSelectedOnTap = false,
    this.onItemTap,
    this.onLongPress,
    this.avatarBuilder,
    this.badgeCountBuilder,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Wrap(
      spacing: 8.0,
      runSpacing: 4.0,
      children: options.map((option) {
        final isSelected = selectedValues.contains(option);

        Widget chip = FilterChip(
          label: Text(labelBuilder(option)),
          avatar: avatarBuilder?.call(option),
          selected: isSelected,
          onSelected: (bool selected) {
            onItemTap?.call(option);
            if (multiple) {
              final newList = List<T>.from(selectedValues);
              if (selected) {
                newList.add(option);
              } else if (!keepSelectedOnTap) {
                newList.remove(option);
              }
              onChanged(newList);
            } else {
              onChanged(selected ? [option] : []);
            }
          },
          backgroundColor: colorScheme.surfaceContainerHighest,
          selectedColor: colorScheme.primaryContainer,
          checkmarkColor: colorScheme.onPrimaryContainer,
          showCheckmark: false,
          shape: StadiumBorder(
            side: BorderSide(
              color: isSelected
                  ? colorScheme.outlineVariant
                  : colorScheme.outline,
            ),
          ),
          labelStyle: TextStyle(
            color: isSelected
                ? colorScheme.onPrimaryContainer
                : colorScheme.onSurface,
            fontWeight: isSelected ? FontWeight.w600 : FontWeight.normal,
          ),
        );

        if (onLongPress != null) {
          chip = GestureDetector(
            onLongPress: () => onLongPress!(option),
            child: chip,
          );
        }

        final badgeCount = badgeCountBuilder?.call(option) ?? 0;
        if (badgeCount > 0) {
          chip = Badge(
            label: Text('$badgeCount'),
            child: chip,
          );
        }

        return chip;
      }).toList(),
    );
  }
}
