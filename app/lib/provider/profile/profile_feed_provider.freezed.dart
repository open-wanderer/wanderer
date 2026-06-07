// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'profile_feed_provider.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$ProfileFeedState {

 List<FeedItem> get items; int get page; int get perPage; int get totalPages; int get totalItems;
/// Create a copy of ProfileFeedState
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ProfileFeedStateCopyWith<ProfileFeedState> get copyWith => _$ProfileFeedStateCopyWithImpl<ProfileFeedState>(this as ProfileFeedState, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ProfileFeedState&&const DeepCollectionEquality().equals(other.items, items)&&(identical(other.page, page) || other.page == page)&&(identical(other.perPage, perPage) || other.perPage == perPage)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages)&&(identical(other.totalItems, totalItems) || other.totalItems == totalItems));
}


@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(items),page,perPage,totalPages,totalItems);

@override
String toString() {
  return 'ProfileFeedState(items: $items, page: $page, perPage: $perPage, totalPages: $totalPages, totalItems: $totalItems)';
}


}

/// @nodoc
abstract mixin class $ProfileFeedStateCopyWith<$Res>  {
  factory $ProfileFeedStateCopyWith(ProfileFeedState value, $Res Function(ProfileFeedState) _then) = _$ProfileFeedStateCopyWithImpl;
@useResult
$Res call({
 List<FeedItem> items, int page, int perPage, int totalPages, int totalItems
});




}
/// @nodoc
class _$ProfileFeedStateCopyWithImpl<$Res>
    implements $ProfileFeedStateCopyWith<$Res> {
  _$ProfileFeedStateCopyWithImpl(this._self, this._then);

  final ProfileFeedState _self;
  final $Res Function(ProfileFeedState) _then;

/// Create a copy of ProfileFeedState
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? items = null,Object? page = null,Object? perPage = null,Object? totalPages = null,Object? totalItems = null,}) {
  return _then(_self.copyWith(
items: null == items ? _self.items : items // ignore: cast_nullable_to_non_nullable
as List<FeedItem>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,perPage: null == perPage ? _self.perPage : perPage // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,totalItems: null == totalItems ? _self.totalItems : totalItems // ignore: cast_nullable_to_non_nullable
as int,
  ));
}

}


/// Adds pattern-matching-related methods to [ProfileFeedState].
extension ProfileFeedStatePatterns on ProfileFeedState {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ProfileFeedState value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ProfileFeedState() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ProfileFeedState value)  $default,){
final _that = this;
switch (_that) {
case _ProfileFeedState():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ProfileFeedState value)?  $default,){
final _that = this;
switch (_that) {
case _ProfileFeedState() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<FeedItem> items,  int page,  int perPage,  int totalPages,  int totalItems)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ProfileFeedState() when $default != null:
return $default(_that.items,_that.page,_that.perPage,_that.totalPages,_that.totalItems);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<FeedItem> items,  int page,  int perPage,  int totalPages,  int totalItems)  $default,) {final _that = this;
switch (_that) {
case _ProfileFeedState():
return $default(_that.items,_that.page,_that.perPage,_that.totalPages,_that.totalItems);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<FeedItem> items,  int page,  int perPage,  int totalPages,  int totalItems)?  $default,) {final _that = this;
switch (_that) {
case _ProfileFeedState() when $default != null:
return $default(_that.items,_that.page,_that.perPage,_that.totalPages,_that.totalItems);case _:
  return null;

}
}

}

/// @nodoc


class _ProfileFeedState extends ProfileFeedState {
  const _ProfileFeedState({required final  List<FeedItem> items, required this.page, required this.perPage, required this.totalPages, required this.totalItems}): _items = items,super._();
  

 final  List<FeedItem> _items;
@override List<FeedItem> get items {
  if (_items is EqualUnmodifiableListView) return _items;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_items);
}

@override final  int page;
@override final  int perPage;
@override final  int totalPages;
@override final  int totalItems;

/// Create a copy of ProfileFeedState
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ProfileFeedStateCopyWith<_ProfileFeedState> get copyWith => __$ProfileFeedStateCopyWithImpl<_ProfileFeedState>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ProfileFeedState&&const DeepCollectionEquality().equals(other._items, _items)&&(identical(other.page, page) || other.page == page)&&(identical(other.perPage, perPage) || other.perPage == perPage)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages)&&(identical(other.totalItems, totalItems) || other.totalItems == totalItems));
}


@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(_items),page,perPage,totalPages,totalItems);

@override
String toString() {
  return 'ProfileFeedState(items: $items, page: $page, perPage: $perPage, totalPages: $totalPages, totalItems: $totalItems)';
}


}

/// @nodoc
abstract mixin class _$ProfileFeedStateCopyWith<$Res> implements $ProfileFeedStateCopyWith<$Res> {
  factory _$ProfileFeedStateCopyWith(_ProfileFeedState value, $Res Function(_ProfileFeedState) _then) = __$ProfileFeedStateCopyWithImpl;
@override @useResult
$Res call({
 List<FeedItem> items, int page, int perPage, int totalPages, int totalItems
});




}
/// @nodoc
class __$ProfileFeedStateCopyWithImpl<$Res>
    implements _$ProfileFeedStateCopyWith<$Res> {
  __$ProfileFeedStateCopyWithImpl(this._self, this._then);

  final _ProfileFeedState _self;
  final $Res Function(_ProfileFeedState) _then;

/// Create a copy of ProfileFeedState
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? items = null,Object? page = null,Object? perPage = null,Object? totalPages = null,Object? totalItems = null,}) {
  return _then(_ProfileFeedState(
items: null == items ? _self._items : items // ignore: cast_nullable_to_non_nullable
as List<FeedItem>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,perPage: null == perPage ? _self.perPage : perPage // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,totalItems: null == totalItems ? _self.totalItems : totalItems // ignore: cast_nullable_to_non_nullable
as int,
  ));
}


}

// dart format on
