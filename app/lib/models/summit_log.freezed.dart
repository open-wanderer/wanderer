// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'summit_log.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$SummitLog {

 String? get id; String get date; String get text; String? get gpx; List<String> get photos; double? get distance;@JsonKey(name: 'elevation_gain') double? get elevationGain;@JsonKey(name: 'elevation_loss') double? get elevationLoss; double? get duration; String get author; String? get trail; String? get iri; String? get created; SummitLogExpand? get expand;
/// Create a copy of SummitLog
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SummitLogCopyWith<SummitLog> get copyWith => _$SummitLogCopyWithImpl<SummitLog>(this as SummitLog, _$identity);

  /// Serializes this SummitLog to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is SummitLog&&(identical(other.id, id) || other.id == id)&&(identical(other.date, date) || other.date == date)&&(identical(other.text, text) || other.text == text)&&(identical(other.gpx, gpx) || other.gpx == gpx)&&const DeepCollectionEquality().equals(other.photos, photos)&&(identical(other.distance, distance) || other.distance == distance)&&(identical(other.elevationGain, elevationGain) || other.elevationGain == elevationGain)&&(identical(other.elevationLoss, elevationLoss) || other.elevationLoss == elevationLoss)&&(identical(other.duration, duration) || other.duration == duration)&&(identical(other.author, author) || other.author == author)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.created, created) || other.created == created)&&(identical(other.expand, expand) || other.expand == expand));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,date,text,gpx,const DeepCollectionEquality().hash(photos),distance,elevationGain,elevationLoss,duration,author,trail,iri,created,expand);

@override
String toString() {
  return 'SummitLog(id: $id, date: $date, text: $text, gpx: $gpx, photos: $photos, distance: $distance, elevationGain: $elevationGain, elevationLoss: $elevationLoss, duration: $duration, author: $author, trail: $trail, iri: $iri, created: $created, expand: $expand)';
}


}

/// @nodoc
abstract mixin class $SummitLogCopyWith<$Res>  {
  factory $SummitLogCopyWith(SummitLog value, $Res Function(SummitLog) _then) = _$SummitLogCopyWithImpl;
@useResult
$Res call({
 String? id, String date, String text, String? gpx, List<String> photos, double? distance,@JsonKey(name: 'elevation_gain') double? elevationGain,@JsonKey(name: 'elevation_loss') double? elevationLoss, double? duration, String author, String? trail, String? iri, String? created, SummitLogExpand? expand
});


$SummitLogExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class _$SummitLogCopyWithImpl<$Res>
    implements $SummitLogCopyWith<$Res> {
  _$SummitLogCopyWithImpl(this._self, this._then);

  final SummitLog _self;
  final $Res Function(SummitLog) _then;

/// Create a copy of SummitLog
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = freezed,Object? date = null,Object? text = null,Object? gpx = freezed,Object? photos = null,Object? distance = freezed,Object? elevationGain = freezed,Object? elevationLoss = freezed,Object? duration = freezed,Object? author = null,Object? trail = freezed,Object? iri = freezed,Object? created = freezed,Object? expand = freezed,}) {
  return _then(_self.copyWith(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,date: null == date ? _self.date : date // ignore: cast_nullable_to_non_nullable
as String,text: null == text ? _self.text : text // ignore: cast_nullable_to_non_nullable
as String,gpx: freezed == gpx ? _self.gpx : gpx // ignore: cast_nullable_to_non_nullable
as String?,photos: null == photos ? _self.photos : photos // ignore: cast_nullable_to_non_nullable
as List<String>,distance: freezed == distance ? _self.distance : distance // ignore: cast_nullable_to_non_nullable
as double?,elevationGain: freezed == elevationGain ? _self.elevationGain : elevationGain // ignore: cast_nullable_to_non_nullable
as double?,elevationLoss: freezed == elevationLoss ? _self.elevationLoss : elevationLoss // ignore: cast_nullable_to_non_nullable
as double?,duration: freezed == duration ? _self.duration : duration // ignore: cast_nullable_to_non_nullable
as double?,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,trail: freezed == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as String?,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,created: freezed == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String?,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as SummitLogExpand?,
  ));
}
/// Create a copy of SummitLog
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$SummitLogExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $SummitLogExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}


/// Adds pattern-matching-related methods to [SummitLog].
extension SummitLogPatterns on SummitLog {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _SummitLog value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _SummitLog() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _SummitLog value)  $default,){
final _that = this;
switch (_that) {
case _SummitLog():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _SummitLog value)?  $default,){
final _that = this;
switch (_that) {
case _SummitLog() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String? id,  String date,  String text,  String? gpx,  List<String> photos,  double? distance, @JsonKey(name: 'elevation_gain')  double? elevationGain, @JsonKey(name: 'elevation_loss')  double? elevationLoss,  double? duration,  String author,  String? trail,  String? iri,  String? created,  SummitLogExpand? expand)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _SummitLog() when $default != null:
return $default(_that.id,_that.date,_that.text,_that.gpx,_that.photos,_that.distance,_that.elevationGain,_that.elevationLoss,_that.duration,_that.author,_that.trail,_that.iri,_that.created,_that.expand);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String? id,  String date,  String text,  String? gpx,  List<String> photos,  double? distance, @JsonKey(name: 'elevation_gain')  double? elevationGain, @JsonKey(name: 'elevation_loss')  double? elevationLoss,  double? duration,  String author,  String? trail,  String? iri,  String? created,  SummitLogExpand? expand)  $default,) {final _that = this;
switch (_that) {
case _SummitLog():
return $default(_that.id,_that.date,_that.text,_that.gpx,_that.photos,_that.distance,_that.elevationGain,_that.elevationLoss,_that.duration,_that.author,_that.trail,_that.iri,_that.created,_that.expand);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String? id,  String date,  String text,  String? gpx,  List<String> photos,  double? distance, @JsonKey(name: 'elevation_gain')  double? elevationGain, @JsonKey(name: 'elevation_loss')  double? elevationLoss,  double? duration,  String author,  String? trail,  String? iri,  String? created,  SummitLogExpand? expand)?  $default,) {final _that = this;
switch (_that) {
case _SummitLog() when $default != null:
return $default(_that.id,_that.date,_that.text,_that.gpx,_that.photos,_that.distance,_that.elevationGain,_that.elevationLoss,_that.duration,_that.author,_that.trail,_that.iri,_that.created,_that.expand);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _SummitLog implements SummitLog {
  const _SummitLog({this.id, required this.date, this.text = "", this.gpx, final  List<String> photos = const [], this.distance, @JsonKey(name: 'elevation_gain') this.elevationGain, @JsonKey(name: 'elevation_loss') this.elevationLoss, this.duration, this.author = "000000000000000", this.trail, this.iri, this.created, this.expand}): _photos = photos;
  factory _SummitLog.fromJson(Map<String, dynamic> json) => _$SummitLogFromJson(json);

@override final  String? id;
@override final  String date;
@override@JsonKey() final  String text;
@override final  String? gpx;
 final  List<String> _photos;
@override@JsonKey() List<String> get photos {
  if (_photos is EqualUnmodifiableListView) return _photos;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_photos);
}

@override final  double? distance;
@override@JsonKey(name: 'elevation_gain') final  double? elevationGain;
@override@JsonKey(name: 'elevation_loss') final  double? elevationLoss;
@override final  double? duration;
@override@JsonKey() final  String author;
@override final  String? trail;
@override final  String? iri;
@override final  String? created;
@override final  SummitLogExpand? expand;

/// Create a copy of SummitLog
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SummitLogCopyWith<_SummitLog> get copyWith => __$SummitLogCopyWithImpl<_SummitLog>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SummitLogToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _SummitLog&&(identical(other.id, id) || other.id == id)&&(identical(other.date, date) || other.date == date)&&(identical(other.text, text) || other.text == text)&&(identical(other.gpx, gpx) || other.gpx == gpx)&&const DeepCollectionEquality().equals(other._photos, _photos)&&(identical(other.distance, distance) || other.distance == distance)&&(identical(other.elevationGain, elevationGain) || other.elevationGain == elevationGain)&&(identical(other.elevationLoss, elevationLoss) || other.elevationLoss == elevationLoss)&&(identical(other.duration, duration) || other.duration == duration)&&(identical(other.author, author) || other.author == author)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.created, created) || other.created == created)&&(identical(other.expand, expand) || other.expand == expand));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,date,text,gpx,const DeepCollectionEquality().hash(_photos),distance,elevationGain,elevationLoss,duration,author,trail,iri,created,expand);

@override
String toString() {
  return 'SummitLog(id: $id, date: $date, text: $text, gpx: $gpx, photos: $photos, distance: $distance, elevationGain: $elevationGain, elevationLoss: $elevationLoss, duration: $duration, author: $author, trail: $trail, iri: $iri, created: $created, expand: $expand)';
}


}

/// @nodoc
abstract mixin class _$SummitLogCopyWith<$Res> implements $SummitLogCopyWith<$Res> {
  factory _$SummitLogCopyWith(_SummitLog value, $Res Function(_SummitLog) _then) = __$SummitLogCopyWithImpl;
@override @useResult
$Res call({
 String? id, String date, String text, String? gpx, List<String> photos, double? distance,@JsonKey(name: 'elevation_gain') double? elevationGain,@JsonKey(name: 'elevation_loss') double? elevationLoss, double? duration, String author, String? trail, String? iri, String? created, SummitLogExpand? expand
});


@override $SummitLogExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class __$SummitLogCopyWithImpl<$Res>
    implements _$SummitLogCopyWith<$Res> {
  __$SummitLogCopyWithImpl(this._self, this._then);

  final _SummitLog _self;
  final $Res Function(_SummitLog) _then;

/// Create a copy of SummitLog
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = freezed,Object? date = null,Object? text = null,Object? gpx = freezed,Object? photos = null,Object? distance = freezed,Object? elevationGain = freezed,Object? elevationLoss = freezed,Object? duration = freezed,Object? author = null,Object? trail = freezed,Object? iri = freezed,Object? created = freezed,Object? expand = freezed,}) {
  return _then(_SummitLog(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,date: null == date ? _self.date : date // ignore: cast_nullable_to_non_nullable
as String,text: null == text ? _self.text : text // ignore: cast_nullable_to_non_nullable
as String,gpx: freezed == gpx ? _self.gpx : gpx // ignore: cast_nullable_to_non_nullable
as String?,photos: null == photos ? _self._photos : photos // ignore: cast_nullable_to_non_nullable
as List<String>,distance: freezed == distance ? _self.distance : distance // ignore: cast_nullable_to_non_nullable
as double?,elevationGain: freezed == elevationGain ? _self.elevationGain : elevationGain // ignore: cast_nullable_to_non_nullable
as double?,elevationLoss: freezed == elevationLoss ? _self.elevationLoss : elevationLoss // ignore: cast_nullable_to_non_nullable
as double?,duration: freezed == duration ? _self.duration : duration // ignore: cast_nullable_to_non_nullable
as double?,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,trail: freezed == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as String?,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,created: freezed == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String?,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as SummitLogExpand?,
  ));
}

/// Create a copy of SummitLog
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$SummitLogExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $SummitLogExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}


/// @nodoc
mixin _$SummitLogExpand {

@JsonKey(name: 'gpx_data') String? get gpxData; Trail? get trail; Actor? get author;
/// Create a copy of SummitLogExpand
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SummitLogExpandCopyWith<SummitLogExpand> get copyWith => _$SummitLogExpandCopyWithImpl<SummitLogExpand>(this as SummitLogExpand, _$identity);

  /// Serializes this SummitLogExpand to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is SummitLogExpand&&(identical(other.gpxData, gpxData) || other.gpxData == gpxData)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.author, author) || other.author == author));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,gpxData,trail,author);

@override
String toString() {
  return 'SummitLogExpand(gpxData: $gpxData, trail: $trail, author: $author)';
}


}

/// @nodoc
abstract mixin class $SummitLogExpandCopyWith<$Res>  {
  factory $SummitLogExpandCopyWith(SummitLogExpand value, $Res Function(SummitLogExpand) _then) = _$SummitLogExpandCopyWithImpl;
@useResult
$Res call({
@JsonKey(name: 'gpx_data') String? gpxData, Trail? trail, Actor? author
});


$TrailCopyWith<$Res>? get trail;$ActorCopyWith<$Res>? get author;

}
/// @nodoc
class _$SummitLogExpandCopyWithImpl<$Res>
    implements $SummitLogExpandCopyWith<$Res> {
  _$SummitLogExpandCopyWithImpl(this._self, this._then);

  final SummitLogExpand _self;
  final $Res Function(SummitLogExpand) _then;

/// Create a copy of SummitLogExpand
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? gpxData = freezed,Object? trail = freezed,Object? author = freezed,}) {
  return _then(_self.copyWith(
gpxData: freezed == gpxData ? _self.gpxData : gpxData // ignore: cast_nullable_to_non_nullable
as String?,trail: freezed == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as Trail?,author: freezed == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as Actor?,
  ));
}
/// Create a copy of SummitLogExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailCopyWith<$Res>? get trail {
    if (_self.trail == null) {
    return null;
  }

  return $TrailCopyWith<$Res>(_self.trail!, (value) {
    return _then(_self.copyWith(trail: value));
  });
}/// Create a copy of SummitLogExpand
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


/// Adds pattern-matching-related methods to [SummitLogExpand].
extension SummitLogExpandPatterns on SummitLogExpand {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _SummitLogExpand value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _SummitLogExpand() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _SummitLogExpand value)  $default,){
final _that = this;
switch (_that) {
case _SummitLogExpand():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _SummitLogExpand value)?  $default,){
final _that = this;
switch (_that) {
case _SummitLogExpand() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function(@JsonKey(name: 'gpx_data')  String? gpxData,  Trail? trail,  Actor? author)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _SummitLogExpand() when $default != null:
return $default(_that.gpxData,_that.trail,_that.author);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function(@JsonKey(name: 'gpx_data')  String? gpxData,  Trail? trail,  Actor? author)  $default,) {final _that = this;
switch (_that) {
case _SummitLogExpand():
return $default(_that.gpxData,_that.trail,_that.author);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function(@JsonKey(name: 'gpx_data')  String? gpxData,  Trail? trail,  Actor? author)?  $default,) {final _that = this;
switch (_that) {
case _SummitLogExpand() when $default != null:
return $default(_that.gpxData,_that.trail,_that.author);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _SummitLogExpand implements SummitLogExpand {
  const _SummitLogExpand({@JsonKey(name: 'gpx_data') this.gpxData, this.trail, this.author});
  factory _SummitLogExpand.fromJson(Map<String, dynamic> json) => _$SummitLogExpandFromJson(json);

@override@JsonKey(name: 'gpx_data') final  String? gpxData;
@override final  Trail? trail;
@override final  Actor? author;

/// Create a copy of SummitLogExpand
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SummitLogExpandCopyWith<_SummitLogExpand> get copyWith => __$SummitLogExpandCopyWithImpl<_SummitLogExpand>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SummitLogExpandToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _SummitLogExpand&&(identical(other.gpxData, gpxData) || other.gpxData == gpxData)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.author, author) || other.author == author));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,gpxData,trail,author);

@override
String toString() {
  return 'SummitLogExpand(gpxData: $gpxData, trail: $trail, author: $author)';
}


}

/// @nodoc
abstract mixin class _$SummitLogExpandCopyWith<$Res> implements $SummitLogExpandCopyWith<$Res> {
  factory _$SummitLogExpandCopyWith(_SummitLogExpand value, $Res Function(_SummitLogExpand) _then) = __$SummitLogExpandCopyWithImpl;
@override @useResult
$Res call({
@JsonKey(name: 'gpx_data') String? gpxData, Trail? trail, Actor? author
});


@override $TrailCopyWith<$Res>? get trail;@override $ActorCopyWith<$Res>? get author;

}
/// @nodoc
class __$SummitLogExpandCopyWithImpl<$Res>
    implements _$SummitLogExpandCopyWith<$Res> {
  __$SummitLogExpandCopyWithImpl(this._self, this._then);

  final _SummitLogExpand _self;
  final $Res Function(_SummitLogExpand) _then;

/// Create a copy of SummitLogExpand
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? gpxData = freezed,Object? trail = freezed,Object? author = freezed,}) {
  return _then(_SummitLogExpand(
gpxData: freezed == gpxData ? _self.gpxData : gpxData // ignore: cast_nullable_to_non_nullable
as String?,trail: freezed == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as Trail?,author: freezed == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as Actor?,
  ));
}

/// Create a copy of SummitLogExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailCopyWith<$Res>? get trail {
    if (_self.trail == null) {
    return null;
  }

  return $TrailCopyWith<$Res>(_self.trail!, (value) {
    return _then(_self.copyWith(trail: value));
  });
}/// Create a copy of SummitLogExpand
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

// dart format on
