// File upload helper, reusable by every feature that attaches photos or documents
// (backend: POST /api/v1/files, internal/files).
//
//   const saved = await uploadFile({ uri, name: "decke.jpg", mimeType: "image/jpeg" });
//   await api.post("/api/v1/blankets/...", { photo_path: saved.path });
//
// To show a stored file use <Image source={fileSource(saved.url)} />, which sends the
// session token as a header. To hand a file to the browser or a viewer app (a PDF, a full-size
// photo) use openStoredFile(url): it fetches a short-lived download link first, so the session
// token never appears in a URL (histories, share sheets and proxy logs would keep it).

import { Linking } from "react-native";

import { API_URL, authed } from "./api";
import { parseErrorBody, ApiError } from "./api-core.ts";
import { getToken } from "./token";
import { absoluteUrl, downloadLinkPath, MAX_UPLOAD_BYTES, uploadErrorMessage } from "./upload-core.ts";

export { MAX_UPLOAD_BYTES, uploadErrorMessage };

export type UploadInput = {
  /** file:// or content:// URI from the image or document picker. */
  uri: string;
  /** File name shown to the server (only used as a hint, the server names the file). */
  name: string;
  /** Best-known MIME type, e.g. image/jpeg or application/pdf. */
  mimeType?: string | null;
  /** Size in bytes if known; checked against the 20 MB limit before uploading. */
  size?: number | null;
};

export type UploadedFile = {
  /** Store this in the domain record (e.g. horse_documents.file_path). */
  path: string;
  /** Relative API URL, "/api/v1/files/<path>". */
  url: string;
  content_type: string;
  size: number;
};

/** Uploads one file as multipart/form-data. Throws ApiError (see uploadErrorMessage for German text). */
export async function uploadFile(input: UploadInput): Promise<UploadedFile> {
  if (input.size && input.size > MAX_UPLOAD_BYTES) {
    throw new ApiError(413, "file_too_large", "File exceeds 20 MB");
  }
  const form = new FormData();
  // React Native's FormData accepts {uri, name, type} objects for files.
  form.append("file", {
    uri: input.uri,
    name: input.name,
    type: input.mimeType || "application/octet-stream",
  } as unknown as Blob);

  const token = getToken();
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;

  let res: Response;
  try {
    // No Content-Type header: fetch adds the multipart boundary itself.
    res = await fetch(`${API_URL.replace(/\/+$/, "")}/api/v1/files`, { method: "POST", headers, body: form });
  } catch {
    throw new ApiError(0, "network", "Network request failed");
  }
  const text = await res.text();
  if (!res.ok) throw parseErrorBody(res.status, text);
  return JSON.parse(text) as UploadedFile;
}

/** Absolute URL of a stored file (no credentials). */
export function fileUrl(url: string): string {
  return absoluteUrl(API_URL, url);
}

/** `<Image source>` for a stored file; sends the session token as Authorization header. */
export function fileSource(url: string): { uri: string; headers: Record<string, string> } {
  const token = getToken();
  return { uri: fileUrl(url), headers: token ? { Authorization: `Bearer ${token}` } : {} };
}

/**
 * A URL that opens the file for five minutes without headers (`POST /api/v1/files/download-link`).
 * Throws ApiError; `validation_failed` for a URL that is not a file route.
 */
export async function downloadLink(url: string): Promise<string> {
  const path = downloadLinkPath(API_URL, url);
  if (!path) throw new ApiError(400, "validation_failed", "not a file route");
  const link = await authed.post<{ url: string }>("/api/v1/files/download-link", { url: path });
  return fileUrl(link.url);
}

/** Opens a stored file in the browser or viewer app through a short-lived download link. */
export async function openStoredFile(url: string): Promise<void> {
  await Linking.openURL(await downloadLink(url));
}
