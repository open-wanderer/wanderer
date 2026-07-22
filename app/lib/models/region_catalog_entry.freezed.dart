// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'region_catalog_entry.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$RegionCatalogEntry {

 String get id; String get name;/// `[minLon, minLat, maxLon, maxLat]` — matches `generator.go`'s pmtiles
/// extract argument order and the region archive path builders.
 List<double> get bbox; CatalogStatus get status; String? get version;@JsonKey(name: 'vector_url') String? get vectorUrl;@JsonKey(name: 'vector_size') int? get vectorSize;@JsonKey(name: 'dem_status') CatalogStatus? get demStatus;@JsonKey(name: 'dem_url') String? get demUrl;@JsonKey(name: 'dem_size') int? get demSize; String? get error;
/// Create a copy of RegionCatalogEntry
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$RegionCatalogEntryCopyWith<RegionCatalogEntry> get copyWith => _$RegionCatalogEntryCopyWithImpl<RegionCatalogEntry>(this as RegionCatalogEntry, _$identity);

  /// Serializes this RegionCatalogEntry to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is RegionCatalogEntry&&(identical(other.id, id) || other.id == id)&&(identical(other.name, name) || other.name == name)&&const DeepCollectionEquality().equals(other.bbox, bbox)&&(identical(other.status, status) || other.status == status)&&(identical(other.version, version) || other.version == version)&&(identical(other.vectorUrl, vectorUrl) || other.vectorUrl == vectorUrl)&&(identical(other.vectorSize, vectorSize) || other.vectorSize == vectorSize)&&(identical(other.demStatus, demStatus) || other.demStatus == demStatus)&&(identical(other.demUrl, demUrl) || other.demUrl == demUrl)&&(identical(other.demSize, demSize) || other.demSize == demSize)&&(identical(other.error, error) || other.error == error));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,name,const DeepCollectionEquality().hash(bbox),status,version,vectorUrl,vectorSize,demStatus,demUrl,demSize,error);

@override
String toString() {
  return 'RegionCatalogEntry(id: $id, name: $name, bbox: $bbox, status: $status, version: $version, vectorUrl: $vectorUrl, vectorSize: $vectorSize, demStatus: $demStatus, demUrl: $demUrl, demSize: $demSize, error: $error)';
}


}

/// @nodoc
abstract mixin class $RegionCatalogEntryCopyWith<$Res>  {
  factory $RegionCatalogEntryCopyWith(RegionCatalogEntry value, $Res Function(RegionCatalogEntry) _then) = _$RegionCatalogEntryCopyWithImpl;
@useResult
$Res call({
 String id, String name, List<double> bbox, CatalogStatus status, String? version,@JsonKey(name: 'vector_url') String? vectorUrl,@JsonKey(name: 'vector_size') int? vectorSize,@JsonKey(name: 'dem_status') CatalogStatus? demStatus,@JsonKey(name: 'dem_url') String? demUrl,@JsonKey(name: 'dem_size') int? demSize, String? error
});




}
/// @nodoc
class _$RegionCatalogEntryCopyWithImpl<$Res>
    implements $RegionCatalogEntryCopyWith<$Res> {
  _$RegionCatalogEntryCopyWithImpl(this._self, this._then);

  final RegionCatalogEntry _self;
  final $Res Function(RegionCatalogEntry) _then;

/// Create a copy of RegionCatalogEntry
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? name = null,Object? bbox = null,Object? status = null,Object? version = freezed,Object? vectorUrl = freezed,Object? vectorSize = freezed,Object? demStatus = freezed,Object? demUrl = freezed,Object? demSize = freezed,Object? error = freezed,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,bbox: null == bbox ? _self.bbox : bbox // ignore: cast_nullable_to_non_nullable
as List<double>,status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as CatalogStatus,version: freezed == version ? _self.version : version // ignore: cast_nullable_to_non_nullable
as String?,vectorUrl: freezed == vectorUrl ? _self.vectorUrl : vectorUrl // ignore: cast_nullable_to_non_nullable
as String?,vectorSize: freezed == vectorSize ? _self.vectorSize : vectorSize // ignore: cast_nullable_to_non_nullable
as int?,demStatus: freezed == demStatus ? _self.demStatus : demStatus // ignore: cast_nullable_to_non_nullable
as CatalogStatus?,demUrl: freezed == demUrl ? _self.demUrl : demUrl // ignore: cast_nullable_to_non_nullable
as String?,demSize: freezed == demSize ? _self.demSize : demSize // ignore: cast_nullable_to_non_nullable
as int?,error: freezed == error ? _self.error : error // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}

}


/// Adds pattern-matching-related methods to [RegionCatalogEntry].
extension RegionCatalogEntryPatterns on RegionCatalogEntry {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _RegionCatalogEntry value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _RegionCatalogEntry() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _RegionCatalogEntry value)  $default,){
final _that = this;
switch (_that) {
case _RegionCatalogEntry():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _RegionCatalogEntry value)?  $default,){
final _that = this;
switch (_that) {
case _RegionCatalogEntry() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String name,  List<double> bbox,  CatalogStatus status,  String? version, @JsonKey(name: 'vector_url')  String? vectorUrl, @JsonKey(name: 'vector_size')  int? vectorSize, @JsonKey(name: 'dem_status')  CatalogStatus? demStatus, @JsonKey(name: 'dem_url')  String? demUrl, @JsonKey(name: 'dem_size')  int? demSize,  String? error)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _RegionCatalogEntry() when $default != null:
return $default(_that.id,_that.name,_that.bbox,_that.status,_that.version,_that.vectorUrl,_that.vectorSize,_that.demStatus,_that.demUrl,_that.demSize,_that.error);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String name,  List<double> bbox,  CatalogStatus status,  String? version, @JsonKey(name: 'vector_url')  String? vectorUrl, @JsonKey(name: 'vector_size')  int? vectorSize, @JsonKey(name: 'dem_status')  CatalogStatus? demStatus, @JsonKey(name: 'dem_url')  String? demUrl, @JsonKey(name: 'dem_size')  int? demSize,  String? error)  $default,) {final _that = this;
switch (_that) {
case _RegionCatalogEntry():
return $default(_that.id,_that.name,_that.bbox,_that.status,_that.version,_that.vectorUrl,_that.vectorSize,_that.demStatus,_that.demUrl,_that.demSize,_that.error);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String name,  List<double> bbox,  CatalogStatus status,  String? version, @JsonKey(name: 'vector_url')  String? vectorUrl, @JsonKey(name: 'vector_size')  int? vectorSize, @JsonKey(name: 'dem_status')  CatalogStatus? demStatus, @JsonKey(name: 'dem_url')  String? demUrl, @JsonKey(name: 'dem_size')  int? demSize,  String? error)?  $default,) {final _that = this;
switch (_that) {
case _RegionCatalogEntry() when $default != null:
return $default(_that.id,_that.name,_that.bbox,_that.status,_that.version,_that.vectorUrl,_that.vectorSize,_that.demStatus,_that.demUrl,_that.demSize,_that.error);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _RegionCatalogEntry implements RegionCatalogEntry {
  const _RegionCatalogEntry({required this.id, required this.name, required final  List<double> bbox, required this.status, this.version, @JsonKey(name: 'vector_url') this.vectorUrl, @JsonKey(name: 'vector_size') this.vectorSize, @JsonKey(name: 'dem_status') this.demStatus, @JsonKey(name: 'dem_url') this.demUrl, @JsonKey(name: 'dem_size') this.demSize, this.error}): _bbox = bbox;
  factory _RegionCatalogEntry.fromJson(Map<String, dynamic> json) => _$RegionCatalogEntryFromJson(json);

@override final  String id;
@override final  String name;
/// `[minLon, minLat, maxLon, maxLat]` — matches `generator.go`'s pmtiles
/// extract argument order and the region archive path builders.
 final  List<double> _bbox;
/// `[minLon, minLat, maxLon, maxLat]` — matches `generator.go`'s pmtiles
/// extract argument order and the region archive path builders.
@override List<double> get bbox {
  if (_bbox is EqualUnmodifiableListView) return _bbox;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_bbox);
}

@override final  CatalogStatus status;
@override final  String? version;
@override@JsonKey(name: 'vector_url') final  String? vectorUrl;
@override@JsonKey(name: 'vector_size') final  int? vectorSize;
@override@JsonKey(name: 'dem_status') final  CatalogStatus? demStatus;
@override@JsonKey(name: 'dem_url') final  String? demUrl;
@override@JsonKey(name: 'dem_size') final  int? demSize;
@override final  String? error;

/// Create a copy of RegionCatalogEntry
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$RegionCatalogEntryCopyWith<_RegionCatalogEntry> get copyWith => __$RegionCatalogEntryCopyWithImpl<_RegionCatalogEntry>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$RegionCatalogEntryToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _RegionCatalogEntry&&(identical(other.id, id) || other.id == id)&&(identical(other.name, name) || other.name == name)&&const DeepCollectionEquality().equals(other._bbox, _bbox)&&(identical(other.status, status) || other.status == status)&&(identical(other.version, version) || other.version == version)&&(identical(other.vectorUrl, vectorUrl) || other.vectorUrl == vectorUrl)&&(identical(other.vectorSize, vectorSize) || other.vectorSize == vectorSize)&&(identical(other.demStatus, demStatus) || other.demStatus == demStatus)&&(identical(other.demUrl, demUrl) || other.demUrl == demUrl)&&(identical(other.demSize, demSize) || other.demSize == demSize)&&(identical(other.error, error) || other.error == error));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,name,const DeepCollectionEquality().hash(_bbox),status,version,vectorUrl,vectorSize,demStatus,demUrl,demSize,error);

@override
String toString() {
  return 'RegionCatalogEntry(id: $id, name: $name, bbox: $bbox, status: $status, version: $version, vectorUrl: $vectorUrl, vectorSize: $vectorSize, demStatus: $demStatus, demUrl: $demUrl, demSize: $demSize, error: $error)';
}


}

/// @nodoc
abstract mixin class _$RegionCatalogEntryCopyWith<$Res> implements $RegionCatalogEntryCopyWith<$Res> {
  factory _$RegionCatalogEntryCopyWith(_RegionCatalogEntry value, $Res Function(_RegionCatalogEntry) _then) = __$RegionCatalogEntryCopyWithImpl;
@override @useResult
$Res call({
 String id, String name, List<double> bbox, CatalogStatus status, String? version,@JsonKey(name: 'vector_url') String? vectorUrl,@JsonKey(name: 'vector_size') int? vectorSize,@JsonKey(name: 'dem_status') CatalogStatus? demStatus,@JsonKey(name: 'dem_url') String? demUrl,@JsonKey(name: 'dem_size') int? demSize, String? error
});




}
/// @nodoc
class __$RegionCatalogEntryCopyWithImpl<$Res>
    implements _$RegionCatalogEntryCopyWith<$Res> {
  __$RegionCatalogEntryCopyWithImpl(this._self, this._then);

  final _RegionCatalogEntry _self;
  final $Res Function(_RegionCatalogEntry) _then;

/// Create a copy of RegionCatalogEntry
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? name = null,Object? bbox = null,Object? status = null,Object? version = freezed,Object? vectorUrl = freezed,Object? vectorSize = freezed,Object? demStatus = freezed,Object? demUrl = freezed,Object? demSize = freezed,Object? error = freezed,}) {
  return _then(_RegionCatalogEntry(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,bbox: null == bbox ? _self._bbox : bbox // ignore: cast_nullable_to_non_nullable
as List<double>,status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as CatalogStatus,version: freezed == version ? _self.version : version // ignore: cast_nullable_to_non_nullable
as String?,vectorUrl: freezed == vectorUrl ? _self.vectorUrl : vectorUrl // ignore: cast_nullable_to_non_nullable
as String?,vectorSize: freezed == vectorSize ? _self.vectorSize : vectorSize // ignore: cast_nullable_to_non_nullable
as int?,demStatus: freezed == demStatus ? _self.demStatus : demStatus // ignore: cast_nullable_to_non_nullable
as CatalogStatus?,demUrl: freezed == demUrl ? _self.demUrl : demUrl // ignore: cast_nullable_to_non_nullable
as String?,demSize: freezed == demSize ? _self.demSize : demSize // ignore: cast_nullable_to_non_nullable
as int?,error: freezed == error ? _self.error : error // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}


}

// dart format on
