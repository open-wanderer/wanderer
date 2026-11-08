/// A Valhalla routing profile resolved from a (sub)category's
/// `settings.valhalla_profile`.
///
/// The vocabulary is OPEN: any Valhalla costing model name is accepted and
/// passed through verbatim, so an operator can map a new category onto a
/// costing (e.g. `auto`) without a code change and without an allowlist edit.
/// The ONLY special-cased form is `bicycle_<type>`, which carries Valhalla's
/// `bicycle_type` alongside the `bicycle` costing.
class ValhallaProfile {
  /// Valhalla's `costing` value — e.g. `'pedestrian'`, `'bicycle'`, `'auto'`,
  /// `'motor_scooter'`.
  final String costing;

  /// Valhalla's `costing_options.bicycle_type` value, capitalized exactly as
  /// Valhalla expects (`Road`/`Hybrid`/`Cross`/`Mountain`). Only ever
  /// non-null when [costing] is `'bicycle'`.
  final String? bicycleType;

  const ValhallaProfile(this.costing, {this.bicycleType});

  /// Matches only the 4 literal `bicycle_<type>` forms.
  ///
  /// CRITICAL: parsing never splits on `_` generally — Valhalla has
  /// legitimately underscored costing names (`motor_scooter`) that must
  /// survive whole. Only these 4 exact strings are ever decomposed.
  static final RegExp _bicycleForm = RegExp(
    r'^bicycle_(road|hybrid|cross|mountain)$',
  );

  /// Cheap format sanity guard. A value failing it is treated as unset so a
  /// typo degrades to the caller's fallback chain instead of producing a
  /// malformed Valhalla request.
  static final RegExp _costingFormat = RegExp(r'^[a-z][a-z_]*$');

  /// Parses a raw `settings.valhalla_profile` value.
  ///
  /// Returns `null` — meaning "unset", never an exception — for a non-String,
  /// empty/blank, or format-guard-failing input.
  static ValhallaProfile? parse(Object? raw) {
    if (raw is! String) return null;

    final value = raw.trim().toLowerCase();
    if (value.isEmpty) return null;
    if (!_costingFormat.hasMatch(value)) return null;

    final bicycleMatch = _bicycleForm.firstMatch(value);
    if (bicycleMatch != null) {
      final type = bicycleMatch.group(1)!;
      return ValhallaProfile(
        'bicycle',
        bicycleType: type[0].toUpperCase() + type.substring(1),
      );
    }

    // Everything else is an opaque costing name, passed through verbatim.
    return ValhallaProfile(value);
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ValhallaProfile &&
          other.costing == costing &&
          other.bicycleType == bicycleType;

  @override
  int get hashCode => Object.hash(costing, bicycleType);

  @override
  String toString() =>
      'ValhallaProfile($costing${bicycleType == null ? '' : ', $bicycleType'})';
}
