/// Escaping for values interpolated into PocketBase filter expressions.
///
/// PocketBase evaluates a client-supplied `filter` in the same query as the
/// collection's own `ListRule` — that is by design, and the only thing
/// stopping a crafted filter from widening a result set is the caller
/// building it correctly. So a value that reaches a filter string has to be
/// escaped at the point of interpolation; there is no layer below that will
/// do it. Dio does not help either: it percent-encodes for transport, which
/// leaves the filter's *meaning* untouched.
///
/// Concretely, `name~'$value'` with a value of `x' || id!='` yields
/// `name~'x' || id!=''` — a different, much broader query than intended.
///
/// Wrap every interpolated value in [escapePbFilterValue] and pass the
/// finished expression through `queryParameters` rather than concatenating it
/// into the path.
library;

/// Escapes [value] for use inside a single-quoted PocketBase filter literal.
///
/// Backslashes first, then quotes — the other order would double-escape the
/// backslashes this function itself introduces.
String escapePbFilterValue(String value) =>
    value.replaceAll('\\', r'\\').replaceAll("'", r"\'");
