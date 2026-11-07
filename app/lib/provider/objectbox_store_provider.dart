import 'package:objectbox/objectbox.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'objectbox_store_provider.g.dart';

@Riverpod(keepAlive: true)
class ObjectBox extends _$ObjectBox {
  @override
  Store build() {
    // This will be overridden in main.dart
    throw UnimplementedError();
  }
}
