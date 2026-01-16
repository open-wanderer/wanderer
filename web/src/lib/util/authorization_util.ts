import { browser } from "$app/environment";
import { env } from "$env/dynamic/public";

const privateRoutes = [
  "/settings",
  "/trail/edit/new",
  "/lists/edit/new",
]

const publicRoutes = [
  "/",
  "/login",
  "/api/v1/auth",
  "/api/v1/user",
  "/api/v1/category",
  "/api/v1/auth/oauth",
  "/register",
  "/auth",
  "/.well-known"
]

export function isRouteProtected(url: URL | undefined) {

  if (url === undefined) {
    return false;
  }

  if (browser && url.hostname !== window.location.hostname) {
    return false;
  }

  const path = url.pathname

  if (env.PUBLIC_PRIVATE_INSTANCE == "true") {
    return !publicRoutes.some(allowedPath =>
      path === allowedPath || path.startsWith(allowedPath + '/')
    );
  }

  return privateRoutes.some(allowedPath =>
    path === allowedPath || path.startsWith(allowedPath + '/')
  );
}