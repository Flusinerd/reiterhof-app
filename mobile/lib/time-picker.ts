/** Conversion between "HH:MM" strings and the `Date` objects the native time pickers work with. */

/** A Date today at the given "HH:MM" (now, when the value is empty or invalid). */
export function timeToDate(value: string): Date {
  const d = new Date();
  const m = /^(\d{1,2}):(\d{2})$/.exec(value.trim());
  if (m && Number(m[1]) <= 23 && Number(m[2]) <= 59) d.setHours(Number(m[1]), Number(m[2]), 0, 0);
  return d;
}

/** "HH:MM" of a Date in local time. */
export function dateToTime(date: Date): string {
  return `${String(date.getHours()).padStart(2, "0")}:${String(date.getMinutes()).padStart(2, "0")}`;
}
