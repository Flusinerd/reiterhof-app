import { createEventInCalendarAsync } from "expo-calendar/legacy";
import { Share } from "react-native";

import { icsUrl } from "./api/requests";
import { eventRange, formatWhen, requestTitle, type HelpRequest } from "./requests";

export type CalendarResult = "calendar" | "shared" | "failed";

/**
 * Adds a request to the device calendar through the system's "new event" screen
 * (no calendar permission needed). If that is not available (Expo Go, web), the
 * event text and the ICS address are shared instead.
 */
export async function addToDeviceCalendar(r: HelpRequest): Promise<CalendarResult> {
  const range = eventRange(r);
  if (!range) return "failed";
  const notes = [r.description, r.tasks.length > 0 ? `Checkliste: ${r.tasks.join(", ")}` : "", `Angefragt von ${r.creator_name}`]
    .filter(Boolean)
    .join("\n");
  try {
    await createEventInCalendarAsync({
      title: requestTitle(r),
      startDate: range.start,
      endDate: range.end,
      allDay: range.allDay,
      location: r.location || undefined,
      notes,
      timeZone: "Europe/Berlin",
    });
    return "calendar";
  } catch {
    return shareRequest(r);
  }
}

/** Share sheet with the event text and the address of the ICS file. */
export async function shareRequest(r: HelpRequest): Promise<CalendarResult> {
  const lines = [requestTitle(r), formatWhen(r), r.location, `Kalenderdatei: ${icsUrl(r.id)}`].filter(Boolean);
  try {
    await Share.share({ message: lines.join("\n") });
    return "shared";
  } catch {
    return "failed";
  }
}

/** Share sheet with the address of the ICS feed of all requests the user helps with. */
export async function shareCalendarFeed(): Promise<CalendarResult> {
  try {
    await Share.share({ message: `Meine Reiterhof-Termine: ${icsUrl()}` });
    return "shared";
  } catch {
    return "failed";
  }
}
