// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'subcategory.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$Subcategory {

 String get id; String get category; String get name;@JsonKey(name: 'short_name') String? get shortName; String? get icon;@JsonKey(name: 'badge_icon') String? get badgeIcon; Map<String, CategoryTranslation>? get translations;
/// Create a copy of Subcategory
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SubcategoryCopyWith<Subcategory> get copyWith => _$SubcategoryCopyWithImpl<Subcategory>(this as Subcategory, _$identity);

  /// Serializes this Subcategory to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Subcategory&&(identical(other.id, id) || other.id == id)&&(identical(other.category, category) || other.category == category)&&(identical(other.name, name) || other.name == name)&&(identical(other.shortName, shortName) || other.shortName == shortName)&&(identical(other.icon, icon) || other.icon == icon)&&(identical(other.badgeIcon, badgeIcon) || other.badgeIcon == badgeIcon)&&const DeepCollectionEquality().equals(other.translations, translations));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,category,name,shortName,icon,badgeIcon,const DeepCollectionEquality().hash(translations));

@override
String toString() {
  return 'Subcategory(id: $id, category: $category, name: $name, shortName: $shortName, icon: $icon, badgeIcon: $badgeIcon, translations: $translations)';
}


}

/// @nodoc
abstract mixin class $SubcategoryCopyWith<$Res>  {
  factory $SubcategoryCopyWith(Subcategory value, $Res Function(Subcategory) _then) = _$SubcategoryCopyWithImpl;
@useResult
$Res call({
 String id, String category, String name,@JsonKey(name: 'short_name') String? shortName, String? icon,@JsonKey(name: 'badge_icon') String? badgeIcon, Map<String, CategoryTranslation>? translations
});




}
/// @nodoc
class _$SubcategoryCopyWithImpl<$Res>
    implements $SubcategoryCopyWith<$Res> {
  _$SubcategoryCopyWithImpl(this._self, this._then);

  final Subcategory _self;
  final $Res Function(Subcategory) _then;

/// Create a copy of Subcategory
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? category = null,Object? name = null,Object? shortName = freezed,Object? icon = freezed,Object? badgeIcon = freezed,Object? translations = freezed,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,category: null == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,shortName: freezed == shortName ? _self.shortName : shortName // ignore: cast_nullable_to_non_nullable
as String?,icon: freezed == icon ? _self.icon : icon // ignore: cast_nullable_to_non_nullable
as String?,badgeIcon: freezed == badgeIcon ? _self.badgeIcon : badgeIcon // ignore: cast_nullable_to_non_nullable
as String?,translations: freezed == translations ? _self.translations : translations // ignore: cast_nullable_to_non_nullable
as Map<String, CategoryTranslation>?,
  ));
}

}


/// Adds pattern-matching-related methods to [Subcategory].
extension SubcategoryPatterns on Subcategory {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Subcategory value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Subcategory() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Subcategory value)  $default,){
final _that = this;
switch (_that) {
case _Subcategory():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Subcategory value)?  $default,){
final _that = this;
switch (_that) {
case _Subcategory() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String category,  String name, @JsonKey(name: 'short_name')  String? shortName,  String? icon, @JsonKey(name: 'badge_icon')  String? badgeIcon,  Map<String, CategoryTranslation>? translations)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Subcategory() when $default != null:
return $default(_that.id,_that.category,_that.name,_that.shortName,_that.icon,_that.badgeIcon,_that.translations);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String category,  String name, @JsonKey(name: 'short_name')  String? shortName,  String? icon, @JsonKey(name: 'badge_icon')  String? badgeIcon,  Map<String, CategoryTranslation>? translations)  $default,) {final _that = this;
switch (_that) {
case _Subcategory():
return $default(_that.id,_that.category,_that.name,_that.shortName,_that.icon,_that.badgeIcon,_that.translations);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String category,  String name, @JsonKey(name: 'short_name')  String? shortName,  String? icon, @JsonKey(name: 'badge_icon')  String? badgeIcon,  Map<String, CategoryTranslation>? translations)?  $default,) {final _that = this;
switch (_that) {
case _Subcategory() when $default != null:
return $default(_that.id,_that.category,_that.name,_that.shortName,_that.icon,_that.badgeIcon,_that.translations);case _:
  return null;

}
}

}

/// @nodoc

@JsonSerializable(explicitToJson: true)
class _Subcategory implements Subcategory {
  const _Subcategory({required this.id, required this.category, required this.name, @JsonKey(name: 'short_name') this.shortName, this.icon, @JsonKey(name: 'badge_icon') this.badgeIcon, final  Map<String, CategoryTranslation>? translations}): _translations = translations;
  factory _Subcategory.fromJson(Map<String, dynamic> json) => _$SubcategoryFromJson(json);

@override final  String id;
@override final  String category;
@override final  String name;
@override@JsonKey(name: 'short_name') final  String? shortName;
@override final  String? icon;
@override@JsonKey(name: 'badge_icon') final  String? badgeIcon;
 final  Map<String, CategoryTranslation>? _translations;
@override Map<String, CategoryTranslation>? get translations {
  final value = _translations;
  if (value == null) return null;
  if (_translations is EqualUnmodifiableMapView) return _translations;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableMapView(value);
}


/// Create a copy of Subcategory
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SubcategoryCopyWith<_Subcategory> get copyWith => __$SubcategoryCopyWithImpl<_Subcategory>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SubcategoryToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _Subcategory&&(identical(other.id, id) || other.id == id)&&(identical(other.category, category) || other.category == category)&&(identical(other.name, name) || other.name == name)&&(identical(other.shortName, shortName) || other.shortName == shortName)&&(identical(other.icon, icon) || other.icon == icon)&&(identical(other.badgeIcon, badgeIcon) || other.badgeIcon == badgeIcon)&&const DeepCollectionEquality().equals(other._translations, _translations));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,category,name,shortName,icon,badgeIcon,const DeepCollectionEquality().hash(_translations));

@override
String toString() {
  return 'Subcategory(id: $id, category: $category, name: $name, shortName: $shortName, icon: $icon, badgeIcon: $badgeIcon, translations: $translations)';
}


}

/// @nodoc
abstract mixin class _$SubcategoryCopyWith<$Res> implements $SubcategoryCopyWith<$Res> {
  factory _$SubcategoryCopyWith(_Subcategory value, $Res Function(_Subcategory) _then) = __$SubcategoryCopyWithImpl;
@override @useResult
$Res call({
 String id, String category, String name,@JsonKey(name: 'short_name') String? shortName, String? icon,@JsonKey(name: 'badge_icon') String? badgeIcon, Map<String, CategoryTranslation>? translations
});




}
/// @nodoc
class __$SubcategoryCopyWithImpl<$Res>
    implements _$SubcategoryCopyWith<$Res> {
  __$SubcategoryCopyWithImpl(this._self, this._then);

  final _Subcategory _self;
  final $Res Function(_Subcategory) _then;

/// Create a copy of Subcategory
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? category = null,Object? name = null,Object? shortName = freezed,Object? icon = freezed,Object? badgeIcon = freezed,Object? translations = freezed,}) {
  return _then(_Subcategory(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,category: null == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,shortName: freezed == shortName ? _self.shortName : shortName // ignore: cast_nullable_to_non_nullable
as String?,icon: freezed == icon ? _self.icon : icon // ignore: cast_nullable_to_non_nullable
as String?,badgeIcon: freezed == badgeIcon ? _self.badgeIcon : badgeIcon // ignore: cast_nullable_to_non_nullable
as String?,translations: freezed == translations ? _self._translations : translations // ignore: cast_nullable_to_non_nullable
as Map<String, CategoryTranslation>?,
  ));
}


}

// dart format on
