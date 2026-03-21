import { z, ZodType } from "zod";
import { NotificationType } from "../notification";
import { Language, type Settings } from "../settings";

const SettingsCreateSchema = z.object({
    unit: z.enum(["metric", "imperial"]).optional(),
    language: z.enum(Object.values(Language) as [Language, ...Language[]]).optional(),
    bio: z.string().optional().nullable(),
    mapFocus: z.enum(["trails", "location"]).optional(),
    location: z.object({
        name: z.string(),
        lat: z.number(),
        lon: z.number()
    }).optional().nullable(),
    category: z.string().optional(),
    tilesets: z.array(z.object({ name: z.string(), url: z.string().url() })).optional().nullable(),
    terrain: z.object({ terrain: z.string().url().or(z.literal('')).optional(), hillshading: z.string().url().or(z.literal('')).optional() }).optional().nullable(),
    user: z.string().optional(),
    privacy: z.object({
        account: z.enum(["public", "private"]),
        trails: z.enum(["public", "private"]),
        lists: z.enum(["public", "private"])
    }).optional().nullable(),
    notifications: z.record(z.enum(Object.values(NotificationType) as [string, ...string[]]), z.object({ web: z.boolean(), email: z.boolean() })).optional().nullable(),
    behavior: z.object({ allowAutoGeolocate: z.boolean() }).optional().nullable(),
}) satisfies ZodType<Settings>
ZodType<Partial<Comment>>

export { SettingsCreateSchema };
