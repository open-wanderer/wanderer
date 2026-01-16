import { z, ZodType } from "zod";
import type { Trail } from "../trail";


const TrailCreateSchema = z.object({
    id: z.string().length(15).optional(),
    name: z.string().min(1, "required"),
    description: z.string().optional(),
    location: z.string().optional(),
    date: z.string().optional().refine((val) => !val || !isNaN(Date.parse(val)), "invalid-date"),
    public: z.boolean(),
    difficulty: z.enum(["easy", "moderate", "difficult"]).optional(),
    lat: z.number().min(-90).max(90).optional(),
    lon: z.number().min(-180).max(180).optional(),
    distance: z.number({ coerce: true }).nonnegative().optional(),
    elevation_gain: z.number({ coerce: true }).nonnegative().optional(),
    elevation_loss: z.number({ coerce: true }).nonnegative().optional(),
    duration: z.number({ coerce: true }).nonnegative().optional(),
    photos: z.array(z.string()).default([]),
    thumbnail: z.number().int().nonnegative().optional(),
    like_count: z.number().int().min(0).optional().default(0),
    category: z.string().length(15).optional().or(z.literal('')),
    tags: z.array(z.string()).default([]),
    gpx: z.string().optional(),
    author: z.string().length(15),

}) satisfies ZodType<Partial<Trail>>

const TrailUpdateSchema = z.object({
    name: z.string(),
    description: z.string().optional(),
    location: z.string().optional(),
    date: z.string().optional().refine((val) => !val || !isNaN(Date.parse(val)), "invalid-date"),
    public: z.boolean().optional(),
    difficulty: z.enum(["easy", "moderate", "difficult"]).optional(),
    lat: z.number().min(-90).max(90).optional(),
    lon: z.number().min(-180).max(180).optional(),
    distance: z.number({ coerce: true }).nonnegative().optional(),
    elevation_gain: z.number({ coerce: true }).nonnegative().optional(),
    elevation_loss: z.number({ coerce: true }).nonnegative().optional(),
    duration: z.number({ coerce: true }).nonnegative().optional(),
    photos: z.array(z.string()).optional(),
    "photos-": z.string().optional(),
    "photos+": z.string().optional(),
    thumbnail: z.number().int().nonnegative().optional(),
    like_count: z.number().int().min(0).optional().default(0),
    category: z.string().optional(),
    tags: z.array(z.string()).optional(),
    gpx: z.string().optional(),
}) satisfies ZodType<Partial<Trail>>

const TrailRecommendSchema = z.object({
    size: z.number({ coerce: true }).optional()
})

export { TrailCreateSchema, TrailUpdateSchema, TrailRecommendSchema };
