const CACHE = "remi-trip-planner-v26";
// Do not precache "/" — HTML must always come from the network so UI updates (templates) are not stuck on an old install snapshot.
const CORE_ASSETS = [
  "/static/app.css",
  "/static/app.js",
  "/manifest.webmanifest"
];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll(CORE_ASSETS)));
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((key) => key !== CACHE).map((key) => caches.delete(key)))
    )
  );
  event.waitUntil(self.clients.claim());
});

self.addEventListener("fetch", (event) => {
  if (event.request.method !== "GET") return;
  const reqURL = new URL(event.request.url);
  const ownOrigin = new URL(self.registration.scope).origin;

  // Third-party requests (Google Maps, fonts, etc.): do not intercept. Passing them through
  // the default handler caused unhandled "Failed to fetch" when fetch failed (blocked, offline,
  // or non-cacheable cross-origin responses).
  if (reqURL.origin !== ownOrigin) {
    return;
  }

  // Network-first for navigations; never fall back to a stale cached "/" dashboard.
  if (event.request.mode === "navigate") {
    event.respondWith(
      fetch(event.request).then((res) => res).catch(() => caches.match(event.request))
    );
    return;
  }

  // Trip page HTML requested via fetch() (live refresh after forms) must hit the network — cache-first
  // below would serve a stale document and drop itinerary day groups until a full reload.
  const acceptHdr = event.request.headers.get("Accept") || "";
  if (reqURL.pathname.startsWith("/trips/") && acceptHdr.includes("text/html")) {
    event.respondWith(fetch(event.request));
    return;
  }

  // For app static assets, prefer network so CSS/JS updates apply immediately after saves.
  if (reqURL.pathname.startsWith("/static/")) {
    event.respondWith(
      fetch(event.request)
        .then((res) => {
          if (res.ok) {
            const clone = res.clone();
            caches.open(CACHE).then((cache) => cache.put(event.request, clone));
          }
          return res;
        })
        .catch(() => caches.match(event.request))
    );
    return;
  }

  // Live autocomplete / geodata: never cache-first — stale empty JSON was breaking airport & location picks.
  if (
    reqURL.pathname.startsWith("/api/location/") ||
    reqURL.pathname.startsWith("/api/flight-airports/") ||
    reqURL.pathname.startsWith("/api/flight-airlines/")
  ) {
    event.respondWith(fetch(event.request));
    return;
  }

  // Stop weather JSON must always be fresh (3h server cache); avoid SW caching stale responses.
  if (/\/itinerary\/[^/]+\/weather$/.test(reqURL.pathname)) {
    event.respondWith(fetch(event.request));
    return;
  }

  event.respondWith(
    caches.match(event.request).then((cached) => {
      if (cached) return cached;
      return fetch(event.request)
        .then((res) => {
          if (res.ok) {
            const clone = res.clone();
            caches.open(CACHE).then((cache) => cache.put(event.request, clone));
          }
          return res;
        })
        .catch(() => new Response("", { status: 503, statusText: "Network Unavailable" }));
    })
  );
});
