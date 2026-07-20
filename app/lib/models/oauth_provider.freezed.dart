// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'oauth_provider.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$OAuthProvider {

 String get name; String get displayName; String get state; String get codeVerifier; String get url; String? get img;
/// Create a copy of OAuthProvider
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$OAuthProviderCopyWith<OAuthProvider> get copyWith => _$OAuthProviderCopyWithImpl<OAuthProvider>(this as OAuthProvider, _$identity);

  /// Serializes this OAuthProvider to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is OAuthProvider&&(identical(other.name, name) || other.name == name)&&(identical(other.displayName, displayName) || other.displayName == displayName)&&(identical(other.state, state) || other.state == state)&&(identical(other.codeVerifier, codeVerifier) || other.codeVerifier == codeVerifier)&&(identical(other.url, url) || other.url == url)&&(identical(other.img, img) || other.img == img));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,name,displayName,state,codeVerifier,url,img);

@override
String toString() {
  return 'OAuthProvider(name: $name, displayName: $displayName, state: $state, codeVerifier: $codeVerifier, url: $url, img: $img)';
}


}

/// @nodoc
abstract mixin class $OAuthProviderCopyWith<$Res>  {
  factory $OAuthProviderCopyWith(OAuthProvider value, $Res Function(OAuthProvider) _then) = _$OAuthProviderCopyWithImpl;
@useResult
$Res call({
 String name, String displayName, String state, String codeVerifier, String url, String? img
});




}
/// @nodoc
class _$OAuthProviderCopyWithImpl<$Res>
    implements $OAuthProviderCopyWith<$Res> {
  _$OAuthProviderCopyWithImpl(this._self, this._then);

  final OAuthProvider _self;
  final $Res Function(OAuthProvider) _then;

/// Create a copy of OAuthProvider
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? name = null,Object? displayName = null,Object? state = null,Object? codeVerifier = null,Object? url = null,Object? img = freezed,}) {
  return _then(_self.copyWith(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,displayName: null == displayName ? _self.displayName : displayName // ignore: cast_nullable_to_non_nullable
as String,state: null == state ? _self.state : state // ignore: cast_nullable_to_non_nullable
as String,codeVerifier: null == codeVerifier ? _self.codeVerifier : codeVerifier // ignore: cast_nullable_to_non_nullable
as String,url: null == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String,img: freezed == img ? _self.img : img // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}

}


/// Adds pattern-matching-related methods to [OAuthProvider].
extension OAuthProviderPatterns on OAuthProvider {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _OAuthProvider value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _OAuthProvider() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _OAuthProvider value)  $default,){
final _that = this;
switch (_that) {
case _OAuthProvider():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _OAuthProvider value)?  $default,){
final _that = this;
switch (_that) {
case _OAuthProvider() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String name,  String displayName,  String state,  String codeVerifier,  String url,  String? img)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _OAuthProvider() when $default != null:
return $default(_that.name,_that.displayName,_that.state,_that.codeVerifier,_that.url,_that.img);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String name,  String displayName,  String state,  String codeVerifier,  String url,  String? img)  $default,) {final _that = this;
switch (_that) {
case _OAuthProvider():
return $default(_that.name,_that.displayName,_that.state,_that.codeVerifier,_that.url,_that.img);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String name,  String displayName,  String state,  String codeVerifier,  String url,  String? img)?  $default,) {final _that = this;
switch (_that) {
case _OAuthProvider() when $default != null:
return $default(_that.name,_that.displayName,_that.state,_that.codeVerifier,_that.url,_that.img);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _OAuthProvider implements OAuthProvider {
  const _OAuthProvider({required this.name, required this.displayName, required this.state, required this.codeVerifier, required this.url, this.img});
  factory _OAuthProvider.fromJson(Map<String, dynamic> json) => _$OAuthProviderFromJson(json);

@override final  String name;
@override final  String displayName;
@override final  String state;
@override final  String codeVerifier;
@override final  String url;
@override final  String? img;

/// Create a copy of OAuthProvider
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$OAuthProviderCopyWith<_OAuthProvider> get copyWith => __$OAuthProviderCopyWithImpl<_OAuthProvider>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$OAuthProviderToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _OAuthProvider&&(identical(other.name, name) || other.name == name)&&(identical(other.displayName, displayName) || other.displayName == displayName)&&(identical(other.state, state) || other.state == state)&&(identical(other.codeVerifier, codeVerifier) || other.codeVerifier == codeVerifier)&&(identical(other.url, url) || other.url == url)&&(identical(other.img, img) || other.img == img));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,name,displayName,state,codeVerifier,url,img);

@override
String toString() {
  return 'OAuthProvider(name: $name, displayName: $displayName, state: $state, codeVerifier: $codeVerifier, url: $url, img: $img)';
}


}

/// @nodoc
abstract mixin class _$OAuthProviderCopyWith<$Res> implements $OAuthProviderCopyWith<$Res> {
  factory _$OAuthProviderCopyWith(_OAuthProvider value, $Res Function(_OAuthProvider) _then) = __$OAuthProviderCopyWithImpl;
@override @useResult
$Res call({
 String name, String displayName, String state, String codeVerifier, String url, String? img
});




}
/// @nodoc
class __$OAuthProviderCopyWithImpl<$Res>
    implements _$OAuthProviderCopyWith<$Res> {
  __$OAuthProviderCopyWithImpl(this._self, this._then);

  final _OAuthProvider _self;
  final $Res Function(_OAuthProvider) _then;

/// Create a copy of OAuthProvider
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? name = null,Object? displayName = null,Object? state = null,Object? codeVerifier = null,Object? url = null,Object? img = freezed,}) {
  return _then(_OAuthProvider(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,displayName: null == displayName ? _self.displayName : displayName // ignore: cast_nullable_to_non_nullable
as String,state: null == state ? _self.state : state // ignore: cast_nullable_to_non_nullable
as String,codeVerifier: null == codeVerifier ? _self.codeVerifier : codeVerifier // ignore: cast_nullable_to_non_nullable
as String,url: null == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String,img: freezed == img ? _self.img : img // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}


}

// dart format on
