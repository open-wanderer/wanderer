// +layout.ts
import '#lib/i18n/index.ts';
import type { LayoutServerLoad } from './$types';
import { ORIGIN } from "$app/env/private";
import type { Settings } from '#lib/models/settings.ts';
import { notifications_index } from '#lib/stores/notification_store.ts';
import type { AuthRecord } from 'pocketbase';

export const load: LayoutServerLoad = async ({ locals, url, fetch }) => {

	let notifications
	if (locals.user?.id) {
		notifications = await notifications_index({ recipient: locals.user.actor }, 1, 10, fetch);
	}
	return { settings: locals.settings as Settings, user: locals.user as AuthRecord, notifications, origin: ORIGIN }
}