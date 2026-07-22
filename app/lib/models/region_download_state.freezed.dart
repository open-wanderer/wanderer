// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'region_download_state.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$RegionDownloadState {

 RegionStatus get status; double? get vectorProgress; double? get demProgress;
/// Create a copy of RegionDownloadState
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$RegionDownloadStateCopyWith<RegionDownloadState> get copyWith => _$RegionDownloadStateCopyWithImpl<RegionDownloadState>(this as RegionDownloadState, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is RegionDownloadState&&(identical(other.status, status) || other.status == status)&&(identical(other.vectorProgress, vectorProgress) || other.vectorProgress == vectorProgress)&&(identical(other.demProgress, demProgress) || other.demProgress == demProgress));
}


@override
int get hashCode => Object.hash(runtimeType,status,vectorProgress,demProgress);

@override
String toString() {
  return 'RegionDownloadState(status: $status, vectorProgress: $vectorProgress, demProgress: $demProgress)';
}


}

/// @nodoc
abstract mixin class $RegionDownloadStateCopyWith<$Res>  {
  factory $RegionDownloadStateCopyWith(RegionDownloadState value, $Res Function(RegionDownloadState) _then) = _$RegionDownloadStateCopyWithImpl;
@useResult
$Res call({
 RegionStatus status, double? vectorProgress, double? demProgress
});




}
/// @nodoc
class _$RegionDownloadStateCopyWithImpl<$Res>
    implements $RegionDownloadStateCopyWith<$Res> {
  _$RegionDownloadStateCopyWithImpl(this._self, this._then);

  final RegionDownloadState _self;
  final $Res Function(RegionDownloadState) _then;

/// Create a copy of RegionDownloadState
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? status = null,Object? vectorProgress = freezed,Object? demProgress = freezed,}) {
  return _then(_self.copyWith(
status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as RegionStatus,vectorProgress: freezed == vectorProgress ? _self.vectorProgress : vectorProgress // ignore: cast_nullable_to_non_nullable
as double?,demProgress: freezed == demProgress ? _self.demProgress : demProgress // ignore: cast_nullable_to_non_nullable
as double?,
  ));
}

}


/// Adds pattern-matching-related methods to [RegionDownloadState].
extension RegionDownloadStatePatterns on RegionDownloadState {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _RegionDownloadState value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _RegionDownloadState() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _RegionDownloadState value)  $default,){
final _that = this;
switch (_that) {
case _RegionDownloadState():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _RegionDownloadState value)?  $default,){
final _that = this;
switch (_that) {
case _RegionDownloadState() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( RegionStatus status,  double? vectorProgress,  double? demProgress)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _RegionDownloadState() when $default != null:
return $default(_that.status,_that.vectorProgress,_that.demProgress);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( RegionStatus status,  double? vectorProgress,  double? demProgress)  $default,) {final _that = this;
switch (_that) {
case _RegionDownloadState():
return $default(_that.status,_that.vectorProgress,_that.demProgress);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( RegionStatus status,  double? vectorProgress,  double? demProgress)?  $default,) {final _that = this;
switch (_that) {
case _RegionDownloadState() when $default != null:
return $default(_that.status,_that.vectorProgress,_that.demProgress);case _:
  return null;

}
}

}

/// @nodoc


class _RegionDownloadState implements RegionDownloadState {
  const _RegionDownloadState({this.status = RegionStatus.notDownloaded, this.vectorProgress, this.demProgress});
  

@override@JsonKey() final  RegionStatus status;
@override final  double? vectorProgress;
@override final  double? demProgress;

/// Create a copy of RegionDownloadState
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$RegionDownloadStateCopyWith<_RegionDownloadState> get copyWith => __$RegionDownloadStateCopyWithImpl<_RegionDownloadState>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _RegionDownloadState&&(identical(other.status, status) || other.status == status)&&(identical(other.vectorProgress, vectorProgress) || other.vectorProgress == vectorProgress)&&(identical(other.demProgress, demProgress) || other.demProgress == demProgress));
}


@override
int get hashCode => Object.hash(runtimeType,status,vectorProgress,demProgress);

@override
String toString() {
  return 'RegionDownloadState(status: $status, vectorProgress: $vectorProgress, demProgress: $demProgress)';
}


}

/// @nodoc
abstract mixin class _$RegionDownloadStateCopyWith<$Res> implements $RegionDownloadStateCopyWith<$Res> {
  factory _$RegionDownloadStateCopyWith(_RegionDownloadState value, $Res Function(_RegionDownloadState) _then) = __$RegionDownloadStateCopyWithImpl;
@override @useResult
$Res call({
 RegionStatus status, double? vectorProgress, double? demProgress
});




}
/// @nodoc
class __$RegionDownloadStateCopyWithImpl<$Res>
    implements _$RegionDownloadStateCopyWith<$Res> {
  __$RegionDownloadStateCopyWithImpl(this._self, this._then);

  final _RegionDownloadState _self;
  final $Res Function(_RegionDownloadState) _then;

/// Create a copy of RegionDownloadState
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? status = null,Object? vectorProgress = freezed,Object? demProgress = freezed,}) {
  return _then(_RegionDownloadState(
status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as RegionStatus,vectorProgress: freezed == vectorProgress ? _self.vectorProgress : vectorProgress // ignore: cast_nullable_to_non_nullable
as double?,demProgress: freezed == demProgress ? _self.demProgress : demProgress // ignore: cast_nullable_to_non_nullable
as double?,
  ));
}


}

// dart format on
