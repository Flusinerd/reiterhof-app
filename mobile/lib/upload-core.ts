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

/** What the server accepts, by file extension. */
const EXTENSION_TYPES: Record<string, string> = {
  jpg: "image/jpeg",
  jpeg: "image/jpeg",
  png: "image/png",
  webp: "image/webp",
  pdf: "application/pdf",
};

const ALLOWED_TYPES = new Set(Object.values(EXTENSION_TYPES));

const TYPE_EXTENSIONS: Record<string, string> = {
  "image/jpeg": "jpg",
  "image/png": "png",
  "image/webp": "webp",
  "application/pdf": "pdf",
};

function extensionOf(path: string): string {
  const clean = path.split(/[?#]/, 1)[0] ?? "";
  const m = /\.([a-z0-9]+)$/i.exec(clean);
  return m ? m[1]!.toLowerCase() : "";
}

/**
 * The Content-Type to declare for an upload. The pickers report the type of the original photo,
 * not of the file they hand over: Android's image picker re-encodes a HEIF photo to a .jpeg file
 * and still says image/heif. So the extension of the file that is actually sent wins; the
 * reported type counts only when it is one the server accepts (web: blob: URLs have no
 * extension, the browser's type is right). Anything else is left to the server's sniffing.
 */
export function uploadContentType(uri: string, mimeType?: string | null): string {
  const byExtension = /^(blob|data):/i.test(uri) ? undefined : EXTENSION_TYPES[extensionOf(uri)];
  if (byExtension) return byExtension;
  const declared = (mimeType ?? "").toLowerCase().split(";", 1)[0]!.trim();
  if (declared === "image/jpg") return "image/jpeg";
  return ALLOWED_TYPES.has(declared) ? declared : "application/octet-stream";
}

/** The file name with an extension that matches the content type ("IMG_1.heic" sent as JPEG -> "IMG_1.jpg"). */
export function uploadFileName(name: string, contentType: string): string {
  const ext = TYPE_EXTENSIONS[contentType];
  const base = name.trim().replace(/[/\\]/g, "_") || "datei";
  if (!ext) return base;
  if (EXTENSION_TYPES[extensionOf(base)] === contentType) return base;
  return `${base.replace(/\.[a-z0-9]+$/i, "")}.${ext}`;
}

/**
 * How the file gets into the multipart body: a web File/Blob from the picker as it is, a
 * blob: or data: URL (web without the File) fetched into a Blob first, anything else (file://,
 * content://, ph://) as React Native's {uri, name, type} object. Browsers turn that object
 * into the text "[object Object]", which is why web uploads used to fail.
 */
export function uploadSource(input: { uri: string; file?: unknown }): "blob" | "fetch" | "native" {
  if (input.file && typeof input.file === "object") return "blob";
  if (/^(blob|data):/i.test(input.uri)) return "fetch";
  return "native";
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
