// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'navigate_response.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$NavigateManeuver {

 String get instruction; double get length;@JsonKey(name: 'begin_shape_index') int get beginShapeIndex; double get bearing;
/// Create a copy of NavigateManeuver
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$NavigateManeuverCopyWith<NavigateManeuver> get copyWith => _$NavigateManeuverCopyWithImpl<NavigateManeuver>(this as NavigateManeuver, _$identity);

  /// Serializes this NavigateManeuver to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is NavigateManeuver&&(identical(other.instruction, instruction) || other.instruction == instruction)&&(identical(other.length, length) || other.length == length)&&(identical(other.beginShapeIndex, beginShapeIndex) || other.beginShapeIndex == beginShapeIndex)&&(identical(other.bearing, bearing) || other.bearing == bearing));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,instruction,length,beginShapeIndex,bearing);

@override
String toString() {
  return 'NavigateManeuver(instruction: $instruction, length: $length, beginShapeIndex: $beginShapeIndex, bearing: $bearing)';
}


}

/// @nodoc
abstract mixin class $NavigateManeuverCopyWith<$Res>  {
  factory $NavigateManeuverCopyWith(NavigateManeuver value, $Res Function(NavigateManeuver) _then) = _$NavigateManeuverCopyWithImpl;
@useResult
$Res call({
 String instruction, double length,@JsonKey(name: 'begin_shape_index') int beginShapeIndex, double bearing
});




}
/// @nodoc
class _$NavigateManeuverCopyWithImpl<$Res>
    implements $NavigateManeuverCopyWith<$Res> {
  _$NavigateManeuverCopyWithImpl(this._self, this._then);

  final NavigateManeuver _self;
  final $Res Function(NavigateManeuver) _then;

/// Create a copy of NavigateManeuver
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? instruction = null,Object? length = null,Object? beginShapeIndex = null,Object? bearing = null,}) {
  return _then(_self.copyWith(
instruction: null == instruction ? _self.instruction : instruction // ignore: cast_nullable_to_non_nullable
as String,length: null == length ? _self.length : length // ignore: cast_nullable_to_non_nullable
as double,beginShapeIndex: null == beginShapeIndex ? _self.beginShapeIndex : beginShapeIndex // ignore: cast_nullable_to_non_nullable
as int,bearing: null == bearing ? _self.bearing : bearing // ignore: cast_nullable_to_non_nullable
as double,
  ));
}

}


/// Adds pattern-matching-related methods to [NavigateManeuver].
extension NavigateManeuverPatterns on NavigateManeuver {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _NavigateManeuver value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _NavigateManeuver() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _NavigateManeuver value)  $default,){
final _that = this;
switch (_that) {
case _NavigateManeuver():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _NavigateManeuver value)?  $default,){
final _that = this;
switch (_that) {
case _NavigateManeuver() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String instruction,  double length, @JsonKey(name: 'begin_shape_index')  int beginShapeIndex,  double bearing)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _NavigateManeuver() when $default != null:
return $default(_that.instruction,_that.length,_that.beginShapeIndex,_that.bearing);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String instruction,  double length, @JsonKey(name: 'begin_shape_index')  int beginShapeIndex,  double bearing)  $default,) {final _that = this;
switch (_that) {
case _NavigateManeuver():
return $default(_that.instruction,_that.length,_that.beginShapeIndex,_that.bearing);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String instruction,  double length, @JsonKey(name: 'begin_shape_index')  int beginShapeIndex,  double bearing)?  $default,) {final _that = this;
switch (_that) {
case _NavigateManeuver() when $default != null:
return $default(_that.instruction,_that.length,_that.beginShapeIndex,_that.bearing);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _NavigateManeuver implements NavigateManeuver {
  const _NavigateManeuver({required this.instruction, required this.length, @JsonKey(name: 'begin_shape_index') required this.beginShapeIndex, this.bearing = 0.0});
  factory _NavigateManeuver.fromJson(Map<String, dynamic> json) => _$NavigateManeuverFromJson(json);

@override final  String instruction;
@override final  double length;
@override@JsonKey(name: 'begin_shape_index') final  int beginShapeIndex;
@override@JsonKey() final  double bearing;

/// Create a copy of NavigateManeuver
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$NavigateManeuverCopyWith<_NavigateManeuver> get copyWith => __$NavigateManeuverCopyWithImpl<_NavigateManeuver>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$NavigateManeuverToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _NavigateManeuver&&(identical(other.instruction, instruction) || other.instruction == instruction)&&(identical(other.length, length) || other.length == length)&&(identical(other.beginShapeIndex, beginShapeIndex) || other.beginShapeIndex == beginShapeIndex)&&(identical(other.bearing, bearing) || other.bearing == bearing));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,instruction,length,beginShapeIndex,bearing);

@override
String toString() {
  return 'NavigateManeuver(instruction: $instruction, length: $length, beginShapeIndex: $beginShapeIndex, bearing: $bearing)';
}


}

/// @nodoc
abstract mixin class _$NavigateManeuverCopyWith<$Res> implements $NavigateManeuverCopyWith<$Res> {
  factory _$NavigateManeuverCopyWith(_NavigateManeuver value, $Res Function(_NavigateManeuver) _then) = __$NavigateManeuverCopyWithImpl;
@override @useResult
$Res call({
 String instruction, double length,@JsonKey(name: 'begin_shape_index') int beginShapeIndex, double bearing
});




}
/// @nodoc
class __$NavigateManeuverCopyWithImpl<$Res>
    implements _$NavigateManeuverCopyWith<$Res> {
  __$NavigateManeuverCopyWithImpl(this._self, this._then);

  final _NavigateManeuver _self;
  final $Res Function(_NavigateManeuver) _then;

/// Create a copy of NavigateManeuver
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? instruction = null,Object? length = null,Object? beginShapeIndex = null,Object? bearing = null,}) {
  return _then(_NavigateManeuver(
instruction: null == instruction ? _self.instruction : instruction // ignore: cast_nullable_to_non_nullable
as String,length: null == length ? _self.length : length // ignore: cast_nullable_to_non_nullable
as double,beginShapeIndex: null == beginShapeIndex ? _self.beginShapeIndex : beginShapeIndex // ignore: cast_nullable_to_non_nullable
as int,bearing: null == bearing ? _self.bearing : bearing // ignore: cast_nullable_to_non_nullable
as double,
  ));
}


}


/// @nodoc
mixin _$NavigateResponse {

 List<NavigateManeuver> get maneuvers; List<List<double>> get shape;
/// Create a copy of NavigateResponse
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$NavigateResponseCopyWith<NavigateResponse> get copyWith => _$NavigateResponseCopyWithImpl<NavigateResponse>(this as NavigateResponse, _$identity);

  /// Serializes this NavigateResponse to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is NavigateResponse&&const DeepCollectionEquality().equals(other.maneuvers, maneuvers)&&const DeepCollectionEquality().equals(other.shape, shape));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(maneuvers),const DeepCollectionEquality().hash(shape));

@override
String toString() {
  return 'NavigateResponse(maneuvers: $maneuvers, shape: $shape)';
}


}

/// @nodoc
abstract mixin class $NavigateResponseCopyWith<$Res>  {
  factory $NavigateResponseCopyWith(NavigateResponse value, $Res Function(NavigateResponse) _then) = _$NavigateResponseCopyWithImpl;
@useResult
$Res call({
 List<NavigateManeuver> maneuvers, List<List<double>> shape
});




}
/// @nodoc
class _$NavigateResponseCopyWithImpl<$Res>
    implements $NavigateResponseCopyWith<$Res> {
  _$NavigateResponseCopyWithImpl(this._self, this._then);

  final NavigateResponse _self;
  final $Res Function(NavigateResponse) _then;

/// Create a copy of NavigateResponse
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? maneuvers = null,Object? shape = null,}) {
  return _then(_self.copyWith(
maneuvers: null == maneuvers ? _self.maneuvers : maneuvers // ignore: cast_nullable_to_non_nullable
as List<NavigateManeuver>,shape: null == shape ? _self.shape : shape // ignore: cast_nullable_to_non_nullable
as List<List<double>>,
  ));
}

}


/// Adds pattern-matching-related methods to [NavigateResponse].
extension NavigateResponsePatterns on NavigateResponse {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _NavigateResponse value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _NavigateResponse() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _NavigateResponse value)  $default,){
final _that = this;
switch (_that) {
case _NavigateResponse():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _NavigateResponse value)?  $default,){
final _that = this;
switch (_that) {
case _NavigateResponse() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<NavigateManeuver> maneuvers,  List<List<double>> shape)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _NavigateResponse() when $default != null:
return $default(_that.maneuvers,_that.shape);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<NavigateManeuver> maneuvers,  List<List<double>> shape)  $default,) {final _that = this;
switch (_that) {
case _NavigateResponse():
return $default(_that.maneuvers,_that.shape);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<NavigateManeuver> maneuvers,  List<List<double>> shape)?  $default,) {final _that = this;
switch (_that) {
case _NavigateResponse() when $default != null:
return $default(_that.maneuvers,_that.shape);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _NavigateResponse implements NavigateResponse {
  const _NavigateResponse({required final  List<NavigateManeuver> maneuvers, required final  List<List<double>> shape}): _maneuvers = maneuvers,_shape = shape;
  factory _NavigateResponse.fromJson(Map<String, dynamic> json) => _$NavigateResponseFromJson(json);

 final  List<NavigateManeuver> _maneuvers;
@override List<NavigateManeuver> get maneuvers {
  if (_maneuvers is EqualUnmodifiableListView) return _maneuvers;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_maneuvers);
}

 final  List<List<double>> _shape;
@override List<List<double>> get shape {
  if (_shape is EqualUnmodifiableListView) return _shape;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_shape);
}


/// Create a copy of NavigateResponse
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$NavigateResponseCopyWith<_NavigateResponse> get copyWith => __$NavigateResponseCopyWithImpl<_NavigateResponse>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$NavigateResponseToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _NavigateResponse&&const DeepCollectionEquality().equals(other._maneuvers, _maneuvers)&&const DeepCollectionEquality().equals(other._shape, _shape));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(_maneuvers),const DeepCollectionEquality().hash(_shape));

@override
String toString() {
  return 'NavigateResponse(maneuvers: $maneuvers, shape: $shape)';
}


}

/// @nodoc
abstract mixin class _$NavigateResponseCopyWith<$Res> implements $NavigateResponseCopyWith<$Res> {
  factory _$NavigateResponseCopyWith(_NavigateResponse value, $Res Function(_NavigateResponse) _then) = __$NavigateResponseCopyWithImpl;
@override @useResult
$Res call({
 List<NavigateManeuver> maneuvers, List<List<double>> shape
});




}
/// @nodoc
class __$NavigateResponseCopyWithImpl<$Res>
    implements _$NavigateResponseCopyWith<$Res> {
  __$NavigateResponseCopyWithImpl(this._self, this._then);

  final _NavigateResponse _self;
  final $Res Function(_NavigateResponse) _then;

/// Create a copy of NavigateResponse
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? maneuvers = null,Object? shape = null,}) {
  return _then(_NavigateResponse(
maneuvers: null == maneuvers ? _self._maneuvers : maneuvers // ignore: cast_nullable_to_non_nullable
as List<NavigateManeuver>,shape: null == shape ? _self._shape : shape // ignore: cast_nullable_to_non_nullable
as List<List<double>>,
  ));
}


}

// dart format on
