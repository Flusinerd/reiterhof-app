import { useEffect, useState } from "react";
import { Image } from "react-native";

import { fileSource } from "@/lib/upload";

import type { AuthImageProps } from "./auth-image";

export type { AuthImageProps };

// Files never change under their path (random names), so a blob URL per path is kept for the
// lifetime of the page. A failed load is not cached.
const BLANK = "data:image/gif;base64,R0lGODlhAQABAAAAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw==";

const blobs = new Map<string, Promise<string>>();

function load(url: string): Promise<string> {
  let blob = blobs.get(url);
  if (!blob) {
    const { uri, headers } = fileSource(url);
    blob = fetch(uri, { headers })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.blob();
      })
      .then((b) => URL.createObjectURL(b));
    blob.catch(() => blobs.delete(url));
    blobs.set(url, blob);
  }
  return blob;
}

/** Web: RN-web's `<Image>` ignores `source.headers`, so the file is fetched with the token first. */
export function AuthImage({ url, onError, ...props }: AuthImageProps) {
  const [src, setSrc] = useState<string | null>(null);
  useEffect(() => {
    let alive = true;
    setSrc(null);
    load(url).then(
      (s) => alive && setSrc(s),
      () => alive && onError?.({ nativeEvent: { error: "load failed" } } as never),
    );
    return () => {
      alive = false;
    };
    // onError is only called on a failed load; it need not restart the fetch.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [url]);
  if (!src) return <Image source={{ uri: BLANK }} {...props} onError={undefined} />;
  return <Image source={{ uri: src }} {...props} onError={onError} />;
}
