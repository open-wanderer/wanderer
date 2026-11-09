import fs from "node:fs";
import { compile } from "svelte/compiler";

// Compilation only. Do not import/mount generated components or start Vite.
for (const filename of [
    "src/lib/components/summit_log/summit_log_modal.svelte",
    "src/lib/components/trail/trail_info_panel.svelte",
    "src/lib/components/base/button.svelte",
    "src/lib/components/base/modal.svelte",
]) {
    const result = compile(fs.readFileSync(filename, "utf8"), { filename, generate: "client", dev: true });
    console.log(JSON.stringify({ filename, compiled: Boolean(result.js.code), warnings: result.warnings.map(warning => ({ code: warning.code, message: warning.message })) }));
}
