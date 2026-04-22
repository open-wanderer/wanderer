// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'actor.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$Actor {

 String get id; String get collectionId; String get collectionName; String get created; String get updated; String get username;@JsonKey(name: 'preferred_username') String get preferredUsername; String? get domain; String? get summary; String? get published; int? get followerCount; int? get followingCount; String get iri; String get inbox; String? get outbox; String? get icon; String? get followers; String? get following; bool get isLocal;@JsonKey(name: 'public_key') String get publicKey;@JsonKey(name: 'last_fetched') String get lastFetched; String get user;
/// Create a copy of Actor
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ActorCopyWith<Actor> get copyWith => _$ActorCopyWithImpl<Actor>(this as Actor, _$identity);

  /// Serializes this Actor to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Actor&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.collectionName, collectionName) || other.collectionName == collectionName)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&(identical(other.username, username) || other.username == username)&&(identical(other.preferredUsername, preferredUsername) || other.preferredUsername == preferredUsername)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.summary, summary) || other.summary == summary)&&(identical(other.published, published) || other.published == published)&&(identical(other.followerCount, followerCount) || other.followerCount == followerCount)&&(identical(other.followingCount, followingCount) || other.followingCount == followingCount)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.inbox, inbox) || other.inbox == inbox)&&(identical(other.outbox, outbox) || other.outbox == outbox)&&(identical(other.icon, icon) || other.icon == icon)&&(identical(other.followers, followers) || other.followers == followers)&&(identical(other.following, following) || other.following == following)&&(identical(other.isLocal, isLocal) || other.isLocal == isLocal)&&(identical(other.publicKey, publicKey) || other.publicKey == publicKey)&&(identical(other.lastFetched, lastFetched) || other.lastFetched == lastFetched)&&(identical(other.user, user) || other.user == user));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hashAll([runtimeType,id,collectionId,collectionName,created,updated,username,preferredUsername,domain,summary,published,followerCount,followingCount,iri,inbox,outbox,icon,followers,following,isLocal,publicKey,lastFetched,user]);

@override
String toString() {
  return 'Actor(id: $id, collectionId: $collectionId, collectionName: $collectionName, created: $created, updated: $updated, username: $username, preferredUsername: $preferredUsername, domain: $domain, summary: $summary, published: $published, followerCount: $followerCount, followingCount: $followingCount, iri: $iri, inbox: $inbox, outbox: $outbox, icon: $icon, followers: $followers, following: $following, isLocal: $isLocal, publicKey: $publicKey, lastFetched: $lastFetched, user: $user)';
}


}

/// @nodoc
abstract mixin class $ActorCopyWith<$Res>  {
  factory $ActorCopyWith(Actor value, $Res Function(Actor) _then) = _$ActorCopyWithImpl;
@useResult
$Res call({
 String id, String collectionId, String collectionName, String created, String updated, String username,@JsonKey(name: 'preferred_username') String preferredUsername, String? domain, String? summary, String? published, int? followerCount, int? followingCount, String iri, String inbox, String? outbox, String? icon, String? followers, String? following, bool isLocal,@JsonKey(name: 'public_key') String publicKey,@JsonKey(name: 'last_fetched') String lastFetched, String user
});




}
/// @nodoc
class _$ActorCopyWithImpl<$Res>
    implements $ActorCopyWith<$Res> {
  _$ActorCopyWithImpl(this._self, this._then);

  final Actor _self;
  final $Res Function(Actor) _then;

/// Create a copy of Actor
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? collectionId = null,Object? collectionName = null,Object? created = null,Object? updated = null,Object? username = null,Object? preferredUsername = null,Object? domain = freezed,Object? summary = freezed,Object? published = freezed,Object? followerCount = freezed,Object? followingCount = freezed,Object? iri = null,Object? inbox = null,Object? outbox = freezed,Object? icon = freezed,Object? followers = freezed,Object? following = freezed,Object? isLocal = null,Object? publicKey = null,Object? lastFetched = null,Object? user = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
as String,collectionName: null == collectionName ? _self.collectionName : collectionName // ignore: cast_nullable_to_non_nullable
as String,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String,updated: null == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as String,username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,preferredUsername: null == preferredUsername ? _self.preferredUsername : preferredUsername // ignore: cast_nullable_to_non_nullable
as String,domain: freezed == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String?,summary: freezed == summary ? _self.summary : summary // ignore: cast_nullable_to_non_nullable
as String?,published: freezed == published ? _self.published : published // ignore: cast_nullable_to_non_nullable
as String?,followerCount: freezed == followerCount ? _self.followerCount : followerCount // ignore: cast_nullable_to_non_nullable
as int?,followingCount: freezed == followingCount ? _self.followingCount : followingCount // ignore: cast_nullable_to_non_nullable
as int?,iri: null == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String,inbox: null == inbox ? _self.inbox : inbox // ignore: cast_nullable_to_non_nullable
as String,outbox: freezed == outbox ? _self.outbox : outbox // ignore: cast_nullable_to_non_nullable
as String?,icon: freezed == icon ? _self.icon : icon // ignore: cast_nullable_to_non_nullable
as String?,followers: freezed == followers ? _self.followers : followers // ignore: cast_nullable_to_non_nullable
as String?,following: freezed == following ? _self.following : following // ignore: cast_nullable_to_non_nullable
as String?,isLocal: null == isLocal ? _self.isLocal : isLocal // ignore: cast_nullable_to_non_nullable
as bool,publicKey: null == publicKey ? _self.publicKey : publicKey // ignore: cast_nullable_to_non_nullable
as String,lastFetched: null == lastFetched ? _self.lastFetched : lastFetched // ignore: cast_nullable_to_non_nullable
as String,user: null == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

}


/// Adds pattern-matching-related methods to [Actor].
extension ActorPatterns on Actor {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Actor value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Actor() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Actor value)  $default,){
final _that = this;
switch (_that) {
case _Actor():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Actor value)?  $default,){
final _that = this;
switch (_that) {
case _Actor() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String collectionId,  String collectionName,  String created,  String updated,  String username, @JsonKey(name: 'preferred_username')  String preferredUsername,  String? domain,  String? summary,  String? published,  int? followerCount,  int? followingCount,  String iri,  String inbox,  String? outbox,  String? icon,  String? followers,  String? following,  bool isLocal, @JsonKey(name: 'public_key')  String publicKey, @JsonKey(name: 'last_fetched')  String lastFetched,  String user)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Actor() when $default != null:
return $default(_that.id,_that.collectionId,_that.collectionName,_that.created,_that.updated,_that.username,_that.preferredUsername,_that.domain,_that.summary,_that.published,_that.followerCount,_that.followingCount,_that.iri,_that.inbox,_that.outbox,_that.icon,_that.followers,_that.following,_that.isLocal,_that.publicKey,_that.lastFetched,_that.user);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String collectionId,  String collectionName,  String created,  String updated,  String username, @JsonKey(name: 'preferred_username')  String preferredUsername,  String? domain,  String? summary,  String? published,  int? followerCount,  int? followingCount,  String iri,  String inbox,  String? outbox,  String? icon,  String? followers,  String? following,  bool isLocal, @JsonKey(name: 'public_key')  String publicKey, @JsonKey(name: 'last_fetched')  String lastFetched,  String user)  $default,) {final _that = this;
switch (_that) {
case _Actor():
return $default(_that.id,_that.collectionId,_that.collectionName,_that.created,_that.updated,_that.username,_that.preferredUsername,_that.domain,_that.summary,_that.published,_that.followerCount,_that.followingCount,_that.iri,_that.inbox,_that.outbox,_that.icon,_that.followers,_that.following,_that.isLocal,_that.publicKey,_that.lastFetched,_that.user);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String collectionId,  String collectionName,  String created,  String updated,  String username, @JsonKey(name: 'preferred_username')  String preferredUsername,  String? domain,  String? summary,  String? published,  int? followerCount,  int? followingCount,  String iri,  String inbox,  String? outbox,  String? icon,  String? followers,  String? following,  bool isLocal, @JsonKey(name: 'public_key')  String publicKey, @JsonKey(name: 'last_fetched')  String lastFetched,  String user)?  $default,) {final _that = this;
switch (_that) {
case _Actor() when $default != null:
return $default(_that.id,_that.collectionId,_that.collectionName,_that.created,_that.updated,_that.username,_that.preferredUsername,_that.domain,_that.summary,_that.published,_that.followerCount,_that.followingCount,_that.iri,_that.inbox,_that.outbox,_that.icon,_that.followers,_that.following,_that.isLocal,_that.publicKey,_that.lastFetched,_that.user);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Actor extends Actor {
  const _Actor({required this.id, required this.collectionId, required this.collectionName, required this.created, required this.updated, required this.username, @JsonKey(name: 'preferred_username') required this.preferredUsername, this.domain, this.summary, this.published, this.followerCount, this.followingCount, required this.iri, required this.inbox, this.outbox, this.icon, this.followers, this.following, this.isLocal = false, @JsonKey(name: 'public_key') required this.publicKey, @JsonKey(name: 'last_fetched') required this.lastFetched, required this.user}): super._();
  factory _Actor.fromJson(Map<String, dynamic> json) => _$ActorFromJson(json);

@override final  String id;
@override final  String collectionId;
@override final  String collectionName;
@override final  String created;
@override final  String updated;
@override final  String username;
@override@JsonKey(name: 'preferred_username') final  String preferredUsername;
@override final  String? domain;
@override final  String? summary;
@override final  String? published;
@override final  int? followerCount;
@override final  int? followingCount;
@override final  String iri;
@override final  String inbox;
@override final  String? outbox;
@override final  String? icon;
@override final  String? followers;
@override final  String? following;
@override@JsonKey() final  bool isLocal;
@override@JsonKey(name: 'public_key') final  String publicKey;
@override@JsonKey(name: 'last_fetched') final  String lastFetched;
@override final  String user;

/// Create a copy of Actor
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ActorCopyWith<_Actor> get copyWith => __$ActorCopyWithImpl<_Actor>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$ActorToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _Actor&&(identical(other.id, id) || other.id == id)&&(identical(other.collectionId, collectionId) || other.collectionId == collectionId)&&(identical(other.collectionName, collectionName) || other.collectionName == collectionName)&&(identical(other.created, created) || other.created == created)&&(identical(other.updated, updated) || other.updated == updated)&&(identical(other.username, username) || other.username == username)&&(identical(other.preferredUsername, preferredUsername) || other.preferredUsername == preferredUsername)&&(identical(other.domain, domain) || other.domain == domain)&&(identical(other.summary, summary) || other.summary == summary)&&(identical(other.published, published) || other.published == published)&&(identical(other.followerCount, followerCount) || other.followerCount == followerCount)&&(identical(other.followingCount, followingCount) || other.followingCount == followingCount)&&(identical(other.iri, iri) || other.iri == iri)&&(identical(other.inbox, inbox) || other.inbox == inbox)&&(identical(other.outbox, outbox) || other.outbox == outbox)&&(identical(other.icon, icon) || other.icon == icon)&&(identical(other.followers, followers) || other.followers == followers)&&(identical(other.following, following) || other.following == following)&&(identical(other.isLocal, isLocal) || other.isLocal == isLocal)&&(identical(other.publicKey, publicKey) || other.publicKey == publicKey)&&(identical(other.lastFetched, lastFetched) || other.lastFetched == lastFetched)&&(identical(other.user, user) || other.user == user));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hashAll([runtimeType,id,collectionId,collectionName,created,updated,username,preferredUsername,domain,summary,published,followerCount,followingCount,iri,inbox,outbox,icon,followers,following,isLocal,publicKey,lastFetched,user]);

@override
String toString() {
  return 'Actor(id: $id, collectionId: $collectionId, collectionName: $collectionName, created: $created, updated: $updated, username: $username, preferredUsername: $preferredUsername, domain: $domain, summary: $summary, published: $published, followerCount: $followerCount, followingCount: $followingCount, iri: $iri, inbox: $inbox, outbox: $outbox, icon: $icon, followers: $followers, following: $following, isLocal: $isLocal, publicKey: $publicKey, lastFetched: $lastFetched, user: $user)';
}


}

/// @nodoc
abstract mixin class _$ActorCopyWith<$Res> implements $ActorCopyWith<$Res> {
  factory _$ActorCopyWith(_Actor value, $Res Function(_Actor) _then) = __$ActorCopyWithImpl;
@override @useResult
$Res call({
 String id, String collectionId, String collectionName, String created, String updated, String username,@JsonKey(name: 'preferred_username') String preferredUsername, String? domain, String? summary, String? published, int? followerCount, int? followingCount, String iri, String inbox, String? outbox, String? icon, String? followers, String? following, bool isLocal,@JsonKey(name: 'public_key') String publicKey,@JsonKey(name: 'last_fetched') String lastFetched, String user
});




}
/// @nodoc
class __$ActorCopyWithImpl<$Res>
    implements _$ActorCopyWith<$Res> {
  __$ActorCopyWithImpl(this._self, this._then);

  final _Actor _self;
  final $Res Function(_Actor) _then;

/// Create a copy of Actor
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? collectionId = null,Object? collectionName = null,Object? created = null,Object? updated = null,Object? username = null,Object? preferredUsername = null,Object? domain = freezed,Object? summary = freezed,Object? published = freezed,Object? followerCount = freezed,Object? followingCount = freezed,Object? iri = null,Object? inbox = null,Object? outbox = freezed,Object? icon = freezed,Object? followers = freezed,Object? following = freezed,Object? isLocal = null,Object? publicKey = null,Object? lastFetched = null,Object? user = null,}) {
  return _then(_Actor(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,collectionId: null == collectionId ? _self.collectionId : collectionId // ignore: cast_nullable_to_non_nullable
as String,collectionName: null == collectionName ? _self.collectionName : collectionName // ignore: cast_nullable_to_non_nullable
as String,created: null == created ? _self.created : created // ignore: cast_nullable_to_non_nullable
as String,updated: null == updated ? _self.updated : updated // ignore: cast_nullable_to_non_nullable
as String,username: null == username ? _self.username : username // ignore: cast_nullable_to_non_nullable
as String,preferredUsername: null == preferredUsername ? _self.preferredUsername : preferredUsername // ignore: cast_nullable_to_non_nullable
as String,domain: freezed == domain ? _self.domain : domain // ignore: cast_nullable_to_non_nullable
as String?,summary: freezed == summary ? _self.summary : summary // ignore: cast_nullable_to_non_nullable
as String?,published: freezed == published ? _self.published : published // ignore: cast_nullable_to_non_nullable
as String?,followerCount: freezed == followerCount ? _self.followerCount : followerCount // ignore: cast_nullable_to_non_nullable
as int?,followingCount: freezed == followingCount ? _self.followingCount : followingCount // ignore: cast_nullable_to_non_nullable
as int?,iri: null == iri ? _self.iri : iri // ignore: cast_nullable_to_non_nullable
as String,inbox: null == inbox ? _self.inbox : inbox // ignore: cast_nullable_to_non_nullable
as String,outbox: freezed == outbox ? _self.outbox : outbox // ignore: cast_nullable_to_non_nullable
as String?,icon: freezed == icon ? _self.icon : icon // ignore: cast_nullable_to_non_nullable
as String?,followers: freezed == followers ? _self.followers : followers // ignore: cast_nullable_to_non_nullable
as String?,following: freezed == following ? _self.following : following // ignore: cast_nullable_to_non_nullable
as String?,isLocal: null == isLocal ? _self.isLocal : isLocal // ignore: cast_nullable_to_non_nullable
as bool,publicKey: null == publicKey ? _self.publicKey : publicKey // ignore: cast_nullable_to_non_nullable
as String,lastFetched: null == lastFetched ? _self.lastFetched : lastFetched // ignore: cast_nullable_to_non_nullable
as String,user: null == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as String,
  ));
}


}

// dart format on
