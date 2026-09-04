// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'waypoint.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$Waypoint {

 String get id; String get collectionId; String get collectionName; String? get name; String? get description; double get lat; double get lon;@JsonKey(name: 'distance_from_start') double? get distanceFromStart;@FaIconDataConverter() FaIconData get icon; List<String> get photos; String get author; String? get trail; DateTime get created; DateTime get updated;// Non-serializable local fields
@JsonKey(includeFromJson: false, includeToJson: false) dynamic get marker;@JsonKey(includeFromJson: false, includeToJson: false) List<String> get localPhotos;/// Carries list identity for a waypoint that has no server id yet
/// yet. Device-local only, never serialized.
@JsonKey(includeFromJson: false, includeToJson: false) String? get localKey;
/// Create a copy of Waypoint
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$WaypointCopyWith<Waypoint> get copyWith => _$WaypointCopyWithImpl<Waypoint>(this as Waypoint, _$identity);

  /// Serializes this Waypoint to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Waypoint&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.collectionName, collectionName) || other.collectionName == collectionName)&&(identical(other.name, name) || other.name == name)&&(identical(other.description, description) || other.description == description)&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon)&&(identical(other.distanceFromStart, distanceFromStart) || other.distanceFromStart == distanceFromStart)&&(identical(other.icon, icon) || other.icon == icon)&&const DeepCollectionEquality().equals(other.photos, photos)&&(identical(other.author, author) || other.author == author)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&const DeepCollectionEquality().equals(other.marker, marker)&&const DeepCollectionEquality().equals(other.localPhotos, localPhotos)&&(identical(other.localKey, localKey) || other.localKey == localKey));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,collectionId,collectionName,name,description,lat,lon,distanceFromStart,icon,const DeepCollectionEquality().hash(photos),author,trail,created,updated,const DeepCollectionEquality().hash(marker),const DeepCollectionEquality().hash(localPhotos),localKey);

@override
String toString() {
  return 'Waypoint(id: $id, collectionId: $collectionId, collectionName: $collectionName, name: $name, description: $description, lat: $lat, lon: $lon, distanceFromStart: $distanceFromStart, icon: $icon, photos: $photos, author: $author, trail: $trail, created: $created, updated: $updated, marker: $marker, localPhotos: $localPhotos, localKey: $localKey)';
}


}

/// @nodoc
abstract mixin class $WaypointCopyWith<$Res>  {
  factory $WaypointCopyWith(Waypoint value, $Res Function(Waypoint) _then) = _$WaypointCopyWithImpl;
@useResult
$Res call({
 String id, String collectionId, String collectionName, String? name, String? description, double lat, double lon,@JsonKey(name: 'distance_from_start') double? distanceFromStart,@FaIconDataConverter() FaIconData icon, List<String> photos, String author, String? trail, DateTime created, DateTime updated,@JsonKey(includeFromJson: false, includeToJson: false) dynamic marker,@JsonKey(includeFromJson: false, includeToJson: false) List<String> localPhotos,@JsonKey(includeFromJson: false, includeToJson: false) String? localKey
});




}
/// @nodoc
class _$WaypointCopyWithImpl<$Res>
    implements $WaypointCopyWith<$Res> {
  _$WaypointCopyWithImpl(this._self, this._then);

  final Waypoint _self;
  final $Res Function(Waypoint) _then;

/// Create a copy of Waypoint
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? collectionId = null,Object? collectionName = null,Object? name = freezed,Object? description = freezed,Object? lat = null,Object? lon = null,Object? distanceFromStart = freezed,Object? icon = null,Object? photos = null,Object? author = null,Object? trail = freezed,Object? created = null,Object? updated = null,Object? marker = freezed,Object? localPhotos = null,Object? localKey = freezed,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
as String,collectionName: null == collectionName ? _self.collectionName : collectionName // ignore: cast_nullable_to_non_nullable
as String,name: freezed == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String?,description: freezed == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String?,lat: null == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double,lon: null == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double,distanceFromStart: freezed == distanceFromStart ? _self.distanceFromStart : distanceFromStart // ignore: cast_nullable_to_non_nullable
as double?,icon: null == icon ? _self.icon : icon // ignore: cast_nullable_to_non_nullable
as FaIconData,photos: null == photos ? _self.photos : photos // ignore: cast_nullable_to_non_nullable
as List<String>,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,trail: freezed == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as String?,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as DateTime,updated: null == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as DateTime,marker: freezed == marker ? _self.marker : marker // ignore: cast_nullable_to_non_nullable
as dynamic,localPhotos: null == localPhotos ? _self.localPhotos : localPhotos // ignore: cast_nullable_to_non_nullable
as List<String>,localKey: freezed == localKey ? _self.localKey : localKey // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}

}


/// Adds pattern-matching-related methods to [Waypoint].
extension WaypointPatterns on Waypoint {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Waypoint value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Waypoint() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Waypoint value)  $default,){
final _that = this;
switch (_that) {
case _Waypoint():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Waypoint value)?  $default,){
final _that = this;
switch (_that) {
case _Waypoint() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String collectionId,  String collectionName,  String? name,  String? description,  double lat,  double lon, @JsonKey(name: 'distance_from_start')  double? distanceFromStart, @FaIconDataConverter()  FaIconData icon,  List<String> photos,  String author,  String? trail,  DateTime created,  DateTime updated, @JsonKey(includeFromJson: false, includeToJson: false)  dynamic marker, @JsonKey(includeFromJson: false, includeToJson: false)  List<String> localPhotos, @JsonKey(includeFromJson: false, includeToJson: false)  String? localKey)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Waypoint() when $default != null:
return $default(_that.id,_that.collectionId,_that.collectionName,_that.name,_that.description,_that.lat,_that.lon,_that.distanceFromStart,_that.icon,_that.photos,_that.author,_that.trail,_that.created,_that.updated,_that.marker,_that.localPhotos,_that.localKey);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String collectionId,  String collectionName,  String? name,  String? description,  double lat,  double lon, @JsonKey(name: 'distance_from_start')  double? distanceFromStart, @FaIconDataConverter()  FaIconData icon,  List<String> photos,  String author,  String? trail,  DateTime created,  DateTime updated, @JsonKey(includeFromJson: false, includeToJson: false)  dynamic marker, @JsonKey(includeFromJson: false, includeToJson: false)  List<String> localPhotos, @JsonKey(includeFromJson: false, includeToJson: false)  String? localKey)  $default,) {final _that = this;
switch (_that) {
case _Waypoint():
return $default(_that.id,_that.collectionId,_that.collectionName,_that.name,_that.description,_that.lat,_that.lon,_that.distanceFromStart,_that.icon,_that.photos,_that.author,_that.trail,_that.created,_that.updated,_that.marker,_that.localPhotos,_that.localKey);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String collectionId,  String collectionName,  String? name,  String? description,  double lat,  double lon, @JsonKey(name: 'distance_from_start')  double? distanceFromStart, @FaIconDataConverter()  FaIconData icon,  List<String> photos,  String author,  String? trail,  DateTime created,  DateTime updated, @JsonKey(includeFromJson: false, includeToJson: false)  dynamic marker, @JsonKey(includeFromJson: false, includeToJson: false)  List<String> localPhotos, @JsonKey(includeFromJson: false, includeToJson: false)  String? localKey)?  $default,) {final _that = this;
switch (_that) {
case _Waypoint() when $default != null:
return $default(_that.id,_that.collectionId,_that.collectionName,_that.name,_that.description,_that.lat,_that.lon,_that.distanceFromStart,_that.icon,_that.photos,_that.author,_that.trail,_that.created,_that.updated,_that.marker,_that.localPhotos,_that.localKey);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Waypoint extends Waypoint {
  const _Waypoint({required this.id, this.collectionId = 'waypoints', this.collectionName = 'waypoints', this.name = "", this.description = "", required this.lat, required this.lon, @JsonKey(name: 'distance_from_start') this.distanceFromStart, @FaIconDataConverter() this.icon = FontAwesomeIcons.circle, final  List<String> photos = const [], this.author = "000000000000000", this.trail, required this.created, required this.updated, @JsonKey(includeFromJson: false, includeToJson: false) this.marker, @JsonKey(includeFromJson: false, includeToJson: false) final  List<String> localPhotos = const [], @JsonKey(includeFromJson: false, includeToJson: false) this.localKey}): _photos = photos,_localPhotos = localPhotos,super._();
  factory _Waypoint.fromJson(Map<String, dynamic> json) => _$WaypointFromJson(json);

@override final  String id;
@override@JsonKey() final  String collectionId;
@override@JsonKey() final  String collectionName;
@override@JsonKey() final  String? name;
@override@JsonKey() final  String? description;
@override final  double lat;
@override final  double lon;
@override@JsonKey(name: 'distance_from_start') final  double? distanceFromStart;
@override@JsonKey()@FaIconDataConverter() final  FaIconData icon;
 final  List<String> _photos;
@override@JsonKey() List<String> get photos {
  if (_photos is EqualUnmodifiableListView) return _photos;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_photos);
}

@override@JsonKey() final  String author;
@override final  String? trail;
@override final  DateTime created;
@override final  DateTime updated;
// Non-serializable local fields
@override@JsonKey(includeFromJson: false, includeToJson: false) final  dynamic marker;
 final  List<String> _localPhotos;
@override@JsonKey(includeFromJson: false, includeToJson: false) List<String> get localPhotos {
  if (_localPhotos is EqualUnmodifiableListView) return _localPhotos;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_localPhotos);
}

/// Carries list identity for a waypoint that has no server id yet
/// yet. Device-local only, never serialized.
@override@JsonKey(includeFromJson: false, includeToJson: false) final  String? localKey;

/// Create a copy of Waypoint
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$WaypointCopyWith<_Waypoint> get copyWith => __$WaypointCopyWithImpl<_Waypoint>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$WaypointToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _Waypoint&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.collectionName, collectionName) || other.collectionName == collectionName)&&(identical(other.name, name) || other.name == name)&&(identical(other.description, description) || other.description == description)&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon)&&(identical(other.distanceFromStart, distanceFromStart) || other.distanceFromStart == distanceFromStart)&&(identical(other.icon, icon) || other.icon == icon)&&const DeepCollectionEquality().equals(other._photos, _photos)&&(identical(other.author, author) || other.author == author)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&const DeepCollectionEquality().equals(other.marker, marker)&&const DeepCollectionEquality().equals(other._localPhotos, _localPhotos)&&(identical(other.localKey, localKey) || other.localKey == localKey));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,collectionId,collectionName,name,description,lat,lon,distanceFromStart,icon,const DeepCollectionEquality().hash(_photos),author,trail,created,updated,const DeepCollectionEquality().hash(marker),const DeepCollectionEquality().hash(_localPhotos),localKey);

@override
String toString() {
  return 'Waypoint(id: $id, collectionId: $collectionId, collectionName: $collectionName, name: $name, description: $description, lat: $lat, lon: $lon, distanceFromStart: $distanceFromStart, icon: $icon, photos: $photos, author: $author, trail: $trail, created: $created, updated: $updated, marker: $marker, localPhotos: $localPhotos, localKey: $localKey)';
}


}

/// @nodoc
abstract mixin class _$WaypointCopyWith<$Res> implements $WaypointCopyWith<$Res> {
  factory _$WaypointCopyWith(_Waypoint value, $Res Function(_Waypoint) _then) = __$WaypointCopyWithImpl;
@override @useResult
$Res call({
 String id, String collectionId, String collectionName, String? name, String? description, double lat, double lon,@JsonKey(name: 'distance_from_start') double? distanceFromStart,@FaIconDataConverter() FaIconData icon, List<String> photos, String author, String? trail, DateTime created, DateTime updated,@JsonKey(includeFromJson: false, includeToJson: false) dynamic marker,@JsonKey(includeFromJson: false, includeToJson: false) List<String> localPhotos,@JsonKey(includeFromJson: false, includeToJson: false) String? localKey
});




}
/// @nodoc
class __$WaypointCopyWithImpl<$Res>
    implements _$WaypointCopyWith<$Res> {
  __$WaypointCopyWithImpl(this._self, this._then);

  final _Waypoint _self;
  final $Res Function(_Waypoint) _then;

/// Create a copy of Waypoint
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? collectionId = null,Object? collectionName = null,Object? name = freezed,Object? description = freezed,Object? lat = null,Object? lon = null,Object? distanceFromStart = freezed,Object? icon = null,Object? photos = null,Object? author = null,Object? trail = freezed,Object? created = null,Object? updated = null,Object? marker = freezed,Object? localPhotos = null,Object? localKey = freezed,}) {
  return _then(_Waypoint(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
as String,collectionName: null == collectionName ? _self.collectionName : collectionName // ignore: cast_nullable_to_non_nullable
as String,name: freezed == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String?,description: freezed == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String?,lat: null == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double,lon: null == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double,distanceFromStart: freezed == distanceFromStart ? _self.distanceFromStart : distanceFromStart // ignore: cast_nullable_to_non_nullable
as double?,icon: null == icon ? _self.icon : icon // ignore: cast_nullable_to_non_nullable
as FaIconData,photos: null == photos ? _self._photos : photos // ignore: cast_nullable_to_non_nullable
as List<String>,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,trail: freezed == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as String?,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as DateTime,updated: null == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as DateTime,marker: freezed == marker ? _self.marker : marker // ignore: cast_nullable_to_non_nullable
as dynamic,localPhotos: null == localPhotos ? _self._localPhotos : localPhotos // ignore: cast_nullable_to_non_nullable
as List<String>,localKey: freezed == localKey ? _self.localKey : localKey // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}


}

// dart format on
