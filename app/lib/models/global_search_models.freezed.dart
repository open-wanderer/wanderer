// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'global_search_models.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$ListSearchResult {

 String get id; String get author;@JsonKey(name: 'author_name') String get authorName;@JsonKey(name: 'author_avatar') String get authorAvatar; String? get avatar; String get name; String get description;@JsonKey(name: 'elevation_gain') double get elevationGain;@JsonKey(name: 'elevation_loss') double get elevationLoss; double get distance; double get duration; String? get domain; bool get public; int get trails; List<String>? get shares; String? get iri;
/// Create a copy of ListSearchResult
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ListSearchResultCopyWith<ListSearchResult> get copyWith => _$ListSearchResultCopyWithImpl<ListSearchResult>(this as ListSearchResult, _$identity);

  /// Serializes this ListSearchResult to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ListSearchResult&&(identical(other.id, id) || other.id == id)&&(identical(other.author, author) || other.author == author)&&(identical(other.authorName, authorName) || other.authorName == authorName)&&(identical(other.authorAvatar, authorAvatar) || other.authorAvatar == authorAvatar)&&(identical(other.avatar, avatar) || other.avatar == avatar)&&(identical(other.name, name) || other.name == name)&&(identical(other.description, description) || other.description == description)&&(identical(other.elevationGain, elevationGain) || other.elevationGain == elevationGain)&&(identical(other.elevationLoss, elevationLoss) || other.elevationLoss == elevationLoss)&&(identical(other.distance, distance) || other.distance == distance)&&(identical(other.duration, duration) || other.duration == duration)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.public, public) || other.public == public)&&(identical(other.trails, trails) || other.trails == trails)&&const DeepCollectionEquality().equals(other.shares, shares)&&(identical(other.iri, iri) || other.iri == iri));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,author,authorName,authorAvatar,avatar,name,description,elevationGain,elevationLoss,distance,duration,domain,public,trails,const DeepCollectionEquality().hash(shares),iri);

@override
String toString() {
  return 'ListSearchResult(id: $id, author: $author, authorName: $authorName, authorAvatar: $authorAvatar, avatar: $avatar, name: $name, description: $description, elevationGain: $elevationGain, elevationLoss: $elevationLoss, distance: $distance, duration: $duration, domain: $domain, public: $public, trails: $trails, shares: $shares, iri: $iri)';
}


}

/// @nodoc
abstract mixin class $ListSearchResultCopyWith<$Res>  {
  factory $ListSearchResultCopyWith(ListSearchResult value, $Res Function(ListSearchResult) _then) = _$ListSearchResultCopyWithImpl;
@useResult
$Res call({
 String id, String author,@JsonKey(name: 'author_name') String authorName,@JsonKey(name: 'author_avatar') String authorAvatar, String? avatar, String name, String description,@JsonKey(name: 'elevation_gain') double elevationGain,@JsonKey(name: 'elevation_loss') double elevationLoss, double distance, double duration, String? domain, bool public, int trails, List<String>? shares, String? iri
});




}
/// @nodoc
class _$ListSearchResultCopyWithImpl<$Res>
    implements $ListSearchResultCopyWith<$Res> {
  _$ListSearchResultCopyWithImpl(this._self, this._then);

  final ListSearchResult _self;
  final $Res Function(ListSearchResult) _then;

/// Create a copy of ListSearchResult
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? author = null,Object? authorName = null,Object? authorAvatar = null,Object? avatar = freezed,Object? name = null,Object? description = null,Object? elevationGain = null,Object? elevationLoss = null,Object? distance = null,Object? duration = null,Object? domain = freezed,Object? public = null,Object? trails = null,Object? shares = freezed,Object? iri = freezed,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,authorName: null == authorName ? _self.authorName : authorName // ignore: cast_nullable_to_non_nullable
as String,authorAvatar: null == authorAvatar ? _self.authorAvatar : authorAvatar // ignore: cast_nullable_to_non_nullable
as String,avatar: freezed == avatar ? _self.avatar : avatar // ignore: cast_nullable_to_non_nullable
as String?,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,description: null == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String,elevationGain: null == elevationGain ? _self.elevationGain : elevationGain // ignore: cast_nullable_to_non_nullable
as double,elevationLoss: null == elevationLoss ? _self.elevationLoss : elevationLoss // ignore: cast_nullable_to_non_nullable
as double,distance: null == distance ? _self.distance : distance // ignore: cast_nullable_to_non_nullable
as double,duration: null == duration ? _self.duration : duration // ignore: cast_nullable_to_non_nullable
as double,domain: freezed == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String?,public: null == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool,trails: null == trails ? _self.trails : trails // ignore: cast_nullable_to_non_nullable
as int,shares: freezed == shares ? _self.shares : shares // ignore: cast_nullable_to_non_nullable
as List<String>?,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}

}


/// Adds pattern-matching-related methods to [ListSearchResult].
extension ListSearchResultPatterns on ListSearchResult {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ListSearchResult value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ListSearchResult() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ListSearchResult value)  $default,){
final _that = this;
switch (_that) {
case _ListSearchResult():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ListSearchResult value)?  $default,){
final _that = this;
switch (_that) {
case _ListSearchResult() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String author, @JsonKey(name: 'author_name')  String authorName, @JsonKey(name: 'author_avatar')  String authorAvatar,  String? avatar,  String name,  String description, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double distance,  double duration,  String? domain,  bool public,  int trails,  List<String>? shares,  String? iri)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ListSearchResult() when $default != null:
return $default(_that.id,_that.author,_that.authorName,_that.authorAvatar,_that.avatar,_that.name,_that.description,_that.elevationGain,_that.elevationLoss,_that.distance,_that.duration,_that.domain,_that.public,_that.trails,_that.shares,_that.iri);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String author, @JsonKey(name: 'author_name')  String authorName, @JsonKey(name: 'author_avatar')  String authorAvatar,  String? avatar,  String name,  String description, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double distance,  double duration,  String? domain,  bool public,  int trails,  List<String>? shares,  String? iri)  $default,) {final _that = this;
switch (_that) {
case _ListSearchResult():
return $default(_that.id,_that.author,_that.authorName,_that.authorAvatar,_that.avatar,_that.name,_that.description,_that.elevationGain,_that.elevationLoss,_that.distance,_that.duration,_that.domain,_that.public,_that.trails,_that.shares,_that.iri);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String author, @JsonKey(name: 'author_name')  String authorName, @JsonKey(name: 'author_avatar')  String authorAvatar,  String? avatar,  String name,  String description, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double distance,  double duration,  String? domain,  bool public,  int trails,  List<String>? shares,  String? iri)?  $default,) {final _that = this;
switch (_that) {
case _ListSearchResult() when $default != null:
return $default(_that.id,_that.author,_that.authorName,_that.authorAvatar,_that.avatar,_that.name,_that.description,_that.elevationGain,_that.elevationLoss,_that.distance,_that.duration,_that.domain,_that.public,_that.trails,_that.shares,_that.iri);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _ListSearchResult implements ListSearchResult {
  const _ListSearchResult({required this.id, required this.author, @JsonKey(name: 'author_name') required this.authorName, @JsonKey(name: 'author_avatar') required this.authorAvatar, this.avatar, required this.name, required this.description, @JsonKey(name: 'elevation_gain') required this.elevationGain, @JsonKey(name: 'elevation_loss') required this.elevationLoss, required this.distance, required this.duration, this.domain, required this.public, required this.trails, final  List<String>? shares, this.iri}): _shares = shares;
  factory _ListSearchResult.fromJson(Map<String, dynamic> json) => _$ListSearchResultFromJson(json);

@override final  String id;
@override final  String author;
@override@JsonKey(name: 'author_name') final  String authorName;
@override@JsonKey(name: 'author_avatar') final  String authorAvatar;
@override final  String? avatar;
@override final  String name;
@override final  String description;
@override@JsonKey(name: 'elevation_gain') final  double elevationGain;
@override@JsonKey(name: 'elevation_loss') final  double elevationLoss;
@override final  double distance;
@override final  double duration;
@override final  String? domain;
@override final  bool public;
@override final  int trails;
 final  List<String>? _shares;
@override List<String>? get shares {
  final value = _shares;
  if (value == null) return null;
  if (_shares is EqualUnmodifiableListView) return _shares;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}

@override final  String? iri;

/// Create a copy of ListSearchResult
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ListSearchResultCopyWith<_ListSearchResult> get copyWith => __$ListSearchResultCopyWithImpl<_ListSearchResult>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$ListSearchResultToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ListSearchResult&&(identical(other.id, id) || other.id == id)&&(identical(other.author, author) || other.author == author)&&(identical(other.authorName, authorName) || other.authorName == authorName)&&(identical(other.authorAvatar, authorAvatar) || other.authorAvatar == authorAvatar)&&(identical(other.avatar, avatar) || other.avatar == avatar)&&(identical(other.name, name) || other.name == name)&&(identical(other.description, description) || other.description == description)&&(identical(other.elevationGain, elevationGain) || other.elevationGain == elevationGain)&&(identical(other.elevationLoss, elevationLoss) || other.elevationLoss == elevationLoss)&&(identical(other.distance, distance) || other.distance == distance)&&(identical(other.duration, duration) || other.duration == duration)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.public, public) || other.public == public)&&(identical(other.trails, trails) || other.trails == trails)&&const DeepCollectionEquality().equals(other._shares, _shares)&&(identical(other.iri, iri) || other.iri == iri));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,author,authorName,authorAvatar,avatar,name,description,elevationGain,elevationLoss,distance,duration,domain,public,trails,const DeepCollectionEquality().hash(_shares),iri);

@override
String toString() {
  return 'ListSearchResult(id: $id, author: $author, authorName: $authorName, authorAvatar: $authorAvatar, avatar: $avatar, name: $name, description: $description, elevationGain: $elevationGain, elevationLoss: $elevationLoss, distance: $distance, duration: $duration, domain: $domain, public: $public, trails: $trails, shares: $shares, iri: $iri)';
}


}

/// @nodoc
abstract mixin class _$ListSearchResultCopyWith<$Res> implements $ListSearchResultCopyWith<$Res> {
  factory _$ListSearchResultCopyWith(_ListSearchResult value, $Res Function(_ListSearchResult) _then) = __$ListSearchResultCopyWithImpl;
@override @useResult
$Res call({
 String id, String author,@JsonKey(name: 'author_name') String authorName,@JsonKey(name: 'author_avatar') String authorAvatar, String? avatar, String name, String description,@JsonKey(name: 'elevation_gain') double elevationGain,@JsonKey(name: 'elevation_loss') double elevationLoss, double distance, double duration, String? domain, bool public, int trails, List<String>? shares, String? iri
});




}
/// @nodoc
class __$ListSearchResultCopyWithImpl<$Res>
    implements _$ListSearchResultCopyWith<$Res> {
  __$ListSearchResultCopyWithImpl(this._self, this._then);

  final _ListSearchResult _self;
  final $Res Function(_ListSearchResult) _then;

/// Create a copy of ListSearchResult
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? author = null,Object? authorName = null,Object? authorAvatar = null,Object? avatar = freezed,Object? name = null,Object? description = null,Object? elevationGain = null,Object? elevationLoss = null,Object? distance = null,Object? duration = null,Object? domain = freezed,Object? public = null,Object? trails = null,Object? shares = freezed,Object? iri = freezed,}) {
  return _then(_ListSearchResult(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,authorName: null == authorName ? _self.authorName : authorName // ignore: cast_nullable_to_non_nullable
as String,authorAvatar: null == authorAvatar ? _self.authorAvatar : authorAvatar // ignore: cast_nullable_to_non_nullable
as String,avatar: freezed == avatar ? _self.avatar : avatar // ignore: cast_nullable_to_non_nullable
as String?,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,description: null == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String,elevationGain: null == elevationGain ? _self.elevationGain : elevationGain // ignore: cast_nullable_to_non_nullable
as double,elevationLoss: null == elevationLoss ? _self.elevationLoss : elevationLoss // ignore: cast_nullable_to_non_nullable
as double,distance: null == distance ? _self.distance : distance // ignore: cast_nullable_to_non_nullable
as double,duration: null == duration ? _self.duration : duration // ignore: cast_nullable_to_non_nullable
as double,domain: freezed == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String?,public: null == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool,trails: null == trails ? _self.trails : trails // ignore: cast_nullable_to_non_nullable
as int,shares: freezed == shares ? _self._shares : shares // ignore: cast_nullable_to_non_nullable
as List<String>?,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}


}

/// @nodoc
mixin _$LocationSearchResult {

 String get name; String get description; double get lat; double get lon; String get category; String get type;
/// Create a copy of LocationSearchResult
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$LocationSearchResultCopyWith<LocationSearchResult> get copyWith => _$LocationSearchResultCopyWithImpl<LocationSearchResult>(this as LocationSearchResult, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is LocationSearchResult&&(identical(other.name, name) || other.name == name)&&(identical(other.description, description) || other.description == description)&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon)&&(identical(other.category, category) || other.category == category)&&(identical(other.type, type) || other.type == type));
}


@override
int get hashCode => Object.hash(runtimeType,name,description,lat,lon,category,type);

@override
String toString() {
  return 'LocationSearchResult(name: $name, description: $description, lat: $lat, lon: $lon, category: $category, type: $type)';
}


}

/// @nodoc
abstract mixin class $LocationSearchResultCopyWith<$Res>  {
  factory $LocationSearchResultCopyWith(LocationSearchResult value, $Res Function(LocationSearchResult) _then) = _$LocationSearchResultCopyWithImpl;
@useResult
$Res call({
 String name, String description, double lat, double lon, String category, String type
});




}
/// @nodoc
class _$LocationSearchResultCopyWithImpl<$Res>
    implements $LocationSearchResultCopyWith<$Res> {
  _$LocationSearchResultCopyWithImpl(this._self, this._then);

  final LocationSearchResult _self;
  final $Res Function(LocationSearchResult) _then;

/// Create a copy of LocationSearchResult
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? name = null,Object? description = null,Object? lat = null,Object? lon = null,Object? category = null,Object? type = null,}) {
  return _then(_self.copyWith(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,description: null == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String,lat: null == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double,lon: null == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double,category: null == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as String,type: null == type ? _self.type : type // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

}


/// Adds pattern-matching-related methods to [LocationSearchResult].
extension LocationSearchResultPatterns on LocationSearchResult {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _LocationSearchResult value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _LocationSearchResult() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _LocationSearchResult value)  $default,){
final _that = this;
switch (_that) {
case _LocationSearchResult():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _LocationSearchResult value)?  $default,){
final _that = this;
switch (_that) {
case _LocationSearchResult() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String name,  String description,  double lat,  double lon,  String category,  String type)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _LocationSearchResult() when $default != null:
return $default(_that.name,_that.description,_that.lat,_that.lon,_that.category,_that.type);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String name,  String description,  double lat,  double lon,  String category,  String type)  $default,) {final _that = this;
switch (_that) {
case _LocationSearchResult():
return $default(_that.name,_that.description,_that.lat,_that.lon,_that.category,_that.type);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String name,  String description,  double lat,  double lon,  String category,  String type)?  $default,) {final _that = this;
switch (_that) {
case _LocationSearchResult() when $default != null:
return $default(_that.name,_that.description,_that.lat,_that.lon,_that.category,_that.type);case _:
  return null;

}
}

}

/// @nodoc


class _LocationSearchResult implements LocationSearchResult {
  const _LocationSearchResult({required this.name, required this.description, required this.lat, required this.lon, required this.category, required this.type});
  

@override final  String name;
@override final  String description;
@override final  double lat;
@override final  double lon;
@override final  String category;
@override final  String type;

/// Create a copy of LocationSearchResult
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$LocationSearchResultCopyWith<_LocationSearchResult> get copyWith => __$LocationSearchResultCopyWithImpl<_LocationSearchResult>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _LocationSearchResult&&(identical(other.name, name) || other.name == name)&&(identical(other.description, description) || other.description == description)&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon)&&(identical(other.category, category) || other.category == category)&&(identical(other.type, type) || other.type == type));
}


@override
int get hashCode => Object.hash(runtimeType,name,description,lat,lon,category,type);

@override
String toString() {
  return 'LocationSearchResult(name: $name, description: $description, lat: $lat, lon: $lon, category: $category, type: $type)';
}


}

/// @nodoc
abstract mixin class _$LocationSearchResultCopyWith<$Res> implements $LocationSearchResultCopyWith<$Res> {
  factory _$LocationSearchResultCopyWith(_LocationSearchResult value, $Res Function(_LocationSearchResult) _then) = __$LocationSearchResultCopyWithImpl;
@override @useResult
$Res call({
 String name, String description, double lat, double lon, String category, String type
});




}
/// @nodoc
class __$LocationSearchResultCopyWithImpl<$Res>
    implements _$LocationSearchResultCopyWith<$Res> {
  __$LocationSearchResultCopyWithImpl(this._self, this._then);

  final _LocationSearchResult _self;
  final $Res Function(_LocationSearchResult) _then;

/// Create a copy of LocationSearchResult
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? name = null,Object? description = null,Object? lat = null,Object? lon = null,Object? category = null,Object? type = null,}) {
  return _then(_LocationSearchResult(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,description: null == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String,lat: null == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double,lon: null == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double,category: null == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as String,type: null == type ? _self.type : type // ignore: cast_nullable_to_non_nullable
as String,
  ));
}


}


/// @nodoc
mixin _$SearchActor {

 String get id; String get username;@JsonKey(name: 'preferred_username') String get preferredUsername; String? get domain; String? get icon; String get iri;
/// Create a copy of SearchActor
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SearchActorCopyWith<SearchActor> get copyWith => _$SearchActorCopyWithImpl<SearchActor>(this as SearchActor, _$identity);

  /// Serializes this SearchActor to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is SearchActor&&(identical(other.id, id) || other.id == id)&&(identical(other.username, username) || other.username == username)&&(identical(other.preferredUsername, preferredUsername) || other.preferredUsername == preferredUsername)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.icon, icon) || other.icon == icon)&&(identical(other.iri, iri) || other.iri == iri));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,username,preferredUsername,domain,icon,iri);

@override
String toString() {
  return 'SearchActor(id: $id, username: $username, preferredUsername: $preferredUsername, domain: $domain, icon: $icon, iri: $iri)';
}


}

/// @nodoc
abstract mixin class $SearchActorCopyWith<$Res>  {
  factory $SearchActorCopyWith(SearchActor value, $Res Function(SearchActor) _then) = _$SearchActorCopyWithImpl;
@useResult
$Res call({
 String id, String username,@JsonKey(name: 'preferred_username') String preferredUsername, String? domain, String? icon, String iri
});




}
/// @nodoc
class _$SearchActorCopyWithImpl<$Res>
    implements $SearchActorCopyWith<$Res> {
  _$SearchActorCopyWithImpl(this._self, this._then);

  final SearchActor _self;
  final $Res Function(SearchActor) _then;

/// Create a copy of SearchActor
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? username = null,Object? preferredUsername = null,Object? domain = freezed,Object? icon = freezed,Object? iri = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,preferredUsername: null == preferredUsername ? _self.preferredUsername : preferredUsername // ignore: cast_nullable_to_non_nullable
as String,domain: freezed == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String?,icon: freezed == icon ? _self.icon : icon // ignore: cast_nullable_to_non_nullable
as String?,iri: null == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

}


/// Adds pattern-matching-related methods to [SearchActor].
extension SearchActorPatterns on SearchActor {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _SearchActor value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _SearchActor() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _SearchActor value)  $default,){
final _that = this;
switch (_that) {
case _SearchActor():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _SearchActor value)?  $default,){
final _that = this;
switch (_that) {
case _SearchActor() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String username, @JsonKey(name: 'preferred_username')  String preferredUsername,  String? domain,  String? icon,  String iri)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _SearchActor() when $default != null:
return $default(_that.id,_that.username,_that.preferredUsername,_that.domain,_that.icon,_that.iri);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String username, @JsonKey(name: 'preferred_username')  String preferredUsername,  String? domain,  String? icon,  String iri)  $default,) {final _that = this;
switch (_that) {
case _SearchActor():
return $default(_that.id,_that.username,_that.preferredUsername,_that.domain,_that.icon,_that.iri);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String username, @JsonKey(name: 'preferred_username')  String preferredUsername,  String? domain,  String? icon,  String iri)?  $default,) {final _that = this;
switch (_that) {
case _SearchActor() when $default != null:
return $default(_that.id,_that.username,_that.preferredUsername,_that.domain,_that.icon,_that.iri);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _SearchActor implements SearchActor {
  const _SearchActor({this.id = '', this.username = '', @JsonKey(name: 'preferred_username') this.preferredUsername = '', this.domain, this.icon, this.iri = ''});
  factory _SearchActor.fromJson(Map<String, dynamic> json) => _$SearchActorFromJson(json);

@override@JsonKey() final  String id;
@override@JsonKey() final  String username;
@override@JsonKey(name: 'preferred_username') final  String preferredUsername;
@override final  String? domain;
@override final  String? icon;
@override@JsonKey() final  String iri;

/// Create a copy of SearchActor
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SearchActorCopyWith<_SearchActor> get copyWith => __$SearchActorCopyWithImpl<_SearchActor>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SearchActorToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _SearchActor&&(identical(other.id, id) || other.id == id)&&(identical(other.username, username) || other.username == username)&&(identical(other.preferredUsername, preferredUsername) || other.preferredUsername == preferredUsername)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.icon, icon) || other.icon == icon)&&(identical(other.iri, iri) || other.iri == iri));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,username,preferredUsername,domain,icon,iri);

@override
String toString() {
  return 'SearchActor(id: $id, username: $username, preferredUsername: $preferredUsername, domain: $domain, icon: $icon, iri: $iri)';
}


}

/// @nodoc
abstract mixin class _$SearchActorCopyWith<$Res> implements $SearchActorCopyWith<$Res> {
  factory _$SearchActorCopyWith(_SearchActor value, $Res Function(_SearchActor) _then) = __$SearchActorCopyWithImpl;
@override @useResult
$Res call({
 String id, String username,@JsonKey(name: 'preferred_username') String preferredUsername, String? domain, String? icon, String iri
});




}
/// @nodoc
class __$SearchActorCopyWithImpl<$Res>
    implements _$SearchActorCopyWith<$Res> {
  __$SearchActorCopyWithImpl(this._self, this._then);

  final _SearchActor _self;
  final $Res Function(_SearchActor) _then;

/// Create a copy of SearchActor
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? username = null,Object? preferredUsername = null,Object? domain = freezed,Object? icon = freezed,Object? iri = null,}) {
  return _then(_SearchActor(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,preferredUsername: null == preferredUsername ? _self.preferredUsername : preferredUsername // ignore: cast_nullable_to_non_nullable
as String,domain: freezed == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String?,icon: freezed == icon ? _self.icon : icon // ignore: cast_nullable_to_non_nullable
as String?,iri: null == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String,
  ));
}


}

// dart format on
