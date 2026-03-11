import type { User, UserAnonymous } from "$lib/models/user";
import { getPb } from "$lib/pocketbase";
import { APIError } from "$lib/util/api_util";
import { type AuthMethodsList } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";

export const currentUser: Writable<User | null> = writable<User | null>()

export async function users_create(user: User) {
    let r = await fetch('/api/v1/user', {
        method: 'PUT',
        body: JSON.stringify({ ...user, passwordConfirm: user.password })
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }


    const createdUser: User = await r.json();

    return createdUser;
}

export async function users_search(q: string, includeSelf: boolean = true) {
    const user = get(currentUser)

    let r = await fetch('/api/v1/user/anonymous?' + new URLSearchParams({
        "filter": `username~"${q}"${includeSelf ? '' : `&&id!="${user?.id}"`}`,
    }), {
        method: 'GET',
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response = await r.json()
    return response.items;

}

export async function users_show(id: string, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let r = await f(`/api/v1/user/anonymous/${id}`, {
        method: 'GET',
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
    const response: UserAnonymous = await r.json()

    return response;
}

export async function users_auth_methods(f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch): Promise<AuthMethodsList> {
    const r = await f('/api/v1/auth/oauth', {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
    return await r.json() as AuthMethodsList


}


export async function login(user: User) {
    const pb = getPb();

    const r = await fetch('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify(user),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
    pb.authStore.loadFromCookie(document.cookie)

}

export async function oauth_login(data: { name: string, code: string, codeVerifier: string }) {
    const pb = getPb();

    const r = await fetch('/api/v1/auth/oauth', {
        method: 'POST',
        body: JSON.stringify(data)
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    pb.authStore.loadFromCookie(document.cookie)

}


export async function logout() {
    const pb = getPb();

    pb.authStore.clear();
}

export async function users_update(user: User | { [K in keyof User]?: User[K] }, avatar?: File) {
    let r = await fetch('/api/v1/user/' + user.id, {
        method: 'POST',
        body: JSON.stringify(user)
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }


    if (avatar) {
        const formData = new FormData();

        formData.append("avatar", avatar);


        r = await fetch(`/api/v1/user/${user.id!}/file`, {
            method: 'POST',
            body: formData,
        })

        if (!r.ok) {
            const response = await r.json();
            throw new APIError(r.status, response.message, response.detail)
        }

    }

    const model: User = await r.json();
    const existing = get(currentUser);
    const merged: User = {
        ...(existing ?? {}),
        ...model,
    } as User;

    if (!(merged as any).actor && (existing as any)?.actor) {
        (merged as any).actor = (existing as any).actor;
    }

    currentUser.set(merged);
}

export async function users_delete(user: User) {
    const r = await fetch('/api/v1/user/' + user.id, {
        method: 'DELETE',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

}

export async function users_reset_password(reset: { email: string }) {
    const r = await fetch('/api/v1/auth/reset', {
        method: 'POST',
        body: JSON.stringify(reset),
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    return await r.json();

}

export async function users_confirm_reset(reset: { password: string, passwordConfirm: string, token: string }) {
    const r = await fetch('/api/v1/auth/confirm-reset', {
        method: 'POST',
        body: JSON.stringify(reset),
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    return await r.json();

}
