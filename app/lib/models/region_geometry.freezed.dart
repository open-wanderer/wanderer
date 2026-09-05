// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'region_geometry.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$RegionGeometry {

 String get path;/// Raw GeoJSON *geometry* object (`Polygon` or `MultiPolygon`) as stored
/// in the `region_geometry.polygon` JSON column. This is NOT a Feature
/// and NOT a FeatureCollection — the caller wraps it in a Feature before
/// handing it to MapLibre.
 Map<String, dynamic> get polygon;/// `[minLon, minLat, maxLon, maxLat]` — the same order [RegionEntity]'s
/// four discrete bbox columns and `generator.go`'s pmtiles extract
/// arguments use.
 List<double> get bbox;
/// Create a copy of RegionGeometry
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$RegionGeometryCopyWith<RegionGeometry> get copyWith => _$RegionGeometryCopyWithImpl<RegionGeometry>(this as RegionGeometry, _$identity);

  /// Serializes this RegionGeometry to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is RegionGeometry&&(identical(other.path, path) || other.path == path)&&const DeepCollectionEquality().equals(other.polygon, polygon)&&const DeepCollectionEquality().equals(other.bbox, bbox));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,path,const DeepCollectionEquality().hash(polygon),const DeepCollectionEquality().hash(bbox));

@override
String toString() {
  return 'RegionGeometry(path: $path, polygon: $polygon, bbox: $bbox)';
}


}

/// @nodoc
abstract mixin class $RegionGeometryCopyWith<$Res>  {
  factory $RegionGeometryCopyWith(RegionGeometry value, $Res Function(RegionGeometry) _then) = _$RegionGeometryCopyWithImpl;
@useResult
$Res call({
 String path, Map<String, dynamic> polygon, List<double> bbox
});




}
/// @nodoc
class _$RegionGeometryCopyWithImpl<$Res>
    implements $RegionGeometryCopyWith<$Res> {
  _$RegionGeometryCopyWithImpl(this._self, this._then);

  final RegionGeometry _self;
  final $Res Function(RegionGeometry) _then;

/// Create a copy of RegionGeometry
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? path = null,Object? polygon = null,Object? bbox = null,}) {
  return _then(_self.copyWith(
path: null == path ? _self.path : path // ignore: cast_nullable_to_non_nullable
as String,polygon: null == polygon ? _self.polygon : polygon // ignore: cast_nullable_to_non_nullable
as Map<String, dynamic>,bbox: null == bbox ? _self.bbox : bbox // ignore: cast_nullable_to_non_nullable
as List<double>,
  ));
}

}


/// Adds pattern-matching-related methods to [RegionGeometry].
extension RegionGeometryPatterns on RegionGeometry {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _RegionGeometry value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _RegionGeometry() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _RegionGeometry value)  $default,){
final _that = this;
switch (_that) {
case _RegionGeometry():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _RegionGeometry value)?  $default,){
final _that = this;
switch (_that) {
case _RegionGeometry() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String path,  Map<String, dynamic> polygon,  List<double> bbox)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _RegionGeometry() when $default != null:
return $default(_that.path,_that.polygon,_that.bbox);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String path,  Map<String, dynamic> polygon,  List<double> bbox)  $default,) {final _that = this;
switch (_that) {
case _RegionGeometry():
return $default(_that.path,_that.polygon,_that.bbox);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String path,  Map<String, dynamic> polygon,  List<double> bbox)?  $default,) {final _that = this;
switch (_that) {
case _RegionGeometry() when $default != null:
return $default(_that.path,_that.polygon,_that.bbox);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _RegionGeometry implements RegionGeometry {
  const _RegionGeometry({required this.path, required final  Map<String, dynamic> polygon, required final  List<double> bbox}): _polygon = polygon,_bbox = bbox;
  factory _RegionGeometry.fromJson(Map<String, dynamic> json) => _$RegionGeometryFromJson(json);

@override final  String path;
/// Raw GeoJSON *geometry* object (`Polygon` or `MultiPolygon`) as stored
/// in the `region_geometry.polygon` JSON column. This is NOT a Feature
/// and NOT a FeatureCollection — the caller wraps it in a Feature before
/// handing it to MapLibre.
 final  Map<String, dynamic> _polygon;
/// Raw GeoJSON *geometry* object (`Polygon` or `MultiPolygon`) as stored
/// in the `region_geometry.polygon` JSON column. This is NOT a Feature
/// and NOT a FeatureCollection — the caller wraps it in a Feature before
/// handing it to MapLibre.
@override Map<String, dynamic> get polygon {
  if (_polygon is EqualUnmodifiableMapView) return _polygon;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableMapView(_polygon);
}

/// `[minLon, minLat, maxLon, maxLat]` — the same order [RegionEntity]'s
/// four discrete bbox columns and `generator.go`'s pmtiles extract
/// arguments use.
 final  List<double> _bbox;
/// `[minLon, minLat, maxLon, maxLat]` — the same order [RegionEntity]'s
/// four discrete bbox columns and `generator.go`'s pmtiles extract
/// arguments use.
@override List<double> get bbox {
  if (_bbox is EqualUnmodifiableListView) return _bbox;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_bbox);
}


/// Create a copy of RegionGeometry
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$RegionGeometryCopyWith<_RegionGeometry> get copyWith => __$RegionGeometryCopyWithImpl<_RegionGeometry>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$RegionGeometryToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _RegionGeometry&&(identical(other.path, path) || other.path == path)&&const DeepCollectionEquality().equals(other._polygon, _polygon)&&const DeepCollectionEquality().equals(other._bbox, _bbox));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,path,const DeepCollectionEquality().hash(_polygon),const DeepCollectionEquality().hash(_bbox));

@override
String toString() {
  return 'RegionGeometry(path: $path, polygon: $polygon, bbox: $bbox)';
}


}

/// @nodoc
abstract mixin class _$RegionGeometryCopyWith<$Res> implements $RegionGeometryCopyWith<$Res> {
  factory _$RegionGeometryCopyWith(_RegionGeometry value, $Res Function(_RegionGeometry) _then) = __$RegionGeometryCopyWithImpl;
@override @useResult
$Res call({
 String path, Map<String, dynamic> polygon, List<double> bbox
});




}
/// @nodoc
class __$RegionGeometryCopyWithImpl<$Res>
    implements _$RegionGeometryCopyWith<$Res> {
  __$RegionGeometryCopyWithImpl(this._self, this._then);

  final _RegionGeometry _self;
  final $Res Function(_RegionGeometry) _then;

/// Create a copy of RegionGeometry
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? path = null,Object? polygon = null,Object? bbox = null,}) {
  return _then(_RegionGeometry(
path: null == path ? _self.path : path // ignore: cast_nullable_to_non_nullable
as String,polygon: null == polygon ? _self._polygon : polygon // ignore: cast_nullable_to_non_nullable
as Map<String, dynamic>,bbox: null == bbox ? _self._bbox : bbox // ignore: cast_nullable_to_non_nullable
as List<double>,
  ));
}


}

// dart format on
