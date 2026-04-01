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

(function remiUploadFilenameGuard() {
  const BLOCKED = new Set([
    "exe",
    "sh",
    "bat",
    "msi",
    "js",
    "py",
    "php",
    "cmd",
    "ps1",
    "com",
    "scr",
    "vbs",
    "jar",
    "dll",
    "app",
    "deb",
    "rpm",
    "dmg",
    "wsf",
    "hta",
    "lnk"
  ]);
  window.remiIsBlockedUploadFilename = (name) => {
    const raw = String(name || "").trim();
    if (!raw) return true;
    const base = raw.replace(/^[\\/]+/, "").split(/[/\\]/).pop() || "";
    if (!base || base === "." || base.includes("..")) return true;
    const parts = base.toLowerCase().split(".");
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i].trim();
      if (part && BLOCKED.has(part)) return true;
    }
    return false;
  };
  window.remiBlockedUploadFilenameMessage =
    "This file type is not allowed — executables and scripts cannot be uploaded.";
})();

(function remiDateFields() {
  const ISO_RE = /^(\d{4})-(\d{2})-(\d{2})$/;
  const ISO_DT_RE = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/;
  const MOBILE_BP = 900;
  const pad2 = (n) => String(n).padStart(2, "0");
  const monthNames = [
    "January",
    "February",
    "March",
    "April",
    "May",
    "June",
    "July",
    "August",
    "September",
    "October",
    "November",
    "December"
  ];

  function isoToDisplay(iso, mdy) {
    const m = String(iso || "").trim().match(ISO_RE);
    if (!m) return "";
    const [, y, mo, d] = m;
    return mdy ? `${mo}-${d}-${y}` : `${d}-${mo}-${y}`;
  }

  function normalizeTimeHM(raw) {
    const src = String(raw || "").trim();
    if (!src) return "";
    const m12 = src.match(/^(\d{1,2}):(\d{2})\s*(am|pm)$/i);
    if (m12) {
      let h = parseInt(m12[1], 10);
      const mm = parseInt(m12[2], 10);
      const ap = m12[3].toLowerCase();
      if (!Number.isFinite(h) || !Number.isFinite(mm) || h < 1 || h > 12 || mm < 0 || mm > 59) return "";
      if (ap === "am") h = h === 12 ? 0 : h;
      else h = h === 12 ? 12 : h + 12;
      return `${pad2(h)}:${pad2(mm)}`;
    }
    const p = src.split(":");
    if (p.length < 2) return "";
    const h = parseInt(p[0], 10);
    const mm = parseInt(p[1], 10);
    if (!Number.isFinite(h) || !Number.isFinite(mm) || h < 0 || h > 23 || mm < 0 || mm > 59) return "";
    return `${pad2(h)}:${pad2(mm)}`;
  }

  function timeHMToDisplay(raw) {
    const t = normalizeTimeHM(raw);
    if (!t) return "";
    let h = parseInt(t.slice(0, 2), 10);
    const m = t.slice(3, 5);
    const isPm = h >= 12;
    h = h % 12;
    if (h === 0) h = 12;
    return `${pad2(h)}:${m} ${isPm ? "PM" : "AM"}`;
  }

  function splitIsoLocalDateTime(dt) {
    const m = String(dt || "").trim().match(ISO_DT_RE);
    if (!m) return { dateIso: "", time: "" };
    return { dateIso: `${m[1]}-${m[2]}-${m[3]}`, time: `${m[4]}:${m[5]}` };
  }

  function clampIso(iso, min, max) {
    if (!iso) return iso;
    if (min && iso < min) return min;
    if (max && iso > max) return max;
    return iso;
  }

  function clampDateTimeLocal(isoDt, min, max) {
    if (!isoDt) return isoDt;
    let v = isoDt;
    if (min && v < min) v = min;
    if (max && v > max) v = max;
    return v;
  }

  function parseISODate(iso) {
    const m = String(iso || "").trim().match(ISO_RE);
    if (!m) return null;
    return new Date(parseInt(m[1], 10), parseInt(m[2], 10) - 1, parseInt(m[3], 10));
  }

  function toISODate(d) {
    return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
  }

  function startOfMonth(d) {
    return new Date(d.getFullYear(), d.getMonth(), 1);
  }

  function isMobileLayout() {
    return window.matchMedia(`(max-width: ${MOBILE_BP}px)`).matches;
  }

  const picker = (() => {
    let host = null;
    let panel = null;
    let title = null;
    let body = null;
    let state = null;
    let escHandler = null;
    let outsideHandler = null;
    let repositionHandler = null;

    const ensure = () => {
      if (host) return;
      host = document.createElement("div");
      host.className = "remi-picker-host hidden";
      host.innerHTML = [
        '<div class="remi-picker-backdrop" data-remi-picker-close></div>',
        '<section class="remi-picker-panel" role="dialog" aria-modal="true">',
        '  <header class="remi-picker-header">',
        '    <h3 class="remi-picker-title"></h3>',
        '    <button type="button" class="remi-picker-close" data-remi-picker-close aria-label="Close picker">',
        '      <span class="material-symbols-outlined">close</span>',
        "    </button>",
        "  </header>",
        '  <div class="remi-picker-body"></div>',
        "</section>"
      ].join("");
      document.body.appendChild(host);
      panel = host.querySelector(".remi-picker-panel");
      title = host.querySelector(".remi-picker-title");
      body = host.querySelector(".remi-picker-body");
      host.querySelectorAll("[data-remi-picker-close]").forEach((btn) => {
        btn.addEventListener("click", () => close("cancel"));
      });
    };

    const positionDesktop = () => {
      if (!state || !state.anchor || !(state.anchor instanceof HTMLElement) || !panel) return;
      if (!state.anchor.isConnected) {
        close("anchor-detached");
        return;
      }
      const r = state.anchor.getBoundingClientRect();
      if (r.bottom < 0 || r.top > window.innerHeight) {
        close("anchor-out-of-view");
        return;
      }
      const panelWidth = Math.max(280, Math.round(r.width));
      const maxLeft = Math.max(12, window.innerWidth - panelWidth - 12);
      const left = Math.min(maxLeft, Math.max(12, Math.round(r.left)));
      panel.style.left = `${left}px`;
      panel.style.top = `${Math.round(r.bottom + 10)}px`;
      panel.style.width = `${panelWidth}px`;
    };

    const close = (reason) => {
      if (!host || !state) return;
      if (escHandler) document.removeEventListener("keydown", escHandler, true);
      if (outsideHandler) document.removeEventListener("pointerdown", outsideHandler, true);
      if (repositionHandler) {
        window.removeEventListener("resize", repositionHandler);
        window.removeEventListener("scroll", repositionHandler, true);
      }
      escHandler = null;
      outsideHandler = null;
      repositionHandler = null;
      host.classList.add("hidden");
      host.classList.remove("mobile", "desktop");
      body.innerHTML = "";
      panel.removeAttribute("style");
      const done = state && state.onClose;
      const s = state;
      state = null;
      if (typeof done === "function") done(reason, s && s.value);
    };

    const open = (opts) => {
      ensure();
      if (state) close("replace");
      state = Object.assign({}, opts || {});
      const mobile = isMobileLayout();
      state.mobile = mobile;
      title.textContent = state.title || "";
      host.classList.remove("hidden");
      host.classList.add(mobile ? "mobile" : "desktop");
      body.innerHTML = "";
      if (typeof state.render === "function") {
        state.render(body, close, state);
      }
      escHandler = (e) => {
        if (e.key === "Escape") {
          e.preventDefault();
          close("cancel");
        }
      };
      outsideHandler = (e) => {
        if (!panel.contains(e.target)) close("cancel");
      };
      document.addEventListener("keydown", escHandler, true);
      document.addEventListener("pointerdown", outsideHandler, true);
      if (!mobile) {
        repositionHandler = () => positionDesktop();
        window.addEventListener("resize", repositionHandler);
        window.addEventListener("scroll", repositionHandler, true);
        positionDesktop();
      }
    };

    return { open, close };
  })();

  function renderDatePicker(root, close, opts) {
    const min = opts.min || "";
    const max = opts.max || "";
    let selectedIso = opts.value || "";
    let viewMonth = startOfMonth(parseISODate(selectedIso) || parseISODate(min) || new Date());
    const weekdays = ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"];

    const wrap = document.createElement("div");
    wrap.className = "remi-date-picker";
    wrap.innerHTML = [
      '<div class="remi-date-picker-head-row">',
      '  <button type="button" class="remi-cal-nav" data-cal-nav="-1" aria-label="Previous month">',
      '    <span class="material-symbols-outlined">chevron_left</span>',
      "  </button>",
      '  <button type="button" class="remi-cal-month-pill" data-cal-label></button>',
      '  <button type="button" class="remi-cal-nav" data-cal-nav="1" aria-label="Next month">',
      '    <span class="material-symbols-outlined">chevron_right</span>',
      "  </button>",
      "</div>",
      '<div class="remi-cal-weekdays"></div>',
      '<div class="remi-cal-grid"></div>',
      '<div class="remi-picker-actions">',
      '  <button type="button" class="secondary-btn" data-cal-cancel>Cancel</button>',
      '  <button type="button" class="primary-btn" data-cal-confirm>Confirm Date</button>',
      "</div>"
    ].join("");
    root.appendChild(wrap);
    const weekRow = wrap.querySelector(".remi-cal-weekdays");
    const grid = wrap.querySelector(".remi-cal-grid");
    const label = wrap.querySelector("[data-cal-label]");
    weekdays.forEach((d) => {
      const el = document.createElement("span");
      el.textContent = d;
      weekRow.appendChild(el);
    });

    const draw = () => {
      label.textContent = `${monthNames[viewMonth.getMonth()]} ${viewMonth.getFullYear()}`;
      grid.innerHTML = "";
      const first = new Date(viewMonth.getFullYear(), viewMonth.getMonth(), 1);
      const daysInMonth = new Date(viewMonth.getFullYear(), viewMonth.getMonth() + 1, 0).getDate();
      const offset = (first.getDay() + 6) % 7;
      for (let i = 0; i < offset; i += 1) {
        const blank = document.createElement("span");
        blank.className = "muted";
        blank.textContent = "";
        grid.appendChild(blank);
      }
      for (let day = 1; day <= daysInMonth; day += 1) {
        const d = new Date(viewMonth.getFullYear(), viewMonth.getMonth(), day);
        const iso = toISODate(d);
        const b = document.createElement("button");
        b.type = "button";
        b.className = "remi-cal-day";
        b.textContent = String(day);
        if ((min && iso < min) || (max && iso > max)) b.disabled = true;
        if (iso === selectedIso) b.classList.add("is-selected");
        b.addEventListener("click", () => {
          selectedIso = iso;
          draw();
          if (!opts.mobile) {
            opts.onConfirm(clampIso(selectedIso, min, max));
            close("confirm");
          }
        });
        grid.appendChild(b);
      }
      wrap.querySelector("[data-cal-confirm]").disabled = !selectedIso;
    };

    draw();
    wrap.querySelectorAll("[data-cal-nav]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const delta = parseInt(btn.getAttribute("data-cal-nav") || "0", 10);
        viewMonth = new Date(viewMonth.getFullYear(), viewMonth.getMonth() + delta, 1);
        draw();
      });
    });
    wrap.querySelector("[data-cal-cancel]").addEventListener("click", () => close("cancel"));
    wrap.querySelector("[data-cal-confirm]").addEventListener("click", () => {
      opts.onConfirm(clampIso(selectedIso, min, max));
      close("confirm");
    });
  }

  function renderTimePicker(root, close, opts) {
    const initial = normalizeTimeHM(opts.value) || "08:30";
    let hour24 = parseInt(initial.slice(0, 2), 10);
    let minute = parseInt(initial.slice(3, 5), 10);
    let period = hour24 >= 12 ? "pm" : "am";
    let hour12 = hour24 % 12;
    if (hour12 === 0) hour12 = 12;
    const wrap = document.createElement("div");
    wrap.className = "remi-time-picker";
    wrap.innerHTML = [
      '<div class="remi-time-selected" data-time-head></div>',
      '<div class="remi-time-grid">',
      '  <div class="remi-time-column">',
      '    <div class="remi-time-label">HOUR</div>',
      '    <div class="remi-time-list" data-hours></div>',
      "  </div>",
      '  <div class="remi-time-column">',
      '    <div class="remi-time-label">MIN</div>',
      '    <div class="remi-time-list" data-minutes></div>',
      "  </div>",
      '  <div class="remi-time-meridian">',
      '    <button type="button" data-ap="am">AM</button>',
      '    <button type="button" data-ap="pm">PM</button>',
      "  </div>",
      "</div>",
      '<div class="remi-picker-actions">',
      '  <button type="button" class="secondary-btn" data-time-cancel>Cancel</button>',
      '  <button type="button" class="primary-btn" data-time-confirm>Confirm Time</button>',
      "</div>"
    ].join("");
    root.appendChild(wrap);
    const hourList = wrap.querySelector("[data-hours]");
    const minuteList = wrap.querySelector("[data-minutes]");
    const head = wrap.querySelector("[data-time-head]");

    const compute = () => {
      let h = hour12 % 12;
      if (period === "pm") h += 12;
      return `${pad2(h)}:${pad2(minute)}`;
    };

    const draw = () => {
      head.textContent = `Selected Time ${timeHMToDisplay(compute())}`;
      hourList.innerHTML = "";
      for (let h = 1; h <= 12; h += 1) {
        const b = document.createElement("button");
        b.type = "button";
        b.className = "remi-time-opt";
        if (h === hour12) b.classList.add("is-selected");
        b.textContent = pad2(h);
        b.addEventListener("click", () => {
          hour12 = h;
          draw();
        });
        hourList.appendChild(b);
      }
      minuteList.innerHTML = "";
      for (let m = 0; m < 60; m += 5) {
        const b = document.createElement("button");
        b.type = "button";
        b.className = "remi-time-opt";
        if (m === minute) b.classList.add("is-selected");
        b.textContent = pad2(m);
        b.addEventListener("click", () => {
          minute = m;
          draw();
        });
        minuteList.appendChild(b);
      }
      wrap.querySelectorAll("[data-ap]").forEach((b) => {
        b.classList.toggle("is-selected", b.getAttribute("data-ap") === period);
      });
    };

    draw();
    wrap.querySelectorAll("[data-ap]").forEach((b) => {
      b.addEventListener("click", () => {
        period = b.getAttribute("data-ap") || "am";
        draw();
      });
    });
    wrap.querySelector("[data-time-cancel]").addEventListener("click", () => close("cancel"));
    wrap.querySelector("[data-time-confirm]").addEventListener("click", () => {
      opts.onConfirm(compute());
      close("confirm");
    });
  }

  function openDatePicker(anchor, options) {
    picker.open({
      anchor,
      title: "Select Date",
      value: options.value || "",
      min: options.min || "",
      max: options.max || "",
      onConfirm: options.onConfirm,
      onClose: options.onClose,
      render: renderDatePicker
    });
  }

  function openTimePicker(anchor, options) {
    picker.open({
      anchor,
      title: options.title || "Set Time",
      value: options.value || "",
      onConfirm: options.onConfirm,
      onClose: options.onClose,
      render: renderTimePicker
    });
  }

  function inferPickerTitle(input, fallback) {
    const label = input?.closest?.("label");
    if (!(label instanceof HTMLElement)) return fallback;
    const text = (label.textContent || "").replace(/\s+/g, " ").trim();
    if (!text) return fallback;
    if (/arriv/i.test(text)) return "Set Arrival Time";
    if (/depart/i.test(text)) return "Set Departure Time";
    if (/start/i.test(text)) return "Set Start Time";
    if (/end/i.test(text)) return "Set End Time";
    if (/check-?in/i.test(text)) return "Set Check-in Time";
    if (/check-?out/i.test(text)) return "Set Check-out Time";
    if (/pick-?up/i.test(text)) return "Set Pick-up Time";
    if (/drop-?off/i.test(text)) return "Set Drop-off Time";
    return fallback;
  }

  function makeFieldOpenable(input, onOpen) {
    input.readOnly = true;
    input.addEventListener("click", (e) => {
      e.preventDefault();
      onOpen();
    });
    input.addEventListener("focus", () => {
      onOpen();
    });
    input.addEventListener("keydown", (e) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        onOpen();
      }
    });
  }

  function wireDateWrap(wrap) {
    if (!(wrap instanceof HTMLElement) || wrap.dataset.remiDateWired === "1") return;
    wrap.dataset.remiDateWired = "1";
    const mdy = wrap.getAttribute("data-mdy") === "1";
    const min = wrap.getAttribute("data-min") || "";
    const max = wrap.getAttribute("data-max") || "";
    const hidden = wrap.querySelector(".remi-date-iso");
    const vis = wrap.querySelector(".remi-date-visible");
    if (!(hidden instanceof HTMLInputElement) || !(vis instanceof HTMLInputElement)) return;
    const required = vis.required;
    const sync = () => {
      vis.value = isoToDisplay(hidden.value, mdy);
      vis.setCustomValidity("");
    };
    const validate = () => {
      vis.setCustomValidity("");
      if (required && !(hidden.value || "").trim()) {
        vis.setCustomValidity("Select a date");
        return false;
      }
      return true;
    };
    sync();
    makeFieldOpenable(vis, () => {
      openDatePicker(vis, {
        value: hidden.value || "",
        min,
        max,
        onConfirm: (iso) => {
          hidden.value = clampIso(iso, min, max);
          sync();
          hidden.dispatchEvent(new Event("change", { bubbles: true }));
        }
      });
    });
    const form = wrap.closest("form");
    if (form) {
      form.addEventListener("submit", (e) => {
        if (!validate()) {
          e.preventDefault();
          e.stopImmediatePropagation();
          vis.reportValidity();
          return;
        }
        hidden.dispatchEvent(new Event("change", { bubbles: true }));
      }, true);
    }
  }

  function wireTimeWrap(wrap) {
    if (!(wrap instanceof HTMLElement) || wrap.dataset.remiTimeWired === "1") return;
    wrap.dataset.remiTimeWired = "1";
    const hidden = wrap.querySelector(".remi-time-iso");
    const vis = wrap.querySelector(".remi-time-visible");
    if (!(hidden instanceof HTMLInputElement) || !(vis instanceof HTMLInputElement)) return;
    const required = vis.required;
    const sync = () => {
      const normalized = normalizeTimeHM(hidden.value);
      hidden.value = normalized;
      vis.value = timeHMToDisplay(normalized);
      vis.setCustomValidity("");
    };
    const validate = () => {
      vis.setCustomValidity("");
      if (required && !(hidden.value || "").trim()) {
        vis.setCustomValidity("Select a time");
        return false;
      }
      return true;
    };
    sync();
    makeFieldOpenable(vis, () => {
      openTimePicker(vis, {
        title: inferPickerTitle(vis, "Set Time"),
        value: hidden.value || "08:30",
        onConfirm: (hm) => {
          hidden.value = normalizeTimeHM(hm);
          sync();
          hidden.dispatchEvent(new Event("change", { bubbles: true }));
        }
      });
    });
    const form = wrap.closest("form");
    if (form) {
      form.addEventListener("submit", (e) => {
        if (!validate()) {
          e.preventDefault();
          e.stopImmediatePropagation();
          vis.reportValidity();
          return;
        }
        hidden.dispatchEvent(new Event("change", { bubbles: true }));
      }, true);
    }
  }

  function wireDateTimeWrap(wrap) {
    if (!(wrap instanceof HTMLElement) || wrap.dataset.remiDatetimeWired === "1") return;
    wrap.dataset.remiDatetimeWired = "1";
    const mdy = wrap.getAttribute("data-mdy") === "1";
    const min = wrap.getAttribute("data-min") || "";
    const max = wrap.getAttribute("data-max") || "";
    const hidden = wrap.querySelector(".remi-datetime-iso");
    const dateVis = wrap.querySelector(".remi-datetime-date-part");
    const timeVis = wrap.querySelector(".remi-datetime-time-part");
    if (!(hidden instanceof HTMLInputElement) || !(dateVis instanceof HTMLInputElement) || !(timeVis instanceof HTMLInputElement)) return;
    const required = dateVis.required || timeVis.required;
    const sync = () => {
      const sp = splitIsoLocalDateTime(hidden.value);
      dateVis.value = sp.dateIso ? isoToDisplay(sp.dateIso, mdy) : "";
      timeVis.value = timeHMToDisplay(sp.time);
      dateVis.setCustomValidity("");
      timeVis.setCustomValidity("");
    };
    if (!(hidden.value || "").trim()) {
      const now = new Date();
      const todayLocal = `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
      let value = `${todayLocal}T${pad2(now.getHours())}:${pad2(now.getMinutes())}`;
      if (min && value < min) value = min;
      if (max && value > max) value = max;
      hidden.value = value;
    }
    const updateDate = (dateIso) => {
      const sp = splitIsoLocalDateTime(hidden.value);
      const t = normalizeTimeHM(sp.time || "08:30");
      const next = dateIso ? `${dateIso}T${t}` : "";
      hidden.value = clampDateTimeLocal(next, min, max);
      sync();
      hidden.dispatchEvent(new Event("change", { bubbles: true }));
    };
    const updateTime = (hm) => {
      const sp = splitIsoLocalDateTime(hidden.value);
      if (!sp.dateIso) {
        const now = new Date();
        sp.dateIso = `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
      }
      const next = `${sp.dateIso}T${normalizeTimeHM(hm)}`;
      hidden.value = clampDateTimeLocal(next, min, max);
      sync();
      hidden.dispatchEvent(new Event("change", { bubbles: true }));
    };
    const validate = () => {
      dateVis.setCustomValidity("");
      timeVis.setCustomValidity("");
      if (!required) return true;
      const sp = splitIsoLocalDateTime(hidden.value);
      if (!sp.dateIso) dateVis.setCustomValidity("Select a date");
      if (!sp.time) timeVis.setCustomValidity("Select a time");
      return !!(sp.dateIso && sp.time);
    };
    sync();
    makeFieldOpenable(dateVis, () => {
      const sp = splitIsoLocalDateTime(hidden.value);
      openDatePicker(dateVis, {
        value: sp.dateIso,
        min: min ? min.slice(0, 10) : "",
        max: max ? max.slice(0, 10) : "",
        onConfirm: (iso) => updateDate(iso)
      });
    });
    makeFieldOpenable(timeVis, () => {
      const sp = splitIsoLocalDateTime(hidden.value);
      openTimePicker(timeVis, {
        value: sp.time || "08:30",
        title: inferPickerTitle(timeVis, "Set Time"),
        onConfirm: (hm) => updateTime(hm)
      });
    });
    const form = wrap.closest("form");
    if (form) {
      form.addEventListener("submit", (e) => {
        if (!validate()) {
          e.preventDefault();
          dateVis.reportValidity();
          timeVis.reportValidity();
          return;
        }
        hidden.dispatchEvent(new Event("change", { bubbles: true }));
      }, true);
    }
  }

  window.remiWireDateFieldsIn = (root) => {
    const scope = root && typeof root.querySelectorAll === "function" ? root : document;
    scope.querySelectorAll("[data-remi-date]").forEach((el) => {
      if (el instanceof HTMLElement) wireDateWrap(el);
    });
  };
  window.remiWireDateTimeFieldsIn = (root) => {
    const scope = root && typeof root.querySelectorAll === "function" ? root : document;
    scope.querySelectorAll("[data-remi-datetime]").forEach((el) => {
      if (el instanceof HTMLElement) wireDateTimeWrap(el);
    });
  };
  window.remiWireTimeFieldsIn = (root) => {
    const scope = root && typeof root.querySelectorAll === "function" ? root : document;
    scope.querySelectorAll("[data-remi-time]").forEach((el) => {
      if (el instanceof HTMLElement) wireTimeWrap(el);
    });
  };

  const boot = () => {
    window.remiWireDateFieldsIn(document);
    window.remiWireDateTimeFieldsIn(document);
    window.remiWireTimeFieldsIn(document);
  };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();

(function remiToastBootstrap() {
  if (typeof window.remiShowToast === "function") return;
  window.remiShowToast = (message) => {
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
})();

const GMAPS_BROWSER_KEY_RE = /^AIza[0-9A-Za-z_-]{20,}$/;

const maskGoogleMapsKeyForDisplay = (key) => {
  const k = String(key || "").trim();
  if (!k) return "••••••••";
  if (k.length <= 8) return "•".repeat(k.length);
  const head = k.slice(0, 4);
  const tail = k.slice(-4);
  const midLen = Math.min(28, Math.max(8, k.length - 8));
  return `${head}${"•".repeat(midLen)}${tail}`;
};

const copyTextToClipboard = async (text) => {
  const t = String(text || "");
  if (!t) return false;
  try {
    if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
      await navigator.clipboard.writeText(t);
      return true;
    }
  } catch (e) {
    /* fall through */
  }
  try {
    const ta = document.createElement("textarea");
    ta.value = t;
    ta.setAttribute("readonly", "");
    ta.style.position = "fixed";
    ta.style.left = "-9999px";
    document.body.appendChild(ta);
    ta.select();
    const ok = document.execCommand("copy");
    ta.remove();
    return ok;
  } catch (e2) {
    return false;
  }
};

const initSiteSettingsGoogleMapsKeySection = (mapForm) => {
  const section = mapForm.querySelector("[data-gmaps-key-section]");
  if (!(section instanceof HTMLElement)) return;

  const editWrap = section.querySelector("[data-gmaps-key-edit]");
  const savedCard = section.querySelector("[data-gmaps-key-saved]");
  const textInput = section.querySelector("[data-gmaps-key-input]");
  const submitHidden = section.querySelector("[data-gmaps-key-submit]");
  const clearCheckbox = section.querySelector("[data-gmaps-clear-checkbox]");
  const maskEl = section.querySelector("[data-gmaps-key-mask]");
  const validationEl = section.querySelector("[data-gmaps-key-validation]");
  const hasSavedCardEl = savedCard instanceof HTMLElement;

  if (
    !editWrap ||
    !(textInput instanceof HTMLInputElement) ||
    !(submitHidden instanceof HTMLInputElement) ||
    !(clearCheckbox instanceof HTMLInputElement)
  ) {
    return;
  }

  const initialSavedKey = (section.getAttribute("data-saved-key") || "").trim();
  let savedKeyForCopy = initialSavedKey;
  let clearOnSave = false;
  let showingSavedCard = initialSavedKey.length > 0;

  const resetValidation = () => {
    if (validationEl instanceof HTMLElement) {
      validationEl.textContent = "";
      validationEl.hidden = true;
      validationEl.classList.remove("gmaps-key-validation--ok", "gmaps-key-validation--warn");
    }
  };

  const updateValidationHint = () => {
    if (!(validationEl instanceof HTMLElement)) return;
    const v = textInput.value.trim();
    if (!v) {
      resetValidation();
      return;
    }
    validationEl.hidden = false;
    if (GMAPS_BROWSER_KEY_RE.test(v)) {
      validationEl.textContent = "Format looks like a valid browser API key.";
      validationEl.classList.remove("gmaps-key-validation--warn");
      validationEl.classList.add("gmaps-key-validation--ok");
    } else {
      validationEl.textContent =
        "Typical browser keys start with AIza and are about 39 characters. Double-check before saving.";
      validationEl.classList.remove("gmaps-key-validation--ok");
      validationEl.classList.add("gmaps-key-validation--warn");
    }
  };

  const render = () => {
    clearCheckbox.checked = clearOnSave;
    const hasStoredKey = Boolean(savedKeyForCopy.trim());
    const showSavedUi = hasSavedCardEl && hasStoredKey && showingSavedCard && !clearOnSave;
    if (showSavedUi) {
      savedCard.hidden = false;
      editWrap.hidden = true;
      if (maskEl) {
        maskEl.textContent = maskGoogleMapsKeyForDisplay(savedKeyForCopy);
      }
      submitHidden.value = "";
      textInput.value = "";
      textInput.removeAttribute("name");
      resetValidation();
    } else {
      if (hasSavedCardEl) {
        savedCard.hidden = true;
      }
      editWrap.hidden = false;
      submitHidden.value = textInput.value.trim();
    }
  };

  const onUpdateKey = () => {
    clearOnSave = false;
    clearCheckbox.checked = false;
    showingSavedCard = false;
    textInput.value = "";
    submitHidden.value = "";
    resetValidation();
    render();
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        try {
          textInput.focus({ preventScroll: false });
        } catch (e) {
          textInput.focus();
        }
      });
    });
  };

  const onDeleteKeyConfirmed = () => {
    savedKeyForCopy = "";
    showingSavedCard = false;
    clearOnSave = true;
    textInput.value = "";
    submitHidden.value = "";
    section.setAttribute("data-saved-key", "");
    section.setAttribute("data-initial-has-key", "0");
    resetValidation();
    render();
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        try {
          textInput.focus({ preventScroll: false });
        } catch (e) {
          textInput.focus();
        }
      });
    });
  };

  render();

  textInput.addEventListener("input", () => {
    if (clearOnSave && textInput.value.trim() !== "") {
      clearOnSave = false;
      clearCheckbox.checked = false;
    }
    submitHidden.value = textInput.value.trim();
    updateValidationHint();
  });

  const openDeleteConfirm = () => {
    const run = window.remiOpenAppConfirm;
    if (typeof run === "function") {
      run({
        title: "Delete Google Maps API key?",
        body: "Trip maps and address or location suggestions will use OpenStreetMap instead of Google. Save the form to apply this change.",
        okText: "Delete key",
        icon: "delete_forever",
        variant: "danger",
        onConfirm: onDeleteKeyConfirmed
      });
      return;
    }
    if (
      window.confirm(
        "Delete the Google Maps API key? Trip maps and address or location suggestions will use OpenStreetMap instead of Google. Save the form to apply."
      )
    ) {
      onDeleteKeyConfirmed();
    }
  };

  mapForm.addEventListener(
    "click",
    (e) => {
      const t = e.target;
      if (!(t instanceof Element)) return;
      const updateBtn = t.closest("[data-gmaps-key-update]");
      if (updateBtn && section.contains(updateBtn)) {
        e.preventDefault();
        onUpdateKey();
        return;
      }
      const deleteBtn = t.closest("[data-gmaps-key-delete]");
      if (deleteBtn && section.contains(deleteBtn)) {
        e.preventDefault();
        openDeleteConfirm();
        return;
      }
      const copyBtn = t.closest("[data-gmaps-key-copy]");
      if (copyBtn && section.contains(copyBtn)) {
        e.preventDefault();
        const toCopy = savedKeyForCopy.trim();
        if (!toCopy) return;
        void (async () => {
          const toast = typeof window.remiShowToast === "function" ? window.remiShowToast : null;
          const ok = await copyTextToClipboard(toCopy);
          if (ok) {
            toast?.("Google Maps API key copied.");
          } else {
            toast?.("Could not copy to clipboard.");
          }
        })();
      }
    },
    true
  );

  mapForm.addEventListener(
    "submit",
    (e) => {
      if (clearOnSave) {
        submitHidden.value = "";
        clearCheckbox.checked = true;
        return;
      }
      clearCheckbox.checked = false;
      submitHidden.value = textInput.value.trim();
      const draft = submitHidden.value;
      if (draft !== "" && !GMAPS_BROWSER_KEY_RE.test(draft)) {
        e.preventDefault();
        if (typeof window.remiShowToast === "function") {
          window.remiShowToast("That value does not look like a Google browser API key. Fix it or delete the key.");
        }
        textInput.focus();
        updateValidationHint();
      }
    },
    true
  );
};

window.addEventListener("load", () => {
  /** Document / Element / DocumentFragment — not `instanceof ParentNode` (ParentNode is not a JS global in Firefox and others). */
  const remiIsDomQueryRoot = (node) => Boolean(node && typeof node.querySelectorAll === "function");

  /** Inject hidden csrf_token into POST forms under `root` when missing (e.g. after HTMX/AJAX DOM swaps). */
  const remiEnsureCsrfOnPostForms = (root) => {
    const csrfMeta = document.querySelector("meta[name='csrf-token']");
    const csrfToken = csrfMeta && csrfMeta.getAttribute("content") ? csrfMeta.getAttribute("content").trim() : "";
    if (!csrfToken) return;
    const scope = remiIsDomQueryRoot(root) ? root : document;
    scope.querySelectorAll("form[method='post'], form[method='POST']").forEach((form) => {
      if (form.querySelector("input[name='csrf_token']")) return;
      const action = (form.getAttribute("action") || "").toLowerCase();
      if (action === "/login" || action === "/setup" || action === "/register" || action.endsWith("/logout")) return;
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = "csrf_token";
      input.value = csrfToken;
      form.appendChild(input);
    });
  };
  window.remiEnsureCsrfOnPostForms = remiEnsureCsrfOnPostForms;

  const csrfMeta = document.querySelector("meta[name='csrf-token']");
  const csrfToken = csrfMeta && csrfMeta.getAttribute("content") ? csrfMeta.getAttribute("content").trim() : "";
  if (csrfToken) {
    remiEnsureCsrfOnPostForms(document);
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

  const showToast =
    typeof window.remiShowToast === "function"
      ? window.remiShowToast
      : (message) => {
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
  document.querySelectorAll("input[type='date']").forEach((input) => {
    if ((input.value || "").trim() !== "") return;
    let value = todayLocal;
    const min = (input.min || "").trim();
    const max = (input.max || "").trim();
    if (min && value < min) value = min;
    if (max && value > max) value = max;
    input.value = value;
  });

  const normalize = (value) => (value || "").trim().toLowerCase();

  const remiSuggestURL = () => {
    const shell = document.querySelector("main.app-shell");
    const u = shell && shell.getAttribute("data-location-suggest-url");
    return (u && u.trim()) || "/api/location/suggest";
  };
  const remiGeocodeURL = () => "/api/location/geocode";
  /** BCP 47 language for geocoder labels: matches <html lang> (en UI → English place names). */
  const remiPreferredLocationLang = () => {
    const fromDoc =
      document.documentElement && document.documentElement.getAttribute("lang");
    if (fromDoc && String(fromDoc).trim()) {
      return String(fromDoc).trim();
    }
    return (typeof navigator !== "undefined" && navigator.language) || "en";
  };
  const remiLocationLangQuery = () =>
    `&lang=${encodeURIComponent(remiPreferredLocationLang())}`;
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

  const GEOCODE_CLIENT_CACHE_MAX = 400;
  const geocodeClientCache = new Map();
  const geocodeClientInflight = new Map();
  const normalizeGeocodeRetryQuery = (value) => {
    const raw = String(value || "").trim();
    if (!raw) return "";
    return raw
      .replace(/\s+/g, " ")
      .replace(/\s*,\s*/g, ", ")
      .replace(/[;|]+/g, ", ")
      .replace(/,+$/g, "")
      .trim();
  };

  /** Must be declared before fillMissingCoords / renderItineraryConnectors (avoid TDZ ReferenceError on load). */
  const geocodeLocationUncached = async (q) => {
    const tryLookup = async (query) => {
      const clean = (query || "").trim();
      if (!clean) return null;
      try {
        const res = await fetch(
          `${remiGeocodeURL()}?q=${encodeURIComponent(clean)}${remiLocationLangQuery()}`,
          {
            credentials: "same-origin",
            headers: { Accept: "application/json" }
          }
        );
        if (res.ok) {
          const top = await res.json();
          const lat = parseFloat(top.lat ?? top.Lat ?? "0");
          const lng = parseFloat(top.lng ?? top.Lon ?? "0");
          if (Number.isFinite(lat) && Number.isFinite(lng) && (lat !== 0 || lng !== 0)) {
            return {
              lat,
              lng,
              displayName: top.displayName || clean
            };
          }
        }
      } catch (e) {
        /* fallback */
      }
      const url = new URL("https://nominatim.openstreetmap.org/search");
      url.searchParams.set("q", clean);
      url.searchParams.set("format", "jsonv2");
      url.searchParams.set("limit", "1");
      const response = await fetch(url.toString(), {
        headers: {
          Accept: "application/json",
          "Accept-Language": remiPreferredLocationLang()
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
      const lat = parseFloat(top.lat || "0");
      const lng = parseFloat(top.lon || "0");
      if (!Number.isFinite(lat) || !Number.isFinite(lng) || (lat === 0 && lng === 0)) {
        return null;
      }
      return {
        lat,
        lng,
        displayName: top.display_name || clean
      };
    };
    const primary = await tryLookup(q);
    if (primary) return primary;
    const retryQuery = normalizeGeocodeRetryQuery(q);
    if (!retryQuery || retryQuery === (q || "").trim()) return null;
    return tryLookup(retryQuery);
  };

  const geocodeLocation = async (locationQuery) => {
    const q = (locationQuery || "").trim();
    if (!q) return null;
    const ck = `${normalize(q)}\0${remiPreferredLocationLang().toLowerCase()}`;
    if (geocodeClientCache.has(ck)) {
      const hit = geocodeClientCache.get(ck);
      return hit ? { lat: hit.lat, lng: hit.lng, displayName: hit.displayName } : null;
    }
    let inflight = geocodeClientInflight.get(ck);
    if (!inflight) {
      inflight = (async () => {
        try {
          return await geocodeLocationUncached(q);
        } finally {
          geocodeClientInflight.delete(ck);
        }
      })();
      geocodeClientInflight.set(ck, inflight);
    }
    const result = await inflight;
    if (result && Number.isFinite(result.lat) && Number.isFinite(result.lng)) {
      if (geocodeClientCache.size >= GEOCODE_CLIENT_CACHE_MAX) {
        geocodeClientCache.delete(geocodeClientCache.keys().next().value);
      }
      geocodeClientCache.set(ck, {
        lat: result.lat,
        lng: result.lng,
        displayName: result.displayName
      });
    }
    return result;
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

  const LOCATION_SUGGEST_CLIENT_CACHE_MAX = 200;
  const locationSuggestClientCache = new Map();
  const locationSuggestClientInflight = new Map();

  const searchLocations = async (locationQuery) => {
    const q = (locationQuery || "").trim();
    if (!q) return [];
    const ck = `${normalize(q)}\0${remiPreferredLocationLang().toLowerCase()}`;
    if (locationSuggestClientCache.has(ck)) {
      return locationSuggestClientCache.get(ck).map((item) => ({ ...item }));
    }
    let inflight = locationSuggestClientInflight.get(ck);
    if (!inflight) {
      inflight = (async () => {
        try {
          try {
            const res = await fetch(
              `${remiSuggestURL()}?q=${encodeURIComponent(q)}${remiLocationLangQuery()}`,
              {
                credentials: "same-origin",
                headers: { Accept: "application/json" }
              }
            );
            if (res.ok) {
              const data = await res.json();
              if (Array.isArray(data) && data.length > 0) {
                return data
                  .map((item) => {
                    const lat = parseFloat(item.lat ?? item.Lat ?? "0");
                    const lng = parseFloat(item.lng ?? item.Lon ?? "0");
                    const displayName = item.displayName || item.display_name || "";
                    const shortName =
                      item.shortName ||
                      (displayName ? displayName.split(",")[0].trim() : "") ||
                      item.name ||
                      displayName;
                    return { lat, lng, displayName, shortName };
                  })
                  .filter(
                    (item) =>
                      (item.displayName || item.shortName) && !Number.isNaN(item.lat) && !Number.isNaN(item.lng)
                  );
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
              Accept: "application/json",
              "Accept-Language": remiPreferredLocationLang()
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
              const shortName =
                (displayName ? displayName.split(",")[0].trim() : "") || name || displayName;
              return { lat, lng, displayName, shortName };
            })
            .filter(
              (item) => (item.displayName || item.shortName) && !Number.isNaN(item.lat) && !Number.isNaN(item.lng)
            );
        } finally {
          locationSuggestClientInflight.delete(ck);
        }
      })();
      locationSuggestClientInflight.set(ck, inflight);
    }
    const out = await inflight;
    if (out.length > 0) {
      if (locationSuggestClientCache.size >= LOCATION_SUGGEST_CLIENT_CACHE_MAX) {
        locationSuggestClientCache.delete(locationSuggestClientCache.keys().next().value);
      }
      locationSuggestClientCache.set(
        ck,
        out.map((item) => ({ ...item }))
      );
    }
    return out.map((item) => ({ ...item }));
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
    const labelField = heroForm.querySelector("[data-home-map-label]");
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
      if (labelField) labelField.value = "";
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
            if (labelField) labelField.value = label;
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

  const DASHBOARD_NEW_TRIP_NAME_HASH = "#dashboard-new-trip-name";
  const focusDashboardNewTripNameFromHash = () => {
    if (!document.body.classList.contains("page-home")) return;
    if (window.location.hash !== DASHBOARD_NEW_TRIP_NAME_HASH) return;
    const el = document.getElementById("dashboard-new-trip-name");
    if (!(el instanceof HTMLInputElement)) return;
    window.requestAnimationFrame(() => {
      el.scrollIntoView({ block: "center", behavior: "smooth" });
      window.requestAnimationFrame(() => {
        try {
          el.focus({ preventScroll: true });
        } catch {
          el.focus();
        }
      });
    });
  };
  if (document.body.classList.contains("page-home")) {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", focusDashboardNewTripNameFromHash, { once: true });
    } else {
      focusDashboardNewTripNameFromHash();
    }
    window.addEventListener("hashchange", focusDashboardNewTripNameFromHash);
  }

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

  document.querySelectorAll("[data-trip-settings-map-name]").forEach((nameInput) => {
    const shell = nameInput.closest("main.app-shell");
    const lookupEnabled = shell ? shell.getAttribute("data-location-lookup-enabled") !== "false" : true;
    const wrap = nameInput.closest(".site-settings-default-place-field") || nameInput.parentElement;
    const latInput = wrap?.querySelector("[data-trip-settings-map-lat]");
    const lngInput = wrap?.querySelector("[data-trip-settings-map-lng]");
    if (!wrap || !latInput || !lngInput) {
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
      wrap.appendChild(suggestionsHost);
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

  /** Row that gets `.editing` while an inline edit form is open (must match CSS .editing .item-actions rules). */
  const INLINE_EDIT_ROW_SELECTOR =
    ".timeline-item, .expense-item, .accommodation-item, .accommodation-card-wrap, .vehicle-rental-item, .flight-card, .reminder-checklist-item";

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
    try {
      const form = document.getElementById(formId);
      if (!form) {
        /* Form can be missing after DOM sync; still clear row editing + show view or actions stay hidden (CSS). */
        const viewId = itineraryViewIdForForm(formId);
        if (viewId && viewId.startsWith("itinerary-view-")) {
          const view = document.getElementById(viewId);
          const row = view?.closest(".timeline-item");
          if (row) row.classList.remove("editing");
          if (view) view.classList.remove("hidden");
        } else if (viewId) {
          const view = document.getElementById(viewId);
          const row = view?.closest(INLINE_EDIT_ROW_SELECTOR);
          if (row) row.classList.remove("editing");
          if (view) view.classList.remove("hidden");
        }
        return;
      }
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
      let row = form.closest(INLINE_EDIT_ROW_SELECTOR);
      if (!row) {
        const vid = itineraryViewIdForForm(formId);
        const v = vid ? document.getElementById(vid) : null;
        if (v) row = v.closest(INLINE_EDIT_ROW_SELECTOR);
      }
      if (row) row.classList.remove("editing");
      const viewId = itineraryViewIdForForm(formId);
      const view = viewId ? document.getElementById(viewId) : null;
      if (view) view.classList.remove("hidden");
    } finally {
      /*
       * Opening edit calls removeAttribute("open") on <details.trip-inline-actions-dropdown>.
       * Summary is display:none; the real UI is in .trip-inline-actions-buttons, which the UA
       * hides when <details> is closed — so row actions vanish until page reload unless we
       * re-apply the same open-state sync used on load (desktop: keep details open for CSS hover).
       */
      window.remiApplyExpenseActionsDropdownOpen?.();
    }
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
      const row = form.closest(INLINE_EDIT_ROW_SELECTOR);
      if (row) row.classList.add("editing");
      const viewId = itineraryViewIdForForm(formId);
      const view = viewId ? document.getElementById(viewId) : null;
      if (view) view.classList.add("hidden");
      form.classList.remove("hidden");
      window.remiResyncVehicleDropoffInForm?.(form);
      const dateInput =
        form.querySelector("input.remi-date-iso[name='itinerary_date']") ||
        form.querySelector("input[name='itinerary_date']");
      if (dateInput) dateInput.dataset.originalValue = dateInput.value;
    });
  };
  document.querySelectorAll("[data-inline-edit-open]").forEach((btn) => wireOneInlineEditOpenBtn(btn));
  window.remiWireInlineEditOpenButtonsIn = (root) => {
    const scope = remiIsDomQueryRoot(root) ? root : document;
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
    const itineraryDateIso =
      itineraryForm.querySelector("input.remi-date-iso[name='itinerary_date']") ||
      itineraryForm.querySelector("input[name='itinerary_date']");
    const itineraryDateInput = itineraryDateIso;
    const itineraryDateVisibleEl = () => {
      const iso = itineraryForm.querySelector("input.remi-date-iso[name='itinerary_date']");
      const wrap = iso?.closest?.("[data-remi-date]");
      return wrap?.querySelector?.(".remi-date-visible") || null;
    };
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
    let geocodeSubmitInProgress = false;
    let allowResolvedSubmit = false;

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
      setDateStatus(
        `This date is within your trip (${tripStartLabel} – ${tripEndLabel}).`,
        "success"
      );
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
        setLocationStatus(
          "Location lookup failed. Try a more specific place name, including city/country.",
          "error"
        );
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
      if (allowResolvedSubmit) {
        allowResolvedSubmit = false;
        return;
      }
      if (geocodeSubmitInProgress) return;
      if (!validateDateInTripRange()) {
        event.preventDefault();
        const vis = itineraryDateVisibleEl();
        if (vis instanceof HTMLElement) vis.focus();
        else itineraryDateInput?.focus?.();
        return;
      }
      if (!locationInput) return;
      const query = locationInput.value;
      if (!query || !query.trim()) return;
      event.preventDefault();
      geocodeSubmitInProgress = true;
      const ok = await resolveLocation(query);
      geocodeSubmitInProgress = false;
      if (!ok) {
        locationInput.focus();
        return;
      }
      allowResolvedSubmit = true;
      itineraryForm.requestSubmit();
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
      const isPreset = v === "default" || v.startsWith("pattern:");
      if (isPreset) {
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
    let accommodationLookupSeq = 0;
    let lastAutoFilledAddress = (accommodationAddressInput?.value || "").trim();
    let addressManualLock = lastAutoFilledAddress !== "";

    const setAccommodationStatus = (message, state) => {
      if (!accommodationStatus) return;
      accommodationStatus.textContent = message;
      accommodationStatus.classList.remove("error", "success");
      if (state) accommodationStatus.classList.add(state);
    };

    const lookupAddress = async (query) => {
      const q = (query || "").trim();
      if (!q) return;
      const seq = ++accommodationLookupSeq;
      setAccommodationStatus("Looking up address...");
      const suggestion = await geocodeLocation(q);
      if (seq !== accommodationLookupSeq) return;
      if (suggestion && suggestion.displayName && accommodationAddressInput) {
        if (addressManualLock) {
          setAccommodationStatus(
            "Address was edited manually. Auto-fill skipped.",
            "success"
          );
          return;
        }
        // Keep address synchronized with the latest accommodation lookup result.
        accommodationAddressInput.value = suggestion.displayName;
        lastAutoFilledAddress = suggestion.displayName.trim();
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
    if (accommodationAddressInput) {
      accommodationAddressInput.addEventListener("input", () => {
        const current = accommodationAddressInput.value.trim();
        if (!current) {
          // Empty means the user is okay with auto-fill taking over again.
          addressManualLock = false;
          lastAutoFilledAddress = "";
          return;
        }
        addressManualLock = current !== lastAutoFilledAddress;
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

  const FILE_FIELD_META_BY_NAME = {
    booking_attachment: { removeName: "remove_attachment", currentName: "current_attachment_path", kind: "document" },
    flight_document: { removeName: "remove_document", currentName: "current_document_path", kind: "document" },
    tab_attachment: { removeName: "remove_tab_attachment", currentName: "current_tab_attachment_path", kind: "document" },
    entry_image: { removeName: "remove_image", currentName: "current_image_path", kind: "image" },
    vehicle_image: { removeName: "remove_vehicle_image", currentName: "current_vehicle_image_path", kind: "image" },
    stop_image: { removeName: "remove_stop_image", currentName: "current_stop_image_path", kind: "image" }
  };

  const remiMaxUploadBytes = () => {
    const host = document.querySelector("main.app-shell");
    const raw = host?.getAttribute("data-max-upload-file-size-mb") || "5";
    const mb = parseInt(raw, 10);
    const safeMB = Number.isFinite(mb) && mb > 0 ? mb : 5;
    return safeMB * 1024 * 1024;
  };

  const remiHumanFileSize = (bytes) => {
    const n = Number(bytes);
    if (!Number.isFinite(n) || n < 0) return "Unknown size";
    if (n < 1024) return `${Math.round(n)} B`;
    const kb = n / 1024;
    if (kb < 1024) return `${kb < 10 ? kb.toFixed(1) : Math.round(kb)} KB`;
    const mb = kb / 1024;
    if (mb < 1024) return `${mb < 10 ? mb.toFixed(1) : mb.toFixed(0)} MB`;
    const gb = mb / 1024;
    return `${gb < 10 ? gb.toFixed(1) : gb.toFixed(0)} GB`;
  };

  const remiFileExt = (name) => {
    const s = String(name || "").trim();
    const i = s.lastIndexOf(".");
    if (i <= 0 || i === s.length - 1) return "";
    return s.slice(i + 1).toLowerCase();
  };

  const remiNameFromPath = (path) => {
    const raw = String(path || "").trim();
    if (!raw) return "";
    const clean = raw.split("?")[0].split("#")[0];
    const slash = Math.max(clean.lastIndexOf("/"), clean.lastIndexOf("\\"));
    const part = slash >= 0 ? clean.slice(slash + 1) : clean;
    try {
      return decodeURIComponent(part);
    } catch {
      return part;
    }
  };

  const remiAttachmentIcon = (kind, ext) => {
    const e = String(ext || "").toLowerCase();
    if (kind === "image") return "image";
    if (["pdf"].includes(e)) return "picture_as_pdf";
    if (["doc", "docx", "txt", "rtf", "md"].includes(e)) return "description";
    if (["xls", "xlsx", "csv"].includes(e)) return "table";
    return "attach_file";
  };

  const remiClearFileInput = (input) => {
    if (!(input instanceof HTMLInputElement)) return;
    const dt = new DataTransfer();
    input.files = dt.files;
    input.value = "";
  };

  const remiSetRemoveControl = (control, shouldRemove) => {
    if (!control) return;
    if (control instanceof HTMLSelectElement) {
      control.value = shouldRemove ? "true" : "false";
      return;
    }
    if (control instanceof HTMLInputElement && control.type === "checkbox") {
      control.checked = shouldRemove;
    }
  };

  const remiReadRemoveControl = (control) => {
    if (!control) return false;
    if (control instanceof HTMLSelectElement) return String(control.value || "").toLowerCase() === "true";
    if (control instanceof HTMLInputElement && control.type === "checkbox") return Boolean(control.checked);
    return false;
  };

  const remiMatchesAccept = (file, acceptList) => {
    const accept = String(acceptList || "").trim();
    if (!accept) return true;
    const name = String(file?.name || "").toLowerCase();
    const ext = remiFileExt(name);
    const mime = String(file?.type || "").toLowerCase();
    return accept
      .split(",")
      .map((t) => t.trim().toLowerCase())
      .filter(Boolean)
      .some((rule) => {
        if (rule === "*/*") return true;
        if (rule.endsWith("/*")) {
          const pref = rule.slice(0, -1);
          return mime.startsWith(pref);
        }
        if (rule.startsWith(".")) {
          return `.${ext}` === rule;
        }
        return mime === rule;
      });
  };

  const remiHideLegacyAttachmentHints = (form, existingUrl) => {
    if (!(form instanceof HTMLElement)) return;
    if (!existingUrl) return;
    form.querySelectorAll("p.edit-form-attachment-open-wrap a[href], p.muted a[href]").forEach((a) => {
      const href = String(a.getAttribute("href") || "").trim();
      if (!href || href !== existingUrl) return;
      const p = a.closest("p");
      if (p instanceof HTMLElement) p.hidden = true;
    });
  };

  const initAttachmentEditField = (fileInput) => {
    if (!(fileInput instanceof HTMLInputElement) || fileInput.dataset.remiAttachmentFieldWired === "1") return;
    const meta = FILE_FIELD_META_BY_NAME[fileInput.name];
    if (!meta) return;
    const form = fileInput.closest("form");
    if (!(form instanceof HTMLFormElement) || !form.classList.contains("item-edit")) return;

    const removeControl = form.querySelector(`[name="${meta.removeName}"]`);
    const currentPathInput =
      meta.currentName ? form.querySelector(`input[type="hidden"][name="${meta.currentName}"]`) : null;
    const existingUrl = String(currentPathInput?.value || "").trim();
    const maxBytes = remiMaxUploadBytes();
    const maxSizeMsg = remiHumanFileSize(maxBytes);

    remiHideLegacyAttachmentHints(form, existingUrl);
    fileInput.dataset.remiAttachmentFieldWired = "1";
    fileInput.hidden = true;
    fileInput.setAttribute("aria-hidden", "true");

    const ui = document.createElement("div");
    ui.className = "edit-attachment-ui";
    const card = document.createElement("div");
    card.className = "edit-attachment-card";
    const iconWrap = document.createElement("span");
    iconWrap.className = "edit-attachment-icon";
    const icon = document.createElement("span");
    icon.className = "material-symbols-outlined";
    icon.setAttribute("aria-hidden", "true");
    iconWrap.appendChild(icon);
    const metaWrap = document.createElement("div");
    metaWrap.className = "edit-attachment-meta";
    const nameEl = document.createElement("p");
    nameEl.className = "edit-attachment-name";
    const subEl = document.createElement("p");
    subEl.className = "edit-attachment-sub";
    metaWrap.append(nameEl, subEl);
    const actions = document.createElement("div");
    actions.className = "edit-attachment-actions";
    const replaceBtn = document.createElement("button");
    replaceBtn.type = "button";
    replaceBtn.className = "edit-attachment-btn";
    replaceBtn.innerHTML =
      '<span class="material-symbols-outlined" aria-hidden="true">sync</span><span>Replace File</span>';
    const removeBtn = document.createElement("button");
    removeBtn.type = "button";
    removeBtn.className = "edit-attachment-btn edit-attachment-btn--remove";
    removeBtn.innerHTML =
      '<span class="material-symbols-outlined" aria-hidden="true">delete</span><span>Remove</span>';
    actions.append(replaceBtn, removeBtn);
    card.append(iconWrap, metaWrap, actions);
    ui.appendChild(card);
    fileInput.insertAdjacentElement("afterend", ui);

    const state = {
      pendingFile: null,
      removeMarked: remiReadRemoveControl(removeControl),
      existingSize: "",
      existingName: remiNameFromPath(existingUrl),
      existingExt: remiFileExt(remiNameFromPath(existingUrl))
    };

    const syncRemoveState = () => {
      const shouldRemove = Boolean(state.removeMarked && !state.pendingFile && existingUrl);
      remiSetRemoveControl(removeControl, shouldRemove);
    };

    const render = () => {
      card.classList.remove("is-pending", "is-remove");
      let fileName = "";
      let sub = "";
      let ext = "";

      if (state.pendingFile) {
        card.classList.add("is-pending");
        fileName = state.pendingFile.name;
        ext = remiFileExt(fileName);
        sub = `${ext ? ext.toUpperCase() : "FILE"} · ${remiHumanFileSize(state.pendingFile.size)} · Pending upload`;
      } else if (existingUrl && !state.removeMarked) {
        fileName = state.existingName || "Attached file";
        ext = state.existingExt;
        sub = `${ext ? ext.toUpperCase() : "FILE"} · ${state.existingSize || "Checking size..."}`;
      } else if (existingUrl && state.removeMarked) {
        card.classList.add("is-remove");
        fileName = state.existingName || "Attached file";
        ext = state.existingExt;
        sub = "Marked for removal. Save changes to apply.";
      } else {
        fileName = meta.kind === "image" ? "No image attached" : "No attachment";
        sub = `Max file size: ${maxSizeMsg}`;
      }

      icon.textContent = remiAttachmentIcon(meta.kind, ext);
      nameEl.textContent = fileName;
      subEl.textContent = sub;
      replaceBtn.querySelector("span:last-child").textContent =
        state.pendingFile || existingUrl ? "Replace File" : "Choose File";
      const canRemove = Boolean(state.pendingFile || existingUrl);
      removeBtn.hidden = !canRemove;
      removeBtn.querySelector("span.material-symbols-outlined").textContent = state.removeMarked ? "undo" : "delete";
      removeBtn.querySelector("span:last-child").textContent = state.removeMarked ? "Undo" : "Remove";
      syncRemoveState();
    };

    replaceBtn.addEventListener("click", () => {
      fileInput.click();
    });

    removeBtn.addEventListener("click", () => {
      if (state.pendingFile) {
        state.pendingFile = null;
        remiClearFileInput(fileInput);
        state.removeMarked = Boolean(existingUrl);
      } else if (existingUrl) {
        state.removeMarked = !state.removeMarked;
      }
      render();
    });

    fileInput.addEventListener("change", () => {
      const f = fileInput.files && fileInput.files[0] ? fileInput.files[0] : null;
      if (!f) {
        state.pendingFile = null;
        render();
        return;
      }
      if (typeof window.remiIsBlockedUploadFilename === "function" && window.remiIsBlockedUploadFilename(f.name)) {
        remiClearFileInput(fileInput);
        window.remiShowToast?.(
          window.remiBlockedUploadFilenameMessage ||
            "This file type is not allowed — executables and scripts cannot be uploaded."
        );
        return;
      }
      if (!remiMatchesAccept(f, fileInput.accept || "")) {
        remiClearFileInput(fileInput);
        window.remiShowToast?.("Selected file type is not allowed for this field.");
        return;
      }
      if (maxBytes > 0 && f.size > maxBytes) {
        remiClearFileInput(fileInput);
        window.remiShowToast?.(`File is too large. Max allowed is ${maxSizeMsg}.`);
        return;
      }
      state.pendingFile = f;
      state.removeMarked = false;
      render();
    });

    if (existingUrl) {
      fetch(existingUrl, { method: "HEAD", credentials: "same-origin" })
        .then((res) => {
          const raw = res.headers.get("content-length");
          const n = raw ? parseInt(raw, 10) : NaN;
          if (Number.isFinite(n) && n >= 0) {
            state.existingSize = remiHumanFileSize(n);
            render();
          } else {
            state.existingSize = "Unknown size";
            render();
          }
        })
        .catch(() => {
          state.existingSize = "Unknown size";
          render();
        });
    }

    render();
  };

  const initAttachmentEditFieldsIn = (root) => {
    const scope = remiIsDomQueryRoot(root) ? root : document;
    scope.querySelectorAll("form.item-edit input[type='file'][name]").forEach((input) => {
      initAttachmentEditField(input);
    });
  };
  window.remiInitAttachmentEditFieldsIn = initAttachmentEditFieldsIn;
  initAttachmentEditFieldsIn(document);

  /** After replacing innerHTML inside a card view, reattach listeners for Edit / Delete / AJAX (e.g. spend row). */
  const rewireInjectedActionUI = (root) => {
    if (!remiIsDomQueryRoot(root)) return;
    window.remiEnsureCsrfOnPostForms?.(root);
    root.querySelectorAll("form[data-app-confirm]").forEach((f) => {
      delete f.dataset.remiConfirmWired;
      window.remiWireAppConfirmOnForm?.(f);
    });
    root.querySelectorAll("form[data-ajax-submit]").forEach((f) => {
      delete f.dataset.remiAjaxWired;
      window.remiBindAjaxSubmitForm?.(f);
    });
    window.remiWireInlineEditOpenButtonsIn?.(root);
    window.remiWireDateFieldsIn?.(root);
    window.remiWireDateTimeFieldsIn?.(root);
    window.remiWireTimeFieldsIn?.(root);
    window.remiInitAttachmentEditFieldsIn?.(root);
    window.remiApplyExpenseActionsDropdownOpen?.();
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
        cache: "no-store",
        credentials: "same-origin",
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
      rewireInjectedActionUI(currentView);
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
      cache: "no-store",
      credentials: "same-origin",
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
    window.remiInitAttachmentEditFieldsIn?.(document);
    window.remiWireDateFieldsIn?.(document);
    window.remiWireDateTimeFieldsIn?.(document);
    window.remiWireTimeFieldsIn?.(document);
    window.remiEnsureCsrfOnPostForms?.(document);
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
        "data-marker-kind",
        "data-itinerary-item-id"
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

  // Refresh all rendered itinerary rows from a fresh server document.
  // Used for booking forms (flight/vehicle/accommodation) that update linked itinerary stops.
  const syncAllRenderedItineraryRows = async () => {
    const currentRows = Array.from(
      document.querySelectorAll("li.timeline-item[data-itinerary-item-id]")
    );
    if (!currentRows.length) return false;
    const freshDoc = await fetchFreshDoc();
    if (!freshDoc) return false;
    let hasAnyFreshMatch = false;
    currentRows.forEach((row) => {
      const itemId = (row.getAttribute("data-itinerary-item-id") || "").trim();
      if (!itemId) return;
      const viewId = `itinerary-view-${itemId}`;
      if (!freshDoc.getElementById(viewId)) return;
      hasAnyFreshMatch = true;
      syncItineraryRow(row, freshDoc, viewId);
    });
    if (!hasAnyFreshMatch) return false;
    await renderItineraryConnectors(document);
    window.remiRefreshTripMapFromItineraryDOM?.();
    window.remiWireInlineEditOpenButtonsIn?.(
      document.querySelector("[data-itinerary-search-root]") || document
    );
    window.remiWireDateFieldsIn?.(document.querySelector("[data-itinerary-search-root]") || document);
    window.remiWireDateTimeFieldsIn?.(document.querySelector("[data-itinerary-search-root]") || document);
    window.remiWireTimeFieldsIn?.(document.querySelector("[data-itinerary-search-root]") || document);
    return true;
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

    await renderItineraryConnectors(document);
    if (typeof window.remiRefreshTripMapFromItineraryDOM === "function") {
      window.remiRefreshTripMapFromItineraryDOM();
    }
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
      rewireInjectedActionUI(currentView);
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
        cache: "no-store",
        credentials: "same-origin",
        headers: { "X-Requested-With": "XMLHttpRequest", Accept: "text/html" }
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
  let pendingConfirmCallback = null;
  const confirmDialogEl = document.getElementById("dialog-confirm-action");
  const confirmDialogUsable =
    confirmDialogEl &&
    typeof confirmDialogEl.showModal === "function" &&
    typeof confirmDialogEl.close === "function";
  if (confirmDialogUsable) {
    const confirmTitleEl = document.getElementById("dialog-confirm-action-title");
    const confirmBodyEl = document.getElementById("dialog-confirm-action-desc");
    const confirmIconEl = document.getElementById("dialog-confirm-action-icon");
    const confirmOkBtn = document.getElementById("dialog-confirm-action-ok");
    const confirmCancelBtn = confirmDialogEl.querySelector("[data-confirm-cancel]");

    const populateConfirmDialog = (opts) => {
      const title = opts.title || "Confirm?";
      const body = (opts.body || "").trim();
      const ok = opts.ok || "Confirm";
      const icon = opts.icon || "help_outline";
      const variant = (opts.variant || "danger").toLowerCase();
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

    const populateConfirmFromForm = (form) => {
      populateConfirmDialog({
        title: form.getAttribute("data-confirm-title") || "Confirm?",
        body: (form.getAttribute("data-confirm-body") || "").trim(),
        ok: form.getAttribute("data-confirm-ok") || "Confirm",
        icon: form.getAttribute("data-confirm-icon") || "help_outline",
        variant: (form.getAttribute("data-confirm-variant") || "danger").toLowerCase()
      });
    };

    const closeConfirmClearPending = () => {
      pendingConfirmSubmitForm = null;
      pendingConfirmCallback = null;
      try {
        confirmDialogEl.close();
      } catch (e) {
        /* ignore */
      }
    };

    confirmOkBtn?.addEventListener("click", () => {
      const cb = pendingConfirmCallback;
      const f = pendingConfirmSubmitForm;
      pendingConfirmCallback = null;
      pendingConfirmSubmitForm = null;
      try {
        confirmDialogEl.close();
      } catch (e) {
        /* ignore */
      }
      if (cb) {
        cb();
        return;
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

    window.remiOpenAppConfirm = (opts) => {
      if (!opts || typeof opts.onConfirm !== "function") return;
      pendingConfirmSubmitForm = null;
      pendingConfirmCallback = opts.onConfirm;
      populateConfirmDialog({
        title: opts.title || "Confirm?",
        body: (opts.body || "").trim(),
        ok: opts.okText || opts.ok || "Confirm",
        icon: opts.icon || "help_outline",
        variant: (opts.variant || "danger").toLowerCase()
      });
      confirmDialogEl.showModal();
    };
  }

  const handleAjaxFormSubmit = async (event) => {
    const form = event.currentTarget;
    if (!(form instanceof HTMLFormElement)) return;
      if (event.defaultPrevented) return;
      event.preventDefault();
      const method = (form.getAttribute("method") || "post").toUpperCase();
      const formData = new FormData(form);
      mergeFormAssociatedControls(form, formData);
      const fileInputs = form.querySelectorAll("input[type='file']");
      if (fileInputs.length && typeof window.remiIsBlockedUploadFilename === "function") {
        for (let j = 0; j < fileInputs.length; j++) {
          const inp = fileInputs[j];
          if (!(inp instanceof HTMLInputElement) || !inp.files) continue;
          for (let i = 0; i < inp.files.length; i++) {
            if (window.remiIsBlockedUploadFilename(inp.files[i].name)) {
              showToast(
                window.remiBlockedUploadFilenameMessage ||
                  "This file type is not allowed — executables and scripts cannot be uploaded."
              );
              return;
            }
          }
        }
      }
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
          const itineraryRow = form.closest(".timeline-item");
          const moved = await smartRepositionItineraryItem(form);
          if (!moved) {
            try { sessionStorage.setItem(TOAST_KEY, inferToastMessage(form)); } catch (e) { /* ignore */ }
            window.location.reload();
            return;
          }
          if (itineraryRow) itineraryRow.classList.remove("editing");
          closeInlineEdit(form.id);
          window.remiApplyExpenseActionsDropdownOpen?.();
          window.remiWireInlineEditOpenButtonsIn?.(itineraryRow || document);
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
          if (/\/trips\/[^/]+\/(accommodation|vehicle-rental|flights)\/[^/]+\/update$/i.test(form.action || "")) {
            await syncAllRenderedItineraryRows();
          }
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
              if (row.classList.contains("timeline-item")) {
                void renderItineraryConnectors(document);
                window.remiRefreshTripMapFromItineraryDOM?.();
              }
            }
          }
        }
        if (document.querySelector(".trip-details-page .budget-tile")) {
          await refreshBudgetTilesFromPage();
        }
        if (form.hasAttribute("data-ajax-reload-on-success")) {
          try {
            sessionStorage.setItem(TOAST_KEY, inferToastMessage(form));
          } catch (e) {
            /* ignore */
          }
          window.location.reload();
          return;
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
      window.remiEnsureCsrfOnPostForms?.(destTbody || document);
      window.setupTabSplitRootsIn?.(destTbody || document);
      window.rewireTabInlineEditOpenButtons?.();
      window.remiWireDateFieldsIn?.(destTbody || document);
      window.remiWireDateTimeFieldsIn?.(destTbody || document);
      window.remiWireTimeFieldsIn?.(destTbody || document);
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
      window.remiEnsureCsrfOnPostForms?.(ul);
      window.rewireTabInlineEditOpenButtons?.();
      window.remiWireDateFieldsIn?.(ul);
      window.remiWireDateTimeFieldsIn?.(ul);
      window.remiWireTimeFieldsIn?.(ul);
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
    window.remiWireDateFieldsIn?.(target);
    window.remiWireDateTimeFieldsIn?.(target);
    window.remiWireTimeFieldsIn?.(target);
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
  const useGoogleMaps = /^AIza[0-9A-Za-z_-]{20,}$/.test(gMapKey);
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

  const parseTripMapPoints = () =>
    Array.from(document.querySelectorAll("[data-map-itinerary-point][data-lat][data-lng]"))
      .map((el) => ({
        itemId: (el.getAttribute("data-itinerary-item-id") || "").trim(),
        lat: parseFloat(el.getAttribute("data-lat") || "0"),
        lng: parseFloat(el.getAttribute("data-lng") || "0"),
        title: el.getAttribute("data-title") || "",
        location: el.getAttribute("data-location") || "",
        day: parseInt(el.getAttribute("data-map-day") || "1", 10) || 1,
        kind: (el.getAttribute("data-marker-kind") || "stop").toLowerCase()
      }))
      .filter((p) => !Number.isNaN(p.lat) && !Number.isNaN(p.lng) && (p.lat !== 0 || p.lng !== 0));

  const points = parseTripMapPoints();

  const readTripMapLegendSelectedDays = (fallbackDayList) => {
    const leg = document.querySelector(".trip-map-day-legend");
    if (!leg) {
      return new Set(fallbackDayList);
    }
    const out = new Set();
    leg.querySelectorAll(".trip-map-day-legend-item").forEach((btn) => {
      if (btn.classList.contains("is-off")) return;
      const d = parseInt(btn.getAttribute("data-day") || "0", 10);
      if (d) out.add(d);
    });
    return out.size > 0 ? out : new Set(fallbackDayList);
  };

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

  function remiMapHexToRgb(hex) {
    const h = String(hex || "").replace(/^#/, "");
    if (h.length === 3) {
      return {
        r: parseInt(h[0] + h[0], 16),
        g: parseInt(h[1] + h[1], 16),
        b: parseInt(h[2] + h[2], 16)
      };
    }
    if (h.length !== 6) return { r: 37, g: 99, b: 235 };
    return {
      r: parseInt(h.slice(0, 2), 16),
      g: parseInt(h.slice(2, 4), 16),
      b: parseInt(h.slice(4, 6), 16)
    };
  }
  function remiMapRgbToHex(r, g, b) {
    const c = (n) => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, "0");
    return `#${c(r)}${c(g)}${c(b)}`;
  }
  function remiMapMixRgb(ringRgb, baseRgb, t) {
    return {
      r: ringRgb.r * t + baseRgb.r * (1 - t),
      g: ringRgb.g * t + baseRgb.g * (1 - t),
      b: ringRgb.b * t + baseRgb.b * (1 - t)
    };
  }
  /** Canvas marker matching Leaflet remi-map-marker (day ring + kind emoji). */
  function remiGoogleMapMarkerBitmap(ringHex, kind, isDark) {
    const size = 34;
    const dpr = typeof window.devicePixelRatio === "number" && window.devicePixelRatio >= 2 ? 2 : 1;
    const canvas = document.createElement("canvas");
    canvas.width = Math.round(size * dpr);
    canvas.height = Math.round(size * dpr);
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      return { url: "", size, anchorX: size / 2, anchorY: size };
    }
    ctx.scale(dpr, dpr);
    const ringRgb = remiMapHexToRgb(ringHex);
    const baseRgb = isDark ? { r: 30, g: 41, b: 59 } : { r: 255, g: 255, b: 255 };
    const mixT = isDark ? 0.28 : 0.22;
    const fillRgb = remiMapMixRgb(ringRgb, baseRgb, mixT);
    const cx = size / 2;
    const cy = size / 2;
    const radius = 13;
    ctx.beginPath();
    ctx.arc(cx, cy, radius, 0, Math.PI * 2);
    ctx.fillStyle = remiMapRgbToHex(fillRgb.r, fillRgb.g, fillRgb.b);
    ctx.fill();
    ctx.strokeStyle = ringHex;
    ctx.lineWidth = 2;
    ctx.stroke();
    const k = String(kind || "stop").toLowerCase();
    const emoji =
      k === "stay"
        ? "\u{1F3E8}"
        : k === "vehicle"
          ? "\u{1F697}"
          : k === "flight"
            ? "\u2708\uFE0F"
            : "\u{1F4CD}";
    const fontPx = Math.floor(size * 0.47);
    ctx.font = `${fontPx}px "Segoe UI Emoji","Apple Color Emoji","Noto Color Emoji",sans-serif`;
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(emoji, cx, cy + 0.5);
    return {
      url: canvas.toDataURL("image/png"),
      size,
      anchorX: size / 2,
      anchorY: size
    };
  }

  /**
   * Day chips under the map (toggle visibility + refit bounds). Matches .trip-map-day-legend in app.css.
   * @param {HTMLElement} mapHost
   * @param {number[]} uniqueDaysSorted
   * @param {(day: number) => string} ringForDayFn
   * @param {(selectedDays: Set<number>) => void} applySelection
   */
  function setupTripMapDayLegend(mapHost, uniqueDaysSorted, ringForDayFn, applySelection) {
    const selectedDays = new Set(uniqueDaysSorted);
    const legendButtons = new Map();
    const syncLegendButtons = () => {
      legendButtons.forEach((btn, d) => {
        const on = selectedDays.has(d);
        btn.classList.toggle("is-off", !on);
        btn.setAttribute("aria-pressed", on ? "true" : "false");
      });
    };
    if (uniqueDaysSorted.length > 0) {
      const leg = document.createElement("div");
      leg.className = "trip-map-day-legend";
      leg.setAttribute("aria-label", "Show or hide markers by itinerary day");
      uniqueDaysSorted.forEach((d) => {
        const ring = ringForDayFn(d);
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
          applySelection(selectedDays);
        });
        legendButtons.set(d, btn);
        leg.appendChild(btn);
      });
      mapHost.insertAdjacentElement("afterend", leg);
    }
    syncLegendButtons();
    applySelection(selectedDays);
  }

  let mapNoticeEl = null;
  let mapProviderFallbackActive = false;
  const setTripMapNotice = (message, mode = "info") => {
    if (!mapEl) return;
    if (!mapNoticeEl) {
      mapNoticeEl = document.createElement("p");
      mapNoticeEl.className = "trip-map-runtime-notice hidden";
      mapEl.insertAdjacentElement("afterend", mapNoticeEl);
    }
    const txt = String(message || "").trim();
    if (!txt) {
      mapNoticeEl.classList.add("hidden");
      mapNoticeEl.textContent = "";
      mapNoticeEl.classList.remove("error", "success");
      return;
    }
    mapNoticeEl.classList.remove("hidden");
    mapNoticeEl.classList.remove("error", "success");
    if (mode === "error" || mode === "success") {
      mapNoticeEl.classList.add(mode);
    }
    mapNoticeEl.textContent = txt;
  };

  const renderOSMEmbedFallback = (message, mode = "info") => {
    if (!mapEl || mapProviderFallbackActive) return;
    mapProviderFallbackActive = true;
    const centerLat = Number.isFinite(startLat) ? startLat : 35.6762;
    const centerLng = Number.isFinite(startLng) ? startLng : 139.6503;
    const spanLng = 0.18;
    const spanLat = 0.12;
    const west = centerLng - spanLng;
    const east = centerLng + spanLng;
    const south = centerLat - spanLat;
    const north = centerLat + spanLat;
    const src =
      "https://www.openstreetmap.org/export/embed.html?bbox=" +
      `${encodeURIComponent(west)},${encodeURIComponent(south)},${encodeURIComponent(east)},${encodeURIComponent(north)}` +
      `&layer=mapnik&marker=${encodeURIComponent(centerLat)},${encodeURIComponent(centerLng)}`;
    mapEl.innerHTML = `<iframe class="trip-map-osm-embed" src="${src}" loading="lazy" referrerpolicy="strict-origin-when-cross-origin" title="Trip map fallback"></iframe>`;
    setTripMapNotice(message, mode);
  };

  let leafletLoadPromise = null;
  const ensureLeafletLoaded = async () => {
    if (typeof L !== "undefined") return true;
    if (leafletLoadPromise) return leafletLoadPromise;
    leafletLoadPromise = new Promise((resolve) => {
      const finish = (ok) => resolve(Boolean(ok && typeof L !== "undefined"));
      const candidates = [
        {
          css: "https://unpkg.com/leaflet@1.9.4/dist/leaflet.css",
          js: "https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"
        },
        {
          css: "https://cdn.jsdelivr.net/npm/leaflet@1.9.4/dist/leaflet.css",
          js: "https://cdn.jsdelivr.net/npm/leaflet@1.9.4/dist/leaflet.js"
        }
      ];
      let idx = 0;
      const tryNext = () => {
        if (typeof L !== "undefined") {
          finish(true);
          return;
        }
        if (idx >= candidates.length) {
          finish(false);
          return;
        }
        const c = candidates[idx++];
        if (!document.querySelector(`link[href="${c.css}"]`)) {
          const link = document.createElement("link");
          link.rel = "stylesheet";
          link.href = c.css;
          document.head.appendChild(link);
        }
        const existing = document.querySelector(`script[src="${c.js}"]`);
        if (existing) {
          window.setTimeout(() => {
            if (typeof L !== "undefined") finish(true);
            else tryNext();
          }, 1400);
          return;
        }
        const script = document.createElement("script");
        script.async = true;
        script.src = c.js;
        script.onload = () => {
          if (typeof L !== "undefined") finish(true);
          else tryNext();
        };
        script.onerror = () => {
          tryNext();
        };
        document.head.appendChild(script);
      };
      tryNext();
      window.setTimeout(() => {
        if (typeof L !== "undefined") finish(true);
        else finish(false);
      }, 8000);
    });
    return leafletLoadPromise;
  };

  const initLeafletTripMap = () => {
    if (typeof L === "undefined") return false;
    const map = L.map("map").setView([startLat, startLng], startZoom);
    const lightLayer = L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      maxZoom: 19,
      attribution: "&copy; OpenStreetMap contributors"
    });
    const darkLayer = L.tileLayer("https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png", {
      maxZoom: 19,
      attribution: "&copy; OpenStreetMap contributors &copy; CARTO"
    });
    /* Keep both basemaps mounted; flip opacity + z-index so theme changes are instant (no blank gap while swapping layers). */
    lightLayer.addTo(map);
    darkLayer.addTo(map);
    const setMapTheme = (dark) => {
      if (dark) {
        lightLayer.setOpacity(0);
        darkLayer.setOpacity(1);
        lightLayer.setZIndex(100);
        darkLayer.setZIndex(200);
      } else {
        darkLayer.setOpacity(0);
        lightLayer.setOpacity(1);
        darkLayer.setZIndex(100);
        lightLayer.setZIndex(200);
      }
      map.invalidateSize(false);
    };
    setMapTheme(document.documentElement.classList.contains("theme-dark"));
    document.addEventListener("remi:themechange", (event) => {
      const dark = Boolean(event?.detail?.dark);
      setMapTheme(dark);
    });

    const markersByDay = new Map();
    const leafletMarkerByItemId = new Map();
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
      if (!markersByDay.has(day)) markersByDay.set(day, []);
      markersByDay.get(day).push(marker);
      if (p.itemId) {
        leafletMarkerByItemId.set(p.itemId, marker);
      }
    });

    const applyLeafletDaySelection = (selectedDays) => {
      const vis = [];
      selectedDays.forEach((d) => {
        (markersByDay.get(d) || []).forEach((m) => {
          vis.push(m);
          if (!map.hasLayer(m)) m.addTo(map);
        });
      });
      markersByDay.forEach((arr, d) => {
        if (selectedDays.has(d)) return;
        arr.forEach((m) => {
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

    const refreshLeafletTripMapFromDOM = () => {
      if (!map) return;
      const fresh = parseTripMapPoints();
      const seen = new Set();
      fresh.forEach((p) => {
        if (!p.itemId) return;
        seen.add(p.itemId);
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
        let m = leafletMarkerByItemId.get(p.itemId);
        if (!m) {
          m = L.marker([p.lat, p.lng], { icon });
          m.bindPopup(
            `<b>${escapeHtmlMap(p.title)}</b><br><span class="trip-map-popup-day">Day ${day}</span><br>${escapeHtmlMap(p.location)}`
          );
          leafletMarkerByItemId.set(p.itemId, m);
        } else {
          m.setLatLng([p.lat, p.lng]);
          m.setIcon(icon);
          m.setPopupContent(
            `<b>${escapeHtmlMap(p.title)}</b><br><span class="trip-map-popup-day">Day ${day}</span><br>${escapeHtmlMap(p.location)}`
          );
        }
      });
      leafletMarkerByItemId.forEach((m, id) => {
        if (seen.has(id)) return;
        if (map.hasLayer(m)) map.removeLayer(m);
        leafletMarkerByItemId.delete(id);
      });
      markersByDay.clear();
      const dayNums = new Set(uniqueDays);
      fresh.forEach((fp) => dayNums.add(Math.max(1, fp.day)));
      dayNums.forEach((d) => markersByDay.set(d, []));
      leafletMarkerByItemId.forEach((m, id) => {
        const pt = fresh.find((x) => x.itemId === id);
        const d = pt ? Math.max(1, pt.day) : 1;
        markersByDay.get(d).push(m);
      });
      applyLeafletDaySelection(readTripMapLegendSelectedDays(uniqueDays));
      map.invalidateSize(false);
    };

    window.remiRefreshTripMapFromItineraryDOM = refreshLeafletTripMapFromDOM;
    setupTripMapDayLegend(mapEl, uniqueDays, ringForDay, applyLeafletDaySelection);
    setTripMapNotice("");
    window.setTimeout(() => {
      if (mapProviderFallbackActive) return;
      const pane = mapEl.querySelector(".leaflet-pane");
      if (!pane) {
        renderOSMEmbedFallback(
          "Leaflet map assets failed to initialize. Showing OpenStreetMap fallback.",
          "error"
        );
        return;
      }
      const panePos = window.getComputedStyle(pane).position;
      if (panePos !== "absolute") {
        renderOSMEmbedFallback(
          "Leaflet styles are unavailable. Showing OpenStreetMap fallback.",
          "error"
        );
      }
    }, 1200);
    return true;
  };

  if (gMapKey && !useGoogleMaps) {
    setTripMapNotice(
      "Google Maps API key is set but does not look like a valid browser key; using OpenStreetMap (Leaflet).",
      "info"
    );
  }

  if (useGoogleMaps) {
    /* Dark basemap via styled maps. Google’s colorScheme only applies at Map construction;
       setOptions({ colorScheme }) after create is ignored (Edge/Chrome), so theme toggles must use styles. */
    const remiGmapDarkStyles = [
      { elementType: "geometry", stylers: [{ color: "#242f3e" }] },
      { elementType: "labels.text.stroke", stylers: [{ color: "#242f3e" }] },
      { elementType: "labels.text.fill", stylers: [{ color: "#746855" }] },
      { featureType: "administrative.locality", elementType: "labels.text.fill", stylers: [{ color: "#d59563" }] },
      { featureType: "poi", elementType: "labels.text.fill", stylers: [{ color: "#d59563" }] },
      { featureType: "poi.park", elementType: "geometry", stylers: [{ color: "#263c3f" }] },
      { featureType: "road", elementType: "geometry", stylers: [{ color: "#38414e" }] },
      { featureType: "road", elementType: "geometry.stroke", stylers: [{ color: "#212a37" }] },
      { featureType: "road.highway", elementType: "geometry", stylers: [{ color: "#746855" }] },
      { featureType: "transit", elementType: "geometry", stylers: [{ color: "#2f3948" }] },
      { featureType: "water", elementType: "geometry", stylers: [{ color: "#17263c" }] }
    ];

    let remiGoogleTripMap = null;
    const remiGoogleTripMarkerEntries = [];

    const googleTripMapDarkNow = () => document.documentElement.classList.contains("theme-dark");

    const syncGoogleTripMapTheme = (darkOverride) => {
      if (!remiGoogleTripMap || !window.google || !google.maps) return;
      const dark =
        typeof darkOverride === "boolean" ? darkOverride : googleTripMapDarkNow();
      remiGoogleTripMap.setOptions({
        styles: dark ? remiGmapDarkStyles : []
      });
      remiGoogleTripMarkerEntries.forEach(({ marker, point }) => {
        const ring = ringForDay(Math.max(1, point.day));
        const bmp = remiGoogleMapMarkerBitmap(ring, point.kind, dark);
        if (!bmp.url) return;
        marker.setIcon({
          url: bmp.url,
          scaledSize: new google.maps.Size(bmp.size, bmp.size),
          anchor: new google.maps.Point(bmp.anchorX, bmp.anchorY)
        });
      });
      if (google.maps.event && typeof google.maps.event.trigger === "function") {
        google.maps.event.trigger(remiGoogleTripMap, "resize");
        requestAnimationFrame(() => {
          if (remiGoogleTripMap) {
            google.maps.event.trigger(remiGoogleTripMap, "resize");
          }
        });
      }
    };

    document.addEventListener("remi:themechange", (ev) => {
      const d =
        ev && ev.detail && typeof ev.detail.dark === "boolean"
          ? ev.detail.dark
          : undefined;
      syncGoogleTripMapTheme(d);
    });

    const initGoogleTripMap = () => {
      if (!window.google || !google.maps) return;
      const dark = googleTripMapDarkNow();
      const mapOpts = {
        center: { lat: startLat, lng: startLng },
        zoom: startZoom,
        mapTypeControl: true,
        streetViewControl: true,
        fullscreenControl: true,
        styles: dark ? remiGmapDarkStyles : []
      };
      remiGoogleTripMap = new google.maps.Map(mapEl, mapOpts);
      const gMap = remiGoogleTripMap;
      remiGoogleTripMarkerEntries.length = 0;
      const googleMarkersByDay = new Map();
      const googleMarkerByItemId = new Map();
      uniqueDays.forEach((d) => googleMarkersByDay.set(d, []));
      const darkMarkers = googleTripMapDarkNow();
      points.forEach((p) => {
        const ring = ringForDay(Math.max(1, p.day));
        const bmp = remiGoogleMapMarkerBitmap(ring, p.kind, darkMarkers);
        const iconOpts =
          bmp.url && typeof google.maps.Size === "function" && typeof google.maps.Point === "function"
            ? {
                url: bmp.url,
                scaledSize: new google.maps.Size(bmp.size, bmp.size),
                anchor: new google.maps.Point(bmp.anchorX, bmp.anchorY)
              }
            : undefined;
        const marker = new google.maps.Marker({
          position: { lat: p.lat, lng: p.lng },
          map: null,
          title: `${p.title} · Day ${p.day}`,
          ...(iconOpts ? { icon: iconOpts } : {})
        });
        remiGoogleTripMarkerEntries.push({ marker, point: p });
        const dNum = Math.max(1, p.day);
        if (!googleMarkersByDay.has(dNum)) googleMarkersByDay.set(dNum, []);
        googleMarkersByDay.get(dNum).push(marker);
        const iw = new google.maps.InfoWindow({
          content: `<div><b>${escapeHtmlMap(p.title)}</b><br><span class="trip-map-popup-day">Day ${p.day}</span><br>${escapeHtmlMap(p.location)}</div>`
        });
        marker.addListener("click", () => iw.open({ anchor: marker, map: gMap }));
        if (p.itemId) {
          googleMarkerByItemId.set(p.itemId, { marker, iw, point: p });
        }
      });

      const applyGoogleDaySelection = (selectedDays) => {
        const vis = [];
        selectedDays.forEach((d) => {
          (googleMarkersByDay.get(d) || []).forEach((m) => {
            vis.push(m);
            m.setMap(gMap);
          });
        });
        googleMarkersByDay.forEach((markers, d) => {
          if (selectedDays.has(d)) return;
          markers.forEach((m) => m.setMap(null));
        });
        if (vis.length === 0) {
          gMap.setCenter({ lat: startLat, lng: startLng });
          gMap.setZoom(startZoom);
          return;
        }
        if (vis.length === 1) {
          const pos = vis[0].getPosition();
          if (pos) {
            gMap.setCenter(pos);
            gMap.setZoom(Math.max(startZoom, 12));
          }
          return;
        }
        const bounds = new google.maps.LatLngBounds();
        vis.forEach((m) => {
          const pos = m.getPosition();
          if (pos) bounds.extend(pos);
        });
        gMap.fitBounds(bounds, 48);
      };

      const refreshGoogleTripMapFromDOM = () => {
        if (!remiGoogleTripMap || !window.google?.maps) return;
        const fresh = parseTripMapPoints();
        const seen = new Set();
        const darkM = googleTripMapDarkNow();
        fresh.forEach((p) => {
          if (!p.itemId) return;
          seen.add(p.itemId);
          let ent = googleMarkerByItemId.get(p.itemId);
          if (!ent) {
            const ring = ringForDay(Math.max(1, p.day));
            const bmp = remiGoogleMapMarkerBitmap(ring, p.kind, darkM);
            const iconOpts =
              bmp.url && typeof google.maps.Size === "function" && typeof google.maps.Point === "function"
                ? {
                    url: bmp.url,
                    scaledSize: new google.maps.Size(bmp.size, bmp.size),
                    anchor: new google.maps.Point(bmp.anchorX, bmp.anchorY)
                  }
                : undefined;
            const marker = new google.maps.Marker({
              position: { lat: p.lat, lng: p.lng },
              map: null,
              title: `${p.title} · Day ${p.day}`,
              ...(iconOpts ? { icon: iconOpts } : {})
            });
            const pointRef = { ...p };
            const iw = new google.maps.InfoWindow({
              content: `<div><b>${escapeHtmlMap(p.title)}</b><br><span class="trip-map-popup-day">Day ${p.day}</span><br>${escapeHtmlMap(p.location)}</div>`
            });
            marker.addListener("click", () => iw.open({ anchor: marker, map: gMap }));
            ent = { marker, iw, point: pointRef };
            googleMarkerByItemId.set(p.itemId, ent);
          } else {
            Object.assign(ent.point, p);
            ent.marker.setPosition({ lat: p.lat, lng: p.lng });
            ent.marker.setTitle(`${p.title} · Day ${p.day}`);
            const ring = ringForDay(Math.max(1, p.day));
            const bmp = remiGoogleMapMarkerBitmap(ring, p.kind, darkM);
            if (bmp.url && typeof google.maps.Size === "function" && typeof google.maps.Point === "function") {
              ent.marker.setIcon({
                url: bmp.url,
                scaledSize: new google.maps.Size(bmp.size, bmp.size),
                anchor: new google.maps.Point(bmp.anchorX, bmp.anchorY)
              });
            }
            ent.iw.setContent(
              `<div><b>${escapeHtmlMap(p.title)}</b><br><span class="trip-map-popup-day">Day ${p.day}</span><br>${escapeHtmlMap(p.location)}</div>`
            );
          }
        });
        googleMarkerByItemId.forEach((ent, id) => {
          if (seen.has(id)) return;
          ent.marker.setMap(null);
          googleMarkerByItemId.delete(id);
        });
        googleMarkersByDay.clear();
        const dayNums = new Set(uniqueDays);
        fresh.forEach((fp) => dayNums.add(Math.max(1, fp.day)));
        dayNums.forEach((d) => googleMarkersByDay.set(d, []));
        googleMarkerByItemId.forEach((ent) => {
          const d = Math.max(1, ent.point.day);
          if (!googleMarkersByDay.has(d)) googleMarkersByDay.set(d, []);
          googleMarkersByDay.get(d).push(ent.marker);
        });
        remiGoogleTripMarkerEntries.length = 0;
        googleMarkerByItemId.forEach((ent) => {
          remiGoogleTripMarkerEntries.push({ marker: ent.marker, point: ent.point });
        });
        applyGoogleDaySelection(readTripMapLegendSelectedDays(uniqueDays));
        syncGoogleTripMapTheme();
      };

      window.remiRefreshTripMapFromItineraryDOM = refreshGoogleTripMapFromDOM;

      setupTripMapDayLegend(mapEl, uniqueDays, ringForDay, applyGoogleDaySelection);
      syncGoogleTripMapTheme();
    };
    let googleInitDone = false;
    let fallbackTried = false;
    const fallbackToLeaflet = async (reason) => {
      if (fallbackTried) return;
      fallbackTried = true;
      if (reason) {
        setTripMapNotice("Google map unavailable. Showing OpenStreetMap fallback.");
      }
      if (initLeafletTripMap()) return;
      const loaded = await ensureLeafletLoaded();
      if (loaded && initLeafletTripMap()) return;
      renderOSMEmbedFallback(
        "Map libraries could not load. Showing basic OpenStreetMap fallback.",
        "error"
      );
    };

    if (window.google && google.maps) {
      try {
        initGoogleTripMap();
        googleInitDone = true;
      } catch (e) {
        fallbackToLeaflet("google-init-error");
      }
    } else {
      window.remiGoogleTripMapInit = () => {
        if (fallbackTried) return;
        googleInitDone = true;
        try {
          initGoogleTripMap();
        } catch (e) {
          fallbackToLeaflet("google-init-error");
        }
        try {
          delete window.remiGoogleTripMapInit;
        } catch (e) {
          window.remiGoogleTripMapInit = undefined;
        }
      };
      const gs = document.createElement("script");
      gs.async = true;
      gs.onerror = () => {
        fallbackToLeaflet("google-script-error");
      };
      gs.src = `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(
        gMapKey
      )}&callback=remiGoogleTripMapInit&v=weekly`;
      document.head.appendChild(gs);
      window.setTimeout(() => {
        if (!googleInitDone) {
          fallbackToLeaflet("google-timeout");
        }
      }, 9000);
    }
  } else {
    if (!initLeafletTripMap()) {
      ensureLeafletLoaded().then((loaded) => {
        if (loaded && initLeafletTripMap()) return;
        renderOSMEmbedFallback(
          "Map libraries could not load. Showing basic OpenStreetMap fallback.",
          "error"
        );
      });
    }
  }
  }

});

(function () {
  const mq = window.matchMedia("(min-width: 681px)");
  const mqTripDocDesktop = window.matchMedia("(min-width: 981px)");
  const applyExpenseActionsDropdownOpen = () => {
    document.querySelectorAll("main[data-long-press-sheet-root] details.trip-inline-actions-dropdown").forEach((el) => {
      if (el.classList.contains("trip-documents-item-actions")) {
        if (mqTripDocDesktop.matches) {
          el.setAttribute("open", "");
        } else {
          el.removeAttribute("open");
        }
        return;
      }
      if (mq.matches) {
        el.setAttribute("open", "");
      } else {
        el.removeAttribute("open");
      }
    });
  };
  window.remiApplyExpenseActionsDropdownOpen = applyExpenseActionsDropdownOpen;
  const init = () => {
    applyExpenseActionsDropdownOpen();
    mq.addEventListener("change", applyExpenseActionsDropdownOpen);
    mqTripDocDesktop.addEventListener("change", applyExpenseActionsDropdownOpen);
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
    ".expense-item, .timeline-item, .reminder-checklist-item, .flight-card, .title-row, .budget-mobile-tx-item, .accommodation-card-wrap, .vehicle-rental-item, .trip-documents-item";

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
    if (el.closest("a, input, textarea, select, label") && !el.closest("a.trip-documents-file-link")) {
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
    if (row.matches(".trip-documents-item")) {
      return (
        row.querySelector(".trip-documents-file-link")?.textContent?.trim() ||
        row.querySelector(".trip-documents-item-titleline")?.textContent?.trim() ||
        "Attachment"
      );
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

(function initMobileSettingsAccordion() {
  const isSiteSettings = document.body.classList.contains("page-site-settings");
  const isTripSettings = document.body.classList.contains("page-trip-settings");
  if (!isSiteSettings && !isTripSettings) {
    return;
  }

  const mqMobile = window.matchMedia("(max-width: 920px)");
  const sectionSelector = isSiteSettings
    ? ".site-settings-page .trip-settings-subcard"
    : ".trip-settings-page .trip-settings-subcard";
  const cards = Array.from(document.querySelectorAll(sectionSelector)).filter((el) => el instanceof HTMLElement);
  if (!cards.length) {
    return;
  }

  /** @type {{ card: HTMLElement, trigger: HTMLElement, panel: HTMLElement }[]} */
  const sections = [];

  const makeCardAccordionReady = (card, idx) => {
    const head = card.querySelector(":scope > .trip-settings-card-head");
    const fallbackTitle = card.querySelector(":scope > h3");
    const trigger = head instanceof HTMLElement ? head : fallbackTitle instanceof HTMLElement ? fallbackTitle : null;
    if (!trigger) {
      return null;
    }

    const panel = document.createElement("div");
    panel.className = "settings-mobile-accordion-panel";
    panel.id = `settings-mobile-accordion-panel-${idx}`;

    const nodesToMove = Array.from(card.children).filter((node) => node !== trigger);
    if (!nodesToMove.length) {
      return null;
    }
    nodesToMove.forEach((node) => panel.appendChild(node));
    card.appendChild(panel);

    card.setAttribute("data-mobile-settings-accordion", "1");
    trigger.classList.add("settings-mobile-accordion-trigger");
    trigger.setAttribute("role", "button");
    trigger.setAttribute("tabindex", "0");
    trigger.setAttribute("aria-controls", panel.id);
    trigger.setAttribute("aria-expanded", "false");
    panel.hidden = true;

    return { card, trigger, panel };
  };

  cards.forEach((card, idx) => {
    const ready = makeCardAccordionReady(card, idx + 1);
    if (ready) {
      sections.push(ready);
    }
  });

  if (!sections.length) {
    return;
  }

  const setOpen = (targetIdx) => {
    sections.forEach((section, idx) => {
      const isOpen = idx === targetIdx;
      section.panel.hidden = !isOpen;
      section.card.classList.toggle("settings-mobile-accordion-open", isOpen);
      section.trigger.setAttribute("aria-expanded", isOpen ? "true" : "false");
    });
  };

  const collapseAll = () => {
    sections.forEach((section) => {
      section.panel.hidden = true;
      section.card.classList.remove("settings-mobile-accordion-open");
      section.trigger.setAttribute("aria-expanded", "false");
    });
  };

  const expandAllDesktop = () => {
    sections.forEach((section) => {
      section.panel.hidden = false;
      section.card.classList.remove("settings-mobile-accordion-open");
      section.trigger.setAttribute("aria-expanded", "true");
    });
  };

  sections.forEach((section, idx) => {
    const toggleCurrent = () => {
      if (!mqMobile.matches) {
        return;
      }
      const currentlyOpen = !section.panel.hidden;
      if (currentlyOpen) {
        collapseAll();
        return;
      }
      setOpen(idx);
    };

    section.trigger.addEventListener("click", () => toggleCurrent());
    section.trigger.addEventListener("keydown", (event) => {
      if (event.key !== "Enter" && event.key !== " ") {
        return;
      }
      event.preventDefault();
      toggleCurrent();
    });
  });

  const syncByViewport = () => {
    if (mqMobile.matches) {
      collapseAll();
    } else {
      expandAllDesktop();
    }
  };

  syncByViewport();
  if (typeof mqMobile.addEventListener === "function") {
    mqMobile.addEventListener("change", syncByViewport);
  } else if (typeof mqMobile.addListener === "function") {
    mqMobile.addListener(syncByViewport);
  }
})();

const wireSiteSettingsGoogleMapsKeyForms = () => {
  document.querySelectorAll("form[data-site-settings-map-form]").forEach((form) => {
    if (!(form instanceof HTMLFormElement)) return;
    if (form.dataset.remiGmapsKeyWired === "1") return;
    try {
      initSiteSettingsGoogleMapsKeySection(form);
      form.dataset.remiGmapsKeyWired = "1";
    } catch (err) {
      console.error("Google Maps key settings UI failed to initialize", err);
    }
  });
};

window.addEventListener("load", () => {
  wireSiteSettingsGoogleMapsKeyForms();
});

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

(function initTripDocumentsPage() {
  const root = document.querySelector("[data-trip-documents-root]");
  if (!root) return;
  const mqTripDocDesktop = window.matchMedia("(min-width: 981px)");
  const searchInput = root.querySelector("[data-trip-doc-search]");
  const categorySelect = root.querySelector("[data-trip-doc-category]");
  const items = Array.from(root.querySelectorAll("[data-trip-doc-item]"));
  const empty = root.querySelector("[data-trip-doc-empty]");
  const norm = (v) => String(v || "").trim().toLowerCase();
  const apply = () => {
    const q = norm(searchInput?.value);
    const cat = norm(categorySelect?.value);
    let visible = 0;
    items.forEach((item) => {
      const hay = norm(item.getAttribute("data-doc-search"));
      const itemCat = norm(item.getAttribute("data-doc-category"));
      const match = (!q || hay.includes(q)) && (!cat || itemCat === cat);
      item.classList.toggle("hidden", !match);
      if (match) visible += 1;
    });
    if (empty) empty.classList.toggle("hidden", visible !== 0);
  };
  searchInput?.addEventListener("input", apply);
  categorySelect?.addEventListener("change", apply);
  apply();

  const dropzone = root.querySelector("[data-trip-doc-dropzone]");
  const input = root.querySelector("[data-trip-doc-file-input]");
  const selectedList = root.querySelector("[data-trip-doc-selected-list]");

  /** @type {File[]} */
  let tripDocStagedFiles = [];

  const tripDocFileKey = (f) => `${f.name}\0${f.size}\0${f.lastModified}`;

  const syncTripDocUploadSubmitLabel = () => {
    const btn = root.querySelector(".trip-documents-upload-submit");
    if (!(btn instanceof HTMLButtonElement)) return;
    const n = tripDocStagedFiles.length;
    const label = n > 1 ? "Upload Documents" : "Upload Document";
    btn.textContent = label;
    btn.setAttribute("aria-label", label);
  };

  const syncTripDocStagedToInput = () => {
    if (!(input instanceof HTMLInputElement)) return;
    const dt = new DataTransfer();
    tripDocStagedFiles.forEach((f) => dt.items.add(f));
    input.files = dt.files;
  };

  const addTripDocFilesToStaged = (fileList) => {
    if (!(input instanceof HTMLInputElement) || !fileList || fileList.length === 0) return;
    const seen = new Set(tripDocStagedFiles.map(tripDocFileKey));
    const isBlocked = window.remiIsBlockedUploadFilename;
    const blockedMsg = window.remiBlockedUploadFilenameMessage;
    for (let i = 0; i < fileList.length; i++) {
      const f = fileList[i];
      if (typeof isBlocked === "function" && isBlocked(f.name)) {
        window.remiShowToast?.(
          blockedMsg || "This file type is not allowed — executables and scripts cannot be uploaded."
        );
        continue;
      }
      const k = tripDocFileKey(f);
      if (!seen.has(k)) {
        seen.add(k);
        tripDocStagedFiles.push(f);
      }
    }
    syncTripDocStagedToInput();
    refreshTripDocSelectedList();
  };

  const refreshTripDocSelectedList = () => {
    syncTripDocUploadSubmitLabel();
    if (!(selectedList instanceof HTMLElement) || !(input instanceof HTMLInputElement)) return;
    selectedList.innerHTML = "";
    if (tripDocStagedFiles.length === 0) {
      selectedList.classList.add("hidden");
      return;
    }
    selectedList.classList.remove("hidden");
    tripDocStagedFiles.forEach((f, index) => {
      const li = document.createElement("li");
      li.className = "trip-documents-selected-files__row";
      const nameSpan = document.createElement("span");
      nameSpan.className = "trip-documents-selected-files__name";
      nameSpan.textContent = f.name;
      const rm = document.createElement("button");
      rm.type = "button";
      rm.className = "trip-documents-selected-files__remove";
      rm.setAttribute("data-trip-doc-remove-staged", String(index));
      rm.setAttribute("aria-label", `Remove ${f.name} from upload queue`);
      rm.setAttribute("title", "Remove file");
      rm.innerHTML = '<span class="material-symbols-outlined" aria-hidden="true">close</span>';
      li.append(nameSpan, rm);
      selectedList.appendChild(li);
    });
  };

  if (dropzone && input instanceof HTMLInputElement) {
    input.addEventListener("change", () => {
      if (input.files && input.files.length) {
        addTripDocFilesToStaged(input.files);
      }
    });
    const setHover = (v) => dropzone.classList.toggle("trip-documents-dropzone--hover", v);
    ["dragenter", "dragover"].forEach((evt) =>
      dropzone.addEventListener(evt, (e) => {
        e.preventDefault();
        setHover(true);
      })
    );
    ["dragleave", "drop"].forEach((evt) =>
      dropzone.addEventListener(evt, (e) => {
        e.preventDefault();
        setHover(false);
      })
    );
    dropzone.addEventListener("drop", (e) => {
      const files = e.dataTransfer?.files;
      if (files && files.length) {
        addTripDocFilesToStaged(files);
      }
    });
    refreshTripDocSelectedList();
  }

  const closeRename = (li) => {
    if (!(li instanceof HTMLElement)) return;
    li.classList.remove("editing");
    const form = li.querySelector("[data-trip-doc-rename-form]");
    if (form) form.classList.add("hidden");
    if (!mqTripDocDesktop.matches) {
      const actions = li.querySelector("details.trip-documents-item-actions");
      if (actions instanceof HTMLDetailsElement) actions.open = false;
    }
  };

  root.addEventListener("click", (e) => {
    const stagedRm = e.target.closest("[data-trip-doc-remove-staged]");
    if (
      stagedRm instanceof HTMLButtonElement &&
      selectedList instanceof HTMLElement &&
      selectedList.contains(stagedRm) &&
      input instanceof HTMLInputElement
    ) {
      e.preventDefault();
      const i = parseInt(stagedRm.getAttribute("data-trip-doc-remove-staged") || "", 10);
      if (Number.isNaN(i) || i < 0 || i >= tripDocStagedFiles.length) return;
      tripDocStagedFiles.splice(i, 1);
      syncTripDocStagedToInput();
      refreshTripDocSelectedList();
      return;
    }
    const openUrlBtn = e.target.closest("[data-trip-doc-open-url]");
    if (openUrlBtn && root.contains(openUrlBtn)) {
      e.preventDefault();
      const u = openUrlBtn.getAttribute("data-trip-doc-open-url");
      if (u) {
        window.open(u, "_blank", "noopener,noreferrer");
      }
      return;
    }
    const openBtn = e.target.closest("[data-trip-doc-rename-toggle]");
    if (openBtn && root.contains(openBtn)) {
      e.preventDefault();
      const li = openBtn.closest(".trip-documents-item");
      if (!(li instanceof HTMLElement)) return;
      root.querySelectorAll(".trip-documents-item.editing").forEach((other) => {
        if (other !== li) closeRename(other);
      });
      li.classList.add("editing");
      const form = li.querySelector("[data-trip-doc-rename-form]");
      if (form) {
        form.classList.remove("hidden");
        const inp = form.querySelector('input[name="display_name"]');
        if (inp instanceof HTMLInputElement) {
          window.requestAnimationFrame(() => inp.focus());
        }
      }
      return;
    }
    const cancelBtn = e.target.closest("[data-trip-doc-rename-cancel]");
    if (cancelBtn && root.contains(cancelBtn)) {
      e.preventDefault();
      const li = cancelBtn.closest(".trip-documents-item");
      closeRename(li);
    }
  });

  root.addEventListener("keydown", (e) => {
    if (e.key !== "Escape") return;
    const li = e.target.closest(".trip-documents-item");
    if (li && root.contains(li) && li.classList.contains("editing")) {
      e.preventDefault();
      closeRename(li);
    }
  });
})();
