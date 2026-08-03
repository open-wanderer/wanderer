import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/pb_filter.dart';

void main() {
  group('escapePbFilterValue', () {
    test('leaves an ordinary value untouched', () {
      expect(escapePbFilterValue('alpine'), 'alpine');
      expect(escapePbFilterValue(''), '');
      expect(escapePbFilterValue('Grüße 山'), 'Grüße 山');
    });

    test('escapes the single quote that would close the literal early', () {
      // The attack this exists for: `name~'x' || id!=''` is a far broader
      // query than the `name~'…'` the caller wrote.
      expect(escapePbFilterValue("x' || id!='"), r"x\' || id!=\'");
    });

    test('escapes backslashes so they cannot smuggle a quote through', () {
      // A lone trailing backslash would otherwise escape the closing quote
      // the caller appends, reopening the literal.
      expect(escapePbFilterValue(r'x\'), r'x\\');
      expect(escapePbFilterValue(r"x\'"), r"x\\\'");
    });

    test('escapes backslashes before quotes, not after', () {
      // Wrong order double-escapes the backslashes this function itself
      // introduces, yielding r"a\\'" — the quote unescaped again.
      expect(escapePbFilterValue("a'"), r"a\'");
      expect(escapePbFilterValue(r'a\b'), r'a\\b');
    });

    test('is idempotent-safe: escaping twice does not corrupt the value', () {
      // Not a supported call pattern, but a double-escape must stay parseable
      // rather than silently changing meaning.
      final once = escapePbFilterValue("o'brien");
      expect(once, r"o\'brien");
      expect(escapePbFilterValue(once), r"o\\\'brien");
    });
  });
}
