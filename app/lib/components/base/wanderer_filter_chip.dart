import 'package:flutter/material.dart';

class WandererFilterChip<T> extends StatelessWidget {
  final List<T> options;
  final List<T> selectedValues;
  final String Function(T) labelBuilder;
  final Function(List<T>) onChanged;
  final bool multiple;
  final Widget? Function(T item)? avatarBuilder;

  const WandererFilterChip({
    super.key,
    required this.options,
    required this.selectedValues,
    required this.labelBuilder,
    required this.onChanged,
    this.multiple = false,
    this.avatarBuilder,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Wrap(
      spacing: 8.0,
      runSpacing: 4.0,
      children: options.map((option) {
        final isSelected = selectedValues.contains(option);

        return FilterChip(
          label: Text(labelBuilder(option)),
          avatar: avatarBuilder?.call(option),
          selected: isSelected,
          onSelected: (bool selected) {
            if (multiple) {
              final newList = List<T>.from(selectedValues);
              if (selected) {
                newList.add(option);
              } else {
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
      }).toList(),
    );
  }
}
