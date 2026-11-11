// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint, type=warning, deprecated_member_use, deprecated_member_use_from_same_package
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'profile_lists_provider.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$ProfileListsState {

 List<ListSearchResult> get lists; int get page; int get perPage; int get totalPages;
/// Create a copy of ProfileListsState
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ProfileListsStateCopyWith<ProfileListsState> get copyWith => _$ProfileListsStateCopyWithImpl<ProfileListsState>(this as ProfileListsState, _$identity);



@override
bool operator ==(Object other) {
  final _this = this as ProfileListsState;
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ProfileListsState&&const DeepCollectionEquality().equals(other.lists, _this.lists)&&(identical(other.page, _this.page) || other.page == _this.page)&&(identical(other.perPage, _this.perPage) || other.perPage == _this.perPage)&&(identical(other.totalPages, _this.totalPages) || other.totalPages == _this.totalPages));
}


@override
int get hashCode {
  final _this = this as ProfileListsState;
  return Object.hash(runtimeType,const DeepCollectionEquality().hash(_this.lists),_this.page,_this.perPage,_this.totalPages);
}

@override
String toString() {
  final _this = this as ProfileListsState;
  return 'ProfileListsState(lists: ${_this.lists}, page: ${_this.page}, perPage: ${_this.perPage}, totalPages: ${_this.totalPages})';
}


}

/// @nodoc
abstract mixin class $ProfileListsStateCopyWith<$Res>  {
  factory $ProfileListsStateCopyWith(ProfileListsState value, $Res Function(ProfileListsState) _then) = _$ProfileListsStateCopyWithImpl;
@useResult
$Res call({
 List<ListSearchResult> lists, int page, int perPage, int totalPages
});




}
/// @nodoc
class _$ProfileListsStateCopyWithImpl<$Res>
    implements $ProfileListsStateCopyWith<$Res> {
  _$ProfileListsStateCopyWithImpl(this._self, this._then);

  final ProfileListsState _self;
  final $Res Function(ProfileListsState) _then;

/// Create a copy of ProfileListsState
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? lists = null,Object? page = null,Object? perPage = null,Object? totalPages = null,}) {
  return _then(ProfileListsState(
lists: null == lists ? _self.lists : lists // ignore: cast_nullable_to_non_nullable
as List<ListSearchResult>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,perPage: null == perPage ? _self.perPage : perPage // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,
  ));
}

}


/// Adds pattern-matching-related methods to [ProfileListsState].
extension ProfileListsStatePatterns on ProfileListsState {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ProfileListsState value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ProfileListsState() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ProfileListsState value)  $default,){
final _that = this;
switch (_that) {
case _ProfileListsState():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ProfileListsState value)?  $default,){
final _that = this;
switch (_that) {
case _ProfileListsState() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<ListSearchResult> lists,  int page,  int perPage,  int totalPages)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ProfileListsState() when $default != null:
return $default(_that.lists,_that.page,_that.perPage,_that.totalPages);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<ListSearchResult> lists,  int page,  int perPage,  int totalPages)  $default,) {final _that = this;
switch (_that) {
case _ProfileListsState():
return $default(_that.lists,_that.page,_that.perPage,_that.totalPages);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<ListSearchResult> lists,  int page,  int perPage,  int totalPages)?  $default,) {final _that = this;
switch (_that) {
case _ProfileListsState() when $default != null:
return $default(_that.lists,_that.page,_that.perPage,_that.totalPages);case _:
  return null;

}
}

}

/// @nodoc


class _ProfileListsState extends ProfileListsState {
  const _ProfileListsState({required  List<ListSearchResult> lists, required this.page, required this.perPage, required this.totalPages}): _lists = lists,super._();
  

 final  List<ListSearchResult> _lists;
@override List<ListSearchResult> get lists {
  if (_lists is EqualUnmodifiableListView) return _lists;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_lists);
}

@override final  int page;
@override final  int perPage;
@override final  int totalPages;

/// Create a copy of ProfileListsState
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ProfileListsStateCopyWith<_ProfileListsState> get copyWith => __$ProfileListsStateCopyWithImpl<_ProfileListsState>(this, _$identity);



@override
bool operator ==(Object other) {
    return identical(this, other) || (other.runtimeType == runtimeType&&other is _ProfileListsState&&const DeepCollectionEquality().equals(other.lists, _lists)&&(identical(other.page, page) || other.page == page)&&(identical(other.perPage, perPage) || other.perPage == perPage)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages));
}


@override
int get hashCode {
    return Object.hash(runtimeType,const DeepCollectionEquality().hash(_lists),page,perPage,totalPages);
}

@override
String toString() {
    return 'ProfileListsState(lists: $lists, page: $page, perPage: $perPage, totalPages: $totalPages)';
}


}

/// @nodoc
abstract mixin class _$ProfileListsStateCopyWith<$Res> implements $ProfileListsStateCopyWith<$Res> {
  factory _$ProfileListsStateCopyWith(_ProfileListsState value, $Res Function(_ProfileListsState) _then) = __$ProfileListsStateCopyWithImpl;
@override @useResult
$Res call({
 List<ListSearchResult> lists, int page, int perPage, int totalPages
});




}
/// @nodoc
class __$ProfileListsStateCopyWithImpl<$Res>
    implements _$ProfileListsStateCopyWith<$Res> {
  __$ProfileListsStateCopyWithImpl(this._self, this._then);

  final _ProfileListsState _self;
  final $Res Function(_ProfileListsState) _then;

/// Create a copy of ProfileListsState
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? lists = null,Object? page = null,Object? perPage = null,Object? totalPages = null,}) {
  return _then(_ProfileListsState(
lists: null == lists ? _self._lists : lists // ignore: cast_nullable_to_non_nullable
as List<ListSearchResult>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,perPage: null == perPage ? _self.perPage : perPage // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,
  ));
}


}

// dart format on
