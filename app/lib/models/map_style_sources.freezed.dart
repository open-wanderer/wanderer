// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'map_style_sources.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$MapStyleSources {

@JsonKey(name: 'tileUrl') String get tileUrl;@JsonKey(name: 'glyphUrl') String get glyphUrl;@JsonKey(name: 'spriteUrl') String get spriteUrl;
/// Create a copy of MapStyleSources
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$MapStyleSourcesCopyWith<MapStyleSources> get copyWith => _$MapStyleSourcesCopyWithImpl<MapStyleSources>(this as MapStyleSources, _$identity);

  /// Serializes this MapStyleSources to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is MapStyleSources&&(identical(other.tileUrl, tileUrl) || other.tileUrl == tileUrl)&&(identical(other.glyphUrl, glyphUrl) || other.glyphUrl == glyphUrl)&&(identical(other.spriteUrl, spriteUrl) || other.spriteUrl == spriteUrl));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,tileUrl,glyphUrl,spriteUrl);

@override
String toString() {
  return 'MapStyleSources(tileUrl: $tileUrl, glyphUrl: $glyphUrl, spriteUrl: $spriteUrl)';
}


}

/// @nodoc
abstract mixin class $MapStyleSourcesCopyWith<$Res>  {
  factory $MapStyleSourcesCopyWith(MapStyleSources value, $Res Function(MapStyleSources) _then) = _$MapStyleSourcesCopyWithImpl;
@useResult
$Res call({
@JsonKey(name: 'tileUrl') String tileUrl,@JsonKey(name: 'glyphUrl') String glyphUrl,@JsonKey(name: 'spriteUrl') String spriteUrl
});




}
/// @nodoc
class _$MapStyleSourcesCopyWithImpl<$Res>
    implements $MapStyleSourcesCopyWith<$Res> {
  _$MapStyleSourcesCopyWithImpl(this._self, this._then);

  final MapStyleSources _self;
  final $Res Function(MapStyleSources) _then;

/// Create a copy of MapStyleSources
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? tileUrl = null,Object? glyphUrl = null,Object? spriteUrl = null,}) {
  return _then(_self.copyWith(
tileUrl: null == tileUrl ? _self.tileUrl : tileUrl // ignore: cast_nullable_to_non_nullable
as String,glyphUrl: null == glyphUrl ? _self.glyphUrl : glyphUrl // ignore: cast_nullable_to_non_nullable
as String,spriteUrl: null == spriteUrl ? _self.spriteUrl : spriteUrl // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

}


/// Adds pattern-matching-related methods to [MapStyleSources].
extension MapStyleSourcesPatterns on MapStyleSources {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _MapStyleSources value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _MapStyleSources() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _MapStyleSources value)  $default,){
final _that = this;
switch (_that) {
case _MapStyleSources():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _MapStyleSources value)?  $default,){
final _that = this;
switch (_that) {
case _MapStyleSources() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function(@JsonKey(name: 'tileUrl')  String tileUrl, @JsonKey(name: 'glyphUrl')  String glyphUrl, @JsonKey(name: 'spriteUrl')  String spriteUrl)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _MapStyleSources() when $default != null:
return $default(_that.tileUrl,_that.glyphUrl,_that.spriteUrl);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function(@JsonKey(name: 'tileUrl')  String tileUrl, @JsonKey(name: 'glyphUrl')  String glyphUrl, @JsonKey(name: 'spriteUrl')  String spriteUrl)  $default,) {final _that = this;
switch (_that) {
case _MapStyleSources():
return $default(_that.tileUrl,_that.glyphUrl,_that.spriteUrl);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function(@JsonKey(name: 'tileUrl')  String tileUrl, @JsonKey(name: 'glyphUrl')  String glyphUrl, @JsonKey(name: 'spriteUrl')  String spriteUrl)?  $default,) {final _that = this;
switch (_that) {
case _MapStyleSources() when $default != null:
return $default(_that.tileUrl,_that.glyphUrl,_that.spriteUrl);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _MapStyleSources implements MapStyleSources {
  const _MapStyleSources({@JsonKey(name: 'tileUrl') required this.tileUrl, @JsonKey(name: 'glyphUrl') required this.glyphUrl, @JsonKey(name: 'spriteUrl') required this.spriteUrl});
  factory _MapStyleSources.fromJson(Map<String, dynamic> json) => _$MapStyleSourcesFromJson(json);

@override@JsonKey(name: 'tileUrl') final  String tileUrl;
@override@JsonKey(name: 'glyphUrl') final  String glyphUrl;
@override@JsonKey(name: 'spriteUrl') final  String spriteUrl;

/// Create a copy of MapStyleSources
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$MapStyleSourcesCopyWith<_MapStyleSources> get copyWith => __$MapStyleSourcesCopyWithImpl<_MapStyleSources>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$MapStyleSourcesToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _MapStyleSources&&(identical(other.tileUrl, tileUrl) || other.tileUrl == tileUrl)&&(identical(other.glyphUrl, glyphUrl) || other.glyphUrl == glyphUrl)&&(identical(other.spriteUrl, spriteUrl) || other.spriteUrl == spriteUrl));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,tileUrl,glyphUrl,spriteUrl);

@override
String toString() {
  return 'MapStyleSources(tileUrl: $tileUrl, glyphUrl: $glyphUrl, spriteUrl: $spriteUrl)';
}


}

/// @nodoc
abstract mixin class _$MapStyleSourcesCopyWith<$Res> implements $MapStyleSourcesCopyWith<$Res> {
  factory _$MapStyleSourcesCopyWith(_MapStyleSources value, $Res Function(_MapStyleSources) _then) = __$MapStyleSourcesCopyWithImpl;
@override @useResult
$Res call({
@JsonKey(name: 'tileUrl') String tileUrl,@JsonKey(name: 'glyphUrl') String glyphUrl,@JsonKey(name: 'spriteUrl') String spriteUrl
});




}
/// @nodoc
class __$MapStyleSourcesCopyWithImpl<$Res>
    implements _$MapStyleSourcesCopyWith<$Res> {
  __$MapStyleSourcesCopyWithImpl(this._self, this._then);

  final _MapStyleSources _self;
  final $Res Function(_MapStyleSources) _then;

/// Create a copy of MapStyleSources
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? tileUrl = null,Object? glyphUrl = null,Object? spriteUrl = null,}) {
  return _then(_MapStyleSources(
tileUrl: null == tileUrl ? _self.tileUrl : tileUrl // ignore: cast_nullable_to_non_nullable
as String,glyphUrl: null == glyphUrl ? _self.glyphUrl : glyphUrl // ignore: cast_nullable_to_non_nullable
as String,spriteUrl: null == spriteUrl ? _self.spriteUrl : spriteUrl // ignore: cast_nullable_to_non_nullable
as String,
  ));
}


}

// dart format on
