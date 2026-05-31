import {
    PluginInstanceUpdateSchema,
} from "$lib/models/api/plugin_instance_schema";
import type { PluginInstance } from "$lib/models/plugin_instance";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function GET(event: RequestEvent) {
    try {
        const r = await show<PluginInstance>(event, Collection.plugin_instances);
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}

export async function POST(event: RequestEvent) {
    try {
        const r = await update<PluginInstance>(
            event,
            PluginInstanceUpdateSchema,
            Collection.plugin_instances,
        );
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}

export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.plugin_instances);
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
