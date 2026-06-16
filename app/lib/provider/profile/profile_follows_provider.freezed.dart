// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'profile_follows_provider.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$ProfileFollowsState {

 List<Actor> get items; int get page; int get totalPages;
/// Create a copy of ProfileFollowsState
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ProfileFollowsStateCopyWith<ProfileFollowsState> get copyWith => _$ProfileFollowsStateCopyWithImpl<ProfileFollowsState>(this as ProfileFollowsState, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ProfileFollowsState&&const DeepCollectionEquality().equals(other.items, items)&&(identical(other.page, page) || other.page == page)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages));
}


@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(items),page,totalPages);

@override
String toString() {
  return 'ProfileFollowsState(items: $items, page: $page, totalPages: $totalPages)';
}


}

/// @nodoc
abstract mixin class $ProfileFollowsStateCopyWith<$Res>  {
  factory $ProfileFollowsStateCopyWith(ProfileFollowsState value, $Res Function(ProfileFollowsState) _then) = _$ProfileFollowsStateCopyWithImpl;
@useResult
$Res call({
 List<Actor> items, int page, int totalPages
});




}
/// @nodoc
class _$ProfileFollowsStateCopyWithImpl<$Res>
    implements $ProfileFollowsStateCopyWith<$Res> {
  _$ProfileFollowsStateCopyWithImpl(this._self, this._then);

  final ProfileFollowsState _self;
  final $Res Function(ProfileFollowsState) _then;

/// Create a copy of ProfileFollowsState
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? items = null,Object? page = null,Object? totalPages = null,}) {
  return _then(_self.copyWith(
items: null == items ? _self.items : items // ignore: cast_nullable_to_non_nullable
as List<Actor>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,
  ));
}

}


/// Adds pattern-matching-related methods to [ProfileFollowsState].
extension ProfileFollowsStatePatterns on ProfileFollowsState {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ProfileFollowsState value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ProfileFollowsState() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ProfileFollowsState value)  $default,){
final _that = this;
switch (_that) {
case _ProfileFollowsState():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ProfileFollowsState value)?  $default,){
final _that = this;
switch (_that) {
case _ProfileFollowsState() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<Actor> items,  int page,  int totalPages)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ProfileFollowsState() when $default != null:
return $default(_that.items,_that.page,_that.totalPages);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<Actor> items,  int page,  int totalPages)  $default,) {final _that = this;
switch (_that) {
case _ProfileFollowsState():
return $default(_that.items,_that.page,_that.totalPages);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<Actor> items,  int page,  int totalPages)?  $default,) {final _that = this;
switch (_that) {
case _ProfileFollowsState() when $default != null:
return $default(_that.items,_that.page,_that.totalPages);case _:
  return null;

}
}

}

/// @nodoc


class _ProfileFollowsState extends ProfileFollowsState {
  const _ProfileFollowsState({required final  List<Actor> items, required this.page, required this.totalPages}): _items = items,super._();
  

 final  List<Actor> _items;
@override List<Actor> get items {
  if (_items is EqualUnmodifiableListView) return _items;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_items);
}

@override final  int page;
@override final  int totalPages;

/// Create a copy of ProfileFollowsState
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ProfileFollowsStateCopyWith<_ProfileFollowsState> get copyWith => __$ProfileFollowsStateCopyWithImpl<_ProfileFollowsState>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ProfileFollowsState&&const DeepCollectionEquality().equals(other._items, _items)&&(identical(other.page, page) || other.page == page)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages));
}


@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(_items),page,totalPages);

@override
String toString() {
  return 'ProfileFollowsState(items: $items, page: $page, totalPages: $totalPages)';
}


}

/// @nodoc
abstract mixin class _$ProfileFollowsStateCopyWith<$Res> implements $ProfileFollowsStateCopyWith<$Res> {
  factory _$ProfileFollowsStateCopyWith(_ProfileFollowsState value, $Res Function(_ProfileFollowsState) _then) = __$ProfileFollowsStateCopyWithImpl;
@override @useResult
$Res call({
 List<Actor> items, int page, int totalPages
});




}
/// @nodoc
class __$ProfileFollowsStateCopyWithImpl<$Res>
    implements _$ProfileFollowsStateCopyWith<$Res> {
  __$ProfileFollowsStateCopyWithImpl(this._self, this._then);

  final _ProfileFollowsState _self;
  final $Res Function(_ProfileFollowsState) _then;

/// Create a copy of ProfileFollowsState
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? items = null,Object? page = null,Object? totalPages = null,}) {
  return _then(_ProfileFollowsState(
items: null == items ? _self._items : items // ignore: cast_nullable_to_non_nullable
as List<Actor>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,
  ));
}


}

// dart format on
