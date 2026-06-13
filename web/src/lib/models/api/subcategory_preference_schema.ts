import { z } from "zod";

const UserSubcategoryPreferenceUpsertSchema = z.object({
    subcategory: z.string().length(15),
    visible: z.boolean(),
});

export { UserSubcategoryPreferenceUpsertSchema };
