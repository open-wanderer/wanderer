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
mixin _$GeoLocation {

 double get lat; double get lng;
/// Create a copy of GeoLocation
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$GeoLocationCopyWith<GeoLocation> get copyWith => _$GeoLocationCopyWithImpl<GeoLocation>(this as GeoLocation, _$identity);

  /// Serializes this GeoLocation to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is GeoLocation&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lng, lng) || other.lng == lng));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,lat,lng);

@override
String toString() {
  return 'GeoLocation(lat: $lat, lng: $lng)';
}


}

/// @nodoc
abstract mixin class $GeoLocationCopyWith<$Res>  {
  factory $GeoLocationCopyWith(GeoLocation value, $Res Function(GeoLocation) _then) = _$GeoLocationCopyWithImpl;
@useResult
$Res call({
 double lat, double lng
});




}
/// @nodoc
class _$GeoLocationCopyWithImpl<$Res>
    implements $GeoLocationCopyWith<$Res> {
  _$GeoLocationCopyWithImpl(this._self, this._then);

  final GeoLocation _self;
  final $Res Function(GeoLocation) _then;

/// Create a copy of GeoLocation
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? lat = null,Object? lng = null,}) {
  return _then(_self.copyWith(
lat: null == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double,lng: null == lng ? _self.lng : lng // ignore: cast_nullable_to_non_nullable
as double,
  ));
}

}


/// Adds pattern-matching-related methods to [GeoLocation].
extension GeoLocationPatterns on GeoLocation {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _GeoLocation value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _GeoLocation() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _GeoLocation value)  $default,){
final _that = this;
switch (_that) {
case _GeoLocation():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _GeoLocation value)?  $default,){
final _that = this;
switch (_that) {
case _GeoLocation() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( double lat,  double lng)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _GeoLocation() when $default != null:
return $default(_that.lat,_that.lng);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( double lat,  double lng)  $default,) {final _that = this;
switch (_that) {
case _GeoLocation():
return $default(_that.lat,_that.lng);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( double lat,  double lng)?  $default,) {final _that = this;
switch (_that) {
case _GeoLocation() when $default != null:
return $default(_that.lat,_that.lng);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _GeoLocation implements GeoLocation {
  const _GeoLocation({required this.lat, required this.lng});
  factory _GeoLocation.fromJson(Map<String, dynamic> json) => _$GeoLocationFromJson(json);

@override final  double lat;
@override final  double lng;

/// Create a copy of GeoLocation
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$GeoLocationCopyWith<_GeoLocation> get copyWith => __$GeoLocationCopyWithImpl<_GeoLocation>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$GeoLocationToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _GeoLocation&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lng, lng) || other.lng == lng));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,lat,lng);

@override
String toString() {
  return 'GeoLocation(lat: $lat, lng: $lng)';
}


}

/// @nodoc
abstract mixin class _$GeoLocationCopyWith<$Res> implements $GeoLocationCopyWith<$Res> {
  factory _$GeoLocationCopyWith(_GeoLocation value, $Res Function(_GeoLocation) _then) = __$GeoLocationCopyWithImpl;
@override @useResult
$Res call({
 double lat, double lng
});




}
/// @nodoc
class __$GeoLocationCopyWithImpl<$Res>
    implements _$GeoLocationCopyWith<$Res> {
  __$GeoLocationCopyWithImpl(this._self, this._then);

  final _GeoLocation _self;
  final $Res Function(_GeoLocation) _then;

/// Create a copy of GeoLocation
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? lat = null,Object? lng = null,}) {
  return _then(_GeoLocation(
lat: null == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double,lng: null == lng ? _self.lng : lng // ignore: cast_nullable_to_non_nullable
as double,
  ));
}


}


/// @nodoc
mixin _$TrailSearchResult {

 String get id; String get collectionId; String get author;@JsonKey(name: 'author_name') String get authorName;@JsonKey(name: 'author_avatar') String get authorAvatar; String get name; String get description; String get location; double get distance;@JsonKey(name: 'elevation_gain') double get elevationGain;@JsonKey(name: 'elevation_loss') double get elevationLoss; double get duration; int get difficulty;// 0 | 1 | 2
@JsonKey(name: 'category_id') String? get categoryId;@JsonKey(name: 'subcategory_id') String? get subcategoryId; bool get completed; int get date; int get created; bool get public; String get thumbnail; String? get polyline; List<String>? get likes;@JsonKey(name: 'like_count') int get likeCount; List<String>? get shares; List<String>? get tags; String? get domain; String? get iri; String get gpx;@JsonKey(name: '_geo') GeoLocation get geo;
/// Create a copy of TrailSearchResult
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailSearchResultCopyWith<TrailSearchResult> get copyWith => _$TrailSearchResultCopyWithImpl<TrailSearchResult>(this as TrailSearchResult, _$identity);

  /// Serializes this TrailSearchResult to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailSearchResult&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.author, author) || other.author == author)&&(identical(other.authorName, authorName) || other.authorName == authorName)&&(identical(other.authorAvatar, authorAvatar) || other.authorAvatar == authorAvatar)&&(identical(other.name, name) || other.name == name)&&(identical(other.description, description) || other.description == description)&&(identical(other.location, location) || other.location == location)&&(identical(other.distance, distance) || other.distance == distance)&&(identical(other.elevationGain, elevationGain) || other.elevationGain == elevationGain)&&(identical(other.elevationLoss, elevationLoss) || other.elevationLoss == elevationLoss)&&(identical(other.duration, duration) || other.duration == duration)&&(identical(other.difficulty, difficulty) || other.difficulty == difficulty)&&(identical(other.categoryId, categoryId) || other.categoryId == categoryId)&&(identical(other.subcategoryId, subcategoryId) || other.subcategoryId == subcategoryId)&&(identical(other.completed, completed) || other.completed == completed)&&(identical(other.date, date) || other.date == date)&&(identical(other.created, created) || other.created == created)&&(identical(other.public, public) || other.public == public)&&(identical(other.thumbnail, thumbnail) || other.thumbnail == thumbnail)&&(identical(other.polyline, polyline) || other.polyline == polyline)&&const DeepCollectionEquality().equals(other.likes, likes)&&(identical(other.likeCount, likeCount) || other.likeCount == likeCount)&&const DeepCollectionEquality().equals(other.shares, shares)&&const DeepCollectionEquality().equals(other.tags, tags)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.gpx, gpx) || other.gpx == gpx)&&(identical(other.geo, geo) || other.geo == geo));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hashAll([runtimeType,id,collectionId,author,authorName,authorAvatar,name,description,location,distance,elevationGain,elevationLoss,duration,difficulty,categoryId,subcategoryId,completed,date,created,public,thumbnail,polyline,const DeepCollectionEquality().hash(likes),likeCount,const DeepCollectionEquality().hash(shares),const DeepCollectionEquality().hash(tags),domain,iri,gpx,geo]);

@override
String toString() {
  return 'TrailSearchResult(id: $id, collectionId: $collectionId, author: $author, authorName: $authorName, authorAvatar: $authorAvatar, name: $name, description: $description, location: $location, distance: $distance, elevationGain: $elevationGain, elevationLoss: $elevationLoss, duration: $duration, difficulty: $difficulty, categoryId: $categoryId, subcategoryId: $subcategoryId, completed: $completed, date: $date, created: $created, public: $public, thumbnail: $thumbnail, polyline: $polyline, likes: $likes, likeCount: $likeCount, shares: $shares, tags: $tags, domain: $domain, iri: $iri, gpx: $gpx, geo: $geo)';
}


}

/// @nodoc
abstract mixin class $TrailSearchResultCopyWith<$Res>  {
  factory $TrailSearchResultCopyWith(TrailSearchResult value, $Res Function(TrailSearchResult) _then) = _$TrailSearchResultCopyWithImpl;
@useResult
$Res call({
 String id, String collectionId, String author,@JsonKey(name: 'author_name') String authorName,@JsonKey(name: 'author_avatar') String authorAvatar, String name, String description, String location, double distance,@JsonKey(name: 'elevation_gain') double elevationGain,@JsonKey(name: 'elevation_loss') double elevationLoss, double duration, int difficulty,@JsonKey(name: 'category_id') String? categoryId,@JsonKey(name: 'subcategory_id') String? subcategoryId, bool completed, int date, int created, bool public, String thumbnail, String? polyline, List<String>? likes,@JsonKey(name: 'like_count') int likeCount, List<String>? shares, List<String>? tags, String? domain, String? iri, String gpx,@JsonKey(name: '_geo') GeoLocation geo
});


$GeoLocationCopyWith<$Res> get geo;

}
/// @nodoc
class _$TrailSearchResultCopyWithImpl<$Res>
    implements $TrailSearchResultCopyWith<$Res> {
  _$TrailSearchResultCopyWithImpl(this._self, this._then);

  final TrailSearchResult _self;
  final $Res Function(TrailSearchResult) _then;

/// Create a copy of TrailSearchResult
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? collectionId = null,Object? author = null,Object? authorName = null,Object? authorAvatar = null,Object? name = null,Object? description = null,Object? location = null,Object? distance = null,Object? elevationGain = null,Object? elevationLoss = null,Object? duration = null,Object? difficulty = null,Object? categoryId = freezed,Object? subcategoryId = freezed,Object? completed = null,Object? date = null,Object? created = null,Object? public = null,Object? thumbnail = null,Object? polyline = freezed,Object? likes = freezed,Object? likeCount = null,Object? shares = freezed,Object? tags = freezed,Object? domain = freezed,Object? iri = freezed,Object? gpx = null,Object? geo = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
as String,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,authorName: null == authorName ? _self.authorName : authorName // ignore: cast_nullable_to_non_nullable
as String,authorAvatar: null == authorAvatar ? _self.authorAvatar : authorAvatar // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,description: null == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String,location: null == location ? _self.location : location // ignore: cast_nullable_to_non_nullable
as String,distance: null == distance ? _self.distance : distance // ignore: cast_nullable_to_non_nullable
as double,elevationGain: null == elevationGain ? _self.elevationGain : elevationGain // ignore: cast_nullable_to_non_nullable
as double,elevationLoss: null == elevationLoss ? _self.elevationLoss : elevationLoss // ignore: cast_nullable_to_non_nullable
as double,duration: null == duration ? _self.duration : duration // ignore: cast_nullable_to_non_nullable
as double,difficulty: null == difficulty ? _self.difficulty : difficulty // ignore: cast_nullable_to_non_nullable
as int,categoryId: freezed == categoryId ? _self.categoryId : categoryId // ignore: cast_nullable_to_non_nullable
as String?,subcategoryId: freezed == subcategoryId ? _self.subcategoryId : subcategoryId // ignore: cast_nullable_to_non_nullable
as String?,completed: null == completed ? _self.completed : completed // ignore: cast_nullable_to_non_nullable
as bool,date: null == date ? _self.date : date // ignore: cast_nullable_to_non_nullable
as int,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as int,public: null == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool,thumbnail: null == thumbnail ? _self.thumbnail : thumbnail // ignore: cast_nullable_to_non_nullable
as String,polyline: freezed == polyline ? _self.polyline : polyline // ignore: cast_nullable_to_non_nullable
as String?,likes: freezed == likes ? _self.likes : likes // ignore: cast_nullable_to_non_nullable
as List<String>?,likeCount: null == likeCount ? _self.likeCount : likeCount // ignore: cast_nullable_to_non_nullable
as int,shares: freezed == shares ? _self.shares : shares // ignore: cast_nullable_to_non_nullable
as List<String>?,tags: freezed == tags ? _self.tags : tags // ignore: cast_nullable_to_non_nullable
as List<String>?,domain: freezed == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String?,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,gpx: null == gpx ? _self.gpx : gpx // ignore: cast_nullable_to_non_nullable
as String,geo: null == geo ? _self.geo : geo // ignore: cast_nullable_to_non_nullable
as GeoLocation,
  ));
}
/// Create a copy of TrailSearchResult
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$GeoLocationCopyWith<$Res> get geo {
  
  return $GeoLocationCopyWith<$Res>(_self.geo, (value) {
    return _then(_self.copyWith(geo: value));
  });
}
}


/// Adds pattern-matching-related methods to [TrailSearchResult].
extension TrailSearchResultPatterns on TrailSearchResult {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailSearchResult value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailSearchResult() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailSearchResult value)  $default,){
final _that = this;
switch (_that) {
case _TrailSearchResult():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailSearchResult value)?  $default,){
final _that = this;
switch (_that) {
case _TrailSearchResult() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String collectionId,  String author, @JsonKey(name: 'author_name')  String authorName, @JsonKey(name: 'author_avatar')  String authorAvatar,  String name,  String description,  String location,  double distance, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double duration,  int difficulty, @JsonKey(name: 'category_id')  String? categoryId, @JsonKey(name: 'subcategory_id')  String? subcategoryId,  bool completed,  int date,  int created,  bool public,  String thumbnail,  String? polyline,  List<String>? likes, @JsonKey(name: 'like_count')  int likeCount,  List<String>? shares,  List<String>? tags,  String? domain,  String? iri,  String gpx, @JsonKey(name: '_geo')  GeoLocation geo)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailSearchResult() when $default != null:
return $default(_that.id,_that.collectionId,_that.author,_that.authorName,_that.authorAvatar,_that.name,_that.description,_that.location,_that.distance,_that.elevationGain,_that.elevationLoss,_that.duration,_that.difficulty,_that.categoryId,_that.subcategoryId,_that.completed,_that.date,_that.created,_that.public,_that.thumbnail,_that.polyline,_that.likes,_that.likeCount,_that.shares,_that.tags,_that.domain,_that.iri,_that.gpx,_that.geo);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String collectionId,  String author, @JsonKey(name: 'author_name')  String authorName, @JsonKey(name: 'author_avatar')  String authorAvatar,  String name,  String description,  String location,  double distance, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double duration,  int difficulty, @JsonKey(name: 'category_id')  String? categoryId, @JsonKey(name: 'subcategory_id')  String? subcategoryId,  bool completed,  int date,  int created,  bool public,  String thumbnail,  String? polyline,  List<String>? likes, @JsonKey(name: 'like_count')  int likeCount,  List<String>? shares,  List<String>? tags,  String? domain,  String? iri,  String gpx, @JsonKey(name: '_geo')  GeoLocation geo)  $default,) {final _that = this;
switch (_that) {
case _TrailSearchResult():
return $default(_that.id,_that.collectionId,_that.author,_that.authorName,_that.authorAvatar,_that.name,_that.description,_that.location,_that.distance,_that.elevationGain,_that.elevationLoss,_that.duration,_that.difficulty,_that.categoryId,_that.subcategoryId,_that.completed,_that.date,_that.created,_that.public,_that.thumbnail,_that.polyline,_that.likes,_that.likeCount,_that.shares,_that.tags,_that.domain,_that.iri,_that.gpx,_that.geo);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String collectionId,  String author, @JsonKey(name: 'author_name')  String authorName, @JsonKey(name: 'author_avatar')  String authorAvatar,  String name,  String description,  String location,  double distance, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double duration,  int difficulty, @JsonKey(name: 'category_id')  String? categoryId, @JsonKey(name: 'subcategory_id')  String? subcategoryId,  bool completed,  int date,  int created,  bool public,  String thumbnail,  String? polyline,  List<String>? likes, @JsonKey(name: 'like_count')  int likeCount,  List<String>? shares,  List<String>? tags,  String? domain,  String? iri,  String gpx, @JsonKey(name: '_geo')  GeoLocation geo)?  $default,) {final _that = this;
switch (_that) {
case _TrailSearchResult() when $default != null:
return $default(_that.id,_that.collectionId,_that.author,_that.authorName,_that.authorAvatar,_that.name,_that.description,_that.location,_that.distance,_that.elevationGain,_that.elevationLoss,_that.duration,_that.difficulty,_that.categoryId,_that.subcategoryId,_that.completed,_that.date,_that.created,_that.public,_that.thumbnail,_that.polyline,_that.likes,_that.likeCount,_that.shares,_that.tags,_that.domain,_that.iri,_that.gpx,_that.geo);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _TrailSearchResult extends TrailSearchResult {
  const _TrailSearchResult({required this.id, this.collectionId = 'trails', required this.author, @JsonKey(name: 'author_name') required this.authorName, @JsonKey(name: 'author_avatar') required this.authorAvatar, required this.name, required this.description, required this.location, required this.distance, @JsonKey(name: 'elevation_gain') required this.elevationGain, @JsonKey(name: 'elevation_loss') required this.elevationLoss, required this.duration, required this.difficulty, @JsonKey(name: 'category_id') this.categoryId, @JsonKey(name: 'subcategory_id') this.subcategoryId, required this.completed, required this.date, required this.created, required this.public, required this.thumbnail, this.polyline, final  List<String>? likes, @JsonKey(name: 'like_count') required this.likeCount, final  List<String>? shares, final  List<String>? tags, this.domain, this.iri, required this.gpx, @JsonKey(name: '_geo') required this.geo}): _likes = likes,_shares = shares,_tags = tags,super._();
  factory _TrailSearchResult.fromJson(Map<String, dynamic> json) => _$TrailSearchResultFromJson(json);

@override final  String id;
@override@JsonKey() final  String collectionId;
@override final  String author;
@override@JsonKey(name: 'author_name') final  String authorName;
@override@JsonKey(name: 'author_avatar') final  String authorAvatar;
@override final  String name;
@override final  String description;
@override final  String location;
@override final  double distance;
@override@JsonKey(name: 'elevation_gain') final  double elevationGain;
@override@JsonKey(name: 'elevation_loss') final  double elevationLoss;
@override final  double duration;
@override final  int difficulty;
// 0 | 1 | 2
@override@JsonKey(name: 'category_id') final  String? categoryId;
@override@JsonKey(name: 'subcategory_id') final  String? subcategoryId;
@override final  bool completed;
@override final  int date;
@override final  int created;
@override final  bool public;
@override final  String thumbnail;
@override final  String? polyline;
 final  List<String>? _likes;
@override List<String>? get likes {
  final value = _likes;
  if (value == null) return null;
  if (_likes is EqualUnmodifiableListView) return _likes;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}

@override@JsonKey(name: 'like_count') final  int likeCount;
 final  List<String>? _shares;
@override List<String>? get shares {
  final value = _shares;
  if (value == null) return null;
  if (_shares is EqualUnmodifiableListView) return _shares;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}

 final  List<String>? _tags;
@override List<String>? get tags {
  final value = _tags;
  if (value == null) return null;
  if (_tags is EqualUnmodifiableListView) return _tags;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}

@override final  String? domain;
@override final  String? iri;
@override final  String gpx;
@override@JsonKey(name: '_geo') final  GeoLocation geo;

/// Create a copy of TrailSearchResult
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailSearchResultCopyWith<_TrailSearchResult> get copyWith => __$TrailSearchResultCopyWithImpl<_TrailSearchResult>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$TrailSearchResultToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailSearchResult&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.author, author) || other.author == author)&&(identical(other.authorName, authorName) || other.authorName == authorName)&&(identical(other.authorAvatar, authorAvatar) || other.authorAvatar == authorAvatar)&&(identical(other.name, name) || other.name == name)&&(identical(other.description, description) || other.description == description)&&(identical(other.location, location) || other.location == location)&&(identical(other.distance, distance) || other.distance == distance)&&(identical(other.elevationGain, elevationGain) || other.elevationGain == elevationGain)&&(identical(other.elevationLoss, elevationLoss) || other.elevationLoss == elevationLoss)&&(identical(other.duration, duration) || other.duration == duration)&&(identical(other.difficulty, difficulty) || other.difficulty == difficulty)&&(identical(other.categoryId, categoryId) || other.categoryId == categoryId)&&(identical(other.subcategoryId, subcategoryId) || other.subcategoryId == subcategoryId)&&(identical(other.completed, completed) || other.completed == completed)&&(identical(other.date, date) || other.date == date)&&(identical(other.created, created) || other.created == created)&&(identical(other.public, public) || other.public == public)&&(identical(other.thumbnail, thumbnail) || other.thumbnail == thumbnail)&&(identical(other.polyline, polyline) || other.polyline == polyline)&&const DeepCollectionEquality().equals(other._likes, _likes)&&(identical(other.likeCount, likeCount) || other.likeCount == likeCount)&&const DeepCollectionEquality().equals(other._shares, _shares)&&const DeepCollectionEquality().equals(other._tags, _tags)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.gpx, gpx) || other.gpx == gpx)&&(identical(other.geo, geo) || other.geo == geo));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hashAll([runtimeType,id,collectionId,author,authorName,authorAvatar,name,description,location,distance,elevationGain,elevationLoss,duration,difficulty,categoryId,subcategoryId,completed,date,created,public,thumbnail,polyline,const DeepCollectionEquality().hash(_likes),likeCount,const DeepCollectionEquality().hash(_shares),const DeepCollectionEquality().hash(_tags),domain,iri,gpx,geo]);

@override
String toString() {
  return 'TrailSearchResult(id: $id, collectionId: $collectionId, author: $author, authorName: $authorName, authorAvatar: $authorAvatar, name: $name, description: $description, location: $location, distance: $distance, elevationGain: $elevationGain, elevationLoss: $elevationLoss, duration: $duration, difficulty: $difficulty, categoryId: $categoryId, subcategoryId: $subcategoryId, completed: $completed, date: $date, created: $created, public: $public, thumbnail: $thumbnail, polyline: $polyline, likes: $likes, likeCount: $likeCount, shares: $shares, tags: $tags, domain: $domain, iri: $iri, gpx: $gpx, geo: $geo)';
}


}

/// @nodoc
abstract mixin class _$TrailSearchResultCopyWith<$Res> implements $TrailSearchResultCopyWith<$Res> {
  factory _$TrailSearchResultCopyWith(_TrailSearchResult value, $Res Function(_TrailSearchResult) _then) = __$TrailSearchResultCopyWithImpl;
@override @useResult
$Res call({
 String id, String collectionId, String author,@JsonKey(name: 'author_name') String authorName,@JsonKey(name: 'author_avatar') String authorAvatar, String name, String description, String location, double distance,@JsonKey(name: 'elevation_gain') double elevationGain,@JsonKey(name: 'elevation_loss') double elevationLoss, double duration, int difficulty,@JsonKey(name: 'category_id') String? categoryId,@JsonKey(name: 'subcategory_id') String? subcategoryId, bool completed, int date, int created, bool public, String thumbnail, String? polyline, List<String>? likes,@JsonKey(name: 'like_count') int likeCount, List<String>? shares, List<String>? tags, String? domain, String? iri, String gpx,@JsonKey(name: '_geo') GeoLocation geo
});


@override $GeoLocationCopyWith<$Res> get geo;

}
/// @nodoc
class __$TrailSearchResultCopyWithImpl<$Res>
    implements _$TrailSearchResultCopyWith<$Res> {
  __$TrailSearchResultCopyWithImpl(this._self, this._then);

  final _TrailSearchResult _self;
  final $Res Function(_TrailSearchResult) _then;

/// Create a copy of TrailSearchResult
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? collectionId = null,Object? author = null,Object? authorName = null,Object? authorAvatar = null,Object? name = null,Object? description = null,Object? location = null,Object? distance = null,Object? elevationGain = null,Object? elevationLoss = null,Object? duration = null,Object? difficulty = null,Object? categoryId = freezed,Object? subcategoryId = freezed,Object? completed = null,Object? date = null,Object? created = null,Object? public = null,Object? thumbnail = null,Object? polyline = freezed,Object? likes = freezed,Object? likeCount = null,Object? shares = freezed,Object? tags = freezed,Object? domain = freezed,Object? iri = freezed,Object? gpx = null,Object? geo = null,}) {
  return _then(_TrailSearchResult(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
as String,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,authorName: null == authorName ? _self.authorName : authorName // ignore: cast_nullable_to_non_nullable
as String,authorAvatar: null == authorAvatar ? _self.authorAvatar : authorAvatar // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,description: null == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String,location: null == location ? _self.location : location // ignore: cast_nullable_to_non_nullable
as String,distance: null == distance ? _self.distance : distance // ignore: cast_nullable_to_non_nullable
as double,elevationGain: null == elevationGain ? _self.elevationGain : elevationGain // ignore: cast_nullable_to_non_nullable
as double,elevationLoss: null == elevationLoss ? _self.elevationLoss : elevationLoss // ignore: cast_nullable_to_non_nullable
as double,duration: null == duration ? _self.duration : duration // ignore: cast_nullable_to_non_nullable
as double,difficulty: null == difficulty ? _self.difficulty : difficulty // ignore: cast_nullable_to_non_nullable
as int,categoryId: freezed == categoryId ? _self.categoryId : categoryId // ignore: cast_nullable_to_non_nullable
as String?,subcategoryId: freezed == subcategoryId ? _self.subcategoryId : subcategoryId // ignore: cast_nullable_to_non_nullable
as String?,completed: null == completed ? _self.completed : completed // ignore: cast_nullable_to_non_nullable
as bool,date: null == date ? _self.date : date // ignore: cast_nullable_to_non_nullable
as int,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as int,public: null == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool,thumbnail: null == thumbnail ? _self.thumbnail : thumbnail // ignore: cast_nullable_to_non_nullable
as String,polyline: freezed == polyline ? _self.polyline : polyline // ignore: cast_nullable_to_non_nullable
as String?,likes: freezed == likes ? _self._likes : likes // ignore: cast_nullable_to_non_nullable
as List<String>?,likeCount: null == likeCount ? _self.likeCount : likeCount // ignore: cast_nullable_to_non_nullable
as int,shares: freezed == shares ? _self._shares : shares // ignore: cast_nullable_to_non_nullable
as List<String>?,tags: freezed == tags ? _self._tags : tags // ignore: cast_nullable_to_non_nullable
as List<String>?,domain: freezed == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String?,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,gpx: null == gpx ? _self.gpx : gpx // ignore: cast_nullable_to_non_nullable
as String,geo: null == geo ? _self.geo : geo // ignore: cast_nullable_to_non_nullable
as GeoLocation,
  ));
}

/// Create a copy of TrailSearchResult
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$GeoLocationCopyWith<$Res> get geo {
  
  return $GeoLocationCopyWith<$Res>(_self.geo, (value) {
    return _then(_self.copyWith(geo: value));
  });
}
}


/// @nodoc
mixin _$ListSearchResult {

 String get id; String get collectionId; String get author;@JsonKey(name: 'author_name') String get authorName;@JsonKey(name: 'author_avatar') String get authorAvatar; String? get avatar; String get name; String get description;@JsonKey(name: 'elevation_gain') double get elevationGain;@JsonKey(name: 'elevation_loss') double get elevationLoss; double get distance; double get duration; String? get domain; bool get public; int get trails; List<String>? get shares; String? get iri;
/// Create a copy of ListSearchResult
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ListSearchResultCopyWith<ListSearchResult> get copyWith => _$ListSearchResultCopyWithImpl<ListSearchResult>(this as ListSearchResult, _$identity);

  /// Serializes this ListSearchResult to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ListSearchResult&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.author, author) || other.author == author)&&(identical(other.authorName, authorName) || other.authorName == authorName)&&(identical(other.authorAvatar, authorAvatar) || other.authorAvatar == authorAvatar)&&(identical(other.avatar, avatar) || other.avatar == avatar)&&(identical(other.name, name) || other.name == name)&&(identical(other.description, description) || other.description == description)&&(identical(other.elevationGain, elevationGain) || other.elevationGain == elevationGain)&&(identical(other.elevationLoss, elevationLoss) || other.elevationLoss == elevationLoss)&&(identical(other.distance, distance) || other.distance == distance)&&(identical(other.duration, duration) || other.duration == duration)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.public, public) || other.public == public)&&(identical(other.trails, trails) || other.trails == trails)&&const DeepCollectionEquality().equals(other.shares, shares)&&(identical(other.iri, iri) || other.iri == iri));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,collectionId,author,authorName,authorAvatar,avatar,name,description,elevationGain,elevationLoss,distance,duration,domain,public,trails,const DeepCollectionEquality().hash(shares),iri);

@override
String toString() {
  return 'ListSearchResult(id: $id, collectionId: $collectionId, author: $author, authorName: $authorName, authorAvatar: $authorAvatar, avatar: $avatar, name: $name, description: $description, elevationGain: $elevationGain, elevationLoss: $elevationLoss, distance: $distance, duration: $duration, domain: $domain, public: $public, trails: $trails, shares: $shares, iri: $iri)';
}


}

/// @nodoc
abstract mixin class $ListSearchResultCopyWith<$Res>  {
  factory $ListSearchResultCopyWith(ListSearchResult value, $Res Function(ListSearchResult) _then) = _$ListSearchResultCopyWithImpl;
@useResult
$Res call({
 String id, String collectionId, String author,@JsonKey(name: 'author_name') String authorName,@JsonKey(name: 'author_avatar') String authorAvatar, String? avatar, String name, String description,@JsonKey(name: 'elevation_gain') double elevationGain,@JsonKey(name: 'elevation_loss') double elevationLoss, double distance, double duration, String? domain, bool public, int trails, List<String>? shares, String? iri
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
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? collectionId = null,Object? author = null,Object? authorName = null,Object? authorAvatar = null,Object? avatar = freezed,Object? name = null,Object? description = null,Object? elevationGain = null,Object? elevationLoss = null,Object? distance = null,Object? duration = null,Object? domain = freezed,Object? public = null,Object? trails = null,Object? shares = freezed,Object? iri = freezed,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String collectionId,  String author, @JsonKey(name: 'author_name')  String authorName, @JsonKey(name: 'author_avatar')  String authorAvatar,  String? avatar,  String name,  String description, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double distance,  double duration,  String? domain,  bool public,  int trails,  List<String>? shares,  String? iri)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ListSearchResult() when $default != null:
return $default(_that.id,_that.collectionId,_that.author,_that.authorName,_that.authorAvatar,_that.avatar,_that.name,_that.description,_that.elevationGain,_that.elevationLoss,_that.distance,_that.duration,_that.domain,_that.public,_that.trails,_that.shares,_that.iri);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String collectionId,  String author, @JsonKey(name: 'author_name')  String authorName, @JsonKey(name: 'author_avatar')  String authorAvatar,  String? avatar,  String name,  String description, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double distance,  double duration,  String? domain,  bool public,  int trails,  List<String>? shares,  String? iri)  $default,) {final _that = this;
switch (_that) {
case _ListSearchResult():
return $default(_that.id,_that.collectionId,_that.author,_that.authorName,_that.authorAvatar,_that.avatar,_that.name,_that.description,_that.elevationGain,_that.elevationLoss,_that.distance,_that.duration,_that.domain,_that.public,_that.trails,_that.shares,_that.iri);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String collectionId,  String author, @JsonKey(name: 'author_name')  String authorName, @JsonKey(name: 'author_avatar')  String authorAvatar,  String? avatar,  String name,  String description, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double distance,  double duration,  String? domain,  bool public,  int trails,  List<String>? shares,  String? iri)?  $default,) {final _that = this;
switch (_that) {
case _ListSearchResult() when $default != null:
return $default(_that.id,_that.collectionId,_that.author,_that.authorName,_that.authorAvatar,_that.avatar,_that.name,_that.description,_that.elevationGain,_that.elevationLoss,_that.distance,_that.duration,_that.domain,_that.public,_that.trails,_that.shares,_that.iri);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _ListSearchResult extends ListSearchResult {
  const _ListSearchResult({required this.id, this.collectionId = 'lists', required this.author, @JsonKey(name: 'author_name') required this.authorName, @JsonKey(name: 'author_avatar') required this.authorAvatar, this.avatar, required this.name, required this.description, @JsonKey(name: 'elevation_gain') required this.elevationGain, @JsonKey(name: 'elevation_loss') required this.elevationLoss, required this.distance, required this.duration, this.domain, required this.public, required this.trails, final  List<String>? shares, this.iri}): _shares = shares,super._();
  factory _ListSearchResult.fromJson(Map<String, dynamic> json) => _$ListSearchResultFromJson(json);

@override final  String id;
@override@JsonKey() final  String collectionId;
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
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ListSearchResult&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.author, author) || other.author == author)&&(identical(other.authorName, authorName) || other.authorName == authorName)&&(identical(other.authorAvatar, authorAvatar) || other.authorAvatar == authorAvatar)&&(identical(other.avatar, avatar) || other.avatar == avatar)&&(identical(other.name, name) || other.name == name)&&(identical(other.description, description) || other.description == description)&&(identical(other.elevationGain, elevationGain) || other.elevationGain == elevationGain)&&(identical(other.elevationLoss, elevationLoss) || other.elevationLoss == elevationLoss)&&(identical(other.distance, distance) || other.distance == distance)&&(identical(other.duration, duration) || other.duration == duration)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.public, public) || other.public == public)&&(identical(other.trails, trails) || other.trails == trails)&&const DeepCollectionEquality().equals(other._shares, _shares)&&(identical(other.iri, iri) || other.iri == iri));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,collectionId,author,authorName,authorAvatar,avatar,name,description,elevationGain,elevationLoss,distance,duration,domain,public,trails,const DeepCollectionEquality().hash(_shares),iri);

@override
String toString() {
  return 'ListSearchResult(id: $id, collectionId: $collectionId, author: $author, authorName: $authorName, authorAvatar: $authorAvatar, avatar: $avatar, name: $name, description: $description, elevationGain: $elevationGain, elevationLoss: $elevationLoss, distance: $distance, duration: $duration, domain: $domain, public: $public, trails: $trails, shares: $shares, iri: $iri)';
}


}

/// @nodoc
abstract mixin class _$ListSearchResultCopyWith<$Res> implements $ListSearchResultCopyWith<$Res> {
  factory _$ListSearchResultCopyWith(_ListSearchResult value, $Res Function(_ListSearchResult) _then) = __$ListSearchResultCopyWithImpl;
@override @useResult
$Res call({
 String id, String collectionId, String author,@JsonKey(name: 'author_name') String authorName,@JsonKey(name: 'author_avatar') String authorAvatar, String? avatar, String name, String description,@JsonKey(name: 'elevation_gain') double elevationGain,@JsonKey(name: 'elevation_loss') double elevationLoss, double distance, double duration, String? domain, bool public, int trails, List<String>? shares, String? iri
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
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? collectionId = null,Object? author = null,Object? authorName = null,Object? authorAvatar = null,Object? avatar = freezed,Object? name = null,Object? description = null,Object? elevationGain = null,Object? elevationLoss = null,Object? distance = null,Object? duration = null,Object? domain = freezed,Object? public = null,Object? trails = null,Object? shares = freezed,Object? iri = freezed,}) {
  return _then(_ListSearchResult(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
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
mixin _$ActorSearchResult {

 String get id; String get username;@JsonKey(name: 'preferred_username') String get preferredUsername;@JsonKey(name: 'is_local') bool get isLocal; String get domain; String? get icon; String get iri;
/// Create a copy of ActorSearchResult
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ActorSearchResultCopyWith<ActorSearchResult> get copyWith => _$ActorSearchResultCopyWithImpl<ActorSearchResult>(this as ActorSearchResult, _$identity);

  /// Serializes this ActorSearchResult to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ActorSearchResult&&(identical(other.id, id) || other.id == id)&&(identical(other.username, username) || other.username == username)&&(identical(other.preferredUsername, preferredUsername) || other.preferredUsername == preferredUsername)&&(identical(other.isLocal, isLocal) || other.isLocal == isLocal)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.icon, icon) || other.icon == icon)&&(identical(other.iri, iri) || other.iri == iri));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,username,preferredUsername,isLocal,domain,icon,iri);

@override
String toString() {
  return 'ActorSearchResult(id: $id, username: $username, preferredUsername: $preferredUsername, isLocal: $isLocal, domain: $domain, icon: $icon, iri: $iri)';
}


}

/// @nodoc
abstract mixin class $ActorSearchResultCopyWith<$Res>  {
  factory $ActorSearchResultCopyWith(ActorSearchResult value, $Res Function(ActorSearchResult) _then) = _$ActorSearchResultCopyWithImpl;
@useResult
$Res call({
 String id, String username,@JsonKey(name: 'preferred_username') String preferredUsername,@JsonKey(name: 'is_local') bool isLocal, String domain, String? icon, String iri
});




}
/// @nodoc
class _$ActorSearchResultCopyWithImpl<$Res>
    implements $ActorSearchResultCopyWith<$Res> {
  _$ActorSearchResultCopyWithImpl(this._self, this._then);

  final ActorSearchResult _self;
  final $Res Function(ActorSearchResult) _then;

/// Create a copy of ActorSearchResult
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? username = null,Object? preferredUsername = null,Object? isLocal = null,Object? domain = null,Object? icon = freezed,Object? iri = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,preferredUsername: null == preferredUsername ? _self.preferredUsername : preferredUsername // ignore: cast_nullable_to_non_nullable
as String,isLocal: null == isLocal ? _self.isLocal : isLocal // ignore: cast_nullable_to_non_nullable
as bool,domain: null == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String,icon: freezed == icon ? _self.icon : icon // ignore: cast_nullable_to_non_nullable
as String?,iri: null == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

}


/// Adds pattern-matching-related methods to [ActorSearchResult].
extension ActorSearchResultPatterns on ActorSearchResult {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ActorSearchResult value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ActorSearchResult() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ActorSearchResult value)  $default,){
final _that = this;
switch (_that) {
case _ActorSearchResult():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ActorSearchResult value)?  $default,){
final _that = this;
switch (_that) {
case _ActorSearchResult() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String username, @JsonKey(name: 'preferred_username')  String preferredUsername, @JsonKey(name: 'is_local')  bool isLocal,  String domain,  String? icon,  String iri)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ActorSearchResult() when $default != null:
return $default(_that.id,_that.username,_that.preferredUsername,_that.isLocal,_that.domain,_that.icon,_that.iri);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String username, @JsonKey(name: 'preferred_username')  String preferredUsername, @JsonKey(name: 'is_local')  bool isLocal,  String domain,  String? icon,  String iri)  $default,) {final _that = this;
switch (_that) {
case _ActorSearchResult():
return $default(_that.id,_that.username,_that.preferredUsername,_that.isLocal,_that.domain,_that.icon,_that.iri);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String username, @JsonKey(name: 'preferred_username')  String preferredUsername, @JsonKey(name: 'is_local')  bool isLocal,  String domain,  String? icon,  String iri)?  $default,) {final _that = this;
switch (_that) {
case _ActorSearchResult() when $default != null:
return $default(_that.id,_that.username,_that.preferredUsername,_that.isLocal,_that.domain,_that.icon,_that.iri);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _ActorSearchResult implements ActorSearchResult {
  const _ActorSearchResult({this.id = '', this.username = '', @JsonKey(name: 'preferred_username') this.preferredUsername = '', @JsonKey(name: 'is_local') this.isLocal = false, this.domain = '', this.icon, this.iri = ''});
  factory _ActorSearchResult.fromJson(Map<String, dynamic> json) => _$ActorSearchResultFromJson(json);

@override@JsonKey() final  String id;
@override@JsonKey() final  String username;
@override@JsonKey(name: 'preferred_username') final  String preferredUsername;
@override@JsonKey(name: 'is_local') final  bool isLocal;
@override@JsonKey() final  String domain;
@override final  String? icon;
@override@JsonKey() final  String iri;

/// Create a copy of ActorSearchResult
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ActorSearchResultCopyWith<_ActorSearchResult> get copyWith => __$ActorSearchResultCopyWithImpl<_ActorSearchResult>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$ActorSearchResultToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ActorSearchResult&&(identical(other.id, id) || other.id == id)&&(identical(other.username, username) || other.username == username)&&(identical(other.preferredUsername, preferredUsername) || other.preferredUsername == preferredUsername)&&(identical(other.isLocal, isLocal) || other.isLocal == isLocal)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.icon, icon) || other.icon == icon)&&(identical(other.iri, iri) || other.iri == iri));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,username,preferredUsername,isLocal,domain,icon,iri);

@override
String toString() {
  return 'ActorSearchResult(id: $id, username: $username, preferredUsername: $preferredUsername, isLocal: $isLocal, domain: $domain, icon: $icon, iri: $iri)';
}


}

/// @nodoc
abstract mixin class _$ActorSearchResultCopyWith<$Res> implements $ActorSearchResultCopyWith<$Res> {
  factory _$ActorSearchResultCopyWith(_ActorSearchResult value, $Res Function(_ActorSearchResult) _then) = __$ActorSearchResultCopyWithImpl;
@override @useResult
$Res call({
 String id, String username,@JsonKey(name: 'preferred_username') String preferredUsername,@JsonKey(name: 'is_local') bool isLocal, String domain, String? icon, String iri
});




}
/// @nodoc
class __$ActorSearchResultCopyWithImpl<$Res>
    implements _$ActorSearchResultCopyWith<$Res> {
  __$ActorSearchResultCopyWithImpl(this._self, this._then);

  final _ActorSearchResult _self;
  final $Res Function(_ActorSearchResult) _then;

/// Create a copy of ActorSearchResult
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? username = null,Object? preferredUsername = null,Object? isLocal = null,Object? domain = null,Object? icon = freezed,Object? iri = null,}) {
  return _then(_ActorSearchResult(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,preferredUsername: null == preferredUsername ? _self.preferredUsername : preferredUsername // ignore: cast_nullable_to_non_nullable
as String,isLocal: null == isLocal ? _self.isLocal : isLocal // ignore: cast_nullable_to_non_nullable
as bool,domain: null == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String,icon: freezed == icon ? _self.icon : icon // ignore: cast_nullable_to_non_nullable
as String?,iri: null == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String,
  ));
}


}

// dart format on
