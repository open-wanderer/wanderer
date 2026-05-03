// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'trail_like.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$TrailLikeExpand {

 Actor get actor; Trail get trail;
/// Create a copy of TrailLikeExpand
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailLikeExpandCopyWith<TrailLikeExpand> get copyWith => _$TrailLikeExpandCopyWithImpl<TrailLikeExpand>(this as TrailLikeExpand, _$identity);

  /// Serializes this TrailLikeExpand to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailLikeExpand&&(identical(other.actor, actor) || other.actor == actor)&&(identical(other.trail, trail) || other.trail == trail));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,actor,trail);

@override
String toString() {
  return 'TrailLikeExpand(actor: $actor, trail: $trail)';
}


}

/// @nodoc
abstract mixin class $TrailLikeExpandCopyWith<$Res>  {
  factory $TrailLikeExpandCopyWith(TrailLikeExpand value, $Res Function(TrailLikeExpand) _then) = _$TrailLikeExpandCopyWithImpl;
@useResult
$Res call({
 Actor actor, Trail trail
});


$ActorCopyWith<$Res> get actor;$TrailCopyWith<$Res> get trail;

}
/// @nodoc
class _$TrailLikeExpandCopyWithImpl<$Res>
    implements $TrailLikeExpandCopyWith<$Res> {
  _$TrailLikeExpandCopyWithImpl(this._self, this._then);

  final TrailLikeExpand _self;
  final $Res Function(TrailLikeExpand) _then;

/// Create a copy of TrailLikeExpand
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? actor = null,Object? trail = null,}) {
  return _then(_self.copyWith(
actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as Actor,trail: null == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as Trail,
  ));
}
/// Create a copy of TrailLikeExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res> get actor {
  
  return $ActorCopyWith<$Res>(_self.actor, (value) {
    return _then(_self.copyWith(actor: value));
  });
}/// Create a copy of TrailLikeExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailCopyWith<$Res> get trail {
  
  return $TrailCopyWith<$Res>(_self.trail, (value) {
    return _then(_self.copyWith(trail: value));
  });
}
}


/// Adds pattern-matching-related methods to [TrailLikeExpand].
extension TrailLikeExpandPatterns on TrailLikeExpand {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailLikeExpand value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailLikeExpand() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailLikeExpand value)  $default,){
final _that = this;
switch (_that) {
case _TrailLikeExpand():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailLikeExpand value)?  $default,){
final _that = this;
switch (_that) {
case _TrailLikeExpand() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( Actor actor,  Trail trail)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailLikeExpand() when $default != null:
return $default(_that.actor,_that.trail);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( Actor actor,  Trail trail)  $default,) {final _that = this;
switch (_that) {
case _TrailLikeExpand():
return $default(_that.actor,_that.trail);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( Actor actor,  Trail trail)?  $default,) {final _that = this;
switch (_that) {
case _TrailLikeExpand() when $default != null:
return $default(_that.actor,_that.trail);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _TrailLikeExpand implements TrailLikeExpand {
  const _TrailLikeExpand({required this.actor, required this.trail});
  factory _TrailLikeExpand.fromJson(Map<String, dynamic> json) => _$TrailLikeExpandFromJson(json);

@override final  Actor actor;
@override final  Trail trail;

/// Create a copy of TrailLikeExpand
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailLikeExpandCopyWith<_TrailLikeExpand> get copyWith => __$TrailLikeExpandCopyWithImpl<_TrailLikeExpand>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$TrailLikeExpandToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailLikeExpand&&(identical(other.actor, actor) || other.actor == actor)&&(identical(other.trail, trail) || other.trail == trail));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,actor,trail);

@override
String toString() {
  return 'TrailLikeExpand(actor: $actor, trail: $trail)';
}


}

/// @nodoc
abstract mixin class _$TrailLikeExpandCopyWith<$Res> implements $TrailLikeExpandCopyWith<$Res> {
  factory _$TrailLikeExpandCopyWith(_TrailLikeExpand value, $Res Function(_TrailLikeExpand) _then) = __$TrailLikeExpandCopyWithImpl;
@override @useResult
$Res call({
 Actor actor, Trail trail
});


@override $ActorCopyWith<$Res> get actor;@override $TrailCopyWith<$Res> get trail;

}
/// @nodoc
class __$TrailLikeExpandCopyWithImpl<$Res>
    implements _$TrailLikeExpandCopyWith<$Res> {
  __$TrailLikeExpandCopyWithImpl(this._self, this._then);

  final _TrailLikeExpand _self;
  final $Res Function(_TrailLikeExpand) _then;

/// Create a copy of TrailLikeExpand
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? actor = null,Object? trail = null,}) {
  return _then(_TrailLikeExpand(
actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as Actor,trail: null == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as Trail,
  ));
}

/// Create a copy of TrailLikeExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res> get actor {
  
  return $ActorCopyWith<$Res>(_self.actor, (value) {
    return _then(_self.copyWith(actor: value));
  });
}/// Create a copy of TrailLikeExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailCopyWith<$Res> get trail {
  
  return $TrailCopyWith<$Res>(_self.trail, (value) {
    return _then(_self.copyWith(trail: value));
  });
}
}


/// @nodoc
mixin _$TrailLike {

 String? get id; String get actor; String get trail; TrailLikeExpand? get expand;
/// Create a copy of TrailLike
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailLikeCopyWith<TrailLike> get copyWith => _$TrailLikeCopyWithImpl<TrailLike>(this as TrailLike, _$identity);

  /// Serializes this TrailLike to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailLike&&(identical(other.id, id) || other.id == id)&&(identical(other.actor, actor) || other.actor == actor)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.expand, expand) || other.expand == expand));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,actor,trail,expand);

@override
String toString() {
  return 'TrailLike(id: $id, actor: $actor, trail: $trail, expand: $expand)';
}


}

/// @nodoc
abstract mixin class $TrailLikeCopyWith<$Res>  {
  factory $TrailLikeCopyWith(TrailLike value, $Res Function(TrailLike) _then) = _$TrailLikeCopyWithImpl;
@useResult
$Res call({
 String? id, String actor, String trail, TrailLikeExpand? expand
});


$TrailLikeExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class _$TrailLikeCopyWithImpl<$Res>
    implements $TrailLikeCopyWith<$Res> {
  _$TrailLikeCopyWithImpl(this._self, this._then);

  final TrailLike _self;
  final $Res Function(TrailLike) _then;

/// Create a copy of TrailLike
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = freezed,Object? actor = null,Object? trail = null,Object? expand = freezed,}) {
  return _then(_self.copyWith(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as String,trail: null == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as String,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as TrailLikeExpand?,
  ));
}
/// Create a copy of TrailLike
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailLikeExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $TrailLikeExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}


/// Adds pattern-matching-related methods to [TrailLike].
extension TrailLikePatterns on TrailLike {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailLike value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailLike() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailLike value)  $default,){
final _that = this;
switch (_that) {
case _TrailLike():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailLike value)?  $default,){
final _that = this;
switch (_that) {
case _TrailLike() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String? id,  String actor,  String trail,  TrailLikeExpand? expand)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailLike() when $default != null:
return $default(_that.id,_that.actor,_that.trail,_that.expand);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String? id,  String actor,  String trail,  TrailLikeExpand? expand)  $default,) {final _that = this;
switch (_that) {
case _TrailLike():
return $default(_that.id,_that.actor,_that.trail,_that.expand);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String? id,  String actor,  String trail,  TrailLikeExpand? expand)?  $default,) {final _that = this;
switch (_that) {
case _TrailLike() when $default != null:
return $default(_that.id,_that.actor,_that.trail,_that.expand);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _TrailLike implements TrailLike {
  const _TrailLike({this.id, required this.actor, required this.trail, this.expand});
  factory _TrailLike.fromJson(Map<String, dynamic> json) => _$TrailLikeFromJson(json);

@override final  String? id;
@override final  String actor;
@override final  String trail;
@override final  TrailLikeExpand? expand;

/// Create a copy of TrailLike
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailLikeCopyWith<_TrailLike> get copyWith => __$TrailLikeCopyWithImpl<_TrailLike>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$TrailLikeToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailLike&&(identical(other.id, id) || other.id == id)&&(identical(other.actor, actor) || other.actor == actor)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.expand, expand) || other.expand == expand));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,actor,trail,expand);

@override
String toString() {
  return 'TrailLike(id: $id, actor: $actor, trail: $trail, expand: $expand)';
}


}

/// @nodoc
abstract mixin class _$TrailLikeCopyWith<$Res> implements $TrailLikeCopyWith<$Res> {
  factory _$TrailLikeCopyWith(_TrailLike value, $Res Function(_TrailLike) _then) = __$TrailLikeCopyWithImpl;
@override @useResult
$Res call({
 String? id, String actor, String trail, TrailLikeExpand? expand
});


@override $TrailLikeExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class __$TrailLikeCopyWithImpl<$Res>
    implements _$TrailLikeCopyWith<$Res> {
  __$TrailLikeCopyWithImpl(this._self, this._then);

  final _TrailLike _self;
  final $Res Function(_TrailLike) _then;

/// Create a copy of TrailLike
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = freezed,Object? actor = null,Object? trail = null,Object? expand = freezed,}) {
  return _then(_TrailLike(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as String,trail: null == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as String,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as TrailLikeExpand?,
  ));
}

/// Create a copy of TrailLike
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailLikeExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $TrailLikeExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}

// dart format on
