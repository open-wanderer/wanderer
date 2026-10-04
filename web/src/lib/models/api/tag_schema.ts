import { z, ZodType } from "zod";
import type { Tag } from "../tag";

// Match PocketBase's existing 5000-character limit (Unicode code points, not
// UTF-16 units). Reject only C0/DEL; retain empty names, other Unicode,
// punctuation and literal markup without trimming or normalization.
const TagNameSchema = z.string()
    .refine(name => !/[\u0000-\u001f\u007f]/u.test(name), "invalid-tag-name")
    .refine(name => Array.from(name).length <= 5000, "too-long");

const TagCreateSchema = z.object({
    id: z.string().length(15).optional(),
    name: TagNameSchema
}) satisfies ZodType<Tag>

const TagUpdateSchema = z.object({
    name: TagNameSchema.optional()
}) satisfies ZodType<Partial<Tag>>

export { TagCreateSchema, TagUpdateSchema };
