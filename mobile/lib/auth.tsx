import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

import { api, ApiError, setUnauthorizedHandler, type Me, type Session, type User } from "./api";
import { clearToken, loadToken, saveToken } from "./token";

/**
 * - `loading`: reading the stored token or the profile
 * - `signedOut`: no (valid) session
 * - `signedIn`: session and profile available (`hasStable` tells whether the user joined a stable)
 * - `error`: a session exists but the server could not be reached; call `refresh()`
 */
export type AuthStatus = "loading" | "signedOut" | "signedIn" | "error";

export type AuthContextValue = {
  status: AuthStatus;
  user: User | null;
  me: Me | null;
  hasStable: boolean;
  /** Completes a magic link: exchanges the emailed token for a session. Throws ApiError. */
  signInWithMagicToken: (token: string) => Promise<void>;
  signInWithGoogle: (idToken: string) => Promise<void>;
  signInWithApple: (idToken: string, name?: string) => Promise<void>;
  /** Joins a stable with an invite code and updates the profile. Throws ApiError. */
  joinStable: (code: string) => Promise<void>;
  /** Re-reads the profile (`/me`). */
  refresh: () => Promise<void>;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export const ME_KEY = ["me"] as const;

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  // undefined: still reading secure storage
  const [token, setToken] = useState<string | null | undefined>(undefined);

  useEffect(() => {
    let alive = true;
    loadToken().then((t) => alive && setToken(t));
    return () => {
      alive = false;
    };
  }, []);

  const endSession = useCallback(async () => {
    await clearToken();
    setToken(null);
    queryClient.clear();
  }, [queryClient]);

  useEffect(() => {
    setUnauthorizedHandler(() => void endSession());
    return () => setUnauthorizedHandler(null);
  }, [endSession]);

  const meQuery = useQuery({
    queryKey: ME_KEY,
    queryFn: api.me,
    enabled: !!token,
    retry: (count, err) => !(err instanceof ApiError && err.status === 401) && count < 2,
  });

  const startSession = useCallback(
    async (session: Session) => {
      await saveToken(session.token);
      queryClient.removeQueries({ queryKey: ME_KEY });
      setToken(session.token);
    },
    [queryClient],
  );

  const value = useMemo<AuthContextValue>(() => {
    let status: AuthStatus;
    if (token === undefined) status = "loading";
    else if (token === null) status = "signedOut";
    else if (meQuery.data) status = "signedIn";
    else if (meQuery.isError) status = "error";
    else status = "loading";

    const me = token ? (meQuery.data ?? null) : null;
    return {
      status,
      user: me?.user ?? null,
      me,
      hasStable: !!me?.stable,
      signInWithMagicToken: async (t) => startSession(await api.verifyMagicLink(t)),
      signInWithGoogle: async (idToken) => startSession(await api.signInWithGoogle(idToken)),
      signInWithApple: async (idToken, name) => startSession(await api.signInWithApple(idToken, name)),
      joinStable: async (code) => {
        queryClient.setQueryData(ME_KEY, await api.joinStable(code));
      },
      refresh: async () => {
        await meQuery.refetch();
      },
      signOut: async () => {
        try {
          await api.logout();
        } catch {
          // offline or already expired: the local session is dropped either way
        }
        await endSession();
      },
    };
  }, [token, meQuery, startSession, endSession, queryClient]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside <AuthProvider>");
  return value;
}
