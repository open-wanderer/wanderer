import 'package:cached_network_image/cached_network_image.dart';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/store/avatar_cache.dart';

// ---------------------------------------------------------------------------
// Actor avatar — the single avatar widget for every surface that shows who
// authored something (profile header, trail card/list/panel, comments, summit
// logs, lists).
//
// Two properties matter and both used to be missing outside the profile
// screen:
//
//   * a load failure renders an explicit glyph. The inline call sites this
//     replaced wrote `onBackgroundImageError: (_, _) => FaIcon(...)`, but
//     that callback is an `ImageErrorListener` returning void — the icon was
//     built and thrown away, leaving a bare grey circle.
//   * the signed-in user's avatar resolves from the on-disk cache first, so
//     it survives offline. That is the common case for a locally recorded,
//     not-yet-synced trail: both the server URL and the DiceBear fallback are
//     network fetches that cannot succeed there.
// ---------------------------------------------------------------------------

/// Test seam for the avatar image provider. `CachedNetworkImageProvider`'s
/// cache manager does real file/network I/O that never completes inside a
/// widget test's FakeAsync zone, so the load-failure regression tests inject
/// `NetworkImage.new` here to route the failure through the test HTTP client.
/// Null (always, in production) means [CachedNetworkImageProvider].
@visibleForTesting
ImageProvider Function(String url)? debugAvatarImageProviderFactory;

ImageProvider _avatarImage(String url) =>
    debugAvatarImageProviderFactory?.call(url) ??
    CachedNetworkImageProvider(url);

/// Dispatches to the disk-cached avatar chain when the author is the
/// signed-in user (so the avatar survives offline), and to the plain network
/// avatar for everyone else — remote actors have no locally cached image to
/// fall back on.
///
/// Takes the author's identity as loose parts rather than an [Actor] because
/// the trail card and list item are handed a `TrailSummary`, which for a
/// search result has only a flat avatar URL and never a real actor object.
/// Use the [ActorAvatar.fromActor] constructor where a full [Actor] is in
/// hand.
class ActorAvatar extends ConsumerWidget {
  /// The author's actor record id. Null when the author could not be
  /// resolved at all.
  final String? actorId;

  /// Absolute avatar URL, when the author's actor is known and has one.
  final String? imageUrl;

  /// Username seeding the remote initials fallback. Null suppresses that
  /// fallback in favour of the glyph — a local trail with an unresolved
  /// author has no meaningful seed, and "Unknown" would only be one more
  /// failing network fetch.
  final String? nameSeed;

  final double radius;

  const ActorAvatar({
    super.key,
    required this.actorId,
    required this.imageUrl,
    required this.nameSeed,
    required this.radius,
  });

  ActorAvatar.fromActor({super.key, Actor? actor, required this.radius})
    : actorId = actor?.id,
      imageUrl = actor?.icon,
      nameSeed = actor?.preferredUsername;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authUser = ref.watch(authProvider).value;
    if (authUser != null && actorId != null && actorId == authUser.actorId) {
      return _CachedAvatar(
        user: authUser,
        isOnline: ref.watch(onlineStatusProvider),
        radius: radius,
      );
    }

    if (imageUrl?.isNotEmpty == true) {
      return _NetworkAvatar(url: imageUrl!, radius: radius);
    }

    // The initials fallback is gated on a resolved author: with no actorId
    // the display name is a placeholder ("Unknown"), and seeding the remote
    // service with it would just be one more network fetch that fails
    // wherever this matters most — offline, on a trail that has not synced.
    if (actorId != null && nameSeed?.isNotEmpty == true) {
      return _NetworkAvatar(
        url:
            'https://api.dicebear.com/7.x/initials/png?seed=$nameSeed&backgroundType=gradientLinear',
        radius: radius,
      );
    }

    return _AvatarGlyph(radius: radius);
  }
}

/// The terminal fallback: a plain circle with a person glyph actually painted
/// inside it.
class _AvatarGlyph extends StatelessWidget {
  final double radius;

  const _AvatarGlyph({required this.radius});

  @override
  Widget build(BuildContext context) {
    return CircleAvatar(
      radius: radius,
      backgroundColor: Colors.grey.shade300,
      child: FaIcon(FontAwesomeIcons.user, size: radius),
    );
  }
}

/// A network avatar that actually renders the glyph when the image fails to
/// load, via a `_failed` state flag. `onBackgroundImageError` is an
/// `ImageErrorListener` returning void, so returning a widget from it — as
/// every inline call site used to — draws nothing at all.
class _NetworkAvatar extends StatefulWidget {
  final String url;
  final double radius;

  const _NetworkAvatar({required this.url, required this.radius});

  @override
  State<_NetworkAvatar> createState() => _NetworkAvatarState();
}

class _NetworkAvatarState extends State<_NetworkAvatar> {
  bool _failed = false;

  @override
  void didUpdateWidget(covariant _NetworkAvatar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.url != widget.url) _failed = false;
  }

  @override
  Widget build(BuildContext context) {
    if (_failed) {
      return _AvatarGlyph(radius: widget.radius);
    }

    return CircleAvatar(
      radius: widget.radius,
      backgroundColor: Colors.grey.shade300,
      backgroundImage: _avatarImage(widget.url),
      onBackgroundImageError: (_, _) {
        if (mounted) setState(() => _failed = true);
      },
    );
  }
}

/// Cached avatar — same cached-file / network / glyph fallback chain as the
/// bottom-nav avatar (`wanderer_layout.dart`'s `_NavAvatar`).
class _CachedAvatar extends StatefulWidget {
  final UserEntity user;
  final bool isOnline;
  final double radius;

  const _CachedAvatar({
    required this.user,
    required this.isOnline,
    required this.radius,
  });

  @override
  State<_CachedAvatar> createState() => _CachedAvatarState();
}

class _CachedAvatarState extends State<_CachedAvatar> {
  late Future<File?> _avatarFuture;
  bool _networkFailed = false;

  @override
  void initState() {
    super.initState();
    _avatarFuture = cachedAvatarFile(widget.user.id, widget.user.avatar);
  }

  @override
  void didUpdateWidget(covariant _CachedAvatar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.user.id != widget.user.id ||
        oldWidget.user.avatar != widget.user.avatar) {
      setState(() {
        _networkFailed = false;
        _avatarFuture = cachedAvatarFile(widget.user.id, widget.user.avatar);
      });
    } else if (!oldWidget.isOnline && widget.isOnline && _networkFailed) {
      setState(() => _networkFailed = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final user = widget.user;

    return FutureBuilder<File?>(
      future: _avatarFuture,
      builder: (context, snapshot) {
        final cachedFile = snapshot.data;
        if (cachedFile != null) {
          return CircleAvatar(
            radius: widget.radius,
            backgroundColor: Colors.grey.shade300,
            backgroundImage: FileImage(cachedFile),
          );
        }

        if (widget.isOnline && !_networkFailed) {
          return CircleAvatar(
            radius: widget.radius,
            backgroundColor: Colors.grey.shade300,
            backgroundImage: _avatarImage(
              user.getFileUrl(user.serverUrl, user.avatar) ??
                  "https://api.dicebear.com/7.x/initials/png?seed=${user.preferredUsername}&backgroundType=gradientLinear",
            ),
            onBackgroundImageError: (_, _) {
              if (mounted) setState(() => _networkFailed = true);
            },
          );
        }

        return _AvatarGlyph(radius: widget.radius);
      },
    );
  }
}
