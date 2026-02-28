import { env } from '$env/dynamic/private'
import { env as envPub } from '$env/dynamic/public'
import type { Settings } from '$lib/models/settings'

import PocketBase from 'pocketbase'
import { isRouteProtected } from '$lib/util/authorization_util'
import { error, json, redirect, text, type Handle } from '@sveltejs/kit'
import { sequence } from '@sveltejs/kit/hooks'
import { MeiliSearch } from 'meilisearch'
import { locale } from 'svelte-i18n'
import type { Actor } from '$lib/models/activitypub/actor'
import { normalizeLocale } from '$lib/i18n/locales'
import { handleError } from '$lib/util/api_util'


function csrf(allowedPaths: string[]): Handle {
  return async ({ event, resolve }) => {
    const { request, url } = event;
    const forbidden =
      isFormContentType(request) &&
      (request.method === "POST" ||
        request.method === "PUT" ||
        request.method === "PATCH" ||
        request.method === "DELETE") &&
      request.headers.get("origin") !== url.origin &&
      !allowedPaths.some(p => url.pathname.startsWith(p));

    if (forbidden) {
      const message = `Cross-site ${request.method} form submissions are forbidden`;
      if (request.headers.get("accept") === "application/json") {
        return json({ message }, { status: 403 });
      }
      return text(message, { status: 403 });
    }

    return resolve(event);
  };
}

function isContentType(request: Request, ...types: string[]) {
  const type = request.headers.get("content-type")?.split(";", 1)[0].trim() ?? "";
  return types.includes(type.toLowerCase());
}
function isFormContentType(request: Request) {
  return isContentType(
    request,
    "application/x-www-form-urlencoded",
    "multipart/form-data",
    "text/plain",
  );
}

const auth: Handle = async ({ event, resolve }) => {
  const pb = new PocketBase(envPub.PUBLIC_POCKETBASE_URL)
  const url = new URL(event.request.url);

  // load the store data from the request cookie string
  pb.authStore.loadFromCookie(event.request.headers.get('cookie') || '')

  const secure = event.url.protocol === "https:"
  let meiliCookie = event.cookies.get('meilisearch_token');
  let meilisearchToken: string | undefined = undefined;
  const currentUserId = pb.authStore.record?.id || 'public';

  if (meiliCookie) {
    const [token, ownerId] = meiliCookie.split('|');

    if (ownerId === currentUserId) {
      meilisearchToken = token;
    } else {
      // Identity mismatch (e.g. just logged in/out)
      event.cookies.delete('meilisearch_token', { path: '/' });
    }
  }

  if (!meilisearchToken) {
    try {
      const tokenResponse = await pb.send("/search/token", { method: "GET", fetch: event.fetch });
      meilisearchToken = tokenResponse.token
      event.cookies.set('meilisearch_token', `${meilisearchToken}|${currentUserId}`, {
        path: '/',
        httpOnly: false,
        maxAge: 60 * 60 * 24,
        sameSite: 'lax',
        secure: secure
      });
    } catch (e) {
      if (url.pathname.startsWith("/api")) {
        return handleError(e)
      }
      throw error(500, "Failed to invalidate meilisearch token: " + e)
    }

  }

  // validate the user existence and if the path is acceesible
  if (!pb.authStore.record && isRouteProtected(url)) {
    if (url.pathname.startsWith("/api")) {
      return json({ message: "Unauthorized" }, { status: 401 })
    }
    throw redirect(302, '/login?r=' + url.pathname);
  } else if (pb.authStore.record && url.pathname === "/login") {
    throw redirect(302, '/');
  } else if (envPub.PUBLIC_DISABLE_SIGNUP === "true" && url.pathname === "/register") {
    throw redirect(302, '/');
  }

  try {
    // get an up-to-date auth store state by verifying and refreshing the loaded auth model (if any)
    if (pb.authStore.isValid) {
      await pb.collection('users').authRefresh({ requestKey: null })
    }
  } catch (_) {
    // clear the auth store on failed refresh
    pb.authStore.clear()
    event.cookies.delete('meilisearch_token', { path: '/' });
  }

  let settings: Settings | undefined;
  let actor: Actor | undefined;

  if (pb.authStore.record) {
    settings = await pb.collection('settings').getFirstListItem<Settings>(`user="${pb.authStore.record.id}"`, { requestKey: null })
    actor = await pb.collection("activitypub_actors").getFirstListItem(`isLocal=1&&user='${pb.authStore.record.id}'`)
  }
  const meiliHost = env.MEILI_URL;
  if (!meiliHost) {
    throw error(500, "Missing MEILI_URL");
  }
  const ms = new MeiliSearch({ host: meiliHost, apiKey: meilisearchToken });

  event.locals.ms = ms
  event.locals.pb = pb
  event.locals.user = pb.authStore.record
  if (event.locals.user) {
    event.locals.user.actor = actor?.id
  }
  event.locals.settings = settings

  const langHeader = event.request.headers.get('accept-language')?.split(',')[0]
  const lang = settings?.language ?? langHeader

  if (lang) {
    const normalizedLocale = normalizeLocale(lang)
    locale.set(normalizedLocale)
    if (pb.authStore.record) {
      pb.authStore.record!.language = normalizedLocale;
    }
  }

  const response = await resolve(event)

  // send back the default 'pb_auth' cookie to the client with the latest store state
  const pbCookie = pb.authStore.exportToCookie({ httpOnly: false, secure: secure, sameSite: "Lax" });
  if (pbCookie) {
    response.headers.append('set-cookie', pbCookie);
  }


  return response
}

const removeLinkFromHeaders: Handle =
  async ({ event, resolve }) => {
    const response = await resolve(event);
    response.headers.delete('link');
    return response;
  }


export const handle = sequence(csrf(['/api/v1']), auth, removeLinkFromHeaders)
