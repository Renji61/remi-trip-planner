if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch(() => {});
  });
}

window.addEventListener("load", () => {
  const THEME_KEY = "remi-theme";
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
    btn.addEventListener("click", () => {
      const nextDark = !document.documentElement.classList.contains("theme-dark");
      document.documentElement.classList.toggle("theme-dark", nextDark);
      try {
        localStorage.setItem(THEME_KEY, nextDark ? "dark" : "light");
      } catch (e) {
        /* ignore */
      }
      syncThemeIcons();
    });
  });
  syncThemeIcons();

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
  const renderItineraryConnectors = (scopeRoot) => {
    const scopes = scopeRoot ? [scopeRoot] : [document];
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
      editPanel.classList.remove("hidden");
      editPanel.scrollIntoView({ behavior: "smooth", block: "start" });
    });
  }
  if (editPanel && closeEditBtn) {
    closeEditBtn.addEventListener("click", () => {
      editPanel.classList.add("hidden");
    });
  }

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
    const row = form.closest(".timeline-item, .expense-item, .accommodation-item, .vehicle-rental-item, .flight-card, .reminder-checklist-item");
    if (row) row.classList.remove("editing");
    const viewId = itineraryViewIdForForm(formId);
    const view = viewId ? document.getElementById(viewId) : null;
    if (view) view.classList.remove("hidden");
  };

  document.querySelectorAll("[data-inline-edit-open]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const formId = btn.getAttribute("data-inline-edit-open");
      const form = formId ? document.getElementById(formId) : null;
      if (!form) return;
      const row = form.closest(".timeline-item, .expense-item, .accommodation-item, .vehicle-rental-item, .flight-card, .reminder-checklist-item");
      if (row) row.classList.add("editing");
      const viewId = itineraryViewIdForForm(formId);
      const view = viewId ? document.getElementById(viewId) : null;
      if (view) view.classList.add("hidden");
      form.classList.remove("hidden");
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

  const mapEl = document.getElementById("map");
  if (!mapEl || typeof L === "undefined") return;

  const defaultLat = parseFloat(mapEl.getAttribute("data-map-lat") || "14.5995");
  const defaultLng = parseFloat(mapEl.getAttribute("data-map-lng") || "120.9842");
  const defaultZoom = parseInt(mapEl.getAttribute("data-map-zoom") || "6", 10);
  const startLat = Number.isNaN(defaultLat) ? 14.5995 : defaultLat;
  const startLng = Number.isNaN(defaultLng) ? 120.9842 : defaultLng;
  const startZoom = Number.isNaN(defaultZoom) ? 6 : defaultZoom;

  const map = L.map("map").setView([startLat, startLng], startZoom);
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
