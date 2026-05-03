import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/components/base/wanderer_filter_chip.dart';

class TrailFilterScreen extends ConsumerWidget {
  const TrailFilterScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final filter = ref.watch(trailFilterProvider);
    final UserEntity user = ref.watch(authProvider).value!;
    final categories = ref.watch(categoryProvider);

    return Scaffold(
      appBar: AppBar(),
      body: filter.when(
        data: (f) {
          return Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24.0),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  filter.value?.toFilterText(actor: user.actorId) ??
                      "Filter loading...",
                ),
                Text(
                  AppLocalizations.of(context)?.categories ?? "Categories",
                  style: TextTheme.of(context).labelLarge,
                ),
                WandererFilterChip<Category>(
                  options: categories.value ?? [],
                  selectedValues: f.category,
                  labelBuilder: (c) => c.name,
                  onChanged: (categories) {
                    ref
                        .read(trailFilterProvider.notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(category: categories),
                        );
                  },
                ),
              ],
            ),
          );
        },
        loading: () => Center(child: CircularProgressIndicator()),
        error: (e, trace) => Text(e.toString()),
      ),
    );
  }
}
