import { Image, type ImageProps } from "react-native";

import { fileSource } from "@/lib/upload";

export type AuthImageProps = Omit<ImageProps, "source"> & {
  /** Relative API path of a stored file ("/api/v1/files/..."). */
  url: string;
};

/**
 * Image of a stored file. Files need the session token, which native `<Image>` sends as a header
 * (`fileSource`). The web build cannot do that, so `auth-image.web.tsx` fetches the file with the
 * header and shows it from a blob URL.
 */
export function AuthImage({ url, ...props }: AuthImageProps) {
  return <Image source={fileSource(url)} {...props} />;
}
