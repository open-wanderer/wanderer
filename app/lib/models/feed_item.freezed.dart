// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'feed_item.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$FeedItem {

 String get id; String get actor; String get type; String get created;
/// Create a copy of FeedItem
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$FeedItemCopyWith<FeedItem> get copyWith => _$FeedItemCopyWithImpl<FeedItem>(this as FeedItem, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is FeedItem&&(identical(other.id, id) || other.id == id)&&(identical(other.actor, actor) || other.actor == actor)&&(identical(other.type, type) || other.type == type)&&(identical(other.created, created) || other.created == created));
}


@override
int get hashCode => Object.hash(runtimeType,id,actor,type,created);

@override
String toString() {
  return 'FeedItem(id: $id, actor: $actor, type: $type, created: $created)';
}


}

/// @nodoc
abstract mixin class $FeedItemCopyWith<$Res>  {
  factory $FeedItemCopyWith(FeedItem value, $Res Function(FeedItem) _then) = _$FeedItemCopyWithImpl;
@useResult
$Res call({
 String id, String actor, String type, String created
});




}
/// @nodoc
class _$FeedItemCopyWithImpl<$Res>
    implements $FeedItemCopyWith<$Res> {
  _$FeedItemCopyWithImpl(this._self, this._then);

  final FeedItem _self;
  final $Res Function(FeedItem) _then;

/// Create a copy of FeedItem
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? actor = null,Object? type = null,Object? created = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as String,type: null == type ? _self.type : type // ignore: cast_nullable_to_non_nullable
as String,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

}


/// Adds pattern-matching-related methods to [FeedItem].
extension FeedItemPatterns on FeedItem {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>({TResult Function( FeedItemTrail value)?  trail,TResult Function( FeedItemList value)?  list,required TResult orElse(),}){
final _that = this;
switch (_that) {
case FeedItemTrail() when trail != null:
return trail(_that);case FeedItemList() when list != null:
return list(_that);case _:
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

@optionalTypeArgs TResult map<TResult extends Object?>({required TResult Function( FeedItemTrail value)  trail,required TResult Function( FeedItemList value)  list,}){
final _that = this;
switch (_that) {
case FeedItemTrail():
return trail(_that);case FeedItemList():
return list(_that);}
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>({TResult? Function( FeedItemTrail value)?  trail,TResult? Function( FeedItemList value)?  list,}){
final _that = this;
switch (_that) {
case FeedItemTrail() when trail != null:
return trail(_that);case FeedItemList() when list != null:
return list(_that);case _:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>({TResult Function( String id,  String actor,  String type,  String created,  TrailSearchResult trail)?  trail,TResult Function( String id,  String actor,  String type,  String created,  ListSearchResult list)?  list,required TResult orElse(),}) {final _that = this;
switch (_that) {
case FeedItemTrail() when trail != null:
return trail(_that.id,_that.actor,_that.type,_that.created,_that.trail);case FeedItemList() when list != null:
return list(_that.id,_that.actor,_that.type,_that.created,_that.list);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>({required TResult Function( String id,  String actor,  String type,  String created,  TrailSearchResult trail)  trail,required TResult Function( String id,  String actor,  String type,  String created,  ListSearchResult list)  list,}) {final _that = this;
switch (_that) {
case FeedItemTrail():
return trail(_that.id,_that.actor,_that.type,_that.created,_that.trail);case FeedItemList():
return list(_that.id,_that.actor,_that.type,_that.created,_that.list);}
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>({TResult? Function( String id,  String actor,  String type,  String created,  TrailSearchResult trail)?  trail,TResult? Function( String id,  String actor,  String type,  String created,  ListSearchResult list)?  list,}) {final _that = this;
switch (_that) {
case FeedItemTrail() when trail != null:
return trail(_that.id,_that.actor,_that.type,_that.created,_that.trail);case FeedItemList() when list != null:
return list(_that.id,_that.actor,_that.type,_that.created,_that.list);case _:
  return null;

}
}

}

/// @nodoc


class FeedItemTrail implements FeedItem {
  const FeedItemTrail({required this.id, required this.actor, required this.type, required this.created, required this.trail});
  

@override final  String id;
@override final  String actor;
@override final  String type;
@override final  String created;
 final  TrailSearchResult trail;

/// Create a copy of FeedItem
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$FeedItemTrailCopyWith<FeedItemTrail> get copyWith => _$FeedItemTrailCopyWithImpl<FeedItemTrail>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is FeedItemTrail&&(identical(other.id, id) || other.id == id)&&(identical(other.actor, actor) || other.actor == actor)&&(identical(other.type, type) || other.type == type)&&(identical(other.created, created) || other.created == created)&&(identical(other.trail, trail) || other.trail == trail));
}


@override
int get hashCode => Object.hash(runtimeType,id,actor,type,created,trail);

@override
String toString() {
  return 'FeedItem.trail(id: $id, actor: $actor, type: $type, created: $created, trail: $trail)';
}


}

/// @nodoc
abstract mixin class $FeedItemTrailCopyWith<$Res> implements $FeedItemCopyWith<$Res> {
  factory $FeedItemTrailCopyWith(FeedItemTrail value, $Res Function(FeedItemTrail) _then) = _$FeedItemTrailCopyWithImpl;
@override @useResult
$Res call({
 String id, String actor, String type, String created, TrailSearchResult trail
});


$TrailSearchResultCopyWith<$Res> get trail;

}
/// @nodoc
class _$FeedItemTrailCopyWithImpl<$Res>
    implements $FeedItemTrailCopyWith<$Res> {
  _$FeedItemTrailCopyWithImpl(this._self, this._then);

  final FeedItemTrail _self;
  final $Res Function(FeedItemTrail) _then;

/// Create a copy of FeedItem
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? actor = null,Object? type = null,Object? created = null,Object? trail = null,}) {
  return _then(FeedItemTrail(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as String,type: null == type ? _self.type : type // ignore: cast_nullable_to_non_nullable
as String,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String,trail: null == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as TrailSearchResult,
  ));
}

/// Create a copy of FeedItem
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailSearchResultCopyWith<$Res> get trail {
  
  return $TrailSearchResultCopyWith<$Res>(_self.trail, (value) {
    return _then(_self.copyWith(trail: value));
  });
}
}

/// @nodoc


class FeedItemList implements FeedItem {
  const FeedItemList({required this.id, required this.actor, required this.type, required this.created, required this.list});
  

@override final  String id;
@override final  String actor;
@override final  String type;
@override final  String created;
 final  ListSearchResult list;

/// Create a copy of FeedItem
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$FeedItemListCopyWith<FeedItemList> get copyWith => _$FeedItemListCopyWithImpl<FeedItemList>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is FeedItemList&&(identical(other.id, id) || other.id == id)&&(identical(other.actor, actor) || other.actor == actor)&&(identical(other.type, type) || other.type == type)&&(identical(other.created, created) || other.created == created)&&(identical(other.list, list) || other.list == list));
}


@override
int get hashCode => Object.hash(runtimeType,id,actor,type,created,list);

@override
String toString() {
  return 'FeedItem.list(id: $id, actor: $actor, type: $type, created: $created, list: $list)';
}


}

/// @nodoc
abstract mixin class $FeedItemListCopyWith<$Res> implements $FeedItemCopyWith<$Res> {
  factory $FeedItemListCopyWith(FeedItemList value, $Res Function(FeedItemList) _then) = _$FeedItemListCopyWithImpl;
@override @useResult
$Res call({
 String id, String actor, String type, String created, ListSearchResult list
});


$ListSearchResultCopyWith<$Res> get list;

}
/// @nodoc
class _$FeedItemListCopyWithImpl<$Res>
    implements $FeedItemListCopyWith<$Res> {
  _$FeedItemListCopyWithImpl(this._self, this._then);

  final FeedItemList _self;
  final $Res Function(FeedItemList) _then;

/// Create a copy of FeedItem
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? actor = null,Object? type = null,Object? created = null,Object? list = null,}) {
  return _then(FeedItemList(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as String,type: null == type ? _self.type : type // ignore: cast_nullable_to_non_nullable
as String,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String,list: null == list ? _self.list : list // ignore: cast_nullable_to_non_nullable
as ListSearchResult,
  ));
}

/// Create a copy of FeedItem
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ListSearchResultCopyWith<$Res> get list {
  
  return $ListSearchResultCopyWith<$Res>(_self.list, (value) {
    return _then(_self.copyWith(list: value));
  });
}
}

// dart format on
