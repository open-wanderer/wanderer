import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function POST(event: RequestEvent) {
    try {
        const formData = await event.request.formData();
        const file = formData.get("file");

        if (!(file instanceof Blob)) {
            return json({ message: "missing_file" }, { status: 400 });
        }

        const r = await event.locals.pb.send("/integration/hammerhead/upload", {
            method: "POST",
            body: formData,
            fetch: event.fetch,
        });
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
