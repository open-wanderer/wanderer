// Accommodates 99 default-sized photos, a GPX file and multipart headers.
export const UPLOAD_BODY_LIMIT = 2 * 1024 * 1024 * 1024;

/** @param {string | undefined} value */
export function uploadBodyLimit(value) {
    if (!value) return UPLOAD_BODY_LIMIT;
    if (value === "Infinity") return Infinity;
    const suffix = value.at(-1)?.toUpperCase();
    const multiplier = { K: 1024, M: 1024 ** 2, G: 1024 ** 3 }[suffix ?? ""] ?? 1;
    const limit = Number(multiplier === 1 ? value : value.slice(0, -1)) * multiplier;
    if (Number.isSafeInteger(limit) && limit > 0) return limit;
    console.warn(`Invalid BODY_SIZE_LIMIT '${value}'; using the 2G default.`);
    return UPLOAD_BODY_LIMIT;
}
