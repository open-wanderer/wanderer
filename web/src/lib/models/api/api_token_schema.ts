import { z, ZodType } from "zod";
import type { APIToken } from "../api_token";

const APITokenCreateSchema = z.object({
    name: z.string().min(1, "required"),   
    expiration: z.string().refine((val) => !isNaN(Date.parse(val)), "invalid-date").refine((val) => Date.parse(val) > new Date().getTime(), "date-in-past").optional(),
    user: z.string().length(15),

}) satisfies ZodType<APIToken>

export { APITokenCreateSchema }