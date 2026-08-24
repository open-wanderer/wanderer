import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const assetPluginActions = [
    "candidates",
    "check",
    "import",
    "import-to-target",
    "import-to-waypoint",
    "materialize-all",
    "delete-remote-assets",
    "remote-assets-summary",
    "repair-remote-assets",
] as const;

type AssetPluginAction = (typeof assetPluginActions)[number];
type ProxyMethod = "GET" | "POST";
type BodyMode = "none" | "required-json" | "optional-json";

const actionConfig = {
    candidates: { method: "POST", body: "required-json" },
    check: { method: "POST", body: "required-json" },
    import: { method: "POST", body: "required-json" },
    "import-to-target": { method: "POST", body: "required-json" },
    "import-to-waypoint": { method: "POST", body: "required-json" },
    "materialize-all": { method: "POST", body: "optional-json" },
    "delete-remote-assets": { method: "POST", body: "none" },
    "remote-assets-summary": { method: "GET", body: "none" },
    "repair-remote-assets": { method: "POST", body: "none" },
} satisfies Record<AssetPluginAction, { method: ProxyMethod; body: BodyMode }>;

const paramsSchema = z.object({
    plugin: z.string().min(1).max(128).regex(/^[a-zA-Z0-9._-]+$/),
    action: z.string().min(1).max(128).regex(/^[a-zA-Z0-9._-]+$/),
});

export async function GET(event: RequestEvent) {
    return proxyPluginAssetAction(event, "GET");
}

export async function POST(event: RequestEvent) {
    return proxyPluginAssetAction(event, "POST");
}

async function proxyPluginAssetAction(event: RequestEvent, method: ProxyMethod) {
    try {
        const params = paramsSchema.parse(event.params);
        if (!isAssetPluginAction(params.action)) {
            return json({ message: "not_found" }, { status: 404 });
        }

        const config = actionConfig[params.action];

        if (config.method !== method) {
            return json(
                { message: "method_not_allowed" },
                { status: 405, headers: { Allow: config.method } },
            );
        }

        const sendOptions: {
            method: ProxyMethod;
            body?: string;
            headers?: Record<string, string>;
        } = { method: config.method };

        if (config.body !== "none") {
            const data =
                config.body === "optional-json"
                    ? await event.request.json().catch(() => ({}))
                    : await event.request.json();
            sendOptions.body = JSON.stringify(data);
            sendOptions.headers = { "Content-Type": "application/json" };
        }

        const r = await event.locals.pb.send(
            `/plugins/assets/${encodeURIComponent(params.plugin)}/${encodeURIComponent(params.action)}`,
            sendOptions,
        );
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}

function isAssetPluginAction(action: string): action is AssetPluginAction {
    return Object.prototype.hasOwnProperty.call(actionConfig, action);
}
