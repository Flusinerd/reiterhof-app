import { LegalScreen } from "@/components/consent-legal-view";
import { IMPRINT_MD, LEGAL_TEXT_VERSION } from "@/lib/consent-legal-texts";

/** Impressum. Reachable without a session (sign-in screen links here). */
export default function Imprint() {
  return <LegalScreen markdown={IMPRINT_MD} eyebrow="Rechtliches" version={LEGAL_TEXT_VERSION} />;
}
