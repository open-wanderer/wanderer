// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'trail_search_provider.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$TrailSearchState {

 List<TrailSearchResult> get trails; int get page; int get perPage; int get totalPages;
/// Create a copy of TrailSearchState
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailSearchStateCopyWith<TrailSearchState> get copyWith => _$TrailSearchStateCopyWithImpl<TrailSearchState>(this as TrailSearchState, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailSearchState&&const DeepCollectionEquality().equals(other.trails, trails)&&(identical(other.page, page) || other.page == page)&&(identical(other.perPage, perPage) || other.perPage == perPage)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages));
}


@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(trails),page,perPage,totalPages);

@override
String toString() {
  return 'TrailSearchState(trails: $trails, page: $page, perPage: $perPage, totalPages: $totalPages)';
}


}

/// @nodoc
abstract mixin class $TrailSearchStateCopyWith<$Res>  {
  factory $TrailSearchStateCopyWith(TrailSearchState value, $Res Function(TrailSearchState) _then) = _$TrailSearchStateCopyWithImpl;
@useResult
$Res call({
 List<TrailSearchResult> trails, int page, int perPage, int totalPages
});




}
/// @nodoc
class _$TrailSearchStateCopyWithImpl<$Res>
    implements $TrailSearchStateCopyWith<$Res> {
  _$TrailSearchStateCopyWithImpl(this._self, this._then);

  final TrailSearchState _self;
  final $Res Function(TrailSearchState) _then;

/// Create a copy of TrailSearchState
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? trails = null,Object? page = null,Object? perPage = null,Object? totalPages = null,}) {
  return _then(_self.copyWith(
trails: null == trails ? _self.trails : trails // ignore: cast_nullable_to_non_nullable
as List<TrailSearchResult>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,perPage: null == perPage ? _self.perPage : perPage // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,
  ));
}

}


/// Adds pattern-matching-related methods to [TrailSearchState].
extension TrailSearchStatePatterns on TrailSearchState {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailSearchState value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailSearchState() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailSearchState value)  $default,){
final _that = this;
switch (_that) {
case _TrailSearchState():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailSearchState value)?  $default,){
final _that = this;
switch (_that) {
case _TrailSearchState() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<TrailSearchResult> trails,  int page,  int perPage,  int totalPages)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailSearchState() when $default != null:
return $default(_that.trails,_that.page,_that.perPage,_that.totalPages);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<TrailSearchResult> trails,  int page,  int perPage,  int totalPages)  $default,) {final _that = this;
switch (_that) {
case _TrailSearchState():
return $default(_that.trails,_that.page,_that.perPage,_that.totalPages);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<TrailSearchResult> trails,  int page,  int perPage,  int totalPages)?  $default,) {final _that = this;
switch (_that) {
case _TrailSearchState() when $default != null:
return $default(_that.trails,_that.page,_that.perPage,_that.totalPages);case _:
  return null;

}
}

}

/// @nodoc


class _TrailSearchState extends TrailSearchState {
  const _TrailSearchState({required final  List<TrailSearchResult> trails, required this.page, required this.perPage, required this.totalPages}): _trails = trails,super._();
  

 final  List<TrailSearchResult> _trails;
@override List<TrailSearchResult> get trails {
  if (_trails is EqualUnmodifiableListView) return _trails;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_trails);
}

@override final  int page;
@override final  int perPage;
@override final  int totalPages;

/// Create a copy of TrailSearchState
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailSearchStateCopyWith<_TrailSearchState> get copyWith => __$TrailSearchStateCopyWithImpl<_TrailSearchState>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailSearchState&&const DeepCollectionEquality().equals(other._trails, _trails)&&(identical(other.page, page) || other.page == page)&&(identical(other.perPage, perPage) || other.perPage == perPage)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages));
}


@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(_trails),page,perPage,totalPages);

@override
String toString() {
  return 'TrailSearchState(trails: $trails, page: $page, perPage: $perPage, totalPages: $totalPages)';
}


}

/// @nodoc
abstract mixin class _$TrailSearchStateCopyWith<$Res> implements $TrailSearchStateCopyWith<$Res> {
  factory _$TrailSearchStateCopyWith(_TrailSearchState value, $Res Function(_TrailSearchState) _then) = __$TrailSearchStateCopyWithImpl;
@override @useResult
$Res call({
 List<TrailSearchResult> trails, int page, int perPage, int totalPages
});




}
/// @nodoc
class __$TrailSearchStateCopyWithImpl<$Res>
    implements _$TrailSearchStateCopyWith<$Res> {
  __$TrailSearchStateCopyWithImpl(this._self, this._then);

  final _TrailSearchState _self;
  final $Res Function(_TrailSearchState) _then;

/// Create a copy of TrailSearchState
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? trails = null,Object? page = null,Object? perPage = null,Object? totalPages = null,}) {
  return _then(_TrailSearchState(
trails: null == trails ? _self._trails : trails // ignore: cast_nullable_to_non_nullable
as List<TrailSearchResult>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,perPage: null == perPage ? _self.perPage : perPage // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,
  ));
}


}

// dart format on
