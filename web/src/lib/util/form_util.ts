/** The dotted paths of the fields that have an error in a Felte errors object. */
export function invalidFields(errors: unknown, prefix = ""): string[] {
    if (Array.isArray(errors) && errors.every((error) => typeof error === "string")) {
        return errors.length > 0 ? [prefix] : [];
    }
    if (typeof errors !== "object" || errors === null) {
        return typeof errors === "string" && errors ? [prefix] : [];
    }
    return Object.entries(errors).flatMap(([key, value]) => invalidFields(value, prefix ? `${prefix}.${key}` : key));
}
