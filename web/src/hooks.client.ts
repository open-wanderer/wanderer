import type { User } from '#lib/models/user.ts';
import { getPb } from '#lib/pocketbase.ts';
import { currentUser } from '#lib/stores/user_store.ts';

const pb = getPb();
pb.authStore.loadFromCookie(document.cookie)
pb.authStore.onChange(() => {
  currentUser.set(pb.authStore.record as User)
  const secure = location.protocol === "https:"
  document.cookie = pb.authStore.exportToCookie({ httpOnly: false, secure: secure, sameSite: "Lax" })
}, true)
