const CACHE = "remi-trip-planner-v15";
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

  // Network-first for navigations; never fall back to a stale cached "/" dashboard.
  if (event.request.mode === "navigate") {
    event.respondWith(
      fetch(event.request).then((res) => res).catch(() => caches.match(event.request))
    );
    return;
  }

  // For app static assets, prefer network so CSS/JS updates apply immediately after saves.
  if (reqURL.pathname.startsWith("/static/")) {
    event.respondWith(
      fetch(event.request)
        .then((res) => {
          const clone = res.clone();
          caches.open(CACHE).then((cache) => cache.put(event.request, clone));
          return res;
        })
        .catch(() => caches.match(event.request))
    );
    return;
  }

  event.respondWith(
    caches.match(event.request).then((cached) => {
      return cached || fetch(event.request).then((res) => {
        const clone = res.clone();
        caches.open(CACHE).then((cache) => cache.put(event.request, clone));
        return res;
      });
    })
  );
});
