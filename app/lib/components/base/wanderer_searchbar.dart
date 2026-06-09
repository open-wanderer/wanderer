import 'dart:async';

import 'package:flutter/material.dart';

class WandererSearchBar extends StatefulWidget {
  final String? hintText;
  final void Function(String)? onChanged;
  final InputDecoration? decoration;
  const WandererSearchBar({
    super.key,
    this.hintText,
    this.onChanged,
    this.decoration,
  });

  @override
  State<WandererSearchBar> createState() => _WandererSearchBarState();
}

class _WandererSearchBarState extends State<WandererSearchBar> {
  Timer? _debounce;

  @override
  Widget build(BuildContext context) {
    return TextField(
      onChanged: (value) {
        if (_debounce?.isActive ?? false) {
          _debounce!.cancel();
        }
        _debounce = Timer(const Duration(milliseconds: 500), () {
          widget.onChanged?.call(value);
        });
      },
      decoration:
          widget.decoration ??
          InputDecoration(
            hintText: widget.hintText,
            hintStyle: TextStyle(color: Colors.grey),
            filled: true,
            fillColor: Theme.of(context).inputDecorationTheme.fillColor,
            contentPadding: const EdgeInsets.all(12),
            prefixIcon: const Icon(Icons.search, color: Colors.black),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(6),
              borderSide: BorderSide(
                color: Theme.of(context).colorScheme.outline,
              ),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(6),
              borderSide: BorderSide(
                color: Theme.of(context).colorScheme.primary,
                width: 1.5,
              ),
            ),
          ),
    );
  }
}
