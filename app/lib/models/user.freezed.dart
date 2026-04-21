// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'user.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$User {

 String get id; String get username; String get email; bool get emailVisibility; bool get verified; String get created; String get updated; String? get avatar; UserExpand? get expand;
/// Create a copy of User
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$UserCopyWith<User> get copyWith => _$UserCopyWithImpl<User>(this as User, _$identity);

  /// Serializes this User to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is User&&(identical(other.id, id) || other.id == id)&&(identical(other.username, username) || other.username == username)&&(identical(other.email, email) || other.email == email)&&(identical(other.emailVisibility, emailVisibility) || other.emailVisibility == emailVisibility)&&(identical(other.verified, verified) || other.verified == verified)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&(identical(other.avatar, avatar) || other.avatar == avatar)&&(identical(other.expand, expand) || other.expand == expand));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,username,email,emailVisibility,verified,created,updated,avatar,expand);

@override
String toString() {
  return 'User(id: $id, username: $username, email: $email, emailVisibility: $emailVisibility, verified: $verified, created: $created, updated: $updated, avatar: $avatar, expand: $expand)';
}


}

/// @nodoc
abstract mixin class $UserCopyWith<$Res>  {
  factory $UserCopyWith(User value, $Res Function(User) _then) = _$UserCopyWithImpl;
@useResult
$Res call({
 String id, String username, String email, bool emailVisibility, bool verified, String created, String updated, String? avatar, UserExpand? expand
});


$UserExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class _$UserCopyWithImpl<$Res>
    implements $UserCopyWith<$Res> {
  _$UserCopyWithImpl(this._self, this._then);

  final User _self;
  final $Res Function(User) _then;

/// Create a copy of User
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? username = null,Object? email = null,Object? emailVisibility = null,Object? verified = null,Object? created = null,Object? updated = null,Object? avatar = freezed,Object? expand = freezed,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,email: null == email ? _self.email : email // ignore: cast_nullable_to_non_nullable
as String,emailVisibility: null == emailVisibility ? _self.emailVisibility : emailVisibility // ignore: cast_nullable_to_non_nullable
as bool,verified: null == verified ? _self.verified : verified // ignore: cast_nullable_to_non_nullable
as bool,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String,updated: null == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as String,avatar: freezed == avatar ? _self.avatar : avatar // ignore: cast_nullable_to_non_nullable
as String?,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as UserExpand?,
  ));
}
/// Create a copy of User
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$UserExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $UserExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}


/// Adds pattern-matching-related methods to [User].
extension UserPatterns on User {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _User value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _User() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _User value)  $default,){
final _that = this;
switch (_that) {
case _User():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _User value)?  $default,){
final _that = this;
switch (_that) {
case _User() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String username,  String email,  bool emailVisibility,  bool verified,  String created,  String updated,  String? avatar,  UserExpand? expand)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _User() when $default != null:
return $default(_that.id,_that.username,_that.email,_that.emailVisibility,_that.verified,_that.created,_that.updated,_that.avatar,_that.expand);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String username,  String email,  bool emailVisibility,  bool verified,  String created,  String updated,  String? avatar,  UserExpand? expand)  $default,) {final _that = this;
switch (_that) {
case _User():
return $default(_that.id,_that.username,_that.email,_that.emailVisibility,_that.verified,_that.created,_that.updated,_that.avatar,_that.expand);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String username,  String email,  bool emailVisibility,  bool verified,  String created,  String updated,  String? avatar,  UserExpand? expand)?  $default,) {final _that = this;
switch (_that) {
case _User() when $default != null:
return $default(_that.id,_that.username,_that.email,_that.emailVisibility,_that.verified,_that.created,_that.updated,_that.avatar,_that.expand);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _User extends User {
  const _User({required this.id, required this.username, required this.email, required this.emailVisibility, required this.verified, required this.created, required this.updated, this.avatar, this.expand}): super._();
  factory _User.fromJson(Map<String, dynamic> json) => _$UserFromJson(json);

@override final  String id;
@override final  String username;
@override final  String email;
@override final  bool emailVisibility;
@override final  bool verified;
@override final  String created;
@override final  String updated;
@override final  String? avatar;
@override final  UserExpand? expand;

/// Create a copy of User
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$UserCopyWith<_User> get copyWith => __$UserCopyWithImpl<_User>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$UserToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _User&&(identical(other.id, id) || other.id == id)&&(identical(other.username, username) || other.username == username)&&(identical(other.email, email) || other.email == email)&&(identical(other.emailVisibility, emailVisibility) || other.emailVisibility == emailVisibility)&&(identical(other.verified, verified) || other.verified == verified)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&(identical(other.avatar, avatar) || other.avatar == avatar)&&(identical(other.expand, expand) || other.expand == expand));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,username,email,emailVisibility,verified,created,updated,avatar,expand);

@override
String toString() {
  return 'User(id: $id, username: $username, email: $email, emailVisibility: $emailVisibility, verified: $verified, created: $created, updated: $updated, avatar: $avatar, expand: $expand)';
}


}

/// @nodoc
abstract mixin class _$UserCopyWith<$Res> implements $UserCopyWith<$Res> {
  factory _$UserCopyWith(_User value, $Res Function(_User) _then) = __$UserCopyWithImpl;
@override @useResult
$Res call({
 String id, String username, String email, bool emailVisibility, bool verified, String created, String updated, String? avatar, UserExpand? expand
});


@override $UserExpandCopyWith<$Res>? get expand;

}
/// @nodoc
class __$UserCopyWithImpl<$Res>
    implements _$UserCopyWith<$Res> {
  __$UserCopyWithImpl(this._self, this._then);

  final _User _self;
  final $Res Function(_User) _then;

/// Create a copy of User
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? username = null,Object? email = null,Object? emailVisibility = null,Object? verified = null,Object? created = null,Object? updated = null,Object? avatar = freezed,Object? expand = freezed,}) {
  return _then(_User(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,email: null == email ? _self.email : email // ignore: cast_nullable_to_non_nullable
as String,emailVisibility: null == emailVisibility ? _self.emailVisibility : emailVisibility // ignore: cast_nullable_to_non_nullable
as bool,verified: null == verified ? _self.verified : verified // ignore: cast_nullable_to_non_nullable
as bool,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String,updated: null == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as String,avatar: freezed == avatar ? _self.avatar : avatar // ignore: cast_nullable_to_non_nullable
as String?,expand: freezed == expand ? _self.expand : expand // ignore: cast_nullable_to_non_nullable
as UserExpand?,
  ));
}

/// Create a copy of User
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$UserExpandCopyWith<$Res>? get expand {
    if (_self.expand == null) {
    return null;
  }

  return $UserExpandCopyWith<$Res>(_self.expand!, (value) {
    return _then(_self.copyWith(expand: value));
  });
}
}


/// @nodoc
mixin _$UserExpand {

@JsonKey(name: 'activitypub_actors_via_user') Actor? get actor;
/// Create a copy of UserExpand
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$UserExpandCopyWith<UserExpand> get copyWith => _$UserExpandCopyWithImpl<UserExpand>(this as UserExpand, _$identity);

  /// Serializes this UserExpand to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is UserExpand&&(identical(other.actor, actor) || other.actor == actor));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,actor);

@override
String toString() {
  return 'UserExpand(actor: $actor)';
}


}

/// @nodoc
abstract mixin class $UserExpandCopyWith<$Res>  {
  factory $UserExpandCopyWith(UserExpand value, $Res Function(UserExpand) _then) = _$UserExpandCopyWithImpl;
@useResult
$Res call({
@JsonKey(name: 'activitypub_actors_via_user') Actor? actor
});


$ActorCopyWith<$Res>? get actor;

}
/// @nodoc
class _$UserExpandCopyWithImpl<$Res>
    implements $UserExpandCopyWith<$Res> {
  _$UserExpandCopyWithImpl(this._self, this._then);

  final UserExpand _self;
  final $Res Function(UserExpand) _then;

/// Create a copy of UserExpand
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? actor = freezed,}) {
  return _then(_self.copyWith(
actor: freezed == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as Actor?,
  ));
}
/// Create a copy of UserExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res>? get actor {
    if (_self.actor == null) {
    return null;
  }

  return $ActorCopyWith<$Res>(_self.actor!, (value) {
    return _then(_self.copyWith(actor: value));
  });
}
}


/// Adds pattern-matching-related methods to [UserExpand].
extension UserExpandPatterns on UserExpand {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _UserExpand value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _UserExpand() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _UserExpand value)  $default,){
final _that = this;
switch (_that) {
case _UserExpand():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _UserExpand value)?  $default,){
final _that = this;
switch (_that) {
case _UserExpand() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function(@JsonKey(name: 'activitypub_actors_via_user')  Actor? actor)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _UserExpand() when $default != null:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function(@JsonKey(name: 'activitypub_actors_via_user')  Actor? actor)  $default,) {final _that = this;
switch (_that) {
case _UserExpand():
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function(@JsonKey(name: 'activitypub_actors_via_user')  Actor? actor)?  $default,) {final _that = this;
switch (_that) {
case _UserExpand() when $default != null:
return $default(_that.actor);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _UserExpand implements UserExpand {
  const _UserExpand({@JsonKey(name: 'activitypub_actors_via_user') this.actor});
  factory _UserExpand.fromJson(Map<String, dynamic> json) => _$UserExpandFromJson(json);

@override@JsonKey(name: 'activitypub_actors_via_user') final  Actor? actor;

/// Create a copy of UserExpand
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$UserExpandCopyWith<_UserExpand> get copyWith => __$UserExpandCopyWithImpl<_UserExpand>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$UserExpandToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _UserExpand&&(identical(other.actor, actor) || other.actor == actor));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,actor);

@override
String toString() {
  return 'UserExpand(actor: $actor)';
}


}

/// @nodoc
abstract mixin class _$UserExpandCopyWith<$Res> implements $UserExpandCopyWith<$Res> {
  factory _$UserExpandCopyWith(_UserExpand value, $Res Function(_UserExpand) _then) = __$UserExpandCopyWithImpl;
@override @useResult
$Res call({
@JsonKey(name: 'activitypub_actors_via_user') Actor? actor
});


@override $ActorCopyWith<$Res>? get actor;

}
/// @nodoc
class __$UserExpandCopyWithImpl<$Res>
    implements _$UserExpandCopyWith<$Res> {
  __$UserExpandCopyWithImpl(this._self, this._then);

  final _UserExpand _self;
  final $Res Function(_UserExpand) _then;

/// Create a copy of UserExpand
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? actor = freezed,}) {
  return _then(_UserExpand(
actor: freezed == actor ? _self.actor : actor // ignore: cast_nullable_to_non_nullable
as Actor?,
  ));
}

/// Create a copy of UserExpand
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$ActorCopyWith<$Res>? get actor {
    if (_self.actor == null) {
    return null;
  }

  return $ActorCopyWith<$Res>(_self.actor!, (value) {
    return _then(_self.copyWith(actor: value));
  });
}
}

// dart format on
