import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

export async function POST(event: RequestEvent) {
    try {
        const { id } = event.params;

        if (id !== event.locals.pb.authStore.record?.id) {
            return json({ message: 'Forbidden' }, { status: 403 });
        }

        const { email, currentPassword } = await event.request.json();

        if (!email) {
            return json({ message: 'email is required' }, { status: 400 });
        }
        if (!currentPassword) {
            return json({ message: 'currentPassword is required' }, { status: 400 });
        }

        const emailChange = await event.locals.pb.send('/user/email', {
            method: 'POST',
            body: JSON.stringify({ email, password: currentPassword }),
        });
        event.locals.pb.authStore.save(emailChange.token, emailChange.record);

        emailChange.record.email = email;

        return json(emailChange.record);
    } catch (e: any) {
        return handleError(e);
    }
}
