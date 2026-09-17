import { createClient } from "@dash/api-client";

/** Same-origin typed client; vite proxies /api in dev. */
export const api = createClient();
