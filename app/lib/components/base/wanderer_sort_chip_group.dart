import 'package:flutter/material.dart';
import 'package:wanderer/models/trail.dart';

class SortOption<T extends Enum> {
  final String text;
  final T value;

  SortOption({required this.text, required this.value});
}

class WandererSortChipGroup<T extends Enum> extends StatelessWidget {
  final List<SortOption<T>> options;
  final T currentSort;
  final SortOrder sortOrder;
  final void Function(T sort, SortOrder order) onSortChanged;

  const WandererSortChipGroup({
    super.key,
    required this.options,
    required this.currentSort,
    required this.sortOrder,
    required this.onSortChanged,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Wrap(
      spacing: 11.0,
      runSpacing: 4.0,
      children: options.map((option) {
        final isSelected = option.value == currentSort;

        final isDescending = sortOrder == SortOrder.desc;
        final turns = isDescending ? 0.5 : 0.0;

        return RawChip(
          label: Text(option.text),
          avatar: isSelected
              ? AnimatedSize(
                  duration: const Duration(milliseconds: 300),
                  curve: Curves.easeOut,
                  child: SizedBox(
                    width: 18,
                    height: 18,
                    child: AnimatedRotation(
                      turns: turns,
                      duration: const Duration(milliseconds: 650),
                      curve: Curves.elasticOut,
                      child: Icon(
                        Icons.arrow_upward,
                        size: 16,
                        color: Colors.white.withValues(alpha: 0.6),
                      ),
                    ),
                  ),
                )
              : null,

          selected: isSelected,
          showCheckmark: false,
          onPressed: () {
            if (isSelected) {
              final nextOrder = sortOrder == SortOrder.asc
                  ? SortOrder.desc
                  : SortOrder.asc;
              onSortChanged(option.value, nextOrder);
            } else {
              onSortChanged(option.value, SortOrder.asc);
            }
          },
          backgroundColor: colorScheme.surfaceContainerHighest,
          selectedColor: colorScheme.primaryContainer,
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
