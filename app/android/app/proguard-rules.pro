# R8 keep rules for release builds.
#
# Everything here protects code that R8 cannot see being used, because the
# only callers are outside the Java/Kotlin it analyses.

# ---------------------------------------------------------------------------
# MapLibre — reached exclusively through JNI from Dart (package:jni bindings in
# maplibre_android). R8 sees zero Java callers for this entire namespace and
# prunes it as dead code.
#
# MapLibre ships consumer rules, but they only cover `public *` in a subset of
# packages: style.expressions, gestures and most of annotations are not covered
# at all. Verified against the release build's own R8 report
# (app/build/app/outputs/mapping/release/usage.txt):
#
#   * style.expressions — 164 members of Expression deleted, including
#     Expression$Converter.convert(String). maplibre_android calls exactly that
#     for any layer declaring a `filter`
#     (maplibre_android-0.3.5 lib/src/style_controller.dart:24-29), so every
#     filtered layer failed to be added. That is what made the route planner's
#     segment lines silently never render in release, while the unfiltered
#     trail-detail and navigation lines kept working.
#   * gestures — 44 classes had members deleted, 0 keep rules covered them.
#   * annotations — 30 classes had members deleted; only BubbleLayout's
#     constructor was kept.
#
# Deliberately kept whole rather than symbol-by-symbol: the failure mode is
# silent at runtime (the throw lands in a `.ignore()`), so narrowing this to the
# members that happen to be called today just defers the next instance of the
# same bug to whenever the plugin starts calling something new.
-keep class org.maplibre.android.** { *; }
-keep interface org.maplibre.android.** { *; }
