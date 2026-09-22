// Dash service worker: cache-first for static assets, network-first for /api.
// Bump CACHE on each deploy shape change; clients drop stale caches on activate.
const CACHE = "dash-v1";
const ASSET = /\.(?:js|css|woff2?|png|svg|webmanifest|ico)$/;

self.addEventListener("install", (e) => {
	self.skipWaiting();
	e.waitUntil(caches.open(CACHE).then((c) => c.add("/")));
});

self.addEventListener("activate", (e) => {
	e.waitUntil(
		caches.keys()
			.then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
			.then(() => self.clients.claim()),
	);
});

self.addEventListener("fetch", (e) => {
	const url = new URL(e.request.url);
	if (e.request.method !== "GET" || url.origin !== self.location.origin) return;

	// API: network-first so widgets stay live; fall back to cache offline.
	if (url.pathname.startsWith("/api/")) {
		e.respondWith(
			fetch(e.request)
				.then((res) => {
					const clone = res.clone();
					if (res.ok) caches.open(CACHE).then((c) => c.put(e.request, clone));
					return res;
				})
				.catch(() => caches.match(e.request)),
		);
		return;
	}

	// Navigations: network-first, fall back to cached "/" (read-only offline).
	if (e.request.mode === "navigate") {
		e.respondWith(fetch(e.request).catch(() => caches.match("/")));
		return;
	}

	// Static assets: cache-first, populate on miss.
	if (ASSET.test(url.pathname)) {
		e.respondWith(
			caches.match(e.request).then(
				(hit) =>
					hit ||
					fetch(e.request).then((res) => {
						const clone = res.clone();
						if (res.ok) caches.open(CACHE).then((c) => c.put(e.request, clone));
						return res;
					}),
			),
		);
	}
});
