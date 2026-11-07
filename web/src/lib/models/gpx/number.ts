/**
 * Decimal numbers as Dart's `double.tryParse` reads them in the app's GPX
 * reader: an optional sign, digits with an optional fraction, an optional
 * exponent. `Number()` also accepts hex (`"0x10"` is 16), binary and octal
 * literals, which made the web read values the app rejects.
 */
const DECIMAL_NUMBER = /^[+-]?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?$/;

export function isDecimalNumber(raw: string): boolean {
  return DECIMAL_NUMBER.test(raw.trim());
}

/**
 * Coerces a GPX coordinate or elevation to a finite number, or `undefined`
 * when the value is absent or not a finite decimal number.
 */
export function parseGpxNumber(raw: unknown): number | undefined {
  if (typeof raw === 'number') {
    return Number.isFinite(raw) ? raw : undefined;
  }
  if (typeof raw !== 'string' || !isDecimalNumber(raw)) {
    return undefined;
  }
  const parsed = Number(raw.trim());
  return Number.isFinite(parsed) ? parsed : undefined;
}
