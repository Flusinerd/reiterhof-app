// Pure helpers behind lib/upload.ts (no React Native imports, unit-tested).

import { ApiError } from "./api-core.ts";

/** Client-side mirror of the backend limit (internal/files MaxSize). */
export const MAX_UPLOAD_BYTES = 20 * 1024 * 1024;

/** German, user-facing text for a failed upload. */
export function uploadErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 413 || err.code === "file_too_large" || err.code === "body_too_large") {
      return "Datei zu groß (höchstens 20 MB).";
    }
    if (err.status === 415 || err.code === "unsupported_type") {
      return "Erlaubt sind Fotos (JPEG, PNG, WebP) und PDFs.";
    }
    if (err.code === "invalid_upload") {
      return "Die Datei konnte nicht gelesen werden. Versuch es mit einem anderen Foto.";
    }
    if (err.code === "network") {
      return "Keine Verbindung. Versuch es gleich noch mal.";
    }
  }
  return "Upload fehlgeschlagen. Versuch es noch mal.";
}

/** Absolute URL for a path returned by the API ("/api/v1/files/..."). */
export function absoluteUrl(baseUrl: string, url: string): string {
  if (/^https?:\/\//.test(url)) return url;
  return `${baseUrl.replace(/\/+$/, "")}${url.startsWith("/") ? "" : "/"}${url}`;
}

/**
 * The API-relative path of a file URL as `POST /api/v1/files/download-link` wants it: no
 * origin, no query. Returns null for anything that is not a file route of this API.
 */
export function downloadLinkPath(baseUrl: string, url: string): string | null {
  const base = baseUrl.replace(/\/+$/, "");
  let path = url;
  if (/^https?:\/\//.test(path)) {
    if (!path.startsWith(base + "/")) return null;
    path = path.slice(base.length);
  }
  path = path.split(/[?#]/, 1)[0] ?? "";
  return /^\/api\/v1\/(files\/|horses\/[^/]+\/documents\/[^/]+\/file$)/.test(path) ? path : null;
}
