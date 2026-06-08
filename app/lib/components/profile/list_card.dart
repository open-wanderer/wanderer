import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/models/global_search_models.dart';

class ListCard extends StatelessWidget {
  final ListSearchResult list;

  const ListCard({super.key, required this.list});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 8),
      width: 160,
      child: Material(
        color: Theme.of(context).cardColor,
        borderRadius: BorderRadius.circular(20),
        child: Ink(
          decoration: BoxDecoration(
            color: Theme.of(context).cardColor,
            borderRadius: BorderRadius.circular(20),
            border: Border.all(
              color: Colors.grey.withValues(alpha: 0.2),
              width: 1,
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              ClipRRect(
                borderRadius: const BorderRadius.vertical(
                  top: Radius.circular(20),
                ),
                child: AspectRatio(
                  aspectRatio: 16 / 9,
                  child: list.avatar != null && list.avatar!.isNotEmpty
                      ? Image(
                          image: NetworkImage(list.avatar!),
                          fit: BoxFit.cover,
                          errorBuilder: (_, _, _) => _placeholder(context),
                        )
                      : _placeholder(context),
                ),
              ),
              Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      list.name,
                      style: const TextStyle(
                        fontWeight: FontWeight.bold,
                        fontSize: 14,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 2),
                    Text(
                      '${list.trails} trails',
                      style: TextStyle(
                        color: Colors.grey[600],
                        fontSize: 12,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _placeholder(BuildContext context) {
    return Container(
      color: Colors.grey.shade200,
      child: const Center(
        child: FaIcon(FontAwesomeIcons.layerGroup, color: Colors.grey),
      ),
    );
  }
}
