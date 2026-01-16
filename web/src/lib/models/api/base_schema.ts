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

const RecordIdSchema = z.object({
    id: z.string().length(15)
})


export { RecordOptionsSchema, RecordListOptionsSchema, RecordIdSchema }