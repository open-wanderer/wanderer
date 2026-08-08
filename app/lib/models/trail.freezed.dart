// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'trail.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$TrailExpand {

 List<Tag>? get tags; Category? get category;@JsonKey(name: 'waypoints_via_trail') List<Waypoint>? get waypointsViaTrail;@JsonKey(name: 'summit_logs_via_trail') List<SummitLog>? get summitLogsViaTrail; Actor? get author;@JsonKey(name: 'comments_via_trail') List<Comment>? get commentsViaTrail;@JsonKey(name: 'gpx_data') String? get gpxData;@JsonKey(includeFromJson: false, includeToJson: false) Gpx? get gpx;@JsonKey(name: 'trail_share_via_trail') List<TrailShare>? get trailShareViaTrail;@JsonKey(name: 'trail_like_via_trail') List<TrailLike>? get trailLikeViaTrail;
/// Create a copy of TrailExpand
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailExpandCopyWith<TrailExpand> get copyWith => _$TrailExpandCopyWithImpl<TrailExpand>(this as TrailExpand, _$identity);

  /// Serializes this TrailExpand to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailExpand&&const DeepCollectionEquality().equals(other.tags, tags)&&(identical(other.category, category) || other.category == category)&&const DeepCollectionEquality().equals(other.waypointsViaTrail, waypointsViaTrail)&&const DeepCollectionEquality().equals(other.summitLogsViaTrail, summitLogsViaTrail)&&(identical(other.author, author) || other.author == author)&&const DeepCollectionEquality().equals(other.commentsViaTrail, commentsViaTrail)&&(identical(other.gpxData, gpxData) || other.gpxData == gpxData)&&(identical(other.gpx, gpx) || other.gpx == gpx)&&const DeepCollectionEquality().equals(other.trailShareViaTrail, trailShareViaTrail)&&const DeepCollectionEquality().equals(other.trailLikeViaTrail, trailLikeViaTrail));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(tags),category,const DeepCollectionEquality().hash(waypointsViaTrail),const DeepCollectionEquality().hash(summitLogsViaTrail),author,const DeepCollectionEquality().hash(commentsViaTrail),gpxData,gpx,const DeepCollectionEquality().hash(trailShareViaTrail),const DeepCollectionEquality().hash(trailLikeViaTrail));

@override
String toString() {
  return 'TrailExpand(tags: $tags, category: $category, waypointsViaTrail: $waypointsViaTrail, summitLogsViaTrail: $summitLogsViaTrail, author: $author, commentsViaTrail: $commentsViaTrail, gpxData: $gpxData, gpx: $gpx, trailShareViaTrail: $trailShareViaTrail, trailLikeViaTrail: $trailLikeViaTrail)';
}


}

/// @nodoc
abstract mixin class $TrailExpandCopyWith<$Res>  {
  factory $TrailExpandCopyWith(TrailExpand value, $Res Function(TrailExpand) _then) = _$TrailExpandCopyWithImpl;
@useResult
$Res call({
 List<Tag>? tags, Category? category,@JsonKey(name: 'waypoints_via_trail') List<Waypoint>? waypointsViaTrail,@JsonKey(name: 'summit_logs_via_trail') List<SummitLog>? summitLogsViaTrail, Actor? author,@JsonKey(name: 'comments_via_trail') List<Comment>? commentsViaTrail,@JsonKey(name: 'gpx_data') String? gpxData,@JsonKey(includeFromJson: false, includeToJson: false) Gpx? gpx,@JsonKey(name: 'trail_share_via_trail') List<TrailShare>? trailShareViaTrail,@JsonKey(name: 'trail_like_via_trail') List<TrailLike>? trailLikeViaTrail
});


$CategoryCopyWith<$Res>? get category;$ActorCopyWith<$Res>? get author;

}
/// @nodoc
class _$TrailExpandCopyWithImpl<$Res>
    implements $TrailExpandCopyWith<$Res> {
  _$TrailExpandCopyWithImpl(this._self, this._then);

  final TrailExpand _self;
  final $Res Function(TrailExpand) _then;

/// Create a copy of TrailExpand
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? tags = freezed,Object? category = freezed,Object? waypointsViaTrail = freezed,Object? summitLogsViaTrail = freezed,Object? author = freezed,Object? commentsViaTrail = freezed,Object? gpxData = freezed,Object? gpx = freezed,Object? trailShareViaTrail = freezed,Object? trailLikeViaTrail = freezed,}) {
  return _then(_self.copyWith(
tags: freezed == tags ? _self.tags : tags // ignore: cast_nullable_to_non_nullable
as List<Tag>?,category: freezed == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as Category?,waypointsViaTrail: freezed == waypointsViaTrail ? _self.waypointsViaTrail : waypointsViaTrail // ignore: cast_nullable_to_non_nullable
as List<Waypoint>?,summitLogsViaTrail: freezed == summitLogsViaTrail ? _self.summitLogsViaTrail : summitLogsViaTrail // ignore: cast_nullable_to_non_nullable
as List<SummitLog>?,author: freezed == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as Actor?,commentsViaTrail: freezed == commentsViaTrail ? _self.commentsViaTrail : commentsViaTrail // ignore: cast_nullable_to_non_nullable
as List<Comment>?,gpxData: freezed == gpxData ? _self.gpxData : gpxData // ignore: cast_nullable_to_non_nullable
as String?,gpx: freezed == gpx ? _self.gpx : gpx // ignore: cast_nullable_to_non_nullable
as Gpx?,trailShareViaTrail: freezed == trailShareViaTrail ? _self.trailShareViaTrail : trailShareViaTrail // ignore: cast_nullable_to_non_nullable
as List<TrailShare>?,trailLikeViaTrail: freezed == trailLikeViaTrail ? _self.trailLikeViaTrail : trailLikeViaTrail // ignore: cast_nullable_to_non_nullable
as List<TrailLike>?,
  ));
}
/// Create a copy of TrailExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$CategoryCopyWith<$Res>? get category {
    if (_self.category == null) {
    return null;
  }

  return $CategoryCopyWith<$Res>(_self.category!, (value) {
    return _then(_self.copyWith(category: value));
  });
}/// Create a copy of TrailExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res>? get author {
    if (_self.author == null) {
    return null;
  }

  return $ActorCopyWith<$Res>(_self.author!, (value) {
    return _then(_self.copyWith(author: value));
  });
}
}


/// Adds pattern-matching-related methods to [TrailExpand].
extension TrailExpandPatterns on TrailExpand {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailExpand value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailExpand() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailExpand value)  $default,){
final _that = this;
switch (_that) {
case _TrailExpand():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailExpand value)?  $default,){
final _that = this;
switch (_that) {
case _TrailExpand() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<Tag>? tags,  Category? category, @JsonKey(name: 'waypoints_via_trail')  List<Waypoint>? waypointsViaTrail, @JsonKey(name: 'summit_logs_via_trail')  List<SummitLog>? summitLogsViaTrail,  Actor? author, @JsonKey(name: 'comments_via_trail')  List<Comment>? commentsViaTrail, @JsonKey(name: 'gpx_data')  String? gpxData, @JsonKey(includeFromJson: false, includeToJson: false)  Gpx? gpx, @JsonKey(name: 'trail_share_via_trail')  List<TrailShare>? trailShareViaTrail, @JsonKey(name: 'trail_like_via_trail')  List<TrailLike>? trailLikeViaTrail)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailExpand() when $default != null:
return $default(_that.tags,_that.category,_that.waypointsViaTrail,_that.summitLogsViaTrail,_that.author,_that.commentsViaTrail,_that.gpxData,_that.gpx,_that.trailShareViaTrail,_that.trailLikeViaTrail);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<Tag>? tags,  Category? category, @JsonKey(name: 'waypoints_via_trail')  List<Waypoint>? waypointsViaTrail, @JsonKey(name: 'summit_logs_via_trail')  List<SummitLog>? summitLogsViaTrail,  Actor? author, @JsonKey(name: 'comments_via_trail')  List<Comment>? commentsViaTrail, @JsonKey(name: 'gpx_data')  String? gpxData, @JsonKey(includeFromJson: false, includeToJson: false)  Gpx? gpx, @JsonKey(name: 'trail_share_via_trail')  List<TrailShare>? trailShareViaTrail, @JsonKey(name: 'trail_like_via_trail')  List<TrailLike>? trailLikeViaTrail)  $default,) {final _that = this;
switch (_that) {
case _TrailExpand():
return $default(_that.tags,_that.category,_that.waypointsViaTrail,_that.summitLogsViaTrail,_that.author,_that.commentsViaTrail,_that.gpxData,_that.gpx,_that.trailShareViaTrail,_that.trailLikeViaTrail);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<Tag>? tags,  Category? category, @JsonKey(name: 'waypoints_via_trail')  List<Waypoint>? waypointsViaTrail, @JsonKey(name: 'summit_logs_via_trail')  List<SummitLog>? summitLogsViaTrail,  Actor? author, @JsonKey(name: 'comments_via_trail')  List<Comment>? commentsViaTrail, @JsonKey(name: 'gpx_data')  String? gpxData, @JsonKey(includeFromJson: false, includeToJson: false)  Gpx? gpx, @JsonKey(name: 'trail_share_via_trail')  List<TrailShare>? trailShareViaTrail, @JsonKey(name: 'trail_like_via_trail')  List<TrailLike>? trailLikeViaTrail)?  $default,) {final _that = this;
switch (_that) {
case _TrailExpand() when $default != null:
return $default(_that.tags,_that.category,_that.waypointsViaTrail,_that.summitLogsViaTrail,_that.author,_that.commentsViaTrail,_that.gpxData,_that.gpx,_that.trailShareViaTrail,_that.trailLikeViaTrail);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _TrailExpand implements TrailExpand {
  const _TrailExpand({final  List<Tag>? tags, this.category, @JsonKey(name: 'waypoints_via_trail') final  List<Waypoint>? waypointsViaTrail, @JsonKey(name: 'summit_logs_via_trail') final  List<SummitLog>? summitLogsViaTrail, this.author, @JsonKey(name: 'comments_via_trail') final  List<Comment>? commentsViaTrail, @JsonKey(name: 'gpx_data') this.gpxData, @JsonKey(includeFromJson: false, includeToJson: false) this.gpx, @JsonKey(name: 'trail_share_via_trail') final  List<TrailShare>? trailShareViaTrail, @JsonKey(name: 'trail_like_via_trail') final  List<TrailLike>? trailLikeViaTrail}): _tags = tags,_waypointsViaTrail = waypointsViaTrail,_summitLogsViaTrail = summitLogsViaTrail,_commentsViaTrail = commentsViaTrail,_trailShareViaTrail = trailShareViaTrail,_trailLikeViaTrail = trailLikeViaTrail;
  factory _TrailExpand.fromJson(Map<String, dynamic> json) => _$TrailExpandFromJson(json);

 final  List<Tag>? _tags;
@override List<Tag>? get tags {
  final value = _tags;
  if (value == null) return null;
  if (_tags is EqualUnmodifiableListView) return _tags;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}

@override final  Category? category;
 final  List<Waypoint>? _waypointsViaTrail;
@override@JsonKey(name: 'waypoints_via_trail') List<Waypoint>? get waypointsViaTrail {
  final value = _waypointsViaTrail;
  if (value == null) return null;
  if (_waypointsViaTrail is EqualUnmodifiableListView) return _waypointsViaTrail;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}

 final  List<SummitLog>? _summitLogsViaTrail;
@override@JsonKey(name: 'summit_logs_via_trail') List<SummitLog>? get summitLogsViaTrail {
  final value = _summitLogsViaTrail;
  if (value == null) return null;
  if (_summitLogsViaTrail is EqualUnmodifiableListView) return _summitLogsViaTrail;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}

@override final  Actor? author;
 final  List<Comment>? _commentsViaTrail;
@override@JsonKey(name: 'comments_via_trail') List<Comment>? get commentsViaTrail {
  final value = _commentsViaTrail;
  if (value == null) return null;
  if (_commentsViaTrail is EqualUnmodifiableListView) return _commentsViaTrail;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}

@override@JsonKey(name: 'gpx_data') final  String? gpxData;
@override@JsonKey(includeFromJson: false, includeToJson: false) final  Gpx? gpx;
 final  List<TrailShare>? _trailShareViaTrail;
@override@JsonKey(name: 'trail_share_via_trail') List<TrailShare>? get trailShareViaTrail {
  final value = _trailShareViaTrail;
  if (value == null) return null;
  if (_trailShareViaTrail is EqualUnmodifiableListView) return _trailShareViaTrail;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}

 final  List<TrailLike>? _trailLikeViaTrail;
@override@JsonKey(name: 'trail_like_via_trail') List<TrailLike>? get trailLikeViaTrail {
  final value = _trailLikeViaTrail;
  if (value == null) return null;
  if (_trailLikeViaTrail is EqualUnmodifiableListView) return _trailLikeViaTrail;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}


/// Create a copy of TrailExpand
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailExpandCopyWith<_TrailExpand> get copyWith => __$TrailExpandCopyWithImpl<_TrailExpand>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$TrailExpandToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailExpand&&const DeepCollectionEquality().equals(other._tags, _tags)&&(identical(other.category, category) || other.category == category)&&const DeepCollectionEquality().equals(other._waypointsViaTrail, _waypointsViaTrail)&&const DeepCollectionEquality().equals(other._summitLogsViaTrail, _summitLogsViaTrail)&&(identical(other.author, author) || other.author == author)&&const DeepCollectionEquality().equals(other._commentsViaTrail, _commentsViaTrail)&&(identical(other.gpxData, gpxData) || other.gpxData == gpxData)&&(identical(other.gpx, gpx) || other.gpx == gpx)&&const DeepCollectionEquality().equals(other._trailShareViaTrail, _trailShareViaTrail)&&const DeepCollectionEquality().equals(other._trailLikeViaTrail, _trailLikeViaTrail));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(_tags),category,const DeepCollectionEquality().hash(_waypointsViaTrail),const DeepCollectionEquality().hash(_summitLogsViaTrail),author,const DeepCollectionEquality().hash(_commentsViaTrail),gpxData,gpx,const DeepCollectionEquality().hash(_trailShareViaTrail),const DeepCollectionEquality().hash(_trailLikeViaTrail));

@override
String toString() {
  return 'TrailExpand(tags: $tags, category: $category, waypointsViaTrail: $waypointsViaTrail, summitLogsViaTrail: $summitLogsViaTrail, author: $author, commentsViaTrail: $commentsViaTrail, gpxData: $gpxData, gpx: $gpx, trailShareViaTrail: $trailShareViaTrail, trailLikeViaTrail: $trailLikeViaTrail)';
}


}

/// @nodoc
abstract mixin class _$TrailExpandCopyWith<$Res> implements $TrailExpandCopyWith<$Res> {
  factory _$TrailExpandCopyWith(_TrailExpand value, $Res Function(_TrailExpand) _then) = __$TrailExpandCopyWithImpl;
@override @useResult
$Res call({
 List<Tag>? tags, Category? category,@JsonKey(name: 'waypoints_via_trail') List<Waypoint>? waypointsViaTrail,@JsonKey(name: 'summit_logs_via_trail') List<SummitLog>? summitLogsViaTrail, Actor? author,@JsonKey(name: 'comments_via_trail') List<Comment>? commentsViaTrail,@JsonKey(name: 'gpx_data') String? gpxData,@JsonKey(includeFromJson: false, includeToJson: false) Gpx? gpx,@JsonKey(name: 'trail_share_via_trail') List<TrailShare>? trailShareViaTrail,@JsonKey(name: 'trail_like_via_trail') List<TrailLike>? trailLikeViaTrail
});


@override $CategoryCopyWith<$Res>? get category;@override $ActorCopyWith<$Res>? get author;

}
/// @nodoc
class __$TrailExpandCopyWithImpl<$Res>
    implements _$TrailExpandCopyWith<$Res> {
  __$TrailExpandCopyWithImpl(this._self, this._then);

  final _TrailExpand _self;
  final $Res Function(_TrailExpand) _then;

/// Create a copy of TrailExpand
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? tags = freezed,Object? category = freezed,Object? waypointsViaTrail = freezed,Object? summitLogsViaTrail = freezed,Object? author = freezed,Object? commentsViaTrail = freezed,Object? gpxData = freezed,Object? gpx = freezed,Object? trailShareViaTrail = freezed,Object? trailLikeViaTrail = freezed,}) {
  return _then(_TrailExpand(
tags: freezed == tags ? _self._tags : tags // ignore: cast_nullable_to_non_nullable
as List<Tag>?,category: freezed == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as Category?,waypointsViaTrail: freezed == waypointsViaTrail ? _self._waypointsViaTrail : waypointsViaTrail // ignore: cast_nullable_to_non_nullable
as List<Waypoint>?,summitLogsViaTrail: freezed == summitLogsViaTrail ? _self._summitLogsViaTrail : summitLogsViaTrail // ignore: cast_nullable_to_non_nullable
as List<SummitLog>?,author: freezed == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as Actor?,commentsViaTrail: freezed == commentsViaTrail ? _self._commentsViaTrail : commentsViaTrail // ignore: cast_nullable_to_non_nullable
as List<Comment>?,gpxData: freezed == gpxData ? _self.gpxData : gpxData // ignore: cast_nullable_to_non_nullable
as String?,gpx: freezed == gpx ? _self.gpx : gpx // ignore: cast_nullable_to_non_nullable
as Gpx?,trailShareViaTrail: freezed == trailShareViaTrail ? _self._trailShareViaTrail : trailShareViaTrail // ignore: cast_nullable_to_non_nullable
as List<TrailShare>?,trailLikeViaTrail: freezed == trailLikeViaTrail ? _self._trailLikeViaTrail : trailLikeViaTrail // ignore: cast_nullable_to_non_nullable
as List<TrailLike>?,
  ));
}

/// Create a copy of TrailExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$CategoryCopyWith<$Res>? get category {
    if (_self.category == null) {
    return null;
  }

  return $CategoryCopyWith<$Res>(_self.category!, (value) {
    return _then(_self.copyWith(category: value));
  });
}/// Create a copy of TrailExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res>? get author {
    if (_self.author == null) {
    return null;
  }

  return $ActorCopyWith<$Res>(_self.author!, (value) {
    return _then(_self.copyWith(author: value));
  });
}
}


/// @nodoc
mixin _$Trail {

 String get id; String get collectionId; String get name; String? get location;@_NullableDateTimeConverter() DateTime? get date; bool get public; bool get completed; double get distance;@JsonKey(name: 'elevation_gain') double get elevationGain;@JsonKey(name: 'elevation_loss') double get elevationLoss; double get duration;// Moving time in seconds for trails recorded in the Wanderer app. No
// @Default: absence is the meaningful "no moving time known" state
// -- `duration` always means GPX-derived elapsed time.
@JsonKey(name: 'moving_duration') double? get movingDuration; TrailDifficulty get difficulty; double? get lat; double? get lon;@JsonKey(name: 'max_lat') double get maxLat;@JsonKey(name: 'max_lon') double get maxLon;@JsonKey(name: 'min_lat') double get minLat;@JsonKey(name: 'min_lon') double get minLon; int get thumbnail; List<String> get photos; String? get gpx; DateTime get created; DateTime get updated; String? get category; String? get subcategory; List<String> get tags; String? get polyline; String? get domain; String? get iri;@JsonKey(name: 'like_count') int get likeCount; TrailExpand? get expand; String get description; String get author;/// This `Trail` instance was read from local storage — it is set only by
/// [TrailEntity.toModel], where it is hardcoded `true`.
///
/// **Provenance, not connectivity.** It answers "did this come off the
/// device?", never "is the device offline right now?". For the latter,
/// watch `onlineStatusProvider`. It was called `isOffline` until the two
/// got conflated in a shipped bug: `TrailMap(offline: trail.isOffline)`
/// selected the online basemap style for any trail that was not
/// downloaded, so editing or viewing one with no connection rendered a
/// blank map. Renamed to `isLocal` so the distinction is
/// legible at every call site; every `TrailMap` mount now derives
/// `offline:` from connectivity instead.
///
/// The only live consumers are thumbnail-path selection in
/// `trail_card.dart` and `trail_list_item.dart` (local file vs network
/// image). **Never gate a destructive action, a badge, or tab visibility
/// on this flag** — `TrailEntity.toModel()` hardcodes it `true` for
/// every cached row, so it cannot distinguish one
/// account's own local capture from another account's download.
/// Destructive-action *availability* derives from library membership and
/// authorship; destructive-action *scoping* derives from `owner`/account
/// for local capture state and from `savedByUserIds` membership for a
/// download. Unsynced and downloaded are not mutually exclusive.
 bool get isLocal; List<String> get localPhotos;/// Permanent local identity minted once at first local save. Device-local
/// only — never serialized. `Trail.toJson()` feeds `util/trail/form_data.dart`,
/// so this MUST carry `includeFromJson/includeToJson: false` or a
/// device-local value would leak into the multipart body sent to the
/// server.
@JsonKey(includeFromJson: false, includeToJson: false) String? get localId;/// This trail's upload/sync lifecycle. Device-local only — never
/// serialized, same rationale as [localId].
@JsonKey(includeFromJson: false, includeToJson: false) TrailSyncState get syncState;
/// Create a copy of Trail
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailCopyWith<Trail> get copyWith => _$TrailCopyWithImpl<Trail>(this as Trail, _$identity);

  /// Serializes this Trail to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Trail&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.name, name) || other.name == name)&&(identical(other.location, location) || other.location == location)&&(identical(other.date, date) || other.date == date)&&(identical(other.public, public) || other.public == public)&&(identical(other.completed, completed) || other.completed == completed)&&(identical(other.distance, distance) || other.distance == distance)&&(identical(other.elevationGain, elevationGain) || other.elevationGain == elevationGain)&&(identical(other.elevationLoss, elevationLoss) || other.elevationLoss == elevationLoss)&&(identical(other.duration, duration) || other.duration == duration)&&(identical(other.movingDuration, movingDuration) || other.movingDuration == movingDuration)&&(identical(other.difficulty, difficulty) || other.difficulty == difficulty)&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon)&&(identical(other.maxLat, maxLat) || other.maxLat == maxLat)&&(identical(other.maxLon, maxLon) || other.maxLon == maxLon)&&(identical(other.minLat, minLat) || other.minLat == minLat)&&(identical(other.minLon, minLon) || other.minLon == minLon)&&(identical(other.thumbnail, thumbnail) || other.thumbnail == thumbnail)&&const DeepCollectionEquality().equals(other.photos, photos)&&(identical(other.gpx, gpx) || other.gpx == gpx)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&(identical(other.category, category) || other.category == category)&&(identical(other.subcategory, subcategory) || other.subcategory == subcategory)&&const DeepCollectionEquality().equals(other.tags, tags)&&(identical(other.polyline, polyline) || other.polyline == polyline)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.likeCount, likeCount) || other.likeCount == likeCount)&&(identical(other.expand, expand) || other.expand == expand)&&(identical(other.description, description) || other.description == description)&&(identical(other.author, author) || other.author == author)&&(identical(other.isLocal, isLocal) || other.isLocal == isLocal)&&const DeepCollectionEquality().equals(other.localPhotos, localPhotos)&&(identical(other.localId, localId) || other.localId == localId)&&(identical(other.syncState, syncState) || other.syncState == syncState));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hashAll([runtimeType,id,collectionId,name,location,date,public,completed,distance,elevationGain,elevationLoss,duration,movingDuration,difficulty,lat,lon,maxLat,maxLon,minLat,minLon,thumbnail,const DeepCollectionEquality().hash(photos),gpx,created,updated,category,subcategory,const DeepCollectionEquality().hash(tags),polyline,domain,iri,likeCount,expand,description,author,isLocal,const DeepCollectionEquality().hash(localPhotos),localId,syncState]);

@override
String toString() {
  return 'Trail(id: $id, collectionId: $collectionId, name: $name, location: $location, date: $date, public: $public, completed: $completed, distance: $distance, elevationGain: $elevationGain, elevationLoss: $elevationLoss, duration: $duration, movingDuration: $movingDuration, difficulty: $difficulty, lat: $lat, lon: $lon, maxLat: $maxLat, maxLon: $maxLon, minLat: $minLat, minLon: $minLon, thumbnail: $thumbnail, photos: $photos, gpx: $gpx, created: $created, updated: $updated, category: $category, subcategory: $subcategory, tags: $tags, polyline: $polyline, domain: $domain, iri: $iri, likeCount: $likeCount, expand: $expand, description: $description, author: $author, isLocal: $isLocal, localPhotos: $localPhotos, localId: $localId, syncState: $syncState)';
}


}

/// @nodoc
abstract mixin class $TrailCopyWith<$Res>  {
  factory $TrailCopyWith(Trail value, $Res Function(Trail) _then) = _$TrailCopyWithImpl;
@useResult
$Res call({
 String id, String collectionId, String name, String? location,@_NullableDateTimeConverter() DateTime? date, bool public, bool completed, double distance,@JsonKey(name: 'elevation_gain') double elevationGain,@JsonKey(name: 'elevation_loss') double elevationLoss, double duration,@JsonKey(name: 'moving_duration') double? movingDuration, TrailDifficulty difficulty, double? lat, double? lon,@JsonKey(name: 'max_lat') double maxLat,@JsonKey(name: 'max_lon') double maxLon,@JsonKey(name: 'min_lat') double minLat,@JsonKey(name: 'min_lon') double minLon, int thumbnail, List<String> photos, String? gpx, DateTime created, DateTime updated, String? category, String? subcategory, List<String> tags, String? polyline, String? domain, String? iri,@JsonKey(name: 'like_count') int likeCount, TrailExpand? expand, String description, String author, bool isLocal, List<String> localPhotos,@JsonKey(includeFromJson: false, includeToJson: false) String? localId,@JsonKey(includeFromJson: false, includeToJson: false) TrailSyncState syncState
});


$TrailExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class _$TrailCopyWithImpl<$Res>
    implements $TrailCopyWith<$Res> {
  _$TrailCopyWithImpl(this._self, this._then);

  final Trail _self;
  final $Res Function(Trail) _then;

/// Create a copy of Trail
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? collectionId = null,Object? name = null,Object? location = freezed,Object? date = freezed,Object? public = null,Object? completed = null,Object? distance = null,Object? elevationGain = null,Object? elevationLoss = null,Object? duration = null,Object? movingDuration = freezed,Object? difficulty = null,Object? lat = freezed,Object? lon = freezed,Object? maxLat = null,Object? maxLon = null,Object? minLat = null,Object? minLon = null,Object? thumbnail = null,Object? photos = null,Object? gpx = freezed,Object? created = null,Object? updated = null,Object? category = freezed,Object? subcategory = freezed,Object? tags = null,Object? polyline = freezed,Object? domain = freezed,Object? iri = freezed,Object? likeCount = null,Object? expand = freezed,Object? description = null,Object? author = null,Object? isLocal = null,Object? localPhotos = null,Object? localId = freezed,Object? syncState = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,location: freezed == location ? _self.location : location // ignore: cast_nullable_to_non_nullable
as String?,date: freezed == date ? _self.date : date // ignore: cast_nullable_to_non_nullable
as DateTime?,public: null == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool,completed: null == completed ? _self.completed : completed // ignore: cast_nullable_to_non_nullable
as bool,distance: null == distance ? _self.distance : distance // ignore: cast_nullable_to_non_nullable
as double,elevationGain: null == elevationGain ? _self.elevationGain : elevationGain // ignore: cast_nullable_to_non_nullable
as double,elevationLoss: null == elevationLoss ? _self.elevationLoss : elevationLoss // ignore: cast_nullable_to_non_nullable
as double,duration: null == duration ? _self.duration : duration // ignore: cast_nullable_to_non_nullable
as double,movingDuration: freezed == movingDuration ? _self.movingDuration : movingDuration // ignore: cast_nullable_to_non_nullable
as double?,difficulty: null == difficulty ? _self.difficulty : difficulty // ignore: cast_nullable_to_non_nullable
as TrailDifficulty,lat: freezed == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double?,lon: freezed == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double?,maxLat: null == maxLat ? _self.maxLat : maxLat // ignore: cast_nullable_to_non_nullable
as double,maxLon: null == maxLon ? _self.maxLon : maxLon // ignore: cast_nullable_to_non_nullable
as double,minLat: null == minLat ? _self.minLat : minLat // ignore: cast_nullable_to_non_nullable
as double,minLon: null == minLon ? _self.minLon : minLon // ignore: cast_nullable_to_non_nullable
as double,thumbnail: null == thumbnail ? _self.thumbnail : thumbnail // ignore: cast_nullable_to_non_nullable
as int,photos: null == photos ? _self.photos : photos // ignore: cast_nullable_to_non_nullable
as List<String>,gpx: freezed == gpx ? _self.gpx : gpx // ignore: cast_nullable_to_non_nullable
as String?,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as DateTime,updated: null == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as DateTime,category: freezed == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as String?,subcategory: freezed == subcategory ? _self.subcategory : subcategory // ignore: cast_nullable_to_non_nullable
as String?,tags: null == tags ? _self.tags : tags // ignore: cast_nullable_to_non_nullable
as List<String>,polyline: freezed == polyline ? _self.polyline : polyline // ignore: cast_nullable_to_non_nullable
as String?,domain: freezed == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String?,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,likeCount: null == likeCount ? _self.likeCount : likeCount // ignore: cast_nullable_to_non_nullable
as int,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as TrailExpand?,description: null == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,isLocal: null == isLocal ? _self.isLocal : isLocal // ignore: cast_nullable_to_non_nullable
as bool,localPhotos: null == localPhotos ? _self.localPhotos : localPhotos // ignore: cast_nullable_to_non_nullable
as List<String>,localId: freezed == localId ? _self.localId : localId // ignore: cast_nullable_to_non_nullable
as String?,syncState: null == syncState ? _self.syncState : syncState // ignore: cast_nullable_to_non_nullable
as TrailSyncState,
  ));
}
/// Create a copy of Trail
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $TrailExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}


/// Adds pattern-matching-related methods to [Trail].
extension TrailPatterns on Trail {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Trail value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Trail() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Trail value)  $default,){
final _that = this;
switch (_that) {
case _Trail():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Trail value)?  $default,){
final _that = this;
switch (_that) {
case _Trail() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String collectionId,  String name,  String? location, @_NullableDateTimeConverter()  DateTime? date,  bool public,  bool completed,  double distance, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double duration, @JsonKey(name: 'moving_duration')  double? movingDuration,  TrailDifficulty difficulty,  double? lat,  double? lon, @JsonKey(name: 'max_lat')  double maxLat, @JsonKey(name: 'max_lon')  double maxLon, @JsonKey(name: 'min_lat')  double minLat, @JsonKey(name: 'min_lon')  double minLon,  int thumbnail,  List<String> photos,  String? gpx,  DateTime created,  DateTime updated,  String? category,  String? subcategory,  List<String> tags,  String? polyline,  String? domain,  String? iri, @JsonKey(name: 'like_count')  int likeCount,  TrailExpand? expand,  String description,  String author,  bool isLocal,  List<String> localPhotos, @JsonKey(includeFromJson: false, includeToJson: false)  String? localId, @JsonKey(includeFromJson: false, includeToJson: false)  TrailSyncState syncState)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Trail() when $default != null:
return $default(_that.id,_that.collectionId,_that.name,_that.location,_that.date,_that.public,_that.completed,_that.distance,_that.elevationGain,_that.elevationLoss,_that.duration,_that.movingDuration,_that.difficulty,_that.lat,_that.lon,_that.maxLat,_that.maxLon,_that.minLat,_that.minLon,_that.thumbnail,_that.photos,_that.gpx,_that.created,_that.updated,_that.category,_that.subcategory,_that.tags,_that.polyline,_that.domain,_that.iri,_that.likeCount,_that.expand,_that.description,_that.author,_that.isLocal,_that.localPhotos,_that.localId,_that.syncState);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String collectionId,  String name,  String? location, @_NullableDateTimeConverter()  DateTime? date,  bool public,  bool completed,  double distance, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double duration, @JsonKey(name: 'moving_duration')  double? movingDuration,  TrailDifficulty difficulty,  double? lat,  double? lon, @JsonKey(name: 'max_lat')  double maxLat, @JsonKey(name: 'max_lon')  double maxLon, @JsonKey(name: 'min_lat')  double minLat, @JsonKey(name: 'min_lon')  double minLon,  int thumbnail,  List<String> photos,  String? gpx,  DateTime created,  DateTime updated,  String? category,  String? subcategory,  List<String> tags,  String? polyline,  String? domain,  String? iri, @JsonKey(name: 'like_count')  int likeCount,  TrailExpand? expand,  String description,  String author,  bool isLocal,  List<String> localPhotos, @JsonKey(includeFromJson: false, includeToJson: false)  String? localId, @JsonKey(includeFromJson: false, includeToJson: false)  TrailSyncState syncState)  $default,) {final _that = this;
switch (_that) {
case _Trail():
return $default(_that.id,_that.collectionId,_that.name,_that.location,_that.date,_that.public,_that.completed,_that.distance,_that.elevationGain,_that.elevationLoss,_that.duration,_that.movingDuration,_that.difficulty,_that.lat,_that.lon,_that.maxLat,_that.maxLon,_that.minLat,_that.minLon,_that.thumbnail,_that.photos,_that.gpx,_that.created,_that.updated,_that.category,_that.subcategory,_that.tags,_that.polyline,_that.domain,_that.iri,_that.likeCount,_that.expand,_that.description,_that.author,_that.isLocal,_that.localPhotos,_that.localId,_that.syncState);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String collectionId,  String name,  String? location, @_NullableDateTimeConverter()  DateTime? date,  bool public,  bool completed,  double distance, @JsonKey(name: 'elevation_gain')  double elevationGain, @JsonKey(name: 'elevation_loss')  double elevationLoss,  double duration, @JsonKey(name: 'moving_duration')  double? movingDuration,  TrailDifficulty difficulty,  double? lat,  double? lon, @JsonKey(name: 'max_lat')  double maxLat, @JsonKey(name: 'max_lon')  double maxLon, @JsonKey(name: 'min_lat')  double minLat, @JsonKey(name: 'min_lon')  double minLon,  int thumbnail,  List<String> photos,  String? gpx,  DateTime created,  DateTime updated,  String? category,  String? subcategory,  List<String> tags,  String? polyline,  String? domain,  String? iri, @JsonKey(name: 'like_count')  int likeCount,  TrailExpand? expand,  String description,  String author,  bool isLocal,  List<String> localPhotos, @JsonKey(includeFromJson: false, includeToJson: false)  String? localId, @JsonKey(includeFromJson: false, includeToJson: false)  TrailSyncState syncState)?  $default,) {final _that = this;
switch (_that) {
case _Trail() when $default != null:
return $default(_that.id,_that.collectionId,_that.name,_that.location,_that.date,_that.public,_that.completed,_that.distance,_that.elevationGain,_that.elevationLoss,_that.duration,_that.movingDuration,_that.difficulty,_that.lat,_that.lon,_that.maxLat,_that.maxLon,_that.minLat,_that.minLon,_that.thumbnail,_that.photos,_that.gpx,_that.created,_that.updated,_that.category,_that.subcategory,_that.tags,_that.polyline,_that.domain,_that.iri,_that.likeCount,_that.expand,_that.description,_that.author,_that.isLocal,_that.localPhotos,_that.localId,_that.syncState);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Trail extends Trail {
  const _Trail({required this.id, this.collectionId = 'trails', required this.name, this.location, @_NullableDateTimeConverter() this.date, this.public = false, this.completed = false, this.distance = 0, @JsonKey(name: 'elevation_gain') this.elevationGain = 0, @JsonKey(name: 'elevation_loss') this.elevationLoss = 0, this.duration = 0, @JsonKey(name: 'moving_duration') this.movingDuration, this.difficulty = TrailDifficulty.easy, this.lat, this.lon, @JsonKey(name: 'max_lat') this.maxLat = 0, @JsonKey(name: 'max_lon') this.maxLon = 0, @JsonKey(name: 'min_lat') this.minLat = 0, @JsonKey(name: 'min_lon') this.minLon = 0, this.thumbnail = 0, final  List<String> photos = const [], this.gpx, required this.created, required this.updated, this.category, this.subcategory, final  List<String> tags = const [], this.polyline, this.domain, this.iri, @JsonKey(name: 'like_count') this.likeCount = 0, this.expand, this.description = "", this.author = "000000000000000", this.isLocal = false, final  List<String> localPhotos = const [], @JsonKey(includeFromJson: false, includeToJson: false) this.localId, @JsonKey(includeFromJson: false, includeToJson: false) this.syncState = TrailSyncState.synced}): _photos = photos,_tags = tags,_localPhotos = localPhotos,super._();
  factory _Trail.fromJson(Map<String, dynamic> json) => _$TrailFromJson(json);

@override final  String id;
@override@JsonKey() final  String collectionId;
@override final  String name;
@override final  String? location;
@override@_NullableDateTimeConverter() final  DateTime? date;
@override@JsonKey() final  bool public;
@override@JsonKey() final  bool completed;
@override@JsonKey() final  double distance;
@override@JsonKey(name: 'elevation_gain') final  double elevationGain;
@override@JsonKey(name: 'elevation_loss') final  double elevationLoss;
@override@JsonKey() final  double duration;
// Moving time in seconds for trails recorded in the Wanderer app. No
// @Default: absence is the meaningful "no moving time known" state
// -- `duration` always means GPX-derived elapsed time.
@override@JsonKey(name: 'moving_duration') final  double? movingDuration;
@override@JsonKey() final  TrailDifficulty difficulty;
@override final  double? lat;
@override final  double? lon;
@override@JsonKey(name: 'max_lat') final  double maxLat;
@override@JsonKey(name: 'max_lon') final  double maxLon;
@override@JsonKey(name: 'min_lat') final  double minLat;
@override@JsonKey(name: 'min_lon') final  double minLon;
@override@JsonKey() final  int thumbnail;
 final  List<String> _photos;
@override@JsonKey() List<String> get photos {
  if (_photos is EqualUnmodifiableListView) return _photos;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_photos);
}

@override final  String? gpx;
@override final  DateTime created;
@override final  DateTime updated;
@override final  String? category;
@override final  String? subcategory;
 final  List<String> _tags;
@override@JsonKey() List<String> get tags {
  if (_tags is EqualUnmodifiableListView) return _tags;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_tags);
}

@override final  String? polyline;
@override final  String? domain;
@override final  String? iri;
@override@JsonKey(name: 'like_count') final  int likeCount;
@override final  TrailExpand? expand;
@override@JsonKey() final  String description;
@override@JsonKey() final  String author;
/// This `Trail` instance was read from local storage — it is set only by
/// [TrailEntity.toModel], where it is hardcoded `true`.
///
/// **Provenance, not connectivity.** It answers "did this come off the
/// device?", never "is the device offline right now?". For the latter,
/// watch `onlineStatusProvider`. It was called `isOffline` until the two
/// got conflated in a shipped bug: `TrailMap(offline: trail.isOffline)`
/// selected the online basemap style for any trail that was not
/// downloaded, so editing or viewing one with no connection rendered a
/// blank map. Renamed to `isLocal` so the distinction is
/// legible at every call site; every `TrailMap` mount now derives
/// `offline:` from connectivity instead.
///
/// The only live consumers are thumbnail-path selection in
/// `trail_card.dart` and `trail_list_item.dart` (local file vs network
/// image). **Never gate a destructive action, a badge, or tab visibility
/// on this flag** — `TrailEntity.toModel()` hardcodes it `true` for
/// every cached row, so it cannot distinguish one
/// account's own local capture from another account's download.
/// Destructive-action *availability* derives from library membership and
/// authorship; destructive-action *scoping* derives from `owner`/account
/// for local capture state and from `savedByUserIds` membership for a
/// download. Unsynced and downloaded are not mutually exclusive.
@override@JsonKey() final  bool isLocal;
 final  List<String> _localPhotos;
@override@JsonKey() List<String> get localPhotos {
  if (_localPhotos is EqualUnmodifiableListView) return _localPhotos;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_localPhotos);
}

/// Permanent local identity minted once at first local save. Device-local
/// only — never serialized. `Trail.toJson()` feeds `util/trail/form_data.dart`,
/// so this MUST carry `includeFromJson/includeToJson: false` or a
/// device-local value would leak into the multipart body sent to the
/// server.
@override@JsonKey(includeFromJson: false, includeToJson: false) final  String? localId;
/// This trail's upload/sync lifecycle. Device-local only — never
/// serialized, same rationale as [localId].
@override@JsonKey(includeFromJson: false, includeToJson: false) final  TrailSyncState syncState;

/// Create a copy of Trail
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailCopyWith<_Trail> get copyWith => __$TrailCopyWithImpl<_Trail>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$TrailToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _Trail&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.name, name) || other.name == name)&&(identical(other.location, location) || other.location == location)&&(identical(other.date, date) || other.date == date)&&(identical(other.public, public) || other.public == public)&&(identical(other.completed, completed) || other.completed == completed)&&(identical(other.distance, distance) || other.distance == distance)&&(identical(other.elevationGain, elevationGain) || other.elevationGain == elevationGain)&&(identical(other.elevationLoss, elevationLoss) || other.elevationLoss == elevationLoss)&&(identical(other.duration, duration) || other.duration == duration)&&(identical(other.movingDuration, movingDuration) || other.movingDuration == movingDuration)&&(identical(other.difficulty, difficulty) || other.difficulty == difficulty)&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon)&&(identical(other.maxLat, maxLat) || other.maxLat == maxLat)&&(identical(other.maxLon, maxLon) || other.maxLon == maxLon)&&(identical(other.minLat, minLat) || other.minLat == minLat)&&(identical(other.minLon, minLon) || other.minLon == minLon)&&(identical(other.thumbnail, thumbnail) || other.thumbnail == thumbnail)&&const DeepCollectionEquality().equals(other._photos, _photos)&&(identical(other.gpx, gpx) || other.gpx == gpx)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&(identical(other.category, category) || other.category == category)&&(identical(other.subcategory, subcategory) || other.subcategory == subcategory)&&const DeepCollectionEquality().equals(other._tags, _tags)&&(identical(other.polyline, polyline) || other.polyline == polyline)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.likeCount, likeCount) || other.likeCount == likeCount)&&(identical(other.expand, expand) || other.expand == expand)&&(identical(other.description, description) || other.description == description)&&(identical(other.author, author) || other.author == author)&&(identical(other.isLocal, isLocal) || other.isLocal == isLocal)&&const DeepCollectionEquality().equals(other._localPhotos, _localPhotos)&&(identical(other.localId, localId) || other.localId == localId)&&(identical(other.syncState, syncState) || other.syncState == syncState));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hashAll([runtimeType,id,collectionId,name,location,date,public,completed,distance,elevationGain,elevationLoss,duration,movingDuration,difficulty,lat,lon,maxLat,maxLon,minLat,minLon,thumbnail,const DeepCollectionEquality().hash(_photos),gpx,created,updated,category,subcategory,const DeepCollectionEquality().hash(_tags),polyline,domain,iri,likeCount,expand,description,author,isLocal,const DeepCollectionEquality().hash(_localPhotos),localId,syncState]);

@override
String toString() {
  return 'Trail(id: $id, collectionId: $collectionId, name: $name, location: $location, date: $date, public: $public, completed: $completed, distance: $distance, elevationGain: $elevationGain, elevationLoss: $elevationLoss, duration: $duration, movingDuration: $movingDuration, difficulty: $difficulty, lat: $lat, lon: $lon, maxLat: $maxLat, maxLon: $maxLon, minLat: $minLat, minLon: $minLon, thumbnail: $thumbnail, photos: $photos, gpx: $gpx, created: $created, updated: $updated, category: $category, subcategory: $subcategory, tags: $tags, polyline: $polyline, domain: $domain, iri: $iri, likeCount: $likeCount, expand: $expand, description: $description, author: $author, isLocal: $isLocal, localPhotos: $localPhotos, localId: $localId, syncState: $syncState)';
}


}

/// @nodoc
abstract mixin class _$TrailCopyWith<$Res> implements $TrailCopyWith<$Res> {
  factory _$TrailCopyWith(_Trail value, $Res Function(_Trail) _then) = __$TrailCopyWithImpl;
@override @useResult
$Res call({
 String id, String collectionId, String name, String? location,@_NullableDateTimeConverter() DateTime? date, bool public, bool completed, double distance,@JsonKey(name: 'elevation_gain') double elevationGain,@JsonKey(name: 'elevation_loss') double elevationLoss, double duration,@JsonKey(name: 'moving_duration') double? movingDuration, TrailDifficulty difficulty, double? lat, double? lon,@JsonKey(name: 'max_lat') double maxLat,@JsonKey(name: 'max_lon') double maxLon,@JsonKey(name: 'min_lat') double minLat,@JsonKey(name: 'min_lon') double minLon, int thumbnail, List<String> photos, String? gpx, DateTime created, DateTime updated, String? category, String? subcategory, List<String> tags, String? polyline, String? domain, String? iri,@JsonKey(name: 'like_count') int likeCount, TrailExpand? expand, String description, String author, bool isLocal, List<String> localPhotos,@JsonKey(includeFromJson: false, includeToJson: false) String? localId,@JsonKey(includeFromJson: false, includeToJson: false) TrailSyncState syncState
});


@override $TrailExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class __$TrailCopyWithImpl<$Res>
    implements _$TrailCopyWith<$Res> {
  __$TrailCopyWithImpl(this._self, this._then);

  final _Trail _self;
  final $Res Function(_Trail) _then;

/// Create a copy of Trail
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? collectionId = null,Object? name = null,Object? location = freezed,Object? date = freezed,Object? public = null,Object? completed = null,Object? distance = null,Object? elevationGain = null,Object? elevationLoss = null,Object? duration = null,Object? movingDuration = freezed,Object? difficulty = null,Object? lat = freezed,Object? lon = freezed,Object? maxLat = null,Object? maxLon = null,Object? minLat = null,Object? minLon = null,Object? thumbnail = null,Object? photos = null,Object? gpx = freezed,Object? created = null,Object? updated = null,Object? category = freezed,Object? subcategory = freezed,Object? tags = null,Object? polyline = freezed,Object? domain = freezed,Object? iri = freezed,Object? likeCount = null,Object? expand = freezed,Object? description = null,Object? author = null,Object? isLocal = null,Object? localPhotos = null,Object? localId = freezed,Object? syncState = null,}) {
  return _then(_Trail(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,location: freezed == location ? _self.location : location // ignore: cast_nullable_to_non_nullable
as String?,date: freezed == date ? _self.date : date // ignore: cast_nullable_to_non_nullable
as DateTime?,public: null == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool,completed: null == completed ? _self.completed : completed // ignore: cast_nullable_to_non_nullable
as bool,distance: null == distance ? _self.distance : distance // ignore: cast_nullable_to_non_nullable
as double,elevationGain: null == elevationGain ? _self.elevationGain : elevationGain // ignore: cast_nullable_to_non_nullable
as double,elevationLoss: null == elevationLoss ? _self.elevationLoss : elevationLoss // ignore: cast_nullable_to_non_nullable
as double,duration: null == duration ? _self.duration : duration // ignore: cast_nullable_to_non_nullable
as double,movingDuration: freezed == movingDuration ? _self.movingDuration : movingDuration // ignore: cast_nullable_to_non_nullable
as double?,difficulty: null == difficulty ? _self.difficulty : difficulty // ignore: cast_nullable_to_non_nullable
as TrailDifficulty,lat: freezed == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double?,lon: freezed == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double?,maxLat: null == maxLat ? _self.maxLat : maxLat // ignore: cast_nullable_to_non_nullable
as double,maxLon: null == maxLon ? _self.maxLon : maxLon // ignore: cast_nullable_to_non_nullable
as double,minLat: null == minLat ? _self.minLat : minLat // ignore: cast_nullable_to_non_nullable
as double,minLon: null == minLon ? _self.minLon : minLon // ignore: cast_nullable_to_non_nullable
as double,thumbnail: null == thumbnail ? _self.thumbnail : thumbnail // ignore: cast_nullable_to_non_nullable
as int,photos: null == photos ? _self._photos : photos // ignore: cast_nullable_to_non_nullable
as List<String>,gpx: freezed == gpx ? _self.gpx : gpx // ignore: cast_nullable_to_non_nullable
as String?,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as DateTime,updated: null == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as DateTime,category: freezed == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as String?,subcategory: freezed == subcategory ? _self.subcategory : subcategory // ignore: cast_nullable_to_non_nullable
as String?,tags: null == tags ? _self._tags : tags // ignore: cast_nullable_to_non_nullable
as List<String>,polyline: freezed == polyline ? _self.polyline : polyline // ignore: cast_nullable_to_non_nullable
as String?,domain: freezed == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String?,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,likeCount: null == likeCount ? _self.likeCount : likeCount // ignore: cast_nullable_to_non_nullable
as int,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as TrailExpand?,description: null == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,isLocal: null == isLocal ? _self.isLocal : isLocal // ignore: cast_nullable_to_non_nullable
as bool,localPhotos: null == localPhotos ? _self._localPhotos : localPhotos // ignore: cast_nullable_to_non_nullable
as List<String>,localId: freezed == localId ? _self.localId : localId // ignore: cast_nullable_to_non_nullable
as String?,syncState: null == syncState ? _self.syncState : syncState // ignore: cast_nullable_to_non_nullable
as TrailSyncState,
  ));
}

/// Create a copy of Trail
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $TrailExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}

/// @nodoc
mixin _$TrailNear {

 double? get lat; double? get lon; double get radius;
/// Create a copy of TrailNear
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailNearCopyWith<TrailNear> get copyWith => _$TrailNearCopyWithImpl<TrailNear>(this as TrailNear, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailNear&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon)&&(identical(other.radius, radius) || other.radius == radius));
}


@override
int get hashCode => Object.hash(runtimeType,lat,lon,radius);

@override
String toString() {
  return 'TrailNear(lat: $lat, lon: $lon, radius: $radius)';
}


}

/// @nodoc
abstract mixin class $TrailNearCopyWith<$Res>  {
  factory $TrailNearCopyWith(TrailNear value, $Res Function(TrailNear) _then) = _$TrailNearCopyWithImpl;
@useResult
$Res call({
 double? lat, double? lon, double radius
});




}
/// @nodoc
class _$TrailNearCopyWithImpl<$Res>
    implements $TrailNearCopyWith<$Res> {
  _$TrailNearCopyWithImpl(this._self, this._then);

  final TrailNear _self;
  final $Res Function(TrailNear) _then;

/// Create a copy of TrailNear
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? lat = freezed,Object? lon = freezed,Object? radius = null,}) {
  return _then(_self.copyWith(
lat: freezed == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double?,lon: freezed == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double?,radius: null == radius ? _self.radius : radius // ignore: cast_nullable_to_non_nullable
as double,
  ));
}

}


/// Adds pattern-matching-related methods to [TrailNear].
extension TrailNearPatterns on TrailNear {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailNear value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailNear() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailNear value)  $default,){
final _that = this;
switch (_that) {
case _TrailNear():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailNear value)?  $default,){
final _that = this;
switch (_that) {
case _TrailNear() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( double? lat,  double? lon,  double radius)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailNear() when $default != null:
return $default(_that.lat,_that.lon,_that.radius);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( double? lat,  double? lon,  double radius)  $default,) {final _that = this;
switch (_that) {
case _TrailNear():
return $default(_that.lat,_that.lon,_that.radius);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( double? lat,  double? lon,  double radius)?  $default,) {final _that = this;
switch (_that) {
case _TrailNear() when $default != null:
return $default(_that.lat,_that.lon,_that.radius);case _:
  return null;

}
}

}

/// @nodoc


class _TrailNear implements TrailNear {
  const _TrailNear({this.lat, this.lon, required this.radius});
  

@override final  double? lat;
@override final  double? lon;
@override final  double radius;

/// Create a copy of TrailNear
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailNearCopyWith<_TrailNear> get copyWith => __$TrailNearCopyWithImpl<_TrailNear>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailNear&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon)&&(identical(other.radius, radius) || other.radius == radius));
}


@override
int get hashCode => Object.hash(runtimeType,lat,lon,radius);

@override
String toString() {
  return 'TrailNear(lat: $lat, lon: $lon, radius: $radius)';
}


}

/// @nodoc
abstract mixin class _$TrailNearCopyWith<$Res> implements $TrailNearCopyWith<$Res> {
  factory _$TrailNearCopyWith(_TrailNear value, $Res Function(_TrailNear) _then) = __$TrailNearCopyWithImpl;
@override @useResult
$Res call({
 double? lat, double? lon, double radius
});




}
/// @nodoc
class __$TrailNearCopyWithImpl<$Res>
    implements _$TrailNearCopyWith<$Res> {
  __$TrailNearCopyWithImpl(this._self, this._then);

  final _TrailNear _self;
  final $Res Function(_TrailNear) _then;

/// Create a copy of TrailNear
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? lat = freezed,Object? lon = freezed,Object? radius = null,}) {
  return _then(_TrailNear(
lat: freezed == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double?,lon: freezed == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double?,radius: null == radius ? _self.radius : radius // ignore: cast_nullable_to_non_nullable
as double,
  ));
}


}

/// @nodoc
mixin _$TrailFilter {

 String get q; List<Category> get category; List<Subcategory> get subcategory; List<Tag> get tags; List<int> get difficulty;// 0, 1, 2
 ActorSearchResult? get author; bool? get public; bool? get shared; bool? get private; TrailNear get near; double get distanceMin; double get distanceMax; double get distanceLimit; double get elevationGainMin; double get elevationGainMax; double get elevationGainLimit; double get elevationLossMin; double get elevationLossMax; double get elevationLossLimit; DateTime? get startDate; DateTime? get endDate; bool? get completed; bool? get liked;// Emitted server-side as an `id IN [...]` whitelist, but only when the
// caller supplies the ids -- see `toFilterText`'s [offlineTrailIds].
// `applyTrailFilter` carries a second, non-redundant half for unsynced
// local captures, which have no server id to whitelist.
 bool get offlineOnly; TrailFilterSort get sort;// "name" | "distance" | "elevation_gain" | "created"
 SortOrder get sortOrder;
/// Create a copy of TrailFilter
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailFilterCopyWith<TrailFilter> get copyWith => _$TrailFilterCopyWithImpl<TrailFilter>(this as TrailFilter, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailFilter&&(identical(other.q, q) || other.q == q)&&const DeepCollectionEquality().equals(other.category, category)&&const DeepCollectionEquality().equals(other.subcategory, subcategory)&&const DeepCollectionEquality().equals(other.tags, tags)&&const DeepCollectionEquality().equals(other.difficulty, difficulty)&&(identical(other.author, author) || other.author == author)&&(identical(other.public, public) || other.public == public)&&(identical(other.shared, shared) || other.shared == shared)&&(identical(other.private, private) || other.private == private)&&(identical(other.near, near) || other.near == near)&&(identical(other.distanceMin, distanceMin) || other.distanceMin == distanceMin)&&(identical(other.distanceMax, distanceMax) || other.distanceMax == distanceMax)&&(identical(other.distanceLimit, distanceLimit) || other.distanceLimit == distanceLimit)&&(identical(other.elevationGainMin, elevationGainMin) || other.elevationGainMin == elevationGainMin)&&(identical(other.elevationGainMax, elevationGainMax) || other.elevationGainMax == elevationGainMax)&&(identical(other.elevationGainLimit, elevationGainLimit) || other.elevationGainLimit == elevationGainLimit)&&(identical(other.elevationLossMin, elevationLossMin) || other.elevationLossMin == elevationLossMin)&&(identical(other.elevationLossMax, elevationLossMax) || other.elevationLossMax == elevationLossMax)&&(identical(other.elevationLossLimit, elevationLossLimit) || other.elevationLossLimit == elevationLossLimit)&&(identical(other.startDate, startDate) || other.startDate == startDate)&&(identical(other.endDate, endDate) || other.endDate == endDate)&&(identical(other.completed, completed) || other.completed == completed)&&(identical(other.liked, liked) || other.liked == liked)&&(identical(other.offlineOnly, offlineOnly) || other.offlineOnly == offlineOnly)&&(identical(other.sort, sort) || other.sort == sort)&&(identical(other.sortOrder, sortOrder) || other.sortOrder == sortOrder));
}


@override
int get hashCode => Object.hashAll([runtimeType,q,const DeepCollectionEquality().hash(category),const DeepCollectionEquality().hash(subcategory),const DeepCollectionEquality().hash(tags),const DeepCollectionEquality().hash(difficulty),author,public,shared,private,near,distanceMin,distanceMax,distanceLimit,elevationGainMin,elevationGainMax,elevationGainLimit,elevationLossMin,elevationLossMax,elevationLossLimit,startDate,endDate,completed,liked,offlineOnly,sort,sortOrder]);

@override
String toString() {
  return 'TrailFilter(q: $q, category: $category, subcategory: $subcategory, tags: $tags, difficulty: $difficulty, author: $author, public: $public, shared: $shared, private: $private, near: $near, distanceMin: $distanceMin, distanceMax: $distanceMax, distanceLimit: $distanceLimit, elevationGainMin: $elevationGainMin, elevationGainMax: $elevationGainMax, elevationGainLimit: $elevationGainLimit, elevationLossMin: $elevationLossMin, elevationLossMax: $elevationLossMax, elevationLossLimit: $elevationLossLimit, startDate: $startDate, endDate: $endDate, completed: $completed, liked: $liked, offlineOnly: $offlineOnly, sort: $sort, sortOrder: $sortOrder)';
}


}

/// @nodoc
abstract mixin class $TrailFilterCopyWith<$Res>  {
  factory $TrailFilterCopyWith(TrailFilter value, $Res Function(TrailFilter) _then) = _$TrailFilterCopyWithImpl;
@useResult
$Res call({
 String q, List<Category> category, List<Subcategory> subcategory, List<Tag> tags, List<int> difficulty, ActorSearchResult? author, bool? public, bool? shared, bool? private, TrailNear near, double distanceMin, double distanceMax, double distanceLimit, double elevationGainMin, double elevationGainMax, double elevationGainLimit, double elevationLossMin, double elevationLossMax, double elevationLossLimit, DateTime? startDate, DateTime? endDate, bool? completed, bool? liked, bool offlineOnly, TrailFilterSort sort, SortOrder sortOrder
});


$ActorSearchResultCopyWith<$Res>? get author;$TrailNearCopyWith<$Res> get near;

}
/// @nodoc
class _$TrailFilterCopyWithImpl<$Res>
    implements $TrailFilterCopyWith<$Res> {
  _$TrailFilterCopyWithImpl(this._self, this._then);

  final TrailFilter _self;
  final $Res Function(TrailFilter) _then;

/// Create a copy of TrailFilter
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? q = null,Object? category = null,Object? subcategory = null,Object? tags = null,Object? difficulty = null,Object? author = freezed,Object? public = freezed,Object? shared = freezed,Object? private = freezed,Object? near = null,Object? distanceMin = null,Object? distanceMax = null,Object? distanceLimit = null,Object? elevationGainMin = null,Object? elevationGainMax = null,Object? elevationGainLimit = null,Object? elevationLossMin = null,Object? elevationLossMax = null,Object? elevationLossLimit = null,Object? startDate = freezed,Object? endDate = freezed,Object? completed = freezed,Object? liked = freezed,Object? offlineOnly = null,Object? sort = null,Object? sortOrder = null,}) {
  return _then(_self.copyWith(
q: null == q ? _self.q : q // ignore: cast_nullable_to_non_nullable
as String,category: null == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as List<Category>,subcategory: null == subcategory ? _self.subcategory : subcategory // ignore: cast_nullable_to_non_nullable
as List<Subcategory>,tags: null == tags ? _self.tags : tags // ignore: cast_nullable_to_non_nullable
as List<Tag>,difficulty: null == difficulty ? _self.difficulty : difficulty // ignore: cast_nullable_to_non_nullable
as List<int>,author: freezed == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as ActorSearchResult?,public: freezed == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool?,shared: freezed == shared ? _self.shared : shared // ignore: cast_nullable_to_non_nullable
as bool?,private: freezed == private ? _self.private : private // ignore: cast_nullable_to_non_nullable
as bool?,near: null == near ? _self.near : near // ignore: cast_nullable_to_non_nullable
as TrailNear,distanceMin: null == distanceMin ? _self.distanceMin : distanceMin // ignore: cast_nullable_to_non_nullable
as double,distanceMax: null == distanceMax ? _self.distanceMax : distanceMax // ignore: cast_nullable_to_non_nullable
as double,distanceLimit: null == distanceLimit ? _self.distanceLimit : distanceLimit // ignore: cast_nullable_to_non_nullable
as double,elevationGainMin: null == elevationGainMin ? _self.elevationGainMin : elevationGainMin // ignore: cast_nullable_to_non_nullable
as double,elevationGainMax: null == elevationGainMax ? _self.elevationGainMax : elevationGainMax // ignore: cast_nullable_to_non_nullable
as double,elevationGainLimit: null == elevationGainLimit ? _self.elevationGainLimit : elevationGainLimit // ignore: cast_nullable_to_non_nullable
as double,elevationLossMin: null == elevationLossMin ? _self.elevationLossMin : elevationLossMin // ignore: cast_nullable_to_non_nullable
as double,elevationLossMax: null == elevationLossMax ? _self.elevationLossMax : elevationLossMax // ignore: cast_nullable_to_non_nullable
as double,elevationLossLimit: null == elevationLossLimit ? _self.elevationLossLimit : elevationLossLimit // ignore: cast_nullable_to_non_nullable
as double,startDate: freezed == startDate ? _self.startDate : startDate // ignore: cast_nullable_to_non_nullable
as DateTime?,endDate: freezed == endDate ? _self.endDate : endDate // ignore: cast_nullable_to_non_nullable
as DateTime?,completed: freezed == completed ? _self.completed : completed // ignore: cast_nullable_to_non_nullable
as bool?,liked: freezed == liked ? _self.liked : liked // ignore: cast_nullable_to_non_nullable
as bool?,offlineOnly: null == offlineOnly ? _self.offlineOnly : offlineOnly // ignore: cast_nullable_to_non_nullable
as bool,sort: null == sort ? _self.sort : sort // ignore: cast_nullable_to_non_nullable
as TrailFilterSort,sortOrder: null == sortOrder ? _self.sortOrder : sortOrder // ignore: cast_nullable_to_non_nullable
as SortOrder,
  ));
}
/// Create a copy of TrailFilter
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorSearchResultCopyWith<$Res>? get author {
    if (_self.author == null) {
    return null;
  }

  return $ActorSearchResultCopyWith<$Res>(_self.author!, (value) {
    return _then(_self.copyWith(author: value));
  });
}/// Create a copy of TrailFilter
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailNearCopyWith<$Res> get near {
  
  return $TrailNearCopyWith<$Res>(_self.near, (value) {
    return _then(_self.copyWith(near: value));
  });
}
}


/// Adds pattern-matching-related methods to [TrailFilter].
extension TrailFilterPatterns on TrailFilter {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailFilter value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailFilter() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailFilter value)  $default,){
final _that = this;
switch (_that) {
case _TrailFilter():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailFilter value)?  $default,){
final _that = this;
switch (_that) {
case _TrailFilter() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String q,  List<Category> category,  List<Subcategory> subcategory,  List<Tag> tags,  List<int> difficulty,  ActorSearchResult? author,  bool? public,  bool? shared,  bool? private,  TrailNear near,  double distanceMin,  double distanceMax,  double distanceLimit,  double elevationGainMin,  double elevationGainMax,  double elevationGainLimit,  double elevationLossMin,  double elevationLossMax,  double elevationLossLimit,  DateTime? startDate,  DateTime? endDate,  bool? completed,  bool? liked,  bool offlineOnly,  TrailFilterSort sort,  SortOrder sortOrder)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailFilter() when $default != null:
return $default(_that.q,_that.category,_that.subcategory,_that.tags,_that.difficulty,_that.author,_that.public,_that.shared,_that.private,_that.near,_that.distanceMin,_that.distanceMax,_that.distanceLimit,_that.elevationGainMin,_that.elevationGainMax,_that.elevationGainLimit,_that.elevationLossMin,_that.elevationLossMax,_that.elevationLossLimit,_that.startDate,_that.endDate,_that.completed,_that.liked,_that.offlineOnly,_that.sort,_that.sortOrder);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String q,  List<Category> category,  List<Subcategory> subcategory,  List<Tag> tags,  List<int> difficulty,  ActorSearchResult? author,  bool? public,  bool? shared,  bool? private,  TrailNear near,  double distanceMin,  double distanceMax,  double distanceLimit,  double elevationGainMin,  double elevationGainMax,  double elevationGainLimit,  double elevationLossMin,  double elevationLossMax,  double elevationLossLimit,  DateTime? startDate,  DateTime? endDate,  bool? completed,  bool? liked,  bool offlineOnly,  TrailFilterSort sort,  SortOrder sortOrder)  $default,) {final _that = this;
switch (_that) {
case _TrailFilter():
return $default(_that.q,_that.category,_that.subcategory,_that.tags,_that.difficulty,_that.author,_that.public,_that.shared,_that.private,_that.near,_that.distanceMin,_that.distanceMax,_that.distanceLimit,_that.elevationGainMin,_that.elevationGainMax,_that.elevationGainLimit,_that.elevationLossMin,_that.elevationLossMax,_that.elevationLossLimit,_that.startDate,_that.endDate,_that.completed,_that.liked,_that.offlineOnly,_that.sort,_that.sortOrder);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String q,  List<Category> category,  List<Subcategory> subcategory,  List<Tag> tags,  List<int> difficulty,  ActorSearchResult? author,  bool? public,  bool? shared,  bool? private,  TrailNear near,  double distanceMin,  double distanceMax,  double distanceLimit,  double elevationGainMin,  double elevationGainMax,  double elevationGainLimit,  double elevationLossMin,  double elevationLossMax,  double elevationLossLimit,  DateTime? startDate,  DateTime? endDate,  bool? completed,  bool? liked,  bool offlineOnly,  TrailFilterSort sort,  SortOrder sortOrder)?  $default,) {final _that = this;
switch (_that) {
case _TrailFilter() when $default != null:
return $default(_that.q,_that.category,_that.subcategory,_that.tags,_that.difficulty,_that.author,_that.public,_that.shared,_that.private,_that.near,_that.distanceMin,_that.distanceMax,_that.distanceLimit,_that.elevationGainMin,_that.elevationGainMax,_that.elevationGainLimit,_that.elevationLossMin,_that.elevationLossMax,_that.elevationLossLimit,_that.startDate,_that.endDate,_that.completed,_that.liked,_that.offlineOnly,_that.sort,_that.sortOrder);case _:
  return null;

}
}

}

/// @nodoc


class _TrailFilter extends TrailFilter {
  const _TrailFilter({required this.q, required final  List<Category> category, final  List<Subcategory> subcategory = const <Subcategory>[], required final  List<Tag> tags, required final  List<int> difficulty, this.author, this.public, this.shared, this.private, required this.near, required this.distanceMin, required this.distanceMax, required this.distanceLimit, required this.elevationGainMin, required this.elevationGainMax, required this.elevationGainLimit, required this.elevationLossMin, required this.elevationLossMax, required this.elevationLossLimit, this.startDate, this.endDate, this.completed, this.liked, this.offlineOnly = false, required this.sort, required this.sortOrder}): _category = category,_subcategory = subcategory,_tags = tags,_difficulty = difficulty,super._();
  

@override final  String q;
 final  List<Category> _category;
@override List<Category> get category {
  if (_category is EqualUnmodifiableListView) return _category;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_category);
}

 final  List<Subcategory> _subcategory;
@override@JsonKey() List<Subcategory> get subcategory {
  if (_subcategory is EqualUnmodifiableListView) return _subcategory;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_subcategory);
}

 final  List<Tag> _tags;
@override List<Tag> get tags {
  if (_tags is EqualUnmodifiableListView) return _tags;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_tags);
}

 final  List<int> _difficulty;
@override List<int> get difficulty {
  if (_difficulty is EqualUnmodifiableListView) return _difficulty;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_difficulty);
}

// 0, 1, 2
@override final  ActorSearchResult? author;
@override final  bool? public;
@override final  bool? shared;
@override final  bool? private;
@override final  TrailNear near;
@override final  double distanceMin;
@override final  double distanceMax;
@override final  double distanceLimit;
@override final  double elevationGainMin;
@override final  double elevationGainMax;
@override final  double elevationGainLimit;
@override final  double elevationLossMin;
@override final  double elevationLossMax;
@override final  double elevationLossLimit;
@override final  DateTime? startDate;
@override final  DateTime? endDate;
@override final  bool? completed;
@override final  bool? liked;
// Emitted server-side as an `id IN [...]` whitelist, but only when the
// caller supplies the ids -- see `toFilterText`'s [offlineTrailIds].
// `applyTrailFilter` carries a second, non-redundant half for unsynced
// local captures, which have no server id to whitelist.
@override@JsonKey() final  bool offlineOnly;
@override final  TrailFilterSort sort;
// "name" | "distance" | "elevation_gain" | "created"
@override final  SortOrder sortOrder;

/// Create a copy of TrailFilter
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailFilterCopyWith<_TrailFilter> get copyWith => __$TrailFilterCopyWithImpl<_TrailFilter>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailFilter&&(identical(other.q, q) || other.q == q)&&const DeepCollectionEquality().equals(other._category, _category)&&const DeepCollectionEquality().equals(other._subcategory, _subcategory)&&const DeepCollectionEquality().equals(other._tags, _tags)&&const DeepCollectionEquality().equals(other._difficulty, _difficulty)&&(identical(other.author, author) || other.author == author)&&(identical(other.public, public) || other.public == public)&&(identical(other.shared, shared) || other.shared == shared)&&(identical(other.private, private) || other.private == private)&&(identical(other.near, near) || other.near == near)&&(identical(other.distanceMin, distanceMin) || other.distanceMin == distanceMin)&&(identical(other.distanceMax, distanceMax) || other.distanceMax == distanceMax)&&(identical(other.distanceLimit, distanceLimit) || other.distanceLimit == distanceLimit)&&(identical(other.elevationGainMin, elevationGainMin) || other.elevationGainMin == elevationGainMin)&&(identical(other.elevationGainMax, elevationGainMax) || other.elevationGainMax == elevationGainMax)&&(identical(other.elevationGainLimit, elevationGainLimit) || other.elevationGainLimit == elevationGainLimit)&&(identical(other.elevationLossMin, elevationLossMin) || other.elevationLossMin == elevationLossMin)&&(identical(other.elevationLossMax, elevationLossMax) || other.elevationLossMax == elevationLossMax)&&(identical(other.elevationLossLimit, elevationLossLimit) || other.elevationLossLimit == elevationLossLimit)&&(identical(other.startDate, startDate) || other.startDate == startDate)&&(identical(other.endDate, endDate) || other.endDate == endDate)&&(identical(other.completed, completed) || other.completed == completed)&&(identical(other.liked, liked) || other.liked == liked)&&(identical(other.offlineOnly, offlineOnly) || other.offlineOnly == offlineOnly)&&(identical(other.sort, sort) || other.sort == sort)&&(identical(other.sortOrder, sortOrder) || other.sortOrder == sortOrder));
}


@override
int get hashCode => Object.hashAll([runtimeType,q,const DeepCollectionEquality().hash(_category),const DeepCollectionEquality().hash(_subcategory),const DeepCollectionEquality().hash(_tags),const DeepCollectionEquality().hash(_difficulty),author,public,shared,private,near,distanceMin,distanceMax,distanceLimit,elevationGainMin,elevationGainMax,elevationGainLimit,elevationLossMin,elevationLossMax,elevationLossLimit,startDate,endDate,completed,liked,offlineOnly,sort,sortOrder]);

@override
String toString() {
  return 'TrailFilter(q: $q, category: $category, subcategory: $subcategory, tags: $tags, difficulty: $difficulty, author: $author, public: $public, shared: $shared, private: $private, near: $near, distanceMin: $distanceMin, distanceMax: $distanceMax, distanceLimit: $distanceLimit, elevationGainMin: $elevationGainMin, elevationGainMax: $elevationGainMax, elevationGainLimit: $elevationGainLimit, elevationLossMin: $elevationLossMin, elevationLossMax: $elevationLossMax, elevationLossLimit: $elevationLossLimit, startDate: $startDate, endDate: $endDate, completed: $completed, liked: $liked, offlineOnly: $offlineOnly, sort: $sort, sortOrder: $sortOrder)';
}


}

/// @nodoc
abstract mixin class _$TrailFilterCopyWith<$Res> implements $TrailFilterCopyWith<$Res> {
  factory _$TrailFilterCopyWith(_TrailFilter value, $Res Function(_TrailFilter) _then) = __$TrailFilterCopyWithImpl;
@override @useResult
$Res call({
 String q, List<Category> category, List<Subcategory> subcategory, List<Tag> tags, List<int> difficulty, ActorSearchResult? author, bool? public, bool? shared, bool? private, TrailNear near, double distanceMin, double distanceMax, double distanceLimit, double elevationGainMin, double elevationGainMax, double elevationGainLimit, double elevationLossMin, double elevationLossMax, double elevationLossLimit, DateTime? startDate, DateTime? endDate, bool? completed, bool? liked, bool offlineOnly, TrailFilterSort sort, SortOrder sortOrder
});


@override $ActorSearchResultCopyWith<$Res>? get author;@override $TrailNearCopyWith<$Res> get near;

}
/// @nodoc
class __$TrailFilterCopyWithImpl<$Res>
    implements _$TrailFilterCopyWith<$Res> {
  __$TrailFilterCopyWithImpl(this._self, this._then);

  final _TrailFilter _self;
  final $Res Function(_TrailFilter) _then;

/// Create a copy of TrailFilter
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? q = null,Object? category = null,Object? subcategory = null,Object? tags = null,Object? difficulty = null,Object? author = freezed,Object? public = freezed,Object? shared = freezed,Object? private = freezed,Object? near = null,Object? distanceMin = null,Object? distanceMax = null,Object? distanceLimit = null,Object? elevationGainMin = null,Object? elevationGainMax = null,Object? elevationGainLimit = null,Object? elevationLossMin = null,Object? elevationLossMax = null,Object? elevationLossLimit = null,Object? startDate = freezed,Object? endDate = freezed,Object? completed = freezed,Object? liked = freezed,Object? offlineOnly = null,Object? sort = null,Object? sortOrder = null,}) {
  return _then(_TrailFilter(
q: null == q ? _self.q : q // ignore: cast_nullable_to_non_nullable
as String,category: null == category ? _self._category : category // ignore: cast_nullable_to_non_nullable
as List<Category>,subcategory: null == subcategory ? _self._subcategory : subcategory // ignore: cast_nullable_to_non_nullable
as List<Subcategory>,tags: null == tags ? _self._tags : tags // ignore: cast_nullable_to_non_nullable
as List<Tag>,difficulty: null == difficulty ? _self._difficulty : difficulty // ignore: cast_nullable_to_non_nullable
as List<int>,author: freezed == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as ActorSearchResult?,public: freezed == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool?,shared: freezed == shared ? _self.shared : shared // ignore: cast_nullable_to_non_nullable
as bool?,private: freezed == private ? _self.private : private // ignore: cast_nullable_to_non_nullable
as bool?,near: null == near ? _self.near : near // ignore: cast_nullable_to_non_nullable
as TrailNear,distanceMin: null == distanceMin ? _self.distanceMin : distanceMin // ignore: cast_nullable_to_non_nullable
as double,distanceMax: null == distanceMax ? _self.distanceMax : distanceMax // ignore: cast_nullable_to_non_nullable
as double,distanceLimit: null == distanceLimit ? _self.distanceLimit : distanceLimit // ignore: cast_nullable_to_non_nullable
as double,elevationGainMin: null == elevationGainMin ? _self.elevationGainMin : elevationGainMin // ignore: cast_nullable_to_non_nullable
as double,elevationGainMax: null == elevationGainMax ? _self.elevationGainMax : elevationGainMax // ignore: cast_nullable_to_non_nullable
as double,elevationGainLimit: null == elevationGainLimit ? _self.elevationGainLimit : elevationGainLimit // ignore: cast_nullable_to_non_nullable
as double,elevationLossMin: null == elevationLossMin ? _self.elevationLossMin : elevationLossMin // ignore: cast_nullable_to_non_nullable
as double,elevationLossMax: null == elevationLossMax ? _self.elevationLossMax : elevationLossMax // ignore: cast_nullable_to_non_nullable
as double,elevationLossLimit: null == elevationLossLimit ? _self.elevationLossLimit : elevationLossLimit // ignore: cast_nullable_to_non_nullable
as double,startDate: freezed == startDate ? _self.startDate : startDate // ignore: cast_nullable_to_non_nullable
as DateTime?,endDate: freezed == endDate ? _self.endDate : endDate // ignore: cast_nullable_to_non_nullable
as DateTime?,completed: freezed == completed ? _self.completed : completed // ignore: cast_nullable_to_non_nullable
as bool?,liked: freezed == liked ? _self.liked : liked // ignore: cast_nullable_to_non_nullable
as bool?,offlineOnly: null == offlineOnly ? _self.offlineOnly : offlineOnly // ignore: cast_nullable_to_non_nullable
as bool,sort: null == sort ? _self.sort : sort // ignore: cast_nullable_to_non_nullable
as TrailFilterSort,sortOrder: null == sortOrder ? _self.sortOrder : sortOrder // ignore: cast_nullable_to_non_nullable
as SortOrder,
  ));
}

/// Create a copy of TrailFilter
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorSearchResultCopyWith<$Res>? get author {
    if (_self.author == null) {
    return null;
  }

  return $ActorSearchResultCopyWith<$Res>(_self.author!, (value) {
    return _then(_self.copyWith(author: value));
  });
}/// Create a copy of TrailFilter
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailNearCopyWith<$Res> get near {
  
  return $TrailNearCopyWith<$Res>(_self.near, (value) {
    return _then(_self.copyWith(near: value));
  });
}
}


/// @nodoc
mixin _$TrailFilterValues {

@JsonKey(name: 'min_distance') double get minDistance;@JsonKey(name: 'max_distance') double get maxDistance;@JsonKey(name: 'min_elevation_gain') double get minElevationGain;@JsonKey(name: 'max_elevation_gain') double get maxElevationGain;@JsonKey(name: 'min_elevation_loss') double get minElevationLoss;@JsonKey(name: 'max_elevation_loss') double get maxElevationLoss;@JsonKey(name: 'min_duration') double get minDuration;@JsonKey(name: 'max_duration') double get maxDuration;
/// Create a copy of TrailFilterValues
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailFilterValuesCopyWith<TrailFilterValues> get copyWith => _$TrailFilterValuesCopyWithImpl<TrailFilterValues>(this as TrailFilterValues, _$identity);

  /// Serializes this TrailFilterValues to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailFilterValues&&(identical(other.minDistance, minDistance) || other.minDistance == minDistance)&&(identical(other.maxDistance, maxDistance) || other.maxDistance == maxDistance)&&(identical(other.minElevationGain, minElevationGain) || other.minElevationGain == minElevationGain)&&(identical(other.maxElevationGain, maxElevationGain) || other.maxElevationGain == maxElevationGain)&&(identical(other.minElevationLoss, minElevationLoss) || other.minElevationLoss == minElevationLoss)&&(identical(other.maxElevationLoss, maxElevationLoss) || other.maxElevationLoss == maxElevationLoss)&&(identical(other.minDuration, minDuration) || other.minDuration == minDuration)&&(identical(other.maxDuration, maxDuration) || other.maxDuration == maxDuration));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,minDistance,maxDistance,minElevationGain,maxElevationGain,minElevationLoss,maxElevationLoss,minDuration,maxDuration);

@override
String toString() {
  return 'TrailFilterValues(minDistance: $minDistance, maxDistance: $maxDistance, minElevationGain: $minElevationGain, maxElevationGain: $maxElevationGain, minElevationLoss: $minElevationLoss, maxElevationLoss: $maxElevationLoss, minDuration: $minDuration, maxDuration: $maxDuration)';
}


}

/// @nodoc
abstract mixin class $TrailFilterValuesCopyWith<$Res>  {
  factory $TrailFilterValuesCopyWith(TrailFilterValues value, $Res Function(TrailFilterValues) _then) = _$TrailFilterValuesCopyWithImpl;
@useResult
$Res call({
@JsonKey(name: 'min_distance') double minDistance,@JsonKey(name: 'max_distance') double maxDistance,@JsonKey(name: 'min_elevation_gain') double minElevationGain,@JsonKey(name: 'max_elevation_gain') double maxElevationGain,@JsonKey(name: 'min_elevation_loss') double minElevationLoss,@JsonKey(name: 'max_elevation_loss') double maxElevationLoss,@JsonKey(name: 'min_duration') double minDuration,@JsonKey(name: 'max_duration') double maxDuration
});




}
/// @nodoc
class _$TrailFilterValuesCopyWithImpl<$Res>
    implements $TrailFilterValuesCopyWith<$Res> {
  _$TrailFilterValuesCopyWithImpl(this._self, this._then);

  final TrailFilterValues _self;
  final $Res Function(TrailFilterValues) _then;

/// Create a copy of TrailFilterValues
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? minDistance = null,Object? maxDistance = null,Object? minElevationGain = null,Object? maxElevationGain = null,Object? minElevationLoss = null,Object? maxElevationLoss = null,Object? minDuration = null,Object? maxDuration = null,}) {
  return _then(_self.copyWith(
minDistance: null == minDistance ? _self.minDistance : minDistance // ignore: cast_nullable_to_non_nullable
as double,maxDistance: null == maxDistance ? _self.maxDistance : maxDistance // ignore: cast_nullable_to_non_nullable
as double,minElevationGain: null == minElevationGain ? _self.minElevationGain : minElevationGain // ignore: cast_nullable_to_non_nullable
as double,maxElevationGain: null == maxElevationGain ? _self.maxElevationGain : maxElevationGain // ignore: cast_nullable_to_non_nullable
as double,minElevationLoss: null == minElevationLoss ? _self.minElevationLoss : minElevationLoss // ignore: cast_nullable_to_non_nullable
as double,maxElevationLoss: null == maxElevationLoss ? _self.maxElevationLoss : maxElevationLoss // ignore: cast_nullable_to_non_nullable
as double,minDuration: null == minDuration ? _self.minDuration : minDuration // ignore: cast_nullable_to_non_nullable
as double,maxDuration: null == maxDuration ? _self.maxDuration : maxDuration // ignore: cast_nullable_to_non_nullable
as double,
  ));
}

}


/// Adds pattern-matching-related methods to [TrailFilterValues].
extension TrailFilterValuesPatterns on TrailFilterValues {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailFilterValues value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailFilterValues() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailFilterValues value)  $default,){
final _that = this;
switch (_that) {
case _TrailFilterValues():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailFilterValues value)?  $default,){
final _that = this;
switch (_that) {
case _TrailFilterValues() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function(@JsonKey(name: 'min_distance')  double minDistance, @JsonKey(name: 'max_distance')  double maxDistance, @JsonKey(name: 'min_elevation_gain')  double minElevationGain, @JsonKey(name: 'max_elevation_gain')  double maxElevationGain, @JsonKey(name: 'min_elevation_loss')  double minElevationLoss, @JsonKey(name: 'max_elevation_loss')  double maxElevationLoss, @JsonKey(name: 'min_duration')  double minDuration, @JsonKey(name: 'max_duration')  double maxDuration)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailFilterValues() when $default != null:
return $default(_that.minDistance,_that.maxDistance,_that.minElevationGain,_that.maxElevationGain,_that.minElevationLoss,_that.maxElevationLoss,_that.minDuration,_that.maxDuration);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function(@JsonKey(name: 'min_distance')  double minDistance, @JsonKey(name: 'max_distance')  double maxDistance, @JsonKey(name: 'min_elevation_gain')  double minElevationGain, @JsonKey(name: 'max_elevation_gain')  double maxElevationGain, @JsonKey(name: 'min_elevation_loss')  double minElevationLoss, @JsonKey(name: 'max_elevation_loss')  double maxElevationLoss, @JsonKey(name: 'min_duration')  double minDuration, @JsonKey(name: 'max_duration')  double maxDuration)  $default,) {final _that = this;
switch (_that) {
case _TrailFilterValues():
return $default(_that.minDistance,_that.maxDistance,_that.minElevationGain,_that.maxElevationGain,_that.minElevationLoss,_that.maxElevationLoss,_that.minDuration,_that.maxDuration);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function(@JsonKey(name: 'min_distance')  double minDistance, @JsonKey(name: 'max_distance')  double maxDistance, @JsonKey(name: 'min_elevation_gain')  double minElevationGain, @JsonKey(name: 'max_elevation_gain')  double maxElevationGain, @JsonKey(name: 'min_elevation_loss')  double minElevationLoss, @JsonKey(name: 'max_elevation_loss')  double maxElevationLoss, @JsonKey(name: 'min_duration')  double minDuration, @JsonKey(name: 'max_duration')  double maxDuration)?  $default,) {final _that = this;
switch (_that) {
case _TrailFilterValues() when $default != null:
return $default(_that.minDistance,_that.maxDistance,_that.minElevationGain,_that.maxElevationGain,_that.minElevationLoss,_that.maxElevationLoss,_that.minDuration,_that.maxDuration);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _TrailFilterValues implements TrailFilterValues {
  const _TrailFilterValues({@JsonKey(name: 'min_distance') required this.minDistance, @JsonKey(name: 'max_distance') required this.maxDistance, @JsonKey(name: 'min_elevation_gain') required this.minElevationGain, @JsonKey(name: 'max_elevation_gain') required this.maxElevationGain, @JsonKey(name: 'min_elevation_loss') required this.minElevationLoss, @JsonKey(name: 'max_elevation_loss') required this.maxElevationLoss, @JsonKey(name: 'min_duration') required this.minDuration, @JsonKey(name: 'max_duration') required this.maxDuration});
  factory _TrailFilterValues.fromJson(Map<String, dynamic> json) => _$TrailFilterValuesFromJson(json);

@override@JsonKey(name: 'min_distance') final  double minDistance;
@override@JsonKey(name: 'max_distance') final  double maxDistance;
@override@JsonKey(name: 'min_elevation_gain') final  double minElevationGain;
@override@JsonKey(name: 'max_elevation_gain') final  double maxElevationGain;
@override@JsonKey(name: 'min_elevation_loss') final  double minElevationLoss;
@override@JsonKey(name: 'max_elevation_loss') final  double maxElevationLoss;
@override@JsonKey(name: 'min_duration') final  double minDuration;
@override@JsonKey(name: 'max_duration') final  double maxDuration;

/// Create a copy of TrailFilterValues
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailFilterValuesCopyWith<_TrailFilterValues> get copyWith => __$TrailFilterValuesCopyWithImpl<_TrailFilterValues>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$TrailFilterValuesToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailFilterValues&&(identical(other.minDistance, minDistance) || other.minDistance == minDistance)&&(identical(other.maxDistance, maxDistance) || other.maxDistance == maxDistance)&&(identical(other.minElevationGain, minElevationGain) || other.minElevationGain == minElevationGain)&&(identical(other.maxElevationGain, maxElevationGain) || other.maxElevationGain == maxElevationGain)&&(identical(other.minElevationLoss, minElevationLoss) || other.minElevationLoss == minElevationLoss)&&(identical(other.maxElevationLoss, maxElevationLoss) || other.maxElevationLoss == maxElevationLoss)&&(identical(other.minDuration, minDuration) || other.minDuration == minDuration)&&(identical(other.maxDuration, maxDuration) || other.maxDuration == maxDuration));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,minDistance,maxDistance,minElevationGain,maxElevationGain,minElevationLoss,maxElevationLoss,minDuration,maxDuration);

@override
String toString() {
  return 'TrailFilterValues(minDistance: $minDistance, maxDistance: $maxDistance, minElevationGain: $minElevationGain, maxElevationGain: $maxElevationGain, minElevationLoss: $minElevationLoss, maxElevationLoss: $maxElevationLoss, minDuration: $minDuration, maxDuration: $maxDuration)';
}


}

/// @nodoc
abstract mixin class _$TrailFilterValuesCopyWith<$Res> implements $TrailFilterValuesCopyWith<$Res> {
  factory _$TrailFilterValuesCopyWith(_TrailFilterValues value, $Res Function(_TrailFilterValues) _then) = __$TrailFilterValuesCopyWithImpl;
@override @useResult
$Res call({
@JsonKey(name: 'min_distance') double minDistance,@JsonKey(name: 'max_distance') double maxDistance,@JsonKey(name: 'min_elevation_gain') double minElevationGain,@JsonKey(name: 'max_elevation_gain') double maxElevationGain,@JsonKey(name: 'min_elevation_loss') double minElevationLoss,@JsonKey(name: 'max_elevation_loss') double maxElevationLoss,@JsonKey(name: 'min_duration') double minDuration,@JsonKey(name: 'max_duration') double maxDuration
});




}
/// @nodoc
class __$TrailFilterValuesCopyWithImpl<$Res>
    implements _$TrailFilterValuesCopyWith<$Res> {
  __$TrailFilterValuesCopyWithImpl(this._self, this._then);

  final _TrailFilterValues _self;
  final $Res Function(_TrailFilterValues) _then;

/// Create a copy of TrailFilterValues
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? minDistance = null,Object? maxDistance = null,Object? minElevationGain = null,Object? maxElevationGain = null,Object? minElevationLoss = null,Object? maxElevationLoss = null,Object? minDuration = null,Object? maxDuration = null,}) {
  return _then(_TrailFilterValues(
minDistance: null == minDistance ? _self.minDistance : minDistance // ignore: cast_nullable_to_non_nullable
as double,maxDistance: null == maxDistance ? _self.maxDistance : maxDistance // ignore: cast_nullable_to_non_nullable
as double,minElevationGain: null == minElevationGain ? _self.minElevationGain : minElevationGain // ignore: cast_nullable_to_non_nullable
as double,maxElevationGain: null == maxElevationGain ? _self.maxElevationGain : maxElevationGain // ignore: cast_nullable_to_non_nullable
as double,minElevationLoss: null == minElevationLoss ? _self.minElevationLoss : minElevationLoss // ignore: cast_nullable_to_non_nullable
as double,maxElevationLoss: null == maxElevationLoss ? _self.maxElevationLoss : maxElevationLoss // ignore: cast_nullable_to_non_nullable
as double,minDuration: null == minDuration ? _self.minDuration : minDuration // ignore: cast_nullable_to_non_nullable
as double,maxDuration: null == maxDuration ? _self.maxDuration : maxDuration // ignore: cast_nullable_to_non_nullable
as double,
  ));
}


}


/// @nodoc
mixin _$TrailBoundingBox {

@JsonKey(name: 'max_lat') double get maxLat;@JsonKey(name: 'min_lat') double get minLat;@JsonKey(name: 'max_lon') double get maxLon;@JsonKey(name: 'min_lon') double get minLon;@JsonKey(name: 'has_trails') bool get hasTrails;
/// Create a copy of TrailBoundingBox
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailBoundingBoxCopyWith<TrailBoundingBox> get copyWith => _$TrailBoundingBoxCopyWithImpl<TrailBoundingBox>(this as TrailBoundingBox, _$identity);

  /// Serializes this TrailBoundingBox to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailBoundingBox&&(identical(other.maxLat, maxLat) || other.maxLat == maxLat)&&(identical(other.minLat, minLat) || other.minLat == minLat)&&(identical(other.maxLon, maxLon) || other.maxLon == maxLon)&&(identical(other.minLon, minLon) || other.minLon == minLon)&&(identical(other.hasTrails, hasTrails) || other.hasTrails == hasTrails));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,maxLat,minLat,maxLon,minLon,hasTrails);

@override
String toString() {
  return 'TrailBoundingBox(maxLat: $maxLat, minLat: $minLat, maxLon: $maxLon, minLon: $minLon, hasTrails: $hasTrails)';
}


}

/// @nodoc
abstract mixin class $TrailBoundingBoxCopyWith<$Res>  {
  factory $TrailBoundingBoxCopyWith(TrailBoundingBox value, $Res Function(TrailBoundingBox) _then) = _$TrailBoundingBoxCopyWithImpl;
@useResult
$Res call({
@JsonKey(name: 'max_lat') double maxLat,@JsonKey(name: 'min_lat') double minLat,@JsonKey(name: 'max_lon') double maxLon,@JsonKey(name: 'min_lon') double minLon,@JsonKey(name: 'has_trails') bool hasTrails
});




}
/// @nodoc
class _$TrailBoundingBoxCopyWithImpl<$Res>
    implements $TrailBoundingBoxCopyWith<$Res> {
  _$TrailBoundingBoxCopyWithImpl(this._self, this._then);

  final TrailBoundingBox _self;
  final $Res Function(TrailBoundingBox) _then;

/// Create a copy of TrailBoundingBox
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? maxLat = null,Object? minLat = null,Object? maxLon = null,Object? minLon = null,Object? hasTrails = null,}) {
  return _then(_self.copyWith(
maxLat: null == maxLat ? _self.maxLat : maxLat // ignore: cast_nullable_to_non_nullable
as double,minLat: null == minLat ? _self.minLat : minLat // ignore: cast_nullable_to_non_nullable
as double,maxLon: null == maxLon ? _self.maxLon : maxLon // ignore: cast_nullable_to_non_nullable
as double,minLon: null == minLon ? _self.minLon : minLon // ignore: cast_nullable_to_non_nullable
as double,hasTrails: null == hasTrails ? _self.hasTrails : hasTrails // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}

}


/// Adds pattern-matching-related methods to [TrailBoundingBox].
extension TrailBoundingBoxPatterns on TrailBoundingBox {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailBoundingBox value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailBoundingBox() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailBoundingBox value)  $default,){
final _that = this;
switch (_that) {
case _TrailBoundingBox():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailBoundingBox value)?  $default,){
final _that = this;
switch (_that) {
case _TrailBoundingBox() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function(@JsonKey(name: 'max_lat')  double maxLat, @JsonKey(name: 'min_lat')  double minLat, @JsonKey(name: 'max_lon')  double maxLon, @JsonKey(name: 'min_lon')  double minLon, @JsonKey(name: 'has_trails')  bool hasTrails)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailBoundingBox() when $default != null:
return $default(_that.maxLat,_that.minLat,_that.maxLon,_that.minLon,_that.hasTrails);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function(@JsonKey(name: 'max_lat')  double maxLat, @JsonKey(name: 'min_lat')  double minLat, @JsonKey(name: 'max_lon')  double maxLon, @JsonKey(name: 'min_lon')  double minLon, @JsonKey(name: 'has_trails')  bool hasTrails)  $default,) {final _that = this;
switch (_that) {
case _TrailBoundingBox():
return $default(_that.maxLat,_that.minLat,_that.maxLon,_that.minLon,_that.hasTrails);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function(@JsonKey(name: 'max_lat')  double maxLat, @JsonKey(name: 'min_lat')  double minLat, @JsonKey(name: 'max_lon')  double maxLon, @JsonKey(name: 'min_lon')  double minLon, @JsonKey(name: 'has_trails')  bool hasTrails)?  $default,) {final _that = this;
switch (_that) {
case _TrailBoundingBox() when $default != null:
return $default(_that.maxLat,_that.minLat,_that.maxLon,_that.minLon,_that.hasTrails);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _TrailBoundingBox implements TrailBoundingBox {
  const _TrailBoundingBox({@JsonKey(name: 'max_lat') required this.maxLat, @JsonKey(name: 'min_lat') required this.minLat, @JsonKey(name: 'max_lon') required this.maxLon, @JsonKey(name: 'min_lon') required this.minLon, @JsonKey(name: 'has_trails') this.hasTrails = false});
  factory _TrailBoundingBox.fromJson(Map<String, dynamic> json) => _$TrailBoundingBoxFromJson(json);

@override@JsonKey(name: 'max_lat') final  double maxLat;
@override@JsonKey(name: 'min_lat') final  double minLat;
@override@JsonKey(name: 'max_lon') final  double maxLon;
@override@JsonKey(name: 'min_lon') final  double minLon;
@override@JsonKey(name: 'has_trails') final  bool hasTrails;

/// Create a copy of TrailBoundingBox
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailBoundingBoxCopyWith<_TrailBoundingBox> get copyWith => __$TrailBoundingBoxCopyWithImpl<_TrailBoundingBox>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$TrailBoundingBoxToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailBoundingBox&&(identical(other.maxLat, maxLat) || other.maxLat == maxLat)&&(identical(other.minLat, minLat) || other.minLat == minLat)&&(identical(other.maxLon, maxLon) || other.maxLon == maxLon)&&(identical(other.minLon, minLon) || other.minLon == minLon)&&(identical(other.hasTrails, hasTrails) || other.hasTrails == hasTrails));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,maxLat,minLat,maxLon,minLon,hasTrails);

@override
String toString() {
  return 'TrailBoundingBox(maxLat: $maxLat, minLat: $minLat, maxLon: $maxLon, minLon: $minLon, hasTrails: $hasTrails)';
}


}

/// @nodoc
abstract mixin class _$TrailBoundingBoxCopyWith<$Res> implements $TrailBoundingBoxCopyWith<$Res> {
  factory _$TrailBoundingBoxCopyWith(_TrailBoundingBox value, $Res Function(_TrailBoundingBox) _then) = __$TrailBoundingBoxCopyWithImpl;
@override @useResult
$Res call({
@JsonKey(name: 'max_lat') double maxLat,@JsonKey(name: 'min_lat') double minLat,@JsonKey(name: 'max_lon') double maxLon,@JsonKey(name: 'min_lon') double minLon,@JsonKey(name: 'has_trails') bool hasTrails
});




}
/// @nodoc
class __$TrailBoundingBoxCopyWithImpl<$Res>
    implements _$TrailBoundingBoxCopyWith<$Res> {
  __$TrailBoundingBoxCopyWithImpl(this._self, this._then);

  final _TrailBoundingBox _self;
  final $Res Function(_TrailBoundingBox) _then;

/// Create a copy of TrailBoundingBox
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? maxLat = null,Object? minLat = null,Object? maxLon = null,Object? minLon = null,Object? hasTrails = null,}) {
  return _then(_TrailBoundingBox(
maxLat: null == maxLat ? _self.maxLat : maxLat // ignore: cast_nullable_to_non_nullable
as double,minLat: null == minLat ? _self.minLat : minLat // ignore: cast_nullable_to_non_nullable
as double,maxLon: null == maxLon ? _self.maxLon : maxLon // ignore: cast_nullable_to_non_nullable
as double,minLon: null == minLon ? _self.minLon : minLon // ignore: cast_nullable_to_non_nullable
as double,hasTrails: null == hasTrails ? _self.hasTrails : hasTrails // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}


}

// dart format on
