// Generated-client entrypoint. Types come from src/schema.d.ts, produced by
// `npm run generate` from ../../openapi.yaml — the API contract.
import createFetchClient from "openapi-fetch";
import type { paths } from "./schema.d.ts";

export type { paths, components } from "./schema.d.ts";

/** API client bound to the OpenAPI contract. Same-origin by default. */
export function createClient(baseUrl = "") {
	return createFetchClient<paths>({ baseUrl });
}
