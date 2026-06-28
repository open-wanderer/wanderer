import { z } from "zod";

const UserSubcategoryPreferenceUpsertSchema = z.object({
    subcategory: z.string().length(15),
    visible: z.boolean(),
});

const UserSubcategoryPreferenceReorderSchema = z.object({
    category: z.string().length(15),
    subcategories: z.array(z.string().length(15)),
});

export {
    UserSubcategoryPreferenceReorderSchema,
    UserSubcategoryPreferenceUpsertSchema,
};
