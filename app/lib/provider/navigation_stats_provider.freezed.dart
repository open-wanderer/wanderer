// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'navigation_stats_provider.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$NavigationStats {

/// Wall-clock elapsed time since the first GPS fix, excluding paused time.
 Duration get elapsed;/// Cumulative Haversine distance travelled, in metres.
 double get distanceMeters;/// Cumulative positive altitude delta (ascent), in metres.
 double get elevationGainMeters;/// Cumulative negative altitude delta (descent), stored as a positive
/// metre value.
 double get elevationLossMeters;/// Instantaneous GPS speed in km/h (guarded against NaN/negative).
 double get currentSpeedKmh;/// Average speed in km/h (distance / elapsed).
 double get averageSpeedKmh;/// Whether stat accumulation is currently frozen by the user.
 bool get isPaused;
/// Create a copy of NavigationStats
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$NavigationStatsCopyWith<NavigationStats> get copyWith => _$NavigationStatsCopyWithImpl<NavigationStats>(this as NavigationStats, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is NavigationStats&&(identical(other.elapsed, elapsed) || other.elapsed == elapsed)&&(identical(other.distanceMeters, distanceMeters) || other.distanceMeters == distanceMeters)&&(identical(other.elevationGainMeters, elevationGainMeters) || other.elevationGainMeters == elevationGainMeters)&&(identical(other.elevationLossMeters, elevationLossMeters) || other.elevationLossMeters == elevationLossMeters)&&(identical(other.currentSpeedKmh, currentSpeedKmh) || other.currentSpeedKmh == currentSpeedKmh)&&(identical(other.averageSpeedKmh, averageSpeedKmh) || other.averageSpeedKmh == averageSpeedKmh)&&(identical(other.isPaused, isPaused) || other.isPaused == isPaused));
}


@override
int get hashCode => Object.hash(runtimeType,elapsed,distanceMeters,elevationGainMeters,elevationLossMeters,currentSpeedKmh,averageSpeedKmh,isPaused);

@override
String toString() {
  return 'NavigationStats(elapsed: $elapsed, distanceMeters: $distanceMeters, elevationGainMeters: $elevationGainMeters, elevationLossMeters: $elevationLossMeters, currentSpeedKmh: $currentSpeedKmh, averageSpeedKmh: $averageSpeedKmh, isPaused: $isPaused)';
}


}

/// @nodoc
abstract mixin class $NavigationStatsCopyWith<$Res>  {
  factory $NavigationStatsCopyWith(NavigationStats value, $Res Function(NavigationStats) _then) = _$NavigationStatsCopyWithImpl;
@useResult
$Res call({
 Duration elapsed, double distanceMeters, double elevationGainMeters, double elevationLossMeters, double currentSpeedKmh, double averageSpeedKmh, bool isPaused
});




}
/// @nodoc
class _$NavigationStatsCopyWithImpl<$Res>
    implements $NavigationStatsCopyWith<$Res> {
  _$NavigationStatsCopyWithImpl(this._self, this._then);

  final NavigationStats _self;
  final $Res Function(NavigationStats) _then;

/// Create a copy of NavigationStats
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? elapsed = null,Object? distanceMeters = null,Object? elevationGainMeters = null,Object? elevationLossMeters = null,Object? currentSpeedKmh = null,Object? averageSpeedKmh = null,Object? isPaused = null,}) {
  return _then(_self.copyWith(
elapsed: null == elapsed ? _self.elapsed : elapsed // ignore: cast_nullable_to_non_nullable
as Duration,distanceMeters: null == distanceMeters ? _self.distanceMeters : distanceMeters // ignore: cast_nullable_to_non_nullable
as double,elevationGainMeters: null == elevationGainMeters ? _self.elevationGainMeters : elevationGainMeters // ignore: cast_nullable_to_non_nullable
as double,elevationLossMeters: null == elevationLossMeters ? _self.elevationLossMeters : elevationLossMeters // ignore: cast_nullable_to_non_nullable
as double,currentSpeedKmh: null == currentSpeedKmh ? _self.currentSpeedKmh : currentSpeedKmh // ignore: cast_nullable_to_non_nullable
as double,averageSpeedKmh: null == averageSpeedKmh ? _self.averageSpeedKmh : averageSpeedKmh // ignore: cast_nullable_to_non_nullable
as double,isPaused: null == isPaused ? _self.isPaused : isPaused // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}

}


/// Adds pattern-matching-related methods to [NavigationStats].
extension NavigationStatsPatterns on NavigationStats {
/// A variant of `map` that fallback to returning `orElse`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _NavigationStats value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _NavigationStats() when $default != null:
return $default(_that);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// Callbacks receives the raw object, upcasted.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case final Subclass2 value:
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _NavigationStats value)  $default,){
final _that = this;
switch (_that) {
case _NavigationStats():
return $default(_that);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `map` that fallback to returning `null`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _NavigationStats value)?  $default,){
final _that = this;
switch (_that) {
case _NavigationStats() when $default != null:
return $default(_that);case _:
  return null;

}
}
/// A variant of `when` that fallback to an `orElse` callback.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( Duration elapsed,  double distanceMeters,  double elevationGainMeters,  double elevationLossMeters,  double currentSpeedKmh,  double averageSpeedKmh,  bool isPaused)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _NavigationStats() when $default != null:
return $default(_that.elapsed,_that.distanceMeters,_that.elevationGainMeters,_that.elevationLossMeters,_that.currentSpeedKmh,_that.averageSpeedKmh,_that.isPaused);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// As opposed to `map`, this offers destructuring.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case Subclass2(:final field2):
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( Duration elapsed,  double distanceMeters,  double elevationGainMeters,  double elevationLossMeters,  double currentSpeedKmh,  double averageSpeedKmh,  bool isPaused)  $default,) {final _that = this;
switch (_that) {
case _NavigationStats():
return $default(_that.elapsed,_that.distanceMeters,_that.elevationGainMeters,_that.elevationLossMeters,_that.currentSpeedKmh,_that.averageSpeedKmh,_that.isPaused);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `when` that fallback to returning `null`
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( Duration elapsed,  double distanceMeters,  double elevationGainMeters,  double elevationLossMeters,  double currentSpeedKmh,  double averageSpeedKmh,  bool isPaused)?  $default,) {final _that = this;
switch (_that) {
case _NavigationStats() when $default != null:
return $default(_that.elapsed,_that.distanceMeters,_that.elevationGainMeters,_that.elevationLossMeters,_that.currentSpeedKmh,_that.averageSpeedKmh,_that.isPaused);case _:
  return null;

}
}

}

/// @nodoc


class _NavigationStats implements NavigationStats {
  const _NavigationStats({this.elapsed = Duration.zero, this.distanceMeters = 0, this.elevationGainMeters = 0, this.elevationLossMeters = 0, this.currentSpeedKmh = 0, this.averageSpeedKmh = 0, this.isPaused = false});
  

/// Wall-clock elapsed time since the first GPS fix, excluding paused time.
@override@JsonKey() final  Duration elapsed;
/// Cumulative Haversine distance travelled, in metres.
@override@JsonKey() final  double distanceMeters;
/// Cumulative positive altitude delta (ascent), in metres.
@override@JsonKey() final  double elevationGainMeters;
/// Cumulative negative altitude delta (descent), stored as a positive
/// metre value.
@override@JsonKey() final  double elevationLossMeters;
/// Instantaneous GPS speed in km/h (guarded against NaN/negative).
@override@JsonKey() final  double currentSpeedKmh;
/// Average speed in km/h (distance / elapsed).
@override@JsonKey() final  double averageSpeedKmh;
/// Whether stat accumulation is currently frozen by the user.
@override@JsonKey() final  bool isPaused;

/// Create a copy of NavigationStats
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$NavigationStatsCopyWith<_NavigationStats> get copyWith => __$NavigationStatsCopyWithImpl<_NavigationStats>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _NavigationStats&&(identical(other.elapsed, elapsed) || other.elapsed == elapsed)&&(identical(other.distanceMeters, distanceMeters) || other.distanceMeters == distanceMeters)&&(identical(other.elevationGainMeters, elevationGainMeters) || other.elevationGainMeters == elevationGainMeters)&&(identical(other.elevationLossMeters, elevationLossMeters) || other.elevationLossMeters == elevationLossMeters)&&(identical(other.currentSpeedKmh, currentSpeedKmh) || other.currentSpeedKmh == currentSpeedKmh)&&(identical(other.averageSpeedKmh, averageSpeedKmh) || other.averageSpeedKmh == averageSpeedKmh)&&(identical(other.isPaused, isPaused) || other.isPaused == isPaused));
}


@override
int get hashCode => Object.hash(runtimeType,elapsed,distanceMeters,elevationGainMeters,elevationLossMeters,currentSpeedKmh,averageSpeedKmh,isPaused);

@override
String toString() {
  return 'NavigationStats(elapsed: $elapsed, distanceMeters: $distanceMeters, elevationGainMeters: $elevationGainMeters, elevationLossMeters: $elevationLossMeters, currentSpeedKmh: $currentSpeedKmh, averageSpeedKmh: $averageSpeedKmh, isPaused: $isPaused)';
}


}

/// @nodoc
abstract mixin class _$NavigationStatsCopyWith<$Res> implements $NavigationStatsCopyWith<$Res> {
  factory _$NavigationStatsCopyWith(_NavigationStats value, $Res Function(_NavigationStats) _then) = __$NavigationStatsCopyWithImpl;
@override @useResult
$Res call({
 Duration elapsed, double distanceMeters, double elevationGainMeters, double elevationLossMeters, double currentSpeedKmh, double averageSpeedKmh, bool isPaused
});




}
/// @nodoc
class __$NavigationStatsCopyWithImpl<$Res>
    implements _$NavigationStatsCopyWith<$Res> {
  __$NavigationStatsCopyWithImpl(this._self, this._then);

  final _NavigationStats _self;
  final $Res Function(_NavigationStats) _then;

/// Create a copy of NavigationStats
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? elapsed = null,Object? distanceMeters = null,Object? elevationGainMeters = null,Object? elevationLossMeters = null,Object? currentSpeedKmh = null,Object? averageSpeedKmh = null,Object? isPaused = null,}) {
  return _then(_NavigationStats(
elapsed: null == elapsed ? _self.elapsed : elapsed // ignore: cast_nullable_to_non_nullable
as Duration,distanceMeters: null == distanceMeters ? _self.distanceMeters : distanceMeters // ignore: cast_nullable_to_non_nullable
as double,elevationGainMeters: null == elevationGainMeters ? _self.elevationGainMeters : elevationGainMeters // ignore: cast_nullable_to_non_nullable
as double,elevationLossMeters: null == elevationLossMeters ? _self.elevationLossMeters : elevationLossMeters // ignore: cast_nullable_to_non_nullable
as double,currentSpeedKmh: null == currentSpeedKmh ? _self.currentSpeedKmh : currentSpeedKmh // ignore: cast_nullable_to_non_nullable
as double,averageSpeedKmh: null == averageSpeedKmh ? _self.averageSpeedKmh : averageSpeedKmh // ignore: cast_nullable_to_non_nullable
as double,isPaused: null == isPaused ? _self.isPaused : isPaused // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}


}

/// @nodoc
mixin _$NavigationStatsSeed {

 double get distanceMeters; double get elevationGainMeters; double get elevationLossMeters; Duration get elapsed; Duration get pausedAccum; bool get isPaused;
/// Create a copy of NavigationStatsSeed
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$NavigationStatsSeedCopyWith<NavigationStatsSeed> get copyWith => _$NavigationStatsSeedCopyWithImpl<NavigationStatsSeed>(this as NavigationStatsSeed, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is NavigationStatsSeed&&(identical(other.distanceMeters, distanceMeters) || other.distanceMeters == distanceMeters)&&(identical(other.elevationGainMeters, elevationGainMeters) || other.elevationGainMeters == elevationGainMeters)&&(identical(other.elevationLossMeters, elevationLossMeters) || other.elevationLossMeters == elevationLossMeters)&&(identical(other.elapsed, elapsed) || other.elapsed == elapsed)&&(identical(other.pausedAccum, pausedAccum) || other.pausedAccum == pausedAccum)&&(identical(other.isPaused, isPaused) || other.isPaused == isPaused));
}


@override
int get hashCode => Object.hash(runtimeType,distanceMeters,elevationGainMeters,elevationLossMeters,elapsed,pausedAccum,isPaused);

@override
String toString() {
  return 'NavigationStatsSeed(distanceMeters: $distanceMeters, elevationGainMeters: $elevationGainMeters, elevationLossMeters: $elevationLossMeters, elapsed: $elapsed, pausedAccum: $pausedAccum, isPaused: $isPaused)';
}


}

/// @nodoc
abstract mixin class $NavigationStatsSeedCopyWith<$Res>  {
  factory $NavigationStatsSeedCopyWith(NavigationStatsSeed value, $Res Function(NavigationStatsSeed) _then) = _$NavigationStatsSeedCopyWithImpl;
@useResult
$Res call({
 double distanceMeters, double elevationGainMeters, double elevationLossMeters, Duration elapsed, Duration pausedAccum, bool isPaused
});




}
/// @nodoc
class _$NavigationStatsSeedCopyWithImpl<$Res>
    implements $NavigationStatsSeedCopyWith<$Res> {
  _$NavigationStatsSeedCopyWithImpl(this._self, this._then);

  final NavigationStatsSeed _self;
  final $Res Function(NavigationStatsSeed) _then;

/// Create a copy of NavigationStatsSeed
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? distanceMeters = null,Object? elevationGainMeters = null,Object? elevationLossMeters = null,Object? elapsed = null,Object? pausedAccum = null,Object? isPaused = null,}) {
  return _then(_self.copyWith(
distanceMeters: null == distanceMeters ? _self.distanceMeters : distanceMeters // ignore: cast_nullable_to_non_nullable
as double,elevationGainMeters: null == elevationGainMeters ? _self.elevationGainMeters : elevationGainMeters // ignore: cast_nullable_to_non_nullable
as double,elevationLossMeters: null == elevationLossMeters ? _self.elevationLossMeters : elevationLossMeters // ignore: cast_nullable_to_non_nullable
as double,elapsed: null == elapsed ? _self.elapsed : elapsed // ignore: cast_nullable_to_non_nullable
as Duration,pausedAccum: null == pausedAccum ? _self.pausedAccum : pausedAccum // ignore: cast_nullable_to_non_nullable
as Duration,isPaused: null == isPaused ? _self.isPaused : isPaused // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}

}


/// Adds pattern-matching-related methods to [NavigationStatsSeed].
extension NavigationStatsSeedPatterns on NavigationStatsSeed {
/// A variant of `map` that fallback to returning `orElse`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _NavigationStatsSeed value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _NavigationStatsSeed() when $default != null:
return $default(_that);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// Callbacks receives the raw object, upcasted.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case final Subclass2 value:
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _NavigationStatsSeed value)  $default,){
final _that = this;
switch (_that) {
case _NavigationStatsSeed():
return $default(_that);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `map` that fallback to returning `null`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _NavigationStatsSeed value)?  $default,){
final _that = this;
switch (_that) {
case _NavigationStatsSeed() when $default != null:
return $default(_that);case _:
  return null;

}
}
/// A variant of `when` that fallback to an `orElse` callback.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( double distanceMeters,  double elevationGainMeters,  double elevationLossMeters,  Duration elapsed,  Duration pausedAccum,  bool isPaused)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _NavigationStatsSeed() when $default != null:
return $default(_that.distanceMeters,_that.elevationGainMeters,_that.elevationLossMeters,_that.elapsed,_that.pausedAccum,_that.isPaused);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// As opposed to `map`, this offers destructuring.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case Subclass2(:final field2):
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( double distanceMeters,  double elevationGainMeters,  double elevationLossMeters,  Duration elapsed,  Duration pausedAccum,  bool isPaused)  $default,) {final _that = this;
switch (_that) {
case _NavigationStatsSeed():
return $default(_that.distanceMeters,_that.elevationGainMeters,_that.elevationLossMeters,_that.elapsed,_that.pausedAccum,_that.isPaused);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `when` that fallback to returning `null`
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( double distanceMeters,  double elevationGainMeters,  double elevationLossMeters,  Duration elapsed,  Duration pausedAccum,  bool isPaused)?  $default,) {final _that = this;
switch (_that) {
case _NavigationStatsSeed() when $default != null:
return $default(_that.distanceMeters,_that.elevationGainMeters,_that.elevationLossMeters,_that.elapsed,_that.pausedAccum,_that.isPaused);case _:
  return null;

}
}

}

/// @nodoc


class _NavigationStatsSeed implements NavigationStatsSeed {
  const _NavigationStatsSeed({required this.distanceMeters, required this.elevationGainMeters, required this.elevationLossMeters, required this.elapsed, required this.pausedAccum, required this.isPaused});
  

@override final  double distanceMeters;
@override final  double elevationGainMeters;
@override final  double elevationLossMeters;
@override final  Duration elapsed;
@override final  Duration pausedAccum;
@override final  bool isPaused;

/// Create a copy of NavigationStatsSeed
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$NavigationStatsSeedCopyWith<_NavigationStatsSeed> get copyWith => __$NavigationStatsSeedCopyWithImpl<_NavigationStatsSeed>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _NavigationStatsSeed&&(identical(other.distanceMeters, distanceMeters) || other.distanceMeters == distanceMeters)&&(identical(other.elevationGainMeters, elevationGainMeters) || other.elevationGainMeters == elevationGainMeters)&&(identical(other.elevationLossMeters, elevationLossMeters) || other.elevationLossMeters == elevationLossMeters)&&(identical(other.elapsed, elapsed) || other.elapsed == elapsed)&&(identical(other.pausedAccum, pausedAccum) || other.pausedAccum == pausedAccum)&&(identical(other.isPaused, isPaused) || other.isPaused == isPaused));
}


@override
int get hashCode => Object.hash(runtimeType,distanceMeters,elevationGainMeters,elevationLossMeters,elapsed,pausedAccum,isPaused);

@override
String toString() {
  return 'NavigationStatsSeed(distanceMeters: $distanceMeters, elevationGainMeters: $elevationGainMeters, elevationLossMeters: $elevationLossMeters, elapsed: $elapsed, pausedAccum: $pausedAccum, isPaused: $isPaused)';
}


}

/// @nodoc
abstract mixin class _$NavigationStatsSeedCopyWith<$Res> implements $NavigationStatsSeedCopyWith<$Res> {
  factory _$NavigationStatsSeedCopyWith(_NavigationStatsSeed value, $Res Function(_NavigationStatsSeed) _then) = __$NavigationStatsSeedCopyWithImpl;
@override @useResult
$Res call({
 double distanceMeters, double elevationGainMeters, double elevationLossMeters, Duration elapsed, Duration pausedAccum, bool isPaused
});




}
/// @nodoc
class __$NavigationStatsSeedCopyWithImpl<$Res>
    implements _$NavigationStatsSeedCopyWith<$Res> {
  __$NavigationStatsSeedCopyWithImpl(this._self, this._then);

  final _NavigationStatsSeed _self;
  final $Res Function(_NavigationStatsSeed) _then;

/// Create a copy of NavigationStatsSeed
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? distanceMeters = null,Object? elevationGainMeters = null,Object? elevationLossMeters = null,Object? elapsed = null,Object? pausedAccum = null,Object? isPaused = null,}) {
  return _then(_NavigationStatsSeed(
distanceMeters: null == distanceMeters ? _self.distanceMeters : distanceMeters // ignore: cast_nullable_to_non_nullable
as double,elevationGainMeters: null == elevationGainMeters ? _self.elevationGainMeters : elevationGainMeters // ignore: cast_nullable_to_non_nullable
as double,elevationLossMeters: null == elevationLossMeters ? _self.elevationLossMeters : elevationLossMeters // ignore: cast_nullable_to_non_nullable
as double,elapsed: null == elapsed ? _self.elapsed : elapsed // ignore: cast_nullable_to_non_nullable
as Duration,pausedAccum: null == pausedAccum ? _self.pausedAccum : pausedAccum // ignore: cast_nullable_to_non_nullable
as Duration,isPaused: null == isPaused ? _self.isPaused : isPaused // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}


}

// dart format on
