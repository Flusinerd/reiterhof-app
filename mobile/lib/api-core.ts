// Pure API client (no React Native imports, so it can be unit-tested with node --test).
// lib/api.ts binds it to the base URL from app.json and the token in secure storage.

/** Error envelope of the backend: {"error":{"code":"...","message":"..."}}. */
export type ErrorEnvelope = { error: { code: string; message: string } };

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

/** Turns an error response body into an ApiError. Never throws; falls back to generic codes. */
export function parseErrorBody(status: number, text: string): ApiError {
  try {
    const body = JSON.parse(text) as Partial<ErrorEnvelope> | null;
    const detail = body?.error;
    if (detail && typeof detail.code === "string") {
      return new ApiError(status, detail.code, typeof detail.message === "string" ? detail.message : detail.code);
    }
  } catch {
    // not JSON (proxy error page, empty body): use the generic error below
  }
  return new ApiError(status, status >= 500 ? "internal" : "unknown", `Request failed with status ${status}`);
}

/** German, user-facing text for an error. Unknown errors get a generic message. */
export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case "network":
        return "Keine Verbindung zum Server. Bitte prüfe deine Internetverbindung.";
      case "rate_limited":
        return "Zu viele Versuche. Bitte warte kurz und versuche es später erneut.";
      case "invalid_token":
        return "Der Link ist ungültig, abgelaufen oder wurde schon verwendet. Fordere einen neuen an.";
      case "invalid_code":
        return "Der Code ist ungültig, abgelaufen oder aufgebraucht.";
      case "already_in_stable":
        return "Du gehörst bereits zu einem Stall.";
      case "email_not_verified":
        return "Die E-Mail-Adresse deines Kontos ist nicht bestätigt.";
      case "not_configured":
        return "Diese Anmeldung ist noch nicht eingerichtet.";
      case "consent_required":
        return "Dafür fehlt deine Einwilligung. Du kannst sie unter Einstellungen und Datenschutz erteilen.";
      case "validation_failed":
        return "Bitte prüfe deine Eingabe.";
      case "unauthorized":
        return "Bitte melde dich an.";
      default:
        return "Etwas ist schiefgelaufen. Bitte versuche es erneut.";
    }
  }
  return "Etwas ist schiefgelaufen. Bitte versuche es erneut.";
}

export type ClientOptions = {
  baseUrl: string;
  /** Returns the bearer token, or null when signed out. */
  getToken: () => string | null | Promise<string | null>;
  /** Called when a request that carried a token got 401 (session revoked or expired). */
  onUnauthorized?: () => void;
  fetchImpl?: typeof fetch;
};

export type Client = ReturnType<typeof createClient>;

export function createClient(options: ClientOptions) {
  const doFetch = options.fetchImpl ?? fetch;
  const base = options.baseUrl.replace(/\/+$/, "");

  async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const token = await options.getToken();
    const headers: Record<string, string> = { Accept: "application/json" };
    if (body !== undefined) headers["Content-Type"] = "application/json";
    if (token) headers.Authorization = `Bearer ${token}`;

    let res: Response;
    try {
      res = await doFetch(`${base}${path}`, {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
      });
    } catch {
      throw new ApiError(0, "network", "Network request failed");
    }
    if (res.status === 204) return undefined as T;
    const text = await res.text();
    if (!res.ok) {
      const err = parseErrorBody(res.status, text);
      if (res.status === 401 && token) options.onUnauthorized?.();
      throw err;
    }
    return (text ? JSON.parse(text) : undefined) as T;
  }

  return {
    get: <T>(path: string) => request<T>("GET", path),
    post: <T>(path: string, body?: unknown) => request<T>("POST", path, body ?? {}),
    patch: <T>(path: string, body: unknown) => request<T>("PATCH", path, body),
    put: <T>(path: string, body: unknown) => request<T>("PUT", path, body),
  };
}
