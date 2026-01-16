import { z, ZodType } from "zod";
import type { Waypoint } from "../waypoint";
import { icons } from "$lib/util/icon_util";

const WaypointCreateSchema = z.object({
    id: z.string().length(15).optional(),
    name: z.string().optional(),
    description: z.string().optional(),
    lat: z.number({ coerce: true }).min(-90).max(90),
    lon: z.number({ coerce: true }).min(-180).max(180),
    distance_from_start: z.number({ coerce: true }).min(0).optional(),
    icon: z.enum(icons).optional(),
    author: z.string().length(15),
    photos: z.array(z.string()).default([]),
    trail: z.string().length(15).optional()
}) satisfies ZodType<Partial<Waypoint>>

const WaypointUpdateSchema = z.object({
    name: z.string().optional(),
    description: z.string().optional(),
    lat: z.number({ coerce: true }).min(-90).max(90).optional(),
    lon: z.number({ coerce: true }).min(-180).max(180).optional(),
    distance_from_start: z.number({ coerce: true }).min(0).optional(),
    icon: z.enum(icons).default("circle").optional(),
    photos: z.array(z.string()).optional(),
    "photos-": z.string().optional(),
    "photos+": z.string().optional(),
}) satisfies ZodType<Partial<Waypoint>>

export { WaypointCreateSchema, WaypointUpdateSchema };
