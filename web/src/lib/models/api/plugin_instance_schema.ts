import { z } from "zod";

const PluginInstanceStatusSchema = z.enum([
    "configured",
    "needs_auth",
    "needs_reauth",
    "syncing",
    "rate_limited",
    "unavailable",
    "unsupported_protocol",
    "error",
    "disabled",
]);

const OptionalJsonRecordSchema = z.preprocess(
    (value) => (value === null ? undefined : value),
    z.record(z.string(), z.unknown()).optional(),
);
const OptionalAuthSchema = z.preprocess(
    (value) => {
        if (value === null) {
            return undefined;
        }
        if (!value || typeof value !== "object" || Array.isArray(value)) {
            return value;
        }
        return Object.fromEntries(
            Object.entries(value).map(([key, fieldValue]) => [
                key,
                fieldValue == null ? "" : String(fieldValue),
            ]),
        );
    },
    z.record(z.string(), z.string()).optional(),
);

const PluginInstanceCreateSchema = z.object({
    user: z.string().length(15),
    plugin_id: z.string().min(1).max(64).regex(/^[a-z0-9][a-z0-9_-]*$/),
    enabled: z.boolean().default(false),
    auth: OptionalAuthSchema,
    config: OptionalJsonRecordSchema,
    state: OptionalJsonRecordSchema,
    status: PluginInstanceStatusSchema.optional(),
});

const PluginInstanceUpdateSchema = z.object({
    enabled: z.boolean().optional(),
    auth: OptionalAuthSchema,
    config: OptionalJsonRecordSchema,
    state: OptionalJsonRecordSchema,
    status: PluginInstanceStatusSchema.optional(),
});

export {
    PluginInstanceCreateSchema,
    PluginInstanceUpdateSchema,
    PluginInstanceStatusSchema,
};
