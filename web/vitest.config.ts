import { defineConfig } from "vitest/config";
import base from "./vite.config.ts";

export default defineConfig({
    ...base,
    test: {
        projects: [
            {
                extends: true,
                test: {
                    name: "unit",
                    include: ["src/**/*.{test,spec}.{js,ts}"],
                    exclude: ["src/**/*.browser.test.ts"],
                },
            },
            {
                extends: true,
                resolve: { conditions: ["browser"] },
                test: {
                    name: "browser",
                    environment: "jsdom",
                    include: ["src/**/*.browser.test.ts"],
                },
            },
        ],
    },
});
