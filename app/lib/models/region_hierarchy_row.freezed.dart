// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'region_hierarchy_row.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$RegionHierarchyRow {

 String get id; String get name; RegionNodeKind get kind;/// `""` for roots, never null — matches Go's `r.GetString("parent")`.
 String get parent; String get path; int get depth;/// Defaults to `0` (Pitfall 4): deliberate forward-compat guard so a
/// row fetched before the backend's `sort_order` field ships (or from
/// an out-of-order rollout) still parses instead of throwing.
@JsonKey(name: 'sort_order') int get sortOrder;
/// Create a copy of RegionHierarchyRow
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$RegionHierarchyRowCopyWith<RegionHierarchyRow> get copyWith => _$RegionHierarchyRowCopyWithImpl<RegionHierarchyRow>(this as RegionHierarchyRow, _$identity);

  /// Serializes this RegionHierarchyRow to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is RegionHierarchyRow&&(identical(other.id, id) || other.id == id)&&(identical(other.name, name) || other.name == name)&&(identical(other.kind, kind) || other.kind == kind)&&(identical(other.parent, parent) || other.parent == parent)&&(identical(other.path, path) || other.path == path)&&(identical(other.depth, depth) || other.depth == depth)&&(identical(other.sortOrder, sortOrder) || other.sortOrder == sortOrder));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,name,kind,parent,path,depth,sortOrder);

@override
String toString() {
  return 'RegionHierarchyRow(id: $id, name: $name, kind: $kind, parent: $parent, path: $path, depth: $depth, sortOrder: $sortOrder)';
}


}

/// @nodoc
abstract mixin class $RegionHierarchyRowCopyWith<$Res>  {
  factory $RegionHierarchyRowCopyWith(RegionHierarchyRow value, $Res Function(RegionHierarchyRow) _then) = _$RegionHierarchyRowCopyWithImpl;
@useResult
$Res call({
 String id, String name, RegionNodeKind kind, String parent, String path, int depth,@JsonKey(name: 'sort_order') int sortOrder
});




}
/// @nodoc
class _$RegionHierarchyRowCopyWithImpl<$Res>
    implements $RegionHierarchyRowCopyWith<$Res> {
  _$RegionHierarchyRowCopyWithImpl(this._self, this._then);

  final RegionHierarchyRow _self;
  final $Res Function(RegionHierarchyRow) _then;

/// Create a copy of RegionHierarchyRow
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? name = null,Object? kind = null,Object? parent = null,Object? path = null,Object? depth = null,Object? sortOrder = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,kind: null == kind ? _self.kind : kind // ignore: cast_nullable_to_non_nullable
as RegionNodeKind,parent: null == parent ? _self.parent : parent // ignore: cast_nullable_to_non_nullable
as String,path: null == path ? _self.path : path // ignore: cast_nullable_to_non_nullable
as String,depth: null == depth ? _self.depth : depth // ignore: cast_nullable_to_non_nullable
as int,sortOrder: null == sortOrder ? _self.sortOrder : sortOrder // ignore: cast_nullable_to_non_nullable
as int,
  ));
}

}


/// Adds pattern-matching-related methods to [RegionHierarchyRow].
extension RegionHierarchyRowPatterns on RegionHierarchyRow {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _RegionHierarchyRow value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _RegionHierarchyRow() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _RegionHierarchyRow value)  $default,){
final _that = this;
switch (_that) {
case _RegionHierarchyRow():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _RegionHierarchyRow value)?  $default,){
final _that = this;
switch (_that) {
case _RegionHierarchyRow() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id,  String name,  RegionNodeKind kind,  String parent,  String path,  int depth, @JsonKey(name: 'sort_order')  int sortOrder)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _RegionHierarchyRow() when $default != null:
return $default(_that.id,_that.name,_that.kind,_that.parent,_that.path,_that.depth,_that.sortOrder);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id,  String name,  RegionNodeKind kind,  String parent,  String path,  int depth, @JsonKey(name: 'sort_order')  int sortOrder)  $default,) {final _that = this;
switch (_that) {
case _RegionHierarchyRow():
return $default(_that.id,_that.name,_that.kind,_that.parent,_that.path,_that.depth,_that.sortOrder);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id,  String name,  RegionNodeKind kind,  String parent,  String path,  int depth, @JsonKey(name: 'sort_order')  int sortOrder)?  $default,) {final _that = this;
switch (_that) {
case _RegionHierarchyRow() when $default != null:
return $default(_that.id,_that.name,_that.kind,_that.parent,_that.path,_that.depth,_that.sortOrder);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _RegionHierarchyRow implements RegionHierarchyRow {
  const _RegionHierarchyRow({required this.id, required this.name, required this.kind, required this.parent, required this.path, required this.depth, @JsonKey(name: 'sort_order') this.sortOrder = 0});
  factory _RegionHierarchyRow.fromJson(Map<String, dynamic> json) => _$RegionHierarchyRowFromJson(json);

@override final  String id;
@override final  String name;
@override final  RegionNodeKind kind;
/// `""` for roots, never null — matches Go's `r.GetString("parent")`.
@override final  String parent;
@override final  String path;
@override final  int depth;
/// Defaults to `0` (Pitfall 4): deliberate forward-compat guard so a
/// row fetched before the backend's `sort_order` field ships (or from
/// an out-of-order rollout) still parses instead of throwing.
@override@JsonKey(name: 'sort_order') final  int sortOrder;

/// Create a copy of RegionHierarchyRow
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$RegionHierarchyRowCopyWith<_RegionHierarchyRow> get copyWith => __$RegionHierarchyRowCopyWithImpl<_RegionHierarchyRow>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$RegionHierarchyRowToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _RegionHierarchyRow&&(identical(other.id, id) || other.id == id)&&(identical(other.name, name) || other.name == name)&&(identical(other.kind, kind) || other.kind == kind)&&(identical(other.parent, parent) || other.parent == parent)&&(identical(other.path, path) || other.path == path)&&(identical(other.depth, depth) || other.depth == depth)&&(identical(other.sortOrder, sortOrder) || other.sortOrder == sortOrder));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,name,kind,parent,path,depth,sortOrder);

@override
String toString() {
  return 'RegionHierarchyRow(id: $id, name: $name, kind: $kind, parent: $parent, path: $path, depth: $depth, sortOrder: $sortOrder)';
}


}

/// @nodoc
abstract mixin class _$RegionHierarchyRowCopyWith<$Res> implements $RegionHierarchyRowCopyWith<$Res> {
  factory _$RegionHierarchyRowCopyWith(_RegionHierarchyRow value, $Res Function(_RegionHierarchyRow) _then) = __$RegionHierarchyRowCopyWithImpl;
@override @useResult
$Res call({
 String id, String name, RegionNodeKind kind, String parent, String path, int depth,@JsonKey(name: 'sort_order') int sortOrder
});




}
/// @nodoc
class __$RegionHierarchyRowCopyWithImpl<$Res>
    implements _$RegionHierarchyRowCopyWith<$Res> {
  __$RegionHierarchyRowCopyWithImpl(this._self, this._then);

  final _RegionHierarchyRow _self;
  final $Res Function(_RegionHierarchyRow) _then;

/// Create a copy of RegionHierarchyRow
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? name = null,Object? kind = null,Object? parent = null,Object? path = null,Object? depth = null,Object? sortOrder = null,}) {
  return _then(_RegionHierarchyRow(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,kind: null == kind ? _self.kind : kind // ignore: cast_nullable_to_non_nullable
as RegionNodeKind,parent: null == parent ? _self.parent : parent // ignore: cast_nullable_to_non_nullable
as String,path: null == path ? _self.path : path // ignore: cast_nullable_to_non_nullable
as String,depth: null == depth ? _self.depth : depth // ignore: cast_nullable_to_non_nullable
as int,sortOrder: null == sortOrder ? _self.sortOrder : sortOrder // ignore: cast_nullable_to_non_nullable
as int,
  ));
}


}

// dart format on
