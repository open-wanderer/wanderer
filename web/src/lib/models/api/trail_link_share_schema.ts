import { z, ZodType } from "zod";
import type { TrailShare } from "../trail_share";
import type { TrailLinkShare } from "../trail_link_share";

const TrailLinkShareCreateSchema = z.object({
    trail: z.string().length(15),
    permission: z.enum(["view", "edit"])

}) satisfies ZodType<TrailLinkShare>

const TrailLinkShareUpdateSchema = z.object({
    permission: z.enum(["view", "edit"])
}) satisfies ZodType<Partial<TrailShare>>


export { TrailLinkShareCreateSchema, TrailLinkShareUpdateSchema };
