import 'package:flutter/material.dart';

class WandererFilterChip<T> extends StatelessWidget {
  final List<T> options;
  final List<T> selectedValues;
  final String Function(T) labelBuilder;
  final Function(List<T>) onChanged;

  const WandererFilterChip({
    super.key,
    required this.options,
    required this.selectedValues,
    required this.labelBuilder,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8.0,
      runSpacing: 4.0,
      children: options.map((option) {
        final isSelected = selectedValues.contains(option);

        return FilterChip(
          label: Text(labelBuilder(option)),
          selected: isSelected,
          onSelected: (bool selected) {
            final newList = List<T>.from(selectedValues);
            if (selected) {
              newList.add(option);
            } else {
              newList.remove(option);
            }
            onChanged(newList);
          },
          backgroundColor: Theme.of(context).canvasColor,
          selectedColor: Theme.of(context).primaryColor.withValues(alpha: 0.2),
          checkmarkColor: Theme.of(context).primaryColor,
          shape: StadiumBorder(
            side: BorderSide(
              color: isSelected
                  ? Theme.of(context).primaryColor
                  : Colors.grey.shade300,
            ),
          ),
          labelStyle: TextStyle(
            color: isSelected ? Theme.of(context).primaryColor : Colors.black87,
            fontWeight: isSelected ? FontWeight.w600 : FontWeight.normal,
          ),
        );
      }).toList(),
    );
  }
}
