// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'category_preference.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$CategoryPreference {

 String? get id; String get user; String get category; bool? get visible; int? get priority;
/// Create a copy of CategoryPreference
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$CategoryPreferenceCopyWith<CategoryPreference> get copyWith => _$CategoryPreferenceCopyWithImpl<CategoryPreference>(this as CategoryPreference, _$identity);

  /// Serializes this CategoryPreference to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is CategoryPreference&&(identical(other.id, id) || other.id == id)&&(identical(other.user, user) || other.user == user)&&(identical(other.category, category) || other.category == category)&&(identical(other.visible, visible) || other.visible == visible)&&(identical(other.priority, priority) || other.priority == priority));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,user,category,visible,priority);

@override
String toString() {
  return 'CategoryPreference(id: $id, user: $user, category: $category, visible: $visible, priority: $priority)';
}


}

/// @nodoc
abstract mixin class $CategoryPreferenceCopyWith<$Res>  {
  factory $CategoryPreferenceCopyWith(CategoryPreference value, $Res Function(CategoryPreference) _then) = _$CategoryPreferenceCopyWithImpl;
@useResult
$Res call({
 String? id, String user, String category, bool? visible, int? priority
});




}
/// @nodoc
class _$CategoryPreferenceCopyWithImpl<$Res>
    implements $CategoryPreferenceCopyWith<$Res> {
  _$CategoryPreferenceCopyWithImpl(this._self, this._then);

  final CategoryPreference _self;
  final $Res Function(CategoryPreference) _then;

/// Create a copy of CategoryPreference
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = freezed,Object? user = null,Object? category = null,Object? visible = freezed,Object? priority = freezed,}) {
  return _then(_self.copyWith(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,user: null == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as String,category: null == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as String,visible: freezed == visible ? _self.visible : visible // ignore: cast_nullable_to_non_nullable
as bool?,priority: freezed == priority ? _self.priority : priority // ignore: cast_nullable_to_non_nullable
as int?,
  ));
}

}


/// Adds pattern-matching-related methods to [CategoryPreference].
extension CategoryPreferencePatterns on CategoryPreference {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _CategoryPreference value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _CategoryPreference() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _CategoryPreference value)  $default,){
final _that = this;
switch (_that) {
case _CategoryPreference():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _CategoryPreference value)?  $default,){
final _that = this;
switch (_that) {
case _CategoryPreference() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String? id,  String user,  String category,  bool? visible,  int? priority)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _CategoryPreference() when $default != null:
return $default(_that.id,_that.user,_that.category,_that.visible,_that.priority);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String? id,  String user,  String category,  bool? visible,  int? priority)  $default,) {final _that = this;
switch (_that) {
case _CategoryPreference():
return $default(_that.id,_that.user,_that.category,_that.visible,_that.priority);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String? id,  String user,  String category,  bool? visible,  int? priority)?  $default,) {final _that = this;
switch (_that) {
case _CategoryPreference() when $default != null:
return $default(_that.id,_that.user,_that.category,_that.visible,_that.priority);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _CategoryPreference implements CategoryPreference {
  const _CategoryPreference({this.id, required this.user, required this.category, this.visible, this.priority});
  factory _CategoryPreference.fromJson(Map<String, dynamic> json) => _$CategoryPreferenceFromJson(json);

@override final  String? id;
@override final  String user;
@override final  String category;
@override final  bool? visible;
@override final  int? priority;

/// Create a copy of CategoryPreference
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$CategoryPreferenceCopyWith<_CategoryPreference> get copyWith => __$CategoryPreferenceCopyWithImpl<_CategoryPreference>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$CategoryPreferenceToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _CategoryPreference&&(identical(other.id, id) || other.id == id)&&(identical(other.user, user) || other.user == user)&&(identical(other.category, category) || other.category == category)&&(identical(other.visible, visible) || other.visible == visible)&&(identical(other.priority, priority) || other.priority == priority));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,user,category,visible,priority);

@override
String toString() {
  return 'CategoryPreference(id: $id, user: $user, category: $category, visible: $visible, priority: $priority)';
}


}

/// @nodoc
abstract mixin class _$CategoryPreferenceCopyWith<$Res> implements $CategoryPreferenceCopyWith<$Res> {
  factory _$CategoryPreferenceCopyWith(_CategoryPreference value, $Res Function(_CategoryPreference) _then) = __$CategoryPreferenceCopyWithImpl;
@override @useResult
$Res call({
 String? id, String user, String category, bool? visible, int? priority
});




}
/// @nodoc
class __$CategoryPreferenceCopyWithImpl<$Res>
    implements _$CategoryPreferenceCopyWith<$Res> {
  __$CategoryPreferenceCopyWithImpl(this._self, this._then);

  final _CategoryPreference _self;
  final $Res Function(_CategoryPreference) _then;

/// Create a copy of CategoryPreference
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = freezed,Object? user = null,Object? category = null,Object? visible = freezed,Object? priority = freezed,}) {
  return _then(_CategoryPreference(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,user: null == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as String,category: null == category ? _self.category : category // ignore: cast_nullable_to_non_nullable
as String,visible: freezed == visible ? _self.visible : visible // ignore: cast_nullable_to_non_nullable
as bool?,priority: freezed == priority ? _self.priority : priority // ignore: cast_nullable_to_non_nullable
as int?,
  ));
}


}

// dart format on
