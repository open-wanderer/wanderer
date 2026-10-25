import type Meilisearch from 'meilisearch';
import PocketBase from 'pocketbase';


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
