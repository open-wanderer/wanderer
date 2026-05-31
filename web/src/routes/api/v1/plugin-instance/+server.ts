import {
    PluginInstanceCreateSchema,
} from "$lib/models/api/plugin_instance_schema";
import type { PluginInstance } from "$lib/models/plugin_instance";
import { Collection, create, handleError, list } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function GET(event: RequestEvent) {
    try {
        const r = await list<PluginInstance>(event, Collection.plugin_instances);
        return json(r);
    } catch (e) {
        return handleError(e);
    }
}

export async function PUT(event: RequestEvent) {
    try {
        const r = await create<PluginInstance>(
            event,
            PluginInstanceCreateSchema,
            Collection.plugin_instances,
        );
        return json(r);
    } catch (e) {
        return handleError(e);
    }
}
