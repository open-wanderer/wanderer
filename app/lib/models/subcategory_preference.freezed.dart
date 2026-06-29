// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'subcategory_preference.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$SubcategoryPreference {

 String? get id; String get user; String get subcategory; bool? get visible; int? get priority;
/// Create a copy of SubcategoryPreference
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$SubcategoryPreferenceCopyWith<SubcategoryPreference> get copyWith => _$SubcategoryPreferenceCopyWithImpl<SubcategoryPreference>(this as SubcategoryPreference, _$identity);

  /// Serializes this SubcategoryPreference to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is SubcategoryPreference&&(identical(other.id, id) || other.id == id)&&(identical(other.user, user) || other.user == user)&&(identical(other.subcategory, subcategory) || other.subcategory == subcategory)&&(identical(other.visible, visible) || other.visible == visible)&&(identical(other.priority, priority) || other.priority == priority));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,user,subcategory,visible,priority);

@override
String toString() {
  return 'SubcategoryPreference(id: $id, user: $user, subcategory: $subcategory, visible: $visible, priority: $priority)';
}


}

/// @nodoc
abstract mixin class $SubcategoryPreferenceCopyWith<$Res>  {
  factory $SubcategoryPreferenceCopyWith(SubcategoryPreference value, $Res Function(SubcategoryPreference) _then) = _$SubcategoryPreferenceCopyWithImpl;
@useResult
$Res call({
 String? id, String user, String subcategory, bool? visible, int? priority
});




}
/// @nodoc
class _$SubcategoryPreferenceCopyWithImpl<$Res>
    implements $SubcategoryPreferenceCopyWith<$Res> {
  _$SubcategoryPreferenceCopyWithImpl(this._self, this._then);

  final SubcategoryPreference _self;
  final $Res Function(SubcategoryPreference) _then;

/// Create a copy of SubcategoryPreference
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = freezed,Object? user = null,Object? subcategory = null,Object? visible = freezed,Object? priority = freezed,}) {
  return _then(_self.copyWith(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,user: null == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as String,subcategory: null == subcategory ? _self.subcategory : subcategory // ignore: cast_nullable_to_non_nullable
as String,visible: freezed == visible ? _self.visible : visible // ignore: cast_nullable_to_non_nullable
as bool?,priority: freezed == priority ? _self.priority : priority // ignore: cast_nullable_to_non_nullable
as int?,
  ));
}

}


/// Adds pattern-matching-related methods to [SubcategoryPreference].
extension SubcategoryPreferencePatterns on SubcategoryPreference {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _SubcategoryPreference value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _SubcategoryPreference() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _SubcategoryPreference value)  $default,){
final _that = this;
switch (_that) {
case _SubcategoryPreference():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _SubcategoryPreference value)?  $default,){
final _that = this;
switch (_that) {
case _SubcategoryPreference() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String? id,  String user,  String subcategory,  bool? visible,  int? priority)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _SubcategoryPreference() when $default != null:
return $default(_that.id,_that.user,_that.subcategory,_that.visible,_that.priority);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String? id,  String user,  String subcategory,  bool? visible,  int? priority)  $default,) {final _that = this;
switch (_that) {
case _SubcategoryPreference():
return $default(_that.id,_that.user,_that.subcategory,_that.visible,_that.priority);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String? id,  String user,  String subcategory,  bool? visible,  int? priority)?  $default,) {final _that = this;
switch (_that) {
case _SubcategoryPreference() when $default != null:
return $default(_that.id,_that.user,_that.subcategory,_that.visible,_that.priority);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _SubcategoryPreference implements SubcategoryPreference {
  const _SubcategoryPreference({this.id, required this.user, required this.subcategory, this.visible, this.priority});
  factory _SubcategoryPreference.fromJson(Map<String, dynamic> json) => _$SubcategoryPreferenceFromJson(json);

@override final  String? id;
@override final  String user;
@override final  String subcategory;
@override final  bool? visible;
@override final  int? priority;

/// Create a copy of SubcategoryPreference
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$SubcategoryPreferenceCopyWith<_SubcategoryPreference> get copyWith => __$SubcategoryPreferenceCopyWithImpl<_SubcategoryPreference>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$SubcategoryPreferenceToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _SubcategoryPreference&&(identical(other.id, id) || other.id == id)&&(identical(other.user, user) || other.user == user)&&(identical(other.subcategory, subcategory) || other.subcategory == subcategory)&&(identical(other.visible, visible) || other.visible == visible)&&(identical(other.priority, priority) || other.priority == priority));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,user,subcategory,visible,priority);

@override
String toString() {
  return 'SubcategoryPreference(id: $id, user: $user, subcategory: $subcategory, visible: $visible, priority: $priority)';
}


}

/// @nodoc
abstract mixin class _$SubcategoryPreferenceCopyWith<$Res> implements $SubcategoryPreferenceCopyWith<$Res> {
  factory _$SubcategoryPreferenceCopyWith(_SubcategoryPreference value, $Res Function(_SubcategoryPreference) _then) = __$SubcategoryPreferenceCopyWithImpl;
@override @useResult
$Res call({
 String? id, String user, String subcategory, bool? visible, int? priority
});




}
/// @nodoc
class __$SubcategoryPreferenceCopyWithImpl<$Res>
    implements _$SubcategoryPreferenceCopyWith<$Res> {
  __$SubcategoryPreferenceCopyWithImpl(this._self, this._then);

  final _SubcategoryPreference _self;
  final $Res Function(_SubcategoryPreference) _then;

/// Create a copy of SubcategoryPreference
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = freezed,Object? user = null,Object? subcategory = null,Object? visible = freezed,Object? priority = freezed,}) {
  return _then(_SubcategoryPreference(
id: freezed == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String?,user: null == user ? _self.user : user // ignore: cast_nullable_to_non_nullable
as String,subcategory: null == subcategory ? _self.subcategory : subcategory // ignore: cast_nullable_to_non_nullable
as String,visible: freezed == visible ? _self.visible : visible // ignore: cast_nullable_to_non_nullable
as bool?,priority: freezed == priority ? _self.priority : priority // ignore: cast_nullable_to_non_nullable
as int?,
  ));
}


}

// dart format on
