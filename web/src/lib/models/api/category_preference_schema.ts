import { z } from "zod";

const UserCategoryPreferenceUpsertSchema = z.object({
    category: z.string().length(15),
    exclude_search: z.boolean().optional(),
    hide_design: z.boolean().optional(),
    exclude_federated: z.boolean().optional(),
});

const UserCategoryPreferenceReorderSchema = z.object({
    categories: z.array(z.string().length(15)),
});

export {
    UserCategoryPreferenceReorderSchema,
    UserCategoryPreferenceUpsertSchema,
};
