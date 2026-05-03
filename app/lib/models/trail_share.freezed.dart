// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'trail_share.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$TrailShareExpand {

 Actor get actor;
/// Create a copy of TrailShareExpand
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailShareExpandCopyWith<TrailShareExpand> get copyWith => _$TrailShareExpandCopyWithImpl<TrailShareExpand>(this as TrailShareExpand, _$identity);

  /// Serializes this TrailShareExpand to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailShareExpand&&(identical(other.actor, actor) || other.actor == actor));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,actor);

@override
String toString() {
  return 'TrailShareExpand(actor: $actor)';
}


}

/// @nodoc
abstract mixin class $TrailShareExpandCopyWith<$Res>  {
  factory $TrailShareExpandCopyWith(TrailShareExpand value, $Res Function(TrailShareExpand) _then) = _$TrailShareExpandCopyWithImpl;
@useResult
$Res call({
 Actor actor
});


$ActorCopyWith<$Res> get actor;

}
/// @nodoc
class _$TrailShareExpandCopyWithImpl<$Res>
    implements $TrailShareExpandCopyWith<$Res> {
  _$TrailShareExpandCopyWithImpl(this._self, this._then);

  final TrailShareExpand _self;
  final $Res Function(TrailShareExpand) _then;

/// Create a copy of TrailShareExpand
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? actor = null,}) {
  return _then(_self.copyWith(
actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as Actor,
  ));
}
/// Create a copy of TrailShareExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res> get actor {
  
  return $ActorCopyWith<$Res>(_self.actor, (value) {
    return _then(_self.copyWith(actor: value));
  });
}
}


/// Adds pattern-matching-related methods to [TrailShareExpand].
extension TrailShareExpandPatterns on TrailShareExpand {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailShareExpand value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailShareExpand() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailShareExpand value)  $default,){
final _that = this;
switch (_that) {
case _TrailShareExpand():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailShareExpand value)?  $default,){
final _that = this;
switch (_that) {
case _TrailShareExpand() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( Actor actor)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailShareExpand() when $default != null:
return $default(_that.actor);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( Actor actor)  $default,) {final _that = this;
switch (_that) {
case _TrailShareExpand():
return $default(_that.actor);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( Actor actor)?  $default,) {final _that = this;
switch (_that) {
case _TrailShareExpand() when $default != null:
return $default(_that.actor);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _TrailShareExpand implements TrailShareExpand {
  const _TrailShareExpand({required this.actor});
  factory _TrailShareExpand.fromJson(Map<String, dynamic> json) => _$TrailShareExpandFromJson(json);

@override final  Actor actor;

/// Create a copy of TrailShareExpand
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailShareExpandCopyWith<_TrailShareExpand> get copyWith => __$TrailShareExpandCopyWithImpl<_TrailShareExpand>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$TrailShareExpandToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailShareExpand&&(identical(other.actor, actor) || other.actor == actor));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,actor);

@override
String toString() {
  return 'TrailShareExpand(actor: $actor)';
}


}

/// @nodoc
abstract mixin class _$TrailShareExpandCopyWith<$Res> implements $TrailShareExpandCopyWith<$Res> {
  factory _$TrailShareExpandCopyWith(_TrailShareExpand value, $Res Function(_TrailShareExpand) _then) = __$TrailShareExpandCopyWithImpl;
@override @useResult
$Res call({
 Actor actor
});


@override $ActorCopyWith<$Res> get actor;

}
/// @nodoc
class __$TrailShareExpandCopyWithImpl<$Res>
    implements _$TrailShareExpandCopyWith<$Res> {
  __$TrailShareExpandCopyWithImpl(this._self, this._then);

  final _TrailShareExpand _self;
  final $Res Function(_TrailShareExpand) _then;

/// Create a copy of TrailShareExpand
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? actor = null,}) {
  return _then(_TrailShareExpand(
actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as Actor,
  ));
}

/// Create a copy of TrailShareExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res> get actor {
  
  return $ActorCopyWith<$Res>(_self.actor, (value) {
    return _then(_self.copyWith(actor: value));
  });
}
}


/// @nodoc
mixin _$TrailShare {

 String? get id; String get actor; String get trail; TrailPermission get permission; TrailShareExpand? get expand;
/// Create a copy of TrailShare
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$TrailShareCopyWith<TrailShare> get copyWith => _$TrailShareCopyWithImpl<TrailShare>(this as TrailShare, _$identity);

  /// Serializes this TrailShare to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is TrailShare&&(identical(other.id, id) || other.id == id)&&(identical(other.actor, actor) || other.actor == actor)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.permission, permission) || other.permission == permission)&&(identical(other.expand, expand) || other.expand == expand));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,actor,trail,permission,expand);

@override
String toString() {
  return 'TrailShare(id: $id, actor: $actor, trail: $trail, permission: $permission, expand: $expand)';
}


}

/// @nodoc
abstract mixin class $TrailShareCopyWith<$Res>  {
  factory $TrailShareCopyWith(TrailShare value, $Res Function(TrailShare) _then) = _$TrailShareCopyWithImpl;
@useResult
$Res call({
 String? id, String actor, String trail, TrailPermission permission, TrailShareExpand? expand
});


$TrailShareExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class _$TrailShareCopyWithImpl<$Res>
    implements $TrailShareCopyWith<$Res> {
  _$TrailShareCopyWithImpl(this._self, this._then);

  final TrailShare _self;
  final $Res Function(TrailShare) _then;

/// Create a copy of TrailShare
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = freezed,Object? actor = null,Object? trail = null,Object? permission = null,Object? expand = freezed,}) {
  return _then(_self.copyWith(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as String,trail: null == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as String,permission: null == permission ? _self.permission : permission // ignore: cast_nullable_to_non_nullable
as TrailPermission,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as TrailShareExpand?,
  ));
}
/// Create a copy of TrailShare
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailShareExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $TrailShareExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}


/// Adds pattern-matching-related methods to [TrailShare].
extension TrailSharePatterns on TrailShare {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _TrailShare value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _TrailShare() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _TrailShare value)  $default,){
final _that = this;
switch (_that) {
case _TrailShare():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _TrailShare value)?  $default,){
final _that = this;
switch (_that) {
case _TrailShare() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String? id,  String actor,  String trail,  TrailPermission permission,  TrailShareExpand? expand)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _TrailShare() when $default != null:
return $default(_that.id,_that.actor,_that.trail,_that.permission,_that.expand);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String? id,  String actor,  String trail,  TrailPermission permission,  TrailShareExpand? expand)  $default,) {final _that = this;
switch (_that) {
case _TrailShare():
return $default(_that.id,_that.actor,_that.trail,_that.permission,_that.expand);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String? id,  String actor,  String trail,  TrailPermission permission,  TrailShareExpand? expand)?  $default,) {final _that = this;
switch (_that) {
case _TrailShare() when $default != null:
return $default(_that.id,_that.actor,_that.trail,_that.permission,_that.expand);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _TrailShare implements TrailShare {
  const _TrailShare({this.id, required this.actor, required this.trail, required this.permission, this.expand});
  factory _TrailShare.fromJson(Map<String, dynamic> json) => _$TrailShareFromJson(json);

@override final  String? id;
@override final  String actor;
@override final  String trail;
@override final  TrailPermission permission;
@override final  TrailShareExpand? expand;

/// Create a copy of TrailShare
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$TrailShareCopyWith<_TrailShare> get copyWith => __$TrailShareCopyWithImpl<_TrailShare>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$TrailShareToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _TrailShare&&(identical(other.id, id) || other.id == id)&&(identical(other.actor, actor) || other.actor == actor)&&(identical(other.trail, trail) || other.trail == trail)&&(identical(other.permission, permission) || other.permission == permission)&&(identical(other.expand, expand) || other.expand == expand));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,actor,trail,permission,expand);

@override
String toString() {
  return 'TrailShare(id: $id, actor: $actor, trail: $trail, permission: $permission, expand: $expand)';
}


}

/// @nodoc
abstract mixin class _$TrailShareCopyWith<$Res> implements $TrailShareCopyWith<$Res> {
  factory _$TrailShareCopyWith(_TrailShare value, $Res Function(_TrailShare) _then) = __$TrailShareCopyWithImpl;
@override @useResult
$Res call({
 String? id, String actor, String trail, TrailPermission permission, TrailShareExpand? expand
});


@override $TrailShareExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class __$TrailShareCopyWithImpl<$Res>
    implements _$TrailShareCopyWith<$Res> {
  __$TrailShareCopyWithImpl(this._self, this._then);

  final _TrailShare _self;
  final $Res Function(_TrailShare) _then;

/// Create a copy of TrailShare
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = freezed,Object? actor = null,Object? trail = null,Object? permission = null,Object? expand = freezed,}) {
  return _then(_TrailShare(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,actor: null == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as String,trail: null == trail ? _self.trail : trail // ignore: cast_nullable_to_non_nullable
as String,permission: null == permission ? _self.permission : permission // ignore: cast_nullable_to_non_nullable
as TrailPermission,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as TrailShareExpand?,
  ));
}

/// Create a copy of TrailShare
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$TrailShareExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $TrailShareExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}

// dart format on
