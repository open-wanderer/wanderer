import { z } from "zod";

const UserCategoryPreferenceUpsertSchema = z.object({
    category: z.string().length(15),
    visible: z.boolean(),
});

const UserCategoryPreferenceReorderSchema = z.object({
    categories: z.array(z.string().length(15)),
});

export {
    UserCategoryPreferenceReorderSchema,
    UserCategoryPreferenceUpsertSchema,
};
