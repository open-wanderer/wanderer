import { MEILI_URL, ORIGIN } from '$app/env/private'
import { PUBLIC_DISABLE_SIGNUP, PUBLIC_POCKETBASE_URL } from '$app/env/public'
import type { Settings } from '#lib/models/settings.ts'

import PocketBase, { ClientResponseError } from 'pocketbase'
import { isRouteProtected } from '#lib/util/authorization_util.ts'
import { error, redirect } from '@sveltejs/kit'
import { sequence, type Handle } from '@sveltejs/kit/hooks'
import { Meilisearch } from 'meilisearch'
import { locale } from 'svelte-i18n'
import type { Actor } from '#lib/models/activitypub/actor.ts'
import { normalizeLocale } from '#lib/i18n/locales.ts'
import { handleError } from '#lib/util/api_util.ts'
import { apiErrorsAsJson, isApiRequest } from '#lib/server/api_errors.ts'
import { limitRequestBodies } from '#lib/server/request_body.ts'

const SEARCH_TOKEN_VERSION = 1;

// adapter-node no longer applies ORIGIN to event.url, which otherwise
// defaults to https behind a plain-http deployment.
function publicOrigin(url: URL): string {
  return ORIGIN ? new URL(ORIGIN).origin : url.origin
}

function csrf(allowedPaths: string[]): Handle {
  return async ({ event, resolve }) => {
    const { request, url } = event;
    const forbidden =
      isFormContentType(request) &&
      (request.method === "POST" ||
        request.method === "PUT" ||
        request.method === "PATCH" ||
        request.method === "DELETE") &&
      request.headers.get("origin") !== publicOrigin(url) &&
      !allowedPaths.some(p => url.pathname.startsWith(p));

    if (forbidden) {
      const message = `Cross-site ${request.method} form submissions are forbidden`;
      if (isApiRequest(url) || request.headers.get("accept") === "application/json") {
        return Response.json({ message }, { status: 403 });
      }
      return new Response(message, { status: 403 });
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
  const pb = new PocketBase(PUBLIC_POCKETBASE_URL)
  pb.beforeSend = (url, options) => {
    // Keep a plain object: the SDK reads headers by key after beforeSend
    // (e.g. to JSON-serialize the body), which a Headers instance breaks.
    const headers: Record<string, string> = {}
    for (const [key, value] of Object.entries(options.headers ?? {})) {
      if (key.toLowerCase() !== 'x-forwarded-for') headers[key] = value
    }
    try {
      headers['X-Forwarded-For'] = event.getClientAddress()
    } catch {
      // ADDRESS_HEADER is set but this request has none (e.g. a healthcheck).
    }
    return { url, options: { ...options, headers } }
  }
  const url = new URL(event.request.url);

  // Handle API token based auth for API requests
  if (event.request.headers.has("Authorization") && url.pathname.startsWith("/api")) {
    const authHeader = event.request.headers.get("Authorization") as string;
    const apiToken = authHeader.replace("Bearer ", "");
    if (apiToken.startsWith("wanderer_key")) {
      try {
        const authData = await pb.send("/auth/token", {
          method: "POST",
          body: JSON.stringify({
            api_token: apiToken
          }),
          fetch: event.fetch,
        })
        pb.authStore.save(authData.token, authData.record)
      } catch (e) {
        // Only a 401 rejects the token; a missing auth endpoint or a backend
        // failure is not a token rejection.
        if (e instanceof ClientResponseError && e.status === 401) {
          return Response.json({ message: "invalid_token" }, {
            status: 401,
            headers: { "WWW-Authenticate": 'Bearer error="invalid_token"' },
          })
        }
        throw error(500, "Failed to verify API token " + e)
      }

    }
  } else {
    // load the store data from the request cookie string
    pb.authStore.loadFromCookie(event.request.headers.get('cookie') || '')
  }


  const secure = publicOrigin(event.url).startsWith("https:")
  let meiliCookie = event.cookies.get('meilisearch_token');
  let meilisearchToken: string | undefined = undefined;
  const currentUserId = pb.authStore.record?.id || 'public';

  if (meiliCookie) {
    const [token, ownerId, version] = meiliCookie.split('|');

    if (ownerId === currentUserId && Number(version) === SEARCH_TOKEN_VERSION) {
      meilisearchToken = token;
    } else {
      // Identity mismatch (e.g. just logged in/out) or stale token version
      event.cookies.delete('meilisearch_token', { path: '/' });
    }
  }

  if (!meilisearchToken) {
    try {
      const tokenResponse = await pb.send("/search/token", { method: "GET", fetch: event.fetch });
      meilisearchToken = tokenResponse.token
      event.cookies.set('meilisearch_token', `${meilisearchToken}|${currentUserId}|${SEARCH_TOKEN_VERSION}`, {
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
      return Response.json({ message: "Unauthorized" }, { status: 401 })
    }
    throw redirect(302, '/login?r=' + url.pathname);
  } else if (pb.authStore.record && url.pathname === "/login") {
    throw redirect(302, '/');
  } else if (PUBLIC_DISABLE_SIGNUP === "true" && url.pathname === "/register") {
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
    actor = await pb.collection("activitypub_actors").getFirstListItem(`is_local=1&&user='${pb.authStore.record.id}'`)
  }
  const meiliHost = MEILI_URL;
  if (!meiliHost) {
    throw error(500, "Missing MEILI_URL");
  }
  const ms = new Meilisearch({ host: meiliHost, apiKey: meilisearchToken });

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


export const handle = sequence(apiErrorsAsJson, limitRequestBodies, csrf(['/api/v1']), auth, removeLinkFromHeaders)