/** Conversion between "YYYY-MM-DD" strings and the `Date` objects the native date pickers work with. */

/** A local Date at noon of the given "YYYY-MM-DD" calendar day (today when empty or invalid). */
export function isoToDate(value: string): Date {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value.trim());
  if (m) {
    const [y, mo, d] = [Number(m[1]), Number(m[2]), Number(m[3])];
    // Noon keeps the calendar day stable across DST changes and time zone offsets.
    const date = new Date(y, mo - 1, d, 12, 0, 0, 0);
    if (date.getFullYear() === y && date.getMonth() === mo - 1 && date.getDate() === d) return date;
  }
  const today = new Date();
  today.setHours(12, 0, 0, 0);
  return today;
}

/** "YYYY-MM-DD" of a Date in local time. */
export function dateToIso(date: Date): string {
  const y = String(date.getFullYear()).padStart(4, "0");
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}
