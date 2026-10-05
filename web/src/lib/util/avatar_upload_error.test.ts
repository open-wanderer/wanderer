import { get } from "svelte/store";
import { _, addMessages, init, locale } from "svelte-i18n";
import { beforeAll, describe, expect, it } from "vitest";
import de from "$lib/i18n/locales/de.json";
import en from "$lib/i18n/locales/en.json";
import { APIError } from "$lib/util/api_util";
import { avatarUploadErrorMessage } from "./avatar_upload_error";

function sizeError(maxSize: unknown, status = 400) {
    return new APIError(status, "Failed to update record.", {
        data: {
            avatar: {
                code: "validation_file_size_limit",
                message: "Failed to upload avatar.png.",
                params: { maxSize, file: "avatar.png" },
            },
        },
    });
}

describe("avatarUploadErrorMessage", () => {
    it.each([
        [5 * 1024 * 1024, 5, "MiB"],
        [20 * 1024 * 1024, 20, "MiB"],
        [1024, 1, "KiB"],
        [1, 1, "bytes"],
        [1048577, 1048577, "bytes"],
    ])("uses the actual limit %s without rounding", (bytes, size, unit) => {
        const message = avatarUploadErrorMessage(sizeError(bytes));
        expect(message).toEqual({
            key: "error-updating-avatar-too-large-with-limit",
            values: { maxSize: size, unit },
        });
    });

    it.each([
        undefined, null, 0, -1, NaN, Infinity, 1.5, "5242880",
        Number.MAX_SAFE_INTEGER + 1, {}, [],
    ])("does not interpolate an invalid limit %s", (maxSize) => {
        expect(avatarUploadErrorMessage(sizeError(maxSize))).toEqual({
            key: "error-updating-avatar-too-large",
        });
    });

    it.each([
        undefined, null, "invalid detail", [],
        { data: null },
        { data: { avatar: [] } },
        { avatar: { code: "validation_file_size_limit", params: { maxSize: 1024 } } },
        { data: { password: { code: "validation_file_size_limit", params: { maxSize: 1024 } } } },
        { data: { avatar: { code: "unknown_error" } } },
        { data: { avatar: { code: "validation_invalid_file", params: { invalidFiles: ["avatar.png"] } } } },
    ])("uses the fallback for unknown or malformed validation details", (detail) => {
        expect(avatarUploadErrorMessage(new APIError(400, "Failed to update record.", detail))).toEqual({
            key: "error-updating-avatar",
        });
    });

    it("recognizes an unsupported image format", () => {
        const error = new APIError(400, "Failed to update record.", {
            data: { avatar: { code: "validation_invalid_mime_type", message: "Unsupported file type." } },
        });
        expect(avatarUploadErrorMessage(error)).toEqual({
            key: "error-updating-avatar-unsupported-type",
        });
    });

    it("does not let validation details override authentication or server errors", () => {
        expect(avatarUploadErrorMessage(sizeError(1024, 401))).toEqual({
            key: "error-updating-avatar-sign-in",
        });
        expect(avatarUploadErrorMessage(sizeError(1024, 503))).toEqual({
            key: "error-updating-avatar-server",
        });
    });

    it("keeps unexpected exceptions generic and gives connection guidance for fetch rejections", () => {
        for (const error of [null, undefined, new Error("Unexpected failure"), new SyntaxError("Invalid JSON"), { status: 401 }]) {
            expect(avatarUploadErrorMessage(error)).toEqual({ key: "error-updating-avatar" });
        }
        expect(avatarUploadErrorMessage(new TypeError("Failed to fetch"))).toEqual({
            key: "error-updating-avatar-network",
        });
    });
});

describe("localized avatar error messages", () => {
    beforeAll(() => {
        addMessages("en", en);
        addMessages("de", de);
        init({ fallbackLocale: "en", initialLocale: "en" });
    });

    it.each([
        ["en", 5 * 1024 * 1024, "5 MiB"],
        ["de", 5 * 1024 * 1024, "5 MiB"],
        ["en", 1024, "1 KiB"],
        ["de", 1, "1 Bytes"],
        ["en", 1048577, "1,048,577 bytes"],
        ["de", 1048577, "1.048.577 Bytes"],
    ])("renders the exact limit for %s", (language, limit, text) => {
        locale.set(language);
        const message = avatarUploadErrorMessage(sizeError(limit));
        const translated = get(_)(message.key, { values: message.values });
        expect(translated).toContain(text);
    });
});
