// +layout.ts
import '#lib/i18n/index.ts'
import { waitLocale } from 'svelte-i18n'
import type { LayoutLoad } from './$types'

export const load: LayoutLoad = async ({ data }) => {
	await waitLocale()

	return data;
}