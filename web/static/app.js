if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch(() => {});
  });
}

window.addEventListener("load", () => {
  const mapEl = document.getElementById("map");
  if (!mapEl || typeof L === "undefined") return;

  const map = L.map("map").setView([14.5995, 120.9842], 6);
  L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
    maxZoom: 19,
    attribution: "&copy; OpenStreetMap contributors"
  }).addTo(map);

  const points = Array.from(document.querySelectorAll("[data-lat][data-lng]"))
    .map((el) => ({
      lat: parseFloat(el.getAttribute("data-lat") || "0"),
      lng: parseFloat(el.getAttribute("data-lng") || "0"),
      title: el.getAttribute("data-title") || "",
      location: el.getAttribute("data-location") || ""
    }))
    .filter((p) => !Number.isNaN(p.lat) && !Number.isNaN(p.lng) && (p.lat !== 0 || p.lng !== 0));

  const latLngs = [];
  points.forEach((p) => {
    const marker = L.marker([p.lat, p.lng]).addTo(map);
    marker.bindPopup(`<b>${p.title}</b><br>${p.location}`);
    latLngs.push([p.lat, p.lng]);
  });
  if (latLngs.length > 0) {
    const line = L.polyline(latLngs, { color: "#2563eb" }).addTo(map);
    map.fitBounds(line.getBounds(), { padding: [20, 20] });
  }
});
