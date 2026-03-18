import { api_tokens_index } from "$lib/stores/api_token_store";
import { type Load } from "@sveltejs/kit";

export const load: Load = async ({ params, fetch }) => {
    const apiTokens = await api_tokens_index(fetch)
    return { apiTokens: apiTokens }
};