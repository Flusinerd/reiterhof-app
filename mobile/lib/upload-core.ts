// Pure helpers behind lib/upload.ts (no React Native imports, unit-tested).

import { ApiError } from "./api-core.ts";

/** Client-side mirror of the backend limit (internal/files MaxSize). */
export const MAX_UPLOAD_BYTES = 20 * 1024 * 1024;

/** German, user-facing text for a failed upload. */
export function uploadErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 413 || err.code === "file_too_large" || err.code === "body_too_large") {
      return "Die Datei ist zu groß (höchstens 20 MB).";
    }
    if (err.status === 415 || err.code === "unsupported_type") {
      return "Erlaubt sind Fotos (JPEG, PNG, HEIC, WebP) und PDF-Dateien.";
    }
    if (err.code === "network") {
      return "Keine Verbindung zum Server. Bitte prüfe deine Internetverbindung.";
    }
  }
  return "Der Upload hat nicht geklappt. Bitte versuche es erneut.";
}

/** Absolute URL for a path returned by the API ("/api/v1/files/..."). */
export function absoluteUrl(baseUrl: string, url: string): string {
  if (/^https?:\/\//.test(url)) return url;
  return `${baseUrl.replace(/\/+$/, "")}${url.startsWith("/") ? "" : "/"}${url}`;
}

/**
 * Appends `?access_token=` for consumers that cannot send headers (Linking.openURL of a PDF).
 * Prefer `Authorization` headers where possible (`<Image source={{ uri, headers }}>`): a token in
 * a URL can end up in logs.
 */
export function withAccessToken(url: string, token: string | null): string {
  if (!token) return url;
  return `${url}${url.includes("?") ? "&" : "?"}access_token=${encodeURIComponent(token)}`;
}
