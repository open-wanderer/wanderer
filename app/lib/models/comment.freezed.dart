// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'comment.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$CommentExpand {

 Actor get author;
/// Create a copy of CommentExpand
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$CommentExpandCopyWith<CommentExpand> get copyWith => _$CommentExpandCopyWithImpl<CommentExpand>(this as CommentExpand, _$identity);

  /// Serializes this CommentExpand to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is CommentExpand&&(identical(other.author, author) || other.author == author));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,author);

@override
String toString() {
  return 'CommentExpand(author: $author)';
}


}

/// @nodoc
abstract mixin class $CommentExpandCopyWith<$Res>  {
  factory $CommentExpandCopyWith(CommentExpand value, $Res Function(CommentExpand) _then) = _$CommentExpandCopyWithImpl;
@useResult
$Res call({
 Actor author
});


$ActorCopyWith<$Res> get author;

}
/// @nodoc
class _$CommentExpandCopyWithImpl<$Res>
    implements $CommentExpandCopyWith<$Res> {
  _$CommentExpandCopyWithImpl(this._self, this._then);

  final CommentExpand _self;
  final $Res Function(CommentExpand) _then;

/// Create a copy of CommentExpand
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? author = null,}) {
  return _then(_self.copyWith(
author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as Actor,
  ));
}
/// Create a copy of CommentExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res> get author {
  
  return $ActorCopyWith<$Res>(_self.author, (value) {
    return _then(_self.copyWith(author: value));
  });
}
}


/// Adds pattern-matching-related methods to [CommentExpand].
extension CommentExpandPatterns on CommentExpand {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _CommentExpand value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _CommentExpand() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _CommentExpand value)  $default,){
final _that = this;
switch (_that) {
case _CommentExpand():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _CommentExpand value)?  $default,){
final _that = this;
switch (_that) {
case _CommentExpand() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( Actor author)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _CommentExpand() when $default != null:
return $default(_that.author);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( Actor author)  $default,) {final _that = this;
switch (_that) {
case _CommentExpand():
return $default(_that.author);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( Actor author)?  $default,) {final _that = this;
switch (_that) {
case _CommentExpand() when $default != null:
return $default(_that.author);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _CommentExpand implements CommentExpand {
  const _CommentExpand({required this.author});
  factory _CommentExpand.fromJson(Map<String, dynamic> json) => _$CommentExpandFromJson(json);

@override final  Actor author;

/// Create a copy of CommentExpand
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$CommentExpandCopyWith<_CommentExpand> get copyWith => __$CommentExpandCopyWithImpl<_CommentExpand>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$CommentExpandToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _CommentExpand&&(identical(other.author, author) || other.author == author));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,author);

@override
String toString() {
  return 'CommentExpand(author: $author)';
}


}

/// @nodoc
abstract mixin class _$CommentExpandCopyWith<$Res> implements $CommentExpandCopyWith<$Res> {
  factory _$CommentExpandCopyWith(_CommentExpand value, $Res Function(_CommentExpand) _then) = __$CommentExpandCopyWithImpl;
@override @useResult
$Res call({
 Actor author
});


@override $ActorCopyWith<$Res> get author;

}
/// @nodoc
class __$CommentExpandCopyWithImpl<$Res>
    implements _$CommentExpandCopyWith<$Res> {
  __$CommentExpandCopyWithImpl(this._self, this._then);

  final _CommentExpand _self;
  final $Res Function(_CommentExpand) _then;

/// Create a copy of CommentExpand
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? author = null,}) {
  return _then(_CommentExpand(
author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as Actor,
  ));
}

/// Create a copy of CommentExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res> get author {
  
  return $ActorCopyWith<$Res>(_self.author, (value) {
    return _then(_self.copyWith(author: value));
  });
}
}


/// @nodoc
mixin _$Comment {

 String? get id; String get text; String get author; String get trail; String? get created; String? get updated; String? get iri; CommentExpand? get expand;
/// Create a copy of Comment
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$CommentCopyWith<Comment> get copyWith => _$CommentCopyWithImpl<Comment>(this as Comment, _$identity);

  /// Serializes this Comment to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Comment&&(identical(other.id, id) || other.id == id)&&(identical(other.text, text) || other.text == text)&&(identical(other.author, author) || other.author == author)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.expand, expand) || other.expand == expand));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,text,author,trail,created,updated,iri,expand);

@override
String toString() {
  return 'Comment(id: $id, text: $text, author: $author, trail: $trail, created: $created, updated: $updated, iri: $iri, expand: $expand)';
}


}

/// @nodoc
abstract mixin class $CommentCopyWith<$Res>  {
  factory $CommentCopyWith(Comment value, $Res Function(Comment) _then) = _$CommentCopyWithImpl;
@useResult
$Res call({
 String? id, String text, String author, String trail, String? created, String? updated, String? iri, CommentExpand? expand
});


$CommentExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class _$CommentCopyWithImpl<$Res>
    implements $CommentCopyWith<$Res> {
  _$CommentCopyWithImpl(this._self, this._then);

  final Comment _self;
  final $Res Function(Comment) _then;

/// Create a copy of Comment
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = freezed,Object? text = null,Object? author = null,Object? trail = null,Object? created = freezed,Object? updated = freezed,Object? iri = freezed,Object? expand = freezed,}) {
  return _then(_self.copyWith(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,text: null == text ? _self.text : text // ignore: cast_nullable_to_non_nullable
as String,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,trail: null == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as String,created: freezed == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String?,updated: freezed == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as String?,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as CommentExpand?,
  ));
}
/// Create a copy of Comment
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$CommentExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $CommentExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}


/// Adds pattern-matching-related methods to [Comment].
extension CommentPatterns on Comment {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Comment value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Comment() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Comment value)  $default,){
final _that = this;
switch (_that) {
case _Comment():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Comment value)?  $default,){
final _that = this;
switch (_that) {
case _Comment() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String? id,  String text,  String author,  String trail,  String? created,  String? updated,  String? iri,  CommentExpand? expand)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Comment() when $default != null:
return $default(_that.id,_that.text,_that.author,_that.trail,_that.created,_that.updated,_that.iri,_that.expand);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String? id,  String text,  String author,  String trail,  String? created,  String? updated,  String? iri,  CommentExpand? expand)  $default,) {final _that = this;
switch (_that) {
case _Comment():
return $default(_that.id,_that.text,_that.author,_that.trail,_that.created,_that.updated,_that.iri,_that.expand);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String? id,  String text,  String author,  String trail,  String? created,  String? updated,  String? iri,  CommentExpand? expand)?  $default,) {final _that = this;
switch (_that) {
case _Comment() when $default != null:
return $default(_that.id,_that.text,_that.author,_that.trail,_that.created,_that.updated,_that.iri,_that.expand);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Comment implements Comment {
  const _Comment({this.id, required this.text, required this.author, required this.trail, this.created, this.updated, this.iri, this.expand});
  factory _Comment.fromJson(Map<String, dynamic> json) => _$CommentFromJson(json);

@override final  String? id;
@override final  String text;
@override final  String author;
@override final  String trail;
@override final  String? created;
@override final  String? updated;
@override final  String? iri;
@override final  CommentExpand? expand;

/// Create a copy of Comment
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$CommentCopyWith<_Comment> get copyWith => __$CommentCopyWithImpl<_Comment>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$CommentToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _Comment&&(identical(other.id, id) || other.id == id)&&(identical(other.text, text) || other.text == text)&&(identical(other.author, author) || other.author == author)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.expand, expand) || other.expand == expand));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,text,author,trail,created,updated,iri,expand);

@override
String toString() {
  return 'Comment(id: $id, text: $text, author: $author, trail: $trail, created: $created, updated: $updated, iri: $iri, expand: $expand)';
}


}

/// @nodoc
abstract mixin class _$CommentCopyWith<$Res> implements $CommentCopyWith<$Res> {
  factory _$CommentCopyWith(_Comment value, $Res Function(_Comment) _then) = __$CommentCopyWithImpl;
@override @useResult
$Res call({
 String? id, String text, String author, String trail, String? created, String? updated, String? iri, CommentExpand? expand
});


@override $CommentExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class __$CommentCopyWithImpl<$Res>
    implements _$CommentCopyWith<$Res> {
  __$CommentCopyWithImpl(this._self, this._then);

  final _Comment _self;
  final $Res Function(_Comment) _then;

/// Create a copy of Comment
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = freezed,Object? text = null,Object? author = null,Object? trail = null,Object? created = freezed,Object? updated = freezed,Object? iri = freezed,Object? expand = freezed,}) {
  return _then(_Comment(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,text: null == text ? _self.text : text // ignore: cast_nullable_to_non_nullable
as String,author: null == author ? _self.author : author // ignore: cast_nullable_to_non_nullable
as String,trail: null == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as String,created: freezed == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String?,updated: freezed == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as String?,iri: freezed == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String?,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as CommentExpand?,
  ));
}

/// Create a copy of Comment
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$CommentExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $CommentExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}

// dart format on
