import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:wanderer/routes/photo_gallery_screen.dart';
import 'dart:io';

class PhotoCollage extends StatelessWidget {
  final List<String> webPhotos;
  final List<String> localPhotos;

  const PhotoCollage({
    super.key,
    required this.webPhotos,
    required this.localPhotos,
  });

  List<PhotoSource> get _allPhotos {
    final sources = <PhotoSource>[];
    for (final url in webPhotos) {
      sources.add(PhotoSource.network(url));
    }
    for (final local in localPhotos) {
      sources.add(PhotoSource.local(local));
    }
    return sources;
  }

  @override
  Widget build(BuildContext context) {
    final photos = _allPhotos;

    if (photos.length == 1) {
      return _tappablePhoto(
        context,
        photos,
        0,
        height: 220,
        borderRadius: BorderRadius.zero,
        forceFitWidth: true,
      );
    }

    if (photos.length == 2) {
      return SizedBox(
        height: 220,
        child: Row(
          children: [
            Expanded(
              child: _tappablePhoto(
                context,
                photos,
                0,
                borderRadius: BorderRadius.zero,
              ),
            ),
            const SizedBox(width: 2),
            Expanded(
              child: _tappablePhoto(
                context,
                photos,
                1,
                borderRadius: BorderRadius.zero,
              ),
            ),
          ],
        ),
      );
    }

    return SizedBox(
      height: 220,
      child: Row(
        children: [
          Expanded(
            flex: 3,
            child: _tappablePhoto(
              context,
              photos,
              0,
              borderRadius: BorderRadius.zero,
            ),
          ),
          const SizedBox(width: 2),
          Expanded(
            flex: 2,
            child: Column(
              children: [
                Expanded(
                  child: _tappablePhoto(
                    context,
                    photos,
                    1,
                    borderRadius: BorderRadius.zero,
                  ),
                ),
                const SizedBox(height: 2),
                Expanded(
                  child: Stack(
                    fit: StackFit.expand,
                    children: [
                      _tappablePhoto(
                        context,
                        photos,
                        2,
                        borderRadius: BorderRadius.zero,
                      ),
                      // Overlay badge for remaining photos
                      if (photos.length > 3)
                        GestureDetector(
                          onTap: () => _openGallery(context, photos, 3),
                          child: Container(
                            color: Colors.black54,
                            child: Center(
                              child: Text(
                                '+${photos.length - 3}',
                                style: const TextStyle(
                                  color: Colors.white,
                                  fontSize: 22,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _tappablePhoto(
    BuildContext context,
    List<PhotoSource> photos,
    int index, {
    double? height,
    BorderRadius borderRadius = BorderRadius.zero,
    bool forceFitWidth = false,
  }) {
    Widget image = photos[index].buildImage(
      fit: forceFitWidth ? BoxFit.fitWidth : BoxFit.cover,
    );

    if (height != null) {
      image = SizedBox(
        height: height,
        width: double.infinity,
        child: ClipRect(
          child: OverflowBox(
            alignment: Alignment.center,
            minHeight: height,
            maxHeight: double.infinity,
            child: image,
          ),
        ),
      );
    } else {
      image = SizedBox.expand(child: image);
    }

    return GestureDetector(
      onTap: () => _openGallery(context, photos, index),
      child: ClipRRect(borderRadius: borderRadius, child: image),
    );
  }

  void _openGallery(
    BuildContext context,
    List<PhotoSource> photos,
    int startIndex,
  ) {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => PhotoGalleryScreen(
          photos: photos,
          initialIndex: startIndex,
          title: '',
        ),
      ),
    );
  }
}

class PhotoSource {
  final String? url;
  final String? localFile;

  const PhotoSource.network(this.url) : localFile = null;
  const PhotoSource.local(this.localFile) : url = null;

  Widget buildImage({required BoxFit fit}) {
    if (url != null) {
      return Image(
        image: CachedNetworkImageProvider(url!),
        fit: fit,
        loadingBuilder: (context, child, progress) {
          if (progress == null) return child;
          return Center(
            child: CircularProgressIndicator(
              value: progress.expectedTotalBytes != null
                  ? progress.cumulativeBytesLoaded /
                        progress.expectedTotalBytes!
                  : null,
              color: Colors.white54,
            ),
          );
        },
        errorBuilder: (_, _, _) => const _BrokenImagePlaceholder(),
      );
    }
    if (localFile != null) {
      return Image.file(
        File(localFile!),
        fit: fit,
        errorBuilder: (_, _, _) => const _BrokenImagePlaceholder(),
      );
    }
    return const _BrokenImagePlaceholder();
  }
}

class _BrokenImagePlaceholder extends StatelessWidget {
  const _BrokenImagePlaceholder();

  @override
  Widget build(BuildContext context) {
    return SvgPicture.asset(
      "assets/svgs/empty_state_trail_${Brightness.light.name}.svg",
      semanticsLabel: 'wanderer logo',
      height: 80,
    );
  }
}
