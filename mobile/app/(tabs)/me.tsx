import { Settings } from "../settings/index";

/** "Ich": the user's links, notifications and account (the settings screen without back button). */
export default function MeTab() {
  return <Settings back={false} />;
}
