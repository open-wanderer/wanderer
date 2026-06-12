import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:textfield_tags/textfield_tags.dart';

class WandererAutocomplete<T> extends StatefulWidget {
  final void Function(String)? onChanged;
  final DynamicTagData<T> Function(String)? onSubmitted;
  final void Function(DynamicTagData<T>)? onDeleted;
  final void Function(DynamicTagData<T>)? onSelected;
  final List<DynamicTagData<T>>? initialTags;
  final String? hintText;
  final AutocompleteOptionsBuilder<DynamicTagData<T>> optionsBuilder;

  const WandererAutocomplete({
    super.key,
    this.onChanged,
    this.hintText,
    this.onDeleted,
    this.onSubmitted,
    this.onSelected,
    this.initialTags,
    required this.optionsBuilder,
  });

  @override
  State<WandererAutocomplete> createState() => _WandererAutocompleteState<T>();
}

class _WandererAutocompleteState<T> extends State<WandererAutocomplete<T>> {
  late double _distanceToField;
  late DynamicTagController<DynamicTagData<T>> _dynamicTagController;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _distanceToField = MediaQuery.of(context).size.width;
  }

  @override
  void dispose() {
    super.dispose();
    _dynamicTagController.dispose();
  }

  @override
  void initState() {
    super.initState();
    _dynamicTagController = DynamicTagController<DynamicTagData<T>>();
  }

  @override
  Widget build(BuildContext context) {
    return Autocomplete<DynamicTagData<T>>(
      optionsViewBuilder: (context, onSelected, options) {
        return Align(
          alignment: Alignment.topCenter,
          child: Material(
            elevation: 4.0,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxHeight: 200),
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: options.length,
                itemBuilder: (BuildContext context, int index) {
                  final DynamicTagData<T> option = options.elementAt(index);
                  return TextButton(
                    onPressed: () {
                      onSelected(option);
                    },
                    child: Align(
                      alignment: Alignment.centerLeft,
                      child: Text(option.tag, textAlign: TextAlign.left),
                    ),
                  );
                },
              ),
            ),
          ),
        );
      },
      fieldViewBuilder:
          (context, textEditingController, focusNode, onFieldSubmitted) {
            return TextFieldTags<DynamicTagData<T>>(
              textfieldTagsController: _dynamicTagController,
              textEditingController: textEditingController,
              focusNode: focusNode,
              textSeparators: const [' ', ','],
              letterCase: LetterCase.normal,
              initialTags: widget.initialTags,
              validator: (DynamicTagData<T> tag) {
                if (_dynamicTagController.getTags!.any(
                  (element) =>
                      element.tag.toLowerCase() == tag.tag.toLowerCase(),
                )) {
                  return 'duplciate tag';
                } else if (tag.tag.isEmpty) {
                  return 'empty tag';
                }
                return null;
              },
              inputFieldBuilder: (context, inputFieldValues) {
                return TextField(
                  controller: inputFieldValues.textEditingController,
                  focusNode: inputFieldValues.focusNode,
                  decoration: InputDecoration(
                    isDense: true,
                    enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(56),
                      borderSide: BorderSide(
                        color: Theme.of(context).colorScheme.outline,
                        width: 1,
                      ),
                    ),
                    focusedBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(56),
                      borderSide: BorderSide(
                        color: Theme.of(context).colorScheme.onSurface,
                        width: 1,
                      ),
                    ),
                    hintText: inputFieldValues.tags.isNotEmpty
                        ? ''
                        : widget.hintText ?? '',
                    hintStyle: TextStyle(color: Colors.grey),
                    errorText: inputFieldValues.error,
                    prefixIconConstraints: BoxConstraints(
                      maxWidth: _distanceToField * 0.74,
                    ),
                    prefixIcon: inputFieldValues.tags.isNotEmpty
                        ? SingleChildScrollView(
                            controller: inputFieldValues.tagScrollController,
                            scrollDirection: Axis.horizontal,
                            child: Row(
                              children: inputFieldValues.tags.map((
                                DynamicTagData<T> tagData,
                              ) {
                                return Container(
                                  decoration: BoxDecoration(
                                    color: Theme.of(context).primaryColor,
                                    borderRadius: const BorderRadius.all(
                                      Radius.circular(20.0),
                                    ),
                                  ),
                                  margin: const EdgeInsets.symmetric(
                                    horizontal: 5.0,
                                  ),
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 10.0,
                                    vertical: 5.0,
                                  ),
                                  child: Row(
                                    mainAxisAlignment:
                                        MainAxisAlignment.spaceBetween,
                                    children: [
                                      InkWell(
                                        child: Text(
                                          tagData.tag,
                                          style: TextStyle(
                                            color: Theme.of(
                                              context,
                                            ).colorScheme.onPrimary,
                                          ),
                                        ),
                                        onTap: () {
                                          // print("${tag.tag} selected");
                                        },
                                      ),
                                      const SizedBox(width: 4.0),
                                      InkWell(
                                        child: FaIcon(
                                          FontAwesomeIcons.circleXmark,
                                          size: 12.0,
                                          color: Theme.of(
                                            context,
                                          ).colorScheme.onPrimary,
                                        ),
                                        onTap: () {
                                          widget.onDeleted?.call(tagData);
                                          inputFieldValues.onTagRemoved(
                                            tagData,
                                          );
                                        },
                                      ),
                                    ],
                                  ),
                                );
                              }).toList(),
                            ),
                          )
                        : null,
                  ),
                  onSubmitted: (value) {
                    if (value.isEmpty) {
                      return;
                    }
                    if (_dynamicTagController.getTags!.any(
                      (element) =>
                          element.tag.toLowerCase() == value.toLowerCase(),
                    )) {
                      return;
                    }
                    if (widget.onSubmitted != null) {
                      DynamicTagData<T> tagData = widget.onSubmitted!(value);
                      inputFieldValues.onTagSubmitted(tagData);
                    }
                  },
                );
              },
            );
          },
      optionsBuilder: widget.optionsBuilder,
      onSelected: (option) {
        final added = _dynamicTagController.onTagSubmitted(option) ?? false;
        if (added) {
          widget.onSelected?.call(option);
        }
      },
    );
  }
}
