if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch(() => {});
  });
}

(function syncVisualViewportHeightVar() {
  const setAppVVh = () => {
    const vv = window.visualViewport;
    const h = vv ? vv.height : window.innerHeight;
    document.documentElement.style.setProperty("--app-vvh", `${Math.round(h)}px`);
  };
  if (window.visualViewport) {
    window.visualViewport.addEventListener("resize", setAppVVh);
    window.visualViewport.addEventListener("scroll", setAppVVh);
  }
  window.addEventListener("resize", setAppVVh);
  window.addEventListener("orientationchange", setAppVVh);
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", setAppVVh, { once: true });
  } else {
    setAppVVh();
  }
})();

window.addEventListener("load", () => {
  const csrfMeta = document.querySelector("meta[name='csrf-token']");
  const csrfToken = csrfMeta && csrfMeta.getAttribute("content") ? csrfMeta.getAttribute("content").trim() : "";
  if (csrfToken) {
    document.querySelectorAll("form[method='post'], form[method='POST']").forEach((form) => {
      if (form.querySelector("input[name='csrf_token']")) return;
      const action = (form.getAttribute("action") || "").toLowerCase();
      if (action === "/login" || action === "/setup" || action === "/register" || action.endsWith("/logout")) return;
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = "csrf_token";
      input.value = csrfToken;
      form.appendChild(input);
    });
  }

  document.querySelectorAll("[data-password-toggle]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const id = btn.getAttribute("data-password-toggle");
      const input = id ? document.getElementById(id) : null;
      if (!input || (input.type !== "password" && input.type !== "text")) return;
      const show = input.type === "password";
      input.type = show ? "text" : "password";
      btn.setAttribute("aria-pressed", show ? "true" : "false");
      btn.setAttribute("aria-label", show ? "Hide password" : "Show password");
      btn.setAttribute("title", show ? "Hide password" : "Show password");
      const icon = btn.querySelector(".material-symbols-outlined");
      if (icon) icon.textContent = show ? "visibility_off" : "visibility";
    });
  });

  const TOAST_KEY = "remi-toast-message";

  const showToast = (message) => {
    if (!message) return;
    const toast = document.createElement("div");
    toast.className = "app-toast";
    toast.textContent = message;
    document.body.appendChild(toast);
    window.setTimeout(() => {
      toast.classList.add("visible");
    }, 10);
    window.setTimeout(() => {
      toast.classList.remove("visible");
      window.setTimeout(() => {
        toast.remove();
      }, 220);
    }, 2600);
  };
  window.remiShowToast = showToast;

  try {
    const pendingToast = sessionStorage.getItem(TOAST_KEY);
    if (pendingToast) {
      sessionStorage.removeItem(TOAST_KEY);
      showToast(pendingToast);
    }
  } catch (e) {
    /* ignore */
  }

  const tripInviteFlash = document.querySelector("[data-trip-invite-flash]");
  if (tripInviteFlash) {
    try {
      const url = new URL(window.location.href);
      if (url.searchParams.has("invite_notice") || url.searchParams.has("invite_email")) {
        url.searchParams.delete("invite_notice");
        url.searchParams.delete("invite_email");
        const q = url.searchParams.toString();
        window.history.replaceState(null, "", url.pathname + (q ? `?${q}` : "") + url.hash);
      }
    } catch (e) {
      /* ignore */
    }
    window.setTimeout(() => {
      tripInviteFlash.remove();
    }, 3000);
  }

  const profileSavedBanner = document.querySelector(".profile-updated-banner");
  if (profileSavedBanner) {
    try {
      const url = new URL(window.location.href);
      if (url.searchParams.get("saved") === "1") {
        url.searchParams.delete("saved");
        const q = url.searchParams.toString();
        window.history.replaceState(null, "", url.pathname + (q ? `?${q}` : "") + url.hash);
      }
    } catch (e) {
      /* ignore */
    }
    window.setTimeout(() => {
      profileSavedBanner.remove();
    }, 3000);
  }

  document.querySelectorAll(".site-settings-flash").forEach((el) => {
    window.setTimeout(() => {
      el.remove();
    }, 3000);
  });

  /** One fetch per trip (sidebar + mobile each render tripMembersPanel; parallel POSTs broke SQLite / raced). */
  const inviteLinkPromiseByTrip = new Map();
  const inviteRootsByTrip = new Map();
  document.querySelectorAll("[data-trip-invite-methods]").forEach((root) => {
    const tripId = root.getAttribute("data-trip-id");
    if (!tripId) return;
    if (!inviteRootsByTrip.has(tripId)) inviteRootsByTrip.set(tripId, []);
    inviteRootsByTrip.get(tripId).push(root);
  });

  inviteRootsByTrip.forEach((roots, tripId) => {
    const csrf = roots[0].getAttribute("data-csrf");
    if (!csrf) return;

    const allLinkInputs = () =>
      roots.map((r) => r.querySelector(".sidebar-invite-link-url")).filter(Boolean);

    const requestInviteLink = () => {
      let p = inviteLinkPromiseByTrip.get(tripId);
      if (p) return p;
      const fd = new FormData();
      fd.set("csrf_token", csrf);
      p = fetch(`/trips/${encodeURIComponent(tripId)}/invite-link`, {
        method: "POST",
        body: fd,
        headers: {
          "X-Requested-With": "XMLHttpRequest",
          Accept: "application/json"
        }
      })
        .then((res) => {
          if (!res.ok) {
            return res.text().then((txt) => {
              throw new Error(txt || res.statusText);
            });
          }
          return res.json();
        })
        .then((data) => {
          if (data && data.url) {
            allLinkInputs().forEach((input) => {
              input.value = data.url;
            });
          }
        })
        .catch(() => {
          inviteLinkPromiseByTrip.delete(tripId);
          if (typeof window.remiShowToast === "function") {
            window.remiShowToast("Could not create invite link. Try again.");
          }
        });
      inviteLinkPromiseByTrip.set(tripId, p);
      return p;
    };

    roots.forEach((root) => {
      const tabs = root.querySelectorAll("[data-invite-tab]");
      const panels = root.querySelectorAll("[data-invite-panel]");
      const linkInput = root.querySelector(".sidebar-invite-link-url");

      const showPanel = (name) => {
        panels.forEach((p) => {
          const on = p.getAttribute("data-invite-panel") === name;
          p.classList.toggle("hidden", !on);
          if (on) {
            p.removeAttribute("hidden");
          } else {
            p.setAttribute("hidden", "true");
          }
        });
        tabs.forEach((t) => {
          const sel = t.getAttribute("data-invite-tab") === name;
          t.setAttribute("aria-selected", sel ? "true" : "false");
          t.classList.toggle("sidebar-invite-tab--active", sel);
        });
        if (name === "link") {
          requestInviteLink();
        }
      };

      tabs.forEach((tab) => {
        tab.addEventListener("click", () => {
          showPanel(tab.getAttribute("data-invite-tab") || "email");
        });
      });

      if (tabs.length === 0) {
        showPanel("link");
      }

      const copyInviteLink = () => {
        if (!(linkInput instanceof HTMLInputElement)) return;
        const v = (linkInput.value || "").trim();
        if (!v) {
          if (typeof window.remiShowToast === "function") {
            window.remiShowToast("Wait for the invite link to load, then try again.");
          }
          return;
        }
        navigator.clipboard.writeText(v).then(
          () => {
            if (typeof window.remiShowToast === "function") {
              window.remiShowToast("Invite link copied.");
            }
          },
          () => {
            if (typeof window.remiShowToast === "function") {
              window.remiShowToast("Could not copy. Select the URL and copy manually.");
            }
          }
        );
      };

      const linkFieldWrap = root.querySelector(".sidebar-invite-link-field");
      if (linkFieldWrap && linkInput) {
        linkFieldWrap.addEventListener("click", () => {
          copyInviteLink();
        });
        if (linkInput instanceof HTMLInputElement) {
          linkInput.addEventListener("click", (e) => {
            e.preventDefault();
            copyInviteLink();
          });
        }
      }
    });
  });

  document.querySelectorAll("[data-remi-tap-copy]").forEach((root) => {
    if (root.closest("[data-trip-invite-methods]")) return;
    const target = root.querySelector("[data-remi-tap-copy-target]") || root.querySelector("input[readonly], textarea[readonly]");
    if (!target || (!(target instanceof HTMLInputElement) && !(target instanceof HTMLTextAreaElement))) return;
    const toastOk = root.getAttribute("data-remi-tap-copy-toast") || "Copied.";
    const copyFn = (ev) => {
      if (ev) ev.preventDefault();
      const v = (target.value || "").trim();
      if (!v) {
        if (typeof window.remiShowToast === "function") window.remiShowToast("Nothing to copy yet.");
        return;
      }
      navigator.clipboard.writeText(v).then(
        () => {
          if (typeof window.remiShowToast === "function") window.remiShowToast(toastOk);
          try {
            target.select();
          } catch (e) {
            /* ignore */
          }
        },
        () => {
          if (typeof window.remiShowToast === "function") {
            window.remiShowToast("Could not copy. Select the text and copy manually.");
          }
        }
      );
    };
    root.addEventListener("click", copyFn);
  });

  const syncThemeIcons = () => {
    const dark = document.documentElement.classList.contains("theme-dark");
    document.querySelectorAll("[data-theme-icon]").forEach((el) => {
      el.textContent = dark ? "light_mode" : "dark_mode";
    });
  };

  document.querySelectorAll("[data-theme-toggle]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const currentlyDark = document.documentElement.classList.contains("theme-dark");
      const nextPref = currentlyDark ? "light" : "dark";
      try {
        localStorage.setItem("remi_theme_override", nextPref);
      } catch (e) {
        /* ignore */
      }
      try {
        document.documentElement.setAttribute("data-theme-pref", nextPref);
      } catch (e2) {
        /* ignore */
      }
      document.documentElement.classList.toggle("theme-dark", nextPref === "dark");
      syncThemeIcons();
      document.dispatchEvent(new CustomEvent("remi:themechange", { detail: { dark: nextPref === "dark" } }));
    });
  });
  syncThemeIcons();

  const syncViewportBottomOffset = () => {
    let offset = 0;
    if (window.visualViewport) {
      const vv = window.visualViewport;
      offset = Math.max(0, window.innerHeight - (vv.height + vv.offsetTop));
    }
    document.documentElement.style.setProperty("--viewport-bottom-offset", `${Math.round(offset)}px`);
  };
  syncViewportBottomOffset();
  window.addEventListener("resize", syncViewportBottomOffset);
  window.addEventListener("orientationchange", syncViewportBottomOffset);
  if (window.visualViewport) {
    window.visualViewport.addEventListener("resize", syncViewportBottomOffset);
    window.visualViewport.addEventListener("scroll", syncViewportBottomOffset);
  }

  const pad2 = (n) => String(n).padStart(2, "0");
  const now = new Date();
  const todayLocal = `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
  const nowLocal = `${todayLocal}T${pad2(now.getHours())}:${pad2(now.getMinutes())}`;
  document.querySelectorAll("input[type='date'], input[type='datetime-local']").forEach((input) => {
    if ((input.value || "").trim() !== "") return;
    if (input.type === "date") {
      let value = todayLocal;
      const min = (input.min || "").trim();
      const max = (input.max || "").trim();
      if (min && value < min) value = min;
      if (max && value > max) value = max;
      input.value = value;
      return;
    }
    if (input.type === "datetime-local") {
      let value = nowLocal;
      const min = (input.min || "").trim();
      const max = (input.max || "").trim();
      if (min && value < min) value = min;
      if (max && value > max) value = max;
      input.value = value;
    }
  });

  const normalize = (value) => (value || "").trim().toLowerCase();

  const remiSuggestURL = () => {
    const shell = document.querySelector("main.app-shell");
    const u = shell && shell.getAttribute("data-location-suggest-url");
    return (u && u.trim()) || "/api/location/suggest";
  };
  const remiGeocodeURL = () => "/api/location/geocode";
  const remiReadDistanceUnit = () => {
    const shell = document.querySelector("main.app-shell");
    const u = shell && shell.getAttribute("data-distance-unit");
    return u && String(u).trim() === "mi" ? "mi" : "km";
  };

  const toRad = (deg) => (deg * Math.PI) / 180;
  const haversineKm = (lat1, lng1, lat2, lng2) => {
    const dLat = toRad(lat2 - lat1);
    const dLng = toRad(lng2 - lng1);
    const a = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
      Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) *
      Math.sin(dLng / 2) * Math.sin(dLng / 2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    return 6371 * c;
  };
  const formatDistance = (km) => {
    const unit = remiReadDistanceUnit();
    if (!Number.isFinite(km) || km <= 0) return unit === "mi" ? "0 mi" : "0 km";
    if (unit === "mi") {
      const mi = km * 0.621371;
      if (mi < 0.05) return "0 mi";
      if (mi < 0.25) return `${Math.round(mi * 5280)} ft`;
      return `${mi.toFixed(1)} mi`;
    }
    if (km < 1) return `${Math.round(km * 1000)} m`;
    return `${km.toFixed(1)} km`;
  };
  const formatDuration = (mins) => {
    const rounded = Math.max(1, Math.round(mins));
    if (rounded < 60) return `${rounded} min`;
    const h = Math.floor(rounded / 60);
    const m = rounded % 60;
    return m === 0 ? `${h} hr` : `${h} hr ${m} min`;
  };
  const parseCoords = (itemEl) => {
    const lat = parseFloat(itemEl.getAttribute("data-lat") || "");
    const lng = parseFloat(itemEl.getAttribute("data-lng") || "");
    if (Number.isNaN(lat) || Number.isNaN(lng)) return null;
    if (lat === 0 && lng === 0) return null;
    return { lat, lng };
  };
  const directionsURL = (from, to, mode) =>
    `https://www.google.com/maps/dir/?api=1&origin=${from.lat},${from.lng}&destination=${to.lat},${to.lng}&travelmode=${mode}`;
  const connectorCoordsCache = new Map();

  /** Must be declared before fillMissingCoords / renderItineraryConnectors (avoid TDZ ReferenceError on load). */
  const geocodeLocation = async (locationQuery) => {
    const q = (locationQuery || "").trim();
    if (!q) return null;
    try {
      const res = await fetch(`${remiGeocodeURL()}?q=${encodeURIComponent(q)}`, {
        credentials: "same-origin",
        headers: { Accept: "application/json" }
      });
      if (res.ok) {
        const top = await res.json();
        const lat = parseFloat(top.lat ?? top.Lat ?? "0");
        const lng = parseFloat(top.lng ?? top.Lon ?? "0");
        if (Number.isFinite(lat) && Number.isFinite(lng) && (lat !== 0 || lng !== 0)) {
          return {
            lat,
            lng,
            displayName: top.displayName || q
          };
        }
      }
    } catch (e) {
      /* fallback */
    }
    const url = new URL("https://nominatim.openstreetmap.org/search");
    url.searchParams.set("q", q);
    url.searchParams.set("format", "jsonv2");
    url.searchParams.set("limit", "1");
    const response = await fetch(url.toString(), {
      headers: {
        Accept: "application/json"
      }
    });
    if (!response.ok) {
      return null;
    }
    const data = await response.json();
    if (!Array.isArray(data) || data.length === 0) {
      return null;
    }
    const top = data[0];
    return {
      lat: parseFloat(top.lat || "0"),
      lng: parseFloat(top.lon || "0"),
      displayName: top.display_name || q
    };
  };

  const fillMissingCoords = async (scopes) => {
    const byLocation = new Map();
    scopes.forEach((scope) => {
      scope.querySelectorAll(".day-items.timeline .timeline-item[data-itinerary-item]").forEach((el) => {
        if (parseCoords(el)) return;
        const loc = (
          el.getAttribute("data-geocode-location") ||
          el.getAttribute("data-location") ||
          ""
        ).trim();
        if (!loc) return;
        if (!byLocation.has(loc)) byLocation.set(loc, []);
        byLocation.get(loc).push(el);
      });
    });
    if (byLocation.size === 0) return;

    const applyCoords = (els, coords) => {
      if (!coords) return;
      els.forEach((el) => {
        el.setAttribute("data-lat", coords.lat);
        el.setAttribute("data-lng", coords.lng);
      });
    };

    const fetches = [];
    for (const [loc, els] of byLocation) {
      if (connectorCoordsCache.has(loc)) {
        applyCoords(els, connectorCoordsCache.get(loc));
      } else {
        fetches.push(
          geocodeLocation(loc).then((result) => {
            const lat = result?.lat;
            const lng = result?.lng;
            const coords =
              Number.isFinite(lat) && Number.isFinite(lng) ? { lat, lng } : null;
            connectorCoordsCache.set(loc, coords);
            applyCoords(els, coords);
          })
        );
      }
    }
    if (fetches.length > 0) await Promise.all(fetches);
  };

  const buildTimelineConnectorLi = (from, to, distanceKm) => {
    const driveMins = (distanceKm / 35) * 60;
    const walkMins = (distanceKm / 4.8) * 60;
    const transitMins = (distanceKm / 14) * 60;

    const li = document.createElement("li");
    li.className = "timeline-connector";
    li.setAttribute("data-itinerary-connector", "");
    li.setAttribute("role", "presentation");

    const rail = document.createElement("div");
    rail.className = "timeline-connector-rail";
    rail.setAttribute("aria-hidden", "true");

    const details = document.createElement("details");
    details.className = "timeline-connector-details";

    const summary = document.createElement("summary");
    summary.className = "timeline-connector-summary";
    summary.setAttribute("aria-label", "Travel time and directions between stops");

    const main = document.createElement("span");
    main.className = "timeline-connector-summary-main";

    const carIcon = document.createElement("span");
    carIcon.className = "material-symbols-outlined timeline-connector-mode-icon";
    carIcon.setAttribute("aria-hidden", "true");
    carIcon.textContent = "directions_car";

    const meta = document.createElement("span");
    meta.className = "timeline-connector-meta";
    meta.textContent = `${formatDuration(driveMins)} · ${formatDistance(distanceKm)}`;

    const chev = document.createElement("span");
    chev.className = "material-symbols-outlined timeline-connector-chevron";
    chev.setAttribute("aria-hidden", "true");
    chev.textContent = "expand_more";

    const dirA = document.createElement("a");
    dirA.className = "timeline-connector-directions";
    dirA.href = directionsURL(from, to, "driving");
    dirA.target = "_blank";
    dirA.rel = "noopener noreferrer";
    dirA.textContent = "Directions";
    dirA.addEventListener("click", (e) => e.stopPropagation());

    main.append(carIcon, meta, dirA, chev);

    summary.append(main);

    const menu = document.createElement("div");
    menu.className = "timeline-connector-dropdown";

    const makeOption = (iconName, durationLabel, mode) => {
      const opt = document.createElement("a");
      opt.className = "timeline-connector-option";
      opt.href = directionsURL(from, to, mode);
      opt.target = "_blank";
      opt.rel = "noopener noreferrer";
      const ic = document.createElement("span");
      ic.className = "material-symbols-outlined timeline-connector-option-icon";
      ic.setAttribute("aria-hidden", "true");
      ic.textContent = iconName;
      const tx = document.createElement("span");
      tx.className = "timeline-connector-option-text";
      tx.textContent = durationLabel;
      opt.append(ic, tx);
      return opt;
    };

    menu.append(
      makeOption(
        "directions_transit",
        `${formatDuration(transitMins)} · ${formatDistance(distanceKm)}`,
        "transit"
      ),
      makeOption(
        "directions_walk",
        `${formatDuration(walkMins)} · ${formatDistance(distanceKm)}`,
        "walking"
      )
    );

    details.append(summary, menu);
    li.append(rail, details);
    return li;
  };

  const drawConnectors = (scopes) => {
    scopes.forEach((scope) => {
      scope.querySelectorAll(".day-items.timeline").forEach((list) => {
        list.querySelectorAll("[data-itinerary-connector]").forEach((el) => el.remove());
        const items = Array.from(list.querySelectorAll(".timeline-item[data-itinerary-item]"))
          .filter((el) => !el.classList.contains("itinerary-search-hidden"));
        for (let i = 0; i < items.length - 1; i++) {
          const current = items[i];
          const next = items[i + 1];
          const from = parseCoords(current);
          const to = parseCoords(next);
          if (!from || !to) continue;
          const distanceKm = haversineKm(from.lat, from.lng, to.lat, to.lng);
          if (!Number.isFinite(distanceKm) || distanceKm <= 0.05) continue;
          next.insertAdjacentElement("beforebegin", buildTimelineConnectorLi(from, to, distanceKm));
        }
      });
    });
  };

  const renderItineraryConnectors = async (scopeRoot) => {
    const scopes = scopeRoot ? [scopeRoot] : [document];
    await fillMissingCoords(scopes);
    drawConnectors(scopes);
  };
  renderItineraryConnectors(document);

  const itinerarySearchInput = document.querySelector("[data-itinerary-search]");
  const itinerarySearchRoot = document.querySelector("[data-itinerary-search-root]");
  if (itinerarySearchInput && itinerarySearchRoot) {
    const applyItinerarySearch = () => {
      const q = normalize(itinerarySearchInput.value);
      const items = itinerarySearchRoot.querySelectorAll("[data-itinerary-item]");
      items.forEach((li) => {
        const blob = normalize(li.getAttribute("data-search-text") || "");
        const match = !q || blob.includes(q);
        li.classList.toggle("itinerary-search-hidden", !match);
      });
      let anyVisible = false;
      itinerarySearchRoot.querySelectorAll(".day-group").forEach((dg) => {
        const visible = dg.querySelectorAll(".timeline-item:not(.itinerary-search-hidden)").length;
        if (visible > 0) anyVisible = true;
        dg.classList.toggle("day-group--search-empty", q.length > 0 && visible === 0);
        if (q.length > 0 && visible > 0) {
          dg.open = true;
        }
      });
      let emptyEl = itinerarySearchRoot.querySelector("[data-itinerary-search-empty-msg]");
      if (q.length > 0 && !anyVisible) {
        if (!emptyEl) {
          emptyEl = document.createElement("p");
          emptyEl.setAttribute("data-itinerary-search-empty-msg", "");
          emptyEl.className = "muted itinerary-search-empty-msg";
          emptyEl.style.marginTop = "10px";
          emptyEl.textContent = "No itinerary items match your search.";
          const heading = itinerarySearchRoot.querySelector("h3");
          if (heading) heading.insertAdjacentElement("afterend", emptyEl);
        }
      } else if (emptyEl) {
        emptyEl.remove();
      }
      renderItineraryConnectors(itinerarySearchRoot);
    };
    itinerarySearchInput.addEventListener("input", applyItinerarySearch);
    itinerarySearchInput.addEventListener("search", applyItinerarySearch);
    // Sync filter state on load (e.g. bfcache restore, autofill) so day groups are not stuck hidden.
    applyItinerarySearch();
  }

  const searchLocations = async (locationQuery) => {
    const q = (locationQuery || "").trim();
    if (!q) return [];
    try {
      const res = await fetch(`${remiSuggestURL()}?q=${encodeURIComponent(q)}`, {
        credentials: "same-origin",
        headers: { Accept: "application/json" }
      });
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data) && data.length > 0) {
          return data
            .map((item) => {
              const lat = parseFloat(item.lat ?? item.Lat ?? "0");
              const lng = parseFloat(item.lng ?? item.Lon ?? "0");
              const displayName = item.displayName || item.display_name || "";
              const shortName = item.shortName || item.name || (displayName ? displayName.split(",")[0].trim() : "") || displayName;
              return { lat, lng, displayName, shortName };
            })
            .filter((item) => (item.displayName || item.shortName) && !Number.isNaN(item.lat) && !Number.isNaN(item.lng));
        }
      }
    } catch (e) {
      /* fallback */
    }
    const url = new URL("https://nominatim.openstreetmap.org/search");
    url.searchParams.set("q", q);
    url.searchParams.set("format", "jsonv2");
    url.searchParams.set("limit", "5");
    const response = await fetch(url.toString(), {
      headers: {
        Accept: "application/json"
      }
    });
    if (!response.ok) {
      return [];
    }
    const data = await response.json();
    if (!Array.isArray(data)) {
      return [];
    }
    return data
      .map((item) => {
        const lat = parseFloat(item.lat || "0");
        const lng = parseFloat(item.lon || "0");
        const displayName = item.display_name || "";
        const name = item.name && String(item.name).trim() ? String(item.name).trim() : "";
        const shortName = name || (displayName ? displayName.split(",")[0].trim() : "") || displayName;
        return { lat, lng, displayName, shortName };
      })
      .filter((item) => (item.displayName || item.shortName) && !Number.isNaN(item.lat) && !Number.isNaN(item.lng));
  };

  const REMI_LOCATION_SUGGEST_BLUR_MS = 300;
  const remiPreventLocationSuggestBlur = (btn) => {
    btn.addEventListener("mousedown", (e) => {
      e.preventDefault();
    });
  };

  document.querySelectorAll("form[data-dashboard-trip-place]").forEach((heroForm) => {
    const shell = document.querySelector("main.app-shell");
    const lookupEnabled = shell ? shell.getAttribute("data-location-lookup-enabled") !== "false" : true;
    const nameInput = heroForm.querySelector("[data-dashboard-trip-name-input]");
    const latInput = heroForm.querySelector("[data-home-map-lat]");
    const lngInput = heroForm.querySelector("[data-home-map-lng]");
    if (!nameInput || !latInput || !lngInput) {
      return;
    }

    let timer = null;
    let token = 0;
    let suggestionsHost = null;
    let pickedLabel = null;

    const ensureHost = () => {
      if (suggestionsHost) return suggestionsHost;
      suggestionsHost = document.createElement("div");
      suggestionsHost.className = "location-suggestions hidden";
      suggestionsHost.setAttribute("role", "listbox");
      const wrap = nameInput.closest(".hero-trip-name-field") || nameInput.parentElement;
      if (wrap) wrap.appendChild(suggestionsHost);
      return suggestionsHost;
    };

    const hide = () => {
      const host = ensureHost();
      host.classList.add("hidden");
      host.innerHTML = "";
      nameInput.setAttribute("aria-expanded", "false");
    };

    const clearCoords = () => {
      latInput.value = "";
      lngInput.value = "";
      pickedLabel = null;
    };

    nameInput.addEventListener("input", () => {
      if (!lookupEnabled) {
        return;
      }
      if (pickedLabel !== null && nameInput.value.trim() !== pickedLabel) {
        clearCoords();
      }
      if (timer) clearTimeout(timer);
      const query = nameInput.value.trim();
      if (query.length < 3) {
        hide();
        return;
      }
      timer = window.setTimeout(async () => {
        const current = ++token;
        const host = ensureHost();
        const suggestions = await searchLocations(query);
        if (current !== token) return;
        host.innerHTML = "";
        if (!suggestions.length) {
          hide();
          return;
        }
        suggestions.forEach((s) => {
          const btn = document.createElement("button");
          btn.type = "button";
          btn.className = "location-suggestion-btn dashboard-trip-place-suggest";
          const title = document.createElement("span");
          title.className = "dashboard-trip-place-suggest__title";
          title.textContent = s.shortName || s.displayName;
          btn.appendChild(title);
          if (s.displayName && s.displayName !== title.textContent) {
            const sub = document.createElement("span");
            sub.className = "dashboard-trip-place-suggest__detail";
            sub.textContent = s.displayName;
            btn.appendChild(sub);
          }
          remiPreventLocationSuggestBlur(btn);
          btn.addEventListener("click", () => {
            const label = (s.shortName || s.displayName || "").trim();
            nameInput.value = label;
            latInput.value = String(s.lat);
            lngInput.value = String(s.lng);
            pickedLabel = label;
            hide();
          });
          host.appendChild(btn);
        });
        host.classList.remove("hidden");
        nameInput.setAttribute("aria-expanded", "true");
      }, 300);
    });

    nameInput.addEventListener("blur", () => {
      window.setTimeout(hide, REMI_LOCATION_SUGGEST_BLUR_MS);
    });
  });

  document.querySelectorAll("form[data-site-settings-map-form]").forEach((mapForm) => {
    const shell = mapForm.closest(".app-shell");
    const lookupEnabled = shell ? shell.getAttribute("data-location-lookup-enabled") !== "false" : true;
    const nameInput = mapForm.querySelector("[data-site-settings-map-name]");
    const latInput = mapForm.querySelector("[data-site-settings-map-lat]");
    const lngInput = mapForm.querySelector("[data-site-settings-map-lng]");
    if (!nameInput || !latInput || !lngInput) {
      return;
    }

    let timer = null;
    let token = 0;
    let suggestionsHost = null;
    let pickedLabel = (nameInput.value || "").trim() || null;

    const ensureHost = () => {
      if (suggestionsHost) return suggestionsHost;
      suggestionsHost = document.createElement("div");
      suggestionsHost.className = "location-suggestions hidden";
      suggestionsHost.setAttribute("role", "listbox");
      const wrap = nameInput.closest(".site-settings-default-place-field") || nameInput.parentElement;
      if (wrap) wrap.appendChild(suggestionsHost);
      return suggestionsHost;
    };

    const hide = () => {
      const host = ensureHost();
      host.classList.add("hidden");
      host.innerHTML = "";
      nameInput.setAttribute("aria-expanded", "false");
    };

    const clearCoords = () => {
      latInput.value = "";
      lngInput.value = "";
      pickedLabel = null;
    };

    nameInput.addEventListener("input", () => {
      const query = nameInput.value.trim();
      if (query === "") {
        clearCoords();
        hide();
        return;
      }
      if (!lookupEnabled) {
        return;
      }
      if (pickedLabel !== null && query !== pickedLabel) {
        clearCoords();
      }
      if (timer) clearTimeout(timer);
      if (query.length < 3) {
        hide();
        return;
      }
      timer = window.setTimeout(async () => {
        const current = ++token;
        const host = ensureHost();
        const suggestions = await searchLocations(query);
        if (current !== token) return;
        host.innerHTML = "";
        if (!suggestions.length) {
          hide();
          return;
        }
        suggestions.forEach((s) => {
          const btn = document.createElement("button");
          btn.type = "button";
          btn.className = "location-suggestion-btn dashboard-trip-place-suggest";
          const title = document.createElement("span");
          title.className = "dashboard-trip-place-suggest__title";
          title.textContent = s.shortName || s.displayName;
          btn.appendChild(title);
          if (s.displayName && s.displayName !== title.textContent) {
            const sub = document.createElement("span");
            sub.className = "dashboard-trip-place-suggest__detail";
            sub.textContent = s.displayName;
            btn.appendChild(sub);
          }
          remiPreventLocationSuggestBlur(btn);
          btn.addEventListener("click", () => {
            const label = (s.shortName || s.displayName || "").trim();
            nameInput.value = label;
            latInput.value = String(s.lat);
            lngInput.value = String(s.lng);
            pickedLabel = label;
            hide();
          });
          host.appendChild(btn);
        });
        host.classList.remove("hidden");
        nameInput.setAttribute("aria-expanded", "true");
      }, 300);
    });

    nameInput.addEventListener("blur", () => {
      window.setTimeout(hide, REMI_LOCATION_SUGGEST_BLUR_MS);
    });
  });

  const editPanel = document.querySelector("[data-edit-panel]");
  const openEditBtn = document.querySelector("[data-edit-toggle='open']");
  const closeEditBtn = document.querySelector("[data-edit-toggle='close']");
  if (editPanel && openEditBtn) {
    openEditBtn.addEventListener("click", () => {
      const topbarDetails = openEditBtn.closest("details.trip-inline-actions-dropdown");
      if (topbarDetails) {
        topbarDetails.removeAttribute("open");
      }
      editPanel.classList.remove("hidden");
      editPanel.scrollIntoView({ behavior: "smooth", block: "start" });
    });
  }
  if (editPanel) {
    const open = new URLSearchParams(window.location.search).get("open");
    if (open === "edit") {
      editPanel.classList.remove("hidden");
      window.setTimeout(() => {
        editPanel.scrollIntoView({ behavior: "smooth", block: "start" });
      }, 40);
    }
  }
  if (editPanel && closeEditBtn) {
    closeEditBtn.addEventListener("click", () => {
      editPanel.classList.add("hidden");
    });
  }

  document.querySelectorAll("[data-trip-reorder-list]").forEach((list) => {
    const fieldName = list.getAttribute("data-trip-reorder-list");
    if (!fieldName) return;
    const form = list.closest("form");
    const hidden = form?.querySelector(`input[type="hidden"][name="${fieldName}"]`);
    if (!hidden) return;
    const syncHidden = () => {
      const keys = [...list.querySelectorAll("[data-reorder-key]")].map((li) => li.getAttribute("data-reorder-key") || "");
      hidden.value = keys.filter(Boolean).join(",");
    };
    let dragEl = null;
    list.querySelectorAll(".trip-layout-reorder-move").forEach((btn) => {
      btn.addEventListener("click", () => {
        const li = btn.closest("[data-reorder-key]");
        const dir = parseInt(btn.getAttribute("data-move") || "0", 10);
        if (!li || !dir) return;
        const sib = dir < 0 ? li.previousElementSibling : li.nextElementSibling;
        if (!sib || !sib.hasAttribute("data-reorder-key")) return;
        if (dir < 0) {
          list.insertBefore(li, sib);
        } else {
          list.insertBefore(sib, li);
        }
        syncHidden();
      });
    });
    list.addEventListener("dragstart", (e) => {
      const li = e.target.closest("[data-reorder-key]");
      if (!li || !list.contains(li)) return;
      dragEl = li;
      e.dataTransfer.effectAllowed = "move";
      try {
        e.dataTransfer.setData("text/plain", li.getAttribute("data-reorder-key") || "");
      } catch (err) {
        /* ignore */
      }
      li.classList.add("trip-layout-reorder-item--dragging");
    });
    list.addEventListener("dragend", (e) => {
      const li = e.target.closest("[data-reorder-key]");
      if (li) li.classList.remove("trip-layout-reorder-item--dragging");
      dragEl = null;
      syncHidden();
    });
    list.addEventListener("dragover", (e) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = "move";
      if (!dragEl) return;
      const li = e.target.closest("[data-reorder-key]");
      if (!li || !list.contains(li) || li === dragEl) return;
      const rect = li.getBoundingClientRect();
      const before = e.clientY < rect.top + rect.height / 2;
      if (before) {
        list.insertBefore(dragEl, li);
      } else {
        list.insertBefore(dragEl, li.nextElementSibling);
      }
    });
    list.addEventListener("drop", (e) => {
      e.preventDefault();
      syncHidden();
    });
  });

  const itineraryViewIdForForm = (formId) => {
    if (!formId) return null;
    if (formId.startsWith("accommodation-itinerary-edit-")) {
      return `itinerary-view-${formId.slice("accommodation-itinerary-edit-".length)}`;
    }
    if (formId.startsWith("vehicle-rental-itinerary-edit-")) {
      return `itinerary-view-${formId.slice("vehicle-rental-itinerary-edit-".length)}`;
    }
    if (formId.startsWith("flight-itinerary-edit-")) {
      return `itinerary-view-${formId.slice("flight-itinerary-edit-".length)}`;
    }
    return formId.replace("-edit-", "-view-");
  };

  const mqBudgetEditMobile = window.matchMedia("(max-width: 920px)");
  const mqTabEditMobile = window.matchMedia("(max-width: 920px)");

  const closeInlineEdit = (formId) => {
    const form = document.getElementById(formId);
    if (!form) return;
    if (form.classList.contains("budget-expense-edit-form")) {
      const expenseId = form.getAttribute("data-budget-expense-id") || "";
      const editTr = expenseId
        ? document.querySelector(`tr.budget-expense-edit-row[data-budget-edit-for="${CSS.escape(expenseId)}"]`)
        : null;
      const viewTr = expenseId
        ? document.querySelector(`tr.budget-tx-view[data-budget-tx-view="${CSS.escape(expenseId)}"]`)
        : null;
      const mobileLi = expenseId
        ? document.querySelector(`li.budget-mobile-tx-item[data-budget-tx-view="${CSS.escape(expenseId)}"]`)
        : null;
      const cell = editTr?.querySelector(".budget-expense-edit-cell");
      if (cell && form.parentElement !== cell) {
        cell.appendChild(form);
      }
      form.classList.add("hidden");
      editTr?.classList.add("hidden");
      viewTr?.classList.remove("editing");
      mobileLi?.classList.remove("editing");
      const dialog = document.getElementById("budget-mobile-expense-edit");
      if (dialog?.open) {
        dialog.close();
      }
      return;
    }
    if (form.classList.contains("tab-expense-edit-form")) {
      const expenseId = form.getAttribute("data-tab-expense-id") || "";
      const editTr = expenseId
        ? document.querySelector(`tr.tab-expense-edit-row[data-tab-edit-for="${CSS.escape(expenseId)}"]`)
        : null;
      const viewTr = expenseId ? document.querySelector(`tr[data-tab-tx-view="${CSS.escape(expenseId)}"]`) : null;
      const viewCard = expenseId
        ? document.querySelector(`.tab-exp-grid-card[data-tab-tx-view="${CSS.escape(expenseId)}"]`)
        : null;
      const cell = editTr?.querySelector(".tab-expense-edit-cell");
      if (cell && form.parentElement !== cell) {
        cell.appendChild(form);
      }
      form.classList.add("hidden");
      editTr?.classList.add("hidden");
      viewTr?.classList.remove("editing");
      viewCard?.classList.remove("editing");
      const dialog = document.getElementById("tab-mobile-expense-edit");
      if (dialog?.open) {
        dialog.close();
      }
      return;
    }
    if (form.classList.contains("tab-settlement-edit-form")) {
      form.classList.add("hidden");
      return;
    }
    form.classList.add("hidden");
    const row = form.closest(".timeline-item, .expense-item, .accommodation-item, .accommodation-card-wrap, .vehicle-rental-item, .flight-card, .reminder-checklist-item");
    if (row) row.classList.remove("editing");
    const viewId = itineraryViewIdForForm(formId);
    const view = viewId ? document.getElementById(viewId) : null;
    if (view) view.classList.remove("hidden");
  };

  const wireOneInlineEditOpenBtn = (btn) => {
    if (!(btn instanceof HTMLElement) || btn.dataset.remiInlineOpenWired === "1") return;
    btn.dataset.remiInlineOpenWired = "1";
    btn.addEventListener("click", () => {
      const actionDetails = btn.closest("details.trip-inline-actions-dropdown");
      if (actionDetails) {
        actionDetails.removeAttribute("open");
      }
      const formId = btn.getAttribute("data-inline-edit-open");
      const form = formId ? document.getElementById(formId) : null;
      if (!form) return;
      if (form.classList.contains("budget-expense-edit-form")) {
        const expenseId = form.getAttribute("data-budget-expense-id") || "";
        const editTr = expenseId
          ? document.querySelector(`tr.budget-expense-edit-row[data-budget-edit-for="${CSS.escape(expenseId)}"]`)
          : null;
        const viewTr = expenseId
          ? document.querySelector(`tr.budget-tx-view[data-budget-tx-view="${CSS.escape(expenseId)}"]`)
          : null;
        const mobileLi = expenseId
          ? document.querySelector(`li.budget-mobile-tx-item[data-budget-tx-view="${CSS.escape(expenseId)}"]`)
          : null;
        if (editTr) editTr.classList.remove("hidden");
        if (viewTr) viewTr.classList.add("editing");
        if (mobileLi) mobileLi.classList.add("editing");
        form.classList.remove("hidden");
        if (mqBudgetEditMobile.matches) {
          const dialog = document.getElementById("budget-mobile-expense-edit");
          const slot = dialog?.querySelector("[data-budget-mobile-edit-slot]");
          if (dialog && slot) {
            slot.appendChild(form);
            dialog.showModal();
            window.requestAnimationFrame(() => {
              form.querySelector("input:not([type='hidden']), select, textarea")?.focus();
            });
          }
        }
        return;
      }
      if (form.classList.contains("tab-expense-edit-form")) {
        const expenseId = form.getAttribute("data-tab-expense-id") || "";
        const editTr = expenseId
          ? document.querySelector(`tr.tab-expense-edit-row[data-tab-edit-for="${CSS.escape(expenseId)}"]`)
          : null;
        const viewTr = expenseId ? document.querySelector(`tr[data-tab-tx-view="${CSS.escape(expenseId)}"]`) : null;
        const viewCard = expenseId
          ? document.querySelector(`.tab-exp-grid-card[data-tab-tx-view="${CSS.escape(expenseId)}"]`)
          : null;
        if (editTr) editTr.classList.remove("hidden");
        if (viewTr) viewTr.classList.add("editing");
        if (viewCard) viewCard.classList.add("editing");
        form.classList.remove("hidden");
        if (mqTabEditMobile.matches) {
          const dialog = document.getElementById("tab-mobile-expense-edit");
          const slot = dialog?.querySelector("[data-tab-mobile-edit-slot]");
          if (dialog && slot) {
            slot.appendChild(form);
            dialog.showModal();
            window.requestAnimationFrame(() => {
              form.querySelector("input:not([type='hidden']), select, textarea")?.focus();
            });
          }
        }
        return;
      }
      if (form.classList.contains("tab-settlement-edit-form")) {
        form.classList.remove("hidden");
        return;
      }
      const row = form.closest(".timeline-item, .expense-item, .accommodation-item, .accommodation-card-wrap, .vehicle-rental-item, .flight-card, .reminder-checklist-item");
      if (row) row.classList.add("editing");
      const viewId = itineraryViewIdForForm(formId);
      const view = viewId ? document.getElementById(viewId) : null;
      if (view) view.classList.add("hidden");
      form.classList.remove("hidden");
      window.remiResyncVehicleDropoffInForm?.(form);
      const dateInput = form.querySelector("input[name='itinerary_date']");
      if (dateInput) dateInput.dataset.originalValue = dateInput.value;
    });
  };
  document.querySelectorAll("[data-inline-edit-open]").forEach((btn) => wireOneInlineEditOpenBtn(btn));
  window.remiWireInlineEditOpenButtonsIn = (root) => {
    const scope = root instanceof ParentNode ? root : document;
    scope.querySelectorAll("[data-inline-edit-open]").forEach((btn) => {
      if (btn instanceof HTMLElement) wireOneInlineEditOpenBtn(btn);
    });
  };
  window.rewireTabInlineEditOpenButtons = () => {
    document.querySelectorAll("main.tab-page [data-inline-edit-open]").forEach((oldBtn) => {
      const parent = oldBtn.parentNode;
      if (!(parent instanceof Node)) return;
      const fresh = oldBtn.cloneNode(true);
      delete fresh.dataset.remiInlineOpenWired;
      parent.replaceChild(fresh, oldBtn);
      wireOneInlineEditOpenBtn(fresh);
    });
  };

  document.addEventListener("click", (event) => {
    const btn = event.target?.closest?.("[data-inline-edit-cancel]");
    if (!(btn instanceof HTMLElement)) return;
    const formId = btn.getAttribute("data-inline-edit-cancel");
    if (!formId) return;
    closeInlineEdit(formId);
  });

  const budgetMobileEditDialog = document.getElementById("budget-mobile-expense-edit");
  budgetMobileEditDialog?.querySelector("[data-budget-mobile-edit-close]")?.addEventListener("click", () => {
    const form = budgetMobileEditDialog.querySelector(".budget-expense-edit-form");
    if (form?.id) closeInlineEdit(form.id);
  });
  budgetMobileEditDialog?.addEventListener("close", () => {
    const slot = budgetMobileEditDialog.querySelector("[data-budget-mobile-edit-slot]");
    const form = slot?.querySelector(".budget-expense-edit-form");
    if (form?.id) closeInlineEdit(form.id);
  });
  const tabMobileEditDialog = document.getElementById("tab-mobile-expense-edit");
  tabMobileEditDialog?.querySelector("[data-tab-mobile-edit-close]")?.addEventListener("click", () => {
    const form = tabMobileEditDialog.querySelector(".tab-expense-edit-form");
    if (form?.id) closeInlineEdit(form.id);
  });
  tabMobileEditDialog?.addEventListener("close", () => {
    const slot = tabMobileEditDialog.querySelector("[data-tab-mobile-edit-slot]");
    const form = slot?.querySelector(".tab-expense-edit-form");
    if (form?.id) closeInlineEdit(form.id);
  });

  const initItineraryForm = (itineraryForm) => {
    const locationInput = itineraryForm.querySelector("[data-location-input]");
    const latitudeInput = itineraryForm.querySelector("[data-latitude-input]");
    const longitudeInput = itineraryForm.querySelector("[data-longitude-input]");
    const locationStatus = itineraryForm.querySelector("[data-location-status]");
    const suggestionBox = itineraryForm.querySelector("[data-location-suggestions]");
    const itineraryDateInput = itineraryForm.querySelector("input[name='itinerary_date']");
    const dateStatus = itineraryForm.querySelector("[data-date-status]");
    const tripStart = itineraryForm.getAttribute("data-trip-start") || "";
    const tripEnd = itineraryForm.getAttribute("data-trip-end") || "";
    const tripStartLabel = itineraryForm.getAttribute("data-trip-start-label") || tripStart;
    const tripEndLabel = itineraryForm.getAttribute("data-trip-end-label") || tripEnd;
    const locationLookupEnabled = itineraryForm.getAttribute("data-location-lookup-enabled") !== "false";
    let cachedLocation = "";
    let cachedCoords = null;
    let suggestionTimer = null;
    let latestQueryToken = 0;

    const setLocationStatus = (message, state) => {
      if (!locationStatus) return;
      locationStatus.textContent = message;
      locationStatus.classList.remove("error", "success");
      if (state) {
        locationStatus.classList.add(state);
      }
    };

    const fillCoordinates = (coords) => {
      if (!latitudeInput || !longitudeInput) return;
      if (!coords) {
        latitudeInput.value = "";
        longitudeInput.value = "";
        return;
      }
      latitudeInput.value = String(coords.lat);
      longitudeInput.value = String(coords.lng);
    };

    const setDateStatus = (message, state) => {
      if (!dateStatus) return;
      dateStatus.textContent = message;
      dateStatus.classList.remove("error", "success");
      if (state) {
        dateStatus.classList.add(state);
      }
    };

    const validateDateInTripRange = () => {
      if (!itineraryDateInput || !tripStart || !tripEnd) return true;
      const selected = itineraryDateInput.value;
      if (!selected) {
        setDateStatus("Please select a date before saving.", "error");
        return false;
      }
      if (selected < tripStart || selected > tripEnd) {
        setDateStatus(`Date must be between ${tripStartLabel} and ${tripEndLabel}.`, "error");
        return false;
      }
      setDateStatus("Date is within your trip range.", "success");
      return true;
    };

    const hideSuggestions = () => {
      if (!suggestionBox) return;
      suggestionBox.classList.add("hidden");
      suggestionBox.innerHTML = "";
    };

    const selectSuggestion = (suggestion) => {
      if (!locationInput) return;
      const cleanName = suggestion.displayName || locationInput.value.trim();
      locationInput.value = cleanName;
      cachedLocation = normalize(cleanName);
      cachedCoords = { lat: suggestion.lat, lng: suggestion.lng, displayName: cleanName };
      fillCoordinates(cachedCoords);
      setLocationStatus("Location selected and ready to plot on the map.", "success");
      hideSuggestions();
    };

    const renderSuggestions = (suggestions) => {
      if (!suggestionBox) return;
      if (!suggestions.length) {
        hideSuggestions();
        return;
      }
      suggestionBox.innerHTML = "";
      suggestions.forEach((suggestion) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "location-suggestion-btn";
        btn.textContent = suggestion.displayName;
        remiPreventLocationSuggestBlur(btn);
        btn.addEventListener("click", () => {
          selectSuggestion(suggestion);
        });
        suggestionBox.appendChild(btn);
      });
      suggestionBox.classList.remove("hidden");
    };

    const queueSuggestions = (query) => {
      if (!locationLookupEnabled) {
        hideSuggestions();
        setLocationStatus("Location lookup is disabled in Settings.");
        return;
      }
      if (!locationInput) return;
      if (suggestionTimer) {
        clearTimeout(suggestionTimer);
      }
      const trimmed = query.trim();
      if (trimmed.length < 3) {
        hideSuggestions();
        if (!trimmed.length) {
          setLocationStatus("Type a location and coordinates will be fetched automatically.");
          fillCoordinates(null);
          cachedLocation = "";
          cachedCoords = null;
        }
        return;
      }
      suggestionTimer = window.setTimeout(async () => {
        const currentToken = ++latestQueryToken;
        setLocationStatus("Searching places...");
        const suggestions = await searchLocations(trimmed);
        if (currentToken !== latestQueryToken) return;
        renderSuggestions(suggestions);
        if (!suggestions.length) {
          setLocationStatus("No matching places yet. Try another keyword.", "error");
        } else {
          setLocationStatus("Select a place to confirm coordinates.");
        }
      }, 320);
    };

    const resolveLocation = async (query) => {
      if (!locationLookupEnabled) {
        fillCoordinates(null);
        hideSuggestions();
        return true;
      }
      const normalizedQuery = normalize(query);
      if (!normalizedQuery) {
        cachedLocation = "";
        cachedCoords = null;
        fillCoordinates(null);
        hideSuggestions();
        setLocationStatus("Type a location and coordinates will be fetched automatically.");
        return true;
      }

      if (normalizedQuery === cachedLocation && cachedCoords) {
        fillCoordinates(cachedCoords);
        return true;
      }

      setLocationStatus("Finding coordinates for this location...");
      const coords = await geocodeLocation(query.trim());
      if (!coords || Number.isNaN(coords.lat) || Number.isNaN(coords.lng)) {
        cachedLocation = "";
        cachedCoords = null;
        fillCoordinates(null);
        setLocationStatus("Location lookup failed. Try a more specific place name.", "error");
        return false;
      }

      cachedLocation = normalizedQuery;
      cachedCoords = coords;
      fillCoordinates(coords);
      setLocationStatus("Location found and ready to plot on the map.", "success");
      hideSuggestions();
      return true;
    };

    if (locationInput) {
      locationInput.addEventListener("input", () => {
        cachedLocation = "";
        cachedCoords = null;
        fillCoordinates(null);
        queueSuggestions(locationInput.value);
      });
      locationInput.addEventListener("blur", () => {
        window.setTimeout(() => {
          hideSuggestions();
        }, REMI_LOCATION_SUGGEST_BLUR_MS);
        void resolveLocation(locationInput.value);
      });
    }
    if (itineraryDateInput) {
      itineraryDateInput.addEventListener("change", () => {
        validateDateInTripRange();
      });
      validateDateInTripRange();
    }

    itineraryForm.addEventListener("submit", async (event) => {
      if (!validateDateInTripRange()) {
        event.preventDefault();
        itineraryDateInput?.focus();
        return;
      }
      if (!locationInput) return;
      const query = locationInput.value;
      if (!query || !query.trim()) return;

      const ok = await resolveLocation(query);
      if (ok) return;

      event.preventDefault();
      locationInput.focus();
    });
  };

  document.querySelectorAll("[data-itinerary-form]").forEach((itineraryForm) => {
    if (!(itineraryForm instanceof HTMLElement)) return;
    initItineraryForm(itineraryForm);
  });

  document.querySelectorAll("input[data-location-autocomplete]").forEach((input) => {
    let timer = null;
    let token = 0;
    let suggestionsHost = null;
    const ensureHost = () => {
      if (suggestionsHost) return suggestionsHost;
      suggestionsHost = document.createElement("div");
      suggestionsHost.className = "location-suggestions hidden";
      const wrapper = input.closest("label") || input.parentElement;
      if (wrapper) {
        wrapper.classList.add("location-autocomplete-anchor");
        wrapper.appendChild(suggestionsHost);
      }
      return suggestionsHost;
    };
    const hide = () => {
      const host = ensureHost();
      host.classList.add("hidden");
      host.innerHTML = "";
    };
    input.addEventListener("input", () => {
      if (timer) clearTimeout(timer);
      const query = input.value.trim();
      if (query.length < 3) {
        hide();
        return;
      }
      timer = window.setTimeout(async () => {
        const current = ++token;
        const host = ensureHost();
        const suggestions = await searchLocations(query);
        if (current !== token) return;
        host.innerHTML = "";
        if (!suggestions.length) {
          hide();
          return;
        }
        suggestions.forEach((suggestion) => {
          const btn = document.createElement("button");
          btn.type = "button";
          btn.className = "location-suggestion-btn";
          btn.textContent = suggestion.displayName;
          remiPreventLocationSuggestBlur(btn);
          btn.addEventListener("click", () => {
            input.value = suggestion.displayName;
            hide();
          });
          host.appendChild(btn);
        });
        host.classList.remove("hidden");
      }, 300);
    });
    input.addEventListener("blur", () => {
      window.setTimeout(hide, REMI_LOCATION_SUGGEST_BLUR_MS);
    });
  });

  const vehicleDropoffSyncByFieldset = new WeakMap();
  document.querySelectorAll(".vehicle-dropoff-fieldset").forEach((fs) => {
    const same = fs.querySelector("[data-vehicle-dropoff-same]");
    const diff = fs.querySelector("[data-vehicle-dropoff-diff]");
    const locWrap = fs.querySelector("[data-vehicle-dropoff-field]");
    const locInp = locWrap?.querySelector("input[name='drop_off_location']");
    const sync = () => {
      const different = Boolean(diff?.checked);
      if (locWrap) {
        locWrap.toggleAttribute("hidden", !different);
        locWrap.setAttribute("aria-hidden", different ? "false" : "true");
      }
      if (locInp) {
        locInp.disabled = !different;
        locInp.required = different;
      }
    };
    same?.addEventListener("change", sync);
    diff?.addEventListener("change", sync);
    vehicleDropoffSyncByFieldset.set(fs, sync);
    sync();
  });

  window.remiResyncVehicleDropoffInForm = (form) => {
    if (!(form instanceof HTMLElement)) return;
    form.querySelectorAll(".vehicle-dropoff-fieldset").forEach((fs) => {
      const fn = vehicleDropoffSyncByFieldset.get(fs);
      if (typeof fn === "function") fn();
    });
  };

  const tripCoverModeEl = document.getElementById("trip_cover_image_mode");
  const tripCoverFieldEl = document.querySelector("[data-trip-cover-field]");
  const tripCoverExistingEl = document.querySelector("input[name='cover_image_existing']");
  if (tripCoverModeEl && tripCoverFieldEl) {
    const previewWrap = tripCoverFieldEl.querySelector("[data-trip-cover-preview-wrap]");
    const urlLabel = tripCoverFieldEl.querySelector("[data-trip-cover-url-label]");
    const fileLabel = tripCoverFieldEl.querySelector("[data-trip-cover-file-label]");
    const urlInput = tripCoverFieldEl.querySelector("[data-trip-cover-url-input]");
    const fileInp = tripCoverFieldEl.querySelector("[data-trip-cover-file-input]");
    let tripCoverBlobUrl = null;

    const revokeTripCoverBlob = () => {
      if (tripCoverBlobUrl) {
        URL.revokeObjectURL(tripCoverBlobUrl);
        tripCoverBlobUrl = null;
      }
    };

    const setTripCoverPreviewEmpty = (message) => {
      if (!previewWrap) return;
      previewWrap.replaceChildren();
      const div = document.createElement("div");
      div.className = "trip-cover-image-field__preview trip-cover-image-field__preview--empty muted";
      div.textContent = message;
      previewWrap.appendChild(div);
    };

    const setTripCoverPreviewSrc = (src) => {
      if (!previewWrap) return;
      previewWrap.replaceChildren();
      const img = document.createElement("img");
      img.className = "trip-cover-image-field__preview";
      img.alt = "Trip cover preview";
      img.referrerPolicy = "no-referrer";
      img.onerror = () => {
        setTripCoverPreviewEmpty("Could not load preview");
      };
      previewWrap.appendChild(img);
      img.src = src;
    };

    const applyTripCoverUrlPreview = () => {
      let u = (urlInput?.value || "").trim();
      if (!u && tripCoverExistingEl) {
        const ex = tripCoverExistingEl.value.trim();
        if (/^https?:\/\//i.test(ex)) u = ex;
      }
      if (!u) {
        setTripCoverPreviewEmpty("Paste an image URL (https) to preview");
        return;
      }
      if (!/^https?:\/\//i.test(u)) {
        setTripCoverPreviewEmpty("Enter a valid http(s) image URL");
        return;
      }
      revokeTripCoverBlob();
      setTripCoverPreviewSrc(u);
    };

    const restoreTripCoverUploadPreview = () => {
      revokeTripCoverBlob();
      const f = fileInp?.files && fileInp.files[0];
      if (f) {
        try {
          tripCoverBlobUrl = URL.createObjectURL(f);
          setTripCoverPreviewSrc(tripCoverBlobUrl);
        } catch {
          setTripCoverPreviewEmpty("Could not read file");
        }
        return;
      }
      const ex = tripCoverExistingEl?.value?.trim() || "";
      if (ex.startsWith("/static/uploads/covers/")) {
        setTripCoverPreviewSrc(ex);
        return;
      }
      setTripCoverPreviewEmpty("Choose an image file below (JPG, PNG, WebP, or GIF)");
    };

    const syncTripCoverUI = () => {
      const v = tripCoverModeEl.value;
      if (v === "clear") {
        revokeTripCoverBlob();
        if (previewWrap) previewWrap.hidden = true;
        if (urlLabel) urlLabel.hidden = true;
        if (fileLabel) fileLabel.hidden = true;
        return;
      }
      if (previewWrap) previewWrap.hidden = false;
      if (urlLabel) urlLabel.hidden = v !== "url";
      if (fileLabel) fileLabel.hidden = v !== "upload";
      if (v === "url") {
        applyTripCoverUrlPreview();
      } else {
        restoreTripCoverUploadPreview();
      }
    };

    tripCoverModeEl.addEventListener("change", syncTripCoverUI);

    let tripCoverUrlTimer = null;
    urlInput?.addEventListener("input", () => {
      if (tripCoverModeEl.value !== "url") return;
      window.clearTimeout(tripCoverUrlTimer);
      tripCoverUrlTimer = window.setTimeout(() => applyTripCoverUrlPreview(), 350);
    });
    urlInput?.addEventListener("change", () => {
      if (tripCoverModeEl.value === "url") applyTripCoverUrlPreview();
    });

    fileInp?.addEventListener("change", () => {
      if (tripCoverModeEl.value !== "upload") return;
      restoreTripCoverUploadPreview();
    });

    syncTripCoverUI();
  }

  const heroModeEl = document.getElementById("dashboard_hero_background_mode");
  const heroFieldEl = document.querySelector("[data-hero-image-field]");
  if (heroModeEl && heroFieldEl) {
    const form = heroModeEl.closest("form");
    const urlInp = form?.querySelector("input[name='dashboard_hero_background_url']");
    const urlLabel = urlInp?.closest("label");
    const uploadHint = document.getElementById("hint-hero-upload");
    const fileInp = heroFieldEl.querySelector("input[type='file']");
    const syncHeroUI = () => {
      const v = heroModeEl.value;
      const showUpload = v === "custom_upload";
      const showUrl = v === "custom_url";
      heroFieldEl.hidden = !showUpload;
      if (urlLabel) urlLabel.hidden = !showUrl;
      if (uploadHint) uploadHint.hidden = !showUpload;
    };
    heroModeEl.addEventListener("change", syncHeroUI);
    syncHeroUI();
    fileInp?.addEventListener("change", () => {
      const f = fileInp.files && fileInp.files[0];
      const wrap = heroFieldEl.querySelector(".hero-image-field__preview-wrap");
      if (!f || !wrap) return;
      try {
        const objUrl = URL.createObjectURL(f);
        wrap.replaceChildren();
        const img = document.createElement("img");
        img.className = "hero-image-field__preview";
        img.alt = "Hero preview";
        img.src = objUrl;
        wrap.appendChild(img);
      } catch (e) {
        /* ignore */
      }
    });
  }

  document.querySelectorAll("[data-accommodation-form]").forEach((accommodationForm) => {
    const accommodationNameInput = accommodationForm.querySelector("[data-accommodation-name]");
    const accommodationAddressInput = accommodationForm.querySelector("[data-accommodation-address]");
    const accommodationStatus = accommodationForm.querySelector("[data-accommodation-status]");
    const mqAccommodationMobile = typeof window.matchMedia === "function" ? window.matchMedia("(max-width: 920px)") : null;
    let accommodationLookupTimer = null;

    const setAccommodationStatus = (message, state) => {
      if (!accommodationStatus) return;
      accommodationStatus.textContent = message;
      accommodationStatus.classList.remove("error", "success");
      if (state) accommodationStatus.classList.add(state);
    };

    const lookupAddress = async (query) => {
      if (!query || !query.trim()) return;
      setAccommodationStatus("Looking up address...");
      const suggestion = await geocodeLocation(query.trim());
      if (suggestion && suggestion.displayName && accommodationAddressInput) {
        if (!accommodationAddressInput.value.trim()) {
          accommodationAddressInput.value = suggestion.displayName;
        }
        setAccommodationStatus("Address auto-filled. Please review before saving.", "success");
        return;
      }
      setAccommodationStatus("Could not auto-fill address. Please enter it manually.", "error");
    };

    if (accommodationNameInput) {
      accommodationNameInput.addEventListener("input", () => {
        if (accommodationLookupTimer) clearTimeout(accommodationLookupTimer);
        accommodationLookupTimer = window.setTimeout(() => {
          void lookupAddress(accommodationNameInput.value);
        }, 420);
      });
      accommodationNameInput.addEventListener("blur", () => {
        void lookupAddress(accommodationNameInput.value);
      });
    }

    accommodationForm.addEventListener("keydown", (event) => {
      if (event.key !== "Enter" || event.shiftKey || event.altKey || event.ctrlKey || event.metaKey) return;
      if (mqAccommodationMobile && !mqAccommodationMobile.matches) return;
      const active = event.target;
      if (!(active instanceof HTMLElement)) return;
      if (active instanceof HTMLTextAreaElement || active instanceof HTMLButtonElement) return;
      if (active instanceof HTMLInputElement) {
        const inputType = (active.type || "").toLowerCase();
        if (["hidden", "submit", "button", "reset", "checkbox", "radio", "file"].includes(inputType)) return;
      }
      if (active instanceof HTMLSelectElement) return;
      const fields = Array.from(accommodationForm.querySelectorAll("input, select, textarea")).filter((el) => {
        if (!(el instanceof HTMLElement)) return false;
        if (el.hasAttribute("disabled")) return false;
        if (el.getAttribute("aria-hidden") === "true") return false;
        if (el.offsetParent === null) return false;
        if (el instanceof HTMLInputElement) {
          const inputType = (el.type || "").toLowerCase();
          if (["hidden", "submit", "button", "reset", "checkbox", "radio", "file"].includes(inputType)) return false;
        }
        return true;
      });
      const idx = fields.indexOf(active);
      if (idx < 0) return;
      const next = fields[idx + 1];
      if (!(next instanceof HTMLElement)) return;
      event.preventDefault();
      next.focus();
      if (next instanceof HTMLInputElement && (next.type === "text" || next.type === "search" || next.type === "tel" || next.type === "url" || next.type === "email")) {
        try {
          const len = next.value.length;
          next.setSelectionRange(len, len);
        } catch (e) {
          /* ignore */
        }
      }
    });
  });

  document.querySelectorAll("[data-vehicle-form]").forEach((vehicleForm) => {
    const mqVehicleMobile = typeof window.matchMedia === "function" ? window.matchMedia("(max-width: 920px)") : null;
    vehicleForm.addEventListener("keydown", (event) => {
      if (event.key !== "Enter" || event.shiftKey || event.altKey || event.ctrlKey || event.metaKey) return;
      if (mqVehicleMobile && !mqVehicleMobile.matches) return;
      const active = event.target;
      if (!(active instanceof HTMLElement)) return;
      if (active instanceof HTMLTextAreaElement || active instanceof HTMLButtonElement) return;
      if (active instanceof HTMLInputElement) {
        const inputType = (active.type || "").toLowerCase();
        if (["hidden", "submit", "button", "reset", "checkbox", "radio", "file"].includes(inputType)) return;
      }
      if (active instanceof HTMLSelectElement) return;
      const fields = Array.from(vehicleForm.querySelectorAll("input, select, textarea")).filter((el) => {
        if (!(el instanceof HTMLElement)) return false;
        if (el.hasAttribute("disabled")) return false;
        if (el.getAttribute("aria-hidden") === "true") return false;
        if (el.offsetParent === null) return false;
        if (el instanceof HTMLInputElement) {
          const inputType = (el.type || "").toLowerCase();
          if (["hidden", "submit", "button", "reset", "checkbox", "radio", "file"].includes(inputType)) return false;
        }
        return true;
      });
      const idx = fields.indexOf(active);
      if (idx < 0) return;
      const next = fields[idx + 1];
      if (!(next instanceof HTMLElement)) return;
      event.preventDefault();
      next.focus();
      if (next instanceof HTMLInputElement && (next.type === "text" || next.type === "search" || next.type === "tel" || next.type === "url" || next.type === "email")) {
        try {
          const len = next.value.length;
          next.setSelectionRange(len, len);
        } catch (e) {
          /* ignore */
        }
      }
    });
  });

  document.querySelectorAll("[data-checklist-builder]").forEach((checklistForm) => {
    const itemInput = checklistForm.querySelector("[data-checklist-item-input]");
    const addBtn = checklistForm.querySelector("[data-checklist-add-btn]");
    const draftList = checklistForm.querySelector("[data-checklist-draft-list]");
    const draftEmpty = checklistForm.querySelector("[data-checklist-draft-empty]");
    const itemsJSONInput = checklistForm.querySelector("[data-checklist-items-json]");
    const statusEl = checklistForm.querySelector("[data-checklist-builder-status]");
    const draftItems = [];

    const setStatus = (message, state) => {
      if (!statusEl) return;
      statusEl.textContent = message;
      statusEl.classList.remove("error", "success");
      if (state) {
        statusEl.classList.add(state);
      }
    };

    const renderDraftItems = () => {
      if (!draftList || !itemsJSONInput) return;
      draftList.innerHTML = "";
      draftItems.forEach((itemText, index) => {
        const li = document.createElement("li");
        li.className = "list-item checklist-draft-item";

        const text = document.createElement("span");
        text.textContent = itemText;
        li.appendChild(text);

        const removeBtn = document.createElement("button");
        removeBtn.type = "button";
        removeBtn.className = "secondary-btn checklist-remove-btn";
        removeBtn.textContent = "Remove";
        removeBtn.addEventListener("click", () => {
          draftItems.splice(index, 1);
          renderDraftItems();
          setStatus("", null);
        });
        li.appendChild(removeBtn);
        draftList.appendChild(li);
      });
      const hasItems = draftItems.length > 0;
      draftList.classList.toggle("hidden", !hasItems);
      if (draftEmpty) {
        draftEmpty.classList.toggle("hidden", hasItems);
      }
      itemsJSONInput.value = JSON.stringify(draftItems);
    };

    const addChecklistItem = () => {
      if (!itemInput) return;
      const value = itemInput.value.trim();
      if (!value) {
        setStatus("Enter an item before adding.", "error");
        itemInput.focus();
        return;
      }
      draftItems.push(value);
      itemInput.value = "";
      renderDraftItems();
      setStatus("Item added. Add more, then click Save.", "success");
      itemInput.focus();
    };

    if (addBtn) {
      addBtn.addEventListener("click", addChecklistItem);
    }
    if (itemInput) {
      itemInput.addEventListener("keydown", (event) => {
        if (event.key !== "Enter") return;
        event.preventDefault();
        addChecklistItem();
      });
    }
    checklistForm.addEventListener("submit", (event) => {
      if (draftItems.length > 0) return;
      event.preventDefault();
      setStatus("Add at least one item before saving.", "error");
      itemInput?.focus();
    });
    checklistForm.addEventListener("submit", () => {
      if (draftItems.length === 0) return;
      try {
        sessionStorage.setItem(TOAST_KEY, "Checklist saved.");
      } catch (e) {
        /* ignore */
      }
    });
  });

  document.querySelectorAll("form[action^='/checklist/'][action$='/update']").forEach((form) => {
    form.addEventListener("submit", () => {
      try {
        sessionStorage.setItem(TOAST_KEY, "Checklist item updated.");
      } catch (e) {
        /* ignore */
      }
    });
  });

  const inferToastMessage = (form) => {
    const explicit = form.getAttribute("data-toast-message");
    if (explicit) return explicit;
    const raw = (form.getAttribute("action") || "").trim();
    const lower = raw.toLowerCase();
    let path = lower;
    if (raw) {
      try {
        path = new URL(raw, window.location.origin).pathname.toLowerCase();
      } catch {
        path = lower;
      }
    }
    if (path.includes("/profile/resend-verify")) return "Verification link sent.";
    if (path.includes("/profile/password")) return "Password updated.";
    if (path === "/profile" || /\/profile$/.test(path)) return "Profile updated.";
    if (path === "/settings" || /\/settings$/.test(path)) return "Settings saved.";
    if (path === "/trips") return "Trip created.";
    if (/\/trips\/[^/]+\/update$/i.test(path)) return "Trip details saved.";
    if (/\/trips\/[^/]+\/delete$/i.test(path)) return "Trip deleted.";
    if (/\/trips\/[^/]+\/archive$/i.test(path)) return "Trip archived.";
    if (/\/trips\/[^/]+\/leave$/i.test(path)) return "You left this trip.";
    if (/\/trips\/[^/]+\/stop-sharing$/i.test(path)) return "Collaborators removed.";
    if (/\/trips\/[^/]+\/invite$/i.test(path)) return "Invitation sent.";
    if (path.includes("/hide-archived")) return "Trip list updated.";
    if (path.includes("/invites/accept")) return "Invite accepted.";
    if (lower.includes("/delete")) return "Deleted successfully.";
    if (lower.includes("/toggle")) return "Updated successfully.";
    if (lower.includes("/update")) return "Saved changes.";
    if (lower.includes("/trips") && !lower.includes("/update")) return "Added successfully.";
    return "Saved successfully.";
  };

  const updateChecklistToggleUI = (form, done) => {
    const row = form.closest(".checklist-view");
    if (!row) return;
    const icon = row.querySelector(".check-btn .material-symbols-outlined");
    const text = row.querySelector("span:not(.material-symbols-outlined)");
    if (icon) icon.textContent = done ? "check" : "circle";
    if (text) text.classList.toggle("done", done);
    const hiddenDone = form.querySelector("input[name='done']");
    if (hiddenDone) hiddenDone.value = done ? "false" : "true";
  };

  const updateChecklistEditUI = (form) => {
    const row = form.closest(".reminder-checklist-item");
    if (!row) return;
    const textInput = form.querySelector("input[name='text']");
    const nextText = (textInput?.value || "").trim();
    if (!nextText) return;
    const textNode = row.querySelector(".checklist-view > span:not(.material-symbols-outlined)");
    if (textNode) {
      textNode.textContent = nextText;
    }
  };

  const refreshInlineViewFromServer = async (form) => {
    const formId = form.id || "";
    if (!formId) return;
    if (formId.startsWith("day-label-form-")) return;
    const viewId = itineraryViewIdForForm(formId);
    if (!viewId) return;
    const currentView = document.getElementById(viewId);
    if (!currentView) return;
    try {
      const response = await fetch(window.location.href, {
        headers: {
          "X-Requested-With": "XMLHttpRequest",
          Accept: "text/html"
        }
      });
      if (!response.ok) return;
      const html = await response.text();
      const parser = new DOMParser();
      const doc = parser.parseFromString(html, "text/html");
      const freshView = doc.getElementById(viewId);
      if (!freshView) return;
      currentView.innerHTML = freshView.innerHTML;
      currentView.className = freshView.className;
    } catch (e) {
      // keep existing UI if refresh fails
    }
  };

  const isItineraryItemForm = (formId) =>
    formId.startsWith("itinerary-edit-") ||
    formId.startsWith("vehicle-rental-itinerary-edit-") ||
    formId.startsWith("accommodation-itinerary-edit-") ||
    formId.startsWith("flight-itinerary-edit-");

  // Shared helper: fetch fresh page HTML once, reuse across calls
  const fetchFreshDoc = async () => {
    const r = await fetch(window.location.href, {
      headers: { "X-Requested-With": "XMLHttpRequest", Accept: "text/html" }
    });
    if (!r.ok) return null;
    return new DOMParser().parseFromString(await r.text(), "text/html");
  };

  window.refreshTheTabPageFromServer = async () => {
    if (!document.querySelector("main.tab-page")) return;
    const doc = await fetchFreshDoc();
    if (!doc) return;
    document.querySelectorAll("[data-tab-refresh-region]").forEach((el) => {
      const k = el.getAttribute("data-tab-refresh-region");
      if (!k) return;
      const fresh = doc.querySelector(`[data-tab-refresh-region="${CSS.escape(k)}"]`);
      if (fresh) el.innerHTML = fresh.innerHTML;
    });
    document.querySelectorAll("[data-tab-refresh-region] form[data-app-confirm]").forEach((f) => {
      delete f.dataset.remiConfirmWired;
    });
    document.querySelectorAll("[data-tab-refresh-region] form[data-ajax-submit]").forEach((f) => {
      delete f.dataset.remiAjaxWired;
    });
    document.querySelectorAll("[data-tab-refresh-region] [data-tab-split-root]").forEach((r) => {
      delete r.dataset.tabSplitBound;
    });
    document.querySelectorAll("[data-tab-refresh-region] form[data-app-confirm]").forEach((f) => {
      window.remiWireAppConfirmOnForm?.(f);
    });
    document.querySelectorAll("[data-tab-refresh-region] form[data-ajax-submit]").forEach((f) => {
      window.remiBindAjaxSubmitForm?.(f);
    });
    window.setupTabSplitRootsIn?.(document);
    window.reinitTabOverTimeChart?.();
    window.rewireTabInlineEditOpenButtons?.();
    window.remiSyncTabExpenseInstantFilter?.();
  };

  // Reposition a DOM row to the correct slot based on the fresh server HTML.
  // rowSelector: CSS selector for the item's row (e.g. "li.timeline-item")
  // viewId: id of the view element inside the row
  // dayGroupAttr: data attribute used to identify day groups (e.g. "data-date")
  // listSelector: CSS selector for the ordered list inside a day group (e.g. "ul.day-items")
  // itemSelector: CSS selector for sibling rows (e.g. "li.timeline-item")
  const repositionInDayGroup = (row, freshDoc, viewId, dayGroupAttr, listSelector, itemSelector) => {
    const freshView = freshDoc.getElementById(viewId);
    if (!freshView) return false;
    const freshLi = freshView.closest("li");
    const freshDayGroup = freshLi ? freshLi.closest(".day-group") : null;
    const newDate = freshDayGroup ? freshDayGroup.getAttribute(dayGroupAttr) : null;

    const targetDayGroup = newDate
      ? document.querySelector(`.day-group[${dayGroupAttr}="${newDate}"]`)
      : row.closest(".day-group");
    if (!targetDayGroup) return false;

    const targetUl = targetDayGroup.querySelector(listSelector);
    if (!targetUl) return false;

    // Correct position from fresh HTML
    const freshTargetUl = freshDayGroup ? freshDayGroup.querySelector(listSelector) : null;
    const freshItemLis = freshTargetUl
      ? Array.from(freshTargetUl.querySelectorAll(`:scope > ${itemSelector}`))
      : [];
    const freshItemIdx = freshItemLis.findIndex(
      (li) => li.querySelector(`#${CSS.escape(viewId)}`) !== null
    );

    const otherItems = Array.from(
      targetUl.querySelectorAll(`:scope > ${itemSelector}`)
    ).filter((li) => li !== row);

    if (freshItemIdx <= 0) {
      const first = targetUl.querySelector(`:scope > ${itemSelector}`);
      if (first && first !== row) targetUl.insertBefore(row, first);
      else if (!first) targetUl.appendChild(row);
    } else {
      const refItem = otherItems[freshItemIdx - 1];
      if (refItem) refItem.insertAdjacentElement("afterend", row);
      else targetUl.appendChild(row);
    }

    targetDayGroup.open = true;
    return true;
  };

  // Sync one itinerary <li> from the fresh document: update content, data attrs, and position.
  const syncItineraryRow = (row, freshDoc, rowViewId) => {
    const freshView = freshDoc.getElementById(rowViewId);
    if (!freshView) return false;
    const freshLi = freshView.closest("li");

    // Update data attributes on the row
    if (freshLi) {
      [
        "data-lat",
        "data-lng",
        "data-title",
        "data-location",
        "data-geocode-location",
        "data-search-text",
        "data-map-day",
        "data-marker-kind"
      ].forEach((attr) => {
        const val = freshLi.getAttribute(attr);
        if (val !== null) row.setAttribute(attr, val);
        else row.removeAttribute(attr);
      });
    }

    // Update view content
    const currentView = document.getElementById(rowViewId);
    if (currentView) {
      currentView.innerHTML = freshView.innerHTML;
      currentView.className = freshView.className;
    }

    return repositionInDayGroup(row, freshDoc, rowViewId, "data-date", "ul.day-items", "li.timeline-item");
  };

  // Return any sibling itinerary forms that share the same entity (flight/vehicle/accommodation).
  const getSiblingItineraryForms = (form) => {
    const action = form.getAttribute("action") || "";
    const m = action.match(/\/(flights|vehicle-rental|accommodation)\/([^/]+)\/update/);
    if (!m) return [];
    const [, entityType, entityId] = m;
    return Array.from(
      document.querySelectorAll(`form[id][action*="/${entityType}/${entityId}/update"]`)
    ).filter((f) => f !== form);
  };

  const smartRepositionItineraryItem = async (form) => {
    const viewId = itineraryViewIdForForm(form.id);
    if (!viewId || !viewId.startsWith("itinerary-view-")) return false;
    const row = form.closest(".timeline-item");
    if (!row) return false;

    const freshDoc = await fetchFreshDoc();
    if (!freshDoc) return false;

    // Sync the directly-edited item
    const ok = syncItineraryRow(row, freshDoc, viewId);
    if (!ok) return false;

    // Sync sibling items linked to the same entity (e.g. depart + arrive for a flight)
    getSiblingItineraryForms(form).forEach((siblingForm) => {
      const siblingViewId = itineraryViewIdForForm(siblingForm.id);
      if (!siblingViewId) return;
      const siblingRow = siblingForm.closest(".timeline-item");
      if (!siblingRow || siblingRow.classList.contains("editing")) return;
      syncItineraryRow(siblingRow, freshDoc, siblingViewId);
    });

    renderItineraryConnectors(document);
    return true;
  };

  const smartRepositionExpenseItem = async (form) => {
    if (form.id && form.id.startsWith("expense-edit-budget-")) {
      return false;
    }
    const expenseId = form.id.replace("expense-edit-", "");
    const viewId = `expense-view-${expenseId}`;
    const row = form.closest(".expense-item");
    if (!row) return false;

    const freshDoc = await fetchFreshDoc();
    if (!freshDoc) return false;

    const freshView = freshDoc.getElementById(viewId);
    if (!freshView) return false;

    // Update view content
    const currentView = document.getElementById(viewId);
    if (currentView) {
      currentView.innerHTML = freshView.innerHTML;
      currentView.className = freshView.className;
    }

    return repositionInDayGroup(
      row, freshDoc, viewId,
      "data-expense-date", "ul.expense-items", "li.expense-item"
    );
  };

  const refreshBudgetTilesFromPage = async () => {
    const tiles = document.querySelectorAll(".trip-details-page .budget-tile");
    if (!tiles.length) return;
    try {
      const response = await fetch(window.location.href, {
        headers: { "X-Requested-With": "XMLHttpRequest", Accept: "text/html" },
        credentials: "same-origin"
      });
      if (!response.ok) return;
      const doc = new DOMParser().parseFromString(await response.text(), "text/html");
      tiles.forEach((tile) => {
        const sel = tile.classList.contains("trip-budget-tile--mobile")
          ? ".budget-tile.trip-budget-tile--mobile"
          : tile.classList.contains("trip-budget-tile--sidebar")
            ? ".budget-tile.trip-budget-tile--sidebar"
            : ".budget-tile";
        const fresh = doc.querySelector(sel);
        if (fresh) {
          tile.innerHTML = fresh.innerHTML;
        }
      });
    } catch (e) {
      /* ignore */
    }
  };

  /** Fields associated via the HTML form="" attribute are omitted from FormData(form) in some browsers. */
  const mergeFormAssociatedControls = (form, formData) => {
    const id = form.getAttribute("id");
    if (!id) return;
    document.querySelectorAll(`[form="${CSS.escape(id)}"]`).forEach((el) => {
      if (
        !(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement || el instanceof HTMLSelectElement)
      ) {
        return;
      }
      if (!el.name || el.disabled) return;
      const t = el.type;
      if (t === "submit" || t === "button" || t === "image" || t === "file" || t === "reset") return;
      if ((t === "checkbox" || t === "radio") && !el.checked) return;
      formData.set(el.name, el.value);
    });
  };

  /** Shared styled confirm: `form[data-app-confirm]` + data-confirm-title, optional body/ok/icon/variant. Runs in capture phase before `data-ajax-submit`. */
  let pendingConfirmSubmitForm = null;
  const confirmDialogEl = document.getElementById("dialog-confirm-action");
  if (confirmDialogEl instanceof HTMLDialogElement) {
    const confirmTitleEl = document.getElementById("dialog-confirm-action-title");
    const confirmBodyEl = document.getElementById("dialog-confirm-action-desc");
    const confirmIconEl = document.getElementById("dialog-confirm-action-icon");
    const confirmOkBtn = document.getElementById("dialog-confirm-action-ok");
    const confirmCancelBtn = confirmDialogEl.querySelector("[data-confirm-cancel]");

    const populateConfirmFromForm = (form) => {
      const title = form.getAttribute("data-confirm-title") || "Confirm?";
      const body = (form.getAttribute("data-confirm-body") || "").trim();
      const ok = form.getAttribute("data-confirm-ok") || "Confirm";
      const icon = form.getAttribute("data-confirm-icon") || "help_outline";
      const variant = (form.getAttribute("data-confirm-variant") || "danger").toLowerCase();
      if (confirmTitleEl) {
        confirmTitleEl.textContent = title;
      }
      if (confirmBodyEl) {
        if (body) {
          confirmBodyEl.textContent = body;
          confirmBodyEl.hidden = false;
          confirmDialogEl.setAttribute("aria-describedby", "dialog-confirm-action-desc");
        } else {
          confirmBodyEl.textContent = "";
          confirmBodyEl.hidden = true;
          confirmDialogEl.removeAttribute("aria-describedby");
        }
      }
      if (confirmIconEl) {
        confirmIconEl.textContent = icon;
      }
      if (confirmOkBtn) {
        confirmOkBtn.textContent = ok;
        confirmOkBtn.classList.toggle("app-confirm-dialog__btn--danger", variant === "danger");
        confirmOkBtn.classList.toggle("app-confirm-dialog__btn--neutral", variant === "neutral");
      }
    };

    const closeConfirmClearPending = () => {
      pendingConfirmSubmitForm = null;
      try {
        confirmDialogEl.close();
      } catch (e) {
        /* ignore */
      }
    };

    confirmOkBtn?.addEventListener("click", () => {
      const f = pendingConfirmSubmitForm;
      pendingConfirmSubmitForm = null;
      try {
        confirmDialogEl.close();
      } catch (e) {
        /* ignore */
      }
      if (f) {
        f.dataset.confirmPass = "1";
        if (typeof f.requestSubmit === "function") {
          f.requestSubmit();
        } else {
          f.submit();
        }
      }
    });
    confirmCancelBtn?.addEventListener("click", closeConfirmClearPending);
    confirmDialogEl.addEventListener("cancel", (e) => {
      e.preventDefault();
      closeConfirmClearPending();
    });
    confirmDialogEl.addEventListener("click", (e) => {
      if (e.target === confirmDialogEl) {
        closeConfirmClearPending();
      }
    });

    const wireAppConfirmOnForm = (form) => {
      if (!(form instanceof HTMLFormElement) || !form.hasAttribute("data-app-confirm")) return;
      if (form.dataset.remiConfirmWired === "1") return;
      form.dataset.remiConfirmWired = "1";
      form.addEventListener(
        "submit",
        (e) => {
          if (form.dataset.confirmPass === "1") {
            delete form.dataset.confirmPass;
            return;
          }
          e.preventDefault();
          e.stopImmediatePropagation();
          pendingConfirmSubmitForm = form;
          populateConfirmFromForm(form);
          confirmDialogEl.showModal();
        },
        true
      );
    };
    document.querySelectorAll("form[data-app-confirm]").forEach((form) => wireAppConfirmOnForm(form));
    window.remiWireAppConfirmOnForm = wireAppConfirmOnForm;
  }

  const handleAjaxFormSubmit = async (event) => {
    const form = event.currentTarget;
    if (!(form instanceof HTMLFormElement)) return;
      if (event.defaultPrevented) return;
      event.preventDefault();
      const method = (form.getAttribute("method") || "post").toUpperCase();
      const formData = new FormData(form);
      mergeFormAssociatedControls(form, formData);
      if (
        form.classList.contains("tab-settlement-form") ||
        form.classList.contains("tab-settlement-edit-form")
      ) {
        const payer = String(formData.get("payer_user_id") ?? "").trim();
        const payee = String(formData.get("payee_user_id") ?? "").trim();
        if (payer && payee && payer === payee) {
          showToast("Choose two different people for payer and payee.");
          return;
        }
      }
      const hasFileInput = Boolean(form.querySelector("input[type='file']"));
      const isMultipartForm = (form.enctype || "").toLowerCase().includes("multipart/form-data") || hasFileInput;
      const requestBody = isMultipartForm ? formData : new URLSearchParams(formData);
      const requestHeaders = {
        "X-Requested-With": "XMLHttpRequest",
        Accept: "application/json"
      };
      if (!isMultipartForm) {
        requestHeaders["Content-Type"] = "application/x-www-form-urlencoded;charset=UTF-8";
      }
      try {
        const response = await fetch(form.action, {
          method,
          body: requestBody,
          headers: requestHeaders
        });
        if (!response.ok) {
          const txt = (await response.text()).trim();
          throw new Error(txt || "Save failed.");
        }
        if (form.classList.contains("check-row")) {
          const done = String(formData.get("done")) === "true";
          updateChecklistToggleUI(form, done);
        }
        if (/\/checklist\/[^/]+\/update$/i.test(form.action || "")) {
          updateChecklistEditUI(form);
        }
        if (form.id && isItineraryItemForm(form.id)) {
          const moved = await smartRepositionItineraryItem(form);
          if (!moved) {
            try { sessionStorage.setItem(TOAST_KEY, inferToastMessage(form)); } catch (e) { /* ignore */ }
            window.location.reload();
            return;
          }
          closeInlineEdit(form.id);
        } else if (form.id && form.id.startsWith("expense-edit-budget-")) {
          try {
            sessionStorage.setItem(TOAST_KEY, inferToastMessage(form));
          } catch (e) {
            /* ignore */
          }
          window.location.reload();
          return;
        } else if (form.classList.contains("tab-expense-edit-form")) {
          await window.refreshTheTabPageFromServer();
          if (form.id) closeInlineEdit(form.id);
        } else if (form.id && form.id.startsWith("expense-edit-")) {
          const moved = await smartRepositionExpenseItem(form);
          if (!moved) {
            try { sessionStorage.setItem(TOAST_KEY, inferToastMessage(form)); } catch (e) { /* ignore */ }
            window.location.reload();
            return;
          }
          closeInlineEdit(form.id);
        } else if (
          document.querySelector("main.tab-page") &&
          (form.classList.contains("tab-log-expense-form") ||
            /\/(tab|group-expenses)\/settlements/.test(form.action || "") ||
            (/\/expenses\/[^/]+\/delete$/i.test(form.action || "") && form.closest("[data-tab-expenses-section]")))
        ) {
          await window.refreshTheTabPageFromServer();
          if (form.classList.contains("tab-log-expense-form")) {
            form.reset();
            const wrap = document.getElementById("tab-split-root-add");
            if (wrap?.parentNode) {
              const clone = wrap.cloneNode(true);
              delete clone.dataset.tabSplitBound;
              wrap.replaceWith(clone);
              window.setupTabSplitRootsIn?.(document);
              const nf = clone.querySelector("form.tab-log-expense-form");
              if (nf) {
                delete nf.dataset.remiAjaxWired;
                window.remiBindAjaxSubmitForm?.(nf);
              }
            }
          }
        } else {
          await refreshInlineViewFromServer(form);
          if (form.id && form.id.includes("-edit-")) {
            closeInlineEdit(form.id);
          }
        }
        let dayLabelInput = form.querySelector("input[data-day-label-input]");
        if (!dayLabelInput && form.id) {
          dayLabelInput = document.querySelector(
            `input[data-day-label-input][form="${CSS.escape(form.id)}"]`
          );
        }
        if (!dayLabelInput && form.elements?.length) {
          dayLabelInput = Array.from(form.elements).find(
            (el) => el instanceof HTMLInputElement && el.hasAttribute("data-day-label-input")
          );
        }
        if (dayLabelInput) {
          dayLabelInput.dataset.initialValue = dayLabelInput.value || "";
          form.classList.remove("day-label-dirty");
        }
        if ((form.action || "").toLowerCase().includes("/delete")) {
          const tabMain = document.querySelector("main.tab-page");
          const skipRowRemove =
            Boolean(tabMain) &&
            Boolean(form.closest("[data-tab-refresh-region], .tab-settle-card, .tab-expenses-section"));
          if (!skipRowRemove) {
            const row = form.closest(
              ".timeline-item, .expense-item, .reminder-checklist-item, .budget-tx-view, .budget-mobile-tx-item"
            );
            if (row) {
              const id = row.getAttribute("data-budget-tx-view");
              if (id) {
                document.querySelector(`tr.budget-expense-edit-row[data-budget-edit-for="${CSS.escape(id)}"]`)?.remove();
                document
                  .querySelector(`li.budget-mobile-tx-item[data-budget-tx-view="${CSS.escape(id)}"]`)
                  ?.remove();
              }
              row.remove();
            }
          }
        }
        if (document.querySelector(".trip-details-page .budget-tile")) {
          await refreshBudgetTilesFromPage();
        }
        showToast(inferToastMessage(form));
      } catch (error) {
        showToast(error?.message || "Unable to save right now.");
      }
  };

  const bindAjaxSubmitForm = (form) => {
    if (!(form instanceof HTMLFormElement) || !form.hasAttribute("data-ajax-submit")) return;
    if (form.dataset.remiAjaxWired === "1") return;
    form.dataset.remiAjaxWired = "1";
    form.addEventListener("submit", handleAjaxFormSubmit);
  };
  window.remiBindAjaxSubmitForm = bindAjaxSubmitForm;
  document.querySelectorAll("form[data-ajax-submit]").forEach((form) => bindAjaxSubmitForm(form));

  document.getElementById("tab-exp-view-all-btn")?.addEventListener("click", async () => {
    const btn = document.getElementById("tab-exp-view-all-btn");
    const url = btn?.getAttribute("data-tab-expenses-more-url");
    if (!btn || !url || btn.disabled) return;
    btn.disabled = true;
    try {
      const r = await fetch(url, { headers: { "X-Requested-With": "XMLHttpRequest", Accept: "text/html" } });
      if (!r.ok) throw new Error("bad response");
      const doc = new DOMParser().parseFromString(await r.text(), "text/html");
      const srcTbody = doc.querySelector("[data-tab-load-more-table] tbody");
      const gridWrap = doc.querySelector("[data-tab-load-more-grid]");
      const destTbody = document.getElementById("tab-transactions-tbody");
      const destGrid = document.querySelector("[data-tab-exp-grid]");
      if (srcTbody && destTbody) {
        Array.from(srcTbody.querySelectorAll("tr")).forEach((tr) => destTbody.appendChild(tr));
      }
      if (gridWrap && destGrid) {
        Array.from(gridWrap.querySelectorAll(".tab-exp-grid-card")).forEach((el) => destGrid.appendChild(el));
      }
      destTbody?.querySelectorAll("[data-tab-split-root]").forEach((root) => {
        delete root.dataset.tabSplitBound;
      });
      destTbody?.querySelectorAll("form[data-app-confirm]").forEach((f) => {
        delete f.dataset.remiConfirmWired;
        window.remiWireAppConfirmOnForm?.(f);
      });
      destTbody?.querySelectorAll("form[data-ajax-submit]").forEach((f) => {
        delete f.dataset.remiAjaxWired;
        window.remiBindAjaxSubmitForm?.(f);
      });
      window.setupTabSplitRootsIn?.(destTbody || document);
      window.rewireTabInlineEditOpenButtons?.();
      window.remiSyncTabExpenseInstantFilter?.();
      btn.textContent = "All transactions shown";
      btn.classList.add("tab-exp-view-all-disabled");
    } catch {
      btn.disabled = false;
      showToast("Could not load remaining transactions.");
    }
  });

  document.getElementById("tab-settlements-view-all-btn")?.addEventListener("click", async () => {
    const btn = document.getElementById("tab-settlements-view-all-btn");
    const url = btn?.getAttribute("data-tab-settlements-more-url");
    const ul = document.getElementById("tab-settlements-ul");
    if (!btn || !url || !ul || btn.disabled) return;
    btn.disabled = true;
    try {
      const r = await fetch(url, { headers: { "X-Requested-With": "XMLHttpRequest", Accept: "text/html" } });
      if (!r.ok) throw new Error("bad response");
      const doc = new DOMParser().parseFromString(await r.text(), "text/html");
      const srcUl = doc.querySelector(".tab-load-more-settlements-src");
      if (srcUl) {
        Array.from(srcUl.querySelectorAll(":scope > li")).forEach((li) => ul.appendChild(li));
      }
      ul.querySelectorAll("form[data-app-confirm]").forEach((f) => {
        delete f.dataset.remiConfirmWired;
        window.remiWireAppConfirmOnForm?.(f);
      });
      ul.querySelectorAll("form[data-ajax-submit]").forEach((f) => {
        delete f.dataset.remiAjaxWired;
        window.remiBindAjaxSubmitForm?.(f);
      });
      window.rewireTabInlineEditOpenButtons?.();
      btn.textContent = "All settlements shown";
      btn.classList.add("tab-exp-view-all-disabled");
    } catch {
      btn.disabled = false;
      showToast("Could not load remaining settlements.");
    }
  });
  document.body?.addEventListener("htmx:afterSwap", (event) => {
    const target = event?.detail?.target;
    if (!(target instanceof HTMLElement)) return;
    if (target.id !== "budget-transactions-tbody") return;
    target.querySelectorAll("form[data-app-confirm]").forEach((f) => {
      delete f.dataset.remiConfirmWired;
      window.remiWireAppConfirmOnForm?.(f);
    });
    target.querySelectorAll("form[data-ajax-submit]").forEach((f) => {
      delete f.dataset.remiAjaxWired;
      window.remiBindAjaxSubmitForm?.(f);
    });
    window.remiWireInlineEditOpenButtonsIn?.(target);
  });

  const formOwnerForField = (input) => {
    if (input.form) return input.form;
    const fid = input.getAttribute("form");
    if (fid) return document.getElementById(fid);
    return input.closest("form");
  };

  document.querySelectorAll(".day-label-inline-save-btn").forEach((btn) => {
    btn.addEventListener("click", (event) => {
      event.stopPropagation();
    });
  });

  document.querySelectorAll("input[data-day-label-input]").forEach((input) => {
    const form = formOwnerForField(input);
    const setDirtyState = () => {
      if (!form) return;
      const initial = (input.dataset.initialValue || "").trim();
      const current = (input.value || "").trim();
      form.classList.toggle("day-label-dirty", current !== initial);
    };
    input.dataset.initialValue = input.value || "";
    setDirtyState();
    input.addEventListener("input", setDirtyState);
    input.addEventListener("keydown", (event) => {
      event.stopPropagation();
      const key = event.key || "";
      const isEnter = key === "Enter" || key === "NumpadEnter" || event.keyCode === 13 || event.which === 13;
      if (!isEnter) return;
      event.preventDefault();
      if (!form) return;
      if (typeof form.requestSubmit === "function") {
        form.requestSubmit();
        return;
      }
      form.submit();
    });
  });

  document.querySelectorAll("[data-day-summary-control]").forEach((el) => {
    ["mousedown", "click", "touchstart"].forEach((eventName) => {
      el.addEventListener(eventName, (event) => {
        event.stopPropagation();
      });
    });
    el.addEventListener("keydown", (event) => {
      event.stopPropagation();
    });
  });

  // Prevent <details>/<summary> toggling when interacting with inline day-label controls.
  // Do not preventDefault on the Save button — that would block form submit (button also has data-day-summary-control).
  // Only intercept click on <summary>: preventDefault stops <details> toggle. Do not use capture-phase
  // mousedown/keydown on summary when the target is the day-label input — that runs *before* the event
  // reaches the input and breaks typing (e.g. Space still toggling the section).
  document.querySelectorAll(".day-group > summary").forEach((summaryEl) => {
    summaryEl.addEventListener("click", (event) => {
      if (!(event.target instanceof Element)) return;
      if (event.target.closest('button[type="submit"]')) return;
      if (event.target.closest("[data-day-summary-control]")) {
        event.preventDefault();
        event.stopPropagation();
      }
    }, true);
  });

  document.querySelectorAll("form[method='post']:not([data-ajax-submit])").forEach((form) => {
    form.addEventListener("submit", () => {
      const action = (form.getAttribute("action") || "").trim().toLowerCase();
      if (!action) return;
      let path = action;
      try {
        path = new URL(action, window.location.origin).pathname.toLowerCase();
      } catch {
        /* keep action */
      }
      if (path === "/logout" || path.endsWith("/logout")) return;
      if (path === "/login" || path.endsWith("/login")) return;
      if (path === "/setup" || path.endsWith("/setup")) return;
      try {
        sessionStorage.setItem(TOAST_KEY, inferToastMessage(form));
      } catch (e) {
        /* ignore */
      }
    });
  });

  const mobileFab = document.querySelector("[data-mobile-fab]");
  const mobileFabToggle = document.querySelector("[data-mobile-fab-toggle]");
  const mobileFabMenu = document.querySelector("[data-mobile-fab-menu]");
  const mobileBackdrop = document.querySelector("[data-mobile-sheet-backdrop]");
  const mobileSheets = Array.from(document.querySelectorAll(".mobile-sheet"));
  const openMobileFab = () => {
    if (!mobileFab || !mobileFabMenu) return;
    mobileFab.classList.add("open");
    mobileFabMenu.classList.remove("hidden");
  };
  const closeMobileFab = () => {
    if (!mobileFab || !mobileFabMenu) return;
    mobileFab.classList.remove("open");
    mobileFabMenu.classList.add("hidden");
  };
  const closeMobileSheets = () => {
    mobileSheets.forEach((sheet) => {
      sheet.classList.add("hidden");
      sheet.setAttribute("aria-hidden", "true");
    });
    if (mobileBackdrop) mobileBackdrop.classList.add("hidden");
  };
  const openMobileSheet = (sheetId) => {
    if (!sheetId) return;
    const target = document.getElementById(sheetId);
    if (!target) return;
    closeMobileSheets();
    target.classList.remove("hidden");
    target.setAttribute("aria-hidden", "false");
    if (mobileBackdrop) mobileBackdrop.classList.remove("hidden");
  };
  if (mobileFab && mobileFabToggle) {
    closeMobileFab();
    mobileFabToggle.addEventListener("click", () => {
      if (mobileFab.classList.contains("open")) {
        closeMobileFab();
      } else {
        openMobileFab();
      }
    });
    document.querySelectorAll("[data-mobile-sheet-open]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const sheetId = btn.getAttribute("data-mobile-sheet-open");
        openMobileSheet(sheetId);
        closeMobileFab();
      });
    });
    document.querySelectorAll("[data-mobile-sheet-close]").forEach((btn) => {
      btn.addEventListener("click", () => {
        closeMobileSheets();
      });
    });
    if (mobileBackdrop) {
      mobileBackdrop.addEventListener("click", () => {
        closeMobileSheets();
      });
    }
    document.addEventListener("keydown", (event) => {
      if (event.key !== "Escape") return;
      closeMobileFab();
      closeMobileSheets();
    });
    window.addEventListener("resize", () => {
      if (window.innerWidth > 920) {
        closeMobileFab();
        closeMobileSheets();
      }
    });
  }

  document.querySelectorAll(".mobile-sheet").forEach((sheet) => {
    sheet.addEventListener(
      "focusin",
      (e) => {
        if (window.innerWidth > 920) return;
        const t = e.target;
        if (!(t instanceof HTMLElement)) return;
        if (!t.matches("input, textarea, select")) return;
        const scrollRoot = t.closest(".mobile-sheet > form");
        window.setTimeout(() => {
          try {
            t.scrollIntoView({ block: "center", inline: "nearest", behavior: "smooth" });
          } catch (err) {
            t.scrollIntoView(true);
          }
          if (scrollRoot) {
            const r = t.getBoundingClientRect();
            const pr = scrollRoot.getBoundingClientRect();
            if (r.bottom > pr.bottom - 8) {
              scrollRoot.scrollTop += r.bottom - pr.bottom + 16;
            }
            if (r.top < pr.top + 8) {
              scrollRoot.scrollTop -= pr.top + 8 - r.top;
            }
          }
        }, 280);
      },
      true
    );
  });

  document.addEventListener(
    "focusin",
    (e) => {
      if (window.innerWidth > 920) return;
      if (!document.querySelector("main.trip-details-page")) return;
      const t = e.target;
      if (!(t instanceof HTMLElement)) return;
      if (!t.matches("input, textarea, select")) return;
      const form = t.closest("form.item-edit");
      if (!form || form.classList.contains("hidden")) return;
      window.setTimeout(() => {
        try {
          t.scrollIntoView({ block: "center", inline: "nearest", behavior: "smooth" });
        } catch (err) {
          t.scrollIntoView(true);
        }
      }, 320);
    },
    true
  );

  const openParamForSheet = new URLSearchParams(window.location.search).get("open");
  const stripOpenQueryParam = () => {
    try {
      const u = new URL(window.location.href);
      u.searchParams.delete("open");
      const qs = u.searchParams.toString();
      window.history.replaceState({}, "", u.pathname + (qs ? `?${qs}` : "") + u.hash);
    } catch (e) {
      /* ignore */
    }
  };
  const stopSheetEl = document.getElementById("mobile-sheet-stop");
  if (stopSheetEl && typeof openMobileSheet === "function" && openParamForSheet === "stop") {
    openMobileSheet("mobile-sheet-stop");
    window.setTimeout(() => {
      const loc = stopSheetEl.querySelector("[data-location-input]");
      const focusable = loc || stopSheetEl.querySelector("input, textarea, select");
      if (focusable && typeof focusable.focus === "function") {
        focusable.focus();
      }
    }, 60);
    stripOpenQueryParam();
  }
  const checklistSheetEl = document.getElementById("mobile-sheet-checklist");
  if (checklistSheetEl && typeof openMobileSheet === "function" && openParamForSheet === "checklist") {
    openMobileSheet("mobile-sheet-checklist");
    window.setTimeout(() => {
      const cat = checklistSheetEl.querySelector('select[name="category"]');
      const focusable = cat || checklistSheetEl.querySelector("input, textarea, select");
      if (focusable && typeof focusable.focus === "function") {
        focusable.focus();
      }
    }, 60);
    stripOpenQueryParam();
  }

  const tripSearchInput = document.querySelector("[data-dashboard-trip-search]");
  const tripListsRoot = document.querySelector("[data-dashboard-trip-lists]");
  if (tripSearchInput && tripListsRoot) {
    const noneEl = tripListsRoot.querySelector("[data-trip-search-none]");
    const sections = () => Array.from(tripListsRoot.querySelectorAll("[data-dashboard-trips-section]"));
    const hideWhenQueryEls = () => Array.from(tripListsRoot.querySelectorAll("[data-trip-search-hide-when-query]"));
    const allCards = () => Array.from(tripListsRoot.querySelectorAll(".trip-card[data-trip-search]"));

    const applyTripSearch = () => {
      const q = normalize(tripSearchInput.value);
      const activeQuery = q.length > 0;
      const totalCards = allCards().length;

      document.body.classList.toggle("dashboard-hero-search-compact", activeQuery);

      hideWhenQueryEls().forEach((el) => {
        el.hidden = activeQuery;
      });

      let anyVisible = false;
      sections().forEach((section) => {
        const cards = section.querySelectorAll(".trip-card[data-trip-search]");
        if (cards.length === 0) {
          section.hidden = activeQuery;
          return;
        }
        if (!activeQuery) {
          cards.forEach((card) => {
            card.hidden = false;
          });
          section.hidden = false;
          return;
        }
        let visible = 0;
        cards.forEach((card) => {
          const hay = normalize(card.getAttribute("data-trip-search") || "");
          const show = hay.includes(q);
          card.hidden = !show;
          if (show) visible++;
        });
        section.hidden = visible === 0;
        if (visible > 0) anyVisible = true;
      });

      if (noneEl) {
        const showNone = activeQuery && totalCards > 0 && !anyVisible;
        noneEl.hidden = !showNone;
      }
    };

    tripSearchInput.addEventListener("input", applyTripSearch);
    tripSearchInput.addEventListener("search", applyTripSearch);
    applyTripSearch();
  }

  const mapEl = document.getElementById("map");
  if (!mapEl) {
    /* no trip map */
  } else {
  const gMapKey = (mapEl.getAttribute("data-google-maps-key") || "").trim();
  const defaultLat = parseFloat(mapEl.getAttribute("data-map-lat") || "35.6762");
  const defaultLng = parseFloat(mapEl.getAttribute("data-map-lng") || "139.6503");
  const defaultZoom = parseInt(mapEl.getAttribute("data-map-zoom") || "6", 10);
  const startLat = Number.isNaN(defaultLat) ? 35.6762 : defaultLat;
  const startLng = Number.isNaN(defaultLng) ? 139.6503 : defaultLng;
  const startZoom = Number.isNaN(defaultZoom) ? 6 : defaultZoom;

  const escapeHtmlMap = (s) =>
    String(s || "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");

  const points = Array.from(document.querySelectorAll("[data-map-itinerary-point][data-lat][data-lng]"))
    .map((el) => ({
      lat: parseFloat(el.getAttribute("data-lat") || "0"),
      lng: parseFloat(el.getAttribute("data-lng") || "0"),
      title: el.getAttribute("data-title") || "",
      location: el.getAttribute("data-location") || "",
      day: parseInt(el.getAttribute("data-map-day") || "1", 10) || 1,
      kind: (el.getAttribute("data-marker-kind") || "stop").toLowerCase()
    }))
    .filter((p) => !Number.isNaN(p.lat) && !Number.isNaN(p.lng) && (p.lat !== 0 || p.lng !== 0));

  if (gMapKey) {
    const initGoogleTripMap = () => {
      if (!window.google || !google.maps) return;
      const gMap = new google.maps.Map(mapEl, {
        center: { lat: startLat, lng: startLng },
        zoom: startZoom,
        mapTypeControl: true,
        streetViewControl: true,
        fullscreenControl: true
      });
      const bounds = new google.maps.LatLngBounds();
      points.forEach((p) => {
        bounds.extend({ lat: p.lat, lng: p.lng });
        const marker = new google.maps.Marker({
          position: { lat: p.lat, lng: p.lng },
          map: gMap,
          title: `${p.title} · Day ${p.day}`
        });
        const iw = new google.maps.InfoWindow({
          content: `<div><b>${escapeHtmlMap(p.title)}</b><br><span class="trip-map-popup-day">Day ${p.day}</span><br>${escapeHtmlMap(p.location)}</div>`
        });
        marker.addListener("click", () => iw.open({ anchor: marker, map: gMap }));
      });
      if (points.length === 0) {
        /* default center */
      } else if (points.length === 1) {
        gMap.setZoom(Math.max(startZoom, 12));
        gMap.setCenter({ lat: points[0].lat, lng: points[0].lng });
      } else {
        gMap.fitBounds(bounds, 48);
      }
    };
    if (window.google && google.maps) {
      initGoogleTripMap();
    } else {
      window.remiGoogleTripMapInit = () => {
        initGoogleTripMap();
        try {
          delete window.remiGoogleTripMapInit;
        } catch (e) {
          window.remiGoogleTripMapInit = undefined;
        }
      };
      const gs = document.createElement("script");
      gs.async = true;
      gs.src = `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(gMapKey)}&callback=remiGoogleTripMapInit`;
      document.head.appendChild(gs);
    }
  } else if (typeof L !== "undefined") {
  const map = L.map("map").setView([startLat, startLng], startZoom);
  const lightLayer = L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
    maxZoom: 19,
    attribution: "&copy; OpenStreetMap contributors"
  });
  const darkLayer = L.tileLayer("https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png", {
    maxZoom: 19,
    attribution: "&copy; OpenStreetMap contributors &copy; CARTO"
  });
  let activeBaseLayer = null;
  const setMapTheme = (dark) => {
    const nextLayer = dark ? darkLayer : lightLayer;
    if (activeBaseLayer === nextLayer) return;
    if (activeBaseLayer) {
      map.removeLayer(activeBaseLayer);
    }
    activeBaseLayer = nextLayer;
    activeBaseLayer.addTo(map);
  };
  setMapTheme(document.documentElement.classList.contains("theme-dark"));
  document.addEventListener("remi:themechange", (event) => {
    const dark = Boolean(event?.detail?.dark);
    setMapTheme(dark);
  });

  const dayPalette = [
    "#2563eb",
    "#7c3aed",
    "#0891b2",
    "#ea580c",
    "#db2777",
    "#65a30d",
    "#e11d48",
    "#0f766e",
    "#a855f7",
    "#0ea5e9",
    "#f59e0b",
    "#14b8a6",
    "#ef4444",
    "#84cc16",
    "#6366f1",
    "#d946ef"
  ];
  const kindGlyph = {
    stay: "hotel",
    vehicle: "directions_car",
    flight: "flight",
    stop: "place"
  };
  const uniqueDays = [...new Set(points.map((p) => Math.max(1, p.day)))].sort((a, b) => a - b);
  const colorByDay = new Map();
  uniqueDays.forEach((d, i) => {
    colorByDay.set(d, dayPalette[i % dayPalette.length]);
  });
  const ringForDay = (day) => colorByDay.get(Math.max(1, day)) || dayPalette[0];

  const markersByDay = new Map();
  uniqueDays.forEach((d) => markersByDay.set(d, []));

  points.forEach((p) => {
    const day = Math.max(1, p.day);
    const ring = ringForDay(day);
    const glyph = kindGlyph[p.kind] || kindGlyph.stop;
    const icon = L.divIcon({
      className: "remi-map-marker-wrap",
      html: `<div class="remi-map-marker" style="--remi-ring:${ring}"><span class="material-symbols-outlined" aria-hidden="true">${glyph}</span></div>`,
      iconSize: [34, 34],
      iconAnchor: [17, 34],
      popupAnchor: [0, -32]
    });
    const marker = L.marker([p.lat, p.lng], { icon });
    marker.bindPopup(
      `<b>${escapeHtmlMap(p.title)}</b><br><span class="trip-map-popup-day">Day ${day}</span><br>${escapeHtmlMap(p.location)}`
    );
    markersByDay.get(day).push(marker);
  });

  const selectedDays = new Set(uniqueDays);
  const legendButtons = new Map();

  const visibleMarkersList = () => {
    const out = [];
    selectedDays.forEach((d) => {
      (markersByDay.get(d) || []).forEach((m) => out.push(m));
    });
    return out;
  };

  const syncLegendButtons = () => {
    legendButtons.forEach((btn, d) => {
      const on = selectedDays.has(d);
      btn.classList.toggle("is-off", !on);
      btn.setAttribute("aria-pressed", on ? "true" : "false");
    });
  };

  const fitMapToVisibleMarkers = () => {
    const vis = visibleMarkersList();
    vis.forEach((m) => {
      if (!map.hasLayer(m)) m.addTo(map);
    });
    uniqueDays.forEach((d) => {
      if (selectedDays.has(d)) return;
      (markersByDay.get(d) || []).forEach((m) => {
        if (map.hasLayer(m)) map.removeLayer(m);
      });
    });
    if (vis.length === 0) {
      map.setView([startLat, startLng], startZoom);
      return;
    }
    if (vis.length === 1) {
      map.setView(vis[0].getLatLng(), Math.max(startZoom, 12));
      return;
    }
    const group = L.featureGroup(vis);
    map.fitBounds(group.getBounds(), { padding: [24, 24], maxZoom: 16 });
  };

  if (uniqueDays.length > 0) {
    const leg = document.createElement("div");
    leg.className = "trip-map-day-legend";
    leg.setAttribute("aria-label", "Show or hide markers by itinerary day");
    uniqueDays.forEach((d) => {
      const ring = ringForDay(d);
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "trip-map-day-legend-item";
      btn.dataset.day = String(d);
      btn.setAttribute("aria-pressed", "true");
      btn.title = `Toggle Day ${d} markers on the map`;
      const sw = document.createElement("span");
      sw.className = "trip-map-day-legend-swatch";
      sw.setAttribute("aria-hidden", "true");
      sw.style.setProperty("--trip-map-swatch", ring);
      btn.appendChild(sw);
      btn.appendChild(document.createTextNode(`Day ${d}`));
      btn.addEventListener("click", () => {
        if (selectedDays.has(d)) selectedDays.delete(d);
        else selectedDays.add(d);
        syncLegendButtons();
        fitMapToVisibleMarkers();
      });
      legendButtons.set(d, btn);
      leg.appendChild(btn);
    });
    mapEl.insertAdjacentElement("afterend", leg);
  }

  syncLegendButtons();
  fitMapToVisibleMarkers();
  }
  }

});

(function () {
  const mq = window.matchMedia("(min-width: 681px)");
  const applyExpenseActionsDropdownOpen = () => {
    document.querySelectorAll("main[data-long-press-sheet-root] details.trip-inline-actions-dropdown").forEach((el) => {
      if (mq.matches) {
        el.setAttribute("open", "");
      } else {
        el.removeAttribute("open");
      }
    });
  };
  const init = () => {
    applyExpenseActionsDropdownOpen();
    mq.addEventListener("change", applyExpenseActionsDropdownOpen);
  };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init, { once: true });
  } else {
    init();
  }
})();

(function () {
  const mqMobile = window.matchMedia("(max-width: 680px)");
  const LONG_MS = 520;
  const MOVE_PX = 14;
  const ROW_SEL =
    ".expense-item, .timeline-item, .reminder-checklist-item, .flight-card, .title-row, .budget-mobile-tx-item, .accommodation-card-wrap, .vehicle-rental-item";

  const sheet = document.getElementById("trip-long-press-sheet");
  const titleEl = document.getElementById("trip-long-press-sheet-title");
  const listEl = document.getElementById("trip-long-press-sheet-list");
  if (!sheet || !titleEl || !listEl) {
    return;
  }
  const cancelBtn = sheet.querySelector(".trip-long-press-sheet__cancel");

  let pressTimer = null;
  let startX = 0;
  let startY = 0;
  let activeRow = null;
  let activeRoot = null;
  let ghostClickGuardUntil = 0;
  let ghostClickGuardRow = null;

  /** @type {{ root: Element; onTouchStart: (e: TouchEvent) => void; onTouchMove: (e: TouchEvent) => void; onTouchEnd: () => void }[]} */
  let boundRoots = [];

  function shouldIgnoreLongPressTarget(el, root) {
    if (!el || !root.contains(el)) {
      return true;
    }
    if (el.closest("a, input, textarea, select, label")) {
      return true;
    }
    if (el.closest(".check-row, .check-btn")) {
      return true;
    }
    if (el.closest(".mobile-fab-wrap, .bottom-nav, .sidebar, .trip-long-press-sheet")) {
      return true;
    }
    if (el.closest(".topbar-right")) {
      return true;
    }
    if (el.closest("[data-day-summary-control], .day-label-inline-form, .day-label-inline-input")) {
      return true;
    }
    if (el.closest("button")) {
      return true;
    }
    if (el.closest("summary")) {
      return true;
    }
    return false;
  }

  function labelForRow(row) {
    if (row.matches(".title-row")) {
      return row.querySelector("h2")?.textContent?.trim() || "Trip";
    }
    if (row.matches(".expense-item")) {
      return row.querySelector(".expense-view-title")?.textContent?.trim() || "Spend";
    }
    if (row.matches(".timeline-item")) {
      return row.querySelector(".itinerary-item-view strong")?.textContent?.trim() || "Stop";
    }
    if (row.matches(".reminder-checklist-item")) {
      const s = row.querySelector(".checklist-view > span");
      return s?.textContent?.trim() || "Checklist item";
    }
    if (row.matches(".flight-card")) {
      return row.querySelector(".flight-view h4")?.textContent?.trim() || "Flight";
    }
    if (row.matches(".budget-mobile-tx-item")) {
      return row.querySelector(".budget-mobile-tx-left strong")?.textContent?.trim() || "Expense";
    }
    if (row.matches(".accommodation-card-wrap")) {
      return row.querySelector(".vehicle-main-top h4")?.textContent?.trim() || "Accommodation";
    }
    if (row.matches(".vehicle-rental-item")) {
      return row.querySelector(".vehicle-main-top h4")?.textContent?.trim() || "Vehicle Rental";
    }
    return "Item";
  }

  function clearPress() {
    if (pressTimer) {
      window.clearTimeout(pressTimer);
    }
    pressTimer = null;
    activeRow = null;
    activeRoot = null;
  }

  function openSheetForRow(row) {
    const actionsRoot = row.querySelector("details.trip-inline-actions-dropdown");
    if (!actionsRoot) {
      return;
    }
    const sourceButtons = actionsRoot.querySelectorAll(".trip-inline-actions-buttons button");
    if (!sourceButtons.length) {
      return;
    }
    titleEl.textContent = labelForRow(row);
    listEl.replaceChildren();
    sourceButtons.forEach((src) => {
      const b = document.createElement("button");
      b.type = "button";
      b.className = "trip-long-press-sheet__action";
      const text = (src.textContent || "").trim();
      b.textContent = text || "Action";
      if (src.classList.contains("danger")) {
        b.classList.add("danger");
      }
      b.addEventListener("click", () => {
        sheet.close();
        window.requestAnimationFrame(() => {
          src.click();
        });
      });
      listEl.appendChild(b);
    });
    ghostClickGuardUntil = performance.now() + 380;
    ghostClickGuardRow = row;
    sheet.showModal();
    try {
      if (window.navigator.vibrate) {
        window.navigator.vibrate(12);
      }
    } catch (e) {
      /* ignore */
    }
  }

  function makeRootHandlers(root) {
    function onTouchStart(e) {
      if (!mqMobile.matches || e.touches.length !== 1) {
        return;
      }
      const target = e.target;
      if (shouldIgnoreLongPressTarget(target, root)) {
        return;
      }
      const row = target.closest(ROW_SEL);
      if (!row || !root.contains(row)) {
        return;
      }
      const actionsRoot = row.querySelector("details.trip-inline-actions-dropdown");
      if (!actionsRoot) {
        return;
      }
      const hasActions = actionsRoot.querySelector(".trip-inline-actions-buttons button");
      if (!hasActions) {
        return;
      }
      const t = e.touches[0];
      startX = t.clientX;
      startY = t.clientY;
      activeRow = row;
      activeRoot = root;
      pressTimer = window.setTimeout(() => {
        pressTimer = null;
        const r = activeRow;
        activeRow = null;
        activeRoot = null;
        if (!r) {
          return;
        }
        openSheetForRow(r);
      }, LONG_MS);
    }

    function onTouchMove(e) {
      if (!pressTimer || !e.touches[0]) {
        return;
      }
      const t = e.touches[0];
      const dx = Math.abs(t.clientX - startX);
      const dy = Math.abs(t.clientY - startY);
      if (dx > MOVE_PX || dy > MOVE_PX) {
        clearPress();
      }
    }

    function onTouchEnd() {
      clearPress();
    }

    return { root, onTouchStart, onTouchMove, onTouchEnd };
  }

  function unbindTripLongPress() {
    boundRoots.forEach(({ root, onTouchStart, onTouchMove, onTouchEnd }) => {
      root.removeEventListener("touchstart", onTouchStart);
      root.removeEventListener("touchmove", onTouchMove);
      root.removeEventListener("touchend", onTouchEnd);
      root.removeEventListener("touchcancel", onTouchEnd);
    });
    boundRoots = [];
    clearPress();
  }

  function bindTripLongPress() {
    if (!mqMobile.matches) {
      return;
    }
    document.querySelectorAll("[data-long-press-sheet-root]").forEach((root) => {
      const { onTouchStart, onTouchMove, onTouchEnd } = makeRootHandlers(root);
      root.addEventListener("touchstart", onTouchStart, { passive: true });
      root.addEventListener("touchmove", onTouchMove, { passive: true });
      root.addEventListener("touchend", onTouchEnd);
      root.addEventListener("touchcancel", onTouchEnd);
      boundRoots.push({ root, onTouchStart, onTouchMove, onTouchEnd });
    });
  }

  document.addEventListener(
    "click",
    (e) => {
      if (performance.now() >= ghostClickGuardUntil || !ghostClickGuardRow) {
        return;
      }
      if (ghostClickGuardRow.contains(e.target)) {
        e.preventDefault();
        e.stopPropagation();
        e.stopImmediatePropagation();
      }
    },
    true
  );

  cancelBtn?.addEventListener("click", () => {
    sheet.close();
  });
  sheet.addEventListener("click", (e) => {
    if (e.target === sheet) {
      sheet.close();
    }
  });

  const syncBind = () => {
    unbindTripLongPress();
    if (mqMobile.matches) {
      bindTripLongPress();
    }
  };

  const boot = () => {
    syncBind();
    if (typeof mqMobile.addEventListener === "function") {
      mqMobile.addEventListener("change", syncBind);
    } else if (typeof mqMobile.addListener === "function") {
      mqMobile.addListener(syncBind);
    }
  };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();

// Trip settings: Trip sections (ui_trip_section_*) are masters for module on/off; Primary Content uses ui_vis_main_*
// for layout visibility only. When a master is off, matching main-column checkboxes are disabled; turning the
// master back on restores their prior checked state. Primary Content visibility does not affect Trip sections or sidebar masters.
window.addEventListener("load", () => {
  if (!document.body.classList.contains("page-trip-settings")) {
    return;
  }
  const form = document.getElementById("trip-settings-save-form");
  if (!form) {
    return;
  }
  const MASTER_KEYS = ["itinerary", "checklist", "stay", "vehicle", "flights", "spends", "the_tab"];
  /** @type {Map<string, boolean>} main-column ui_vis_main_* checked state when master goes off */
  const preservedMain = new Map();
  /** @type {Map<string, boolean>} sidebar widget checked state when a master gates it (e.g. add_stop, budget, checklist) */
  const preservedSidebar = new Map();
  /** @type {boolean | null} Trip sections Group Expenses master when Expenses was turned off */
  let preservedTheTabMaster = null;

  function masterCheckbox(key) {
    return form.querySelector(`input[type="checkbox"][data-vis-master="${key}"]`);
  }

  function setMainColumnFromMaster(key) {
    const m = masterCheckbox(key);
    const on = Boolean(m && m.checked);
    const col = form.querySelector(`input[type="checkbox"][data-vis-main-col-key="${key}"]`);
    const label = form.querySelector(`.trip-settings-sec-vis-column-label[data-vis-main-col="${key}"]`);
    if (!col || !label) {
      return;
    }
    if (!on) {
      if (!preservedMain.has(key)) {
        preservedMain.set(key, col.checked);
      }
      col.disabled = true;
      col.checked = false;
      label.classList.add("trip-settings-sec-vis-mirror-label--locked");
    } else {
      col.disabled = false;
      if (preservedMain.has(key)) {
        col.checked = preservedMain.get(key);
      }
      label.classList.remove("trip-settings-sec-vis-mirror-label--locked");
    }
    const hint = label.querySelector(".trip-settings-vis-live-hint");
    if (hint) {
      if (on) {
        hint.setAttribute("hidden", "");
      } else {
        hint.removeAttribute("hidden");
      }
    }
  }

  function applySpendsToTheTabMaster() {
    const spendsOn = Boolean(masterCheckbox("spends")?.checked);
    const tabM = masterCheckbox("the_tab");
    if (!tabM) {
      return;
    }
    if (!spendsOn) {
      if (preservedTheTabMaster === null) {
        preservedTheTabMaster = tabM.checked;
      }
      tabM.disabled = true;
      tabM.checked = false;
    } else {
      tabM.disabled = false;
      if (preservedTheTabMaster !== null) {
        tabM.checked = preservedTheTabMaster;
        preservedTheTabMaster = null;
      }
    }
  }

  function applyAddTabWidgetGate() {
    const spendsOn = Boolean(masterCheckbox("spends")?.checked);
    const tabOn = Boolean(masterCheckbox("the_tab")?.checked);
    const ok = spendsOn && tabOn;
    form.querySelectorAll('[data-add-tab-widget-gate="1"]').forEach((lab) => {
      const cb = lab.querySelector('input[type="checkbox"][name^="ui_vis_sidebar_"]');
      if (!cb) {
        return;
      }
      const sk = (cb.getAttribute("name") || "").replace(/^ui_vis_sidebar_/, "");
      if (!sk) {
        return;
      }
      if (!ok) {
        if (!preservedSidebar.has(sk)) {
          preservedSidebar.set(sk, cb.checked);
        }
        cb.checked = false;
        cb.disabled = true;
        lab.classList.add("trip-settings-toggle-row--muted");
      } else {
        cb.disabled = false;
        lab.classList.remove("trip-settings-toggle-row--muted");
        if (preservedSidebar.has(sk)) {
          cb.checked = preservedSidebar.get(sk);
        }
      }
    });
  }

  function applySpendsGatedSidebar() {
    const spendsOn = Boolean(masterCheckbox("spends")?.checked);
    form.querySelectorAll('[data-spends-master-gated="1"]').forEach((lab) => {
      const cb = lab.querySelector('input[type="checkbox"][name^="ui_vis_sidebar_"]');
      if (!cb) {
        return;
      }
      const sk = (cb.getAttribute("name") || "").replace(/^ui_vis_sidebar_/, "");
      if (!sk) {
        return;
      }
      if (!spendsOn) {
        if (!preservedSidebar.has(sk)) {
          preservedSidebar.set(sk, cb.checked);
        }
        cb.checked = false;
        cb.disabled = true;
        lab.classList.add("trip-settings-toggle-row--muted");
      } else {
        cb.disabled = false;
        lab.classList.remove("trip-settings-toggle-row--muted");
        if (preservedSidebar.has(sk)) {
          cb.checked = preservedSidebar.get(sk);
        }
      }
    });
  }

  function applyItineraryGatedSidebar() {
    const on = Boolean(masterCheckbox("itinerary")?.checked);
    form.querySelectorAll('[data-itinerary-master-gated="1"]').forEach((lab) => {
      const cb = lab.querySelector('input[type="checkbox"][name^="ui_vis_sidebar_"]');
      if (!cb) {
        return;
      }
      const sk = (cb.getAttribute("name") || "").replace(/^ui_vis_sidebar_/, "");
      if (!sk) {
        return;
      }
      if (!on) {
        if (!preservedSidebar.has(sk)) {
          preservedSidebar.set(sk, cb.checked);
        }
        cb.checked = false;
        cb.disabled = true;
        lab.classList.add("trip-settings-toggle-row--muted");
      } else {
        cb.disabled = false;
        lab.classList.remove("trip-settings-toggle-row--muted");
        if (preservedSidebar.has(sk)) {
          cb.checked = preservedSidebar.get(sk);
        }
      }
    });
  }

  function applyChecklistGatedSidebar() {
    const on = Boolean(masterCheckbox("checklist")?.checked);
    form.querySelectorAll('[data-checklist-master-gated="1"]').forEach((lab) => {
      const cb = lab.querySelector('input[type="checkbox"][name^="ui_vis_sidebar_"]');
      if (!cb) {
        return;
      }
      const sk = (cb.getAttribute("name") || "").replace(/^ui_vis_sidebar_/, "");
      if (!sk) {
        return;
      }
      if (!on) {
        if (!preservedSidebar.has(sk)) {
          preservedSidebar.set(sk, cb.checked);
        }
        cb.checked = false;
        cb.disabled = true;
        lab.classList.add("trip-settings-toggle-row--muted");
      } else {
        cb.disabled = false;
        lab.classList.remove("trip-settings-toggle-row--muted");
        if (preservedSidebar.has(sk)) {
          cb.checked = preservedSidebar.get(sk);
        }
      }
    });
  }

  function syncTripSectionMasters() {
    applySpendsToTheTabMaster();
    MASTER_KEYS.forEach((k) => setMainColumnFromMaster(k));
    applySpendsGatedSidebar();
    applyAddTabWidgetGate();
    applyItineraryGatedSidebar();
    applyChecklistGatedSidebar();
  }

  MASTER_KEYS.forEach((key) => {
    masterCheckbox(key)?.addEventListener("change", syncTripSectionMasters);
  });

  syncTripSectionMasters();

  (function initTripGuestsEditor() {
    const seedEl = document.getElementById("trip-guests-seed");
    const editorRoot = document.querySelector("[data-trip-guests-editor]");
    const list = editorRoot?.querySelector("[data-trip-guests-list]");
    const emptyEl = editorRoot?.querySelector("[data-trip-guests-empty]");
    const patchField = editorRoot?.querySelector("[data-trip-guests-patch-field]");
    const tpl = document.getElementById("trip-guest-row-template");
    const input = editorRoot?.querySelector("[data-trip-guest-input]");
    const addBtn = editorRoot?.querySelector("[data-trip-guest-add]");
    if (!seedEl || !editorRoot || !list || !patchField || !(tpl instanceof HTMLTemplateElement)) {
      return;
    }

    /** @type {Set<string>} */
    const initialServerIds = new Set();

    function readSeed() {
      try {
        const raw = seedEl.textContent?.trim() || "[]";
        const arr = JSON.parse(raw);
        return Array.isArray(arr) ? arr : [];
      } catch {
        return [];
      }
    }

    function syncEmptyState() {
      const has = Boolean(list.querySelector("[data-trip-guest-row]"));
      if (emptyEl) {
        emptyEl.toggleAttribute("hidden", has);
      }
    }

    /**
     * @param {string} name
     * @param {string} [serverId]
     */
    function appendRow(name, serverId) {
      const node = tpl.content.firstElementChild?.cloneNode(true);
      if (!(node instanceof HTMLElement)) {
        return;
      }
      if (serverId) {
        node.setAttribute("data-server-guest-id", serverId);
      }
      const nameEl = node.querySelector("[data-guest-name]");
      if (nameEl) {
        nameEl.textContent = name;
      }
      node.querySelector("[data-trip-guest-remove]")?.addEventListener("click", () => {
        node.remove();
        syncEmptyState();
      });
      list.appendChild(node);
      syncEmptyState();
    }

    const seed = readSeed();
    for (const row of seed) {
      if (row && row.id) {
        initialServerIds.add(String(row.id));
      }
      if (row && row.name) {
        appendRow(String(row.name), row.id ? String(row.id) : "");
      }
    }
    syncEmptyState();

    addBtn?.addEventListener("click", () => {
      const name = (input?.value || "").trim();
      if (!name || !input) {
        return;
      }
      appendRow(name, "");
      input.value = "";
      input.focus();
    });

    input?.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        addBtn?.click();
      }
    });

    form.addEventListener("submit", () => {
      const rows = list.querySelectorAll("[data-trip-guest-row]");
      const currentIds = new Set();
      /** @type {string[]} */
      const addNames = [];
      rows.forEach((row) => {
        if (!(row instanceof HTMLElement)) {
          return;
        }
        const id = (row.getAttribute("data-server-guest-id") || "").trim();
        const nameEl = row.querySelector("[data-guest-name]");
        const name = (nameEl?.textContent || "").trim();
        if (!name) {
          return;
        }
        if (id) {
          currentIds.add(id);
        } else {
          addNames.push(name);
        }
      });
      const remove = [...initialServerIds].filter((sid) => !currentIds.has(sid));
      patchField.value = JSON.stringify({ remove, add: addNames });
    });
  })();
});

(function initMobileProfileSheets() {
  function openProfileFromButton(btn) {
    let sel = (btn.getAttribute("data-mobile-profile-target") || "").trim();
    if (!sel) {
      sel = "#trip-mobile-profile-sheet";
    }
    let sheet = null;
    try {
      sheet = document.querySelector(sel);
    } catch (e) {
      return;
    }
    if (sheet instanceof HTMLDialogElement) {
      sheet.showModal();
    }
  }

  document.querySelectorAll("[data-trip-mobile-profile-open], [data-mobile-profile-open]").forEach((btn) => {
    btn.addEventListener("click", () => openProfileFromButton(btn));
  });

  document.querySelectorAll("[data-trip-mobile-profile-close], [data-mobile-profile-close]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const d = btn.closest("dialog");
      if (d instanceof HTMLDialogElement) {
        d.close();
      }
    });
  });

  document.querySelectorAll("dialog.trip-mobile-profile-sheet").forEach((el) => {
    if (!(el instanceof HTMLDialogElement)) {
      return;
    }
    el.addEventListener("click", (e) => {
      if (e.target === el) {
        el.close();
      }
    });
  });
})();

(function initDashboardProfileMenu() {
  const root = document.querySelector("[data-dashboard-profile-menu-root]");
  if (!(root instanceof HTMLElement)) {
    return;
  }
  const toggle = root.querySelector("[data-dashboard-profile-menu-toggle]");
  const panel = root.querySelector(".dashboard-profile-menu__panel");
  if (!(toggle instanceof HTMLButtonElement) || !(panel instanceof HTMLElement)) {
    return;
  }

  const closeMenu = () => {
    panel.hidden = true;
    toggle.setAttribute("aria-expanded", "false");
  };
  const openMenu = () => {
    panel.hidden = false;
    toggle.setAttribute("aria-expanded", "true");
  };
  const toggleMenu = () => {
    if (panel.hidden) {
      openMenu();
    } else {
      closeMenu();
    }
  };

  toggle.addEventListener("click", (event) => {
    event.preventDefault();
    toggleMenu();
  });

  document.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof Node)) {
      return;
    }
    if (!root.contains(target)) {
      closeMenu();
    }
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeMenu();
    }
  });
})();

(function initMobileEntryCarousels() {
  const mq = window.matchMedia("(max-width: 680px)");

  function slideElements(track) {
    return Array.from(track.querySelectorAll(":scope > li"));
  }

  function setupCarousel(wrap) {
    const viewport = wrap.querySelector(".mobile-entry-carousel__viewport");
    const track = wrap.querySelector(".mobile-entry-carousel__track");
    const prevBtn = wrap.querySelector(".mobile-entry-carousel__hint--prev");
    const nextBtn = wrap.querySelector(".mobile-entry-carousel__hint--next");
    if (!viewport || !track || !prevBtn || !nextBtn) return;

    /* Per-category checklist uses a vertical list on small screens (see app.css); skip horizontal carousel. */
    if (wrap.closest(".reminder-checklist-card")) return;

    const setSlideSize = () => {
      const w = viewport.clientWidth;
      if (w > 0) {
        track.style.setProperty("--mobile-carousel-slide", `${w}px`);
      }
    };

    function currentIndex() {
      const slides = slideElements(track);
      if (!slides.length) return 0;
      const w = viewport.clientWidth || 1;
      let i = Math.round(viewport.scrollLeft / w);
      if (i < 0) i = 0;
      if (i >= slides.length) i = slides.length - 1;
      return i;
    }

    function go(delta) {
      const slides = slideElements(track);
      if (slides.length <= 1) return;
      let i = currentIndex();
      i = (i + delta + slides.length) % slides.length;
      const w = viewport.clientWidth || 1;
      viewport.scrollTo({ left: i * w, behavior: "smooth" });
    }

    const syncHints = () => {
      const slides = slideElements(track);
      const show = mq.matches && slides.length > 1;
      prevBtn.toggleAttribute("hidden", !show);
      nextBtn.toggleAttribute("hidden", !show);
      wrap.classList.toggle("mobile-entry-carousel--single", slides.length <= 1);
      setSlideSize();
    };

    prevBtn.addEventListener("click", () => go(-1));
    nextBtn.addEventListener("click", () => go(1));

    if (typeof ResizeObserver !== "undefined") {
      const ro = new ResizeObserver(() => {
        setSlideSize();
      });
      ro.observe(viewport);
    } else {
      window.addEventListener("resize", setSlideSize);
      window.addEventListener("orientationchange", setSlideSize);
    }

    const mo = new MutationObserver(() => {
      syncHints();
    });
    mo.observe(track, { childList: true });

    if (typeof mq.addEventListener === "function") {
      mq.addEventListener("change", syncHints);
    } else if (typeof mq.addListener === "function") {
      mq.addListener(syncHints);
    }
    syncHints();
    requestAnimationFrame(() => setSlideSize());
  }

  const boot = () => {
    document.querySelectorAll("[data-mobile-entry-carousel]").forEach(setupCarousel);
  };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();

(function initTabSplitRoots() {
  function parseJSON(raw) {
    if (raw == null || !String(raw).trim()) return { participants: [], weights: {} };
    try {
      return JSON.parse(String(raw));
    } catch {
      return { participants: [], weights: {} };
    }
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function participantLabelForKey(root, key) {
    for (const cb of root.querySelectorAll("[data-tab-participant-cb]")) {
      if (cb.value === key) {
        const lab = cb.closest("label");
        if (!lab) {
          return key;
        }
        const nameEl = lab.querySelector(".tab-split-bd-name");
        if (nameEl) {
          return nameEl.textContent.trim().replace(/\s+/g, " ");
        }
        return lab.textContent.trim().replace(/\s+/g, " ");
      }
    }
    return key;
  }

  function currencySym(root) {
    return (root.getAttribute("data-currency-symbol") || "$").trim() || "$";
  }

  /** Match Go trips.roundMoney */
  function roundMoney(x) {
    return Math.round(x * 100) / 100;
  }

  /** Match Go trips.fixRoundingDrift: add penny drift to lexicographically last participant key. */
  function fixRoundingDriftAmounts(amounts, keys, target) {
    const out = { ...amounts };
    let sum = 0;
    keys.forEach((k) => {
      sum += out[k] || 0;
    });
    const drift = roundMoney(target - sum);
    if (Math.abs(drift) < 0.001) {
      return out;
    }
    const sorted = [...keys].sort();
    const last = sorted[sorted.length - 1];
    if (last) {
      out[last] = roundMoney((out[last] || 0) + drift);
    }
    return out;
  }

  /** Dollar shares from percentages (same rounding as server SharesForExpense percent branch). */
  function percentAmountsForKeys(total, keys, weights) {
    /** @type {Record<string, number>} */
    const raw = {};
    keys.forEach((k) => {
      const p = weights[k] || 0;
      raw[k] = roundMoney((total * p) / 100);
    });
    return fixRoundingDriftAmounts(raw, keys, total);
  }

  /** Dollar shares from integer weights (same as Go TabSplitShares branch in SharesForExpense). */
  function sharesAmountsForKeys(total, keys, weights) {
    /** @type {Record<string, number>} */
    const raw = {};
    let totalW = 0;
    keys.forEach((k) => {
      const w = weights[k] || 0;
      if (w > 0) {
        totalW += w;
      }
    });
    if (totalW <= 0) {
      keys.forEach((k) => {
        raw[k] = 0;
      });
      return fixRoundingDriftAmounts(raw, keys, total);
    }
    keys.forEach((k) => {
      const w = weights[k] || 0;
      raw[k] = roundMoney((total * w) / totalW);
    });
    return fixRoundingDriftAmounts(raw, keys, total);
  }

  /** Weights from the control set for the active split mode. */
  function collectWeights(root) {
    const mode = (root.querySelector("[data-tab-split-mode]")?.value || "equal").toLowerCase();
    /** @type {Record<string, number>} */
    const weights = {};
    const parseNum = (v) => {
      const n = parseFloat(String(v).replace(/,/g, ""));
      return Number.isNaN(n) ? 0 : n;
    };
    if (mode === "exact") {
      root.querySelectorAll("input.tab-exact-input[data-tab-weight-key]").forEach((inp) => {
        const k = inp.getAttribute("data-tab-weight-key");
        if (!k) return;
        weights[k] = parseNum(inp.value);
      });
    } else if (mode === "percent") {
      root.querySelectorAll("input.tab-percent-input[data-tab-weight-key]").forEach((inp) => {
        const k = inp.getAttribute("data-tab-weight-key");
        if (!k) return;
        weights[k] = parseNum(inp.value);
      });
    } else if (mode === "shares") {
      root.querySelectorAll("input.tab-share-count-input[data-tab-weight-key]").forEach((inp) => {
        const k = inp.getAttribute("data-tab-weight-key");
        if (!k) return;
        const n = parseInt(String(inp.value).trim(), 10);
        weights[k] = Number.isFinite(n) && n > 0 ? n : 0;
      });
    } else {
      root.querySelectorAll("input.tab-split-weight-input[data-tab-weight-key]").forEach((inp) => {
        const k = inp.getAttribute("data-tab-weight-key");
        if (!k) return;
        weights[k] = parseNum(inp.value);
      });
    }
    return weights;
  }

  function percentSelectedSum(root) {
    const keys = selectedKeys(root);
    const w = collectWeights(root);
    return keys.reduce((s, k) => s + (w[k] || 0), 0);
  }

  /** Aligns with server NormalizeTabSplitPayload percent tolerance (0.05). */
  function percentMatches100(root) {
    return Math.abs(percentSelectedSum(root) - 100) <= 0.05 + 1e-9;
  }

  function exactSelectedSum(root) {
    const keys = selectedKeys(root);
    const weights = collectWeights(root);
    let sum = 0;
    keys.forEach((k) => {
      sum += weights[k] || 0;
    });
    return sum;
  }

  function exactSumsMatch(root) {
    const total = parseFloat(String(root.querySelector("[data-tab-amount-input]")?.value || "")) || 0;
    const sum = exactSelectedSum(root);
    return Math.abs(total - sum) < 0.005;
  }

  /** Selected participants must each have an integer share count ≥ 1 (matches server TabSplitShares validation). */
  function sharesSplitValid(root) {
    const keys = selectedKeys(root);
    const w = collectWeights(root);
    for (let i = 0; i < keys.length; i += 1) {
      const x = w[keys[i]];
      if (!(x >= 1) || !Number.isFinite(x)) {
        return false;
      }
    }
    return true;
  }

  function sharesAllocatedSumMatchesTotal(root) {
    const total = parseFloat(String(root.querySelector("[data-tab-amount-input]")?.value || "")) || 0;
    const keys = selectedKeys(root);
    if (keys.length === 0) {
      return true;
    }
    if (!sharesSplitValid(root)) {
      return false;
    }
    const w = collectWeights(root);
    const amounts = sharesAmountsForKeys(total, keys, w);
    let sum = 0;
    keys.forEach((k) => {
      sum += amounts[k] || 0;
    });
    return Math.abs(roundMoney(sum) - roundMoney(total)) < 0.005;
  }

  function applyExactSplitLayout(root, showExact) {
    root.classList.toggle("tab-split-root--exact", showExact);
    root.querySelectorAll("[data-tab-exact-field]").forEach((el) => {
      el.classList.toggle("hidden", !showExact);
      el.setAttribute("aria-hidden", showExact ? "false" : "true");
    });
    root.querySelectorAll("[data-tab-bd-readonly-trailing]").forEach((el) => {
      el.classList.toggle("hidden", showExact);
    });
    root.querySelectorAll("input.tab-exact-input").forEach((inp) => {
      inp.disabled = !showExact;
    });
    const foot = root.querySelector("[data-tab-exact-footer]");
    if (foot) foot.classList.toggle("hidden", !showExact);
  }

  function updateExactSplitFooter(root) {
    const mode = (root.querySelector("[data-tab-split-mode]")?.value || "equal").toLowerCase();
    const remainingEl = root.querySelector("[data-tab-exact-remaining]");
    const matchEl = root.querySelector("[data-tab-exact-match]");
    if (!remainingEl || mode !== "exact") return;
    const sym = currencySym(root);
    const total = parseFloat(String(root.querySelector("[data-tab-amount-input]")?.value || "")) || 0;
    const sum = exactSelectedSum(root);
    const remaining = total - sum;
    let remText;
    if (remaining < -0.005) {
      remText = `-${sym}${(-remaining).toFixed(2)}`;
    } else {
      remText = `${sym}${remaining.toFixed(2)}`;
    }
    remainingEl.textContent = remText;
    if (matchEl) {
      const ok = Math.abs(remaining) < 0.005;
      matchEl.classList.toggle("hidden", !ok);
    }
  }

  function applyPercentSplitLayout(root, on) {
    root.classList.toggle("tab-split-root--percent", on);
    root.querySelectorAll("[data-tab-percent-field]").forEach((el) => {
      el.classList.toggle("hidden", !on);
      el.setAttribute("aria-hidden", on ? "false" : "true");
    });
    root.querySelectorAll("input.tab-percent-input").forEach((inp) => {
      inp.disabled = !on;
    });
    const foot = root.querySelector("[data-tab-percent-footer]");
    if (foot) foot.classList.toggle("hidden", !on);
  }

  function applySharesSplitLayout(root, on) {
    root.classList.toggle("tab-split-root--shares", on);
    root.querySelectorAll("[data-tab-shares-field]").forEach((el) => {
      el.classList.toggle("hidden", !on);
      el.setAttribute("aria-hidden", on ? "false" : "true");
    });
    root.querySelectorAll("input.tab-share-count-input").forEach((inp) => {
      inp.disabled = !on;
    });
  }

  function updatePercentFooter(root) {
    const mode = (root.querySelector("[data-tab-split-mode]")?.value || "equal").toLowerCase();
    const remEl = root.querySelector("[data-tab-percent-remaining]");
    const matchEl = root.querySelector("[data-tab-percent-match]");
    if (!remEl || mode !== "percent") return;
    const sum = percentSelectedSum(root);
    const rem = 100 - sum;
    if (rem < -0.05) {
      remEl.textContent = `${rem.toFixed(1)}%`;
    } else if (rem > 0.05) {
      remEl.textContent = `+${rem.toFixed(1)}%`;
    } else {
      remEl.textContent = "0.0%";
    }
    if (matchEl) {
      matchEl.classList.toggle("hidden", !percentMatches100(root));
    }
  }

  function hasPositiveTabAmount(root) {
    const el = root.querySelector("[data-tab-amount-input]");
    if (!(el instanceof HTMLInputElement)) {
      return true;
    }
    const raw = String(el.value ?? "").trim();
    if (raw === "") {
      return false;
    }
    const n = parseFloat(raw);
    return Number.isFinite(n) && n > 0;
  }

  /** Inline message when split is used without a valid positive amount (see data-tab-amount-warning). */
  function updateTabSplitAmountWarning(root) {
    const warn = root.querySelector("[data-tab-amount-warning]");
    if (!(warn instanceof HTMLElement)) {
      return;
    }
    if (hasPositiveTabAmount(root)) {
      warn.classList.add("hidden");
      return;
    }
    warn.classList.remove("hidden");
  }

  function refreshSplitSubmitState(root) {
    const btn = root.querySelector(".tab-submit-btn");
    if (!btn) return;
    const mode = (root.querySelector("[data-tab-split-mode]")?.value || "equal").toLowerCase();
    btn.removeAttribute("aria-disabled");
    btn.removeAttribute("title");
    const amtInp = root.querySelector("[data-tab-amount-input]");
    if (amtInp instanceof HTMLInputElement && !hasPositiveTabAmount(root)) {
      btn.disabled = true;
      btn.setAttribute("aria-disabled", "true");
      btn.setAttribute("title", "Enter a positive expense amount first.");
      return;
    }
    const total = parseFloat(String(root.querySelector("[data-tab-amount-input]")?.value || "")) || 0;

    if (mode === "exact") {
      const block = total > 0 && !exactSumsMatch(root);
      btn.disabled = block;
      if (block) {
        btn.setAttribute("aria-disabled", "true");
        btn.setAttribute("title", "Exact amounts must add up to the expense total.");
      }
      return;
    }
    if (mode === "percent") {
      const block = total > 0 && !percentMatches100(root);
      btn.disabled = block;
      if (block) {
        btn.setAttribute("aria-disabled", "true");
        btn.setAttribute("title", "Percentages must add up to 100%.");
      }
      return;
    }
    if (mode === "shares") {
      const block =
        total > 0 &&
        (selectedKeys(root).length === 0 || !sharesSplitValid(root) || !sharesAllocatedSumMatchesTotal(root));
      btn.disabled = block;
      if (block && total > 0) {
        btn.setAttribute("aria-disabled", "true");
        btn.setAttribute(
          "title",
          "Each selected person needs at least one share, and split totals must match the expense amount.",
        );
      }
      return;
    }
    btn.disabled = false;
  }

  function renderSplitBreakdown(root) {
    const sym = currencySym(root);
    const amount = parseFloat(String(root.querySelector("[data-tab-amount-input]")?.value || "")) || 0;
    const mode = (root.querySelector("[data-tab-split-mode]")?.value || "equal").toLowerCase();
    const keys = selectedKeys(root);
    /** @type {Record<string, number>} */
    const amounts = {};
    const weights = collectWeights(root);
    if (keys.length === 0) {
      root.querySelectorAll("[data-tab-bd-amt]").forEach((el) => {
        el.textContent = `${sym}0.00`;
      });
      root.querySelectorAll("[data-tab-bd-share-tag]").forEach((t) => {
        t.textContent = "";
        t.classList.add("hidden");
      });
      updateExactSplitFooter(root);
      updatePercentFooter(root);
      refreshSplitSubmitState(root);
      return;
    }
    if (mode === "equal") {
      const share = amount / keys.length;
      keys.forEach((k) => {
        amounts[k] = share;
      });
    } else if (mode === "exact") {
      keys.forEach((k) => {
        amounts[k] = weights[k] || 0;
      });
    } else if (mode === "percent") {
      const fixed = percentAmountsForKeys(amount, keys, weights);
      keys.forEach((k) => {
        amounts[k] = fixed[k] || 0;
      });
    } else if (mode === "shares") {
      const fixed = sharesAmountsForKeys(amount, keys, weights);
      keys.forEach((k) => {
        amounts[k] = fixed[k] || 0;
      });
    }
    root.querySelectorAll("[data-tab-bd-amt]").forEach((el) => {
      const k = el.getAttribute("data-tab-bd-amt") || "";
      if (keys.includes(k)) {
        el.textContent = `${sym}${(amounts[k] || 0).toFixed(2)}`;
      } else {
        el.textContent = `${sym}0.00`;
      }
    });
    root.querySelectorAll("[data-tab-bd-share-tag]").forEach((t) => {
      t.textContent = "";
      t.classList.add("hidden");
    });
    updateExactSplitFooter(root);
    updatePercentFooter(root);
    refreshSplitSubmitState(root);
  }

  /** Sync split row role lines (Owner • Payer, etc.) with the paid-by select. */
  function updateTabSplitRoleLines(root) {
    const sel = root.querySelector("[data-tab-paid-by]");
    const paidBy = (sel?.value || "").trim();
    root.querySelectorAll("label.tab-split-breakdown-row[data-tab-split-row-key]").forEach((row) => {
      const key = row.getAttribute("data-tab-split-row-key") || "";
      const isGuest = row.getAttribute("data-tab-split-is-guest") === "1";
      const isOwner = row.getAttribute("data-tab-split-is-owner") === "1";
      const isYou = row.getAttribute("data-tab-split-is-you") === "1";
      const isPayer = paidBy !== "" && paidBy === key;
      const line = row.querySelector("[data-tab-bd-role-line]");
      if (!line) return;
      let base;
      if (isGuest) {
        base = "Guest";
      } else if (isOwner) {
        base = "Owner";
      } else if (isYou) {
        base = "Participant (You)";
      } else {
        base = "Participant";
      }
      line.textContent = isPayer ? `${base} • Payer` : base;
      line.classList.toggle("tab-split-bd-role-line--owner-payer", Boolean(isPayer && isOwner && !isGuest));
      row.classList.toggle("tab-split-breakdown-row--payer", Boolean(isPayer));
    });
  }

  function wirePaidByChips(root) {
    const sel = root.querySelector("[data-tab-paid-by]");
    if (!sel) {
      return;
    }
    const chips = root.querySelectorAll("[data-tab-paid-chip]");
    function syncFromSelect() {
      const v = sel.value;
      chips.forEach((c) => {
        c.classList.toggle("tab-paid-chip--active", c.value === v);
      });
      updateTabSplitRoleLines(root);
    }
    chips.forEach((btn) => {
      btn.addEventListener("click", () => {
        sel.value = btn.value;
        syncFromSelect();
      });
    });
    sel.addEventListener("change", syncFromSelect);
    syncFromSelect();
  }

  function selectedKeys(root) {
    return Array.from(root.querySelectorAll("[data-tab-participant-cb]:checked")).map((cb) => cb.value);
  }

  function syncJSON(root) {
    const keys = selectedKeys(root);
    const raw = collectWeights(root);
    const weights = {};
    keys.forEach((k) => {
      weights[k] = raw[k] || 0;
    });
    const payload = { participants: keys, weights };
    const hid = root.querySelector("[data-tab-split-json]");
    if (hid) hid.value = JSON.stringify(payload);
  }

  function syncWeightsUI(root, mode) {
    root.classList.toggle("tab-split-root--equal", String(mode || "equal").toLowerCase() === "equal");
    const box = root.querySelector("[data-tab-split-weights]");
    if (!box) return;
    const keys = selectedKeys(root);
    const existing = {};
    root.querySelectorAll("[data-tab-weight-key]").forEach((inp) => {
      const k = inp.getAttribute("data-tab-weight-key");
      if (k) existing[k] = inp.value;
    });

    if (mode === "equal") {
      box.classList.add("hidden");
      box.innerHTML = "";
      applyExactSplitLayout(root, false);
      applyPercentSplitLayout(root, false);
      applySharesSplitLayout(root, false);
      return;
    }

    if (mode === "exact") {
      box.classList.add("hidden");
      box.innerHTML = "";
      applyPercentSplitLayout(root, false);
      applySharesSplitLayout(root, false);
      applyExactSplitLayout(root, true);
      const amount = parseFloat(String(root.querySelector("[data-tab-amount-input]")?.value || "")) || 0;
      const hasAmount = amount > 0;
      const per = keys.length ? amount / keys.length : 0;
      root.querySelectorAll("input.tab-exact-input").forEach((inp) => {
        const k = inp.getAttribute("data-tab-weight-key");
        if (!k) return;
        if (!keys.includes(k)) {
          inp.value = "";
          return;
        }
        const prev = existing[k];
        if (hasAmount && prev != null && String(prev).trim() !== "") {
          inp.value = prev;
        } else {
          inp.value = hasAmount ? per.toFixed(2) : "";
        }
      });
      syncJSON(root);
      updateExactSplitFooter(root);
      refreshSplitSubmitState(root);
      renderSplitBreakdown(root);
      return;
    }

    if (mode === "percent") {
      box.classList.add("hidden");
      box.innerHTML = "";
      applyExactSplitLayout(root, false);
      applySharesSplitLayout(root, false);
      applyPercentSplitLayout(root, true);
      const n = keys.length;
      const each = n > 0 ? 100 / n : 0;
      root.querySelectorAll("input.tab-percent-input").forEach((inp) => {
        const k = inp.getAttribute("data-tab-weight-key");
        if (!k) return;
        if (!keys.includes(k)) {
          inp.value = "";
          return;
        }
        const prev = existing[k];
        if (prev != null && String(prev).trim() !== "") {
          inp.value = prev;
        } else {
          inp.value = each > 0 ? each.toFixed(2) : "";
        }
      });
      syncJSON(root);
      updatePercentFooter(root);
      refreshSplitSubmitState(root);
      renderSplitBreakdown(root);
      return;
    }

    if (mode === "shares") {
      box.classList.add("hidden");
      box.innerHTML = "";
      applyExactSplitLayout(root, false);
      applyPercentSplitLayout(root, false);
      applySharesSplitLayout(root, true);
      root.querySelectorAll("input.tab-share-count-input").forEach((inp) => {
        const k = inp.getAttribute("data-tab-weight-key");
        if (!k) return;
        if (!keys.includes(k)) {
          inp.value = "1";
          return;
        }
        const prev = existing[k];
        if (prev != null && String(prev).trim() !== "") {
          const n = Math.floor(Math.abs(parseFloat(String(prev).replace(/,/g, ""))));
          inp.value = n >= 1 ? String(n) : "1";
        } else {
          inp.value = "1";
        }
      });
      syncJSON(root);
      refreshSplitSubmitState(root);
      renderSplitBreakdown(root);
      return;
    }

    applyExactSplitLayout(root, false);
    applyPercentSplitLayout(root, false);
    applySharesSplitLayout(root, false);
    box.classList.add("hidden");
    box.innerHTML = "";
    renderSplitBreakdown(root);
  }

  function setMode(root, mode, skipSync) {
    const m = (mode || "equal").toLowerCase();
    root.querySelectorAll("[data-tab-split-mode-btn]").forEach((b) => {
      b.classList.toggle("tab-split-mode-btn--active", b.getAttribute("data-tab-split-mode-btn") === m);
    });
    const hid = root.querySelector("[data-tab-split-mode]");
    if (hid) hid.value = m;
    const hint = root.querySelector("[data-tab-split-hint]");
    if (hint) {
      const hints = {
        equal: "Split equally among selected people.",
        exact: "Enter each person’s share in dollars (must add up to the expense total).",
        percent: "Percentages should add up to 100%.",
        shares: "Allocate costs using a proportional share system.",
      };
      hint.textContent = hints[m] || "";
    }
    syncWeightsUI(root, m);
    renderSplitBreakdown(root);
    if (!skipSync) syncJSON(root);
    updateTabSplitAmountWarning(root);
  }

  function applyInitial(root) {
    if (root.hasAttribute("data-initial-json")) {
      const initMode = (root.getAttribute("data-initial-mode") || "equal").toLowerCase();
      const initJson = parseJSON(root.getAttribute("data-initial-json"));
      if (initJson.participants?.length) {
        root.querySelectorAll("[data-tab-participant-cb]").forEach((cb) => {
          cb.checked = initJson.participants.includes(cb.value);
        });
      } else {
        root.querySelectorAll("[data-tab-participant-cb]").forEach((cb) => {
          cb.checked = true;
        });
      }
      setMode(root, initMode, true);
      Object.entries(initJson.weights || {}).forEach(([k, v]) => {
        const found = Array.from(root.querySelectorAll("[data-tab-weight-key]")).find(
          (i) => i.getAttribute("data-tab-weight-key") === k,
        );
        if (!found) return;
        if (found.matches("input.tab-share-count-input")) {
          const n = Math.floor(Number(v));
          found.value = Number.isFinite(n) && n >= 1 ? String(n) : "1";
        } else {
          found.value = v === 0 ? "" : String(v);
        }
      });
      syncJSON(root);
      return;
    }
    const tripDef = parseJSON(root.getAttribute("data-trip-default-json"));
    const tripMode = (root.getAttribute("data-trip-default-mode") || "").trim().toLowerCase();
    if (tripMode && tripDef.participants?.length) {
      root.querySelectorAll("[data-tab-participant-cb]").forEach((cb) => {
        cb.checked = tripDef.participants.includes(cb.value);
      });
      setMode(root, tripMode, true);
      const allowSeedWeights = tripMode !== "exact" || hasPositiveTabAmount(root);
      if (allowSeedWeights) {
        Object.entries(tripDef.weights || {}).forEach(([k, v]) => {
          const found = Array.from(root.querySelectorAll("[data-tab-weight-key]")).find(
            (i) => i.getAttribute("data-tab-weight-key") === k,
          );
          if (!found) return;
          if (found.matches("input.tab-share-count-input")) {
            const n = Math.floor(Number(v));
            found.value = Number.isFinite(n) && n >= 1 ? String(n) : "1";
          } else {
            found.value = v === 0 ? "" : String(v);
          }
        });
      }
      syncJSON(root);
      return;
    }
    const equalBoot = parseJSON(root.getAttribute("data-equal-bootstrap"));
    if (equalBoot.participants?.length) {
      root.querySelectorAll("[data-tab-participant-cb]").forEach((cb) => {
        cb.checked = equalBoot.participants.includes(cb.value);
      });
    }
    setMode(root, "equal", false);
  }

  function setupRoot(root) {
    if (root.dataset.tabSplitBound === "1") return;
    root.dataset.tabSplitBound = "1";
    root.querySelectorAll("[data-tab-split-mode-btn]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const next = btn.getAttribute("data-tab-split-mode-btn") || "equal";
        setMode(root, next);
        if (!hasPositiveTabAmount(root)) {
          const amt = root.querySelector("[data-tab-amount-input]");
          if (amt instanceof HTMLInputElement) {
            amt.focus();
          }
        }
      });
    });
    root.querySelectorAll("[data-tab-participant-cb]").forEach((cb) => {
      cb.addEventListener("change", () => {
        const mode = root.querySelector("[data-tab-split-mode]")?.value || "equal";
        syncWeightsUI(root, mode);
        syncJSON(root);
        renderSplitBreakdown(root);
        updateTabSplitAmountWarning(root);
      });
    });
    root.querySelector("[data-tab-amount-input]")?.addEventListener("input", () => {
      syncJSON(root);
      renderSplitBreakdown(root);
      updateTabSplitAmountWarning(root);
    });
    root.addEventListener("input", (ev) => {
      const t = ev.target;
      if (
        t instanceof HTMLInputElement &&
        (t.matches("input.tab-exact-input[data-tab-weight-key]") ||
          t.matches("input.tab-percent-input[data-tab-weight-key]") ||
          t.matches("input.tab-share-count-input[data-tab-weight-key]"))
      ) {
        if (t.matches("input.tab-share-count-input[data-tab-weight-key]")) {
          let n = parseInt(String(t.value).trim(), 10);
          if (!Number.isFinite(n) || n < 1) {
            n = 1;
          }
          if (String(t.value) !== String(n)) {
            t.value = String(n);
          }
        }
        syncJSON(root);
        renderSplitBreakdown(root);
      }
    });
    root.addEventListener("click", (ev) => {
      const t = ev.target;
      if (!(t instanceof Element)) return;
      const dec = t.closest("[data-tab-share-dec]");
      const inc = t.closest("[data-tab-share-inc]");
      if (!dec && !inc) return;
      const stepper = (dec || inc).closest(".tab-share-stepper");
      const inp = stepper?.querySelector("input.tab-share-count-input[data-tab-weight-key]");
      if (!(inp instanceof HTMLInputElement)) return;
      let n = parseInt(String(inp.value).trim(), 10);
      if (!Number.isFinite(n) || n < 1) n = 1;
      if (dec) n = Math.max(1, n - 1);
      else n = n + 1;
      inp.value = String(n);
      inp.dispatchEvent(new Event("input", { bubbles: true }));
    });
    root.querySelector("form")?.addEventListener("submit", (e) => {
      if (!hasPositiveTabAmount(root)) {
        e.preventDefault();
        updateTabSplitAmountWarning(root);
        const amt = root.querySelector("[data-tab-amount-input]");
        if (amt instanceof HTMLInputElement) {
          amt.focus();
        }
        return;
      }
      syncJSON(root);
      const mode = (root.querySelector("[data-tab-split-mode]")?.value || "equal").toLowerCase();
      if (mode === "exact" && !exactSumsMatch(root)) {
        e.preventDefault();
        updateExactSplitFooter(root);
        refreshSplitSubmitState(root);
      }
      if (mode === "percent" && !percentMatches100(root)) {
        e.preventDefault();
        updatePercentFooter(root);
        refreshSplitSubmitState(root);
      }
      const amt = parseFloat(String(root.querySelector("[data-tab-amount-input]")?.value || "")) || 0;
      if (
        mode === "shares" &&
        amt > 0 &&
        (selectedKeys(root).length === 0 || !sharesSplitValid(root) || !sharesAllocatedSumMatchesTotal(root))
      ) {
        e.preventDefault();
        refreshSplitSubmitState(root);
      }
    });
    applyInitial(root);
    wirePaidByChips(root);
    renderSplitBreakdown(root);
    updateTabSplitAmountWarning(root);
  }

  const boot = () => {
    document.querySelectorAll("[data-tab-split-root]").forEach(setupRoot);
  };
  window.setupTabSplitRootsIn = (container) => {
    if (!container) return;
    container.querySelectorAll("[data-tab-split-root]").forEach((root) => {
      if (root.dataset.tabSplitBound === "1") return;
      setupRoot(root);
    });
  };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();

(function initTabExpenseInstantFilter() {
  const norm = (s) => String(s || "").trim().toLowerCase();

  function apply() {
    const section = document.querySelector("[data-tab-expenses-section]");
    if (!section) return;
    const input = section.querySelector("[data-tab-exp-filter-input]");
    const catSelect = section.querySelector("[data-tab-exp-filter-category]");
    const q = norm(input?.value);
    const catVal = norm(catSelect?.value);
    const isListView =
      typeof window.matchMedia === "function"
        ? !window.matchMedia("(max-width: 920px)").matches
        : true;
    const rows = section.querySelectorAll("tr.tab-expense-row");
    const cards = section.querySelectorAll(".tab-exp-grid-card");
    const emptyEl = section.querySelector("[data-tab-exp-client-empty]");

    rows.forEach((tr) => {
      const hay = norm(tr.getAttribute("data-tab-exp-search"));
      const rowCat = norm(tr.getAttribute("data-tab-exp-category"));
      const qMatch = !q || hay.includes(q);
      const catMatch = !catVal || rowCat === catVal;
      const match = qMatch && catMatch;
      tr.classList.toggle("tab-exp-filter-hidden", !match);
      const id = tr.getAttribute("data-tab-tx-view");
      const editRow = id
        ? section.querySelector(`tr.tab-expense-edit-row[data-tab-edit-for="${CSS.escape(id)}"]`)
        : null;
      if (editRow) {
        editRow.classList.toggle("tab-exp-filter-hidden", !match);
      }
    });

    cards.forEach((card) => {
      const hay = norm(card.getAttribute("data-tab-exp-search"));
      const rowCat = norm(card.getAttribute("data-tab-exp-category"));
      const qMatch = !q || hay.includes(q);
      const catMatch = !catVal || rowCat === catVal;
      const match = qMatch && catMatch;
      card.classList.toggle("tab-exp-filter-hidden", !match);
    });

    let anyVisible = false;
    if (isListView) {
      rows.forEach((tr) => {
        if (!tr.classList.contains("tab-exp-filter-hidden")) anyVisible = true;
      });
    } else {
      cards.forEach((card) => {
        if (!card.classList.contains("tab-exp-filter-hidden")) anyVisible = true;
      });
    }

    if (emptyEl) {
      const count = isListView ? rows.length : cards.length;
      const filtersActive = Boolean(q || catVal);
      const showEmpty = Boolean(filtersActive && count > 0 && !anyVisible);
      emptyEl.classList.toggle("hidden", !showEmpty);
    }
  }

  function wire() {
    const section = document.querySelector("[data-tab-expenses-section]");
    if (!section || section.dataset.remiTabExpFilterWired === "1") return;
    section.dataset.remiTabExpFilterWired = "1";
    const input = section.querySelector("[data-tab-exp-filter-input]");
    const catSelect = section.querySelector("[data-tab-exp-filter-category]");
    const form = section.querySelector("[data-tab-exp-filter-form]");
    input?.addEventListener("input", () => apply());
    input?.addEventListener("keydown", (e) => {
      if (e.key === "Enter") e.preventDefault();
    });
    catSelect?.addEventListener("change", () => apply());
    form?.addEventListener("submit", (e) => {
      e.preventDefault();
    });
    apply();
    const mq = window.matchMedia?.("(max-width: 920px)");
    if (mq?.addEventListener) {
      mq.addEventListener("change", () => apply());
    } else if (mq?.addListener) {
      mq.addListener(() => apply());
    }
  }

  window.remiSyncTabExpenseInstantFilter = apply;

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", wire, { once: true });
  } else {
    wire();
  }
})();

const TAB_OVER_TIME_PAGE_SIZE = 7;

/** When opening Group Expenses via #add-group-expense (or legacy #add-tab), scroll to Log new expense after layout. */
function scrollToTabLogExpenseFromHash() {
  if (!document.body.classList.contains("page-the-tab")) return;
  const h = (window.location.hash || "").trim().toLowerCase();
  if (h !== "#add-tab" && h !== "#add-group-expense") return;
  const el = document.getElementById("add-group-expense") || document.getElementById("add-tab");
  if (!el) return;
  const go = () => {
    el.scrollIntoView({ behavior: "smooth", block: "start" });
  };
  if (typeof requestAnimationFrame === "function") {
    requestAnimationFrame(() => requestAnimationFrame(go));
  } else {
    setTimeout(go, 50);
  }
}

function initTabOverTimeChart() {
  const canvas = document.getElementById("tab-chart-over-time");
  const dataEl = document.getElementById("tab-over-time-data");
  const carousel = document.querySelector("[data-tab-over-time-carousel]");
  if (!canvas || !dataEl || typeof Chart === "undefined") {
    return;
  }
  /** @type {Array<{label?: string, date?: string, amount?: number}>} */
  let points = [];
  try {
    points = JSON.parse(dataEl.textContent?.trim() || "[]");
  } catch {
    points = [];
  }
  const emptyEl = document.querySelector('[data-tab-chart-empty="over-time"]');
  if (!Array.isArray(points) || points.length === 0) {
    canvas.classList.add("hidden");
    emptyEl?.classList.remove("hidden");
    carousel?.classList.add("hidden");
    return;
  }
  emptyEl?.classList.add("hidden");
  canvas.classList.remove("hidden");
  carousel?.classList.remove("hidden");

  const prevBtn = carousel?.querySelector("[data-tab-over-time-prev]");
  const nextBtn = carousel?.querySelector("[data-tab-over-time-next]");

  const existing = Chart.getChart(canvas);
  if (existing) {
    existing.destroy();
  }

  const dark = document.documentElement.classList.contains("theme-dark");
  const lineColor = dark ? "#5eead4" : "#0f766e";
  const fillTop = dark ? "rgba(94, 234, 212, 0.35)" : "rgba(15, 118, 110, 0.2)";
  const fillBot = dark ? "rgba(94, 234, 212, 0.02)" : "rgba(15, 118, 110, 0.02)";
  const gridColor = dark ? "rgba(255,255,255,0.08)" : "rgba(15, 23, 42, 0.08)";
  const tickColor = dark ? "rgba(226,232,240,0.55)" : "rgba(100,116,139,0.9)";

  const totalPages = Math.max(1, Math.ceil(points.length / TAB_OVER_TIME_PAGE_SIZE));
  const state = { page: 0, chunk: /** @type {typeof points} */ ([]) };

  function sliceForPage() {
    const start = state.page * TAB_OVER_TIME_PAGE_SIZE;
    return points.slice(start, start + TAB_OVER_TIME_PAGE_SIZE);
  }

  function syncNav() {
    const multi = totalPages > 1;
    if (carousel) {
      carousel.classList.toggle("mobile-entry-carousel--single", !multi);
    }
    if (prevBtn instanceof HTMLButtonElement) {
      prevBtn.disabled = state.page <= 0;
      prevBtn.toggleAttribute("hidden", !multi);
    }
    if (nextBtn instanceof HTMLButtonElement) {
      nextBtn.disabled = state.page >= totalPages - 1;
      nextBtn.toggleAttribute("hidden", !multi);
    }
  }

  function draw() {
    state.chunk = sliceForPage();
    const labels = state.chunk.map((p) => p.label || "");
    const data = state.chunk.map((p) => (typeof p.amount === "number" ? p.amount : 0));
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      return;
    }
    const grad = ctx.createLinearGradient(0, 0, 0, canvas.height || 200);
    grad.addColorStop(0, fillTop);
    grad.addColorStop(1, fillBot);

    const old = Chart.getChart(canvas);
    if (old) {
      old.destroy();
    }

    new Chart(ctx, {
      type: "line",
      data: {
        labels,
        datasets: [
          {
            data,
            borderColor: lineColor,
            backgroundColor: grad,
            fill: true,
            tension: 0.35,
            borderWidth: 3,
            pointRadius: 0,
            pointHoverRadius: 4,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              title(items) {
                const i = items[0]?.dataIndex;
                if (typeof i !== "number") return "";
                const d = state.chunk[i]?.date;
                return (d && String(d)) || state.chunk[i]?.label || "";
              },
              label(ctx2) {
                const v = ctx2.parsed.y;
                return typeof v === "number" ? v.toFixed(2) : "";
              },
            },
          },
        },
        scales: {
          x: {
            grid: { color: gridColor },
            ticks: {
              color: tickColor,
              maxRotation: 45,
              minRotation: 0,
              autoSkip: false,
              maxTicksLimit: TAB_OVER_TIME_PAGE_SIZE + 1,
            },
          },
          y: {
            beginAtZero: true,
            grid: { color: gridColor },
            ticks: { color: tickColor },
          },
        },
      },
    });
  }

  state.page = 0;
  draw();
  syncNav();

  if (prevBtn instanceof HTMLButtonElement) {
    prevBtn.onclick = () => {
      if (state.page > 0) {
        state.page -= 1;
        draw();
        syncNav();
      }
    };
  }
  if (nextBtn instanceof HTMLButtonElement) {
    nextBtn.onclick = () => {
      if (state.page < totalPages - 1) {
        state.page += 1;
        draw();
        syncNav();
      }
    };
  }
}

(function bootTabOverTimeChart() {
  initTabOverTimeChart();
  window.reinitTabOverTimeChart = () => {
    const canvas = document.getElementById("tab-chart-over-time");
    if (canvas && typeof Chart !== "undefined") {
      const ch = Chart.getChart(canvas);
      if (ch) ch.destroy();
    }
    initTabOverTimeChart();
    scrollToTabLogExpenseFromHash();
  };
  if (document.body.classList.contains("page-the-tab")) {
    window.addEventListener("hashchange", scrollToTabLogExpenseFromHash);
    if (document.readyState === "complete") {
      scrollToTabLogExpenseFromHash();
    } else {
      window.addEventListener("load", () => scrollToTabLogExpenseFromHash(), { once: true });
    }
  }
})();

(function initTabBalanceViewToggle() {
  const main = document.querySelector("main.tab-page");
  if (!main) return;
  main.addEventListener("click", (e) => {
    const btn = e.target.closest("[data-tab-balance-view-btn]");
    if (!btn || !main.contains(btn)) return;
    const section = btn.closest("[data-tab-balances-section]");
    if (!section) return;
    const view = btn.getAttribute("data-tab-balance-view-btn");
    if (view !== "net" && view !== "debts") return;
    const isDebts = view === "debts";
    const panelNet = section.querySelector('[data-tab-balance-panel="net"]');
    const panelDebts = section.querySelector('[data-tab-balance-panel="debts"]');
    const badge = section.querySelector("[data-tab-balance-pending-badge]");
    const formField = main.querySelector("input[data-tab-balance-form-field]");
    section.querySelectorAll("[data-tab-balance-view-btn]").forEach((b) => {
      const v = b.getAttribute("data-tab-balance-view-btn");
      const on = (v === "debts") === isDebts;
      b.classList.toggle("tab-balance-toggle-btn--active", on);
      b.setAttribute("aria-selected", on ? "true" : "false");
    });
    panelNet?.classList.toggle("hidden", isDebts);
    panelDebts?.classList.toggle("hidden", !isDebts);
    badge?.classList.toggle("hidden", !isDebts);
    if (formField instanceof HTMLInputElement) {
      formField.disabled = !isDebts;
      formField.value = isDebts ? "debts" : "";
    }
    try {
      const u = new URL(window.location.href);
      if (isDebts) {
        u.searchParams.set("balance_view", "debts");
      } else {
        u.searchParams.delete("balance_view");
      }
      const qs = u.searchParams.toString();
      window.history.replaceState(null, "", u.pathname + (qs ? `?${qs}` : "") + u.hash);
    } catch {
      /* ignore */
    }
  });
})();

(function initTabSimplifySettleFill() {
  const main = document.querySelector("main.tab-page");
  if (!main) return;
  main.addEventListener("click", (e) => {
    const btn = e.target.closest("[data-tab-settle-payer][data-tab-settle-payee]");
    if (!btn || !main.contains(btn)) return;
    const form = document.querySelector("[data-tab-settlement-form]");
    if (!(form instanceof HTMLFormElement)) return;
    const payerSel = form.querySelector('select[name="payer_user_id"]');
    const payeeSel = form.querySelector('select[name="payee_user_id"]');
    const amountInp = form.querySelector('input[name="amount"]');
    const pKey = btn.getAttribute("data-tab-settle-payer") || "";
    const eKey = btn.getAttribute("data-tab-settle-payee") || "";
    const amt = btn.getAttribute("data-tab-settle-amount") || "";
    if (!(payerSel instanceof HTMLSelectElement) || !(payeeSel instanceof HTMLSelectElement)) return;
    if (Array.from(payerSel.options).some((o) => o.value === pKey)) payerSel.value = pKey;
    if (Array.from(payeeSel.options).some((o) => o.value === eKey)) payeeSel.value = eKey;
    if (amountInp instanceof HTMLInputElement) amountInp.value = amt;
    document.getElementById("record-settlement")?.scrollIntoView({ behavior: "smooth", block: "start" });
    payerSel.focus();
  });
})();
