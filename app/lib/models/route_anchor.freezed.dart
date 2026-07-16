// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'route_anchor.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$RouteAnchor {

 String get id;// generated via UniqueKey().toString() at creation
 double get lat; double get lon;
/// Create a copy of RouteAnchor
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$RouteAnchorCopyWith<RouteAnchor> get copyWith => _$RouteAnchorCopyWithImpl<RouteAnchor>(this as RouteAnchor, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is RouteAnchor&&(identical(other.id, id) || other.id == id)&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon));
}


@override
int get hashCode => Object.hash(runtimeType,id,lat,lon);

@override
String toString() {
  return 'RouteAnchor(id: $id, lat: $lat, lon: $lon)';
}


}

/// @nodoc
abstract mixin class $RouteAnchorCopyWith<$Res>  {
  factory $RouteAnchorCopyWith(RouteAnchor value, $Res Function(RouteAnchor) _then) = _$RouteAnchorCopyWithImpl;
@useResult
$Res call({
 String id, double lat, double lon
});




}
/// @nodoc
class _$RouteAnchorCopyWithImpl<$Res>
    implements $RouteAnchorCopyWith<$Res> {
  _$RouteAnchorCopyWithImpl(this._self, this._then);

  final RouteAnchor _self;
  final $Res Function(RouteAnchor) _then;

/// Create a copy of RouteAnchor
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? lat = null,Object? lon = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,lat: null == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double,lon: null == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double,
  ));
}

}


/// Adds pattern-matching-related methods to [RouteAnchor].
extension RouteAnchorPatterns on RouteAnchor {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _RouteAnchor value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _RouteAnchor() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _RouteAnchor value)  $default,){
final _that = this;
switch (_that) {
case _RouteAnchor():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _RouteAnchor value)?  $default,){
final _that = this;
switch (_that) {
case _RouteAnchor() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  double lat,  double lon)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _RouteAnchor() when $default != null:
return $default(_that.id,_that.lat,_that.lon);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  double lat,  double lon)  $default,) {final _that = this;
switch (_that) {
case _RouteAnchor():
return $default(_that.id,_that.lat,_that.lon);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  double lat,  double lon)?  $default,) {final _that = this;
switch (_that) {
case _RouteAnchor() when $default != null:
return $default(_that.id,_that.lat,_that.lon);case _:
  return null;

}
}

}

/// @nodoc


class _RouteAnchor extends RouteAnchor {
  const _RouteAnchor({required this.id, required this.lat, required this.lon}): super._();
  

@override final  String id;
// generated via UniqueKey().toString() at creation
@override final  double lat;
@override final  double lon;

/// Create a copy of RouteAnchor
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$RouteAnchorCopyWith<_RouteAnchor> get copyWith => __$RouteAnchorCopyWithImpl<_RouteAnchor>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _RouteAnchor&&(identical(other.id, id) || other.id == id)&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon));
}


@override
int get hashCode => Object.hash(runtimeType,id,lat,lon);

@override
String toString() {
  return 'RouteAnchor(id: $id, lat: $lat, lon: $lon)';
}


}

/// @nodoc
abstract mixin class _$RouteAnchorCopyWith<$Res> implements $RouteAnchorCopyWith<$Res> {
  factory _$RouteAnchorCopyWith(_RouteAnchor value, $Res Function(_RouteAnchor) _then) = __$RouteAnchorCopyWithImpl;
@override @useResult
$Res call({
 String id, double lat, double lon
});




}
/// @nodoc
class __$RouteAnchorCopyWithImpl<$Res>
    implements _$RouteAnchorCopyWith<$Res> {
  __$RouteAnchorCopyWithImpl(this._self, this._then);

  final _RouteAnchor _self;
  final $Res Function(_RouteAnchor) _then;

/// Create a copy of RouteAnchor
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? lat = null,Object? lon = null,}) {
  return _then(_RouteAnchor(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,lat: null == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double,lon: null == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double,
  ));
}


}

/// @nodoc
mixin _$RouteSegment {

 String get beforeAnchorId; String get afterAnchorId; List<Geographic> get polyline; SegmentState get state;
/// Create a copy of RouteSegment
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$RouteSegmentCopyWith<RouteSegment> get copyWith => _$RouteSegmentCopyWithImpl<RouteSegment>(this as RouteSegment, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is RouteSegment&&(identical(other.beforeAnchorId, beforeAnchorId) || other.beforeAnchorId == beforeAnchorId)&&(identical(other.afterAnchorId, afterAnchorId) || other.afterAnchorId == afterAnchorId)&&const DeepCollectionEquality().equals(other.polyline, polyline)&&(identical(other.state, state) || other.state == state));
}


@override
int get hashCode => Object.hash(runtimeType,beforeAnchorId,afterAnchorId,const DeepCollectionEquality().hash(polyline),state);

@override
String toString() {
  return 'RouteSegment(beforeAnchorId: $beforeAnchorId, afterAnchorId: $afterAnchorId, polyline: $polyline, state: $state)';
}


}

/// @nodoc
abstract mixin class $RouteSegmentCopyWith<$Res>  {
  factory $RouteSegmentCopyWith(RouteSegment value, $Res Function(RouteSegment) _then) = _$RouteSegmentCopyWithImpl;
@useResult
$Res call({
 String beforeAnchorId, String afterAnchorId, List<Geographic> polyline, SegmentState state
});




}
/// @nodoc
class _$RouteSegmentCopyWithImpl<$Res>
    implements $RouteSegmentCopyWith<$Res> {
  _$RouteSegmentCopyWithImpl(this._self, this._then);

  final RouteSegment _self;
  final $Res Function(RouteSegment) _then;

/// Create a copy of RouteSegment
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? beforeAnchorId = null,Object? afterAnchorId = null,Object? polyline = null,Object? state = null,}) {
  return _then(_self.copyWith(
beforeAnchorId: null == beforeAnchorId ? _self.beforeAnchorId : beforeAnchorId // ignore: cast_nullable_to_non_nullable
as String,afterAnchorId: null == afterAnchorId ? _self.afterAnchorId : afterAnchorId // ignore: cast_nullable_to_non_nullable
as String,polyline: null == polyline ? _self.polyline : polyline // ignore: cast_nullable_to_non_nullable
as List<Geographic>,state: null == state ? _self.state : state // ignore: cast_nullable_to_non_nullable
as SegmentState,
  ));
}

}


/// Adds pattern-matching-related methods to [RouteSegment].
extension RouteSegmentPatterns on RouteSegment {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _RouteSegment value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _RouteSegment() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _RouteSegment value)  $default,){
final _that = this;
switch (_that) {
case _RouteSegment():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _RouteSegment value)?  $default,){
final _that = this;
switch (_that) {
case _RouteSegment() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String beforeAnchorId,  String afterAnchorId,  List<Geographic> polyline,  SegmentState state)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _RouteSegment() when $default != null:
return $default(_that.beforeAnchorId,_that.afterAnchorId,_that.polyline,_that.state);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String beforeAnchorId,  String afterAnchorId,  List<Geographic> polyline,  SegmentState state)  $default,) {final _that = this;
switch (_that) {
case _RouteSegment():
return $default(_that.beforeAnchorId,_that.afterAnchorId,_that.polyline,_that.state);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String beforeAnchorId,  String afterAnchorId,  List<Geographic> polyline,  SegmentState state)?  $default,) {final _that = this;
switch (_that) {
case _RouteSegment() when $default != null:
return $default(_that.beforeAnchorId,_that.afterAnchorId,_that.polyline,_that.state);case _:
  return null;

}
}

}

/// @nodoc


class _RouteSegment implements RouteSegment {
  const _RouteSegment({required this.beforeAnchorId, required this.afterAnchorId, required final  List<Geographic> polyline, required this.state}): _polyline = polyline;
  

@override final  String beforeAnchorId;
@override final  String afterAnchorId;
 final  List<Geographic> _polyline;
@override List<Geographic> get polyline {
  if (_polyline is EqualUnmodifiableListView) return _polyline;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_polyline);
}

@override final  SegmentState state;

/// Create a copy of RouteSegment
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$RouteSegmentCopyWith<_RouteSegment> get copyWith => __$RouteSegmentCopyWithImpl<_RouteSegment>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _RouteSegment&&(identical(other.beforeAnchorId, beforeAnchorId) || other.beforeAnchorId == beforeAnchorId)&&(identical(other.afterAnchorId, afterAnchorId) || other.afterAnchorId == afterAnchorId)&&const DeepCollectionEquality().equals(other._polyline, _polyline)&&(identical(other.state, state) || other.state == state));
}


@override
int get hashCode => Object.hash(runtimeType,beforeAnchorId,afterAnchorId,const DeepCollectionEquality().hash(_polyline),state);

@override
String toString() {
  return 'RouteSegment(beforeAnchorId: $beforeAnchorId, afterAnchorId: $afterAnchorId, polyline: $polyline, state: $state)';
}


}

/// @nodoc
abstract mixin class _$RouteSegmentCopyWith<$Res> implements $RouteSegmentCopyWith<$Res> {
  factory _$RouteSegmentCopyWith(_RouteSegment value, $Res Function(_RouteSegment) _then) = __$RouteSegmentCopyWithImpl;
@override @useResult
$Res call({
 String beforeAnchorId, String afterAnchorId, List<Geographic> polyline, SegmentState state
});




}
/// @nodoc
class __$RouteSegmentCopyWithImpl<$Res>
    implements _$RouteSegmentCopyWith<$Res> {
  __$RouteSegmentCopyWithImpl(this._self, this._then);

  final _RouteSegment _self;
  final $Res Function(_RouteSegment) _then;

/// Create a copy of RouteSegment
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? beforeAnchorId = null,Object? afterAnchorId = null,Object? polyline = null,Object? state = null,}) {
  return _then(_RouteSegment(
beforeAnchorId: null == beforeAnchorId ? _self.beforeAnchorId : beforeAnchorId // ignore: cast_nullable_to_non_nullable
as String,afterAnchorId: null == afterAnchorId ? _self.afterAnchorId : afterAnchorId // ignore: cast_nullable_to_non_nullable
as String,polyline: null == polyline ? _self._polyline : polyline // ignore: cast_nullable_to_non_nullable
as List<Geographic>,state: null == state ? _self.state : state // ignore: cast_nullable_to_non_nullable
as SegmentState,
  ));
}


}

/// @nodoc
mixin _$RouteAnchorsSnapshot {

 List<RouteAnchor> get anchors; List<RouteSegment> get segments;
/// Create a copy of RouteAnchorsSnapshot
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$RouteAnchorsSnapshotCopyWith<RouteAnchorsSnapshot> get copyWith => _$RouteAnchorsSnapshotCopyWithImpl<RouteAnchorsSnapshot>(this as RouteAnchorsSnapshot, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is RouteAnchorsSnapshot&&const DeepCollectionEquality().equals(other.anchors, anchors)&&const DeepCollectionEquality().equals(other.segments, segments));
}


@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(anchors),const DeepCollectionEquality().hash(segments));

@override
String toString() {
  return 'RouteAnchorsSnapshot(anchors: $anchors, segments: $segments)';
}


}

/// @nodoc
abstract mixin class $RouteAnchorsSnapshotCopyWith<$Res>  {
  factory $RouteAnchorsSnapshotCopyWith(RouteAnchorsSnapshot value, $Res Function(RouteAnchorsSnapshot) _then) = _$RouteAnchorsSnapshotCopyWithImpl;
@useResult
$Res call({
 List<RouteAnchor> anchors, List<RouteSegment> segments
});




}
/// @nodoc
class _$RouteAnchorsSnapshotCopyWithImpl<$Res>
    implements $RouteAnchorsSnapshotCopyWith<$Res> {
  _$RouteAnchorsSnapshotCopyWithImpl(this._self, this._then);

  final RouteAnchorsSnapshot _self;
  final $Res Function(RouteAnchorsSnapshot) _then;

/// Create a copy of RouteAnchorsSnapshot
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? anchors = null,Object? segments = null,}) {
  return _then(_self.copyWith(
anchors: null == anchors ? _self.anchors : anchors // ignore: cast_nullable_to_non_nullable
as List<RouteAnchor>,segments: null == segments ? _self.segments : segments // ignore: cast_nullable_to_non_nullable
as List<RouteSegment>,
  ));
}

}


/// Adds pattern-matching-related methods to [RouteAnchorsSnapshot].
extension RouteAnchorsSnapshotPatterns on RouteAnchorsSnapshot {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _RouteAnchorsSnapshot value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _RouteAnchorsSnapshot() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _RouteAnchorsSnapshot value)  $default,){
final _that = this;
switch (_that) {
case _RouteAnchorsSnapshot():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _RouteAnchorsSnapshot value)?  $default,){
final _that = this;
switch (_that) {
case _RouteAnchorsSnapshot() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<RouteAnchor> anchors,  List<RouteSegment> segments)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _RouteAnchorsSnapshot() when $default != null:
return $default(_that.anchors,_that.segments);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<RouteAnchor> anchors,  List<RouteSegment> segments)  $default,) {final _that = this;
switch (_that) {
case _RouteAnchorsSnapshot():
return $default(_that.anchors,_that.segments);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<RouteAnchor> anchors,  List<RouteSegment> segments)?  $default,) {final _that = this;
switch (_that) {
case _RouteAnchorsSnapshot() when $default != null:
return $default(_that.anchors,_that.segments);case _:
  return null;

}
}

}

/// @nodoc


class _RouteAnchorsSnapshot implements RouteAnchorsSnapshot {
  const _RouteAnchorsSnapshot({required final  List<RouteAnchor> anchors, required final  List<RouteSegment> segments}): _anchors = anchors,_segments = segments;
  

 final  List<RouteAnchor> _anchors;
@override List<RouteAnchor> get anchors {
  if (_anchors is EqualUnmodifiableListView) return _anchors;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_anchors);
}

 final  List<RouteSegment> _segments;
@override List<RouteSegment> get segments {
  if (_segments is EqualUnmodifiableListView) return _segments;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_segments);
}


/// Create a copy of RouteAnchorsSnapshot
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$RouteAnchorsSnapshotCopyWith<_RouteAnchorsSnapshot> get copyWith => __$RouteAnchorsSnapshotCopyWithImpl<_RouteAnchorsSnapshot>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _RouteAnchorsSnapshot&&const DeepCollectionEquality().equals(other._anchors, _anchors)&&const DeepCollectionEquality().equals(other._segments, _segments));
}


@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(_anchors),const DeepCollectionEquality().hash(_segments));

@override
String toString() {
  return 'RouteAnchorsSnapshot(anchors: $anchors, segments: $segments)';
}


}

/// @nodoc
abstract mixin class _$RouteAnchorsSnapshotCopyWith<$Res> implements $RouteAnchorsSnapshotCopyWith<$Res> {
  factory _$RouteAnchorsSnapshotCopyWith(_RouteAnchorsSnapshot value, $Res Function(_RouteAnchorsSnapshot) _then) = __$RouteAnchorsSnapshotCopyWithImpl;
@override @useResult
$Res call({
 List<RouteAnchor> anchors, List<RouteSegment> segments
});




}
/// @nodoc
class __$RouteAnchorsSnapshotCopyWithImpl<$Res>
    implements _$RouteAnchorsSnapshotCopyWith<$Res> {
  __$RouteAnchorsSnapshotCopyWithImpl(this._self, this._then);

  final _RouteAnchorsSnapshot _self;
  final $Res Function(_RouteAnchorsSnapshot) _then;

/// Create a copy of RouteAnchorsSnapshot
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? anchors = null,Object? segments = null,}) {
  return _then(_RouteAnchorsSnapshot(
anchors: null == anchors ? _self._anchors : anchors // ignore: cast_nullable_to_non_nullable
as List<RouteAnchor>,segments: null == segments ? _self._segments : segments // ignore: cast_nullable_to_non_nullable
as List<RouteSegment>,
  ));
}


}

// dart format on
