// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'server_instance.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$ServerInstance {

 String get name; String get url; String get description; String get image; List<String> get region; List<String> get language; List<String> get category;
/// Create a copy of ServerInstance
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$ServerInstanceCopyWith<ServerInstance> get copyWith => _$ServerInstanceCopyWithImpl<ServerInstance>(this as ServerInstance, _$identity);

  /// Serializes this ServerInstance to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is ServerInstance&&(identical(other.name, name) || other.name == name)&&(identical(other.url, url) || other.url == url)&&(identical(other.description, description) || other.description == description)&&(identical(other.image, image) || other.image == image)&&const DeepCollectionEquality().equals(other.region, region)&&const DeepCollectionEquality().equals(other.language, language)&&const DeepCollectionEquality().equals(other.category, category));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,name,url,description,image,const DeepCollectionEquality().hash(region),const DeepCollectionEquality().hash(language),const DeepCollectionEquality().hash(category));

@override
String toString() {
  return 'ServerInstance(name: $name, url: $url, description: $description, image: $image, region: $region, language: $language, category: $category)';
}


}

/// @nodoc
abstract mixin class $ServerInstanceCopyWith<$Res>  {
  factory $ServerInstanceCopyWith(ServerInstance value, $Res Function(ServerInstance) _then) = _$ServerInstanceCopyWithImpl;
@useResult
$Res call({
 String name, String url, String description, String image, List<String> region, List<String> language, List<String> category
});




}
/// @nodoc
class _$ServerInstanceCopyWithImpl<$Res>
    implements $ServerInstanceCopyWith<$Res> {
  _$ServerInstanceCopyWithImpl(this._self, this._then);

  final ServerInstance _self;
  final $Res Function(ServerInstance) _then;

/// Create a copy of ServerInstance
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? name = null,Object? url = null,Object? description = null,Object? image = null,Object? region = null,Object? language = null,Object? category = null,}) {
  return _then(_self.copyWith(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,url: null == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String,description: null == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String,image: null == image ? _self.image : image // ignore: cast_nullable_to_non_nullable
as String,region: null == region ? _self.region : region // ignore: cast_nullable_to_non_nullable
as List<String>,language: null == language ? _self.language : language // ignore: cast_nullable_to_non_nullable
as List<String>,category: null == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as List<String>,
  ));
}

}


/// Adds pattern-matching-related methods to [ServerInstance].
extension ServerInstancePatterns on ServerInstance {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _ServerInstance value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _ServerInstance() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _ServerInstance value)  $default,){
final _that = this;
switch (_that) {
case _ServerInstance():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _ServerInstance value)?  $default,){
final _that = this;
switch (_that) {
case _ServerInstance() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String name,  String url,  String description,  String image,  List<String> region,  List<String> language,  List<String> category)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _ServerInstance() when $default != null:
return $default(_that.name,_that.url,_that.description,_that.image,_that.region,_that.language,_that.category);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String name,  String url,  String description,  String image,  List<String> region,  List<String> language,  List<String> category)  $default,) {final _that = this;
switch (_that) {
case _ServerInstance():
return $default(_that.name,_that.url,_that.description,_that.image,_that.region,_that.language,_that.category);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String name,  String url,  String description,  String image,  List<String> region,  List<String> language,  List<String> category)?  $default,) {final _that = this;
switch (_that) {
case _ServerInstance() when $default != null:
return $default(_that.name,_that.url,_that.description,_that.image,_that.region,_that.language,_that.category);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _ServerInstance implements ServerInstance {
  const _ServerInstance({required this.name, required this.url, required this.description, required this.image, final  List<String> region = const [], final  List<String> language = const [], final  List<String> category = const []}): _region = region,_language = language,_category = category;
  factory _ServerInstance.fromJson(Map<String, dynamic> json) => _$ServerInstanceFromJson(json);

@override final  String name;
@override final  String url;
@override final  String description;
@override final  String image;
 final  List<String> _region;
@override@JsonKey() List<String> get region {
  if (_region is EqualUnmodifiableListView) return _region;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_region);
}

 final  List<String> _language;
@override@JsonKey() List<String> get language {
  if (_language is EqualUnmodifiableListView) return _language;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_language);
}

 final  List<String> _category;
@override@JsonKey() List<String> get category {
  if (_category is EqualUnmodifiableListView) return _category;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_category);
}


/// Create a copy of ServerInstance
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$ServerInstanceCopyWith<_ServerInstance> get copyWith => __$ServerInstanceCopyWithImpl<_ServerInstance>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$ServerInstanceToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _ServerInstance&&(identical(other.name, name) || other.name == name)&&(identical(other.url, url) || other.url == url)&&(identical(other.description, description) || other.description == description)&&(identical(other.image, image) || other.image == image)&&const DeepCollectionEquality().equals(other._region, _region)&&const DeepCollectionEquality().equals(other._language, _language)&&const DeepCollectionEquality().equals(other._category, _category));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,name,url,description,image,const DeepCollectionEquality().hash(_region),const DeepCollectionEquality().hash(_language),const DeepCollectionEquality().hash(_category));

@override
String toString() {
  return 'ServerInstance(name: $name, url: $url, description: $description, image: $image, region: $region, language: $language, category: $category)';
}


}

/// @nodoc
abstract mixin class _$ServerInstanceCopyWith<$Res> implements $ServerInstanceCopyWith<$Res> {
  factory _$ServerInstanceCopyWith(_ServerInstance value, $Res Function(_ServerInstance) _then) = __$ServerInstanceCopyWithImpl;
@override @useResult
$Res call({
 String name, String url, String description, String image, List<String> region, List<String> language, List<String> category
});




}
/// @nodoc
class __$ServerInstanceCopyWithImpl<$Res>
    implements _$ServerInstanceCopyWith<$Res> {
  __$ServerInstanceCopyWithImpl(this._self, this._then);

  final _ServerInstance _self;
  final $Res Function(_ServerInstance) _then;

/// Create a copy of ServerInstance
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? name = null,Object? url = null,Object? description = null,Object? image = null,Object? region = null,Object? language = null,Object? category = null,}) {
  return _then(_ServerInstance(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,url: null == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String,description: null == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String,image: null == image ? _self.image : image // ignore: cast_nullable_to_non_nullable
as String,region: null == region ? _self._region : region // ignore: cast_nullable_to_non_nullable
as List<String>,language: null == language ? _self._language : language // ignore: cast_nullable_to_non_nullable
as List<String>,category: null == category ? _self._category : category // ignore: cast_nullable_to_non_nullable
as List<String>,
  ));
}


}

// dart format on
