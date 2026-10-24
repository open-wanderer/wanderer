import { z } from 'zod';

// Mirrors the Go backend's region id allow-list (services/regions/config.go):
// a seeded materialized path such as "algeria.algeria_central".
const RegionIdSchema = z.object({
  id: z
    .string()
    .regex(/^[a-z0-9][a-z0-9_.'-]*$/, 'Invalid region id')
    .refine((v) => !v.includes('..'), 'Invalid region id'),
});

export { RegionIdSchema };
