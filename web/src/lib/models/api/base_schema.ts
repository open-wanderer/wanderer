import { z } from "zod";


const RecordOptionsSchema = z.object({
    expand: z.string().optional(),
    query: z.record(z.string()).optional(),
    requestKey: z.string().optional()
})

const RecordListOptionsSchema = RecordOptionsSchema.extend({
    page: z.number({ coerce: true }).int().nonnegative().optional(),
    perPage: z.number({ coerce: true }).int().optional(),
    sort: z.string().optional(),
    filter: z.string().optional(),
    expand: z.string().optional(),
    requestKey: z.string().optional()
})

const RecordIdValueSchema = z.string().regex(/^[a-z0-9]{15}$/);

const RecordIdSchema = z.object({
    id: RecordIdValueSchema
})


export { RecordOptionsSchema, RecordListOptionsSchema, RecordIdSchema, RecordIdValueSchema }
