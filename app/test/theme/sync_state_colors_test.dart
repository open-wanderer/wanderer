import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/theme/sync_state_colors.dart';
import 'package:wanderer/theme/theme.dart';

double _contrast(Color a, Color b) {
  final la = a.computeLuminance();
  final lb = b.computeLuminance();
  final hi = la > lb ? la : lb;
  final lo = la > lb ? lb : la;
  return (hi + 0.05) / (lo + 0.05);
}

void main() {
  final light = AppTheme.createTheme(Brightness.light);
  final dark = AppTheme.createTheme(Brightness.dark);

  group('draft (values unchanged from the pre-fold AppColors)', () {
    test('light', () {
      final c = syncStateColors(TrailSyncState.draft, light);
      expect(c.container, const Color(0xFFFFECB3));
      expect(c.onContainer, const Color(0xFF7A4F01));
      expect(c.icon, c.onContainer);
    });

    test('dark', () {
      final c = syncStateColors(TrailSyncState.draft, dark);
      expect(c.container, const Color(0xFFFFB300).withValues(alpha: 0.18));
      expect(c.onContainer, const Color(0xFFFFE082));
      expect(c.icon, c.onContainer);
    });
  });

  group('failed', () {
    test('light is an opaque red container with a dark red label', () {
      final c = syncStateColors(TrailSyncState.failed, light);
      expect(c.container, const Color(0xFFFFDAD6));
      expect(c.container.a, 1.0);
      expect(c.onContainer, const Color(0xFF93000A));
      expect(c.icon, c.onContainer);
    });

    test('dark is a translucent red with a light red label', () {
      final c = syncStateColors(TrailSyncState.failed, dark);
      expect(c.container, const Color(0xFFE53935).withValues(alpha: 0.18));
      expect(c.container.a, lessThan(1.0));
      expect(c.onContainer, const Color(0xFFFFB4AB));
      expect(c.icon, c.onContainer);
    });
  });

  group('pending / uploading / synced are neutral', () {
    for (final theme in [light, dark]) {
      final name = theme.brightness.name;
      test('pending ($name)', () {
        final c = syncStateColors(TrailSyncState.pending, theme);
        expect(c.container, theme.colorScheme.surfaceContainerHighest);
        expect(c.onContainer, theme.colorScheme.onSurface);
        expect(c.icon, theme.colorScheme.onSurface.withValues(alpha: 0.6));
      });

      test('uploading ($name)', () {
        final c = syncStateColors(TrailSyncState.uploading, theme);
        expect(c.container, theme.colorScheme.surfaceContainerHighest);
        expect(c.onContainer, theme.colorScheme.onSurface);
        expect(c.icon, theme.progressIndicatorTheme.color);
      });

      test('synced returns the pending colors ($name)', () {
        final s = syncStateColors(TrailSyncState.synced, theme);
        final p = syncStateColors(TrailSyncState.pending, theme);
        expect(s.container, p.container);
        expect(s.onContainer, p.onContainer);
        expect(s.icon, p.icon);
      });
    }
  });

  group('contrast and distinctness', () {
    for (final theme in [light, dark]) {
      final name = theme.brightness.name;
      for (final state in [TrailSyncState.draft, TrailSyncState.failed]) {
        test('${state.name} label has AA contrast ($name)', () {
          final c = syncStateColors(state, theme);
          final surface = Color.alphaBlend(
            c.container,
            theme.colorScheme.surface,
          );
          expect(_contrast(c.onContainer, surface), greaterThanOrEqualTo(4.5));
        });
      }

      test('failed, draft and pending containers differ ($name)', () {
        final failed = syncStateColors(TrailSyncState.failed, theme).container;
        final draft = syncStateColors(TrailSyncState.draft, theme).container;
        final pending = syncStateColors(
          TrailSyncState.pending,
          theme,
        ).container;
        expect(failed, isNot(pending));
        expect(draft, isNot(failed));
      });
    }
  });
}
