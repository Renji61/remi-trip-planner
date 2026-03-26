if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch(() => {});
  });
}

window.addEventListener("load", () => {
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

  const syncThemeIcons = () => {
    const dark = document.documentElement.classList.contains("theme-dark");
    document.querySelectorAll("[data-theme-icon]").forEach((el) => {
      el.textContent = dark ? "light_mode" : "dark_mode";
    });
  };

  document.querySelectorAll("[data-theme-toggle]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const currentlyDark = document.documentElement.classList.contains("theme-dark");
      const nextPref = currentlyDark ? "light" : "dark";
      const fd = new FormData();
      fd.append("theme_preference", nextPref);
      try {
        const res = await fetch("/settings/theme", { method: "POST", body: fd, credentials: "same-origin" });
        if (!res.ok) return;
      } catch (e) {
        return;
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
    if (!Number.isFinite(km) || km <= 0) return "0 km";
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

  const fillMissingCoords = async (scopes) => {
    const byLocation = new Map();
    scopes.forEach((scope) => {
      scope.querySelectorAll(".day-items.timeline .timeline-item[data-itinerary-item]").forEach((el) => {
        if (parseCoords(el)) return;
        const loc = (el.getAttribute("data-location") || "").trim();
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
            const coords = (result && result.lat && result.lng) ? { lat: result.lat, lng: result.lng } : null;
            connectorCoordsCache.set(loc, coords);
            applyCoords(els, coords);
          })
        );
      }
    }
    if (fetches.length > 0) await Promise.all(fetches);
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
          const driveMins = (distanceKm / 35) * 60;
          const walkMins = (distanceKm / 4.8) * 60;
          const li = document.createElement("li");
          li.className = "timeline-connector";
          li.setAttribute("data-itinerary-connector", "");
          li.innerHTML = `
            <div class="timeline-connector-card">
              <div class="timeline-connector-row">
                <span>${formatDuration(driveMins)} · ${formatDistance(distanceKm)} ·</span>
                <a href="${directionsURL(from, to, "driving")}" target="_blank" rel="noreferrer">Directions</a>
              </div>
              <div class="timeline-connector-row">
                <span>${formatDuration(walkMins)} · ${formatDistance(distanceKm)} ·</span>
                <a href="${directionsURL(from, to, "walking")}" target="_blank" rel="noreferrer">Walk</a>
              </div>
            </div>
          `;
          next.insertAdjacentElement("beforebegin", li);
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

  const geocodeLocation = async (locationQuery) => {
    if (!locationQuery) return null;
    const url = new URL("https://nominatim.openstreetmap.org/search");
    url.searchParams.set("q", locationQuery);
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
      displayName: top.display_name || locationQuery
    };
  };

  const searchLocations = async (locationQuery) => {
    if (!locationQuery) return [];
    const url = new URL("https://nominatim.openstreetmap.org/search");
    url.searchParams.set("q", locationQuery);
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
      .map((item) => ({
        lat: parseFloat(item.lat || "0"),
        lng: parseFloat(item.lon || "0"),
        displayName: item.display_name || ""
      }))
      .filter((item) => item.displayName && !Number.isNaN(item.lat) && !Number.isNaN(item.lng));
  };

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

  const closeInlineEdit = (formId) => {
    const form = document.getElementById(formId);
    if (!form) return;
    form.classList.add("hidden");
    const row = form.closest(".timeline-item, .expense-item, .accommodation-item, .accommodation-card-wrap, .vehicle-rental-item, .flight-card, .reminder-checklist-item");
    if (row) row.classList.remove("editing");
    const viewId = itineraryViewIdForForm(formId);
    const view = viewId ? document.getElementById(viewId) : null;
    if (view) view.classList.remove("hidden");
  };

  document.querySelectorAll("[data-inline-edit-open]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const actionDetails = btn.closest("details.trip-inline-actions-dropdown");
      if (actionDetails) {
        actionDetails.removeAttribute("open");
      }
      const formId = btn.getAttribute("data-inline-edit-open");
      const form = formId ? document.getElementById(formId) : null;
      if (!form) return;
      const row = form.closest(".timeline-item, .expense-item, .accommodation-item, .accommodation-card-wrap, .vehicle-rental-item, .flight-card, .reminder-checklist-item");
      if (row) row.classList.add("editing");
      const viewId = itineraryViewIdForForm(formId);
      const view = viewId ? document.getElementById(viewId) : null;
      if (view) view.classList.add("hidden");
      form.classList.remove("hidden");
      const dateInput = form.querySelector("input[name='itinerary_date']");
      if (dateInput) dateInput.dataset.originalValue = dateInput.value;
    });
  });

  document.querySelectorAll("[data-inline-edit-cancel]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const formId = btn.getAttribute("data-inline-edit-cancel");
      if (!formId) return;
      closeInlineEdit(formId);
    });
  });

  const itineraryForm = document.querySelector("[data-itinerary-form]");
  if (itineraryForm) {
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
        }, 150);
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
  }

  document.querySelectorAll("[data-itinerary-form]").forEach((formEl) => {
    if (formEl === itineraryForm) return;
    const locationInput = formEl.querySelector("[data-location-input]");
    const latitudeInput = formEl.querySelector("[data-latitude-input]");
    const longitudeInput = formEl.querySelector("[data-longitude-input]");
    const locationStatus = formEl.querySelector("[data-location-status]");
    const suggestionBox = formEl.querySelector("[data-location-suggestions]");
    const locationLookupEnabled = formEl.getAttribute("data-location-lookup-enabled") !== "false";
    let suggestionTimer = null;
    let latestQueryToken = 0;

    const setStatus = (message, state) => {
      if (!locationStatus) return;
      locationStatus.textContent = message;
      locationStatus.classList.remove("error", "success");
      if (state) locationStatus.classList.add(state);
    };
    const fillCoordinates = (coords) => {
      if (!latitudeInput || !longitudeInput) return;
      latitudeInput.value = coords ? String(coords.lat) : "";
      longitudeInput.value = coords ? String(coords.lng) : "";
    };
    const hideSuggestions = () => {
      if (!suggestionBox) return;
      suggestionBox.classList.add("hidden");
      suggestionBox.innerHTML = "";
    };
    const renderSuggestions = (suggestions) => {
      if (!suggestionBox) return;
      suggestionBox.innerHTML = "";
      if (!suggestions.length) {
        hideSuggestions();
        return;
      }
      suggestions.forEach((suggestion) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "location-suggestion-btn";
        btn.textContent = suggestion.displayName;
        btn.addEventListener("click", () => {
          if (locationInput) locationInput.value = suggestion.displayName;
          fillCoordinates(suggestion);
          setStatus("Location selected and ready to plot on the map.", "success");
          hideSuggestions();
        });
        suggestionBox.appendChild(btn);
      });
      suggestionBox.classList.remove("hidden");
    };

    if (locationInput) {
      locationInput.addEventListener("input", () => {
        fillCoordinates(null);
        if (!locationLookupEnabled) return;
        const trimmed = locationInput.value.trim();
        if (suggestionTimer) clearTimeout(suggestionTimer);
        if (trimmed.length < 3) {
          hideSuggestions();
          return;
        }
        suggestionTimer = window.setTimeout(async () => {
          const token = ++latestQueryToken;
          setStatus("Searching places...");
          const suggestions = await searchLocations(trimmed);
          if (token !== latestQueryToken) return;
          renderSuggestions(suggestions);
          setStatus(suggestions.length ? "Select a place to confirm coordinates." : "No matching places found.", suggestions.length ? null : "error");
        }, 320);
      });
      locationInput.addEventListener("blur", () => {
        window.setTimeout(hideSuggestions, 150);
      });
    }
    formEl.addEventListener("submit", async (event) => {
      if (!locationInput || !locationLookupEnabled) return;
      const query = locationInput.value.trim();
      if (!query) return;
      const coords = await geocodeLocation(query);
      if (!coords) return;
      fillCoordinates(coords);
    });
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
      window.setTimeout(hide, 150);
    });
  });

  document.querySelectorAll("[data-accommodation-form]").forEach((accommodationForm) => {
    const accommodationNameInput = accommodationForm.querySelector("[data-accommodation-name]");
    const accommodationAddressInput = accommodationForm.querySelector("[data-accommodation-address]");
    const accommodationStatus = accommodationForm.querySelector("[data-accommodation-status]");
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
    const action = (form.getAttribute("action") || "").toLowerCase();
    if (action.includes("/delete")) return "Deleted successfully.";
    if (action.includes("/toggle")) return "Updated successfully.";
    if (action.includes("/update")) return "Saved changes.";
    if (action.includes("/trips") && !action.includes("/update")) return "Added successfully.";
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
      ["data-lat", "data-lng", "data-title", "data-location", "data-search-text"].forEach((attr) => {
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

  document.querySelectorAll("form[data-ajax-submit]").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      if (event.defaultPrevented) return;
      event.preventDefault();
      const method = (form.getAttribute("method") || "post").toUpperCase();
      const formData = new FormData(form);
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
        } else if (form.id && form.id.startsWith("expense-edit-")) {
          const moved = await smartRepositionExpenseItem(form);
          if (!moved) {
            try { sessionStorage.setItem(TOAST_KEY, inferToastMessage(form)); } catch (e) { /* ignore */ }
            window.location.reload();
            return;
          }
          closeInlineEdit(form.id);
        } else {
          await refreshInlineViewFromServer(form);
          if (form.id && form.id.includes("-edit-")) {
            closeInlineEdit(form.id);
          }
        }
        const dayLabelInput = form.querySelector("input[data-day-label-input]");
        if (dayLabelInput) {
          dayLabelInput.dataset.initialValue = dayLabelInput.value || "";
          form.classList.remove("day-label-dirty");
        }
        if ((form.action || "").toLowerCase().includes("/delete")) {
          const row = form.closest(".timeline-item, .expense-item, .reminder-checklist-item");
          if (row) row.remove();
        }
        showToast(inferToastMessage(form));
      } catch (error) {
        showToast(error?.message || "Unable to save right now.");
      }
    });
  });

  document.querySelectorAll("input[data-day-label-input]").forEach((input) => {
    const form = input.closest("form");
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
  document.querySelectorAll(".day-group > summary").forEach((summaryEl) => {
    summaryEl.addEventListener("click", (event) => {
      if (event.target instanceof Element && event.target.closest("[data-day-summary-control]")) {
        event.preventDefault();
        event.stopPropagation();
      }
    }, true);
    summaryEl.addEventListener("mousedown", (event) => {
      if (event.target instanceof Element && event.target.closest("[data-day-summary-control]")) {
        event.stopPropagation();
      }
    }, true);
    summaryEl.addEventListener("touchstart", (event) => {
      if (event.target instanceof Element && event.target.closest("[data-day-summary-control]")) {
        event.stopPropagation();
      }
    }, true);
    summaryEl.addEventListener("keydown", (event) => {
      if (event.target instanceof Element && event.target.closest("[data-day-summary-control]")) {
        event.stopPropagation();
      }
    }, true);
  });

  document.querySelectorAll("form[method='post']:not([data-ajax-submit])").forEach((form) => {
    form.addEventListener("submit", () => {
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
  if (!mapEl || typeof L === "undefined") return;

  const defaultLat = parseFloat(mapEl.getAttribute("data-map-lat") || "14.5995");
  const defaultLng = parseFloat(mapEl.getAttribute("data-map-lng") || "120.9842");
  const defaultZoom = parseInt(mapEl.getAttribute("data-map-zoom") || "6", 10);
  const startLat = Number.isNaN(defaultLat) ? 14.5995 : defaultLat;
  const startLng = Number.isNaN(defaultLng) ? 120.9842 : defaultLng;
  const startZoom = Number.isNaN(defaultZoom) ? 6 : defaultZoom;

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
  let routeLine = null;
  if (latLngs.length > 0) {
    const dark = document.documentElement.classList.contains("theme-dark");
    routeLine = L.polyline(latLngs, { color: dark ? "#60a5fa" : "#2563eb" }).addTo(map);
    map.fitBounds(routeLine.getBounds(), { padding: [20, 20] });
  }
  document.addEventListener("remi:themechange", (event) => {
    if (!routeLine) return;
    const dark = Boolean(event?.detail?.dark);
    routeLine.setStyle({ color: dark ? "#60a5fa" : "#2563eb" });
  });
});

(function () {
  const mq = window.matchMedia("(min-width: 681px)");
  const applyExpenseActionsDropdownOpen = () => {
    document.querySelectorAll(".trip-details-page details.trip-inline-actions-dropdown").forEach((el) => {
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
  const ROW_SEL = ".expense-item, .timeline-item, .reminder-checklist-item, .flight-card, .title-row";

  const root = document.querySelector(".trip-details-page");
  const sheet = document.getElementById("trip-long-press-sheet");
  const titleEl = document.getElementById("trip-long-press-sheet-title");
  const listEl = document.getElementById("trip-long-press-sheet-list");
  if (!root || !sheet || !titleEl || !listEl) {
    return;
  }
  const cancelBtn = sheet.querySelector(".trip-long-press-sheet__cancel");

  function shouldIgnoreLongPressTarget(el) {
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
    return "Item";
  }

  let pressTimer = null;
  let startX = 0;
  let startY = 0;
  let activeRow = null;
  let ghostClickGuardUntil = 0;
  let ghostClickGuardRow = null;

  function clearPress() {
    if (pressTimer) {
      window.clearTimeout(pressTimer);
    }
    pressTimer = null;
    activeRow = null;
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

  function onTouchStart(e) {
    if (!mqMobile.matches || e.touches.length !== 1) {
      return;
    }
    const target = e.target;
    if (shouldIgnoreLongPressTarget(target)) {
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
    pressTimer = window.setTimeout(() => {
      pressTimer = null;
      const r = activeRow;
      activeRow = null;
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

  function bindTripLongPress() {
    if (!mqMobile.matches) {
      return;
    }
    root.addEventListener("touchstart", onTouchStart, { passive: true });
    root.addEventListener("touchmove", onTouchMove, { passive: true });
    root.addEventListener("touchend", onTouchEnd);
    root.addEventListener("touchcancel", onTouchEnd);
  }

  function unbindTripLongPress() {
    root.removeEventListener("touchstart", onTouchStart);
    root.removeEventListener("touchmove", onTouchMove);
    root.removeEventListener("touchend", onTouchEnd);
    root.removeEventListener("touchcancel", onTouchEnd);
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
    mqMobile.addEventListener("change", syncBind);
  };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();
