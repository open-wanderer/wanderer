import { SummitLogCreateSchema } from '$lib/models/api/summit_log_schema';
import type { SummitLog } from '$lib/models/summit_log';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

export async function GET(event: RequestEvent) {
    try {
        const summitLogs = await list<SummitLog>(event, Collection.summit_logs);
        removeTimeFromDates(summitLogs.items)
        return json(summitLogs)

    } catch (e) {
        return handleError(e)
    }
}

export async function PUT(event: RequestEvent) {
    try {
        const r = await create<SummitLog>(event, SummitLogCreateSchema, Collection.summit_logs)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}


function removeTimeFromDates(logs: SummitLog[]) {
    logs.forEach(l => l.date = l.date.substring(0, 10));

}