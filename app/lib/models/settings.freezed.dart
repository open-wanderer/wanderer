// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'settings.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$SettingsLocation {

 String get name; double get lat; double get lon;
/// Create a copy of SettingsLocation
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SettingsLocationCopyWith<SettingsLocation> get copyWith => _$SettingsLocationCopyWithImpl<SettingsLocation>(this as SettingsLocation, _$identity);

  /// Serializes this SettingsLocation to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is SettingsLocation&&(identical(other.name, name) || other.name == name)&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,name,lat,lon);

@override
String toString() {
  return 'SettingsLocation(name: $name, lat: $lat, lon: $lon)';
}


}

/// @nodoc
abstract mixin class $SettingsLocationCopyWith<$Res>  {
  factory $SettingsLocationCopyWith(SettingsLocation value, $Res Function(SettingsLocation) _then) = _$SettingsLocationCopyWithImpl;
@useResult
$Res call({
 String name, double lat, double lon
});




}
/// @nodoc
class _$SettingsLocationCopyWithImpl<$Res>
    implements $SettingsLocationCopyWith<$Res> {
  _$SettingsLocationCopyWithImpl(this._self, this._then);

  final SettingsLocation _self;
  final $Res Function(SettingsLocation) _then;

/// Create a copy of SettingsLocation
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? name = null,Object? lat = null,Object? lon = null,}) {
  return _then(_self.copyWith(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,lat: null == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double,lon: null == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double,
  ));
}

}


/// Adds pattern-matching-related methods to [SettingsLocation].
extension SettingsLocationPatterns on SettingsLocation {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _SettingsLocation value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _SettingsLocation() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _SettingsLocation value)  $default,){
final _that = this;
switch (_that) {
case _SettingsLocation():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _SettingsLocation value)?  $default,){
final _that = this;
switch (_that) {
case _SettingsLocation() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String name,  double lat,  double lon)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _SettingsLocation() when $default != null:
return $default(_that.name,_that.lat,_that.lon);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String name,  double lat,  double lon)  $default,) {final _that = this;
switch (_that) {
case _SettingsLocation():
return $default(_that.name,_that.lat,_that.lon);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String name,  double lat,  double lon)?  $default,) {final _that = this;
switch (_that) {
case _SettingsLocation() when $default != null:
return $default(_that.name,_that.lat,_that.lon);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _SettingsLocation implements SettingsLocation {
  const _SettingsLocation({required this.name, required this.lat, required this.lon});
  factory _SettingsLocation.fromJson(Map<String, dynamic> json) => _$SettingsLocationFromJson(json);

@override final  String name;
@override final  double lat;
@override final  double lon;

/// Create a copy of SettingsLocation
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SettingsLocationCopyWith<_SettingsLocation> get copyWith => __$SettingsLocationCopyWithImpl<_SettingsLocation>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SettingsLocationToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _SettingsLocation&&(identical(other.name, name) || other.name == name)&&(identical(other.lat, lat) || other.lat == lat)&&(identical(other.lon, lon) || other.lon == lon));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,name,lat,lon);

@override
String toString() {
  return 'SettingsLocation(name: $name, lat: $lat, lon: $lon)';
}


}

/// @nodoc
abstract mixin class _$SettingsLocationCopyWith<$Res> implements $SettingsLocationCopyWith<$Res> {
  factory _$SettingsLocationCopyWith(_SettingsLocation value, $Res Function(_SettingsLocation) _then) = __$SettingsLocationCopyWithImpl;
@override @useResult
$Res call({
 String name, double lat, double lon
});




}
/// @nodoc
class __$SettingsLocationCopyWithImpl<$Res>
    implements _$SettingsLocationCopyWith<$Res> {
  __$SettingsLocationCopyWithImpl(this._self, this._then);

  final _SettingsLocation _self;
  final $Res Function(_SettingsLocation) _then;

/// Create a copy of SettingsLocation
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? name = null,Object? lat = null,Object? lon = null,}) {
  return _then(_SettingsLocation(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,lat: null == lat ? _self.lat : lat // ignore: cast_nullable_to_non_nullable
as double,lon: null == lon ? _self.lon : lon // ignore: cast_nullable_to_non_nullable
as double,
  ));
}


}


/// @nodoc
mixin _$SettingsPrivacy {

 String get account; String get trails; String get lists;
/// Create a copy of SettingsPrivacy
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SettingsPrivacyCopyWith<SettingsPrivacy> get copyWith => _$SettingsPrivacyCopyWithImpl<SettingsPrivacy>(this as SettingsPrivacy, _$identity);

  /// Serializes this SettingsPrivacy to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is SettingsPrivacy&&(identical(other.account, account) || other.account == account)&&(identical(other.trails, trails) || other.trails == trails)&&(identical(other.lists, lists) || other.lists == lists));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,account,trails,lists);

@override
String toString() {
  return 'SettingsPrivacy(account: $account, trails: $trails, lists: $lists)';
}


}

/// @nodoc
abstract mixin class $SettingsPrivacyCopyWith<$Res>  {
  factory $SettingsPrivacyCopyWith(SettingsPrivacy value, $Res Function(SettingsPrivacy) _then) = _$SettingsPrivacyCopyWithImpl;
@useResult
$Res call({
 String account, String trails, String lists
});




}
/// @nodoc
class _$SettingsPrivacyCopyWithImpl<$Res>
    implements $SettingsPrivacyCopyWith<$Res> {
  _$SettingsPrivacyCopyWithImpl(this._self, this._then);

  final SettingsPrivacy _self;
  final $Res Function(SettingsPrivacy) _then;

/// Create a copy of SettingsPrivacy
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? account = null,Object? trails = null,Object? lists = null,}) {
  return _then(_self.copyWith(
account: null == account ? _self.account : account // ignore: cast_nullable_to_non_nullable
as String,trails: null == trails ? _self.trails : trails // ignore: cast_nullable_to_non_nullable
as String,lists: null == lists ? _self.lists : lists // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

}


/// Adds pattern-matching-related methods to [SettingsPrivacy].
extension SettingsPrivacyPatterns on SettingsPrivacy {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _SettingsPrivacy value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _SettingsPrivacy() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _SettingsPrivacy value)  $default,){
final _that = this;
switch (_that) {
case _SettingsPrivacy():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _SettingsPrivacy value)?  $default,){
final _that = this;
switch (_that) {
case _SettingsPrivacy() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String account,  String trails,  String lists)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _SettingsPrivacy() when $default != null:
return $default(_that.account,_that.trails,_that.lists);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String account,  String trails,  String lists)  $default,) {final _that = this;
switch (_that) {
case _SettingsPrivacy():
return $default(_that.account,_that.trails,_that.lists);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String account,  String trails,  String lists)?  $default,) {final _that = this;
switch (_that) {
case _SettingsPrivacy() when $default != null:
return $default(_that.account,_that.trails,_that.lists);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _SettingsPrivacy implements SettingsPrivacy {
  const _SettingsPrivacy({required this.account, required this.trails, required this.lists});
  factory _SettingsPrivacy.fromJson(Map<String, dynamic> json) => _$SettingsPrivacyFromJson(json);

@override final  String account;
@override final  String trails;
@override final  String lists;

/// Create a copy of SettingsPrivacy
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SettingsPrivacyCopyWith<_SettingsPrivacy> get copyWith => __$SettingsPrivacyCopyWithImpl<_SettingsPrivacy>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SettingsPrivacyToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _SettingsPrivacy&&(identical(other.account, account) || other.account == account)&&(identical(other.trails, trails) || other.trails == trails)&&(identical(other.lists, lists) || other.lists == lists));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,account,trails,lists);

@override
String toString() {
  return 'SettingsPrivacy(account: $account, trails: $trails, lists: $lists)';
}


}

/// @nodoc
abstract mixin class _$SettingsPrivacyCopyWith<$Res> implements $SettingsPrivacyCopyWith<$Res> {
  factory _$SettingsPrivacyCopyWith(_SettingsPrivacy value, $Res Function(_SettingsPrivacy) _then) = __$SettingsPrivacyCopyWithImpl;
@override @useResult
$Res call({
 String account, String trails, String lists
});




}
/// @nodoc
class __$SettingsPrivacyCopyWithImpl<$Res>
    implements _$SettingsPrivacyCopyWith<$Res> {
  __$SettingsPrivacyCopyWithImpl(this._self, this._then);

  final _SettingsPrivacy _self;
  final $Res Function(_SettingsPrivacy) _then;

/// Create a copy of SettingsPrivacy
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? account = null,Object? trails = null,Object? lists = null,}) {
  return _then(_SettingsPrivacy(
account: null == account ? _self.account : account // ignore: cast_nullable_to_non_nullable
as String,trails: null == trails ? _self.trails : trails // ignore: cast_nullable_to_non_nullable
as String,lists: null == lists ? _self.lists : lists // ignore: cast_nullable_to_non_nullable
as String,
  ));
}


}


/// @nodoc
mixin _$NotificationPreference {

 bool get web; bool get email;
/// Create a copy of NotificationPreference
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$NotificationPreferenceCopyWith<NotificationPreference> get copyWith => _$NotificationPreferenceCopyWithImpl<NotificationPreference>(this as NotificationPreference, _$identity);

  /// Serializes this NotificationPreference to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is NotificationPreference&&(identical(other.web, web) || other.web == web)&&(identical(other.email, email) || other.email == email));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,web,email);

@override
String toString() {
  return 'NotificationPreference(web: $web, email: $email)';
}


}

/// @nodoc
abstract mixin class $NotificationPreferenceCopyWith<$Res>  {
  factory $NotificationPreferenceCopyWith(NotificationPreference value, $Res Function(NotificationPreference) _then) = _$NotificationPreferenceCopyWithImpl;
@useResult
$Res call({
 bool web, bool email
});




}
/// @nodoc
class _$NotificationPreferenceCopyWithImpl<$Res>
    implements $NotificationPreferenceCopyWith<$Res> {
  _$NotificationPreferenceCopyWithImpl(this._self, this._then);

  final NotificationPreference _self;
  final $Res Function(NotificationPreference) _then;

/// Create a copy of NotificationPreference
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? web = null,Object? email = null,}) {
  return _then(_self.copyWith(
web: null == web ? _self.web : web // ignore: cast_nullable_to_non_nullable
as bool,email: null == email ? _self.email : email // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}

}


/// Adds pattern-matching-related methods to [NotificationPreference].
extension NotificationPreferencePatterns on NotificationPreference {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _NotificationPreference value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _NotificationPreference() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _NotificationPreference value)  $default,){
final _that = this;
switch (_that) {
case _NotificationPreference():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _NotificationPreference value)?  $default,){
final _that = this;
switch (_that) {
case _NotificationPreference() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( bool web,  bool email)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _NotificationPreference() when $default != null:
return $default(_that.web,_that.email);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( bool web,  bool email)  $default,) {final _that = this;
switch (_that) {
case _NotificationPreference():
return $default(_that.web,_that.email);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( bool web,  bool email)?  $default,) {final _that = this;
switch (_that) {
case _NotificationPreference() when $default != null:
return $default(_that.web,_that.email);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _NotificationPreference implements NotificationPreference {
  const _NotificationPreference({required this.web, required this.email});
  factory _NotificationPreference.fromJson(Map<String, dynamic> json) => _$NotificationPreferenceFromJson(json);

@override final  bool web;
@override final  bool email;

/// Create a copy of NotificationPreference
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$NotificationPreferenceCopyWith<_NotificationPreference> get copyWith => __$NotificationPreferenceCopyWithImpl<_NotificationPreference>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$NotificationPreferenceToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _NotificationPreference&&(identical(other.web, web) || other.web == web)&&(identical(other.email, email) || other.email == email));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,web,email);

@override
String toString() {
  return 'NotificationPreference(web: $web, email: $email)';
}


}

/// @nodoc
abstract mixin class _$NotificationPreferenceCopyWith<$Res> implements $NotificationPreferenceCopyWith<$Res> {
  factory _$NotificationPreferenceCopyWith(_NotificationPreference value, $Res Function(_NotificationPreference) _then) = __$NotificationPreferenceCopyWithImpl;
@override @useResult
$Res call({
 bool web, bool email
});




}
/// @nodoc
class __$NotificationPreferenceCopyWithImpl<$Res>
    implements _$NotificationPreferenceCopyWith<$Res> {
  __$NotificationPreferenceCopyWithImpl(this._self, this._then);

  final _NotificationPreference _self;
  final $Res Function(_NotificationPreference) _then;

/// Create a copy of NotificationPreference
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? web = null,Object? email = null,}) {
  return _then(_NotificationPreference(
web: null == web ? _self.web : web // ignore: cast_nullable_to_non_nullable
as bool,email: null == email ? _self.email : email // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}


}


/// @nodoc
mixin _$Behavior {

 bool? get allowAutoGeolocate; int? get mapClusteringMaxZoom; bool? get showTrailStartMarker;
/// Create a copy of Behavior
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$BehaviorCopyWith<Behavior> get copyWith => _$BehaviorCopyWithImpl<Behavior>(this as Behavior, _$identity);

  /// Serializes this Behavior to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Behavior&&(identical(other.allowAutoGeolocate, allowAutoGeolocate) || other.allowAutoGeolocate == allowAutoGeolocate)&&(identical(other.mapClusteringMaxZoom, mapClusteringMaxZoom) || other.mapClusteringMaxZoom == mapClusteringMaxZoom)&&(identical(other.showTrailStartMarker, showTrailStartMarker) || other.showTrailStartMarker == showTrailStartMarker));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,allowAutoGeolocate,mapClusteringMaxZoom,showTrailStartMarker);

@override
String toString() {
  return 'Behavior(allowAutoGeolocate: $allowAutoGeolocate, mapClusteringMaxZoom: $mapClusteringMaxZoom, showTrailStartMarker: $showTrailStartMarker)';
}


}

/// @nodoc
abstract mixin class $BehaviorCopyWith<$Res>  {
  factory $BehaviorCopyWith(Behavior value, $Res Function(Behavior) _then) = _$BehaviorCopyWithImpl;
@useResult
$Res call({
 bool? allowAutoGeolocate, int? mapClusteringMaxZoom, bool? showTrailStartMarker
});




}
/// @nodoc
class _$BehaviorCopyWithImpl<$Res>
    implements $BehaviorCopyWith<$Res> {
  _$BehaviorCopyWithImpl(this._self, this._then);

  final Behavior _self;
  final $Res Function(Behavior) _then;

/// Create a copy of Behavior
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? allowAutoGeolocate = freezed,Object? mapClusteringMaxZoom = freezed,Object? showTrailStartMarker = freezed,}) {
  return _then(_self.copyWith(
allowAutoGeolocate: freezed == allowAutoGeolocate ? _self.allowAutoGeolocate : allowAutoGeolocate // ignore: cast_nullable_to_non_nullable
as bool?,mapClusteringMaxZoom: freezed == mapClusteringMaxZoom ? _self.mapClusteringMaxZoom : mapClusteringMaxZoom // ignore: cast_nullable_to_non_nullable
as int?,showTrailStartMarker: freezed == showTrailStartMarker ? _self.showTrailStartMarker : showTrailStartMarker // ignore: cast_nullable_to_non_nullable
as bool?,
  ));
}

}


/// Adds pattern-matching-related methods to [Behavior].
extension BehaviorPatterns on Behavior {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Behavior value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Behavior() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Behavior value)  $default,){
final _that = this;
switch (_that) {
case _Behavior():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Behavior value)?  $default,){
final _that = this;
switch (_that) {
case _Behavior() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( bool? allowAutoGeolocate,  int? mapClusteringMaxZoom,  bool? showTrailStartMarker)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Behavior() when $default != null:
return $default(_that.allowAutoGeolocate,_that.mapClusteringMaxZoom,_that.showTrailStartMarker);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( bool? allowAutoGeolocate,  int? mapClusteringMaxZoom,  bool? showTrailStartMarker)  $default,) {final _that = this;
switch (_that) {
case _Behavior():
return $default(_that.allowAutoGeolocate,_that.mapClusteringMaxZoom,_that.showTrailStartMarker);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( bool? allowAutoGeolocate,  int? mapClusteringMaxZoom,  bool? showTrailStartMarker)?  $default,) {final _that = this;
switch (_that) {
case _Behavior() when $default != null:
return $default(_that.allowAutoGeolocate,_that.mapClusteringMaxZoom,_that.showTrailStartMarker);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Behavior implements Behavior {
  const _Behavior({this.allowAutoGeolocate, this.mapClusteringMaxZoom, this.showTrailStartMarker});
  factory _Behavior.fromJson(Map<String, dynamic> json) => _$BehaviorFromJson(json);

@override final  bool? allowAutoGeolocate;
@override final  int? mapClusteringMaxZoom;
@override final  bool? showTrailStartMarker;

/// Create a copy of Behavior
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$BehaviorCopyWith<_Behavior> get copyWith => __$BehaviorCopyWithImpl<_Behavior>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$BehaviorToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _Behavior&&(identical(other.allowAutoGeolocate, allowAutoGeolocate) || other.allowAutoGeolocate == allowAutoGeolocate)&&(identical(other.mapClusteringMaxZoom, mapClusteringMaxZoom) || other.mapClusteringMaxZoom == mapClusteringMaxZoom)&&(identical(other.showTrailStartMarker, showTrailStartMarker) || other.showTrailStartMarker == showTrailStartMarker));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,allowAutoGeolocate,mapClusteringMaxZoom,showTrailStartMarker);

@override
String toString() {
  return 'Behavior(allowAutoGeolocate: $allowAutoGeolocate, mapClusteringMaxZoom: $mapClusteringMaxZoom, showTrailStartMarker: $showTrailStartMarker)';
}


}

/// @nodoc
abstract mixin class _$BehaviorCopyWith<$Res> implements $BehaviorCopyWith<$Res> {
  factory _$BehaviorCopyWith(_Behavior value, $Res Function(_Behavior) _then) = __$BehaviorCopyWithImpl;
@override @useResult
$Res call({
 bool? allowAutoGeolocate, int? mapClusteringMaxZoom, bool? showTrailStartMarker
});




}
/// @nodoc
class __$BehaviorCopyWithImpl<$Res>
    implements _$BehaviorCopyWith<$Res> {
  __$BehaviorCopyWithImpl(this._self, this._then);

  final _Behavior _self;
  final $Res Function(_Behavior) _then;

/// Create a copy of Behavior
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? allowAutoGeolocate = freezed,Object? mapClusteringMaxZoom = freezed,Object? showTrailStartMarker = freezed,}) {
  return _then(_Behavior(
allowAutoGeolocate: freezed == allowAutoGeolocate ? _self.allowAutoGeolocate : allowAutoGeolocate // ignore: cast_nullable_to_non_nullable
as bool?,mapClusteringMaxZoom: freezed == mapClusteringMaxZoom ? _self.mapClusteringMaxZoom : mapClusteringMaxZoom // ignore: cast_nullable_to_non_nullable
as int?,showTrailStartMarker: freezed == showTrailStartMarker ? _self.showTrailStartMarker : showTrailStartMarker // ignore: cast_nullable_to_non_nullable
as bool?,
  ));
}


}


/// @nodoc
mixin _$Settings {

 String? get id; String? get unit; Language? get language; String? get bio; SettingsLocation? get location; String? get user; SettingsPrivacy? get privacy; Map<String, NotificationPreference>? get notifications; Behavior? get behavior;
/// Create a copy of Settings
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SettingsCopyWith<Settings> get copyWith => _$SettingsCopyWithImpl<Settings>(this as Settings, _$identity);

  /// Serializes this Settings to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Settings&&(identical(other.id, id) || other.id == id)&&(identical(other.unit, unit) || other.unit == unit)&&(identical(other.language, language) || other.language == language)&&(identical(other.bio, bio) || other.bio == bio)&&(identical(other.location, location) || other.location == location)&&(identical(other.user, user) || other.user == user)&&(identical(other.privacy, privacy) || other.privacy == privacy)&&const DeepCollectionEquality().equals(other.notifications, notifications)&&(identical(other.behavior, behavior) || other.behavior == behavior));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,unit,language,bio,location,user,privacy,const DeepCollectionEquality().hash(notifications),behavior);

@override
String toString() {
  return 'Settings(id: $id, unit: $unit, language: $language, bio: $bio, location: $location, user: $user, privacy: $privacy, notifications: $notifications, behavior: $behavior)';
}


}

/// @nodoc
abstract mixin class $SettingsCopyWith<$Res>  {
  factory $SettingsCopyWith(Settings value, $Res Function(Settings) _then) = _$SettingsCopyWithImpl;
@useResult
$Res call({
 String? id, String? unit, Language? language, String? bio, SettingsLocation? location, String? user, SettingsPrivacy? privacy, Map<String, NotificationPreference>? notifications, Behavior? behavior
});


$SettingsLocationCopyWith<$Res>? get location;$SettingsPrivacyCopyWith<$Res>? get privacy;$BehaviorCopyWith<$Res>? get behavior;

}
/// @nodoc
class _$SettingsCopyWithImpl<$Res>
    implements $SettingsCopyWith<$Res> {
  _$SettingsCopyWithImpl(this._self, this._then);

  final Settings _self;
  final $Res Function(Settings) _then;

/// Create a copy of Settings
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = freezed,Object? unit = freezed,Object? language = freezed,Object? bio = freezed,Object? location = freezed,Object? user = freezed,Object? privacy = freezed,Object? notifications = freezed,Object? behavior = freezed,}) {
  return _then(_self.copyWith(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,unit: freezed == unit ? _self.unit : unit // ignore: cast_nullable_to_non_nullable
as String?,language: freezed == language ? _self.language : language // ignore: cast_nullable_to_non_nullable
as Language?,bio: freezed == bio ? _self.bio : bio // ignore: cast_nullable_to_non_nullable
as String?,location: freezed == location ? _self.location : location // ignore: cast_nullable_to_non_nullable
as SettingsLocation?,user: freezed == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as String?,privacy: freezed == privacy ? _self.privacy : privacy // ignore: cast_nullable_to_non_nullable
as SettingsPrivacy?,notifications: freezed == notifications ? _self.notifications : notifications // ignore: cast_nullable_to_non_nullable
as Map<String, NotificationPreference>?,behavior: freezed == behavior ? _self.behavior : behavior // ignore: cast_nullable_to_non_nullable
as Behavior?,
  ));
}
/// Create a copy of Settings
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$SettingsLocationCopyWith<$Res>? get location {
    if (_self.location == null) {
    return null;
  }

  return $SettingsLocationCopyWith<$Res>(_self.location!, (value) {
    return _then(_self.copyWith(location: value));
  });
}/// Create a copy of Settings
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$SettingsPrivacyCopyWith<$Res>? get privacy {
    if (_self.privacy == null) {
    return null;
  }

  return $SettingsPrivacyCopyWith<$Res>(_self.privacy!, (value) {
    return _then(_self.copyWith(privacy: value));
  });
}/// Create a copy of Settings
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$BehaviorCopyWith<$Res>? get behavior {
    if (_self.behavior == null) {
    return null;
  }

  return $BehaviorCopyWith<$Res>(_self.behavior!, (value) {
    return _then(_self.copyWith(behavior: value));
  });
}
}


/// Adds pattern-matching-related methods to [Settings].
extension SettingsPatterns on Settings {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Settings value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Settings() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Settings value)  $default,){
final _that = this;
switch (_that) {
case _Settings():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Settings value)?  $default,){
final _that = this;
switch (_that) {
case _Settings() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String? id,  String? unit,  Language? language,  String? bio,  SettingsLocation? location,  String? user,  SettingsPrivacy? privacy,  Map<String, NotificationPreference>? notifications,  Behavior? behavior)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Settings() when $default != null:
return $default(_that.id,_that.unit,_that.language,_that.bio,_that.location,_that.user,_that.privacy,_that.notifications,_that.behavior);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String? id,  String? unit,  Language? language,  String? bio,  SettingsLocation? location,  String? user,  SettingsPrivacy? privacy,  Map<String, NotificationPreference>? notifications,  Behavior? behavior)  $default,) {final _that = this;
switch (_that) {
case _Settings():
return $default(_that.id,_that.unit,_that.language,_that.bio,_that.location,_that.user,_that.privacy,_that.notifications,_that.behavior);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String? id,  String? unit,  Language? language,  String? bio,  SettingsLocation? location,  String? user,  SettingsPrivacy? privacy,  Map<String, NotificationPreference>? notifications,  Behavior? behavior)?  $default,) {final _that = this;
switch (_that) {
case _Settings() when $default != null:
return $default(_that.id,_that.unit,_that.language,_that.bio,_that.location,_that.user,_that.privacy,_that.notifications,_that.behavior);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Settings implements Settings {
  const _Settings({this.id, this.unit, this.language, this.bio, this.location, this.user, this.privacy, final  Map<String, NotificationPreference>? notifications, this.behavior}): _notifications = notifications;
  factory _Settings.fromJson(Map<String, dynamic> json) => _$SettingsFromJson(json);

@override final  String? id;
@override final  String? unit;
@override final  Language? language;
@override final  String? bio;
@override final  SettingsLocation? location;
@override final  String? user;
@override final  SettingsPrivacy? privacy;
 final  Map<String, NotificationPreference>? _notifications;
@override Map<String, NotificationPreference>? get notifications {
  final value = _notifications;
  if (value == null) return null;
  if (_notifications is EqualUnmodifiableMapView) return _notifications;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableMapView(value);
}

@override final  Behavior? behavior;

/// Create a copy of Settings
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SettingsCopyWith<_Settings> get copyWith => __$SettingsCopyWithImpl<_Settings>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SettingsToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _Settings&&(identical(other.id, id) || other.id == id)&&(identical(other.unit, unit) || other.unit == unit)&&(identical(other.language, language) || other.language == language)&&(identical(other.bio, bio) || other.bio == bio)&&(identical(other.location, location) || other.location == location)&&(identical(other.user, user) || other.user == user)&&(identical(other.privacy, privacy) || other.privacy == privacy)&&const DeepCollectionEquality().equals(other._notifications, _notifications)&&(identical(other.behavior, behavior) || other.behavior == behavior));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,unit,language,bio,location,user,privacy,const DeepCollectionEquality().hash(_notifications),behavior);

@override
String toString() {
  return 'Settings(id: $id, unit: $unit, language: $language, bio: $bio, location: $location, user: $user, privacy: $privacy, notifications: $notifications, behavior: $behavior)';
}


}

/// @nodoc
abstract mixin class _$SettingsCopyWith<$Res> implements $SettingsCopyWith<$Res> {
  factory _$SettingsCopyWith(_Settings value, $Res Function(_Settings) _then) = __$SettingsCopyWithImpl;
@override @useResult
$Res call({
 String? id, String? unit, Language? language, String? bio, SettingsLocation? location, String? user, SettingsPrivacy? privacy, Map<String, NotificationPreference>? notifications, Behavior? behavior
});


@override $SettingsLocationCopyWith<$Res>? get location;@override $SettingsPrivacyCopyWith<$Res>? get privacy;@override $BehaviorCopyWith<$Res>? get behavior;

}
/// @nodoc
class __$SettingsCopyWithImpl<$Res>
    implements _$SettingsCopyWith<$Res> {
  __$SettingsCopyWithImpl(this._self, this._then);

  final _Settings _self;
  final $Res Function(_Settings) _then;

/// Create a copy of Settings
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = freezed,Object? unit = freezed,Object? language = freezed,Object? bio = freezed,Object? location = freezed,Object? user = freezed,Object? privacy = freezed,Object? notifications = freezed,Object? behavior = freezed,}) {
  return _then(_Settings(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,unit: freezed == unit ? _self.unit : unit // ignore: cast_nullable_to_non_nullable
as String?,language: freezed == language ? _self.language : language // ignore: cast_nullable_to_non_nullable
as Language?,bio: freezed == bio ? _self.bio : bio // ignore: cast_nullable_to_non_nullable
as String?,location: freezed == location ? _self.location : location // ignore: cast_nullable_to_non_nullable
as SettingsLocation?,user: freezed == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as String?,privacy: freezed == privacy ? _self.privacy : privacy // ignore: cast_nullable_to_non_nullable
as SettingsPrivacy?,notifications: freezed == notifications ? _self._notifications : notifications // ignore: cast_nullable_to_non_nullable
as Map<String, NotificationPreference>?,behavior: freezed == behavior ? _self.behavior : behavior // ignore: cast_nullable_to_non_nullable
as Behavior?,
  ));
}

/// Create a copy of Settings
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$SettingsLocationCopyWith<$Res>? get location {
    if (_self.location == null) {
    return null;
  }

  return $SettingsLocationCopyWith<$Res>(_self.location!, (value) {
    return _then(_self.copyWith(location: value));
  });
}/// Create a copy of Settings
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$SettingsPrivacyCopyWith<$Res>? get privacy {
    if (_self.privacy == null) {
    return null;
  }

  return $SettingsPrivacyCopyWith<$Res>(_self.privacy!, (value) {
    return _then(_self.copyWith(privacy: value));
  });
}/// Create a copy of Settings
/// with the given fields replaced by the non-null parameter values.
@override
@pragma('vm:prefer-inline')
$BehaviorCopyWith<$Res>? get behavior {
    if (_self.behavior == null) {
    return null;
  }

  return $BehaviorCopyWith<$Res>(_self.behavior!, (value) {
    return _then(_self.copyWith(behavior: value));
  });
}
}

// dart format on
