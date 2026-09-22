import { createClient } from "@dash/api-client";

/** Same-origin typed client; vite proxies /api in dev. */
export const api = createClient();

api.use({
	onResponse({ request, response }) {
		// A 401 from a real endpoint means auth is enabled and the session is
		// gone — tell the shell to drop back to the login gate. Auth endpoints
		// handle their own 401s (bad password shows inline).
		if (response.status === 401 && !request.url.includes("/api/auth/")) {
			window.dispatchEvent(new Event("dash:unauthorized"));
		}
		return response;
	},
});
