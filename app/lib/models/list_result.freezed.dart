// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'list_result.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$ListResult<T> {

 List<T> get items; int get page; int get perPage; int get totalPages; int get totalItems;
/// Create a copy of ListResult
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ListResultCopyWith<T, ListResult<T>> get copyWith => _$ListResultCopyWithImpl<T, ListResult<T>>(this as ListResult<T>, _$identity);

  /// Serializes this ListResult to a JSON map.
  Map<String, dynamic> toJson(Object? Function(T) toJsonT);


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ListResult<T>&&const DeepCollectionEquality().equals(other.items, items)&&(identical(other.page, page) || other.page == page)&&(identical(other.perPage, perPage) || other.perPage == perPage)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages)&&(identical(other.totalItems, totalItems) || other.totalItems == totalItems));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(items),page,perPage,totalPages,totalItems);

@override
String toString() {
  return 'ListResult<$T>(items: $items, page: $page, perPage: $perPage, totalPages: $totalPages, totalItems: $totalItems)';
}


}

/// @nodoc
abstract mixin class $ListResultCopyWith<T,$Res>  {
  factory $ListResultCopyWith(ListResult<T> value, $Res Function(ListResult<T>) _then) = _$ListResultCopyWithImpl;
@useResult
$Res call({
 List<T> items, int page, int perPage, int totalPages, int totalItems
});




}
/// @nodoc
class _$ListResultCopyWithImpl<T,$Res>
    implements $ListResultCopyWith<T, $Res> {
  _$ListResultCopyWithImpl(this._self, this._then);

  final ListResult<T> _self;
  final $Res Function(ListResult<T>) _then;

/// Create a copy of ListResult
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? items = null,Object? page = null,Object? perPage = null,Object? totalPages = null,Object? totalItems = null,}) {
  return _then(_self.copyWith(
items: null == items ? _self.items : items // ignore: cast_nullable_to_non_nullable
as List<T>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,perPage: null == perPage ? _self.perPage : perPage // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,totalItems: null == totalItems ? _self.totalItems : totalItems // ignore: cast_nullable_to_non_nullable
as int,
  ));
}

}


/// Adds pattern-matching-related methods to [ListResult].
extension ListResultPatterns<T> on ListResult<T> {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ListResult<T> value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ListResult() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ListResult<T> value)  $default,){
final _that = this;
switch (_that) {
case _ListResult():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ListResult<T> value)?  $default,){
final _that = this;
switch (_that) {
case _ListResult() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<T> items,  int page,  int perPage,  int totalPages,  int totalItems)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ListResult() when $default != null:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<T> items,  int page,  int perPage,  int totalPages,  int totalItems)  $default,) {final _that = this;
switch (_that) {
case _ListResult():
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<T> items,  int page,  int perPage,  int totalPages,  int totalItems)?  $default,) {final _that = this;
switch (_that) {
case _ListResult() when $default != null:
return $default(_that.items,_that.page,_that.perPage,_that.totalPages,_that.totalItems);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable(genericArgumentFactories: true)

class _ListResult<T> implements ListResult<T> {
  const _ListResult({required final  List<T> items, required this.page, required this.perPage, required this.totalPages, required this.totalItems}): _items = items;
  factory _ListResult.fromJson(Map<String, dynamic> json,T Function(Object?) fromJsonT) => _$ListResultFromJson(json,fromJsonT);

 final  List<T> _items;
@override List<T> get items {
  if (_items is EqualUnmodifiableListView) return _items;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_items);
}

@override final  int page;
@override final  int perPage;
@override final  int totalPages;
@override final  int totalItems;

/// Create a copy of ListResult
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ListResultCopyWith<T, _ListResult<T>> get copyWith => __$ListResultCopyWithImpl<T, _ListResult<T>>(this, _$identity);

@override
Map<String, dynamic> toJson(Object? Function(T) toJsonT) {
  return _$ListResultToJson<T>(this, toJsonT);
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ListResult<T>&&const DeepCollectionEquality().equals(other._items, _items)&&(identical(other.page, page) || other.page == page)&&(identical(other.perPage, perPage) || other.perPage == perPage)&&(identical(other.totalPages, totalPages) || other.totalPages == totalPages)&&(identical(other.totalItems, totalItems) || other.totalItems == totalItems));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(_items),page,perPage,totalPages,totalItems);

@override
String toString() {
  return 'ListResult<$T>(items: $items, page: $page, perPage: $perPage, totalPages: $totalPages, totalItems: $totalItems)';
}


}

/// @nodoc
abstract mixin class _$ListResultCopyWith<T,$Res> implements $ListResultCopyWith<T, $Res> {
  factory _$ListResultCopyWith(_ListResult<T> value, $Res Function(_ListResult<T>) _then) = __$ListResultCopyWithImpl;
@override @useResult
$Res call({
 List<T> items, int page, int perPage, int totalPages, int totalItems
});




}
/// @nodoc
class __$ListResultCopyWithImpl<T,$Res>
    implements _$ListResultCopyWith<T, $Res> {
  __$ListResultCopyWithImpl(this._self, this._then);

  final _ListResult<T> _self;
  final $Res Function(_ListResult<T>) _then;

/// Create a copy of ListResult
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? items = null,Object? page = null,Object? perPage = null,Object? totalPages = null,Object? totalItems = null,}) {
  return _then(_ListResult<T>(
items: null == items ? _self._items : items // ignore: cast_nullable_to_non_nullable
as List<T>,page: null == page ? _self.page : page // ignore: cast_nullable_to_non_nullable
as int,perPage: null == perPage ? _self.perPage : perPage // ignore: cast_nullable_to_non_nullable
as int,totalPages: null == totalPages ? _self.totalPages : totalPages // ignore: cast_nullable_to_non_nullable
as int,totalItems: null == totalItems ? _self.totalItems : totalItems // ignore: cast_nullable_to_non_nullable
as int,
  ));
}


}

// dart format on
