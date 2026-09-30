import { API_URL } from "./api";
import { icsUrl } from "./api/requests";
import { getToken } from "./token";
import { deliverFile } from "./web-download";
import { icsFileName } from "./web-share-core";
import type { HelpRequest } from "./requests";

export type CalendarResult = "calendar" | "shared" | "failed";

/**
 * Web variant of `lib/requests-calendar.ts`. A browser cannot open the system's "new event"
 * screen, so the ICS file of the request is fetched (the endpoint needs the bearer token, which
 * a plain link cannot send) and handed over: the iOS share sheet ("In Kalender öffnen" /
 * "Zu Kalender hinzufügen") where the Web Share API can share files, otherwise a download.
 */
async function fetchIcs(path: string): Promise<Blob | null> {
  const token = getToken();
  try {
    const res = await fetch(`${API_URL}${path}`, { headers: token ? { Authorization: `Bearer ${token}` } : {} });
    if (!res.ok) return null;
    const text = await res.text();
    return new Blob([text], { type: "text/calendar" });
  } catch {
    return null;
  }
}

async function deliver(path: string, fileName: string, title: string): Promise<CalendarResult> {
  const blob = await fetchIcs(path);
  if (!blob) return "failed";
  try {
    const result = await deliverFile(blob, fileName, title);
    return result === "shared" || result === "cancelled" ? "shared" : "calendar";
  } catch {
    return "failed";
  }
}

const pathOf = (id?: string) => icsUrl(id).slice(API_URL.length);

export function addToDeviceCalendar(r: HelpRequest): Promise<CalendarResult> {
  return deliver(pathOf(r.id), icsFileName(r.id), "Stallfunk-Termin");
}

export function shareRequest(r: HelpRequest): Promise<CalendarResult> {
  return addToDeviceCalendar(r);
}

/** The ICS of all requests the user helps with, as a file (a snapshot, not a subscription). */
export function shareCalendarFeed(): Promise<CalendarResult> {
  return deliver(pathOf(), icsFileName(), "Meine Stallfunk-Termine");
}
