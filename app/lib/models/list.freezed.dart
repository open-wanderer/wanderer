// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'list.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$ListExpand {

 List<Trail>? get trails; Actor? get author;
/// Create a copy of ListExpand
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ListExpandCopyWith<ListExpand> get copyWith => _$ListExpandCopyWithImpl<ListExpand>(this as ListExpand, _$identity);

  /// Serializes this ListExpand to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ListExpand&&const DeepCollectionEquality().equals(other.trails, trails)&&(identical(other.author, author) || other.author == author));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(trails),author);

@override
String toString() {
  return 'ListExpand(trails: $trails, author: $author)';
}


}

/// @nodoc
abstract mixin class $ListExpandCopyWith<$Res>  {
  factory $ListExpandCopyWith(ListExpand value, $Res Function(ListExpand) _then) = _$ListExpandCopyWithImpl;
@useResult
$Res call({
 List<Trail>? trails, Actor? author
});


$ActorCopyWith<$Res>? get author;

}
/// @nodoc
class _$ListExpandCopyWithImpl<$Res>
    implements $ListExpandCopyWith<$Res> {
  _$ListExpandCopyWithImpl(this._self, this._then);

  final ListExpand _self;
  final $Res Function(ListExpand) _then;

/// Create a copy of ListExpand
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? trails = freezed,Object? author = freezed,}) {
  return _then(_self.copyWith(
trails: freezed == trails ? _self.trails : trails // ignore: cast_nullable_to_non_nullable
as List<Trail>?,author: freezed == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as Actor?,
  ));
}
/// Create a copy of ListExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res>? get author {
    if (_self.author == null) {
    return null;
  }

  return $ActorCopyWith<$Res>(_self.author!, (value) {
    return _then(_self.copyWith(author: value));
  });
}
}


/// Adds pattern-matching-related methods to [ListExpand].
extension ListExpandPatterns on ListExpand {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ListExpand value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ListExpand() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ListExpand value)  $default,){
final _that = this;
switch (_that) {
case _ListExpand():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ListExpand value)?  $default,){
final _that = this;
switch (_that) {
case _ListExpand() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( List<Trail>? trails,  Actor? author)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ListExpand() when $default != null:
return $default(_that.trails,_that.author);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( List<Trail>? trails,  Actor? author)  $default,) {final _that = this;
switch (_that) {
case _ListExpand():
return $default(_that.trails,_that.author);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( List<Trail>? trails,  Actor? author)?  $default,) {final _that = this;
switch (_that) {
case _ListExpand() when $default != null:
return $default(_that.trails,_that.author);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _ListExpand implements ListExpand {
  const _ListExpand({final  List<Trail>? trails, this.author}): _trails = trails;
  factory _ListExpand.fromJson(Map<String, dynamic> json) => _$ListExpandFromJson(json);

 final  List<Trail>? _trails;
@override List<Trail>? get trails {
  final value = _trails;
  if (value == null) return null;
  if (_trails is EqualUnmodifiableListView) return _trails;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(value);
}

@override final  Actor? author;

/// Create a copy of ListExpand
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ListExpandCopyWith<_ListExpand> get copyWith => __$ListExpandCopyWithImpl<_ListExpand>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$ListExpandToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ListExpand&&const DeepCollectionEquality().equals(other._trails, _trails)&&(identical(other.author, author) || other.author == author));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,const DeepCollectionEquality().hash(_trails),author);

@override
String toString() {
  return 'ListExpand(trails: $trails, author: $author)';
}


}

/// @nodoc
abstract mixin class _$ListExpandCopyWith<$Res> implements $ListExpandCopyWith<$Res> {
  factory _$ListExpandCopyWith(_ListExpand value, $Res Function(_ListExpand) _then) = __$ListExpandCopyWithImpl;
@override @useResult
$Res call({
 List<Trail>? trails, Actor? author
});


@override $ActorCopyWith<$Res>? get author;

}
/// @nodoc
class __$ListExpandCopyWithImpl<$Res>
    implements _$ListExpandCopyWith<$Res> {
  __$ListExpandCopyWithImpl(this._self, this._then);

  final _ListExpand _self;
  final $Res Function(_ListExpand) _then;

/// Create a copy of ListExpand
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? trails = freezed,Object? author = freezed,}) {
  return _then(_ListExpand(
trails: freezed == trails ? _self._trails : trails // ignore: cast_nullable_to_non_nullable
as List<Trail>?,author: freezed == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as Actor?,
  ));
}

/// Create a copy of ListExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res>? get author {
    if (_self.author == null) {
    return null;
  }

  return $ActorCopyWith<$Res>(_self.author!, (value) {
    return _then(_self.copyWith(author: value));
  });
}
}


/// @nodoc
mixin _$WandererList {

 String get id; String get collectionId; String get name; bool get public; String? get description; String? get avatar; List<String> get trails; String? get iri; ListExpand? get expand; DateTime? get created; DateTime? get updated; String get author;
/// Create a copy of WandererList
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$WandererListCopyWith<WandererList> get copyWith => _$WandererListCopyWithImpl<WandererList>(this as WandererList, _$identity);

  /// Serializes this WandererList to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is WandererList&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.name, name) || other.name == name)&&(identical(other.public, public) || other.public == public)&&(identical(other.description, description) || other.description == description)&&(identical(other.avatar, avatar) || other.avatar == avatar)&&const DeepCollectionEquality().equals(other.trails, trails)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.expand, expand) || other.expand == expand)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&(identical(other.author, author) || other.author == author));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,collectionId,name,public,description,avatar,const DeepCollectionEquality().hash(trails),iri,expand,created,updated,author);

@override
String toString() {
  return 'WandererList(id: $id, collectionId: $collectionId, name: $name, public: $public, description: $description, avatar: $avatar, trails: $trails, iri: $iri, expand: $expand, created: $created, updated: $updated, author: $author)';
}


}

/// @nodoc
abstract mixin class $WandererListCopyWith<$Res>  {
  factory $WandererListCopyWith(WandererList value, $Res Function(WandererList) _then) = _$WandererListCopyWithImpl;
@useResult
$Res call({
 String id, String collectionId, String name, bool public, String? description, String? avatar, List<String> trails, String? iri, ListExpand? expand, DateTime? created, DateTime? updated, String author
});


$ListExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class _$WandererListCopyWithImpl<$Res>
    implements $WandererListCopyWith<$Res> {
  _$WandererListCopyWithImpl(this._self, this._then);

  final WandererList _self;
  final $Res Function(WandererList) _then;

/// Create a copy of WandererList
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? collectionId = null,Object? name = null,Object? public = null,Object? description = freezed,Object? avatar = freezed,Object? trails = null,Object? iri = freezed,Object? expand = freezed,Object? created = freezed,Object? updated = freezed,Object? author = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,public: null == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool,description: freezed == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String?,avatar: freezed == avatar ? _self.avatar : avatar // ignore: cast_nullable_to_non_nullable
as String?,trails: null == trails ? _self.trails : trails // ignore: cast_nullable_to_non_nullable
as List<String>,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as ListExpand?,created: freezed == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as DateTime?,updated: freezed == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as DateTime?,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,
  ));
}
/// Create a copy of WandererList
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ListExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $ListExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}


/// Adds pattern-matching-related methods to [WandererList].
extension WandererListPatterns on WandererList {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _WandererList value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _WandererList() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _WandererList value)  $default,){
final _that = this;
switch (_that) {
case _WandererList():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _WandererList value)?  $default,){
final _that = this;
switch (_that) {
case _WandererList() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String collectionId,  String name,  bool public,  String? description,  String? avatar,  List<String> trails,  String? iri,  ListExpand? expand,  DateTime? created,  DateTime? updated,  String author)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _WandererList() when $default != null:
return $default(_that.id,_that.collectionId,_that.name,_that.public,_that.description,_that.avatar,_that.trails,_that.iri,_that.expand,_that.created,_that.updated,_that.author);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String collectionId,  String name,  bool public,  String? description,  String? avatar,  List<String> trails,  String? iri,  ListExpand? expand,  DateTime? created,  DateTime? updated,  String author)  $default,) {final _that = this;
switch (_that) {
case _WandererList():
return $default(_that.id,_that.collectionId,_that.name,_that.public,_that.description,_that.avatar,_that.trails,_that.iri,_that.expand,_that.created,_that.updated,_that.author);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String collectionId,  String name,  bool public,  String? description,  String? avatar,  List<String> trails,  String? iri,  ListExpand? expand,  DateTime? created,  DateTime? updated,  String author)?  $default,) {final _that = this;
switch (_that) {
case _WandererList() when $default != null:
return $default(_that.id,_that.collectionId,_that.name,_that.public,_that.description,_that.avatar,_that.trails,_that.iri,_that.expand,_that.created,_that.updated,_that.author);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _WandererList extends WandererList {
  const _WandererList({required this.id, this.collectionId = 'lists', required this.name, this.public = false, this.description, this.avatar, final  List<String> trails = const [], this.iri, this.expand, this.created, this.updated, this.author = '000000000000000'}): _trails = trails,super._();
  factory _WandererList.fromJson(Map<String, dynamic> json) => _$WandererListFromJson(json);

@override final  String id;
@override@JsonKey() final  String collectionId;
@override final  String name;
@override@JsonKey() final  bool public;
@override final  String? description;
@override final  String? avatar;
 final  List<String> _trails;
@override@JsonKey() List<String> get trails {
  if (_trails is EqualUnmodifiableListView) return _trails;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_trails);
}

@override final  String? iri;
@override final  ListExpand? expand;
@override final  DateTime? created;
@override final  DateTime? updated;
@override@JsonKey() final  String author;

/// Create a copy of WandererList
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$WandererListCopyWith<_WandererList> get copyWith => __$WandererListCopyWithImpl<_WandererList>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$WandererListToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _WandererList&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.name, name) || other.name == name)&&(identical(other.public, public) || other.public == public)&&(identical(other.description, description) || other.description == description)&&(identical(other.avatar, avatar) || other.avatar == avatar)&&const DeepCollectionEquality().equals(other._trails, _trails)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.expand, expand) || other.expand == expand)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&(identical(other.author, author) || other.author == author));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,collectionId,name,public,description,avatar,const DeepCollectionEquality().hash(_trails),iri,expand,created,updated,author);

@override
String toString() {
  return 'WandererList(id: $id, collectionId: $collectionId, name: $name, public: $public, description: $description, avatar: $avatar, trails: $trails, iri: $iri, expand: $expand, created: $created, updated: $updated, author: $author)';
}


}

/// @nodoc
abstract mixin class _$WandererListCopyWith<$Res> implements $WandererListCopyWith<$Res> {
  factory _$WandererListCopyWith(_WandererList value, $Res Function(_WandererList) _then) = __$WandererListCopyWithImpl;
@override @useResult
$Res call({
 String id, String collectionId, String name, bool public, String? description, String? avatar, List<String> trails, String? iri, ListExpand? expand, DateTime? created, DateTime? updated, String author
});


@override $ListExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class __$WandererListCopyWithImpl<$Res>
    implements _$WandererListCopyWith<$Res> {
  __$WandererListCopyWithImpl(this._self, this._then);

  final _WandererList _self;
  final $Res Function(_WandererList) _then;

/// Create a copy of WandererList
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? collectionId = null,Object? name = null,Object? public = null,Object? description = freezed,Object? avatar = freezed,Object? trails = null,Object? iri = freezed,Object? expand = freezed,Object? created = freezed,Object? updated = freezed,Object? author = null,}) {
  return _then(_WandererList(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,public: null == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool,description: freezed == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String?,avatar: freezed == avatar ? _self.avatar : avatar // ignore: cast_nullable_to_non_nullable
as String?,trails: null == trails ? _self._trails : trails // ignore: cast_nullable_to_non_nullable
as List<String>,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as ListExpand?,created: freezed == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as DateTime?,updated: freezed == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as DateTime?,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

/// Create a copy of WandererList
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ListExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $ListExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}

/// @nodoc
mixin _$ListFilter {

 String get q; String? get author; bool? get public; bool? get shared; ListFilterSort get sort; SortOrder get sortOrder;
/// Create a copy of ListFilter
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ListFilterCopyWith<ListFilter> get copyWith => _$ListFilterCopyWithImpl<ListFilter>(this as ListFilter, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ListFilter&&(identical(other.q, q) || other.q == q)&&(identical(other.author, author) || other.author == author)&&(identical(other.public, public) || other.public == public)&&(identical(other.shared, shared) || other.shared == shared)&&(identical(other.sort, sort) || other.sort == sort)&&(identical(other.sortOrder, sortOrder) || other.sortOrder == sortOrder));
}


@override
int get hashCode => Object.hash(runtimeType,q,author,public,shared,sort,sortOrder);

@override
String toString() {
  return 'ListFilter(q: $q, author: $author, public: $public, shared: $shared, sort: $sort, sortOrder: $sortOrder)';
}


}

/// @nodoc
abstract mixin class $ListFilterCopyWith<$Res>  {
  factory $ListFilterCopyWith(ListFilter value, $Res Function(ListFilter) _then) = _$ListFilterCopyWithImpl;
@useResult
$Res call({
 String q, String? author, bool? public, bool? shared, ListFilterSort sort, SortOrder sortOrder
});




}
/// @nodoc
class _$ListFilterCopyWithImpl<$Res>
    implements $ListFilterCopyWith<$Res> {
  _$ListFilterCopyWithImpl(this._self, this._then);

  final ListFilter _self;
  final $Res Function(ListFilter) _then;

/// Create a copy of ListFilter
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? q = null,Object? author = freezed,Object? public = freezed,Object? shared = freezed,Object? sort = null,Object? sortOrder = null,}) {
  return _then(_self.copyWith(
q: null == q ? _self.q : q // ignore: cast_nullable_to_non_nullable
as String,author: freezed == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String?,public: freezed == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool?,shared: freezed == shared ? _self.shared : shared // ignore: cast_nullable_to_non_nullable
as bool?,sort: null == sort ? _self.sort : sort // ignore: cast_nullable_to_non_nullable
as ListFilterSort,sortOrder: null == sortOrder ? _self.sortOrder : sortOrder // ignore: cast_nullable_to_non_nullable
as SortOrder,
  ));
}

}


/// Adds pattern-matching-related methods to [ListFilter].
extension ListFilterPatterns on ListFilter {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ListFilter value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ListFilter() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ListFilter value)  $default,){
final _that = this;
switch (_that) {
case _ListFilter():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ListFilter value)?  $default,){
final _that = this;
switch (_that) {
case _ListFilter() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String q,  String? author,  bool? public,  bool? shared,  ListFilterSort sort,  SortOrder sortOrder)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ListFilter() when $default != null:
return $default(_that.q,_that.author,_that.public,_that.shared,_that.sort,_that.sortOrder);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String q,  String? author,  bool? public,  bool? shared,  ListFilterSort sort,  SortOrder sortOrder)  $default,) {final _that = this;
switch (_that) {
case _ListFilter():
return $default(_that.q,_that.author,_that.public,_that.shared,_that.sort,_that.sortOrder);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String q,  String? author,  bool? public,  bool? shared,  ListFilterSort sort,  SortOrder sortOrder)?  $default,) {final _that = this;
switch (_that) {
case _ListFilter() when $default != null:
return $default(_that.q,_that.author,_that.public,_that.shared,_that.sort,_that.sortOrder);case _:
  return null;

}
}

}

/// @nodoc


class _ListFilter extends ListFilter {
  const _ListFilter({required this.q, this.author, this.public, this.shared, required this.sort, required this.sortOrder}): super._();
  

@override final  String q;
@override final  String? author;
@override final  bool? public;
@override final  bool? shared;
@override final  ListFilterSort sort;
@override final  SortOrder sortOrder;

/// Create a copy of ListFilter
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ListFilterCopyWith<_ListFilter> get copyWith => __$ListFilterCopyWithImpl<_ListFilter>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ListFilter&&(identical(other.q, q) || other.q == q)&&(identical(other.author, author) || other.author == author)&&(identical(other.public, public) || other.public == public)&&(identical(other.shared, shared) || other.shared == shared)&&(identical(other.sort, sort) || other.sort == sort)&&(identical(other.sortOrder, sortOrder) || other.sortOrder == sortOrder));
}


@override
int get hashCode => Object.hash(runtimeType,q,author,public,shared,sort,sortOrder);

@override
String toString() {
  return 'ListFilter(q: $q, author: $author, public: $public, shared: $shared, sort: $sort, sortOrder: $sortOrder)';
}


}

/// @nodoc
abstract mixin class _$ListFilterCopyWith<$Res> implements $ListFilterCopyWith<$Res> {
  factory _$ListFilterCopyWith(_ListFilter value, $Res Function(_ListFilter) _then) = __$ListFilterCopyWithImpl;
@override @useResult
$Res call({
 String q, String? author, bool? public, bool? shared, ListFilterSort sort, SortOrder sortOrder
});




}
/// @nodoc
class __$ListFilterCopyWithImpl<$Res>
    implements _$ListFilterCopyWith<$Res> {
  __$ListFilterCopyWithImpl(this._self, this._then);

  final _ListFilter _self;
  final $Res Function(_ListFilter) _then;

/// Create a copy of ListFilter
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? q = null,Object? author = freezed,Object? public = freezed,Object? shared = freezed,Object? sort = null,Object? sortOrder = null,}) {
  return _then(_ListFilter(
q: null == q ? _self.q : q // ignore: cast_nullable_to_non_nullable
as String,author: freezed == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String?,public: freezed == public ? _self.public : public // ignore: cast_nullable_to_non_nullable
as bool?,shared: freezed == shared ? _self.shared : shared // ignore: cast_nullable_to_non_nullable
as bool?,sort: null == sort ? _self.sort : sort // ignore: cast_nullable_to_non_nullable
as ListFilterSort,sortOrder: null == sortOrder ? _self.sortOrder : sortOrder // ignore: cast_nullable_to_non_nullable
as SortOrder,
  ));
}


}

// dart format on
