import { LegalScreen } from "@/components/consent-legal-view";
import { LEGAL_TEXT_VERSION, PRIVACY_MD } from "@/lib/consent-legal-texts";

/** Datenschutzerklärung. Reachable without a session (sign-in screen links here). */
export default function PrivacyText() {
  return <LegalScreen markdown={PRIVACY_MD} eyebrow="Rechtliches" version={LEGAL_TEXT_VERSION} />;
}
