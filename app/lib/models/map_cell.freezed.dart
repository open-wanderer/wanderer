// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'map_cell.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$MapCellInfoList {

 List<MapCellInfo> get cells;
/// Create a copy of MapCellInfoList
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$MapCellInfoListCopyWith<MapCellInfoList> get copyWith => _$MapCellInfoListCopyWithImpl<MapCellInfoList>(this as MapCellInfoList, _$identity);

  /// Serializes this MapCellInfoList to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is MapCellInfoList&&const DeepCollectionEquality().equals(other.cells, cells));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(cells));

@override
String toString() {
  return 'MapCellInfoList(cells: $cells)';
}


}

/// @nodoc
abstract mixin class $MapCellInfoListCopyWith<$Res>  {
  factory $MapCellInfoListCopyWith(MapCellInfoList value, $Res Function(MapCellInfoList) _then) = _$MapCellInfoListCopyWithImpl;
@useResult
$Res call({
 List<MapCellInfo> cells
});




}
/// @nodoc
class _$MapCellInfoListCopyWithImpl<$Res>
    implements $MapCellInfoListCopyWith<$Res> {
  _$MapCellInfoListCopyWithImpl(this._self, this._then);

  final MapCellInfoList _self;
  final $Res Function(MapCellInfoList) _then;

/// Create a copy of MapCellInfoList
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? cells = null,}) {
  return _then(_self.copyWith(
cells: null == cells ? _self.cells : cells // ignore: cast_nullable_to_non_nullable
as List<MapCellInfo>,
  ));
}

}


/// Adds pattern-matching-related methods to [MapCellInfoList].
extension MapCellInfoListPatterns on MapCellInfoList {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _MapCellInfoList value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _MapCellInfoList() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _MapCellInfoList value)  $default,){
final _that = this;
switch (_that) {
case _MapCellInfoList():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _MapCellInfoList value)?  $default,){
final _that = this;
switch (_that) {
case _MapCellInfoList() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<MapCellInfo> cells)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _MapCellInfoList() when $default != null:
return $default(_that.cells);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<MapCellInfo> cells)  $default,) {final _that = this;
switch (_that) {
case _MapCellInfoList():
return $default(_that.cells);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<MapCellInfo> cells)?  $default,) {final _that = this;
switch (_that) {
case _MapCellInfoList() when $default != null:
return $default(_that.cells);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _MapCellInfoList implements MapCellInfoList {
  const _MapCellInfoList({required final  List<MapCellInfo> cells}): _cells = cells;
  factory _MapCellInfoList.fromJson(Map<String, dynamic> json) => _$MapCellInfoListFromJson(json);

 final  List<MapCellInfo> _cells;
@override List<MapCellInfo> get cells {
  if (_cells is EqualUnmodifiableListView) return _cells;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_cells);
}


/// Create a copy of MapCellInfoList
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$MapCellInfoListCopyWith<_MapCellInfoList> get copyWith => __$MapCellInfoListCopyWithImpl<_MapCellInfoList>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$MapCellInfoListToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _MapCellInfoList&&const DeepCollectionEquality().equals(other._cells, _cells));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(_cells));

@override
String toString() {
  return 'MapCellInfoList(cells: $cells)';
}


}

/// @nodoc
abstract mixin class _$MapCellInfoListCopyWith<$Res> implements $MapCellInfoListCopyWith<$Res> {
  factory _$MapCellInfoListCopyWith(_MapCellInfoList value, $Res Function(_MapCellInfoList) _then) = __$MapCellInfoListCopyWithImpl;
@override @useResult
$Res call({
 List<MapCellInfo> cells
});




}
/// @nodoc
class __$MapCellInfoListCopyWithImpl<$Res>
    implements _$MapCellInfoListCopyWith<$Res> {
  __$MapCellInfoListCopyWithImpl(this._self, this._then);

  final _MapCellInfoList _self;
  final $Res Function(_MapCellInfoList) _then;

/// Create a copy of MapCellInfoList
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? cells = null,}) {
  return _then(_MapCellInfoList(
cells: null == cells ? _self._cells : cells // ignore: cast_nullable_to_non_nullable
as List<MapCellInfo>,
  ));
}


}


/// @nodoc
mixin _$MapCellInfo {

 String get key; MapCellStatus get status; String get url;@JsonKey(name: 'size_bytes') int? get sizeBytes;
/// Create a copy of MapCellInfo
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$MapCellInfoCopyWith<MapCellInfo> get copyWith => _$MapCellInfoCopyWithImpl<MapCellInfo>(this as MapCellInfo, _$identity);

  /// Serializes this MapCellInfo to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is MapCellInfo&&(identical(other.key, key) || other.key == key)&&(identical(other.status, status) || other.status == status)&&(identical(other.url, url) || other.url == url)&&(identical(other.sizeBytes, sizeBytes) || other.sizeBytes == sizeBytes));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,key,status,url,sizeBytes);

@override
String toString() {
  return 'MapCellInfo(key: $key, status: $status, url: $url, sizeBytes: $sizeBytes)';
}


}

/// @nodoc
abstract mixin class $MapCellInfoCopyWith<$Res>  {
  factory $MapCellInfoCopyWith(MapCellInfo value, $Res Function(MapCellInfo) _then) = _$MapCellInfoCopyWithImpl;
@useResult
$Res call({
 String key, MapCellStatus status, String url,@JsonKey(name: 'size_bytes') int? sizeBytes
});




}
/// @nodoc
class _$MapCellInfoCopyWithImpl<$Res>
    implements $MapCellInfoCopyWith<$Res> {
  _$MapCellInfoCopyWithImpl(this._self, this._then);

  final MapCellInfo _self;
  final $Res Function(MapCellInfo) _then;

/// Create a copy of MapCellInfo
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? key = null,Object? status = null,Object? url = null,Object? sizeBytes = freezed,}) {
  return _then(_self.copyWith(
key: null == key ? _self.key : key // ignore: cast_nullable_to_non_nullable
as String,status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as MapCellStatus,url: null == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String,sizeBytes: freezed == sizeBytes ? _self.sizeBytes : sizeBytes // ignore: cast_nullable_to_non_nullable
as int?,
  ));
}

}


/// Adds pattern-matching-related methods to [MapCellInfo].
extension MapCellInfoPatterns on MapCellInfo {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _MapCellInfo value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _MapCellInfo() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _MapCellInfo value)  $default,){
final _that = this;
switch (_that) {
case _MapCellInfo():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _MapCellInfo value)?  $default,){
final _that = this;
switch (_that) {
case _MapCellInfo() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String key,  MapCellStatus status,  String url, @JsonKey(name: 'size_bytes')  int? sizeBytes)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _MapCellInfo() when $default != null:
return $default(_that.key,_that.status,_that.url,_that.sizeBytes);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String key,  MapCellStatus status,  String url, @JsonKey(name: 'size_bytes')  int? sizeBytes)  $default,) {final _that = this;
switch (_that) {
case _MapCellInfo():
return $default(_that.key,_that.status,_that.url,_that.sizeBytes);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String key,  MapCellStatus status,  String url, @JsonKey(name: 'size_bytes')  int? sizeBytes)?  $default,) {final _that = this;
switch (_that) {
case _MapCellInfo() when $default != null:
return $default(_that.key,_that.status,_that.url,_that.sizeBytes);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _MapCellInfo implements MapCellInfo {
  const _MapCellInfo({required this.key, required this.status, required this.url, @JsonKey(name: 'size_bytes') this.sizeBytes});
  factory _MapCellInfo.fromJson(Map<String, dynamic> json) => _$MapCellInfoFromJson(json);

@override final  String key;
@override final  MapCellStatus status;
@override final  String url;
@override@JsonKey(name: 'size_bytes') final  int? sizeBytes;

/// Create a copy of MapCellInfo
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$MapCellInfoCopyWith<_MapCellInfo> get copyWith => __$MapCellInfoCopyWithImpl<_MapCellInfo>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$MapCellInfoToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _MapCellInfo&&(identical(other.key, key) || other.key == key)&&(identical(other.status, status) || other.status == status)&&(identical(other.url, url) || other.url == url)&&(identical(other.sizeBytes, sizeBytes) || other.sizeBytes == sizeBytes));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,key,status,url,sizeBytes);

@override
String toString() {
  return 'MapCellInfo(key: $key, status: $status, url: $url, sizeBytes: $sizeBytes)';
}


}

/// @nodoc
abstract mixin class _$MapCellInfoCopyWith<$Res> implements $MapCellInfoCopyWith<$Res> {
  factory _$MapCellInfoCopyWith(_MapCellInfo value, $Res Function(_MapCellInfo) _then) = __$MapCellInfoCopyWithImpl;
@override @useResult
$Res call({
 String key, MapCellStatus status, String url,@JsonKey(name: 'size_bytes') int? sizeBytes
});




}
/// @nodoc
class __$MapCellInfoCopyWithImpl<$Res>
    implements _$MapCellInfoCopyWith<$Res> {
  __$MapCellInfoCopyWithImpl(this._self, this._then);

  final _MapCellInfo _self;
  final $Res Function(_MapCellInfo) _then;

/// Create a copy of MapCellInfo
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? key = null,Object? status = null,Object? url = null,Object? sizeBytes = freezed,}) {
  return _then(_MapCellInfo(
key: null == key ? _self.key : key // ignore: cast_nullable_to_non_nullable
as String,status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as MapCellStatus,url: null == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String,sizeBytes: freezed == sizeBytes ? _self.sizeBytes : sizeBytes // ignore: cast_nullable_to_non_nullable
as int?,
  ));
}


}


/// @nodoc
mixin _$MapCellStatusResponse {

 MapCellStatus get status;@JsonKey(name: 'download_url') String? get downloadUrl;@JsonKey(name: 'status_url') String? get statusUrl;@JsonKey(name: 'size_bytes') int? get sizeBytes;@JsonKey(name: 'dem_download_url') String? get demDownloadUrl; String? get error;
/// Create a copy of MapCellStatusResponse
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$MapCellStatusResponseCopyWith<MapCellStatusResponse> get copyWith => _$MapCellStatusResponseCopyWithImpl<MapCellStatusResponse>(this as MapCellStatusResponse, _$identity);

  /// Serializes this MapCellStatusResponse to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is MapCellStatusResponse&&(identical(other.status, status) || other.status == status)&&(identical(other.downloadUrl, downloadUrl) || other.downloadUrl == downloadUrl)&&(identical(other.statusUrl, statusUrl) || other.statusUrl == statusUrl)&&(identical(other.sizeBytes, sizeBytes) || other.sizeBytes == sizeBytes)&&(identical(other.demDownloadUrl, demDownloadUrl) || other.demDownloadUrl == demDownloadUrl)&&(identical(other.error, error) || other.error == error));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,status,downloadUrl,statusUrl,sizeBytes,demDownloadUrl,error);

@override
String toString() {
  return 'MapCellStatusResponse(status: $status, downloadUrl: $downloadUrl, statusUrl: $statusUrl, sizeBytes: $sizeBytes, demDownloadUrl: $demDownloadUrl, error: $error)';
}


}

/// @nodoc
abstract mixin class $MapCellStatusResponseCopyWith<$Res>  {
  factory $MapCellStatusResponseCopyWith(MapCellStatusResponse value, $Res Function(MapCellStatusResponse) _then) = _$MapCellStatusResponseCopyWithImpl;
@useResult
$Res call({
 MapCellStatus status,@JsonKey(name: 'download_url') String? downloadUrl,@JsonKey(name: 'status_url') String? statusUrl,@JsonKey(name: 'size_bytes') int? sizeBytes,@JsonKey(name: 'dem_download_url') String? demDownloadUrl, String? error
});




}
/// @nodoc
class _$MapCellStatusResponseCopyWithImpl<$Res>
    implements $MapCellStatusResponseCopyWith<$Res> {
  _$MapCellStatusResponseCopyWithImpl(this._self, this._then);

  final MapCellStatusResponse _self;
  final $Res Function(MapCellStatusResponse) _then;

/// Create a copy of MapCellStatusResponse
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? status = null,Object? downloadUrl = freezed,Object? statusUrl = freezed,Object? sizeBytes = freezed,Object? demDownloadUrl = freezed,Object? error = freezed,}) {
  return _then(_self.copyWith(
status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as MapCellStatus,downloadUrl: freezed == downloadUrl ? _self.downloadUrl : downloadUrl // ignore: cast_nullable_to_non_nullable
as String?,statusUrl: freezed == statusUrl ? _self.statusUrl : statusUrl // ignore: cast_nullable_to_non_nullable
as String?,sizeBytes: freezed == sizeBytes ? _self.sizeBytes : sizeBytes // ignore: cast_nullable_to_non_nullable
as int?,demDownloadUrl: freezed == demDownloadUrl ? _self.demDownloadUrl : demDownloadUrl // ignore: cast_nullable_to_non_nullable
as String?,error: freezed == error ? _self.error : error // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}

}


/// Adds pattern-matching-related methods to [MapCellStatusResponse].
extension MapCellStatusResponsePatterns on MapCellStatusResponse {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _MapCellStatusResponse value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _MapCellStatusResponse() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _MapCellStatusResponse value)  $default,){
final _that = this;
switch (_that) {
case _MapCellStatusResponse():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _MapCellStatusResponse value)?  $default,){
final _that = this;
switch (_that) {
case _MapCellStatusResponse() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( MapCellStatus status, @JsonKey(name: 'download_url')  String? downloadUrl, @JsonKey(name: 'status_url')  String? statusUrl, @JsonKey(name: 'size_bytes')  int? sizeBytes, @JsonKey(name: 'dem_download_url')  String? demDownloadUrl,  String? error)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _MapCellStatusResponse() when $default != null:
return $default(_that.status,_that.downloadUrl,_that.statusUrl,_that.sizeBytes,_that.demDownloadUrl,_that.error);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( MapCellStatus status, @JsonKey(name: 'download_url')  String? downloadUrl, @JsonKey(name: 'status_url')  String? statusUrl, @JsonKey(name: 'size_bytes')  int? sizeBytes, @JsonKey(name: 'dem_download_url')  String? demDownloadUrl,  String? error)  $default,) {final _that = this;
switch (_that) {
case _MapCellStatusResponse():
return $default(_that.status,_that.downloadUrl,_that.statusUrl,_that.sizeBytes,_that.demDownloadUrl,_that.error);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( MapCellStatus status, @JsonKey(name: 'download_url')  String? downloadUrl, @JsonKey(name: 'status_url')  String? statusUrl, @JsonKey(name: 'size_bytes')  int? sizeBytes, @JsonKey(name: 'dem_download_url')  String? demDownloadUrl,  String? error)?  $default,) {final _that = this;
switch (_that) {
case _MapCellStatusResponse() when $default != null:
return $default(_that.status,_that.downloadUrl,_that.statusUrl,_that.sizeBytes,_that.demDownloadUrl,_that.error);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _MapCellStatusResponse implements MapCellStatusResponse {
  const _MapCellStatusResponse({required this.status, @JsonKey(name: 'download_url') this.downloadUrl, @JsonKey(name: 'status_url') this.statusUrl, @JsonKey(name: 'size_bytes') this.sizeBytes, @JsonKey(name: 'dem_download_url') this.demDownloadUrl, this.error});
  factory _MapCellStatusResponse.fromJson(Map<String, dynamic> json) => _$MapCellStatusResponseFromJson(json);

@override final  MapCellStatus status;
@override@JsonKey(name: 'download_url') final  String? downloadUrl;
@override@JsonKey(name: 'status_url') final  String? statusUrl;
@override@JsonKey(name: 'size_bytes') final  int? sizeBytes;
@override@JsonKey(name: 'dem_download_url') final  String? demDownloadUrl;
@override final  String? error;

/// Create a copy of MapCellStatusResponse
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$MapCellStatusResponseCopyWith<_MapCellStatusResponse> get copyWith => __$MapCellStatusResponseCopyWithImpl<_MapCellStatusResponse>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$MapCellStatusResponseToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _MapCellStatusResponse&&(identical(other.status, status) || other.status == status)&&(identical(other.downloadUrl, downloadUrl) || other.downloadUrl == downloadUrl)&&(identical(other.statusUrl, statusUrl) || other.statusUrl == statusUrl)&&(identical(other.sizeBytes, sizeBytes) || other.sizeBytes == sizeBytes)&&(identical(other.demDownloadUrl, demDownloadUrl) || other.demDownloadUrl == demDownloadUrl)&&(identical(other.error, error) || other.error == error));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,status,downloadUrl,statusUrl,sizeBytes,demDownloadUrl,error);

@override
String toString() {
  return 'MapCellStatusResponse(status: $status, downloadUrl: $downloadUrl, statusUrl: $statusUrl, sizeBytes: $sizeBytes, demDownloadUrl: $demDownloadUrl, error: $error)';
}


}

/// @nodoc
abstract mixin class _$MapCellStatusResponseCopyWith<$Res> implements $MapCellStatusResponseCopyWith<$Res> {
  factory _$MapCellStatusResponseCopyWith(_MapCellStatusResponse value, $Res Function(_MapCellStatusResponse) _then) = __$MapCellStatusResponseCopyWithImpl;
@override @useResult
$Res call({
 MapCellStatus status,@JsonKey(name: 'download_url') String? downloadUrl,@JsonKey(name: 'status_url') String? statusUrl,@JsonKey(name: 'size_bytes') int? sizeBytes,@JsonKey(name: 'dem_download_url') String? demDownloadUrl, String? error
});




}
/// @nodoc
class __$MapCellStatusResponseCopyWithImpl<$Res>
    implements _$MapCellStatusResponseCopyWith<$Res> {
  __$MapCellStatusResponseCopyWithImpl(this._self, this._then);

  final _MapCellStatusResponse _self;
  final $Res Function(_MapCellStatusResponse) _then;

/// Create a copy of MapCellStatusResponse
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? status = null,Object? downloadUrl = freezed,Object? statusUrl = freezed,Object? sizeBytes = freezed,Object? demDownloadUrl = freezed,Object? error = freezed,}) {
  return _then(_MapCellStatusResponse(
status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as MapCellStatus,downloadUrl: freezed == downloadUrl ? _self.downloadUrl : downloadUrl // ignore: cast_nullable_to_non_nullable
as String?,statusUrl: freezed == statusUrl ? _self.statusUrl : statusUrl // ignore: cast_nullable_to_non_nullable
as String?,sizeBytes: freezed == sizeBytes ? _self.sizeBytes : sizeBytes // ignore: cast_nullable_to_non_nullable
as int?,demDownloadUrl: freezed == demDownloadUrl ? _self.demDownloadUrl : demDownloadUrl // ignore: cast_nullable_to_non_nullable
as String?,error: freezed == error ? _self.error : error // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}


}

// dart format on
