// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'profile_trails_provider.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$ProfileTrailsState {

 List<TrailSummary> get trails; int get page; int get perPage; int get totalPages;/// True when the last network fetch failed and this state is showing
/// only what's on this device (REC-06). Decided from the fetch outcome
/// itself, never from `onlineStatusProvider`'s optimistic default
/// (RESEARCH.md Pitfall 5).
 bool get offline;/// True when this state is for the signed-in hiker's own handle --
/// only then does the local half of the merge run (T-36-07-02).
 bool get isOwnHandle;
/// Create a copy of ProfileTrailsState
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ProfileTrailsStateCopyWith<ProfileTrailsState> get copyWith => _$ProfileTrailsStateCopyWithImpl<ProfileTrailsState>(this as ProfileTrailsState, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ProfileTrailsState&&const DeepCollectionEquality().equals(other.trails, trails)&&(identical(other.page, page) || other.page == page)&&(identical(other.perPage, perPage) || other.perPage == perPage)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages)&&(identical(other.offline, offline) || other.offline == offline)&&(identical(other.isOwnHandle, isOwnHandle) || other.isOwnHandle == isOwnHandle));
}


@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(trails),page,perPage,totalPages,offline,isOwnHandle);

@override
String toString() {
  return 'ProfileTrailsState(trails: $trails, page: $page, perPage: $perPage, totalPages: $totalPages, offline: $offline, isOwnHandle: $isOwnHandle)';
}


}

/// @nodoc
abstract mixin class $ProfileTrailsStateCopyWith<$Res>  {
  factory $ProfileTrailsStateCopyWith(ProfileTrailsState value, $Res Function(ProfileTrailsState) _then) = _$ProfileTrailsStateCopyWithImpl;
@useResult
$Res call({
 List<TrailSummary> trails, int page, int perPage, int totalPages, bool offline, bool isOwnHandle
});




}
/// @nodoc
class _$ProfileTrailsStateCopyWithImpl<$Res>
    implements $ProfileTrailsStateCopyWith<$Res> {
  _$ProfileTrailsStateCopyWithImpl(this._self, this._then);

  final ProfileTrailsState _self;
  final $Res Function(ProfileTrailsState) _then;

/// Create a copy of ProfileTrailsState
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? trails = null,Object? page = null,Object? perPage = null,Object? totalPages = null,Object? offline = null,Object? isOwnHandle = null,}) {
  return _then(_self.copyWith(
trails: null == trails ? _self.trails : trails // ignore: cast_nullable_to_non_nullable
as List<TrailSummary>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,perPage: null == perPage ? _self.perPage : perPage // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,offline: null == offline ? _self.offline : offline // ignore: cast_nullable_to_non_nullable
as bool,isOwnHandle: null == isOwnHandle ? _self.isOwnHandle : isOwnHandle // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}

}


/// Adds pattern-matching-related methods to [ProfileTrailsState].
extension ProfileTrailsStatePatterns on ProfileTrailsState {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ProfileTrailsState value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ProfileTrailsState() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ProfileTrailsState value)  $default,){
final _that = this;
switch (_that) {
case _ProfileTrailsState():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ProfileTrailsState value)?  $default,){
final _that = this;
switch (_that) {
case _ProfileTrailsState() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<TrailSummary> trails,  int page,  int perPage,  int totalPages,  bool offline,  bool isOwnHandle)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ProfileTrailsState() when $default != null:
return $default(_that.trails,_that.page,_that.perPage,_that.totalPages,_that.offline,_that.isOwnHandle);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<TrailSummary> trails,  int page,  int perPage,  int totalPages,  bool offline,  bool isOwnHandle)  $default,) {final _that = this;
switch (_that) {
case _ProfileTrailsState():
return $default(_that.trails,_that.page,_that.perPage,_that.totalPages,_that.offline,_that.isOwnHandle);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<TrailSummary> trails,  int page,  int perPage,  int totalPages,  bool offline,  bool isOwnHandle)?  $default,) {final _that = this;
switch (_that) {
case _ProfileTrailsState() when $default != null:
return $default(_that.trails,_that.page,_that.perPage,_that.totalPages,_that.offline,_that.isOwnHandle);case _:
  return null;

}
}

}

/// @nodoc


class _ProfileTrailsState extends ProfileTrailsState {
  const _ProfileTrailsState({required final  List<TrailSummary> trails, required this.page, required this.perPage, required this.totalPages, this.offline = false, this.isOwnHandle = false}): _trails = trails,super._();
  

 final  List<TrailSummary> _trails;
@override List<TrailSummary> get trails {
  if (_trails is EqualUnmodifiableListView) return _trails;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_trails);
}

@override final  int page;
@override final  int perPage;
@override final  int totalPages;
/// True when the last network fetch failed and this state is showing
/// only what's on this device (REC-06). Decided from the fetch outcome
/// itself, never from `onlineStatusProvider`'s optimistic default
/// (RESEARCH.md Pitfall 5).
@override@JsonKey() final  bool offline;
/// True when this state is for the signed-in hiker's own handle --
/// only then does the local half of the merge run (T-36-07-02).
@override@JsonKey() final  bool isOwnHandle;

/// Create a copy of ProfileTrailsState
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ProfileTrailsStateCopyWith<_ProfileTrailsState> get copyWith => __$ProfileTrailsStateCopyWithImpl<_ProfileTrailsState>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ProfileTrailsState&&const DeepCollectionEquality().equals(other._trails, _trails)&&(identical(other.page, page) || other.page == page)&&(identical(other.perPage, perPage) || other.perPage == perPage)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages)&&(identical(other.offline, offline) || other.offline == offline)&&(identical(other.isOwnHandle, isOwnHandle) || other.isOwnHandle == isOwnHandle));
}


@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(_trails),page,perPage,totalPages,offline,isOwnHandle);

@override
String toString() {
  return 'ProfileTrailsState(trails: $trails, page: $page, perPage: $perPage, totalPages: $totalPages, offline: $offline, isOwnHandle: $isOwnHandle)';
}


}

/// @nodoc
abstract mixin class _$ProfileTrailsStateCopyWith<$Res> implements $ProfileTrailsStateCopyWith<$Res> {
  factory _$ProfileTrailsStateCopyWith(_ProfileTrailsState value, $Res Function(_ProfileTrailsState) _then) = __$ProfileTrailsStateCopyWithImpl;
@override @useResult
$Res call({
 List<TrailSummary> trails, int page, int perPage, int totalPages, bool offline, bool isOwnHandle
});




}
/// @nodoc
class __$ProfileTrailsStateCopyWithImpl<$Res>
    implements _$ProfileTrailsStateCopyWith<$Res> {
  __$ProfileTrailsStateCopyWithImpl(this._self, this._then);

  final _ProfileTrailsState _self;
  final $Res Function(_ProfileTrailsState) _then;

/// Create a copy of ProfileTrailsState
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? trails = null,Object? page = null,Object? perPage = null,Object? totalPages = null,Object? offline = null,Object? isOwnHandle = null,}) {
  return _then(_ProfileTrailsState(
trails: null == trails ? _self._trails : trails // ignore: cast_nullable_to_non_nullable
as List<TrailSummary>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,perPage: null == perPage ? _self.perPage : perPage // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,offline: null == offline ? _self.offline : offline // ignore: cast_nullable_to_non_nullable
as bool,isOwnHandle: null == isOwnHandle ? _self.isOwnHandle : isOwnHandle // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}


}

// dart format on
