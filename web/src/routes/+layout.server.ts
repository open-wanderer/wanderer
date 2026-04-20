// +layout.ts
import '$lib/i18n';
import type { LayoutServerLoad } from './$types';
import { env } from "$env/dynamic/private";
import { env as publicEnv } from "$env/dynamic/public";
import type { Settings } from '$lib/models/settings';
import { notifications_index } from '$lib/stores/notification_store';
import type { AuthRecord } from 'pocketbase';
import { readdirSync, existsSync } from 'fs';
import { resolve, join } from 'path';

function getMdPages(): Record<string, string[]> {
	const candidates = [
		resolve('static/md/custom'),
		resolve('build/client/md/custom'),
	];
	const mdDir = candidates.find(p => existsSync(p));
	if (!mdDir) return {};

	const result: Record<string, string[]> = {};
	const entries = readdirSync(mdDir, { withFileTypes: true });
	for (const entry of entries) {
		if (!entry.isDirectory()) continue;
		const langDir = join(mdDir, entry.name);
		const files = readdirSync(langDir)
			.filter(f => f.endsWith('.md'))
			.map(f => f.replace(/\.md$/, ''));
		if (files.length > 0) {
			result[entry.name] = files;
		}
	}
	return result;
}

export const load: LayoutServerLoad = async ({ locals, url, fetch }) => {

	let notifications
	if (locals.user?.id) {
		notifications = await notifications_index({ recipient: locals.user.actor }, 1, 10, fetch);
	}
	const mdPages = getMdPages();
	const appTitle = publicEnv.PUBLIC_TITLE || "wanderer";
	return { settings: locals.settings as Settings, user: locals.user as AuthRecord, notifications, origin: env.ORIGIN, mdPages, appTitle }
}