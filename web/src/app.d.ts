import type Meilisearch from 'meilisearch';
import PocketBase from 'pocketbase';
import type { TrailFilter } from '#lib/models/trail.ts';


// See https://kit.svelte.dev/docs/types#app
// for information about these interfaces
declare global {
	namespace App {
		interface Error {
            message: string;
            originURL?: string;
        }
		// interface Locals {}
		// interface PageData {}
		interface PageState {
			lightbox?: { id: string; index: number };
			mapSearchFilter?: TrailFilter;
		}
		// interface Platform {}
		interface Locals {
			pb: PocketBase
			ms: Meilisearch
			user: AuthModel | null,
			settings: Settings | null

		}
	}
}

export { };
