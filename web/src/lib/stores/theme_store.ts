import { browser } from "$app/environment";
import { get, writable, type Writable } from "svelte/store";

type Theme = "dark" | "light"

export const theme: Writable<Theme> = writable(getDefaultTheme());

function getDefaultTheme(): Theme {
    if (browser) {
        if (localStorage.getItem("theme")) {
            return localStorage.getItem("theme") as Theme;
        } else if (document.documentElement.classList.contains("light")) {
            return "light"
        } else if (document.documentElement.classList.contains("dark")) {
            return "dark";
        }
    }

    return "light";

}

export function toggleTheme() {
    const currentTheme = get(theme);
    const newTheme = currentTheme === "light" ? "dark" : "light";
    document.documentElement.classList.remove(currentTheme)
    document.documentElement.classList.add(newTheme)
    document.querySelector("meta[name='color-scheme'" )?.setAttribute("content", newTheme)
    theme.set(newTheme)
    localStorage.setItem("theme", newTheme);
}