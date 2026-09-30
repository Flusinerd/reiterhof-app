// Incremental parser for text/event-stream (pure, unit-tested).

export type SseMessage = {
  /** `event:` field; "message" when absent. */
  event: string;
  /** Joined `data:` lines. */
  data: string;
};

/**
 * Feed decoded text chunks with `push`; complete events are returned. Handles chunks that
 * split lines or events, CRLF, comments (`:` lines, e.g. the heartbeat) and multi-line data.
 */
export function createSseParser() {
  let buffer = "";
  let event = "";
  let data: string[] = [];

  function flush(out: SseMessage[]) {
    if (data.length > 0) out.push({ event: event || "message", data: data.join("\n") });
    event = "";
    data = [];
  }

  return {
    push(chunk: string): SseMessage[] {
      buffer += chunk;
      const out: SseMessage[] = [];
      let nl: number;
      while ((nl = buffer.search(/\r\n|\n|\r/)) !== -1) {
        const sep = buffer.startsWith("\r\n", nl) ? 2 : 1;
        // A lone trailing "\r" may be the first half of "\r\n": wait for more input.
        if (buffer[nl] === "\r" && nl + 1 === buffer.length) break;
        const line = buffer.slice(0, nl);
        buffer = buffer.slice(nl + sep);
        if (line === "") {
          flush(out);
        } else if (line.startsWith(":")) {
          // comment / heartbeat
        } else {
          const colon = line.indexOf(":");
          const field = colon === -1 ? line : line.slice(0, colon);
          let value = colon === -1 ? "" : line.slice(colon + 1);
          if (value.startsWith(" ")) value = value.slice(1);
          if (field === "event") event = value;
          else if (field === "data") data.push(value);
        }
      }
      return out;
    },
  };
}
