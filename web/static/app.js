if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch(() => {});
  });
}

/** Login: avoid first-submit with an empty password when the browser/password manager has not filled the field yet (Enter in username, or very fast submit). */
(function remiLoginAutofillGuard() {
  const run = () => {
    document.querySelectorAll('form[action="/login"]').forEach((form) => {
      const pw = form.querySelector('input[name="password"]');
      const ident = form.querySelector('input[name="identifier"]');
      if (!pw || pw.dataset.remiLoginGuard === "1") return;
      pw.dataset.remiLoginGuard = "1";
      const unlock = () => {
        pw.removeAttribute("readonly");
      };
      if (!pw.value) {
        pw.setAttribute("readonly", "");
      }
      ident?.addEventListener("focus", unlock);
      ident?.addEventListener("input", unlock);
      pw.addEventListener("focus", unlock);
      pw.addEventListener("pointerdown", unlock);
      form.addEventListener(
        "submit",
        (e) => {
          if (pw.value !== "") return;
          e.preventDefault();
          unlock();
          pw.focus();
        },
        true
      );
    });
  };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", run);
  } else {
    run();
  }
})();

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

(function remiBookingAttachmentValidate() {
  const SNIFF_LEN = 512;
  const MSG_CONTENT = "Unsupported file type selected. Please upload an image or PDF document.";
  const MSG_EMPTY = "the selected file is empty";
  const MSG_BLOCKED =
    "this file type is not allowed — executables and scripts (.exe, .sh, .bat, .msi, .js, .py, .php, etc.) cannot be uploaded";

  function extFromDeclared(filename) {
    const base = String(filename || "")
      .trim()
      .replace(/^[\\/]+/, "")
      .split(/[/\\]/)
      .pop();
    if (!base) return "";
    const dot = base.lastIndexOf(".");
    return dot >= 0 ? base.slice(dot).toLowerCase() : "";
  }

  function trimBOM(u8) {
    if (u8.length >= 3 && u8[0] === 0xef && u8[1] === 0xbb && u8[2] === 0xbf) return u8.subarray(3);
    if (u8.length >= 2 && u8[0] === 0xfe && u8[1] === 0xff) return u8.subarray(2);
    if (u8.length >= 2 && u8[0] === 0xff && u8[1] === 0xfe) return u8.subarray(2);
    return u8;
  }

  function asciiLowerText(u8, max) {
    const n = Math.min(u8.length, max);
    let s = "";
    for (let i = 0; i < n; i++) {
      let c = u8[i];
      if (c >= 0x41 && c <= 0x5a) c += 0x20;
      s += String.fromCharCode(c);
    }
    return s;
  }

  function looksDangerous(head) {
    if (head.length >= 2 && head[0] === 0x4d && head[1] === 0x5a) return MSG_BLOCKED;
    if (head.length >= 4 && head[0] === 0x7f && head[1] === 0x45 && head[2] === 0x4c && head[3] === 0x46)
      return MSG_BLOCKED;
    if (head.length >= 2 && head[0] === 0x23 && head[1] === 0x21) return MSG_BLOCKED;
    const t = trimBOM(head);
    if (t.length >= 5 && asciiLowerText(t, 5) === "<?php") return MSG_BLOCKED;
    if (t.length >= 4 && asciiLowerText(t, 4) === "<?=") return MSG_BLOCKED;
    if (t.length >= 7 && asciiLowerText(t, 7) === "<script") return MSG_BLOCKED;
    return null;
  }

  function detectKind(u8, filename) {
    if (!u8.length) return "unknown";
    if (u8.length >= 4 && u8[0] === 0x25 && u8[1] === 0x50 && u8[2] === 0x44 && u8[3] === 0x46) return "pdf";
    if (u8.length >= 3 && u8[0] === 0xff && u8[1] === 0xd8 && u8[2] === 0xff) return "jpeg";
    if (
      u8.length >= 8 &&
      u8[0] === 0x89 &&
      u8[1] === 0x50 &&
      u8[2] === 0x4e &&
      u8[3] === 0x47 &&
      u8[4] === 0x0d &&
      u8[5] === 0x0a &&
      u8[6] === 0x1a &&
      u8[7] === 0x0a
    )
      return "png";
    if (
      u8.length >= 6 &&
      ((u8[0] === 0x47 && u8[1] === 0x49 && u8[2] === 0x46 && u8[3] === 0x38 && u8[4] === 0x37 && u8[5] === 0x61) ||
        (u8[0] === 0x47 && u8[1] === 0x49 && u8[2] === 0x46 && u8[3] === 0x38 && u8[4] === 0x39 && u8[5] === 0x61))
    )
      return "gif";
    if (
      u8.length >= 12 &&
      u8[0] === 0x52 &&
      u8[1] === 0x49 &&
      u8[2] === 0x46 &&
      u8[3] === 0x46 &&
      u8[8] === 0x57 &&
      u8[9] === 0x45 &&
      u8[10] === 0x42 &&
      u8[11] === 0x50
    )
      return "webp";
    if (u8.length >= 2 && u8[0] === 0x42 && u8[1] === 0x4d) return "bmp";
    if (
      u8.length >= 4 &&
      ((u8[0] === 0x49 && u8[1] === 0x49 && u8[2] === 0x2a && u8[3] === 0x00) ||
        (u8[0] === 0x4d && u8[1] === 0x4d && u8[2] === 0x00 && u8[3] === 0x2a))
    )
      return "tiff";
    if (u8.length >= 12 && u8[4] === 0x66 && u8[5] === 0x74 && u8[6] === 0x79 && u8[7] === 0x70) {
      const b = asciiLowerText(u8.subarray(8, 12), 4);
      if (b === "heic" || b === "heix" || b === "mif1" || b === "msf1") return "heic";
    }
    if (u8.length >= 4 && u8[0] === 0xd0 && u8[1] === 0xcf && u8[2] === 0x11 && u8[3] === 0xe0) return "ole";
    if (u8.length >= 4 && u8[0] === 0x50 && u8[1] === 0x4b) {
      const th = u8[2];
      const fo = u8[3];
      if ([3, 5, 7].indexOf(th) !== -1 && [4, 6, 8].indexOf(fo) !== -1) return "zip";
    }
    let i = 0;
    while (i < u8.length && (u8[i] === 9 || u8[i] === 10 || u8[i] === 13 || u8[i] === 32)) i++;
    if (i < u8.length && u8[i] === 0x3c) {
      const slice = u8.subarray(i, Math.min(u8.length, i + 256));
      let lt = "";
      try {
        lt = new TextDecoder("utf-8", { fatal: false }).decode(slice).toLowerCase();
      } catch (e) {
        lt = asciiLowerText(slice, slice.length);
      }
      const de = extFromDeclared(filename);
      if (de === ".svg" && (lt.startsWith("<?xml") || lt.startsWith("<!doctype") || lt.startsWith("<svg")))
        return "svg";
    }
    return "unknown";
  }

  function canonicalExt(kind, declaredExt) {
    const d = declaredExt.toLowerCase();
    switch (kind) {
      case "pdf":
        return ".pdf";
      case "jpeg":
        return d === ".jpeg" ? ".jpeg" : ".jpg";
      case "png":
        return ".png";
      case "gif":
        return ".gif";
      case "webp":
        return ".webp";
      case "bmp":
        return ".bmp";
      case "tiff":
        return d === ".tif" ? ".tif" : ".tiff";
      case "heic":
        return d === ".heif" ? ".heif" : ".heic";
      case "ole":
        return ".doc";
      case "zip":
        if (d === ".docx" || d === ".xlsx" || d === ".pptx") return d;
        return "";
      case "svg":
        return ".svg";
      default:
        return "";
    }
  }

  function bookingAttachmentOk(kind, declaredExt) {
    const de = declaredExt.toLowerCase();
    switch (kind) {
      case "pdf":
      case "jpeg":
      case "png":
      case "gif":
      case "webp":
      case "bmp":
      case "tiff":
      case "heic":
      case "ole":
        return true;
      case "zip":
        return de === ".docx" || de === ".xlsx" || de === ".pptx";
      case "svg":
        return de === ".svg";
      default:
        return false;
    }
  }

  window.remiValidateBookingAttachmentFile = (file, maxBytes) => {
    return new Promise((resolve) => {
      if (!(file instanceof File)) {
        resolve({ ok: false, message: MSG_CONTENT });
        return;
      }
      const name = file.name || "";
      if (typeof window.remiIsBlockedUploadFilename === "function" && window.remiIsBlockedUploadFilename(name)) {
        resolve({
          ok: false,
          message: window.remiBlockedUploadFilenameMessage || MSG_BLOCKED
        });
        return;
      }
      if (file.size === 0) {
        resolve({ ok: false, message: MSG_EMPTY });
        return;
      }
      if (typeof maxBytes === "number" && maxBytes > 0 && file.size > maxBytes) {
        const mb = Math.max(1, Math.floor(maxBytes / (1024 * 1024)));
        resolve({ ok: false, message: `file exceeds upload limit (${mb} MB)` });
        return;
      }
      const chunk = file.slice(0, SNIFF_LEN);
      const reader = new FileReader();
      reader.onerror = () => resolve({ ok: false, message: "Could not read the selected file." });
      reader.onload = () => {
        const ab = reader.result;
        if (!(ab instanceof ArrayBuffer)) {
          resolve({ ok: false, message: MSG_CONTENT });
          return;
        }
        const u8 = new Uint8Array(ab);
        const danger = looksDangerous(u8);
        if (danger) {
          resolve({ ok: false, message: danger });
          return;
        }
        const kind = detectKind(u8, name);
        const de = extFromDeclared(name);
        const ext = canonicalExt(kind, de);
        if (!ext || !bookingAttachmentOk(kind, de)) {
          resolve({ ok: false, message: MSG_CONTENT });
          return;
        }
        resolve({ ok: true });
      };
      reader.readAsArrayBuffer(chunk);
    });
  };
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
    const src = String(raw || "")
      .trim()
      .replace(/\b(a\.?m\.?)\b/gi, "am")
      .replace(/\b(p\.?m\.?)\b/gi, "pm")
      .replace(/\s+/g, " ");
    if (!src) return "";
    const compact12 = src.match(/^(\d{1,2})(\d{2})(am|pm)$/i);
    if (compact12) {
      let h = parseInt(compact12[1], 10);
      const mm = parseInt(compact12[2], 10);
      const ap = compact12[3].toLowerCase();
      if (!Number.isFinite(h) || !Number.isFinite(mm) || h < 1 || h > 12 || mm < 0 || mm > 59) return "";
      if (ap === "am") h = h === 12 ? 0 : h;
      else h = h === 12 ? 12 : h + 12;
      return `${pad2(h)}:${pad2(mm)}`;
    }
    const m12 = src.match(/^(\d{1,2}):(\d{1,2})\s*(am|pm)$/i);
    if (m12) {
      let h = parseInt(m12[1], 10);
      const mm = parseInt(m12[2], 10);
      const ap = m12[3].toLowerCase();
      if (!Number.isFinite(h) || !Number.isFinite(mm) || h < 1 || h > 12 || mm < 0 || mm > 59) return "";
      if (ap === "am") h = h === 12 ? 0 : h;
      else h = h === 12 ? 12 : h + 12;
      return `${pad2(h)}:${pad2(mm)}`;
    }
    const monly = src.match(/^(\d{1,2})\s*(am|pm)$/i);
    if (monly) {
      let h = parseInt(monly[1], 10);
      const ap = monly[2].toLowerCase();
      if (!Number.isFinite(h) || h < 1 || h > 12) return "";
      if (ap === "am") h = h === 12 ? 0 : h;
      else h = h === 12 ? 12 : h + 12;
      return `${pad2(h)}:00`;
    }
    const compact24 = src.match(/^(\d{3,4})$/);
    if (compact24) {
      const token = compact24[1];
      const hh = token.length === 3 ? parseInt(token.slice(0, 1), 10) : parseInt(token.slice(0, 2), 10);
      const mm = parseInt(token.slice(-2), 10);
      if (!Number.isFinite(hh) || !Number.isFinite(mm) || hh < 0 || hh > 23 || mm < 0 || mm > 59) return "";
      return `${pad2(hh)}:${pad2(mm)}`;
    }
    const h24only = src.match(/^(\d{1,2})$/);
    if (h24only) {
      const hh = parseInt(h24only[1], 10);
      if (!Number.isFinite(hh) || hh < 0 || hh > 23) return "";
      return `${pad2(hh)}:00`;
    }
    const m24 = src.match(/^(\d{1,2}):(\d{1,2})$/);
    if (!m24) return "";
    const h = parseInt(m24[1], 10);
    const mm = parseInt(m24[2], 10);
    if (!Number.isFinite(h) || !Number.isFinite(mm) || h < 0 || h > 23 || mm < 0 || mm > 59) return "";
    return `${pad2(h)}:${pad2(mm)}`;
  }

  function snapMinuteToPickerStep(mm) {
    if (!Number.isFinite(mm) || mm < 0 || mm > 59) return 0;
    return Math.min(55, Math.max(0, Math.round(mm / 5) * 5));
  }

  function hmToPickerParts(hm) {
    const t = normalizeTimeHM(hm);
    if (!t) return null;
    const h24 = parseInt(t.slice(0, 2), 10);
    const mm = parseInt(t.slice(3, 5), 10);
    const period = h24 >= 12 ? "pm" : "am";
    let hour12 = h24 % 12;
    if (hour12 === 0) hour12 = 12;
    return { hour12, minute: snapMinuteToPickerStep(mm), period };
  }

  /** Partial typed values: move lists while the time picker is open (5-minute steps for minutes). */
  function parseDisplayForTimePickerView(t, st) {
    const s = String(t || "").trim().replace(/\s+/g, " ");
    if (!s) return false;
    const m = s.match(/^(\d{1,2})(?::(\d{0,2}))?\s*(am|pm)?$/i);
    if (!m) return false;
    const h = parseInt(m[1], 10);
    const minStr = m[2];
    const mm = minStr === undefined || minStr === "" ? null : parseInt(minStr, 10);
    const ap = m[3] ? m[3].toLowerCase() : null;
    if (!Number.isFinite(h)) return false;
    if (h >= 13 && h <= 23) {
      st.period = "pm";
      let h12 = h % 12;
      if (h12 === 0) h12 = 12;
      st.hour12 = h12;
      if (Number.isFinite(mm) && mm >= 0 && mm <= 59) st.minute = snapMinuteToPickerStep(mm);
      return true;
    }
    if (h === 0) {
      if (ap === "am") {
        st.hour12 = 12;
        st.period = "am";
        if (Number.isFinite(mm) && mm >= 0 && mm <= 59) st.minute = snapMinuteToPickerStep(mm);
        return true;
      }
      return false;
    }
    if (h >= 1 && h <= 12) {
      st.hour12 = h;
      if (ap) st.period = ap;
      if (Number.isFinite(mm) && mm >= 0 && mm <= 59) st.minute = snapMinuteToPickerStep(mm);
      return true;
    }
    return false;
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

  function timeHMToDisplay24(raw) {
    const t = normalizeTimeHM(raw);
    return t || "";
  }

  function dateTimeLocalToDisplay(raw, mdy, uiTime24h) {
    const sp = splitIsoLocalDateTime(raw);
    if (!sp.dateIso || !sp.time) return "";
    const timeDisp = uiTime24h ? timeHMToDisplay24(sp.time) : timeHMToDisplay(sp.time);
    return `${isoToDisplay(sp.dateIso, mdy)}, ${timeDisp}`;
  }

  function digitsOnly(s) {
    return String(s || "").replace(/\D/g, "");
  }

  /**
   * Hyphens between MM-DD-YYYY groups; when isDelete is true, omit a trailing separator
   * that has no digits in the following group yet (so backspace can remove the last digit).
   */
  function formatDateDigitsWithSeparators(digs, isDelete) {
    const d = digs.slice(0, 8);
    if (!d) return "";
    if (d.length <= 2) {
      let out = d;
      if (d.length === 2 && !isDelete) out += "-";
      return out;
    }
    let out = `${d.slice(0, 2)}-`;
    if (d.length <= 4) {
      out += d.slice(2, 4);
      if (d.length === 4 && !isDelete) out += "-";
      return out;
    }
    return `${d.slice(0, 2)}-${d.slice(2, 4)}-${d.slice(4)}`;
  }

  function formatTimeDigitsWithColon(digs, isDelete) {
    const d = digs.slice(0, 4);
    if (!d) return "";
    if (d.length <= 2) {
      let out = d;
      if (d.length === 2 && !isDelete) out += ":";
      return out;
    }
    if (d.length === 3) return `${d[0]}:${d.slice(1)}`;
    return `${d.slice(0, 2)}:${d.slice(2, 4)}`;
  }

  /**
   * Preserves a user-typed ":" while still auto-inserting ":" for digit-only times (e.g. 1400 → 14:00).
   * Without this, "14:" collapses to "14" and the user cannot type minutes.
   */
  function formatTimePartTypingFromRest(rest, isDelete) {
    const r = String(rest || "");
    if (!r.trim()) return "";
    const ci = r.indexOf(":");
    if (ci >= 0) {
      const h = digitsOnly(r.slice(0, ci)).slice(0, 2);
      const m = digitsOnly(r.slice(ci + 1)).slice(0, 2);
      return `${h}:${m}`;
    }
    return formatTimeDigitsWithColon(digitsOnly(r).slice(0, 4), isDelete);
  }

  /** After typing a digit, move caret past an auto-inserted - or : so the next key goes in the next group. */
  function skipCaretPastSeparatorRun(segment, caretInSegment) {
    let c = Math.min(Math.max(0, caretInSegment | 0), segment.length);
    while (c < segment.length && /[-/:]/.test(segment[c])) c++;
    return c;
  }

  /** Map Nth digit position (1-based count) to cursor index after that digit in formatted date. */
  function cursorAfterNthDigitInDate(formatted, n) {
    if (n <= 0) return 0;
    let seen = 0;
    for (let i = 0; i < formatted.length; i++) {
      if (/\d/.test(formatted[i])) {
        seen++;
        if (seen === n) return i + 1;
      }
    }
    return formatted.length;
  }

  function caretInCombinedFromDigitIndex(dateFmt, timeFmt, digitIndex) {
    const sep = ", ";
    if (digitIndex <= 8) {
      const c0 = cursorAfterNthDigitInDate(dateFmt, digitIndex);
      return skipCaretPastSeparatorRun(dateFmt, c0);
    }
    const td = digitIndex - 8;
    const inner0 = cursorAfterNthDigitInDate(timeFmt, td);
    const inner = skipCaretPastSeparatorRun(timeFmt, inner0);
    return dateFmt.length + sep.length + inner;
  }

  /** Date-only typing mask used by split mobile date part. */
  function remiFormatDatePartTypingMask(raw, oldCaret, isDelete) {
    const src = String(raw || "");
    const pos = Math.min(Math.max(0, oldCaret | 0), src.length);
    const digsBefore = digitsOnly(src.slice(0, pos)).length;
    const dateFmt = formatDateDigitsWithSeparators(digitsOnly(src), isDelete);
    let caret = cursorAfterNthDigitInDate(dateFmt, digsBefore);
    caret = skipCaretPastSeparatorRun(dateFmt, caret);
    return { text: dateFmt, caret: Math.min(Math.max(0, caret), dateFmt.length) };
  }

  /**
   * Inserts "-" in the date segment and ":" in the time segment while typing the combined field.
   * isDelete: from InputEvent.inputType so trailing separators are not re-added on backspace.
   * Returns { text, caret }.
   */
  function remiFormatCombinedDateTimeTypingMask(raw, mdy, oldCaret, isDelete) {
    const src = String(raw || "");
    const pos = Math.min(Math.max(0, oldCaret | 0), src.length);
    const digsAll = digitsOnly(src);
    const cidx = src.indexOf(",");

    if (cidx < 0 && digsAll.length === 8 && !isDelete) {
      const dateFmt = formatDateDigitsWithSeparators(digsAll, false);
      const out = `${dateFmt}, `;
      return { text: out, caret: out.length };
    }

    if (cidx < 0 && digsAll.length > 8) {
      const dateD = digsAll.slice(0, 8);
      const timeD = digsAll.slice(8, 12);
      const dateFmt = formatDateDigitsWithSeparators(dateD, isDelete);
      const timeFmt = formatTimeDigitsWithColon(timeD, isDelete);
      const out = `${dateFmt}, ${timeFmt}`;
      const digitBefore = digitsOnly(src.slice(0, pos)).length;
      const caret = Math.min(caretInCombinedFromDigitIndex(dateFmt, timeFmt, digitBefore), out.length);
      return { text: out, caret };
    }

    if (cidx < 0) {
      const head = src;
      const digsBefore = digitsOnly(head.slice(0, pos)).length;
      const dateFmt = formatDateDigitsWithSeparators(digitsOnly(head), isDelete);
      let caret = cursorAfterNthDigitInDate(dateFmt, digsBefore);
      caret = skipCaretPastSeparatorRun(dateFmt, caret);
      return { text: dateFmt, caret };
    }

    const dateRaw = src.slice(0, cidx);
    const afterComma = src.slice(cidx + 1);
    const spMatch = afterComma.match(/^\s*/);
    const lead = spMatch ? spMatch[0] : "";
    const rest0 = afterComma.slice(lead.length);
    const apmM = rest0.match(/\s*([ap]m)\s*$/i);
    let rest = rest0;
    let apm = "";
    if (apmM) {
      apm = ` ${apmM[1].toUpperCase()}`;
      rest = rest0.slice(0, apmM.index).trimEnd();
    }
    const dateFmt = formatDateDigitsWithSeparators(digitsOnly(dateRaw), isDelete);
    const timeFmt = formatTimePartTypingFromRest(rest, isDelete);
    if (isDelete && !timeFmt && !apm) {
      const caret0 = Math.min(Math.max(0, pos), dateFmt.length);
      return { text: dateFmt, caret: caret0 };
    }
    const out = `${dateFmt}, ${timeFmt}${apm}`;
    const timeStart = dateFmt.length + 2;
    let caret = out.length;
    if (pos <= cidx) {
      const digsBefore = digitsOnly(dateRaw.slice(0, pos)).length;
      let c = cursorAfterNthDigitInDate(dateFmt, digsBefore);
      caret = skipCaretPastSeparatorRun(dateFmt, c);
    } else {
      const timeSegStart = cidx + 1 + lead.length;
      const coreEndInSrc = timeSegStart + rest.length;
      if (pos >= coreEndInSrc) {
        caret = Math.min(timeStart + timeFmt.length + Math.max(0, pos - coreEndInSrc), out.length);
      } else {
        const rel = Math.max(0, pos - timeSegStart);
        const digsInTimeBefore = digitsOnly(rest.slice(0, Math.min(rel, rest.length))).length;
        let inner = cursorAfterNthDigitInDate(timeFmt, digsInTimeBefore);
        inner = skipCaretPastSeparatorRun(timeFmt, inner);
        caret = timeStart + inner;
      }
    }
    caret = Math.min(Math.max(0, caret), out.length);
    return { text: out, caret };
  }

  function splitCombinedDisplayDateTimeParts(raw, mdy) {
    const src = String(raw || "")
      .trim()
      .replace("T", " ")
      .replace(/\s+/g, " ");
    const commaIdx = src.indexOf(",");
    if (commaIdx >= 0) {
      return { datePart: src.slice(0, commaIdx).trim(), timePart: src.slice(commaIdx + 1).trim() };
    }
    const digs = digitsOnly(src);
    if (digs.length > 8) {
      const dateStr = formatDateDigitsWithSeparators(digs.slice(0, 8), false);
      const timeStr = formatTimeDigitsWithColon(digs.slice(8, 12));
      return { datePart: dateStr, timePart: timeStr };
    }
    const m = src.match(/^(.+?)\s+(\d{1,2}([:.]\d{0,2})?.*)$/i);
    if (m) return { datePart: m[1].trim(), timePart: m[2].trim() };
    return { datePart: src, timePart: "" };
  }

  /** Avoid snapping "1" or "12" to HH:00 while the user is still typing the time portion. */
  function isTimePartIncompleteForCombinedInput(timePartRaw) {
    const t = String(timePartRaw || "").trim();
    if (!t) return true;
    if (/am|pm/i.test(t)) return !normalizeTimeHM(t);
    if (/^\d{1,2}$/.test(t.replace(/\s/g, ""))) return true;
    if (/^\d{3,4}$/.test(t.replace(/\s/g, ""))) return false;
    if (!t.includes(":")) return true;
    const colonIdx = t.indexOf(":");
    const after = t.slice(colonIdx + 1).replace(/\s*(am|pm).*$/i, "").trim();
    const mDigits = after.replace(/\D/g, "");
    if (mDigits.length === 0) return true;
    if (mDigits.length === 1) return true;
    return false;
  }

  /** In 12h mode, once a complete time is present without AM/PM, canon to explicit AM/PM display. */
  function combinedTimeNeeds12hCanonVisual(timePartRaw) {
    const tp = String(timePartRaw || "").trim();
    if (!tp || /am|pm/i.test(tp)) return false;
    if (isTimePartIncompleteForCombinedInput(tp)) return false;
    const hm = normalizeTimeHM(tp);
    return Boolean(hm);
  }

  function setHMPeriod(hmRaw, targetPeriod) {
    const hm = normalizeTimeHM(hmRaw);
    const period = String(targetPeriod || "").toLowerCase() === "pm" ? "pm" : "am";
    if (!hm) return "";
    let hh = parseInt(hm.slice(0, 2), 10);
    if (!Number.isFinite(hh)) return "";
    if (period === "am") {
      if (hh >= 12) hh -= 12;
    } else if (hh < 12) {
      hh += 12;
    }
    return `${pad2(hh)}:${hm.slice(3, 5)}`;
  }

  /**
   * Product rule: when users type explicit 24h time in 12h UI (e.g. 14:34 / 1434),
   * normalize minutes to :00 before converting to AM/PM display.
   */
  function maybeForceTopOfHourForTyped24h(timePartRaw, hmRaw) {
    const hm = normalizeTimeHM(hmRaw);
    if (!hm) return "";
    const tp = String(timePartRaw || "").trim();
    if (!tp || /am|pm/i.test(tp)) return hm;
    const compact = tp.replace(/\s/g, "");
    if (/^\d{3,4}$/.test(compact)) return `${hm.slice(0, 2)}:00`;
    const m = compact.match(/^(\d{1,2}):(\d{2})$/);
    if (!m) return hm;
    const hh = parseInt(m[1], 10);
    if (!Number.isFinite(hh)) return hm;
    if (hh === 0 || hh >= 13) return `${pad2(hh)}:00`;
    return hm;
  }

  /** Standalone time visible field (mobile): insert ":" between hour and minute digits. */
  function remiFormatTimePartTypingMask(raw, oldCaret, isDelete) {
    const src = String(raw || "");
    const pos = Math.min(Math.max(0, oldCaret | 0), src.length);
    const apmM = src.match(/\s*([ap]m)\s*$/i);
    const apmStart = apmM ? apmM.index : src.length;
    let rest = (apmM ? src.slice(0, apmM.index) : src).trimEnd();
    const apm = apmM ? ` ${apmM[1].toUpperCase()}` : "";
    const timeFmt = formatTimePartTypingFromRest(rest, isDelete);
    const out = `${timeFmt}${apm}`;
    let caret = out.length;
    if (pos >= apmStart) {
      caret = Math.min(timeFmt.length + Math.max(0, pos - apmStart), out.length);
    } else {
      const digsBefore = digitsOnly(rest.slice(0, Math.min(pos, rest.length))).length;
      let inner = cursorAfterNthDigitInDate(timeFmt, digsBefore);
      inner = skipCaretPastSeparatorRun(timeFmt, inner);
      caret = inner;
    }
    return { text: out, caret: Math.min(Math.max(0, caret), out.length) };
  }

  /** requireCompleteTime: when true (live typing), do not snap bare hours like "1" to 01:00. */
  function parseCombinedDateTimeDisplay(raw, mdy, requireCompleteTime) {
    const src = String(raw || "")
      .trim()
      .replace("T", " ")
      .replace(/\s+/g, " ");
    if (!src) return "";
    const { datePart, timePart } = splitCombinedDisplayDateTimeParts(src, mdy);
    if (requireCompleteTime && isTimePartIncompleteForCombinedInput(timePart)) return "";
    const dateIso = parseDisplayToIso(datePart, mdy);
    const hm = normalizeTimeHM(timePart);
    if (!dateIso || !hm) return "";
    return `${dateIso}T${hm}`;
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

  function remiSetFormDateIso(form, iso) {
    if (!(form instanceof HTMLFormElement)) return;
    const s = String(iso || "").trim();
    if (!s) return;
    const wrap = form.querySelector("[data-remi-date]");
    if (!(wrap instanceof HTMLElement)) return;
    const mdy = wrap.getAttribute("data-mdy") === "1";
    const min = wrap.getAttribute("data-min") || "";
    const max = wrap.getAttribute("data-max") || "";
    const hidden = wrap.querySelector(".remi-date-iso");
    const vis = wrap.querySelector(".remi-date-visible");
    if (!(hidden instanceof HTMLInputElement) || !(vis instanceof HTMLInputElement)) return;
    const v = clampIso(s, min, max);
    hidden.value = v;
    vis.value = isoToDisplay(v, mdy);
    vis.setCustomValidity("");
  }

  function remiUpdateUnifiedSplitJSON(form) {
    if (!(form instanceof HTMLFormElement)) return;
    const splitJson = form.querySelector(".remi-unified-expense-split-json");
    const fromTab = form.querySelector(".remi-unified-expense-from-tab");
    if (!(splitJson instanceof HTMLInputElement) || !fromTab || fromTab.disabled) return;
    const keys = Array.from(form.querySelectorAll(".remi-unified-expense-participant-cb:checked"))
      .map((el) => (el instanceof HTMLInputElement ? (el.getAttribute("data-participant-key") || "").trim() : ""))
      .filter(Boolean);
    splitJson.value = keys.length ? JSON.stringify({ participants: keys }) : "";
  }

  function remiSyncUnifiedExpenseGroupMode(form, on) {
    if (!(form instanceof HTMLFormElement)) return;
    if (form.dataset.unifiedExpenseCanSplit !== "1") return;
    const fromTab = form.querySelector(".remi-unified-expense-from-tab");
    const splitMode = form.querySelector(".remi-unified-expense-split-mode");
    const splitJson = form.querySelector(".remi-unified-expense-split-json");
    const fieldset = form.querySelector(".remi-unified-expense-split-fieldset");
    if (!(fromTab instanceof HTMLInputElement) || !(splitMode instanceof HTMLInputElement) || !(splitJson instanceof HTMLInputElement)) {
      return;
    }
    const personalLabel = form.querySelector(".remi-unified-expense-submit-personal");
    const groupLabel = form.querySelector(".remi-unified-expense-submit-group");
    if (on) {
      fromTab.removeAttribute("disabled");
      splitMode.removeAttribute("disabled");
      splitJson.removeAttribute("disabled");
      if (fieldset instanceof HTMLFieldSetElement) {
        fieldset.classList.remove("hidden");
        fieldset.removeAttribute("disabled");
      }
    } else {
      fromTab.setAttribute("disabled", "");
      splitMode.setAttribute("disabled", "");
      splitJson.setAttribute("disabled", "");
      if (fieldset instanceof HTMLFieldSetElement) {
        fieldset.classList.add("hidden");
        fieldset.setAttribute("disabled", "");
      }
    }
    remiUpdateUnifiedSplitJSON(form);
    if (personalLabel instanceof HTMLElement) personalLabel.classList.toggle("hidden", on);
    if (groupLabel instanceof HTMLElement) groupLabel.classList.toggle("hidden", !on);
  }

  function remiWireUnifiedExpenseForm(form) {
    if (!(form instanceof HTMLFormElement) || !form.matches("[data-remi-unified-expense-add-form]")) return;
    if (form.dataset.remiUnifiedExpenseWired === "1") return;
    form.dataset.remiUnifiedExpenseWired = "1";
    const splitCb = form.querySelector(".remi-unified-expense-split-checkbox");
    const selectAllBtn = form.querySelector(".remi-unified-expense-select-all");
    const cancelBtn = form.querySelector(".remi-unified-expense-cancel");
    if (splitCb instanceof HTMLInputElement) {
      splitCb.addEventListener("change", () => {
        remiSyncUnifiedExpenseGroupMode(form, splitCb.checked);
      });
    }
    form.querySelectorAll(".remi-unified-expense-participant-cb").forEach((el) => {
      el.addEventListener("change", () => remiUpdateUnifiedSplitJSON(form));
    });
    if (selectAllBtn instanceof HTMLElement) {
      selectAllBtn.addEventListener("click", (ev) => {
        ev.preventDefault();
        const cbs = form.querySelectorAll(".remi-unified-expense-participant-cb");
        const anyUnchecked = Array.from(cbs).some((c) => c instanceof HTMLInputElement && !c.checked);
        cbs.forEach((c) => {
          if (c instanceof HTMLInputElement) c.checked = anyUnchecked;
        });
        remiUpdateUnifiedSplitJSON(form);
      });
    }
    form.addEventListener(
      "submit",
      (ev) => {
        const fromTab = form.querySelector(".remi-unified-expense-from-tab");
        if (fromTab instanceof HTMLInputElement && !fromTab.disabled) {
          remiUpdateUnifiedSplitJSON(form);
          const splitJson = form.querySelector(".remi-unified-expense-split-json");
          const raw = splitJson instanceof HTMLInputElement ? splitJson.value.trim() : "";
          if (!raw) {
            ev.preventDefault();
            (window.remiShowToast || alert)("Choose at least one person to split with.");
            return;
          }
          try {
            const o = JSON.parse(raw);
            if (!o.participants || !o.participants.length) {
              ev.preventDefault();
              (window.remiShowToast || alert)("Choose at least one person to split with.");
            }
          } catch {
            ev.preventDefault();
            (window.remiShowToast || alert)("Split selection is invalid. Try again.");
          }
        }
      },
      true
    );
    if (cancelBtn instanceof HTMLElement) {
      cancelBtn.addEventListener("click", (ev) => {
        ev.preventDefault();
        remiResetExpenseAddForm(form);
        const fly = form.closest("[data-desktop-expense-flyout]");
        if (fly instanceof HTMLElement) {
          fly.classList.add("hidden");
          fly.setAttribute("aria-hidden", "true");
          return;
        }
        const sheet = form.closest(".mobile-sheet");
        const closer = sheet?.querySelector("[data-mobile-sheet-close]");
        if (closer instanceof HTMLElement) closer.click();
      });
    }
  }

  function remiResetExpenseAddForm(form) {
    if (!(form instanceof HTMLFormElement)) return;
    const titleInp = form.querySelector('input[name="title"]');
    if (titleInp instanceof HTMLInputElement) titleInp.value = "";
    const notesInp = form.querySelector('input[name="notes"]');
    if (notesInp instanceof HTMLInputElement) notesInp.value = "";
    const amtInp = form.querySelector('input[name="amount"]');
    if (amtInp instanceof HTMLInputElement) amtInp.value = "";
    const amtHero = form.querySelector(".remi-unified-expense-amount-input");
    if (amtHero instanceof HTMLInputElement) amtHero.value = "";
    form.querySelectorAll('select[name="category"], select[name="payment_method"]').forEach((el) => {
      if (el instanceof HTMLSelectElement) el.selectedIndex = 0;
    });
    const def = (form.dataset.remiDefaultSpentOn || "").trim();
    if (def) remiSetFormDateIso(form, def);
    const tabFile = form.querySelector('input[type="file"][name="tab_attachment"]');
    if (tabFile instanceof HTMLInputElement) tabFile.value = "";
    if (form.matches("[data-remi-unified-expense-add-form]")) {
      const splitCb = form.querySelector(".remi-unified-expense-split-checkbox");
      if (splitCb instanceof HTMLInputElement && !splitCb.disabled) {
        splitCb.checked = false;
        window.remiSyncUnifiedExpenseGroupMode?.(form, false);
      }
      form.querySelectorAll(".remi-unified-expense-participant-cb").forEach((el) => {
        if (el instanceof HTMLInputElement) el.checked = true;
      });
      window.remiUpdateUnifiedSplitJSON?.(form);
    }
    if (form.closest(".mobile-sheet")) {
      window.remiNotifyMobileSheetFormSaved?.(form);
    }
  }

  window.remiSetFormDateIso = remiSetFormDateIso;
  window.remiResetExpenseAddForm = remiResetExpenseAddForm;
  window.remiWireUnifiedExpenseForm = remiWireUnifiedExpenseForm;
  window.remiSyncUnifiedExpenseGroupMode = remiSyncUnifiedExpenseGroupMode;
  window.remiUpdateUnifiedSplitJSON = remiUpdateUnifiedSplitJSON;

  /** Reject attrs like "T00:00" from empty trip dates — they corrupt string compares and ISO parsing. */
  function isUsableTripDateTimeBound(s) {
    return /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/.test(String(s || "").trim());
  }

  function clampDateTimeLocal(isoDt, min, max) {
    if (!isoDt) return isoDt;
    let v = isoDt;
    const mn = String(min || "").trim();
    const mx = String(max || "").trim();
    if (isUsableTripDateTimeBound(mn) && v < mn) v = mn;
    if (isUsableTripDateTimeBound(mx) && v > mx) v = mx;
    return v;
  }

  function isDateTimeWithinBounds(isoDt, min, max) {
    const v = String(isoDt || "").trim();
    if (!v) return false;
    const mn = String(min || "").trim();
    const mx = String(max || "").trim();
    if (isUsableTripDateTimeBound(mn) && v < mn) return false;
    if (isUsableTripDateTimeBound(mx) && v > mx) return false;
    return true;
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

  /** While the date picker is open, typing in the visible field updates the calendar (see renderDatePicker). */
  let activeDatePickerSync = null;
  /** While the time picker is open, typing updates hour/minute/AM-PM lists (see renderTimePicker). */
  let activeTimePickerSync = null;

  function parseDisplayToIso(trimmed, mdy) {
    const t = String(trimmed || "").trim();
    if (!t) return "";
    const parts = t.split(/[-/.]/).map((p) => p.trim()).filter((p) => p.length > 0);
    if (parts.length !== 3) return "";
    const n1 = parseInt(parts[0], 10);
    const n2 = parseInt(parts[1], 10);
    const n3 = parseInt(parts[2], 10);
    if (![n1, n2, n3].every((n) => Number.isFinite(n))) return "";
    let d;
    let mo;
    let y;
    if (mdy) {
      mo = n1;
      d = n2;
      y = n3;
    } else {
      d = n1;
      mo = n2;
      y = n3;
    }
    if (y < 1000 || y > 9999 || mo < 1 || mo > 12 || d < 1 || d > 31) return "";
    const dt = new Date(y, mo - 1, d);
    if (dt.getFullYear() !== y || dt.getMonth() !== mo - 1 || dt.getDate() !== d) return "";
    return toISODate(dt);
  }

  function parseDisplayForCalendarView(trimmed, mdy, viewFallback) {
    const t = String(trimmed || "").trim();
    if (!t) return null;
    const fb = viewFallback instanceof Date ? viewFallback : new Date();
    const y0 = fb.getFullYear();
    const full = parseDisplayToIso(t, mdy);
    if (full) {
      const d = parseISODate(full);
      return d ? startOfMonth(d) : null;
    }
    const parts = t.split(/[-/.]/).map((p) => p.trim()).filter((p) => p.length > 0);
    if (parts.length === 0) return null;
    if (parts.length === 1) {
      const n = parseInt(parts[0], 10);
      if (Number.isFinite(n) && n >= 1 && n <= 12) return new Date(y0, n - 1, 1);
      return null;
    }
    if (parts.length === 2) {
      const n1 = parseInt(parts[0], 10);
      const n2 = parseInt(parts[1], 10);
      if (!Number.isFinite(n1) || !Number.isFinite(n2)) return null;
      if (n2 >= 1000 && n2 <= 9999 && n1 >= 1 && n1 <= 12) return new Date(n2, n1 - 1, 1);
      if (n1 >= 1000 && n1 <= 9999 && n2 >= 1 && n2 <= 12) return new Date(n1, n2 - 1, 1);
      if (mdy && n1 >= 1 && n1 <= 12) return new Date(y0, n1 - 1, 1);
      if (!mdy && n2 >= 1 && n2 <= 12) return new Date(y0, n2 - 1, 1);
      return null;
    }
    if (parts.length >= 3) {
      const n1 = parseInt(parts[0], 10);
      const n2 = parseInt(parts[1], 10);
      const n3 = parseInt(parts[2], 10);
      if (mdy && Number.isFinite(n1) && n1 >= 1 && n1 <= 12) {
        const yy = Number.isFinite(n3) && n3 >= 1000 && n3 <= 9999 ? n3 : y0;
        return new Date(yy, n1 - 1, 1);
      }
      if (!mdy && Number.isFinite(n2) && n2 >= 1 && n2 <= 12) {
        const yy = Number.isFinite(n3) && n3 >= 1000 && n3 <= 9999 ? n3 : y0;
        return new Date(yy, n2 - 1, 1);
      }
    }
    return null;
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
      const vh = window.innerHeight;
      if (r.bottom < 0 || r.top > vh) {
        close("anchor-out-of-view");
        return;
      }
      const PICKER_PANEL_MAX = 360;
      const PICKER_PANEL_MIN = 280;
      const vwCap = Math.max(200, window.innerWidth - 24);
      const panelWidth = Math.min(
        PICKER_PANEL_MAX,
        vwCap,
        Math.max(PICKER_PANEL_MIN, Math.round(r.width))
      );
      const maxLeft = Math.max(12, window.innerWidth - panelWidth - 12);
      const left = Math.min(maxLeft, Math.max(12, Math.round(r.left)));
      panel.style.left = `${left}px`;
      panel.style.width = `${panelWidth}px`;
      panel.style.maxHeight = "";

      const margin = 12;
      const gap = 10;
      const finalize = () => {
        const h = panel.getBoundingClientRect().height;
        const roomBelow = vh - margin - (r.bottom + gap);
        const roomAbove = r.top - gap - margin;
        let top = r.bottom + gap;

        if (h <= roomBelow) {
          /* below anchor */
        } else if (h <= roomAbove) {
          top = r.top - gap - h;
        } else {
          const preferBelow = roomBelow >= roomAbove;
          const cap = Math.max(160, preferBelow ? roomBelow : roomAbove);
          panel.style.maxHeight = `${Math.floor(cap)}px`;
          const h2 = panel.getBoundingClientRect().height;
          top = preferBelow ? r.bottom + gap : r.top - gap - h2;
          if (top < margin) top = margin;
          if (top + h2 > vh - margin) top = Math.max(margin, vh - margin - h2);
        }

        if (!panel.style.maxHeight && top + h > vh - margin) {
          top = Math.max(margin, vh - margin - h);
        }
        panel.style.top = `${Math.round(top)}px`;
      };

      requestAnimationFrame(() => {
        finalize();
        requestAnimationFrame(finalize);
      });
    };

    const close = (reason) => {
      if (!host || !state) return;
      activeDatePickerSync = null;
      activeTimePickerSync = null;
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
    const clearable = Boolean(opts.clearable);
    const mdyPicker = Boolean(opts.mdy);
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
      '<div class="remi-cal-grid"></div>'
    ].join("");
    const actions = document.createElement("div");
    actions.className = "remi-picker-actions";
    if (clearable) {
      const bClear = document.createElement("button");
      bClear.type = "button";
      bClear.className = "secondary-btn remi-picker-clear-date";
      bClear.textContent = "Clear date";
      bClear.addEventListener("click", () => {
        opts.onConfirm("");
        close("clear");
      });
      actions.appendChild(bClear);
    }
    const bCancel = document.createElement("button");
    bCancel.type = "button";
    bCancel.className = "secondary-btn";
    bCancel.setAttribute("data-cal-cancel", "");
    bCancel.textContent = "Cancel";
    actions.appendChild(bCancel);
    const bConfirm = document.createElement("button");
    bConfirm.type = "button";
    bConfirm.className = "primary-btn";
    bConfirm.setAttribute("data-cal-confirm", "");
    bConfirm.textContent = "Confirm Date";
    actions.appendChild(bConfirm);
    wrap.appendChild(actions);
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

    activeDatePickerSync = (rawDisplay) => {
      const t = String(rawDisplay || "").trim();
      const iso = parseDisplayToIso(t, mdyPicker);
      if (iso) {
        selectedIso = clampIso(iso, min, max);
        const parsed = parseISODate(selectedIso);
        if (parsed) viewMonth = startOfMonth(parsed);
        draw();
        return;
      }
      const vm = parseDisplayForCalendarView(t, mdyPicker, viewMonth);
      if (vm) {
        viewMonth = vm;
        draw();
      }
    };
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
      requestAnimationFrame(() => {
        hourList.querySelector(".is-selected")?.scrollIntoView({ block: "nearest" });
        minuteList.querySelector(".is-selected")?.scrollIntoView({ block: "nearest" });
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

    activeTimePickerSync = (rawDisplay) => {
      const t = String(rawDisplay || "").trim();
      const full = normalizeTimeHM(t);
      if (full) {
        const p = hmToPickerParts(full);
        if (p) {
          hour12 = p.hour12;
          minute = p.minute;
          period = p.period;
          draw();
        }
        return;
      }
      const st = { hour12, minute, period };
      if (parseDisplayForTimePickerView(t, st)) {
        hour12 = st.hour12;
        minute = st.minute;
        period = st.period;
        draw();
      }
    };
  }

  function openDatePicker(anchor, options) {
    picker.open({
      anchor,
      title: "Select Date",
      value: options.value || "",
      min: options.min || "",
      max: options.max || "",
      mdy: Boolean(options.mdy),
      clearable: Boolean(options.clearable),
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
    const getMin = () => wrap.getAttribute("data-min") || "";
    const getMax = () => wrap.getAttribute("data-max") || "";
    const clearable = wrap.getAttribute("data-clearable") === "1";
    const hidden = wrap.querySelector(".remi-date-iso");
    const vis = wrap.querySelector(".remi-date-visible");
    const calBtn = wrap.querySelector(".remi-date-calendar-btn");
    const clearBtn = wrap.querySelector("[data-remi-date-clear]");
    if (!(hidden instanceof HTMLInputElement) || !(vis instanceof HTMLInputElement)) return;
    if (!(calBtn instanceof HTMLButtonElement)) return;
    if (!isMobileLayout()) {
      const requiredNative = vis.required;
      vis.type = "date";
      vis.readOnly = false;
      vis.removeAttribute("placeholder");
      const syncNative = () => {
        const min = getMin();
        const max = getMax();
        if (min) vis.min = min;
        else vis.removeAttribute("min");
        if (max) vis.max = max;
        else vis.removeAttribute("max");
        const v = clampIso((hidden.value || "").trim(), min, max);
        hidden.value = v;
        vis.value = v || "";
        vis.setCustomValidity("");
        if (clearBtn instanceof HTMLButtonElement) {
          const has = Boolean((hidden.value || "").trim());
          clearBtn.hidden = !clearable || !has;
          clearBtn.setAttribute("aria-hidden", has ? "false" : "true");
        }
      };
      const validateNative = () => {
        vis.setCustomValidity("");
        if (requiredNative && !(hidden.value || "").trim()) {
          vis.setCustomValidity("Select a date");
          return false;
        }
        return true;
      };
      syncNative();
      hidden.addEventListener("change", () => syncNative());
      const pushNative = () => {
        const raw = (vis.value || "").trim();
        const min = getMin();
        const max = getMax();
        hidden.value = raw ? clampIso(raw, min, max) : "";
        hidden.dispatchEvent(new Event("change", { bubbles: true }));
        syncNative();
      };
      vis.addEventListener("change", pushNative);
      vis.addEventListener("input", pushNative);
      calBtn.hidden = true;
      calBtn.setAttribute("aria-hidden", "true");
      if (clearBtn instanceof HTMLButtonElement && clearable) {
        clearBtn.addEventListener("click", (e) => {
          e.preventDefault();
          e.stopPropagation();
          hidden.value = "";
          syncNative();
          hidden.dispatchEvent(new Event("change", { bubbles: true }));
        });
      }
      const formNative = wrap.closest("form");
      if (formNative) {
        formNative.addEventListener(
          "submit",
          (e) => {
            if (!validateNative()) {
              e.preventDefault();
              e.stopImmediatePropagation();
              vis.reportValidity();
              return;
            }
            hidden.dispatchEvent(new Event("change", { bubbles: true }));
          },
          true
        );
      }
      return;
    }
    const required = vis.required;
    const sync = () => {
      vis.value = isoToDisplay(hidden.value, mdy);
      vis.setCustomValidity("");
      if (clearBtn instanceof HTMLButtonElement) {
        const has = Boolean((hidden.value || "").trim());
        clearBtn.hidden = !clearable || !has;
        clearBtn.setAttribute("aria-hidden", has ? "false" : "true");
      }
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
    hidden.addEventListener("change", () => sync());
    if (clearBtn instanceof HTMLButtonElement && clearable) {
      clearBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        hidden.value = "";
        sync();
        hidden.dispatchEvent(new Event("change", { bubbles: true }));
      });
    }
    vis.readOnly = false;
    const openCal = () => {
      openDatePicker(vis, {
        value: hidden.value || "",
        min: getMin(),
        max: getMax(),
        mdy,
        clearable,
        onConfirm: (iso) => {
          hidden.value = clampIso(iso, getMin(), getMax());
          sync();
          hidden.dispatchEvent(new Event("change", { bubbles: true }));
        }
      });
    };
    calBtn.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      openCal();
    });
    vis.addEventListener("input", () => {
      const iso = parseDisplayToIso(vis.value.trim(), mdy);
      if (iso) {
        hidden.value = clampIso(iso, getMin(), getMax());
        hidden.dispatchEvent(new Event("change", { bubbles: true }));
      }
      if (typeof activeDatePickerSync === "function") activeDatePickerSync(vis.value);
    });
    vis.addEventListener("blur", () => {
      const iso = parseDisplayToIso(vis.value.trim(), mdy);
      if (iso) hidden.value = clampIso(iso, getMin(), getMax());
      else if (!(vis.value || "").trim()) hidden.value = "";
      sync();
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
    if (!isMobileLayout()) {
      const requiredNativeT = vis.required;
      vis.type = "time";
      vis.readOnly = false;
      vis.removeAttribute("placeholder");
      vis.step = "60";
      const syncNativeT = () => {
        const normalized = normalizeTimeHM(hidden.value);
        hidden.value = normalized;
        vis.value = normalized || "";
        vis.setCustomValidity("");
      };
      const validateNativeT = () => {
        vis.setCustomValidity("");
        if (requiredNativeT && !(hidden.value || "").trim()) {
          vis.setCustomValidity("Select a time");
          return false;
        }
        return true;
      };
      syncNativeT();
      hidden.addEventListener("change", () => syncNativeT());
      const pushNativeT = () => {
        const raw = (vis.value || "").trim();
        hidden.value = raw ? normalizeTimeHM(raw) : "";
        hidden.dispatchEvent(new Event("change", { bubbles: true }));
        syncNativeT();
      };
      vis.addEventListener("change", pushNativeT);
      vis.addEventListener("input", pushNativeT);
      const clockBtnNative = wrap.querySelector(".remi-time-clock-btn");
      if (clockBtnNative instanceof HTMLElement) {
        clockBtnNative.hidden = true;
        clockBtnNative.setAttribute("aria-hidden", "true");
      }
      const formNativeT = wrap.closest("form");
      if (formNativeT) {
        formNativeT.addEventListener(
          "submit",
          (e) => {
            if (!validateNativeT()) {
              e.preventDefault();
              e.stopImmediatePropagation();
              vis.reportValidity();
              return;
            }
            hidden.dispatchEvent(new Event("change", { bubbles: true }));
          },
          true
        );
      }
      return;
    }
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
    hidden.addEventListener("change", () => sync());
    const clockBtn = wrap.querySelector(".remi-time-clock-btn");
    vis.readOnly = false;
    const openTime = () => {
      openTimePicker(vis, {
        title: inferPickerTitle(vis, "Set Time"),
        value: hidden.value || "08:30",
        onConfirm: (hm) => {
          hidden.value = normalizeTimeHM(hm);
          sync();
          hidden.dispatchEvent(new Event("change", { bubbles: true }));
        }
      });
    };
    if (clockBtn instanceof HTMLButtonElement) {
      clockBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        openTime();
      });
    } else {
      makeFieldOpenable(vis, openTime);
    }
    vis.addEventListener("input", () => {
      const hm = normalizeTimeHM(vis.value.trim());
      if (hm) {
        hidden.value = hm;
        hidden.dispatchEvent(new Event("change", { bubbles: true }));
      }
      if (typeof activeTimePickerSync === "function") activeTimePickerSync(vis.value);
    });
    vis.addEventListener("blur", () => {
      const hm = normalizeTimeHM(vis.value.trim());
      if (hm) hidden.value = hm;
      else if (!(vis.value || "").trim()) hidden.value = "";
      sync();
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
    /** Empty trip dates in templates can yield attrs like "T00:00" — ignore unless YYYY-MM-DD prefix is valid. */
    const tripRangeDateIso = (attr) => {
      const s = String(attr || "").trim();
      if (s.length < 10) return "";
      const head = s.slice(0, 10);
      return /^\d{4}-\d{2}-\d{2}$/.test(head) ? head : "";
    };
    const mdy = wrap.getAttribute("data-mdy") === "1";
    const min = wrap.getAttribute("data-min") || "";
    const max = wrap.getAttribute("data-max") || "";
    const uiTime24h = wrap.getAttribute("data-time-24h") === "1";
    const hidden = wrap.querySelector(".remi-datetime-iso");
    const combinedVis = wrap.querySelector(".remi-datetime-combined-part");
    const desktopCalBtn = wrap.querySelector(".remi-datetime-desktop-calendar-btn");
    const dateVis = wrap.querySelector(".remi-datetime-date-part");
    const timeVis = wrap.querySelector(".remi-datetime-time-part");
    if (!(hidden instanceof HTMLInputElement)) return;
    const hasDesktopCombined = combinedVis instanceof HTMLInputElement;
    const hasMobileParts = dateVis instanceof HTMLInputElement && timeVis instanceof HTMLInputElement;
    if (!hasDesktopCombined && !hasMobileParts) return;
    const required =
      (combinedVis instanceof HTMLInputElement && combinedVis.required) ||
      (dateVis instanceof HTMLInputElement && dateVis.required) ||
      (timeVis instanceof HTMLInputElement && timeVis.required);
    const formForWrap = wrap.closest("form");
    const formAction = formForWrap instanceof HTMLFormElement ? (formForWrap.getAttribute("action") || "") : "";
    const isTripBoundDateTime = /\/trips(\/|$)/i.test(formAction);
    const minDateIso = tripRangeDateIso(min);
    const maxDateIso = tripRangeDateIso(max);
    const rangeMessage =
      minDateIso && maxDateIso
        ? `This date is outside your trip (${isoToDisplay(minDateIso, mdy)} – ${isoToDisplay(maxDateIso, mdy)}).`
        : "This date is outside your trip.";
    const useRangeValidation = isTripBoundDateTime && Boolean(minDateIso && maxDateIso);
    let inlineErr = null;
    if (useRangeValidation) {
      inlineErr = wrap.querySelector(".remi-datetime-inline-error");
      if (!(inlineErr instanceof HTMLElement)) {
        inlineErr = document.createElement("div");
        inlineErr.className = "location-hint error remi-datetime-inline-error";
        inlineErr.hidden = true;
        wrap.appendChild(inlineErr);
      }
    }
    const setRangeError = (on) => {
      if (!useRangeValidation || !(inlineErr instanceof HTMLElement)) return;
      inlineErr.textContent = on ? rangeMessage : "";
      inlineErr.hidden = !on;
      wrap.classList.toggle("remi-datetime-field--has-error", Boolean(on));
    };
    const hasClearableDatetime = () => {
      if ((hidden.value || "").trim()) return true;
      if (combinedVis instanceof HTMLInputElement && (combinedVis.value || "").trim()) return true;
      if (dateVis instanceof HTMLInputElement && (dateVis.value || "").trim()) return true;
      if (timeVis instanceof HTMLInputElement && (timeVis.value || "").trim()) return true;
      return false;
    };
    const updateDatetimeClearButtons = () => {
      const has = hasClearableDatetime();
      wrap.querySelectorAll("[data-remi-datetime-clear]").forEach((el) => {
        if (el instanceof HTMLButtonElement) {
          el.disabled = !has;
          el.setAttribute("aria-disabled", has ? "false" : "true");
        }
      });
    };
    const isMobileLayout = () => window.matchMedia(`(max-width: ${MOBILE_BP}px)`).matches;
    const sync = () => {
      const sp = splitIsoLocalDateTime(hidden.value);
      if (dateVis instanceof HTMLInputElement) {
        dateVis.value = sp.dateIso ? isoToDisplay(sp.dateIso, mdy) : "";
        dateVis.setCustomValidity("");
      }
      if (timeVis instanceof HTMLInputElement) {
        timeVis.value = sp.time ? (uiTime24h ? timeHMToDisplay24(sp.time) : timeHMToDisplay(sp.time)) : "";
        timeVis.setCustomValidity("");
      }
      if (combinedVis instanceof HTMLInputElement) {
        combinedVis.value = dateTimeLocalToDisplay(hidden.value, mdy, uiTime24h);
        combinedVis.setCustomValidity("");
      }
      if (useRangeValidation) {
        const hasOutOfRange = sp.dateIso && sp.time && !isDateTimeWithinBounds(`${sp.dateIso}T${sp.time}`, min, max);
        setRangeError(Boolean(hasOutOfRange));
      }
      updateDatetimeClearButtons();
    };
    const syncModeDisabled = () => {
      const mobile = isMobileLayout();
      if (combinedVis instanceof HTMLInputElement) combinedVis.disabled = mobile;
      if (dateVis instanceof HTMLInputElement) dateVis.disabled = !mobile;
      if (timeVis instanceof HTMLInputElement) timeVis.disabled = !mobile;
    };
    const getVisibleDateIso = () => {
      if (dateVis instanceof HTMLInputElement) {
        const iso = parseDisplayToIso((dateVis.value || "").trim(), mdy);
        if (iso) return iso;
      }
      if (combinedVis instanceof HTMLInputElement) {
        const parsed = parseCombinedDateTimeDisplay(combinedVis.value, mdy, false);
        const sp = splitIsoLocalDateTime(parsed);
        if (sp.dateIso) return sp.dateIso;
      }
      const sp = splitIsoLocalDateTime(hidden.value);
      return sp.dateIso || "";
    };
    const getVisibleTimeHM = () => {
      if (timeVis instanceof HTMLInputElement) {
        const hm = normalizeTimeHM((timeVis.value || "").trim());
        if (hm) return hm;
      }
      if (combinedVis instanceof HTMLInputElement) {
        const parsed = parseCombinedDateTimeDisplay(combinedVis.value, mdy, false);
        const sp = splitIsoLocalDateTime(parsed);
        if (sp.time) return sp.time;
      }
      const sp = splitIsoLocalDateTime(hidden.value);
      return normalizeTimeHM(sp.time || "") || "";
    };
    const updateDate = (dateIso) => {
      const t = normalizeTimeHM(getVisibleTimeHM() || "08:30");
      const next = dateIso ? `${dateIso}T${t}` : "";
      if (useRangeValidation && next && !isDateTimeWithinBounds(next, min, max)) {
        setRangeError(true);
        return;
      }
      setRangeError(false);
      hidden.value = useRangeValidation ? next : clampDateTimeLocal(next, min, max);
      sync();
      hidden.dispatchEvent(new Event("change", { bubbles: true }));
    };
    const updateTime = (hm) => {
      let dateIso = getVisibleDateIso();
      if (!dateIso) {
        const now = new Date();
        dateIso = `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
      }
      const next = `${dateIso}T${normalizeTimeHM(hm)}`;
      if (useRangeValidation && !isDateTimeWithinBounds(next, min, max)) {
        setRangeError(true);
        return;
      }
      setRangeError(false);
      hidden.value = useRangeValidation ? next : clampDateTimeLocal(next, min, max);
      sync();
      hidden.dispatchEvent(new Event("change", { bubbles: true }));
    };
    const validate = () => {
      if (dateVis instanceof HTMLInputElement) dateVis.setCustomValidity("");
      if (timeVis instanceof HTMLInputElement) timeVis.setCustomValidity("");
      if (combinedVis instanceof HTMLInputElement) combinedVis.setCustomValidity("");
      setRangeError(false);
      if (!isMobileLayout() && combinedVis instanceof HTMLInputElement) {
        const raw = (combinedVis.value || "").trim();
        if (raw) {
          const parsed = parseCombinedDateTimeDisplay(raw, mdy, false);
          if (parsed) {
            if (useRangeValidation) {
              if (!isDateTimeWithinBounds(parsed, min, max)) {
                combinedVis.setCustomValidity(rangeMessage);
                setRangeError(true);
                return false;
              }
              hidden.value = parsed;
            } else {
              hidden.value = clampDateTimeLocal(parsed, min, max);
            }
          } else {
            combinedVis.setCustomValidity("Enter date and time");
            return false;
          }
        }
      }
      if (!required) return true;
      const sp = splitIsoLocalDateTime(hidden.value);
      if (isMobileLayout()) {
        if (!sp.dateIso && dateVis instanceof HTMLInputElement) dateVis.setCustomValidity("Select a date");
        if (!sp.time && timeVis instanceof HTMLInputElement) timeVis.setCustomValidity("Select a time");
      } else if ((!sp.dateIso || !sp.time) && combinedVis instanceof HTMLInputElement) {
        combinedVis.setCustomValidity("Enter date and time");
      }
      return !!(sp.dateIso && sp.time);
    };
    wrap.querySelectorAll("[data-remi-datetime-clear]").forEach((el) => {
      if (!(el instanceof HTMLButtonElement)) return;
      el.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        hidden.value = "";
        setRangeError(false);
        if (combinedVis instanceof HTMLInputElement) combinedVis.setCustomValidity("");
        if (dateVis instanceof HTMLInputElement) dateVis.setCustomValidity("");
        if (timeVis instanceof HTMLInputElement) timeVis.setCustomValidity("");
        sync();
        hidden.dispatchEvent(new Event("change", { bubbles: true }));
      });
    });
    sync();
    syncModeDisabled();
    window.addEventListener("resize", syncModeDisabled);
    if (dateVis instanceof HTMLInputElement) dateVis.readOnly = false;
    if (timeVis instanceof HTMLInputElement) timeVis.readOnly = false;
    if (combinedVis instanceof HTMLInputElement) combinedVis.readOnly = false;
    if (combinedVis instanceof HTMLInputElement) {
      let lastCombinedDigitCount = -1;
      let lastCombinedValueLen = -1;
      combinedVis.addEventListener("focus", () => {
        lastCombinedDigitCount = digitsOnly(combinedVis.value).length;
        lastCombinedValueLen = combinedVis.value.length;
      });
      combinedVis.addEventListener("beforeinput", (ev) => {
        if (isMobileLayout()) return;
        const t = String(ev.inputType || "");
        if (
          t === "deleteContentBackward" ||
          t === "deleteContentForward" ||
          t === "deleteByCut" ||
          t === "deleteCompositionText"
        ) {
          combinedVis.dataset.remiDatetimeDelete = "1";
        }
      });
      /*
       * Acceptance guardrails for combined desktop datetime typing:
       * 1) Full text entry must remain deletable from the right via backspace.
       * 2) Explicit AM/PM intent (A/P keys) must persist and not be auto-reverted.
       * 3) In 12h UI, explicit 24h entry like 14:34/1434 canonicalizes to 02:00 PM.
       */
      combinedVis.addEventListener("input", (ev) => {
        if (isMobileLayout()) return;
        try {
          const pos0 = combinedVis.selectionStart ?? combinedVis.value.length;
          const vlen = combinedVis.value.length;
          const dcnt = digitsOnly(combinedVis.value).length;
          const fromBeforeInput = combinedVis.dataset.remiDatetimeDelete === "1";
          if (fromBeforeInput) delete combinedVis.dataset.remiDatetimeDelete;
          const isDel =
            fromBeforeInput ||
            ev.inputType === "deleteContentBackward" ||
            ev.inputType === "deleteContentForward" ||
            ev.inputType === "deleteByCut" ||
            ev.inputType === "deleteCompositionText" ||
            (lastCombinedDigitCount >= 0 && dcnt < lastCombinedDigitCount) ||
            (lastCombinedValueLen >= 0 && vlen < lastCombinedValueLen);
          const masked = remiFormatCombinedDateTimeTypingMask(combinedVis.value, mdy, pos0, isDel);
          if (masked.text !== combinedVis.value) {
            combinedVis.value = masked.text;
            combinedVis.setSelectionRange(masked.caret, masked.caret);
          }
          lastCombinedDigitCount = digitsOnly(combinedVis.value).length;
          lastCombinedValueLen = combinedVis.value.length;
          let isoDt = parseCombinedDateTimeDisplay(combinedVis.value, mdy, true);
          if (isoDt) {
            const { timePart } = splitCombinedDisplayDateTimeParts(combinedVis.value, mdy);
            if (!uiTime24h) {
              const spTyped = splitIsoLocalDateTime(isoDt);
              const hmAdj = maybeForceTopOfHourForTyped24h(timePart, spTyped.time);
              if (spTyped.dateIso && hmAdj) isoDt = `${spTyped.dateIso}T${hmAdj}`;
            }
            if (useRangeValidation && !isDateTimeWithinBounds(isoDt, min, max)) {
              setRangeError(true);
              return;
            }
            setRangeError(false);
            const nextIso = useRangeValidation ? isoDt : clampDateTimeLocal(isoDt, min, max);
            hidden.value = nextIso;
            hidden.dispatchEvent(new Event("change", { bubbles: true }));
            /* Never sync() here while focused: if clamped !== isoDt (outside trip min/max), sync() would
             * replace the visible text with the clamped value and feel like the edit "reverted". Blur still normalizes. */
            /* Do not canonicalize during delete; otherwise trailing tokens can reappear and block backspace flow. */
            if (nextIso === isoDt && !uiTime24h && !isDel) {
              if (combinedTimeNeeds12hCanonVisual(timePart)) {
                const canon = dateTimeLocalToDisplay(nextIso, mdy, uiTime24h);
                if (canon && canon.replace(/\s+/g, " ").trim() !== combinedVis.value.replace(/\s+/g, " ").trim()) {
                  combinedVis.value = canon;
                  combinedVis.setSelectionRange(canon.length, canon.length);
                }
              }
            }
          }
        } finally {
          updateDatetimeClearButtons();
        }
      });
      combinedVis.addEventListener("keydown", (ev) => {
        if (isMobileLayout() || uiTime24h) return;
        const k = String(ev.key || "");
        const lower = k.toLowerCase();
        const explicitPeriod = lower === "a" ? "am" : lower === "p" ? "pm" : "";
        if (!explicitPeriod) return;
        const raw = String(combinedVis.value || "");
        const parts = splitCombinedDisplayDateTimeParts(raw, mdy);
        const datePart = String(parts.datePart || "").trim();
        const timePart = String(parts.timePart || "").trim();
        if (!datePart || !timePart) return;
        const hm = normalizeTimeHM(timePart);
        if (!hm) return;
        const nextHm = setHMPeriod(hm, explicitPeriod);
        if (!nextHm) return;
        const nextTimeDisp = timeHMToDisplay(nextHm);
        const nextText = `${datePart}, ${nextTimeDisp}`;
        combinedVis.value = nextText;
        combinedVis.setSelectionRange(nextText.length, nextText.length);
        lastCombinedDigitCount = digitsOnly(nextText).length;
        lastCombinedValueLen = nextText.length;
        const parsed = parseCombinedDateTimeDisplay(nextText, mdy, false);
        if (parsed) {
          if (useRangeValidation && !isDateTimeWithinBounds(parsed, min, max)) {
            setRangeError(true);
            ev.preventDefault();
            return;
          }
          setRangeError(false);
          const nextIso = useRangeValidation ? parsed : clampDateTimeLocal(parsed, min, max);
          if (nextIso === parsed) {
            hidden.value = nextIso;
            hidden.dispatchEvent(new Event("change", { bubbles: true }));
          }
        }
        ev.preventDefault();
      });
      combinedVis.addEventListener("blur", () => {
        if (isMobileLayout()) return;
        combinedVis.setCustomValidity("");
        const raw = (combinedVis.value || "").trim();
        if (!raw) {
          setRangeError(false);
          hidden.value = "";
          hidden.dispatchEvent(new Event("change", { bubbles: true }));
          sync();
          return;
        }
        const isoDt = parseCombinedDateTimeDisplay(raw, mdy);
        if (isoDt) {
          if (useRangeValidation) {
            if (!isDateTimeWithinBounds(isoDt, min, max)) {
              combinedVis.setCustomValidity(rangeMessage);
              setRangeError(true);
              return;
            }
            setRangeError(false);
            hidden.value = isoDt;
          } else {
            hidden.value = clampDateTimeLocal(isoDt, min, max);
          }
          hidden.dispatchEvent(new Event("change", { bubbles: true }));
          sync();
          return;
        }
        setRangeError(false);
        combinedVis.setCustomValidity("Enter date and time");
      });
    }
    if (dateVis instanceof HTMLInputElement) {
      let lastDateDigitCount = -1;
      let lastDateValueLen = -1;
      dateVis.addEventListener("focus", () => {
        lastDateDigitCount = digitsOnly(dateVis.value).length;
        lastDateValueLen = dateVis.value.length;
      });
      dateVis.addEventListener("beforeinput", (ev) => {
        if (!isMobileLayout()) return;
        const t = String(ev.inputType || "");
        if (
          t === "deleteContentBackward" ||
          t === "deleteContentForward" ||
          t === "deleteByCut" ||
          t === "deleteCompositionText"
        ) {
          dateVis.dataset.remiDatetimeDelete = "1";
        }
      });
      dateVis.addEventListener("input", (ev) => {
        if (!isMobileLayout()) return;
        try {
          const p0 = dateVis.selectionStart ?? dateVis.value.length;
          const vlen = dateVis.value.length;
          const dcnt = digitsOnly(dateVis.value).length;
          const fromBeforeInput = dateVis.dataset.remiDatetimeDelete === "1";
          if (fromBeforeInput) delete dateVis.dataset.remiDatetimeDelete;
          const isDel =
            fromBeforeInput ||
            ev.inputType === "deleteContentBackward" ||
            ev.inputType === "deleteContentForward" ||
            ev.inputType === "deleteByCut" ||
            ev.inputType === "deleteCompositionText" ||
            (lastDateDigitCount >= 0 && dcnt < lastDateDigitCount) ||
            (lastDateValueLen >= 0 && vlen < lastDateValueLen);
          const dMasked = remiFormatDatePartTypingMask(dateVis.value, p0, isDel);
          if (dMasked.text !== dateVis.value) {
            dateVis.value = dMasked.text;
            dateVis.setSelectionRange(dMasked.caret, dMasked.caret);
          }
          lastDateDigitCount = digitsOnly(dateVis.value).length;
          lastDateValueLen = dateVis.value.length;
          const iso = parseDisplayToIso(dateVis.value.trim(), mdy);
          if (iso) {
            const sp = splitIsoLocalDateTime(hidden.value);
            const tPart = normalizeTimeHM(sp.time || "08:30");
            hidden.value = clampDateTimeLocal(`${iso}T${tPart}`, min, max);
            hidden.dispatchEvent(new Event("change", { bubbles: true }));
          }
          if (typeof activeDatePickerSync === "function") activeDatePickerSync(dateVis.value);
        } finally {
          updateDatetimeClearButtons();
        }
      });
      dateVis.addEventListener("blur", () => {
        if (!isMobileLayout()) return;
        const iso = parseDisplayToIso(dateVis.value.trim(), mdy);
        if (iso) {
          const sp = splitIsoLocalDateTime(hidden.value);
          const tPart = normalizeTimeHM(sp.time || "08:30");
          hidden.value = clampDateTimeLocal(`${iso}T${tPart}`, min, max);
        } else if (!(dateVis.value || "").trim()) {
          updateDate("");
        }
        sync();
      });
    }
    if (timeVis instanceof HTMLInputElement) {
      let lastTimeDigitCount = -1;
      let lastTimeValueLen = -1;
      timeVis.addEventListener("focus", () => {
        lastTimeDigitCount = digitsOnly(timeVis.value).length;
        lastTimeValueLen = timeVis.value.length;
      });
      timeVis.addEventListener("beforeinput", (ev) => {
        if (!isMobileLayout()) return;
        const t = String(ev.inputType || "");
        if (
          t === "deleteContentBackward" ||
          t === "deleteContentForward" ||
          t === "deleteByCut" ||
          t === "deleteCompositionText"
        ) {
          timeVis.dataset.remiDatetimeDelete = "1";
        }
      });
      /* Same normalization rules as combined field, but for mobile split time input. */
      timeVis.addEventListener("input", (ev) => {
        if (!isMobileLayout()) return;
        try {
          const t0 = timeVis.selectionStart ?? timeVis.value.length;
          const vlen = timeVis.value.length;
          const dcnt = digitsOnly(timeVis.value).length;
          const fromBeforeInput = timeVis.dataset.remiDatetimeDelete === "1";
          if (fromBeforeInput) delete timeVis.dataset.remiDatetimeDelete;
          const isDel =
            fromBeforeInput ||
            ev.inputType === "deleteContentBackward" ||
            ev.inputType === "deleteContentForward" ||
            ev.inputType === "deleteByCut" ||
            ev.inputType === "deleteCompositionText" ||
            (lastTimeDigitCount >= 0 && dcnt < lastTimeDigitCount) ||
            (lastTimeValueLen >= 0 && vlen < lastTimeValueLen);
          const tMasked = remiFormatTimePartTypingMask(timeVis.value, t0, isDel);
          if (tMasked.text !== timeVis.value) {
            timeVis.value = tMasked.text;
            timeVis.setSelectionRange(tMasked.caret, tMasked.caret);
          }
          lastTimeDigitCount = digitsOnly(timeVis.value).length;
          lastTimeValueLen = timeVis.value.length;
          const tv = timeVis.value.trim();
          if (isTimePartIncompleteForCombinedInput(tv)) {
            if (typeof activeTimePickerSync === "function") activeTimePickerSync(timeVis.value);
            return;
          }
          const hm = normalizeTimeHM(tv);
          if (hm) {
            const hmAdj = uiTime24h ? hm : maybeForceTopOfHourForTyped24h(tv, hm);
            if (hmAdj) updateTime(hmAdj);
          }
          if (typeof activeTimePickerSync === "function") activeTimePickerSync(timeVis.value);
        } finally {
          updateDatetimeClearButtons();
        }
      });
      timeVis.addEventListener("keydown", (ev) => {
        if (!isMobileLayout() || uiTime24h) return;
        const k = String(ev.key || "");
        const lower = k.toLowerCase();
        const explicitPeriod = lower === "a" ? "am" : lower === "p" ? "pm" : "";
        if (!explicitPeriod) return;
        const raw = String(timeVis.value || "").trim();
        if (isTimePartIncompleteForCombinedInput(raw)) return;
        const hm = normalizeTimeHM(raw);
        if (!hm) return;
        const nextHm = setHMPeriod(hm, explicitPeriod);
        if (!nextHm) return;
        const nextDisp = timeHMToDisplay(nextHm);
        timeVis.value = nextDisp;
        const dateIso = getVisibleDateIso();
        if (dateIso) {
          const nextIsoRaw = `${dateIso}T${nextHm}`;
          const nextIso = clampDateTimeLocal(nextIsoRaw, min, max);
          if (nextIso === nextIsoRaw) {
            hidden.value = nextIso;
            hidden.dispatchEvent(new Event("change", { bubbles: true }));
          }
        }
        ev.preventDefault();
      });
      timeVis.addEventListener("blur", () => {
        if (!isMobileLayout()) return;
        const hm = normalizeTimeHM(timeVis.value.trim());
        if (hm) updateTime(hm);
        else sync();
      });
    }
    const dateCalBtn = wrap.querySelector(".remi-datetime-date-wrap .remi-date-calendar-btn");
    const openDateCal = () => {
      const dateIso = getVisibleDateIso();
      /* Desktop: OS/browser native date picker (no custom "Select Date" modal). */
      if (!isMobileLayout()) {
        const desktopWrap = wrap.querySelector(".remi-datetime-desktop-wrap");
        if (!(desktopWrap instanceof HTMLElement)) return;
        let nd = wrap.querySelector("input.remi-datetime-native-date");
        if (!nd) {
          nd = document.createElement("input");
          nd.type = "date";
          nd.className = "remi-datetime-native-date";
          nd.setAttribute("aria-hidden", "true");
          nd.tabIndex = -1;
          desktopWrap.appendChild(nd);
          nd.addEventListener("change", () => {
            const v = nd.value;
            if (v) updateDate(v);
          });
        }
        const dmin = tripRangeDateIso(min);
        const dmax = tripRangeDateIso(max);
        nd.min = dmin || "";
        nd.max = dmax || "";
        nd.value = dateIso || "";
        requestAnimationFrame(() => {
          if (typeof nd.showPicker === "function") {
            nd.showPicker().catch(() => {
              nd.focus();
            });
          } else {
            nd.focus();
          }
        });
        return;
      }
      const anchor = !isMobileLayout() && combinedVis instanceof HTMLInputElement ? combinedVis : dateVis;
      if (!(anchor instanceof HTMLInputElement)) return;
      openDatePicker(anchor, {
        value: dateIso,
        min: tripRangeDateIso(min),
        max: tripRangeDateIso(max),
        mdy,
        onConfirm: (iso) => updateDate(iso)
      });
    };
    if (desktopCalBtn instanceof HTMLButtonElement) {
      desktopCalBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        openDateCal();
      });
    }
    if (dateCalBtn instanceof HTMLButtonElement) {
      dateCalBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        openDateCal();
      });
    } else {
      if (dateVis instanceof HTMLInputElement) makeFieldOpenable(dateVis, openDateCal);
    }
    hidden.addEventListener("change", () => {
      if (combinedVis instanceof HTMLInputElement && document.activeElement === combinedVis) return;
      sync();
    });
    const timeClockBtn = wrap.querySelector(".remi-datetime-time-wrap .remi-time-clock-btn");
    const openTimeCal = () => {
      if (!(timeVis instanceof HTMLInputElement)) return;
      const hm = getVisibleTimeHM() || "08:30";
      openTimePicker(timeVis, {
        value: hm,
        title: inferPickerTitle(timeVis, "Set Time"),
        onConfirm: (hm) => updateTime(hm)
      });
    };
    if (timeClockBtn instanceof HTMLButtonElement) {
      timeClockBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        openTimeCal();
      });
    } else {
      if (timeVis instanceof HTMLInputElement) makeFieldOpenable(timeVis, openTimeCal);
    }
    const form = wrap.closest("form");
    if (form) {
      form.addEventListener("submit", (e) => {
        if (!validate()) {
          e.preventDefault();
          if (isMobileLayout()) {
            if (dateVis instanceof HTMLInputElement) dateVis.reportValidity();
            if (timeVis instanceof HTMLInputElement) timeVis.reportValidity();
          } else if (combinedVis instanceof HTMLInputElement) {
            combinedVis.reportValidity();
          }
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

const AIRLABS_KEY_MIN_LEN = 8;

const initSiteSettingsAirLabsKeySection = (mapForm) => {
  const section = mapForm.querySelector("[data-airlabs-key-section]");
  if (!(section instanceof HTMLElement)) return;

  const editWrap = section.querySelector("[data-airlabs-key-edit]");
  const savedCard = section.querySelector("[data-airlabs-key-saved]");
  const textInput = section.querySelector("[data-airlabs-key-input]");
  const submitHidden = section.querySelector("[data-airlabs-key-submit]");
  const clearCheckbox = section.querySelector("[data-airlabs-clear-checkbox]");
  const maskEl = section.querySelector("[data-airlabs-key-mask]");
  const validationEl = section.querySelector("[data-airlabs-key-validation]");
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
    if (v.length >= 16) {
      validationEl.textContent = "Length looks reasonable for an AirLabs API key.";
      validationEl.classList.remove("gmaps-key-validation--warn");
      validationEl.classList.add("gmaps-key-validation--ok");
    } else {
      validationEl.textContent =
        "AirLabs keys are often longer. Double-check the value from your AirLabs dashboard before saving.";
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
        title: "Delete AirLabs API key?",
        body: "Flight airport suggestions will use your saved cache and map location providers only. Save the form to apply this change.",
        okText: "Delete key",
        icon: "delete_forever",
        variant: "danger",
        onConfirm: onDeleteKeyConfirmed
      });
      return;
    }
    if (
      window.confirm(
        "Delete the AirLabs API key? Flight airport suggestions will use your saved cache and map location providers only. Save the form to apply."
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
      const updateBtn = t.closest("[data-airlabs-key-update]");
      if (updateBtn && section.contains(updateBtn)) {
        e.preventDefault();
        onUpdateKey();
        return;
      }
      const deleteBtn = t.closest("[data-airlabs-key-delete]");
      if (deleteBtn && section.contains(deleteBtn)) {
        e.preventDefault();
        openDeleteConfirm();
        return;
      }
      const copyBtn = t.closest("[data-airlabs-key-copy]");
      if (copyBtn && section.contains(copyBtn)) {
        e.preventDefault();
        const toCopy = savedKeyForCopy.trim();
        if (!toCopy) return;
        void (async () => {
          const toast = typeof window.remiShowToast === "function" ? window.remiShowToast : null;
          const ok = await copyTextToClipboard(toCopy);
          if (ok) {
            toast?.("AirLabs API key copied.");
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
      if (draft !== "" && draft.length < AIRLABS_KEY_MIN_LEN) {
        e.preventDefault();
        if (typeof window.remiShowToast === "function") {
          window.remiShowToast(
            `That value is too short to be a valid AirLabs key (minimum ${AIRLABS_KEY_MIN_LEN} characters). Fix it or delete the key.`
          );
        }
        textInput.focus();
        updateValidationHint();
      }
    },
    true
  );
};

const OPENWEATHER_KEY_MIN_LEN = 8;

const initSiteSettingsOpenWeatherKeySection = (mapForm) => {
  const section = mapForm.querySelector("[data-openweather-key-section]");
  if (!(section instanceof HTMLElement)) return;

  const editWrap = section.querySelector("[data-openweather-key-edit]");
  const savedCard = section.querySelector("[data-openweather-key-saved]");
  const textInput = section.querySelector("[data-openweather-key-input]");
  const submitHidden = section.querySelector("[data-openweather-key-submit]");
  const clearCheckbox = section.querySelector("[data-openweather-clear-checkbox]");
  const maskEl =
    section.querySelector("[data-openweather-key-mask]") ||
    section.querySelector("[data-openweather-key-masked]");
  const validationEl = section.querySelector("[data-openweather-key-validation]");
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
    if (v.length >= 20) {
      validationEl.textContent = "Length looks reasonable for an OpenWeatherMap API key.";
      validationEl.classList.remove("gmaps-key-validation--warn");
      validationEl.classList.add("gmaps-key-validation--ok");
    } else {
      validationEl.textContent =
        "OpenWeather keys are often 32 characters. Check your OpenWeather dashboard.";
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
        title: "Delete OpenWeatherMap API key?",
        body: "Stop weather previews on trip pages will be hidden. Save the form to apply this change.",
        okText: "Delete key",
        icon: "delete_forever",
        variant: "danger",
        onConfirm: onDeleteKeyConfirmed
      });
      return;
    }
    if (window.confirm("Delete the OpenWeatherMap API key? Save the form to apply.")) {
      onDeleteKeyConfirmed();
    }
  };

  mapForm.addEventListener(
    "click",
    (e) => {
      const t = e.target;
      if (!(t instanceof Element)) return;
      const updateBtn = t.closest("[data-openweather-key-update]");
      if (updateBtn && section.contains(updateBtn)) {
        e.preventDefault();
        onUpdateKey();
        return;
      }
      const deleteBtn = t.closest("[data-openweather-key-delete]");
      if (deleteBtn && section.contains(deleteBtn)) {
        e.preventDefault();
        openDeleteConfirm();
        return;
      }
      const copyBtn = t.closest("[data-openweather-key-copy]");
      if (copyBtn && section.contains(copyBtn)) {
        e.preventDefault();
        const toCopy = savedKeyForCopy.trim();
        if (!toCopy) return;
        void (async () => {
          const toast = typeof window.remiShowToast === "function" ? window.remiShowToast : null;
          const ok = await copyTextToClipboard(toCopy);
          if (ok) {
            toast?.("OpenWeatherMap API key copied.");
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
      if (draft !== "" && draft.length < OPENWEATHER_KEY_MIN_LEN) {
        e.preventDefault();
        if (typeof window.remiShowToast === "function") {
          window.remiShowToast(
            `That value is too short to be a valid OpenWeather key (minimum ${OPENWEATHER_KEY_MIN_LEN} characters). Fix it or delete the key.`
          );
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
    if (sessionStorage.getItem("remi_focus_budget_expense_title")) {
      sessionStorage.removeItem("remi_focus_budget_expense_title");
      window.requestAnimationFrame(() => {
        document.getElementById("add-expense")?.scrollIntoView({ block: "nearest", behavior: "smooth" });
        const el = document.querySelector("#add-expense input[name='title']");
        if (el instanceof HTMLElement) el.focus();
      });
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
  const inviteLinkValueByTrip = new Map();
  const inviteLinkStorageKey = (tripId) => `remiInviteLink:${String(tripId || "").trim()}`;
  const readInviteLinkCache = (tripId) => {
    const key = inviteLinkStorageKey(tripId);
    if (!key || typeof window.sessionStorage === "undefined") return "";
    try {
      return String(window.sessionStorage.getItem(key) || "").trim();
    } catch (e) {
      return "";
    }
  };
  const writeInviteLinkCache = (tripId, url) => {
    const key = inviteLinkStorageKey(tripId);
    const value = String(url || "").trim();
    if (!key || !value || typeof window.sessionStorage === "undefined") return;
    try {
      window.sessionStorage.setItem(key, value);
    } catch (e) {
      /* ignore cache failures */
    }
  };
  const collectInviteRootsByTrip = (scope) => {
    const rootMap = new Map();
    const base =
      scope instanceof HTMLElement || scope instanceof Document ? scope : document;
    base.querySelectorAll("[data-trip-invite-methods]").forEach((root) => {
      const tripId = String(root.getAttribute("data-trip-id") || "").trim();
      if (!tripId) return;
      if (!rootMap.has(tripId)) rootMap.set(tripId, []);
      rootMap.get(tripId).push(root);
    });
    return rootMap;
  };
  const initTripInviteMethodsIn = (scope = document) => {
    const inviteRootsByTrip = collectInviteRootsByTrip(scope);
    inviteRootsByTrip.forEach((roots, tripId) => {
      const csrf = String(roots[0]?.getAttribute("data-csrf") || "").trim();
      if (!csrf) return;
      const allLinkInputs = () =>
        roots
          .map((r) => r.querySelector(".sidebar-invite-link-url"))
          .filter((input) => input instanceof HTMLInputElement);
      const applyInviteLinkValue = (url) => {
        const value = String(url || "").trim();
        if (!value) return;
        inviteLinkValueByTrip.set(tripId, value);
        writeInviteLinkCache(tripId, value);
        allLinkInputs().forEach((input) => {
          input.value = value;
        });
      };
      const cachedUrl =
        inviteLinkValueByTrip.get(tripId) || readInviteLinkCache(tripId);
      if (cachedUrl) {
        applyInviteLinkValue(cachedUrl);
      }
      const requestInviteLink = () => {
        const existingUrl =
          inviteLinkValueByTrip.get(tripId) || readInviteLinkCache(tripId);
        if (existingUrl) {
          applyInviteLinkValue(existingUrl);
          return Promise.resolve(existingUrl);
        }
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
            const url = String(data?.url || "").trim();
            if (url) {
              applyInviteLinkValue(url);
            }
            return url;
          })
          .catch(() => {
            inviteLinkPromiseByTrip.delete(tripId);
            if (typeof window.remiShowToast === "function") {
              window.remiShowToast("Could not create invite link. Try again.");
            }
            return "";
          });
        inviteLinkPromiseByTrip.set(tripId, p);
        return p;
      };

      roots.forEach((root) => {
        if (!(root instanceof HTMLElement)) return;
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
            const knownUrl =
              inviteLinkValueByTrip.get(tripId) || readInviteLinkCache(tripId);
            if (knownUrl) {
              applyInviteLinkValue(knownUrl);
              return;
            }
            requestInviteLink();
          }
        };

        if (root.dataset.remiInviteMethodsBound !== "1") {
          tabs.forEach((tab) => {
            tab.addEventListener("click", () => {
              showPanel(tab.getAttribute("data-invite-tab") || "email");
            });
          });

          const copyInviteLink = () => {
            if (!(linkInput instanceof HTMLInputElement)) return;
            const v = (linkInput.value || "").trim();
            if (!v) {
              requestInviteLink().then((freshUrl) => {
                if (freshUrl) {
                  try {
                    navigator.clipboard.writeText(freshUrl);
                    if (typeof window.remiShowToast === "function") {
                      window.remiShowToast("Invite link copied.");
                    }
                  } catch (e) {
                    if (typeof window.remiShowToast === "function") {
                      window.remiShowToast("Could not copy. Select the URL and copy manually.");
                    }
                  }
                }
              });
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
          root.dataset.remiInviteMethodsBound = "1";
        }

        if (tabs.length === 0) {
          showPanel("link");
        }
      });
    });
  };
  window.remiInitTripInviteMethodsIn = initTripInviteMethodsIn;
  initTripInviteMethodsIn(document);

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
  const remiFlightAirportSuggestURL = () => {
    const shell = document.querySelector("main.app-shell");
    const u = shell && shell.getAttribute("data-flight-airport-suggest-url");
    return (u && u.trim()) || "/api/flight-airports/suggest";
  };
  const remiFlightAirlineSuggestURL = () => {
    const shell = document.querySelector("main.app-shell");
    const u = shell && shell.getAttribute("data-flight-airline-suggest-url");
    return (u && u.trim()) || "/api/flight-airlines/suggest";
  };
  const remiGeocodeURL = () => "/api/location/geocode";
  const remiPlaceDetailsURL = () => "/api/location/place-details";
  const remiPlaceDetailsClientCache = new Map();
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

  const fetchPlaceDetailsForItinerary = async (placeId) => {
    const id = String(placeId || "").trim();
    if (!id) return null;
    if (remiPlaceDetailsClientCache.has(id)) return remiPlaceDetailsClientCache.get(id);
    try {
      const res = await fetch(
        `${remiPlaceDetailsURL()}?place_id=${encodeURIComponent(id)}${remiLocationLangQuery()}`,
        { credentials: "same-origin", headers: { Accept: "application/json" } }
      );
      if (!res.ok) return null;
      const data = await res.json();
      remiPlaceDetailsClientCache.set(id, data);
      return data;
    } catch (e) {
      return null;
    }
  };

  const remiGooglePlacesTimeToMinutes = (t) => {
    const raw = String(t || "").replace(/\D/g, "");
    if (raw.length < 3) return null;
    const pad = raw.length <= 2 ? raw.padStart(4, "0") : raw.slice(-4).padStart(4, "0");
    const h = parseInt(pad.slice(0, 2), 10);
    const m = parseInt(pad.slice(2), 10);
    if (!Number.isFinite(h) || !Number.isFinite(m)) return null;
    return h * 60 + m;
  };

  const remiCollectSameDayIntervals = (periods, weekday) => {
    const out = [];
    if (!Array.isArray(periods)) return out;
    for (const p of periods) {
      if (!p || !p.open || !p.close) continue;
      if (p.open.day === weekday && p.close.day === weekday) {
        const a = remiGooglePlacesTimeToMinutes(p.open.time);
        const b = remiGooglePlacesTimeToMinutes(p.close.time);
        if (a != null && b != null && b > a) out.push([a, b]);
      }
    }
    return out;
  };

  const remiWeekDayFromYMD = (ymd) => {
    const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(String(ymd || ""));
    if (!m) return new Date().getDay();
    const y = parseInt(m[1], 10);
    const mo = parseInt(m[2], 10) - 1;
    const d = parseInt(m[3], 10);
    return new Date(y, mo, d).getDay();
  };

  const remiReadTripClock24h = () => {
    const m = document.querySelector("main.app-shell.trip-details-page");
    return Boolean(m && m.getAttribute("data-effective-clock-24h") === "1");
  };
  const remiFormatMinutesClock = (mins) => {
    if (mins == null || !Number.isFinite(mins)) return "";
    const h = Math.floor(mins / 60) % 24;
    const mm = Math.round(mins % 60);
    const dt = new Date();
    dt.setHours(h, mm, 0, 0);
    if (remiReadTripClock24h()) {
      return dt.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", hour12: false });
    }
    return dt.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit", hour12: true });
  };

  const remiSummarizeHoursFromPlaceDetail = (detail, ymd) => {
    const day = remiWeekDayFromYMD(ymd);
    const oh = detail && detail.openingHours;
    if (!oh) {
      return { chipLine: "Opening hours not available.", openMins: null, closeMins: null, status: "unavailable" };
    }
    const ivs = remiCollectSameDayIntervals(oh.periods, day);
    if (ivs.length) {
      const [a, b] = ivs[0];
      const chipLine = `${remiFormatMinutesClock(a)} – ${remiFormatMinutesClock(b)}`;
      return { chipLine, openMins: a, closeMins: b, status: "open" };
    }
    const wt = oh.weekday_text;
    if (Array.isArray(wt) && wt[day]) {
      const line = String(wt[day]);
      if (/closed/i.test(line)) {
        return { chipLine: "Opening hours not available.", openMins: null, closeMins: null, status: "closed" };
      }
      const rest = line.replace(/^[^:：]+[:：]\s*/, "").trim();
      if (!rest) {
        return { chipLine: "Opening hours not available.", openMins: null, closeMins: null, status: "unavailable" };
      }
      return { chipLine: rest, openMins: null, closeMins: null, status: "open" };
    }
    return { chipLine: "Opening hours not available.", openMins: null, closeMins: null, status: "unavailable" };
  };

  const remiIsoLocalMinutes = (iso) => {
    const s = String(iso || "").trim();
    if (!s) return null;
    const d = new Date(s);
    if (Number.isNaN(d.getTime())) return null;
    return d.getHours() * 60 + d.getMinutes();
  };

  const remiScheduleOutsideVenue = (detail, ymd, startIso, endIso) => {
    const oh = detail && detail.openingHours;
    if (!oh || !Array.isArray(oh.periods)) return false;
    const day = remiWeekDayFromYMD(ymd);
    const ivs = remiCollectSameDayIntervals(oh.periods, day);
    if (!ivs.length) return Boolean(String(startIso || "").trim());
    const sMin = remiIsoLocalMinutes(startIso);
    const eMin = remiIsoLocalMinutes(endIso);
    if (sMin == null) return false;
    if (eMin == null || eMin <= sMin) return !ivs.some(([a, b]) => sMin >= a && sMin <= b);
    return !ivs.some(([a, b]) => sMin >= a && eMin <= b);
  };

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

  const buildTimelineConnectorLi = (fromEl, toEl, from, to, distanceKm) => {
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

    const tripIdMatch = window.location.pathname.match(/\/trips\/([^/]+)/);
    const tripId = tripIdMatch ? tripIdMatch[1] : "";
    const fromId = (fromEl?.getAttribute("data-itinerary-item-id") || "").trim();
    const toId = (toEl?.getAttribute("data-itinerary-item-id") || "").trim();
    if (tripId && fromId && toId) {
      const addRow = document.createElement("div");
      addRow.className = "timeline-connector-add-commute";
      const addBtn = document.createElement("button");
      addBtn.type = "button";
      addBtn.className = "timeline-connector-add-commute-btn secondary-btn";
      addBtn.textContent = "Add commute here";
      addBtn.setAttribute("aria-label", "Add commute leg between these items");
      addBtn.addEventListener("click", (ev) => {
        ev.preventDefault();
        ev.stopPropagation();
        const form =
          document.getElementById("sidebar-add-commute-form") ||
          document.getElementById("mobile-add-commute-form");
        if (!form) {
          return;
        }
        const applyPair = () => {
          const setSel = (name, id) => {
            const sel = form.querySelector(`select[name='${name}']`);
            if (sel) sel.value = id;
          };
          setSel("commute_from_item_id", fromId);
          setSel("commute_to_item_id", toId);
          form.closest("details")?.setAttribute("open", "");
          form.scrollIntoView({ behavior: "smooth", block: "nearest" });
          const title = form.querySelector("input[name='title']");
          title?.focus?.();
        };
        const hostSheet = form.closest(".mobile-sheet");
        if (hostSheet instanceof HTMLElement && hostSheet.classList.contains("hidden")) {
          const sid = hostSheet.getAttribute("id");
          const opener = sid ? document.querySelector(`[data-mobile-sheet-open="${sid}"]`) : null;
          if (opener instanceof HTMLElement) opener.click();
          window.setTimeout(applyPair, 380);
          return;
        }
        applyPair();
      });
      addRow.append(addBtn);
      menu.append(addRow);
    }

    details.append(summary, menu);
    li.append(rail, details);
    return li;
  };

  const isCommuteTimelineItem = (el) =>
    (el?.getAttribute?.("data-item-kind") || el?.getAttribute?.("data-marker-kind") || "").trim().toLowerCase() ===
    "commute";

  const drawConnectors = (scopes) => {
    scopes.forEach((scope) => {
      scope.querySelectorAll(".day-items.timeline").forEach((list) => {
        list.querySelectorAll("[data-itinerary-connector]").forEach((el) => el.remove());
        const items = Array.from(list.querySelectorAll(".timeline-item[data-itinerary-item]"))
          .filter((el) => !el.classList.contains("itinerary-search-hidden"));
        for (let i = 0; i < items.length; i++) {
          const current = items[i];
          if (isCommuteTimelineItem(current)) continue;
          const from = parseCoords(current);
          if (!from) continue;
          const curId = (current.getAttribute("data-itinerary-item-id") || "").trim();
          for (let j = i + 1; j < items.length; j++) {
            const cand = items[j];
            if (isCommuteTimelineItem(cand)) continue;
            const to = parseCoords(cand);
            if (!to) break;
            const nextId = (cand.getAttribute("data-itinerary-item-id") || "").trim();
            let blocked = false;
            for (let k = i + 1; k < j; k++) {
              const between = items[k];
              if (!isCommuteTimelineItem(between)) continue;
              if (
                (between.getAttribute("data-commute-from") || "").trim() === curId &&
                (between.getAttribute("data-commute-to") || "").trim() === nextId
              ) {
                blocked = true;
                break;
              }
            }
            if (blocked) break;
            const distanceKm = haversineKm(from.lat, from.lng, to.lat, to.lng);
            if (!Number.isFinite(distanceKm) || distanceKm <= 0.05) break;
            cand.insertAdjacentElement("beforebegin", buildTimelineConnectorLi(current, cand, from, to, distanceKm));
            break;
          }
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

  let remiItineraryQuickEditEscapeAbort = null;
  const detachItineraryQuickEditEscape = () => {
    if (remiItineraryQuickEditEscapeAbort) {
      remiItineraryQuickEditEscapeAbort.abort();
      remiItineraryQuickEditEscapeAbort = null;
    }
  };
  const attachItineraryQuickEditEscape = (onEscape) => {
    detachItineraryQuickEditEscape();
    const ac = new AbortController();
    remiItineraryQuickEditEscapeAbort = ac;
    document.addEventListener(
      "keydown",
      (ev) => {
        if (ev.key !== "Escape") return;
        ev.preventDefault();
        onEscape();
      },
      { capture: true, signal: ac.signal }
    );
  };

  let tripCalendarRoot = document.querySelector("[data-itinerary-calendar-root]");
  if (tripCalendarRoot) {
    let listPanel = tripCalendarRoot.querySelector("[data-itinerary-list-panel]");
    let calendarPanels = tripCalendarRoot.querySelector("[data-itinerary-calendar-panels]");
    let dayView = tripCalendarRoot.querySelector("[data-itinerary-day-view]");
    let weekView = tripCalendarRoot.querySelector("[data-itinerary-week-view]");
    let toolbar = tripCalendarRoot.querySelector("[data-itinerary-calendar-toolbar]");
    let rangeLabel = tripCalendarRoot.querySelector("[data-itinerary-range-label]");
    let tripID = (tripCalendarRoot.getAttribute("data-trip-id") || "").trim();
    let tripStartRaw = (tripCalendarRoot.getAttribute("data-trip-start") || "").trim();
    let tripEndRaw = (tripCalendarRoot.getAttribute("data-trip-end") || "").trim();
    let quickEditModal = tripCalendarRoot.querySelector("[data-itinerary-quick-edit-modal]");
    let quickEditBody = tripCalendarRoot.querySelector("[data-itinerary-quick-edit-body]");
    let quickEditPreview = tripCalendarRoot.querySelector("[data-itinerary-quick-edit-preview]");
    let quickEditFormWrap = tripCalendarRoot.querySelector("[data-itinerary-quick-edit-form-wrap]");
    let quickEditPanel = tripCalendarRoot.querySelector("[data-itinerary-quick-edit-panel]");
    let quickEditTitle = tripCalendarRoot.querySelector("[data-itinerary-quick-edit-title]");
    let quickEditModeLabel = tripCalendarRoot.querySelector("[data-itinerary-quick-edit-mode-label]");
    const desktopCalFlyout = document.querySelector("[data-desktop-calendar-flyout]");
    const desktopCalFlyoutLabel = desktopCalFlyout?.querySelector("[data-desktop-calendar-flyout-label]");
    const quickAddState = { date: "", startMin: 8 * 60, lastClientX: null, lastClientY: null };
    if (listPanel && calendarPanels && dayView && weekView && toolbar && rangeLabel && tripStartRaw && tripEndRaw) {
      const DAY_MINUTES = 24 * 60;
      const computeItineraryPxPerMin = () => {
        if (typeof window === "undefined" || typeof window.matchMedia !== "function") return 1.2;
        if (window.matchMedia("(max-width: 420px)").matches) return 0.88;
        if (window.matchMedia("(max-width: 720px)").matches) return 0.98;
        return 1.2;
      };
      const state = {
        view: "list",
        weekOffset: 0,
        overrides: new Map(),
        dragOverCol: null,
        dragHintEl: null,
        dragHintMinute: null,
        resizeState: null,
        weekScrollTop: 0,
        dayScrollTop: 0,
        since: "",
        selectedDate: tripStartRaw,
        lastDragEndAt: 0,
        pxPerMin: 1.2,
        alldayDnD: null
      };
      let calendarQuickEditUiAbort = null;
      const toDate = (iso) => {
        const p = new Date(`${iso}T00:00:00`);
        return Number.isNaN(p.getTime()) ? null : p;
      };
      let startDate = null;
      let endDate = null;
      let totalTripDays = 1;
      const reanchorTripCalendarDates = () => {
        startDate = toDate(tripStartRaw);
        endDate = toDate(tripEndRaw);
        totalTripDays =
          startDate && endDate && startDate <= endDate
            ? Math.floor((endDate - startDate) / 86400000) + 1
            : 1;
      };
      reanchorTripCalendarDates();
      const reshellItineraryCalendarDom = () => {
        const r = document.querySelector("[data-itinerary-calendar-root]");
        if (!r) return false;
        tripCalendarRoot = r;
        listPanel = r.querySelector("[data-itinerary-list-panel]");
        calendarPanels = r.querySelector("[data-itinerary-calendar-panels]");
        dayView = r.querySelector("[data-itinerary-day-view]");
        weekView = r.querySelector("[data-itinerary-week-view]");
        toolbar = r.querySelector("[data-itinerary-calendar-toolbar]");
        rangeLabel = r.querySelector("[data-itinerary-range-label]");
        tripID = (r.getAttribute("data-trip-id") || "").trim();
        tripStartRaw = (r.getAttribute("data-trip-start") || "").trim();
        tripEndRaw = (r.getAttribute("data-trip-end") || "").trim();
        quickEditModal = r.querySelector("[data-itinerary-quick-edit-modal]");
        quickEditBody = r.querySelector("[data-itinerary-quick-edit-body]");
        quickEditPreview = r.querySelector("[data-itinerary-quick-edit-preview]");
        quickEditFormWrap = r.querySelector("[data-itinerary-quick-edit-form-wrap]");
        quickEditPanel = r.querySelector("[data-itinerary-quick-edit-panel]");
        quickEditTitle = r.querySelector("[data-itinerary-quick-edit-title]");
        quickEditModeLabel = r.querySelector("[data-itinerary-quick-edit-mode-label]");
        reanchorTripCalendarDates();
        return Boolean(listPanel && calendarPanels && dayView && weekView && toolbar && rangeLabel && tripStartRaw && tripEndRaw);
      };
      window.remiReshellItineraryCalendarDom = reshellItineraryCalendarDom;
      const pad2 = (v) => String(v).padStart(2, "0");
      /** YYYY-MM-DD for the user's local calendar day (not UTC — toISOString would shift columns vs headers). */
      const fmtDate = (d) => {
        if (!(d instanceof Date) || Number.isNaN(d.getTime())) return "";
        return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
      };
      const toTime = (mins) => `${pad2(Math.floor(mins / 60) % 24)}:${pad2(mins % 60)}`;
      const parseTime = (raw) => {
        const s = String(raw || "").trim();
        const m = s.match(/^(\d{1,2}):(\d{2})(?:\s*([AP]M))?$/i);
        if (!m) return null;
        let h = parseInt(m[1], 10);
        const min = parseInt(m[2], 10);
        const ap = (m[3] || "").toUpperCase();
        if (ap) {
          if (h === 12) h = 0;
          if (ap === "PM") h += 12;
        }
        if (h < 0 || h > 23 || min < 0 || min > 59) return null;
        return h * 60 + min;
      };
      const parseDateTimeLocal = (raw) => {
        const d = new Date(raw);
        return Number.isNaN(d.getTime()) ? null : d;
      };
      const toDateTimeLocal = (d) => `${fmtDate(d)}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
      const addMinutes = (dt, mins) => new Date(dt.getTime() + mins * 60000);
      const dateInRange = (iso) => {
        if (!iso) return false;
        return iso >= tripStartRaw && iso <= tripEndRaw;
      };
      const toDayOffset = (iso) => {
        const d = toDate(iso);
        if (!d || !startDate) return 0;
        return Math.floor((d - startDate) / 86400000);
      };
      const clampISO = (iso) => {
        if (!iso) return tripStartRaw;
        if (iso < tripStartRaw) return tripStartRaw;
        if (iso > tripEndRaw) return tripEndRaw;
        return iso;
      };
      const shiftISO = (iso, days) => {
        const base = toDate(iso) || toDate(tripStartRaw) || new Date();
        base.setDate(base.getDate() + days);
        return fmtDate(base);
      };
      const syncWeekOffsetFromSelectedDate = () => {
        const dayOffset = Math.max(0, Math.min(totalTripDays - 1, toDayOffset(state.selectedDate)));
        state.weekOffset = Math.floor(dayOffset / 7) * 7;
      };
      if (!startDate || !endDate || startDate > endDate) {
        // Skip calendar render only; keep the rest of app.js behavior alive.
      } else {
      const todayISO = fmtDate(new Date());
      if (todayISO >= tripStartRaw && todayISO <= tripEndRaw) {
        state.selectedDate = todayISO;
      }
      syncWeekOffsetFromSelectedDate();
      const inferRoleFromTitle = (title) => {
        const t = (title || "").toLowerCase();
        if (t.includes("depart")) return "depart";
        if (t.includes("arrive") || t.includes("arrival")) return "arrive";
        if (t.includes("check-out")) return "checkout";
        if (t.includes("check in") || t.includes("check-in")) return "checkin";
        if (t.includes("drop-off") || t.includes("drop off")) return "dropoff";
        if (t.includes("pick-up") || t.includes("pick up") || t.includes("pickup")) return "pickup";
        return "single";
      };
      const inferRole = (row, title) => {
        const leg = (row?.getAttribute?.("data-itinerary-booking-leg") || "").trim().toLowerCase();
        if (
          leg === "depart" ||
          leg === "arrive" ||
          leg === "checkin" ||
          leg === "checkout" ||
          leg === "pickup" ||
          leg === "dropoff"
        ) {
          return leg;
        }
        return inferRoleFromTitle(title);
      };
      const bookingPairFields = (category) => {
        if (category === "flight") return { startField: "depart_at", endField: "arrive_at", primaryRole: "depart", secondaryRole: "arrive" };
        if (category === "vehicle") return { startField: "pick_up_at", endField: "drop_off_at", primaryRole: "pickup", secondaryRole: "dropoff" };
        if (category === "accommodation") return { startField: "check_in_at", endField: "check_out_at", primaryRole: "checkin", secondaryRole: "checkout" };
        return null;
      };
      const parseBookingWindow = (form, category) => {
        const pair = bookingPairFields(category);
        if (!form || !pair) return null;
        const startAt = parseDateTimeLocal(form.querySelector(`input[name='${pair.startField}']`)?.value || "");
        const endAt = parseDateTimeLocal(form.querySelector(`input[name='${pair.endField}']`)?.value || "");
        if (!startAt || !endAt) return null;
        let s = startAt;
        let e = endAt;
        if (e <= s) {
          const fallback = new Date(s.getTime() + 60 * 60000);
          e = fallback;
        }
        return { ...pair, startAt: s, endAt: e };
      };
      const parseEventRows = () =>
        Array.from(listPanel.querySelectorAll(".timeline-item[data-itinerary-item-id]")).map((row) => {
          const itemID = (row.getAttribute("data-itinerary-item-id") || "").trim();
          const dayGroup = row.closest(".day-group");
          const date = (dayGroup?.getAttribute("data-date") || "").trim();
          const view = row.querySelector(".itinerary-item-view");
          const title = (
            row.getAttribute("data-title") ||
            view?.querySelector(".itinerary-stop-card__title")?.textContent ||
            view?.querySelector(".vehicle-main-top h4")?.textContent ||
            view?.querySelector(".flight-main-head h4")?.textContent ||
            view?.querySelector(":scope > strong")?.textContent ||
            view?.querySelector("strong")?.textContent ||
            "Stop"
          ).trim();
          const markerKind = (row.getAttribute("data-marker-kind") || "stop").trim() || "stop";
          let form = row.querySelector(`form#itinerary-edit-${CSS.escape(itemID)}`);
          let category = "stop";
          if (!form) {
            form = row.querySelector(`form#accommodation-itinerary-edit-${CSS.escape(itemID)}`);
            if (form) category = "accommodation";
          }
          if (!form) {
            form = row.querySelector(`form#vehicle-rental-itinerary-edit-${CSS.escape(itemID)}`);
            if (form) category = "vehicle";
          }
          if (!form) {
            form = row.querySelector(`form#flight-itinerary-edit-${CSS.escape(itemID)}`);
            if (form) category = "flight";
          }
          if (!form) {
            form = row.querySelector(`form#commute-itinerary-edit-${CSS.escape(itemID)}`);
            if (form) category = "commute";
          }
          let startMin = 8 * 60;
          let endMin = 9 * 60;
          let usedAbsoluteDaySpan = false;
          if (form && (category === "stop" || category === "commute")) {
            const ISO_DT = /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2})/;
            const startH = form.querySelector("input.remi-datetime-iso[name='start_at']");
            const endH = form.querySelector("input.remi-datetime-iso[name='end_at']");
            const ms = startH instanceof HTMLInputElement ? String(startH.value || "").match(ISO_DT) : null;
            const me = endH instanceof HTMLInputElement ? String(endH.value || "").match(ISO_DT) : null;
            if (ms && me) {
              const startDT = new Date(`${ms[1]}T${ms[2]}:${ms[3]}:00`);
              const endDT = new Date(`${me[1]}T${me[2]}:${me[3]}:00`);
              if (!Number.isNaN(startDT.getTime()) && !Number.isNaN(endDT.getTime()) && endDT > startDT) {
                const rowMidnight = new Date(`${date}T00:00:00`);
                if (!Number.isNaN(rowMidnight.getTime())) {
                  startMin = Math.round((startDT - rowMidnight) / 60000);
                  endMin = Math.round((endDT - rowMidnight) / 60000);
                  usedAbsoluteDaySpan = true;
                }
              }
            }
            if (!usedAbsoluteDaySpan && ms) {
              startMin = parseInt(ms[2], 10) * 60 + parseInt(ms[3], 10);
              endMin = me
                ? parseInt(me[2], 10) * 60 + parseInt(me[3], 10)
                : startMin + 60;
            } else if (!usedAbsoluteDaySpan) {
              const s = parseTime(form.querySelector("input[name='start_time']")?.value || "");
              const e = parseTime(form.querySelector("input[name='end_time']")?.value || "");
              if (s !== null) startMin = s;
              endMin = e !== null ? e : startMin + 60;
            }
          } else {
            const meta = (view?.querySelector(".meta")?.textContent || "").trim();
            const bits = meta.split("-").map((s) => s.trim()).filter(Boolean);
            const first = parseTime(bits[0] || "");
            const second = parseTime(bits[1] || "");
            if (first !== null) startMin = first;
            endMin = second !== null ? second : startMin + 60;
          }
          if (!usedAbsoluteDaySpan && endMin <= startMin) endMin = Math.min(DAY_MINUTES, startMin + 60);
          const role = inferRole(row, title);
          const bookingWindow = parseBookingWindow(form, category);
          if (bookingWindow && role === bookingWindow.secondaryRole) {
            // Secondary leg is rendered through the primary card as a spanning duration.
            return null;
          }
          let calTitle = title;
          if (category === "flight" && form) {
            const iata3 = (s) => {
              const t = String(s || "").trim().toUpperCase();
              return /^[A-Z0-9]{3}$/.test(t) ? t : "";
            };
            const depI = iata3(form.querySelector("input[name='depart_airport_iata']")?.value);
            const arrI = iata3(form.querySelector("input[name='arrive_airport_iata']")?.value);
            if (depI && arrI) calTitle = `Flight from ${depI} to ${arrI}`;
            else if (depI) calTitle = `Flight from ${depI}`;
            else if (arrI) calTitle = `Flight to ${arrI}`;
          }
          return {
            itemID, date, startMin, endMin, form, category, markerKind, title: calTitle,
            role, row,
            pairedSpan: Boolean(bookingWindow),
            pairedStartField: bookingWindow?.startField || "",
            pairedEndField: bookingWindow?.endField || "",
            pairedStartAt: bookingWindow?.startAt || null,
            pairedEndAt: bookingWindow?.endAt || null
          };
        }).filter((e) => e && e.itemID && e.date && e.form);
      const expandRenderableEntries = (baseEntries) => {
        const out = [];
        const accPropertyNameFromForm = (f) => {
          const n = f?.querySelector("input[name='name']")?.value;
          return String(n || "").trim() || "Accommodation";
        };
        baseEntries.forEach((e) => {
          if (!e.pairedSpan || !e.pairedStartAt || !e.pairedEndAt) {
            out.push({ ...e, sourceItemID: e.itemID, segmentStartAt: new Date(`${e.date}T${toTime(e.startMin)}`) });
            return;
          }
          /* Accommodation: all-day "Stay" per night + explicit 1h check-in / check-out blocks (Today / Week calendar). */
          if (e.category === "accommodation" && e.role === "checkin") {
            const prop = accPropertyNameFromForm(e.form);
            const clipStart = new Date(Math.max(e.pairedStartAt.getTime(), startDate.getTime()));
            const clipEnd = new Date(Math.min(e.pairedEndAt.getTime(), endDate.getTime() + 24 * 60 * 60000));
            const firstStayDay = fmtDate(clipStart);
            const lastStayDay = fmtDate(clipEnd);
            const walk = new Date(`${firstStayDay}T00:00:00`);
            const walkEnd = new Date(`${lastStayDay}T00:00:00`);
            while (walk <= walkEnd) {
              const key = fmtDate(walk);
              if (key >= tripStartRaw && key <= tripEndRaw) {
                out.push({
                  ...e,
                  itemID: `${e.itemID}::stay::${key}`,
                  sourceItemID: e.itemID,
                  date: key,
                  startMin: 0,
                  endMin: DAY_MINUTES,
                  title: `Stay: ${prop}`,
                  pairedSpan: false,
                  pairedStartAt: null,
                  pairedEndAt: null,
                  pairedStartField: "",
                  pairedEndField: "",
                  calAccLeg: "stay",
                  segmentStartAt: new Date(walk)
                });
              }
              walk.setDate(walk.getDate() + 1);
            }
            const cinDay = fmtDate(e.pairedStartAt);
            if (cinDay >= tripStartRaw && cinDay <= tripEndRaw) {
              const sm = e.pairedStartAt.getHours() * 60 + e.pairedStartAt.getMinutes();
              const em = Math.min(DAY_MINUTES, sm + 60);
              out.push({
                ...e,
                itemID: `${e.itemID}::cal-checkin`,
                sourceItemID: e.itemID,
                date: cinDay,
                startMin: sm,
                endMin: Math.max(sm + 15, em),
                title: `Check-in: ${prop}`,
                pairedSpan: false,
                pairedStartAt: null,
                pairedEndAt: null,
                pairedStartField: "",
                pairedEndField: "",
                calAccLeg: "checkin",
                segmentStartAt: new Date(e.pairedStartAt)
              });
            }
            const coutDay = fmtDate(e.pairedEndAt);
            if (coutDay >= tripStartRaw && coutDay <= tripEndRaw) {
              const endM = e.pairedEndAt.getHours() * 60 + e.pairedEndAt.getMinutes();
              const sm = Math.max(0, endM - 60);
              out.push({
                ...e,
                itemID: `${e.itemID}::cal-checkout`,
                sourceItemID: e.itemID,
                date: coutDay,
                startMin: sm,
                endMin: Math.max(sm + 15, endM),
                title: `Check-out: ${prop}`,
                pairedSpan: false,
                pairedStartAt: null,
                pairedEndAt: null,
                pairedStartField: "",
                pairedEndField: "",
                calAccLeg: "checkout",
                segmentStartAt: new Date(`${coutDay}T${toTime(sm)}`)
              });
            }
            return;
          }
          /* Vehicle rental: 1h pick-up / drop-off; all-day "Renting" only when pick-up and drop-off fall on different calendar days. */
          if (e.category === "vehicle" && e.role === "pickup") {
            const veh = String(e.form?.querySelector("input[name='vehicle_detail']")?.value || "").trim() || "Vehicle rental";
            const clipStart = new Date(Math.max(e.pairedStartAt.getTime(), startDate.getTime()));
            const clipEnd = new Date(Math.min(e.pairedEndAt.getTime(), endDate.getTime() + 24 * 60 * 60000));
            const pickupDay = fmtDate(e.pairedStartAt);
            const dropoffDay = fmtDate(e.pairedEndAt);
            const multiCalendarDay = pickupDay !== dropoffDay;
            if (multiCalendarDay) {
              const firstR = fmtDate(clipStart);
              const lastR = fmtDate(clipEnd);
              const walk = new Date(`${firstR}T00:00:00`);
              const walkEnd = new Date(`${lastR}T00:00:00`);
              while (walk <= walkEnd) {
                const key = fmtDate(walk);
                if (key >= tripStartRaw && key <= tripEndRaw) {
                  out.push({
                    ...e,
                    itemID: `${e.itemID}::renting::${key}`,
                    sourceItemID: e.itemID,
                    date: key,
                    startMin: 0,
                    endMin: DAY_MINUTES,
                    title: `Renting: ${veh}`,
                    pairedSpan: false,
                    pairedStartAt: null,
                    pairedEndAt: null,
                    pairedStartField: "",
                    pairedEndField: "",
                    calVehLeg: "renting",
                    segmentStartAt: new Date(walk)
                  });
                }
                walk.setDate(walk.getDate() + 1);
              }
            }
            const puDay = fmtDate(e.pairedStartAt);
            if (puDay >= tripStartRaw && puDay <= tripEndRaw) {
              const sm = e.pairedStartAt.getHours() * 60 + e.pairedStartAt.getMinutes();
              const em = Math.min(DAY_MINUTES, sm + 60);
              out.push({
                ...e,
                itemID: `${e.itemID}::cal-pickup`,
                sourceItemID: e.itemID,
                date: puDay,
                startMin: sm,
                endMin: Math.max(sm + 15, em),
                title: `Pick-up: ${veh}`,
                pairedSpan: false,
                pairedStartAt: null,
                pairedEndAt: null,
                pairedStartField: "",
                pairedEndField: "",
                calVehLeg: "pickup",
                segmentStartAt: new Date(e.pairedStartAt)
              });
            }
            const doDay = fmtDate(e.pairedEndAt);
            if (doDay >= tripStartRaw && doDay <= tripEndRaw) {
              const endM = e.pairedEndAt.getHours() * 60 + e.pairedEndAt.getMinutes();
              const sm = Math.max(0, endM - 60);
              out.push({
                ...e,
                itemID: `${e.itemID}::cal-dropoff`,
                sourceItemID: e.itemID,
                date: doDay,
                startMin: sm,
                endMin: Math.max(sm + 15, endM),
                title: `Drop-off: ${veh}`,
                pairedSpan: false,
                pairedStartAt: null,
                pairedEndAt: null,
                pairedStartField: "",
                pairedEndField: "",
                calVehLeg: "dropoff",
                segmentStartAt: new Date(`${doDay}T${toTime(sm)}`)
              });
            }
            return;
          }
          /* Overnight flights: one calendar block per local day (00:00–24:00 clip). */
          const clipStart = new Date(Math.max(e.pairedStartAt.getTime(), startDate.getTime()));
          const clipEnd = new Date(Math.min(e.pairedEndAt.getTime(), endDate.getTime() + 24 * 60 * 60000));
          const cur = new Date(`${fmtDate(clipStart)}T00:00:00`);
          const last = new Date(`${fmtDate(clipEnd)}T00:00:00`);
          while (cur <= last) {
            const key = fmtDate(cur);
            const dayStart = new Date(`${key}T00:00:00`);
            const dayEnd = new Date(dayStart);
            dayEnd.setDate(dayEnd.getDate() + 1);
            const segStart = new Date(Math.max(e.pairedStartAt.getTime(), dayStart.getTime()));
            const segEnd = new Date(Math.min(e.pairedEndAt.getTime(), dayEnd.getTime()));
            if (segEnd > segStart && key >= tripStartRaw && key <= tripEndRaw) {
              const startMin = segStart.getHours() * 60 + segStart.getMinutes();
              let endMin = segEnd.getHours() * 60 + segEnd.getMinutes();
              if (endMin <= startMin) endMin = DAY_MINUTES;
              out.push({
                ...e,
                itemID: `${e.itemID}::${key}`,
                sourceItemID: e.itemID,
                date: key,
                startMin,
                endMin: Math.max(startMin + 15, endMin),
                segmentStartAt: segStart
              });
            }
            cur.setDate(cur.getDate() + 1);
          }
        });
        return out;
      };
      const pairedSpanResizeFlags = (e) => {
        if (e.calVehLeg === "renting") {
          return { allowResizeStart: false, allowResizeEnd: false };
        }
        if (e.calVehLeg === "pickup" || e.calVehLeg === "dropoff") {
          return { allowResizeStart: true, allowResizeEnd: true };
        }
        if (e.calAccLeg === "stay") {
          return { allowResizeStart: false, allowResizeEnd: false };
        }
        if (e.calAccLeg === "checkin" || e.calAccLeg === "checkout") {
          return { allowResizeStart: true, allowResizeEnd: true };
        }
        if (!e.pairedSpan || !e.pairedStartAt || !e.pairedEndAt) {
          return { allowResizeStart: true, allowResizeEnd: true };
        }
        const startDay = fmtDate(e.pairedStartAt);
        const endDay = fmtDate(e.pairedEndAt);
        return {
          allowResizeStart: e.date === startDay,
          allowResizeEnd: e.date === endDay
        };
      };
      const getRenderableEntries = () =>
        expandRenderableEntries(parseEventRows()).map((e) => ({ ...e, ...(state.overrides.get(e.itemID) || {}) }));
      const weekDates = () => {
        const out = [];
        const weekStart = new Date(startDate);
        weekStart.setDate(weekStart.getDate() + state.weekOffset);
        for (let i = 0; i < 7; i++) {
          const d = new Date(weekStart);
          d.setDate(weekStart.getDate() + i);
          if (d > endDate) break;
          out.push(d);
        }
        return out;
      };
        const isAllDayEvent = (e) =>
          e &&
          e.startMin === 0 &&
          (e.endMin >= DAY_MINUTES - 1 || e.endMin === DAY_MINUTES);
        const alldayDragMovable = (e) => {
          if (!e) return false;
          if (e.calVehLeg === "renting") return false;
          if (e.calAccLeg === "stay") return false;
          if (!isAllDayEvent(e)) return false;
          if (e.category === "stop" || e.category === "commute") {
            if (e.pairedSpan) return false;
            if (String(e.itemID || "").includes("::")) return false;
            return true;
          }
          if (e.category !== "accommodation") return false;
          if (e.pairedSpan && e.pairedStartAt && e.pairedEndAt && String(e.itemID || "").includes("::")) {
            return true;
          }
          if (!e.pairedSpan && !String(e.itemID || "").includes("::")) {
            const f = e.form;
            return Boolean(
              f &&
              f.querySelector("input.remi-datetime-iso[name='check_in_at']") &&
              f.querySelector("input.remi-datetime-iso[name='check_out_at']")
            );
          }
          return false;
        };
        const applyCollisionOffsets = (entries) => {
          const sorted = [...entries].sort((a, b) => {
            if (a.startMin !== b.startMin) return a.startMin - b.startMin;
            return (b.endMin - b.startMin) - (a.endMin - a.startMin);
          });
          const groups = [];
          let group = [];
          let groupEnd = -1;
          sorted.forEach((e) => {
            if (!group.length || e.startMin < groupEnd) {
              group.push(e);
              groupEnd = Math.max(groupEnd, e.endMin);
            } else {
              groups.push(group);
              group = [e];
              groupEnd = e.endMin;
            }
          });
          if (group.length) groups.push(group);
          groups.forEach((g) => {
            const lanes = [];
            g.forEach((e) => {
              let lane = 0;
              while (lane < lanes.length && lanes[lane] > e.startMin) lane++;
              lanes[lane] = e.endMin;
              e.lane = lane;
              e.laneCount = Math.max(1, lanes.length);
            });
            const count = Math.max(1, lanes.length);
            g.forEach((e) => {
              e.laneCount = count;
            });
          });
        };
      const submitFormAjax = async (form) => {
        const formData = new FormData(form);
        const hasFileInput = Boolean(form.querySelector("input[type='file']"));
        const isMultipart = (form.enctype || "").toLowerCase().includes("multipart/form-data") || hasFileInput;
        const body = isMultipart ? formData : new URLSearchParams(formData);
        const headers = { "X-Requested-With": "XMLHttpRequest", Accept: "application/json" };
        if (!isMultipart) headers["Content-Type"] = "application/x-www-form-urlencoded;charset=UTF-8";
        const res = await fetch(form.action, { method: (form.method || "POST").toUpperCase(), headers, body });
        if (!res.ok) throw new Error((await res.text()).trim() || "Unable to save right now.");
      };
      const setField = (form, name, value) => {
        const target = form.querySelector(`input.remi-datetime-iso[name='${name}'], input.remi-date-iso[name='${name}'], input.remi-time-iso[name='${name}'], input[name='${name}']`);
        if (!target) return;
        target.value = value;
        target.dispatchEvent(new Event("change", { bubbles: true }));
      };
      const linkedFieldForEntry = (entry) => {
        if (!entry) return "";
        if (entry.category === "flight") {
          if (entry.role === "depart") return "depart_at";
          if (entry.role === "arrive") return "arrive_at";
        }
        if (entry.category === "vehicle") {
          if (entry.role === "pickup") return "pick_up_at";
          if (entry.role === "dropoff") return "drop_off_at";
        }
        if (entry.category === "accommodation") {
          if (entry.role === "checkin") return "check_in_at";
          if (entry.role === "checkout") return "check_out_at";
        }
        return "";
      };
      const applyResizeToForm = (entry, nextDate, nextStart, nextEnd, edge) => {
        if (entry.calVehLeg === "renting") {
          return;
        }
        if (entry.calVehLeg === "pickup") {
          const startDT = new Date(`${nextDate}T${toTime(nextStart)}`);
          if (!Number.isNaN(startDT.getTime())) {
            setField(entry.form, "pick_up_at", toDateTimeLocal(startDT));
          }
          return;
        }
        if (entry.calVehLeg === "dropoff") {
          const endDT = new Date(`${nextDate}T${toTime(nextEnd)}`);
          if (!Number.isNaN(endDT.getTime())) {
            setField(entry.form, "drop_off_at", toDateTimeLocal(endDT));
          }
          return;
        }
        if (entry.calAccLeg === "stay") {
          return;
        }
        if (entry.calAccLeg === "checkin") {
          const startDT = new Date(`${nextDate}T${toTime(nextStart)}`);
          if (!Number.isNaN(startDT.getTime())) {
            setField(entry.form, "check_in_at", toDateTimeLocal(startDT));
          }
          return;
        }
        if (entry.calAccLeg === "checkout") {
          const endDT = new Date(`${nextDate}T${toTime(nextEnd)}`);
          if (!Number.isNaN(endDT.getTime())) {
            setField(entry.form, "check_out_at", toDateTimeLocal(endDT));
          }
          return;
        }
        if (entry.category === "stop" || entry.category === "commute") {
          const startAtEl = entry.form.querySelector("input.remi-datetime-iso[name='start_at']");
          if (startAtEl instanceof HTMLInputElement) {
            setField(entry.form, "start_at", toDateTimeLocal(new Date(`${nextDate}T${toTime(nextStart)}`)));
            setField(entry.form, "end_at", toDateTimeLocal(new Date(`${nextDate}T${toTime(nextEnd)}`)));
            return;
          }
          setField(entry.form, "itinerary_date", nextDate);
          setField(entry.form, "start_time", toTime(nextStart));
          setField(entry.form, "end_time", toTime(nextEnd));
          return;
        }
        if (entry.pairedSpan && entry.pairedStartField && entry.pairedEndField) {
          const nextDT = toDateTimeLocal(new Date(`${nextDate}T${toTime(edge === "start" ? nextStart : nextEnd)}`));
          if (edge === "start") setField(entry.form, entry.pairedStartField, nextDT);
          else setField(entry.form, entry.pairedEndField, nextDT);
          return;
        }
        const singleField = linkedFieldForEntry(entry);
        if (singleField) {
          setField(entry.form, singleField, toDateTimeLocal(new Date(`${nextDate}T${toTime(edge === "start" ? nextStart : nextEnd)}`)));
        }
      };
      const addCalendarDays = (d, n) => {
        const t = new Date(d.getTime());
        t.setDate(t.getDate() + n);
        return t;
      };
      const applyAccommodationAllDayDateShift = (entry, toDate) => {
        if (!entry?.form) return false;
        const dDays = toDayOffset(toDate) - toDayOffset(entry.date);
        if (dDays === 0) return true;
        const inEl = entry.form.querySelector("input.remi-datetime-iso[name='check_in_at']");
        const outEl = entry.form.querySelector("input.remi-datetime-iso[name='check_out_at']");
        if (!(inEl instanceof HTMLInputElement) || !(outEl instanceof HTMLInputElement)) {
          showToast("Open the stay in the editor to move it.");
          return false;
        }
        const fromParse = (raw) => parseDateTimeLocal(String(raw || "").trim());
        if (String(entry.itemID || "").includes("::") && entry.pairedStartAt && entry.pairedEndAt) {
          const newIn = addCalendarDays(entry.pairedStartAt, dDays);
          const newOut = addCalendarDays(entry.pairedEndAt, dDays);
          if (newOut.getTime() <= newIn.getTime()) {
            showToast("Those dates are not valid for this stay.");
            return false;
          }
          const inDay = fmtDate(newIn);
          if (inDay < tripStartRaw || inDay > tripEndRaw) {
            showToast("Check-in would land outside the trip.");
            return false;
          }
          setField(entry.form, "check_in_at", toDateTimeLocal(newIn));
          setField(entry.form, "check_out_at", toDateTimeLocal(newOut));
          return true;
        }
        const curIn = fromParse(inEl.value);
        const curOut = fromParse(outEl.value);
        if (!curIn || !curOut) {
          showToast("Open the stay in the editor to move it.");
          return false;
        }
        const newIn = addCalendarDays(curIn, dDays);
        const newOut = addCalendarDays(curOut, dDays);
        if (newOut.getTime() <= newIn.getTime()) {
          showToast("Those dates are not valid for this stay.");
          return false;
        }
        const inDay2 = fmtDate(newIn);
        if (inDay2 < tripStartRaw || inDay2 > tripEndRaw) {
          showToast("Check-in would land outside the trip.");
          return false;
        }
        setField(entry.form, "check_in_at", toDateTimeLocal(newIn));
        setField(entry.form, "check_out_at", toDateTimeLocal(newOut));
        return true;
      };
      const closeDesktopFlyout = () => {
        if (!desktopCalFlyout) return;
        desktopCalFlyout.classList.add("hidden");
        desktopCalFlyout.setAttribute("aria-hidden", "true");
        desktopCalFlyout.classList.remove("is-expanded");
      };
      const setDesktopFlyoutExpanded = (expanded) => {
        if (!desktopCalFlyout) return;
        desktopCalFlyout.classList.toggle("is-expanded", expanded);
      };
      const openDesktopFlyout = (date, startMin, clientX, clientY) => {
        if (!desktopCalFlyout) return;
        quickAddState.date = date;
        quickAddState.startMin = startMin;
        quickAddState.lastClientX = typeof clientX === "number" ? clientX : null;
        quickAddState.lastClientY = typeof clientY === "number" ? clientY : null;
        setDesktopFlyoutExpanded(true);
        if (desktopCalFlyoutLabel) {
          const d = new Date(`${date}T00:00:00`);
          desktopCalFlyoutLabel.textContent = `Quick add: ${d.toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" })} at ${toTime(startMin)}`;
        }
        desktopCalFlyout.classList.remove("hidden");
        desktopCalFlyout.setAttribute("aria-hidden", "false");
        if (typeof clientX === "number" && typeof clientY === "number") {
          const placeNearPointer = () => {
            const vw = window.innerWidth;
            const vh = window.innerHeight;
            const pad = 12;
            const rect = desktopCalFlyout.getBoundingClientRect();
            const width = rect.width || 300;
            const height = rect.height || 200;
            const left = Math.max(pad, Math.min(vw - width - pad, clientX + 12));
            const top = Math.max(pad, Math.min(vh - height - pad, clientY + 12));
            desktopCalFlyout.style.left = `${left}px`;
            desktopCalFlyout.style.top = `${top}px`;
            desktopCalFlyout.style.right = "auto";
            desktopCalFlyout.style.bottom = "auto";
          };
          placeNearPointer();
          requestAnimationFrame(placeNearPointer);
        }
      };
      const focusFormField = (form) => {
        if (!form) return;
        const first = form.querySelector("input[type='text'], input[type='search'], input:not([type='hidden']), textarea, select");
        first?.focus?.();
      };
      const findAddForm = (kind) => {
        if (kind === "expense") {
          const fly = document.querySelector("[data-desktop-expense-flyout] form[data-remi-unified-expense-add-form]");
          if (fly && window.innerWidth > 920) return fly;
          const side = document.querySelector("form.remi-unified-expense-form--sidebar");
          if (side) return side;
          return document.querySelector("#mobile-sheet-expense form[data-remi-unified-expense-add-form]");
        }
        const forms = Array.from(document.querySelectorAll("form[action]"));
        const matches = forms.filter((f) => {
          const action = (f.getAttribute("action") || "");
          if (action.includes("/update")) return false;
          if (kind === "commute") return action.includes("/itinerary/commute");
          if (kind === "stop")
            return action.includes("/itinerary") && !action.includes("/itinerary/commute");
          if (kind === "accommodation") return action.includes("/accommodation");
          if (kind === "vehicle") return action.includes("/vehicle-rental");
          if (kind === "flight") return action.includes("/flights");
          return false;
        });
        return matches.find((f) => !f.closest(".mobile-sheet")) || matches[0] || null;
      };
      const sheetFormForKindOnDesktop = (kind) => {
        if (window.innerWidth <= 920 || !document.querySelector("main.trip-fab-flyout-root")) return null;
        const id =
          kind === "stop"
            ? "mobile-sheet-stop"
            : kind === "commute"
              ? "mobile-sheet-commute"
            : kind === "expense"
              ? "mobile-sheet-expense"
              : kind === "accommodation"
                ? "mobile-sheet-accommodation"
                : kind === "vehicle"
                  ? "mobile-sheet-vehicle"
                  : kind === "flight"
                    ? "mobile-sheet-flight"
                    : "";
        if (!id) return null;
        const sheet = document.getElementById(id);
        if (kind === "commute") {
          return sheet?.querySelector("form.remi-add-commute-form") || null;
        }
        if (kind === "expense") {
          return sheet?.querySelector("form[data-remi-unified-expense-add-form]") || null;
        }
        return sheet?.querySelector("form[action]:not(.remi-add-commute-form)") || null;
      };
      const showTripQuickSheetPanel = (sheetEl) => {
        if (!sheetEl) return;
        document.querySelectorAll(".mobile-sheet").forEach((s) => {
          s.classList.remove("mobile-sheet--trip-fab-flyout-open", "mobile-sheet--dragging");
          s.style.transform = "";
          s.classList.add("hidden");
          s.setAttribute("aria-hidden", "true");
        });
        const bd = document.querySelector("[data-mobile-sheet-backdrop]");
        sheetEl.classList.remove("hidden");
        sheetEl.setAttribute("aria-hidden", "false");
        if (bd) bd.classList.remove("hidden");
        if (sheetEl.classList.contains("mobile-sheet--trip-fab-flyout") && document.querySelector("main.trip-fab-flyout-root")) {
          sheetEl.classList.remove("mobile-sheet--trip-fab-flyout-open");
          void sheetEl.offsetWidth;
          requestAnimationFrame(() => {
            requestAnimationFrame(() => {
              sheetEl.classList.add("mobile-sheet--trip-fab-flyout-open");
            });
          });
        }
      };
      const prefillAddForm = (kind) => {
        const date = quickAddState.date || tripStartRaw;
        const startMin = quickAddState.startMin ?? (8 * 60);
        const startDT = new Date(`${date}T${toTime(startMin)}`);
        const form = sheetFormForKindOnDesktop(kind) || findAddForm(kind);
        if (!form) return;
        if (kind === "expense") {
          window.remiSetFormDateIso?.(form, date);
          const expFly = form.closest("[data-desktop-expense-flyout]");
          if (expFly instanceof HTMLElement && window.innerWidth > 920) {
            expFly.classList.remove("hidden");
            expFly.setAttribute("aria-hidden", "false");
            const cx = quickAddState.lastClientX;
            const cy = quickAddState.lastClientY;
            const placeExp = () => {
              if (typeof cx !== "number" || typeof cy !== "number") return;
              const vw = window.innerWidth;
              const vh = window.innerHeight;
              const pad = 12;
              const rect = expFly.getBoundingClientRect();
              const width = rect.width || 400;
              const height = rect.height || 480;
              const left = Math.max(pad, Math.min(vw - width - pad, cx + 12));
              const top = Math.max(pad, Math.min(vh - height - pad, cy + 12));
              expFly.style.left = `${left}px`;
              expFly.style.top = `${top}px`;
              expFly.style.right = "auto";
              expFly.style.bottom = "auto";
            };
            placeExp();
            requestAnimationFrame(placeExp);
            window.setTimeout(() => focusFormField(form), 80);
            return;
          }
        }
        if (kind === "stop" || kind === "commute") {
          const startAtEl = form.querySelector("input.remi-datetime-iso[name='start_at']");
          if (startAtEl instanceof HTMLInputElement) {
            const endM = Math.min(DAY_MINUTES, startMin + 60);
            setField(form, "start_at", toDateTimeLocal(new Date(`${date}T${toTime(startMin)}`)));
            setField(form, "end_at", toDateTimeLocal(new Date(`${date}T${toTime(endM)}`)));
          } else {
            setField(form, "itinerary_date", date);
            setField(form, "start_time", toTime(startMin));
            setField(form, "end_time", toTime(Math.min(DAY_MINUTES, startMin + 60)));
          }
        } else if (kind === "flight") {
          setField(form, "depart_at", toDateTimeLocal(startDT));
          setField(form, "arrive_at", toDateTimeLocal(addMinutes(startDT, 90)));
        } else if (kind === "vehicle") {
          setField(form, "pick_up_at", toDateTimeLocal(startDT));
          setField(form, "drop_off_at", toDateTimeLocal(addMinutes(startDT, 120)));
        } else if (kind === "accommodation") {
          const out = new Date(startDT);
          out.setDate(out.getDate() + 1);
          setField(form, "check_in_at", toDateTimeLocal(startDT));
          setField(form, "check_out_at", toDateTimeLocal(out));
        }
        const mobileSheet = form.closest(".mobile-sheet");
        const onTripDesktop =
          !!document.querySelector("main.trip-fab-flyout-root") && window.innerWidth > 920 && mobileSheet;
        if (kind === "commute") {
          form.closest("details")?.setAttribute("open", "");
        }
        if (mobileSheet) {
          if (onTripDesktop) {
            showTripQuickSheetPanel(mobileSheet);
            window.setTimeout(() => {
              form.scrollIntoView({ behavior: "smooth", block: "nearest" });
              focusFormField(form);
            }, 60);
          } else {
            const sheetId = mobileSheet.getAttribute("id");
            if (sheetId) {
              const opener = document.querySelector(`[data-mobile-sheet-open="${sheetId}"]`);
              if (opener instanceof HTMLElement) opener.click();
            }
            window.setTimeout(() => {
              mobileSheet.classList.remove("hidden");
              mobileSheet.setAttribute("aria-hidden", "false");
              focusFormField(form);
            }, 40);
          }
        } else {
          form.closest("details")?.setAttribute("open", "");
          form.scrollIntoView({ behavior: "smooth", block: "center" });
          window.setTimeout(() => focusFormField(form), 260);
        }
      };
      const syncWeekHeaderAlignment = () => {
        if (!reshellItineraryCalendarDom()) return;
        const head = weekView.querySelector(".trip-itinerary-week-head");
        const allday = weekView.querySelector("[data-itinerary-allday-row]");
        const body = weekView.querySelector(".trip-itinerary-week-body");
        if (!head || !body) return;
        const gutter = Math.max(0, body.offsetWidth - body.clientWidth);
        head.style.paddingRight = `${gutter}px`;
        if (allday) allday.style.paddingRight = `${gutter}px`;
      };
      const renderCalendar = () => {
        if (!reshellItineraryCalendarDom()) return;
        state.pxPerMin = computeItineraryPxPerMin();
        const pxm = state.pxPerMin;
        const prevWeekSurface = weekView.querySelector("[data-itinerary-scroll-surface]");
        const prevDaySurface = dayView.querySelector("[data-itinerary-scroll-surface]");
        if (prevWeekSurface) state.weekScrollTop = prevWeekSurface.scrollTop;
        if (prevDaySurface) state.dayScrollTop = prevDaySurface.scrollTop;
        // Day view navigates by calendar day; weekOffset must include selectedDate or render snaps
        // selectedDate back to the first column and Prev/Next appear broken.
        if (state.view === "day") {
          syncWeekOffsetFromSelectedDate();
        }
        const entries = getRenderableEntries();
        const escapeCalEntryTitleHtml = (s) =>
          String(s || "")
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;");
        const escapeCalEntryTitleAttr = (s) =>
          String(s || "")
            .replace(/&/g, "&amp;")
            .replace(/"/g, "&quot;")
            .replace(/</g, "&lt;")
            .replace(/\n/g, " ");
        /** Calendar-only labels (Today / Week): shorten some booking prefixes; keep IATA flight lines and stay/check-in/out as authored. */
        const calendarEntryDisplayTitle = (raw) => {
          let t = String(raw || "").trim();
          if (!t) return t;
          if (/^flight from\s+/i.test(t) || /^flight to\s+/i.test(t)) return t;
          if (/^stay:\s*/i.test(t) || /^check-in:\s*/i.test(t) || /^check-out:\s*/i.test(t)) return t;
          if (/^renting:\s*/i.test(t) || /^pick-up:\s*/i.test(t) || /^drop-off:\s*/i.test(t)) return t;
          const rep = (re, to) => {
            t = t.replace(re, to);
          };
          rep(/^Departure:\s*/i, "Flight: ");
          rep(/^Arrival:\s*/i, "Flight: ");
          rep(/^Pick Up:\s*/i, "Vehicle Rental: ");
          rep(/^Drop Off:\s*/i, "Vehicle Rental: ");
          return t.trim();
        };
        const calEntryIcon = (e) =>
          e.markerKind === "flight"
            ? "flight"
            : e.markerKind === "stay"
              ? "hotel"
              : e.markerKind === "vehicle"
                ? "directions_car"
                : e.markerKind === "commute"
                  ? "directions_transit"
                  : "pin_drop";
        const entryPositionStyle = (e, top, h) => {
          const lane = e.lane || 0;
          const laneCount = Math.max(1, e.laneCount || 1);
          if (laneCount <= 1) {
            return `top:${top}px;height:${h}px;left:6px;right:6px;`;
          }
          const gapPx = 1;
          const seg = `(100% - 12px - ${(laneCount - 1) * gapPx}px)`;
          const w = `calc((${seg}) * ${(1 / laneCount).toFixed(8)})`;
          const l = `calc(6px + (${seg}) * ${(lane / laneCount).toFixed(8)} + ${lane * gapPx}px)`;
          return `top:${top}px;height:${h}px;left:${l};width:${w};right:auto;`;
        };
        const renderCalEntryArticle = (e) => {
          const top = Math.max(0, e.startMin * pxm);
          const h = Math.max(40, (e.endMin - e.startMin) * pxm);
          const lane = e.lane || 0;
          const laneCount = Math.max(1, e.laneCount || 1);
          const crowdedClass = laneCount >= 6 ? " is-crowded" : "";
          const rf = pairedSpanResizeFlags(e);
          const hideStart = rf.allowResizeStart ? "" : " hidden";
          const hideEnd = rf.allowResizeEnd ? "" : " hidden";
          const calTitlePlain = calendarEntryDisplayTitle(e.title);
          const calTitleHtml = escapeCalEntryTitleHtml(calTitlePlain);
          const calTitleAttr = escapeCalEntryTitleAttr(calTitlePlain);
          const timeHtml = escapeCalEntryTitleHtml(`${toTime(e.startMin)} – ${toTime(e.endMin)}`);
          return `<article class="trip-itinerary-entry${crowdedClass}" data-itinerary-drag-item="${e.itemID}" data-kind="${e.markerKind}" data-lane-count="${laneCount}" data-it-lane="${lane}" data-it-lanes="${laneCount}" title="${calTitleAttr}" style="${entryPositionStyle(e, top, h)}">
              <span class="trip-itinerary-resize-handle trip-itinerary-resize-handle--start" data-itinerary-resize-handle="start" aria-hidden="true"${hideStart}></span>
              <div class="trip-itinerary-entry__stack">
                <div class="trip-itinerary-entry__time">${timeHtml}</div>
                <div class="trip-itinerary-entry__row">
                  <span class="material-symbols-outlined trip-itinerary-entry__ico" aria-hidden="true">${calEntryIcon(e)}</span>
                  <span class="trip-itinerary-entry__text">${calTitleHtml}</span>
                </div>
              </div>
              <span class="trip-itinerary-resize-handle trip-itinerary-resize-handle--end" data-itinerary-resize-handle="end" aria-hidden="true"${hideEnd}></span>
            </article>`;
        };
        const renderAlldayChips = (alldayEntries) => {
          if (!alldayEntries.length) return "";
          return alldayEntries
            .map((e) => {
              const titlePlain = calendarEntryDisplayTitle(e.title);
              const t = escapeCalEntryTitleHtml(titlePlain);
              const tAttr = escapeCalEntryTitleAttr(titlePlain);
              const dnd = alldayDragMovable(e) ? ' draggable="true" data-itinerary-allday-dnd="1"' : "";
              return `<div class="trip-itinerary-allday-chip"${dnd} data-itinerary-drag-item="${e.itemID}" data-itinerary-allday-from="${e.date}" data-kind="${e.markerKind}" title="${tAttr}"><div class="trip-itinerary-allday-chip__stack"><div class="trip-itinerary-allday-chip__row"><span class="material-symbols-outlined trip-itinerary-allday-chip__ico" aria-hidden="true">${calEntryIcon(
                e
              )}</span><span class="trip-itinerary-allday-chip__text">${t}</span></div></div></div>`;
            })
            .join("");
        };
        const wDates = weekDates();
        const visibleCols = Math.max(1, wDates.length);
        const compactCols = visibleCols > 1 && visibleCols < 7;
        const colDef = compactCols ? "minmax(0, 1fr)" : "minmax(120px, 1fr)";
        const wf = wDates[0];
        const wl = wDates[wDates.length - 1];
        const selected = toDate(state.selectedDate);
        rangeLabel.textContent = state.view === "day"
          ? (selected
            ? selected.toLocaleDateString(undefined, { weekday: "long", month: "short", day: "numeric" })
            : "Today")
          : (wf && wl
            ? `${wf.toLocaleDateString(undefined, { month: "short", day: "numeric" })} - ${wl.toLocaleDateString(undefined, { month: "short", day: "numeric" })}`
            : "Week");
        const hourLines = Array.from({ length: 25 }, (_, h) =>
          `<div class="trip-itinerary-hour-line" style="top:${h * 60 * pxm}px"></div>`
        ).join("");
        const timeTicks = Array.from({ length: 24 }, (_, h) =>
          `<span style="top:${h * 60 * pxm}px">${pad2(h)}:00</span>`
        ).join("");
        const weekDateKeySet = new Set(wDates.map((d) => fmtDate(d)));
        const buildWeekAlldayRowInner = () => {
          const weekAllday = entries.filter((e) => isAllDayEvent(e) && weekDateKeySet.has(e.date));
          const mergeKeyOf = (e) => {
            if (e.calAccLeg === "stay") return `stay|${String(e.sourceItemID || e.itemID || "").trim()}`;
            if (e.calVehLeg === "renting") return `rent|${String(e.sourceItemID || e.itemID || "").trim()}`;
            return "";
          };
          const groups = new Map();
          const solos = [];
          weekAllday.forEach((e) => {
            const mk = mergeKeyOf(e);
            if (!mk) {
              solos.push(e);
              return;
            }
            if (!groups.has(mk)) groups.set(mk, []);
            groups.get(mk).push(e);
          });
          const segments = [];
          const pushSeg = (startIdx, endIdx, rep) => {
            if (
              typeof startIdx !== "number" ||
              typeof endIdx !== "number" ||
              startIdx < 0 ||
              endIdx < 0 ||
              startIdx >= visibleCols ||
              endIdx >= visibleCols ||
              startIdx > endIdx
            ) {
              return;
            }
            segments.push({ startIdx, endIdx, rep });
          };
          groups.forEach((list) => {
            const byDate = new Map(list.map((ev) => [ev.date, ev]));
            const days = [...byDate.keys()].sort();
            let i = 0;
            while (i < days.length) {
              const startIso = days[i];
              let j = i;
              while (j + 1 < days.length) {
                const a = toDate(days[j]);
                const b = toDate(days[j + 1]);
                if (!a || !b) break;
                if (Math.round((b.getTime() - a.getTime()) / 86400000) !== 1) break;
                j++;
              }
              const endIso = days[j];
              const si = wDates.findIndex((d) => fmtDate(d) === startIso);
              const ei = wDates.findIndex((d) => fmtDate(d) === endIso);
              pushSeg(si, ei, byDate.get(startIso));
              i = j + 1;
            }
          });
          solos.forEach((e) => {
            const si = wDates.findIndex((d) => fmtDate(d) === e.date);
            pushSeg(si, si, e);
          });
          const segs = segments.filter((s) => s.rep);
          segs.sort((a, b) => a.startIdx - b.startIdx || b.endIdx - b.startIdx - (a.endIdx - a.startIdx));
          segs.forEach((seg) => {
            let t = 0;
            while (
              segs.some(
                (o) =>
                  o !== seg &&
                  typeof o.track === "number" &&
                  o.track === t &&
                  seg.startIdx <= o.endIdx &&
                  o.startIdx <= seg.endIdx
              )
            ) {
              t++;
            }
            seg.track = t;
          });
          const maxTrack = segs.reduce((m, s) => Math.max(m, typeof s.track === "number" ? s.track : 0), 0);
          const BAR_H = 30;
          const GAP = 5;
          const TOP_PAD = 6;
          const N = Math.max(1, visibleCols);
          const colsHtml = wDates
            .map((d) => {
              const key = fmtDate(d);
              return `<div class="trip-itinerary-allday-week-col" data-itinerary-allday-date="${key}"></div>`;
            })
            .join("");
          const chipHtmlFor = (e) => {
            const titlePlain = calendarEntryDisplayTitle(e.title);
            const t = escapeCalEntryTitleHtml(titlePlain);
            const tAttr = escapeCalEntryTitleAttr(titlePlain);
            const dnd = alldayDragMovable(e) ? ' draggable="true" data-itinerary-allday-dnd="1"' : "";
            return `<div class="trip-itinerary-allday-chip"${dnd} data-itinerary-drag-item="${e.itemID}" data-itinerary-allday-from="${e.date}" data-kind="${e.markerKind}" title="${tAttr}"><div class="trip-itinerary-allday-chip__stack"><div class="trip-itinerary-allday-chip__row"><span class="material-symbols-outlined trip-itinerary-allday-chip__ico" aria-hidden="true">${calEntryIcon(
              e
            )}</span><span class="trip-itinerary-allday-chip__text">${t}</span></div></div></div>`;
          };
          const barsHtml = segs
            .map((seg) => {
              const span = seg.endIdx - seg.startIdx + 1;
              const leftPct = (seg.startIdx / N) * 100;
              const widthPct = (span / N) * 100;
              const topPx = TOP_PAD + seg.track * (BAR_H + GAP);
              return `<div class="trip-itinerary-allday-week-bar" style="left:${leftPct}%;width:${widthPct}%;top:${topPx}px;height:${BAR_H}px">${chipHtmlFor(seg.rep)}</div>`;
            })
            .join("");
          const stageMinH = TOP_PAD + (maxTrack + 1) * (BAR_H + GAP) + 4;
          return `<div class="trip-itinerary-allday-week-stage trip-itinerary-allday-cell--has" style="grid-column:2 / span ${N};min-height:${stageMinH}px">
            <div class="trip-itinerary-allday-week-cols">${colsHtml}</div>
            <div class="trip-itinerary-allday-week-bars">${barsHtml}</div>
          </div>`;
        };
        const hasAnyAllday = wDates.some((d) => {
          const key = fmtDate(d);
          return entries.some((e) => e.date === key && isAllDayEvent(e));
        });
        const weekCols = wDates.map((d) => {
          const key = fmtDate(d);
          const dayEntries = entries.filter((e) => e.date === key);
          const timed = dayEntries.filter((e) => !isAllDayEvent(e));
          applyCollisionOffsets(timed);
          const cards = timed.map((e) => renderCalEntryArticle(e)).join("");
          const selectedClass = state.selectedDate === key ? " is-selected" : "";
          return `<div class="trip-itinerary-day-col${selectedClass}" data-itinerary-drop-date="${key}" style="min-height:${DAY_MINUTES * pxm}px">${hourLines}${cards}</div>`;
        }).join("");
        const alldayRow = hasAnyAllday
          ? `<div class="trip-itinerary-allday-row" style="grid-template-columns:56px repeat(${visibleCols}, ${colDef})" data-itinerary-allday-row>
            <div class="trip-itinerary-allday-gutter" role="rowheader">All day</div>
            ${buildWeekAlldayRowInner()}
          </div>`
          : "";
        weekView.innerHTML = `<section class="trip-itinerary-calendar-grid${compactCols ? " is-compact-cols" : ""}${hasAnyAllday ? " has-allday-row" : ""}">
          <header class="trip-itinerary-week-head" style="grid-template-columns:56px repeat(${visibleCols}, ${colDef})"><div></div>${wDates
            .map((d) => {
              const key = fmtDate(d);
              const selected = state.selectedDate === key ? " is-selected" : "";
              return `<div class="${selected}" data-itinerary-select-date="${key}">${d.toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" })}</div>`;
            })
            .join("")}</header>
          ${alldayRow}
          <div class="trip-itinerary-week-body" style="grid-template-columns:56px repeat(${visibleCols}, ${colDef})" data-itinerary-scroll-surface><aside class="trip-itinerary-time-col" style="min-height:${DAY_MINUTES * pxm}px">${timeTicks}</aside>${weekCols}</div>
        </section>`;
        let dayDate = state.selectedDate;
        if (!wDates.some((d) => fmtDate(d) === dayDate)) {
          dayDate = wDates[0] ? fmtDate(wDates[0]) : tripStartRaw;
        }
        state.selectedDate = dayDate;
        const dayEntriesRaw = entries.filter((e) => e.date === dayDate);
        const dayAllday = dayEntriesRaw.filter((e) => isAllDayEvent(e));
        const dayTimed = dayEntriesRaw.filter((e) => !isAllDayEvent(e));
        applyCollisionOffsets(dayTimed);
        const dayHeadline = new Date(`${dayDate}T00:00:00`).toLocaleDateString(undefined, {
          weekday: "long",
          month: "short",
          day: "numeric"
        });
        const dayHeadlineAttr = dayHeadline.replace(/&/g, "&amp;").replace(/"/g, "&quot;");
        const dayAlldayHtml = dayAllday.length
          ? `<div class="trip-itinerary-day-allday" role="region" aria-label="All-day events" data-itinerary-allday-date="${dayDate}">${renderAlldayChips(
              dayAllday
            )}</div>`
          : "";
        dayView.innerHTML = `<section class="trip-itinerary-day-wrap${dayAllday.length ? " has-day-allday" : ""}" aria-label="Schedule for ${dayHeadlineAttr}">
          <header class="trip-itinerary-day-head"><small>Drag handles to adjust duration · tap an entry to edit</small></header>
          ${dayAlldayHtml}
          <div class="trip-itinerary-day-body trip-itinerary-day-body--grid" data-itinerary-scroll-surface><aside class="trip-itinerary-time-col" style="min-height:${DAY_MINUTES * pxm}px">${timeTicks}</aside><div class="trip-itinerary-day-col trip-itinerary-day-col--single" data-itinerary-drop-date="${dayDate}" style="min-height:${DAY_MINUTES * pxm}px">${hourLines}${dayTimed.map((e) => renderCalEntryArticle(e)).join("")}</div></div>
        </section>`;
        const nextWeekSurface = weekView.querySelector("[data-itinerary-scroll-surface]");
        const nextDaySurface = dayView.querySelector("[data-itinerary-scroll-surface]");
        if (nextWeekSurface) nextWeekSurface.scrollTop = state.weekScrollTop;
        if (nextDaySurface) nextDaySurface.scrollTop = state.dayScrollTop;
        syncWeekHeaderAlignment();
      };
      const toggleView = (view) => {
        if (!reshellItineraryCalendarDom()) return;
        state.view = view;
        tripCalendarRoot.querySelectorAll("[data-itinerary-view-toggle]").forEach((btn) => {
          const on = btn.getAttribute("data-itinerary-view-toggle") === view;
          btn.classList.toggle("is-active", on);
          btn.setAttribute("aria-selected", on ? "true" : "false");
        });
        const showCalendar = view !== "list";
        listPanel.classList.toggle("hidden", showCalendar);
        toolbar.classList.toggle("hidden", !showCalendar);
        calendarPanels.classList.toggle("hidden", !showCalendar);
        dayView.classList.toggle("hidden", view !== "day");
        weekView.classList.toggle("hidden", view !== "week");
        if (showCalendar) renderCalendar();
      };
      const onResizeCommit = async (entry, nextStart, nextEnd, edge, colEl) => {
        const targetDate = entry.date;
        if (!entry || !targetDate) return;
        const rf = pairedSpanResizeFlags(entry);
        if (edge === "start" && !rf.allowResizeStart) return;
        if (edge === "end" && !rf.allowResizeEnd) return;
        const rollback = { date: entry.date, startMin: entry.startMin, endMin: entry.endMin };
        state.overrides.set(entry.itemID, { date: targetDate, startMin: nextStart, endMin: nextEnd });
        renderCalendar();
        applyResizeToForm(entry, targetDate, nextStart, nextEnd, edge);
        try {
          await submitFormAjax(entry.form);
          await syncAllRenderedItineraryRows();
          state.overrides.delete(entry.itemID);
          renderCalendar();
          showToast("Time updated.");
        } catch (error) {
          state.overrides.set(entry.itemID, rollback);
          renderCalendar();
          state.overrides.delete(entry.itemID);
          showToast(error?.message || "Update failed. Reverted.");
        } finally {
          if (state.dragOverCol) state.dragOverCol.classList.remove("is-drop-target");
          state.dragOverCol = null;
          if (state.dragHintEl) {
            state.dragHintEl.remove();
            state.dragHintEl = null;
          }
          colEl?.classList.remove("is-drop-target");
        }
      };
      const onAllDayMoveCommit = async (entry, toDate) => {
        if (!entry || !toDate) return;
        if (toDate === entry.date) return;
        if (!dateInRange(toDate)) {
          showToast("That date is outside the trip range.");
          return;
        }
        if (!alldayDragMovable(entry)) return;
        const isAcc = entry.category === "accommodation";
        const form = entry.form;
        const accRollback =
          isAcc && form
            ? {
                checkIn: form.querySelector("input.remi-datetime-iso[name='check_in_at']")?.value,
                checkOut: form.querySelector("input.remi-datetime-iso[name='check_out_at']")?.value
              }
            : null;
        const rollback = { date: entry.date, startMin: entry.startMin, endMin: entry.endMin };
        state.overrides.set(entry.itemID, { date: toDate, startMin: 0, endMin: DAY_MINUTES - 1 });
        renderCalendar();
        if (isAcc) {
          if (!applyAccommodationAllDayDateShift(entry, toDate)) {
            state.overrides.delete(entry.itemID);
            renderCalendar();
            return;
          }
        } else {
          applyResizeToForm(entry, toDate, 0, DAY_MINUTES - 1, "end");
        }
        try {
          await submitFormAjax(entry.form);
          await syncAllRenderedItineraryRows();
          state.overrides.delete(entry.itemID);
          state.lastDragEndAt = Date.now();
          renderCalendar();
          showToast(isAcc ? "Stay dates updated." : "Event moved.");
        } catch (error) {
          if (isAcc && accRollback && form) {
            if (accRollback.checkIn != null) setField(form, "check_in_at", accRollback.checkIn);
            if (accRollback.checkOut != null) setField(form, "check_out_at", accRollback.checkOut);
          } else {
            applyResizeToForm(entry, rollback.date, rollback.startMin, rollback.endMin, "end");
          }
          state.overrides.delete(entry.itemID);
          renderCalendar();
          showToast(error?.message || "Update failed. Reverted.");
        }
      };
      const bindCalendarInteractions = () => {
        if (desktopCalFlyout) {
          const desktopExpenseFlyout = document.querySelector("[data-desktop-expense-flyout]");
          desktopCalFlyout.querySelectorAll("[data-desktop-calendar-add]").forEach((btn) => {
            btn.addEventListener("click", (ev) => {
              ev.stopPropagation();
              const kind = btn.getAttribute("data-desktop-calendar-add") || "";
              if (kind === "expense") {
                closeDesktopFlyout();
                window.setTimeout(() => prefillAddForm(kind), 0);
                return;
              }
              prefillAddForm(kind);
              closeDesktopFlyout();
            });
          });
          if (desktopExpenseFlyout instanceof HTMLElement) {
            document.addEventListener("click", (event) => {
              if (desktopExpenseFlyout.classList.contains("hidden")) return;
              const t = event.target;
              if (!(t instanceof HTMLElement)) return;
              if (t.closest("[data-desktop-expense-flyout]")) return;
              desktopExpenseFlyout.classList.add("hidden");
              desktopExpenseFlyout.setAttribute("aria-hidden", "true");
            });
            document.addEventListener("keydown", (event) => {
              if (event.key !== "Escape") return;
              if (desktopExpenseFlyout.classList.contains("hidden")) return;
              desktopExpenseFlyout.classList.add("hidden");
              desktopExpenseFlyout.setAttribute("aria-hidden", "true");
            });
          }
          document.addEventListener("click", (event) => {
            if (desktopCalFlyout.classList.contains("hidden")) return;
            const target = event.target;
            if (!(target instanceof HTMLElement)) return;
            if (target.closest("[data-desktop-calendar-flyout]")) return;
            if (target.closest("[data-itinerary-drop-date]")) return;
            closeDesktopFlyout();
          });
          document.addEventListener("keydown", (event) => {
            if (event.key === "Escape") closeDesktopFlyout();
          });
        }
        const clearDropHint = () => {
          if (state.dragHintEl) {
            state.dragHintEl.remove();
            state.dragHintEl = null;
          }
          state.dragHintMinute = null;
        };
        const updateDropHint = (col, rawY) => {
          if (!col) {
            clearDropHint();
            return;
          }
          const pxm = state.pxPerMin;
          const snapped = Math.max(0, Math.min(DAY_MINUTES - 15, Math.round(rawY / pxm / 15) * 15));
          const top = snapped * pxm;
          if (!state.dragHintEl || state.dragHintEl.parentElement !== col) {
            clearDropHint();
            const hint = document.createElement("div");
            hint.className = "trip-itinerary-drop-hint";
            hint.innerHTML = '<span class="trip-itinerary-drop-hint__badge"></span>';
            col.appendChild(hint);
            state.dragHintEl = hint;
          }
          state.dragHintEl.style.top = `${top}px`;
          const badge = state.dragHintEl.querySelector(".trip-itinerary-drop-hint__badge");
          if (badge) badge.textContent = toTime(snapped);
          if (state.dragHintMinute !== null && state.dragHintMinute !== snapped) {
            state.dragHintEl.classList.remove("is-snap-pulse");
            // Restart animation on every 15-min boundary crossing.
            void state.dragHintEl.offsetWidth;
            state.dragHintEl.classList.add("is-snap-pulse");
          }
          state.dragHintMinute = snapped;
        };
        if (!window.__remiItineraryCalViewNavDelegation) {
          window.__remiItineraryCalViewNavDelegation = true;
          document.addEventListener("click", (e) => {
            const root = e.target?.closest?.("[data-itinerary-calendar-root]");
            if (!root) return;
            const viewBtn = e.target?.closest?.("[data-itinerary-view-toggle]");
            if (viewBtn && root.contains(viewBtn)) {
              if (!reshellItineraryCalendarDom()) return;
              toggleView(viewBtn.getAttribute("data-itinerary-view-toggle") || "list");
              return;
            }
            const weekNavBtn = e.target?.closest?.("[data-itinerary-week-nav]");
            if (!weekNavBtn || !root.contains(weekNavBtn)) return;
            if (!reshellItineraryCalendarDom()) return;
            const mode = weekNavBtn.getAttribute("data-itinerary-week-nav") || "";
            if (mode === "today") {
              const t = fmtDate(new Date());
              state.selectedDate = clampISO(t);
              syncWeekOffsetFromSelectedDate();
              renderCalendar();
              return;
            }
            const dir = mode === "next" ? 1 : -1;
            if (state.view === "day") {
              state.selectedDate = clampISO(shiftISO(state.selectedDate, dir));
              syncWeekOffsetFromSelectedDate();
              renderCalendar();
              return;
            }
            const maxOffset = Math.max(0, Math.ceil(totalTripDays / 7) * 7 - 7);
            state.weekOffset = Math.max(0, Math.min(maxOffset, state.weekOffset + dir * 7));
            if (!dateInRange(state.selectedDate)) {
              state.selectedDate = clampISO(state.selectedDate);
            }
            renderCalendar();
          });
        }
        const attachTripCalendarRootListeners = () => {
          if (!tripCalendarRoot || tripCalendarRoot.dataset.remiItineraryCalRootBound === "1") return;
          tripCalendarRoot.dataset.remiItineraryCalRootBound = "1";
        tripCalendarRoot.addEventListener("click", (e) => {
          const dayBtn = e.target?.closest?.("[data-itinerary-select-date]");
          if (!dayBtn) return;
          state.selectedDate = dayBtn.getAttribute("data-itinerary-select-date") || state.selectedDate;
          if (state.view !== "list") renderCalendar();
        });
        tripCalendarRoot.addEventListener("click", (e) => {
          if (!desktopCalFlyout) return;
          if (state.view === "list") return;
          if (e.target?.closest?.("[data-itinerary-drag-item]")) return;
          const col = e.target?.closest?.("[data-itinerary-drop-date]");
          if (!col) return;
          const date = col.getAttribute("data-itinerary-drop-date") || "";
          if (!date) return;
          const rect = col.getBoundingClientRect();
          const y = e.clientY - rect.top;
          const pxm = state.pxPerMin;
          const snapped = Math.max(0, Math.min(DAY_MINUTES - 15, Math.round(y / pxm / 15) * 15));
          openDesktopFlyout(date, snapped, e.clientX, e.clientY);
        });
        const autoScrollDragSurface = (surface, clientY) => {
          if (!surface) return;
          const r = surface.getBoundingClientRect();
          const edge = 64;
          let dy = 0;
          if (clientY < r.top + edge) dy = -18;
          else if (clientY > r.bottom - edge) dy = 18;
          if (dy !== 0) surface.scrollTop += dy;
        };
        const clearResizeState = () => {
          const rs = state.resizeState;
          if (!rs) return;
          rs.card?.classList.remove("is-resizing");
          if (rs.card) {
            rs.card.style.top = rs.origTop || "";
            rs.card.style.height = rs.origHeight || "";
          }
          state.resizeState = null;
          clearDropHint();
          if (state.dragOverCol) state.dragOverCol.classList.remove("is-drop-target");
          state.dragOverCol = null;
        };
        tripCalendarRoot.addEventListener("pointerdown", (e) => {
          closeDesktopFlyout();
          const handle = e.target?.closest?.("[data-itinerary-resize-handle]");
          if (!handle) return;
          const card = handle.closest("[data-itinerary-drag-item]");
          if (!card) return;
          const itemID = card.getAttribute("data-itinerary-drag-item") || "";
          const entry = getRenderableEntries().find((x) => x.itemID === itemID);
          if (!entry) return;
          const edge = handle.getAttribute("data-itinerary-resize-handle") === "start" ? "start" : "end";
          const rf = pairedSpanResizeFlags(entry);
          if (edge === "start" && !rf.allowResizeStart) return;
          if (edge === "end" && !rf.allowResizeEnd) return;
          e.preventDefault();
          e.stopPropagation();
          state.resizeState = {
            pointerId: e.pointerId,
            card,
            entry,
            edge,
            startY: e.clientY,
            startMin: entry.startMin,
            endMin: entry.endMin,
            origTop: card.style.top,
            origHeight: card.style.height
          };
          card.classList.add("is-resizing");
          card.setPointerCapture?.(e.pointerId);
          state.lastDragEndAt = Date.now();
        }, { passive: false });
        tripCalendarRoot.addEventListener("pointermove", (e) => {
          const rs = state.resizeState;
          if (!rs || rs.pointerId !== e.pointerId) return;
          e.preventDefault();
          const col = rs.card.closest("[data-itinerary-drop-date]");
          if (!col) return;
          const surface = col.closest("[data-itinerary-scroll-surface]");
          autoScrollDragSurface(surface, e.clientY);
          const pxm = state.pxPerMin;
          const delta = Math.round((e.clientY - rs.startY) / pxm / 15) * 15;
          let nextStart = rs.startMin;
          let nextEnd = rs.endMin;
          if (rs.edge === "start") {
            nextStart = Math.max(0, Math.min(nextEnd - 15, rs.startMin + delta));
          } else {
            nextEnd = Math.min(DAY_MINUTES, Math.max(nextStart + 15, rs.endMin + delta));
          }
          const topPx = nextStart * pxm;
          const heightPx = Math.max(40, (nextEnd - nextStart) * pxm);
          rs.nextStart = nextStart;
          rs.nextEnd = nextEnd;
          rs.card.style.top = `${topPx}px`;
          rs.card.style.height = `${heightPx}px`;
          state.dragOverCol = col;
          col.classList.add("is-drop-target");
          updateDropHint(col, rs.edge === "start" ? topPx : (nextEnd * pxm));
          if (surface) {
            if (surface === weekView.querySelector("[data-itinerary-scroll-surface]")) state.weekScrollTop = surface.scrollTop;
            if (surface === dayView.querySelector("[data-itinerary-scroll-surface]")) state.dayScrollTop = surface.scrollTop;
          }
        }, { passive: false });
        const finishPointerDrop = (e, cancelled) => {
          const rs = state.resizeState;
          if (!rs || rs.pointerId !== e.pointerId) return;
          const col = rs.card.closest("[data-itinerary-drop-date]");
          const hasChange = rs.nextStart !== undefined && rs.nextEnd !== undefined
            && (rs.nextStart !== rs.startMin || rs.nextEnd !== rs.endMin);
          const edge = rs.edge;
          const entry = rs.entry;
          const nextStart = rs.nextStart ?? rs.startMin;
          const nextEnd = rs.nextEnd ?? rs.endMin;
          clearResizeState();
          if (!cancelled && hasChange) {
            void onResizeCommit(entry, nextStart, nextEnd, edge, col);
            state.lastDragEndAt = Date.now();
            e.preventDefault();
          }
        };
        tripCalendarRoot.addEventListener("pointerup", (e) => finishPointerDrop(e, false), { passive: false });
        tripCalendarRoot.addEventListener("pointercancel", (e) => finishPointerDrop(e, true), { passive: true });
        const alldayDropTargetFromEvent = (e) =>
          e?.target?.closest?.(
            ".trip-itinerary-allday-cell, .trip-itinerary-allday-week-col[data-itinerary-allday-date], .trip-itinerary-day-allday[data-itinerary-allday-date]"
          );
        const clearAlldayDropHighlights = () => {
          if (!tripCalendarRoot) return;
          tripCalendarRoot
            .querySelectorAll(
              ".trip-itinerary-allday-cell.is-drop-target, .trip-itinerary-allday-week-col.is-drop-target, .trip-itinerary-day-allday.is-drop-target"
            )
            .forEach((c) => c.classList.remove("is-drop-target"));
        };
        tripCalendarRoot.addEventListener("dragstart", (e) => {
          const chip = e.target?.closest?.(".trip-itinerary-allday-chip[data-itinerary-allday-dnd='1']");
          if (!chip) return;
          const itemID = chip.getAttribute("data-itinerary-drag-item") || "";
          const from = chip.getAttribute("data-itinerary-allday-from") || "";
          const entry = getRenderableEntries().find((x) => x.itemID === itemID);
          if (!alldayDragMovable(entry)) {
            e.preventDefault();
            return;
          }
          state.alldayDnD = { itemID, from };
          e.dataTransfer.setData("text/plain", "remi-allday");
          e.dataTransfer.effectAllowed = "move";
          chip.classList.add("is-dragging");
        });
        tripCalendarRoot.addEventListener("dragend", (e) => {
          const chip = e.target?.closest?.(".trip-itinerary-allday-chip");
          chip?.classList.remove("is-dragging");
          state.alldayDnD = null;
          clearAlldayDropHighlights();
        });
        tripCalendarRoot.addEventListener("dragover", (e) => {
          if (!state.alldayDnD) return;
          const cell = alldayDropTargetFromEvent(e);
          e.preventDefault();
          if (e.dataTransfer) e.dataTransfer.dropEffect = "move";
          clearAlldayDropHighlights();
          if (cell) cell.classList.add("is-drop-target");
        });
        tripCalendarRoot.addEventListener("drop", (e) => {
          const cell = alldayDropTargetFromEvent(e);
          if (!cell || !state.alldayDnD) return;
          e.preventDefault();
          e.stopPropagation();
          const toDate = cell.getAttribute("data-itinerary-allday-date") || "";
          const dnd = state.alldayDnD;
          state.alldayDnD = null;
          clearAlldayDropHighlights();
          if (!toDate || toDate === dnd.from) return;
          const entry = getRenderableEntries().find((x) => x.itemID === dnd.itemID);
          if (!alldayDragMovable(entry)) return;
          void onAllDayMoveCommit(entry, toDate);
          state.lastDragEndAt = Date.now();
        });
        const stripPreviewCloneIds = (root) => {
          if (!root) return;
          root.removeAttribute("id");
          root.querySelectorAll("[id]").forEach((n) => n.removeAttribute("id"));
        };
        const escapeAttrSelector = (id) => {
          const s = String(id || "");
          if (typeof CSS !== "undefined" && typeof CSS.escape === "function") return CSS.escape(s);
          return s.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
        };
        const bookingPreviewLegOrder = { depart: 0, checkin: 0, pickup: 0, arrive: 1, checkout: 1, dropoff: 1 };
        const legSortKey = (r) => bookingPreviewLegOrder[(r.getAttribute("data-itinerary-booking-leg") || "").trim().toLowerCase()] ?? 99;
        const returnFormsFromQuickEditWrap = () => {
          if (!quickEditFormWrap) return;
          quickEditFormWrap.querySelectorAll("form").forEach((f) => {
            const fid = f.id || "";
            let viewId = null;
            if (fid.startsWith("accommodation-itinerary-edit-")) {
              viewId = `itinerary-view-${fid.slice("accommodation-itinerary-edit-".length)}`;
            } else if (fid.startsWith("vehicle-rental-itinerary-edit-")) {
              viewId = `itinerary-view-${fid.slice("vehicle-rental-itinerary-edit-".length)}`;
            } else if (fid.startsWith("flight-itinerary-edit-")) {
              viewId = `itinerary-view-${fid.slice("flight-itinerary-edit-".length)}`;
            } else if (fid.startsWith("commute-itinerary-edit-")) {
              viewId = `itinerary-view-${fid.slice("commute-itinerary-edit-".length)}`;
            } else if (fid.startsWith("itinerary-edit-")) {
              viewId = `itinerary-view-${fid.slice("itinerary-edit-".length)}`;
            }
            if (!viewId) return;
            const viewEl = document.getElementById(viewId);
            const r = viewEl?.closest(".timeline-item");
            if (r) {
              r.appendChild(f);
              f.classList.add("hidden");
            }
          });
          quickEditFormWrap.replaceChildren();
        };
        tripCalendarRoot.addEventListener("click", (e) => {
          if (e.target?.closest?.("[data-itinerary-resize-handle]")) return;
          const card = e.target?.closest?.("[data-itinerary-drag-item]");
          if (!card || !quickEditModal || !quickEditBody || !quickEditPreview || !quickEditFormWrap) return;
          if (Date.now() - state.lastDragEndAt < 180) return;
          const entry = getRenderableEntries().find((x) => x.itemID === (card.getAttribute("data-itinerary-drag-item") || ""));
          if (!entry) return;
          const form = entry.form;
          const row = entry.row;

          const resetQuickEditChrome = () => {
            if (quickEditModeLabel) quickEditModeLabel.textContent = "Quick view";
            const editBtn = quickEditModal.querySelector("[data-itinerary-quick-edit-to-edit]");
            if (editBtn) editBtn.removeAttribute("hidden");
            if (quickEditPanel) quickEditPanel.setAttribute("aria-label", "Itinerary quick view");
          };

          const closeModal = () => {
            if (calendarQuickEditUiAbort) {
              calendarQuickEditUiAbort.abort();
              calendarQuickEditUiAbort = null;
            }
            detachItineraryQuickEditEscape();
            returnFormsFromQuickEditWrap();
            quickEditPreview.innerHTML = "";
            quickEditFormWrap.classList.add("hidden");
            quickEditPreview.classList.remove("hidden");
            resetQuickEditChrome();
            delete quickEditModal.remiQuickEditSession;
            if (quickEditBody) {
              delete quickEditBody.remiQuickEditPreviewBackup;
              delete quickEditBody.remiQuickEditTitleBackup;
            }
            quickEditModal.classList.add("hidden");
            quickEditModal.setAttribute("aria-hidden", "true");
          };

          const openEditMode = () => {
            if (quickEditPreview && quickEditBody) {
              quickEditBody.remiQuickEditPreviewBackup = quickEditPreview.innerHTML;
              if (quickEditTitle) quickEditBody.remiQuickEditTitleBackup = quickEditTitle.textContent;
            }
            quickEditPreview.innerHTML = "";
            quickEditPreview.classList.add("hidden");
            quickEditFormWrap.classList.remove("hidden");
            quickEditFormWrap.appendChild(form);
            form.classList.remove("hidden");
            if (quickEditModeLabel) quickEditModeLabel.textContent = "Edit";
            const editBtn = quickEditModal.querySelector("[data-itinerary-quick-edit-to-edit]");
            if (editBtn) editBtn.setAttribute("hidden", "");
            if (quickEditPanel) quickEditPanel.setAttribute("aria-label", "Edit itinerary item");

            const onSubmit = async (ev) => {
              ev.preventDefault();
              ev.stopImmediatePropagation();
              try {
                await submitFormAjax(form);
                await syncAllRenderedItineraryRows();
                closeModal();
                renderCalendar();
                showToast("Changes saved.");
              } catch (err) {
                showToast(err?.message || "Unable to save right now.");
              }
            };
            form.addEventListener("submit", onSubmit, { once: true, capture: true });
          };

          returnFormsFromQuickEditWrap();
          if (calendarQuickEditUiAbort) {
            calendarQuickEditUiAbort.abort();
            calendarQuickEditUiAbort = null;
          }
          calendarQuickEditUiAbort = new AbortController();
          const quickEditUiSignal = calendarQuickEditUiAbort.signal;

          const bookingId = row?.getAttribute("data-itinerary-booking-id") || "";
          const mk = row?.getAttribute("data-marker-kind") || "";
          const rawEntryTitle = String(entry.title || "").trim();
          let modalTitle = rawEntryTitle;
          if (mk === "stay") {
            modalTitle =
              rawEntryTitle
                .replace(/^Stay:\s*/i, "")
                .replace(/^Check-in:\s*/i, "")
                .replace(/^Check-out:\s*/i, "")
                .trim() || rawEntryTitle;
          } else if (mk === "vehicle") {
            modalTitle =
              rawEntryTitle
                .replace(/^Renting:\s*/i, "")
                .replace(/^Pick-up:\s*/i, "")
                .replace(/^Drop-off:\s*/i, "")
                .trim() || rawEntryTitle;
          }
          if (quickEditTitle) quickEditTitle.textContent = modalTitle ? ` ${modalTitle}` : "";
          quickEditPreview.innerHTML = "";
          quickEditFormWrap.classList.add("hidden");
          quickEditPreview.classList.remove("hidden");

          const useCombinedPreview =
            Boolean(listPanel) &&
            bookingId &&
            (mk === "flight" || mk === "stay" || mk === "vehicle");
          if (useCombinedPreview) {
            const peers = Array.from(
              listPanel.querySelectorAll(`[data-itinerary-booking-id="${escapeAttrSelector(bookingId)}"]`)
            ).filter((r) => (r.getAttribute("data-marker-kind") || "") === mk);
            /* Each leg row renders the full booking card; cloning every peer duplicates the UI. */
            if (peers.length > 1) {
              peers.sort((a, b) => legSortKey(a) - legSortKey(b));
              for (const peer of peers) {
                const vr = peer.querySelector(".itinerary-item-view");
                if (!vr) continue;
                const c = vr.cloneNode(true);
                stripPreviewCloneIds(c);
                quickEditPreview.appendChild(c);
                break;
              }
            }
          }
          if (!quickEditPreview.firstChild) {
            const viewRoot = row?.querySelector(".itinerary-item-view");
            if (viewRoot) {
              const clone = viewRoot.cloneNode(true);
              stripPreviewCloneIds(clone);
              quickEditPreview.appendChild(clone);
            } else {
              const fallback = document.createElement("div");
              fallback.className = "trip-itinerary-quick-edit__preview-fallback";
              const titleEl = document.createElement("p");
              titleEl.className = "trip-itinerary-quick-edit__preview-fallback-title";
              titleEl.textContent = entry.title;
              fallback.appendChild(titleEl);
              const metaEl = document.createElement("p");
              metaEl.className = "muted";
              metaEl.textContent = `${toTime(entry.startMin)} – ${toTime(entry.endMin)}`;
              fallback.appendChild(metaEl);
              quickEditPreview.appendChild(fallback);
            }
          }

          resetQuickEditChrome();
          quickEditModal.remiQuickEditSession = {
            closeModal,
            openEditMode,
            signal: quickEditUiSignal
          };
          quickEditModal.classList.remove("hidden");
          quickEditModal.setAttribute("aria-hidden", "false");

          quickEditModal.querySelectorAll("[data-itinerary-quick-edit-close]").forEach((btn) => {
            btn.addEventListener("click", closeModal, { once: true, signal: quickEditUiSignal });
          });
          const toEditBtn = quickEditModal.querySelector("[data-itinerary-quick-edit-to-edit]");
          if (toEditBtn) {
            toEditBtn.addEventListener("click", () => openEditMode(), { once: true, signal: quickEditUiSignal });
          }
          attachItineraryQuickEditEscape(closeModal);
        });
        };
        attachTripCalendarRootListeners();
        window.remiRewireTripItineraryCalendarAfterLiveRefresh = () => {
          if (!reshellItineraryCalendarDom()) return;
          attachTripCalendarRootListeners();
        };
      };
      const startChangePolling = () => {
        if (window.remiRealtimeTripSyncEnabled) return;
        if (!tripID) return;
        const POLL_BASE_MS = 8000;
        const POLL_HIDDEN_MS = 20000;
        const POLL_MAX_MS = 45000;
        const POLL_BUSY_MS = 4000;
        let pollFailures = 0;
        let pollInFlight = false;
        let pollTimer = null;
        const schedule = (ms) => {
          if (pollTimer) window.clearTimeout(pollTimer);
          pollTimer = window.setTimeout(runPoll, Math.max(800, ms));
        };
        const nextDelay = () => {
          if (document.hidden) return POLL_HIDDEN_MS;
          if (pollFailures <= 0) return POLL_BASE_MS;
          return Math.min(POLL_MAX_MS, POLL_BASE_MS * Math.pow(2, pollFailures));
        };
        const quickEditOpen = () =>
          Boolean(quickEditModal) &&
          !quickEditModal.classList.contains("hidden") &&
          quickEditModal.getAttribute("aria-hidden") !== "true";
        const runPoll = async () => {
          if (pollInFlight) {
            schedule(POLL_BUSY_MS);
            return;
          }
          if (state.resizeState || state.alldayDnD || quickEditOpen()) {
            schedule(POLL_BUSY_MS);
            return;
          }
          pollInFlight = true;
          try {
            const url = `/api/v1/trips/${encodeURIComponent(tripID)}/changes${state.since ? `?since=${encodeURIComponent(state.since)}` : ""}`;
            const res = await fetch(url, { credentials: "same-origin", headers: { Accept: "application/json" } });
            if (!res.ok) {
              pollFailures = Math.min(5, pollFailures + 1);
              return;
            }
            const body = await res.json();
            const changes = Array.isArray(body?.changes) ? body.changes : [];
            if (changes.length) {
              const last = changes[changes.length - 1];
              if (last?.changed_at) state.since = String(last.changed_at);
              await syncAllRenderedItineraryRows();
              if (state.view !== "list") renderCalendar();
            }
            pollFailures = 0;
          } catch (e) {
            pollFailures = Math.min(5, pollFailures + 1);
          } finally {
            pollInFlight = false;
            schedule(nextDelay());
          }
        };
        document.addEventListener("visibilitychange", () => {
          if (!document.hidden) schedule(600);
        });
        schedule(POLL_BASE_MS);
      };
      let itineraryCalResizeT = null;
      const onItineraryLayoutResize = () => {
        if (itineraryCalResizeT) clearTimeout(itineraryCalResizeT);
        itineraryCalResizeT = setTimeout(() => {
          itineraryCalResizeT = null;
          syncWeekHeaderAlignment();
          if (state.view === "day" || state.view === "week") renderCalendar();
        }, 150);
      };
      bindCalendarInteractions();
      renderCalendar();
      window.addEventListener("resize", onItineraryLayoutResize);
      startChangePolling();
      toggleView("list");
      }
    }
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
                    const placeId = String(item.placeId || item.place_id || "").trim();
                    const placeName = String(item.placeName || item.place_name || shortName || "").trim();
                    const formattedAddress = String(
                      item.formattedAddress || item.formatted_address || displayName || ""
                    ).trim();
                    return {
                      lat,
                      lng,
                      displayName,
                      shortName,
                      placeId,
                      placeName,
                      formattedAddress
                    };
                  })
                  .filter((item) => {
                    const label = item.displayName || item.shortName || item.placeName;
                    if (!label) return false;
                    if (item.placeId) return true;
                    return !Number.isNaN(item.lat) && !Number.isNaN(item.lng) && (item.lat !== 0 || item.lng !== 0);
                  });
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
              return {
                lat,
                lng,
                displayName,
                shortName,
                placeId: "",
                placeName: shortName,
                formattedAddress: displayName
              };
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

  /** Home dashboard "Start a New Adventure": end date snaps to start when start changes; end cannot be before start. */
  const remiWireDashboardHeroTripDates = (form) => {
    if (!(form instanceof HTMLFormElement)) return;
    const startH = form.querySelector('input.remi-date-iso[name="start_date"]');
    const endH = form.querySelector('input.remi-date-iso[name="end_date"]');
    const endWrap = endH?.closest?.("[data-remi-date]");
    if (!(startH instanceof HTMLInputElement) || !(endH instanceof HTMLInputElement) || !(endWrap instanceof HTMLElement)) {
      return;
    }
    const ISO_D = /^\d{4}-\d{2}-\d{2}$/;
    const syncNativeEndMin = () => {
      const s = String(startH.value || "").trim();
      const endVis = endWrap.querySelector(".remi-date-visible");
      if (endVis instanceof HTMLInputElement && endVis.type === "date") {
        if (ISO_D.test(s)) endVis.min = s;
        else endVis.removeAttribute("min");
      }
    };
    const applyStartAsMinOnEnd = () => {
      const s = String(startH.value || "").trim();
      if (ISO_D.test(s)) {
        endWrap.setAttribute("data-min", s);
      } else {
        endWrap.removeAttribute("data-min");
      }
      syncNativeEndMin();
    };
    const snapEndToStart = () => {
      const s = String(startH.value || "").trim();
      if (!ISO_D.test(s)) return;
      endH.value = s;
      endH.dispatchEvent(new Event("change", { bubbles: true }));
    };
    startH.addEventListener("change", () => {
      applyStartAsMinOnEnd();
      snapEndToStart();
    });
    applyStartAsMinOnEnd();
    if (ISO_D.test(String(startH.value || "").trim())) {
      snapEndToStart();
    }
  };

  const scheduleDashboardHeroTripDates = () => {
    document.querySelectorAll("form[data-dashboard-trip-place]").forEach((form) => {
      if (form instanceof HTMLFormElement) remiWireDashboardHeroTripDates(form);
    });
  };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", scheduleDashboardHeroTripDates, { once: true });
  } else {
    scheduleDashboardHeroTripDates();
  }

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
    /* Fab flyout edit clones: must map to the same itinerary-view-* as the real form (and before non-flyout prefixes). */
    if (formId.startsWith("itinerary-edit-flyout-")) {
      return `itinerary-view-${formId.slice("itinerary-edit-flyout-".length)}`;
    }
    if (formId.startsWith("accommodation-itinerary-edit-flyout-")) {
      return `itinerary-view-${formId.slice("accommodation-itinerary-edit-flyout-".length)}`;
    }
    if (formId.startsWith("vehicle-rental-itinerary-edit-flyout-")) {
      return `itinerary-view-${formId.slice("vehicle-rental-itinerary-edit-flyout-".length)}`;
    }
    if (formId.startsWith("flight-itinerary-edit-flyout-")) {
      return `itinerary-view-${formId.slice("flight-itinerary-edit-flyout-".length)}`;
    }
    if (formId.startsWith("commute-itinerary-edit-flyout-")) {
      return `itinerary-view-${formId.slice("commute-itinerary-edit-flyout-".length)}`;
    }
    if (formId.startsWith("accommodation-itinerary-edit-")) {
      return `itinerary-view-${formId.slice("accommodation-itinerary-edit-".length)}`;
    }
    if (formId.startsWith("vehicle-rental-itinerary-edit-")) {
      return `itinerary-view-${formId.slice("vehicle-rental-itinerary-edit-".length)}`;
    }
    if (formId.startsWith("flight-itinerary-edit-")) {
      return `itinerary-view-${formId.slice("flight-itinerary-edit-".length)}`;
    }
    if (formId.startsWith("commute-itinerary-edit-")) {
      return `itinerary-view-${formId.slice("commute-itinerary-edit-".length)}`;
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
      const quickEditBodyEl = document.querySelector("[data-itinerary-quick-edit-body]");
      const quickEditModalEl = document.querySelector("[data-itinerary-quick-edit-modal]");
      if (quickEditBodyEl && quickEditModalEl && quickEditBodyEl.contains(form)) {
        detachItineraryQuickEditEscape();
        const viewId = itineraryViewIdForForm(formId);
        const view = viewId ? document.getElementById(viewId) : null;
        const row = view?.closest(".timeline-item");
        if (row) row.appendChild(form);
        if (typeof form.reset === "function") form.reset();
        form.classList.add("hidden");
        if (row) row.classList.remove("editing");
        if (view) view.classList.remove("hidden");
        const previewPane = quickEditBodyEl.querySelector("[data-itinerary-quick-edit-preview]");
        const formWrapEl = quickEditBodyEl.querySelector("[data-itinerary-quick-edit-form-wrap]");
        const modeLbl = quickEditModalEl.querySelector("[data-itinerary-quick-edit-mode-label]");
        const editBtnReset = quickEditModalEl.querySelector("[data-itinerary-quick-edit-to-edit]");
        const panelEl = quickEditModalEl.querySelector("[data-itinerary-quick-edit-panel]");
        const titleEl = quickEditModalEl.querySelector("[data-itinerary-quick-edit-title]");
        if (previewPane) {
          const backupHtml = quickEditBodyEl.remiQuickEditPreviewBackup;
          previewPane.innerHTML = typeof backupHtml === "string" ? backupHtml : "";
          previewPane.classList.remove("hidden");
        }
        if (titleEl && quickEditBodyEl.remiQuickEditTitleBackup != null) {
          titleEl.textContent = quickEditBodyEl.remiQuickEditTitleBackup;
        }
        if (formWrapEl) formWrapEl.classList.add("hidden");
        if (modeLbl) modeLbl.textContent = "Quick view";
        if (editBtnReset) editBtnReset.removeAttribute("hidden");
        if (panelEl) panelEl.setAttribute("aria-label", "Itinerary quick view");
        quickEditModalEl.classList.remove("hidden");
        quickEditModalEl.setAttribute("aria-hidden", "false");
        const sess = quickEditModalEl.remiQuickEditSession;
        if (sess?.closeModal && sess?.openEditMode && sess?.signal) {
          attachItineraryQuickEditEscape(sess.closeModal);
          if (editBtnReset) {
            editBtnReset.addEventListener("click", () => sess.openEditMode(), { once: true, signal: sess.signal });
          }
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
      if (window.remiHandleDesktopTripFlyoutEdit?.(form, formId)) return;
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
      if (dateInput instanceof HTMLInputElement) dateInput.dataset.originalValue = dateInput.value;
      const startAtSnap = form.querySelector("input.remi-datetime-iso[name='start_at']");
      if (startAtSnap instanceof HTMLInputElement && !dateInput) {
        startAtSnap.dataset.originalValue = startAtSnap.value;
      }
      if (form.classList.contains("remi-checklist-inline-edit-form")) {
        window.requestAnimationFrame(() => {
          form.querySelector("input[name='text']")?.focus();
        });
      }
    });
    const tag = (btn.tagName || "").toLowerCase();
    if (tag !== "button" && tag !== "a") {
      btn.addEventListener("keydown", (e) => {
        if (e.key !== "Enter" && e.key !== " ") return;
        e.preventDefault();
        btn.click();
      });
    }
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
    const flyoutEditBody = btn.closest(".remi-trip-flyout-edit-body");
    closeInlineEdit(formId);
    if (flyoutEditBody instanceof HTMLElement) {
      const sheet = flyoutEditBody.closest(".mobile-sheet");
      if (sheet instanceof HTMLElement) {
        clearDirtyStateForSheet(sheet);
      }
      window.remiCloseDrawer?.();
    }
  });

  document.addEventListener("click", (event) => {
    const btn = event.target?.closest?.("[data-trip-keep-note-edit-cancel]");
    if (!(btn instanceof HTMLElement)) return;
    const details = btn.closest("details.trip-keep-details");
    if (!(details instanceof HTMLDetailsElement)) return;
    const form = details.querySelector("form.trip-keep-form");
    if (form instanceof HTMLFormElement) {
      form.reset();
    }
    details.open = false;
    details.removeAttribute("open");
  });

  document.addEventListener("click", (event) => {
    const btn = event.target?.closest?.("[data-trip-keep-note-open-edit]");
    if (!(btn instanceof HTMLElement)) return;
    const id = btn.getAttribute("data-trip-keep-note-open-edit");
    if (!id) return;
    const det = document.getElementById(`keep-note-details-${id}`);
    if (det instanceof HTMLDetailsElement) {
      det.open = true;
      window.requestAnimationFrame(() => {
        det.querySelector("input:not([type='hidden']), textarea")?.focus();
      });
    }
  });

  (function initTripKeepNoteViewDialog() {
    const dialog = document.getElementById("trip-keep-note-view-dialog");
    if (!(dialog instanceof HTMLDialogElement)) return;
    const titleEl = dialog.querySelector("#trip-keep-note-view-dialog-title");
    const bodyEl = dialog.querySelector("#trip-keep-note-view-dialog-body");
    const close = () => dialog.close();
    dialog.querySelector("[data-trip-keep-note-view-close]")?.addEventListener("click", close);
    dialog.addEventListener("click", (e) => {
      if (e.target === dialog) close();
    });
    dialog.addEventListener("close", () => {
      if (bodyEl) bodyEl.textContent = "";
    });
    document.addEventListener("click", (e) => {
      const btn = e.target?.closest?.("[data-trip-keep-note-view-more]");
      if (!(btn instanceof HTMLElement)) return;
      e.preventDefault();
      const tid = btn.getAttribute("data-note-template-id");
      const tpl = tid ? document.getElementById(tid) : null;
      if (!(tpl instanceof HTMLTemplateElement)) return;
      const fullText = tpl.content.textContent ?? "";
      const rawTitle = (btn.getAttribute("data-note-title") || "").trim();
      if (titleEl) titleEl.textContent = rawTitle || "Note";
      if (bodyEl) bodyEl.textContent = fullText;
      dialog.showModal();
    });
  })();

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

  const initAddStopTabs = () => {
    document.querySelectorAll("[data-add-stop-tabs]").forEach((host) => {
      if (!(host instanceof HTMLElement)) return;
      if (host.dataset.addStopTabsInit === "1") return;
      host.dataset.addStopTabsInit = "1";
      const tabs = host.querySelectorAll("[data-add-stop-tab]");
      const panels = host.querySelectorAll("[data-add-stop-panel]");
      const activate = (id) => {
        tabs.forEach((btn) => {
          if (!(btn instanceof HTMLElement)) return;
          const on = btn.getAttribute("data-add-stop-tab") === id;
          btn.setAttribute("aria-selected", on ? "true" : "false");
          btn.classList.toggle("add-stop-tabs__tab--active", on);
          btn.tabIndex = on ? 0 : -1;
        });
        panels.forEach((panel) => {
          if (!(panel instanceof HTMLElement)) return;
          const on = panel.getAttribute("data-add-stop-panel") === id;
          panel.hidden = !on;
          panel.classList.toggle("hidden", !on);
        });
      };
      tabs.forEach((btn) => {
        btn.addEventListener("click", () => {
          const id = btn.getAttribute("data-add-stop-tab");
          if (id) activate(id);
        });
      });
    });
  };
  initAddStopTabs();

  const initItineraryForm = (itineraryForm) => {
    if (!(itineraryForm instanceof HTMLElement)) return;
    if (itineraryForm.dataset.remiItineraryFormBound === "1") return;
    itineraryForm.dataset.remiItineraryFormBound = "1";
    const locationInput = itineraryForm.querySelector("[data-location-input]");
    const latitudeInput = itineraryForm.querySelector("[data-latitude-input]");
    const longitudeInput = itineraryForm.querySelector("[data-longitude-input]");
    const locationStatus = itineraryForm.querySelector("[data-location-status]");
    const suggestionBox = itineraryForm.querySelector("[data-location-suggestions]");
    const startAtIso = itineraryForm.querySelector("input.remi-datetime-iso[name='start_at']");
    const endAtIso = itineraryForm.querySelector("input.remi-datetime-iso[name='end_at']");
    const itineraryDateIso =
      itineraryForm.querySelector("input.remi-date-iso[name='itinerary_date']") ||
      itineraryForm.querySelector("input[type='hidden'][name='itinerary_date']") ||
      itineraryForm.querySelector("input[name='itinerary_date']");
    const itineraryDateInput =
      (itineraryDateIso instanceof HTMLInputElement ? itineraryDateIso : null) ||
      (startAtIso instanceof HTMLInputElement ? startAtIso : null);
    const itineraryDateVisibleEl = () => {
      const isoLegacy = itineraryForm.querySelector("input.remi-date-iso[name='itinerary_date']");
      const wrapLegacy = isoLegacy?.closest?.("[data-remi-date]");
      if (wrapLegacy) {
        const v = wrapLegacy.querySelector(".remi-date-visible");
        if (v instanceof HTMLElement) return v;
      }
      const s = itineraryForm.querySelector("input.remi-datetime-iso[name='start_at']");
      const wrapDt = s?.closest?.("[data-remi-datetime]");
      if (wrapDt) {
        const combined = wrapDt.querySelector(".remi-datetime-combined-part");
        if (combined instanceof HTMLElement) return combined;
        const part = wrapDt.querySelector(".remi-datetime-date-part");
        if (part instanceof HTMLElement) return part;
      }
      return null;
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

    const titleInput = itineraryForm.querySelector("input[name='title']");
    const openingHoursInput = itineraryForm.querySelector("input[name='opening_hours']");
    const googlePlaceIdInput = itineraryForm.querySelector("[data-google-place-id-input]");
    const venueHoursJsonInput = itineraryForm.querySelector("[data-venue-hours-json-input]");
    const chipWrap = itineraryForm.querySelector("[data-place-hours-wrap]");
    const chip = itineraryForm.querySelector("[data-place-hours-chip]");
    const timeWarningEl = itineraryForm.querySelector("[data-venue-hours-time-warning]");
    const locationInputWrap = itineraryForm.querySelector("[data-itinerary-location-wrap]");
    const locationClearBtn = itineraryForm.querySelector("[data-itinerary-location-clear]");
    let latestPlaceDetails = null;
    let latestGooglePlaceId = "";

    const syncLocationClearBtn = () => {
      if (!(locationClearBtn instanceof HTMLButtonElement) || !locationInputWrap) return;
      const hasText = locationInput instanceof HTMLInputElement && (locationInput.value || "").trim() !== "";
      const hasPlace =
        googlePlaceIdInput instanceof HTMLInputElement && (googlePlaceIdInput.value || "").trim() !== "";
      const chipTxt = chip ? (chip.textContent || "").trim() : "";
      const chipVisible = Boolean(chipTxt && chipWrap && !chipWrap.hasAttribute("hidden"));
      const show = hasText || hasPlace || chipVisible;
      locationClearBtn.hidden = !show;
      locationInputWrap.classList.toggle("itinerary-location-input-wrap--clearable", show);
    };

    const clearTimeWarnings = () => {
      itineraryForm.querySelectorAll(".remi-datetime-field--venue-hours-warn").forEach((el) => {
        el.classList.remove("remi-datetime-field--venue-hours-warn");
      });
      if (timeWarningEl) {
        timeWarningEl.textContent = "";
        timeWarningEl.classList.add("hidden");
      }
    };

    const applyTimeWarnings = (on) => {
      const sw = startAtIso?.closest?.("[data-remi-datetime]");
      const ew = endAtIso?.closest?.("[data-remi-datetime]");
      if (sw instanceof HTMLElement) sw.classList.toggle("remi-datetime-field--venue-hours-warn", Boolean(on));
      if (ew instanceof HTMLElement) ew.classList.toggle("remi-datetime-field--venue-hours-warn", Boolean(on));
      if (timeWarningEl) {
        if (on) {
          timeWarningEl.textContent =
            "Warning: Scheduled time is outside venue operating hours.";
          timeWarningEl.classList.remove("hidden");
        } else {
          timeWarningEl.textContent = "";
          timeWarningEl.classList.add("hidden");
        }
      }
    };

    const venueYmd = () => {
      const raw = startAtIso instanceof HTMLInputElement ? startAtIso.value : "";
      const s = String(raw || "").trim();
      if (s.length >= 10) return s.slice(0, 10);
      return todayLocal;
    };

    const syncVenueHoursHiddenField = () => {
      if (!(venueHoursJsonInput instanceof HTMLInputElement)) return;
      if (!latestGooglePlaceId || !latestPlaceDetails) {
        venueHoursJsonInput.value = "";
        return;
      }
      const ymd = venueYmd();
      const o = remiSummarizeHoursFromPlaceDetail(latestPlaceDetails, ymd);
      const payload = {
        date: ymd,
        google_place_id: latestGooglePlaceId,
        status: o.status,
        summary: o.chipLine
      };
      if (o.openMins != null && o.closeMins != null) {
        payload.open_mins = o.openMins;
        payload.close_mins = o.closeMins;
      }
      venueHoursJsonInput.value = JSON.stringify(payload);
    };

    const validateTimesAgainstVenue = () => {
      if (!latestGooglePlaceId || !latestPlaceDetails) {
        clearTimeWarnings();
        return;
      }
      const ymd = venueYmd();
      const startV = startAtIso instanceof HTMLInputElement ? startAtIso.value : "";
      const endV = endAtIso instanceof HTMLInputElement ? endAtIso.value : "";
      const bad = remiScheduleOutsideVenue(latestPlaceDetails, ymd, startV, endV);
      applyTimeWarnings(bad);
    };

    const refreshPlaceChip = () => {
      if (!chip) return;
      if (!latestGooglePlaceId || !latestPlaceDetails) return;
      const o = remiSummarizeHoursFromPlaceDetail(latestPlaceDetails, venueYmd());
      chip.textContent = o.chipLine;
      chipWrap?.removeAttribute("hidden");
      syncVenueHoursHiddenField();
      if (openingHoursInput instanceof HTMLInputElement) {
        openingHoursInput.value = o.chipLine;
      }
      validateTimesAgainstVenue();
    };

    const setChipLoading = (on) => {
      chip?.classList.toggle("place-hours-chip--loading", Boolean(on));
      if (on) chipWrap?.removeAttribute("hidden");
    };

    const resetVenueAssistantState = () => {
      latestPlaceDetails = null;
      latestGooglePlaceId = "";
      if (googlePlaceIdInput instanceof HTMLInputElement) googlePlaceIdInput.value = "";
      if (venueHoursJsonInput instanceof HTMLInputElement) venueHoursJsonInput.value = "";
      if (titleInput instanceof HTMLInputElement) titleInput.value = "";
      if (openingHoursInput instanceof HTMLInputElement) openingHoursInput.value = "";
      if (chip) chip.textContent = "";
      chipWrap?.setAttribute("hidden", "");
      chip?.classList.remove("place-hours-chip--loading");
      clearTimeWarnings();
      syncLocationClearBtn();
    };

    itineraryForm.addEventListener(
      "remi-itinerary-form-reset-save-another",
      () => {
        resetVenueAssistantState();
        cachedLocation = "";
        cachedCoords = null;
        fillCoordinates(null);
        hideSuggestions();
        setLocationStatus("Type a location and coordinates will be fetched automatically.", "");
        const tabsHost = itineraryForm.querySelector("[data-add-stop-tabs]");
        if (tabsHost) {
          const detailsTab = tabsHost.querySelector("[data-add-stop-tab='details']");
          if (detailsTab instanceof HTMLElement) detailsTab.click();
        }
      },
      false
    );

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
      if (!tripStart || !tripEnd) return true;
      const pickDate = (raw) => {
        const s = String(raw || "").trim();
        if (!s) return "";
        return s.length >= 10 ? s.slice(0, 10) : "";
      };
      if (startAtIso instanceof HTMLInputElement && endAtIso instanceof HTMLInputElement) {
        const d0 = pickDate(startAtIso.value);
        const d1 = pickDate(endAtIso.value);
        if (!d0) {
          setDateStatus("Please select start date and time before saving.", "error");
          return false;
        }
        if (d0 < tripStart || d0 > tripEnd) {
          setDateStatus(`Dates must be between ${tripStartLabel} and ${tripEndLabel}.`, "error");
          return false;
        }
        if (d1) {
          if (d1 < tripStart || d1 > tripEnd) {
            setDateStatus(`Dates must be between ${tripStartLabel} and ${tripEndLabel}.`, "error");
            return false;
          }
          if (d0 !== d1) {
            setDateStatus("Start and end must be on the same calendar day.", "error");
            return false;
          }
          const fs = String(startAtIso.value || "").trim();
          const fe = String(endAtIso.value || "").trim();
          if (fe && fs && fe < fs) {
            setDateStatus("End time must be on or after start time.", "error");
            return false;
          }
        }
        setDateStatus("", "");
        return true;
      }
      if (!(itineraryDateIso instanceof HTMLInputElement)) return true;
      const selected = pickDate(itineraryDateIso.value);
      if (!selected) {
        setDateStatus("Please select a date before saving.", "error");
        return false;
      }
      if (selected < tripStart || selected > tripEnd) {
        setDateStatus(`Date must be between ${tripStartLabel} and ${tripEndLabel}.`, "error");
        return false;
      }
      setDateStatus("", "");
      return true;
    };

    const hideSuggestions = () => {
      if (!suggestionBox) return;
      suggestionBox.classList.add("hidden");
      suggestionBox.innerHTML = "";
    };

    const clearItineraryLocationChoice = () => {
      if (locationInput instanceof HTMLInputElement) locationInput.value = "";
      hideSuggestions();
      cachedLocation = "";
      cachedCoords = null;
      fillCoordinates(null);
      resetVenueAssistantState();
      setLocationStatus("Type a location and coordinates will be fetched automatically.", "");
    };

    const selectSuggestion = async (suggestion) => {
      if (!locationInput) return;
      hideSuggestions();
      const pid = String(suggestion.placeId || "").trim();
      if (pid && locationLookupEnabled) {
        setChipLoading(true);
        setLocationStatus("Loading place details…");
        const detail = await fetchPlaceDetailsForItinerary(pid);
        setChipLoading(false);
        if (!detail || (detail.lat === 0 && detail.lng === 0 && !detail.formattedAddress)) {
          setLocationStatus("Could not load details for that place. Try another result.", "error");
          return;
        }
        latestPlaceDetails = detail;
        latestGooglePlaceId = pid;
        if (googlePlaceIdInput instanceof HTMLInputElement) googlePlaceIdInput.value = pid;
        const addr =
          String(detail.formattedAddress || "").trim() ||
          String(suggestion.formattedAddress || "").trim() ||
          String(suggestion.displayName || "").trim();
        locationInput.value = addr;
        const t =
          String(detail.placeName || "").trim() ||
          String(suggestion.placeName || "").trim() ||
          String(suggestion.shortName || "").trim();
        if (titleInput instanceof HTMLInputElement && t) titleInput.value = t;
        fillCoordinates({ lat: detail.lat, lng: detail.lng });
        cachedLocation = normalize(addr);
        cachedCoords = { lat: detail.lat, lng: detail.lng, displayName: addr };
        setLocationStatus("", "");
        refreshPlaceChip();
        syncLocationClearBtn();
        return;
      }
      resetVenueAssistantState();
      const cleanName = suggestion.displayName || locationInput.value.trim();
      locationInput.value = cleanName;
      const t0 =
        String(suggestion.placeName || "").trim() ||
        String(suggestion.shortName || "").trim() ||
        (cleanName ? cleanName.split(",")[0].trim() : "");
      if (titleInput instanceof HTMLInputElement && t0) titleInput.value = t0;
      cachedLocation = normalize(cleanName);
      cachedCoords = { lat: suggestion.lat, lng: suggestion.lng, displayName: cleanName };
      fillCoordinates(cachedCoords);
      setLocationStatus("", "");
      if (chip) {
        chip.textContent = "Opening hours not available.";
      }
      chipWrap?.removeAttribute("hidden");
      syncVenueHoursHiddenField();
      validateTimesAgainstVenue();
      syncLocationClearBtn();
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
        const line1 = document.createElement("span");
        line1.className = "location-suggestion-btn__primary";
        line1.textContent = suggestion.placeName || suggestion.shortName || suggestion.displayName;
        const line2 = document.createElement("span");
        line2.className = "location-suggestion-btn__secondary muted small";
        line2.textContent = suggestion.formattedAddress || suggestion.displayName || "";
        btn.appendChild(line1);
        btn.appendChild(line2);
        remiPreventLocationSuggestBlur(btn);
        btn.addEventListener("click", () => {
          void selectSuggestion(suggestion);
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
          resetVenueAssistantState();
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
        resetVenueAssistantState();
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
      setLocationStatus("", "");
      hideSuggestions();
      syncLocationClearBtn();
      return true;
    };

    if (locationInput) {
      locationInput.addEventListener("input", () => {
        cachedLocation = "";
        cachedCoords = null;
        fillCoordinates(null);
        if (!(locationInput.value || "").trim()) {
          resetVenueAssistantState();
        }
        queueSuggestions(locationInput.value);
        syncLocationClearBtn();
      });
      locationInput.addEventListener("blur", () => {
        window.setTimeout(() => {
          hideSuggestions();
        }, REMI_LOCATION_SUGGEST_BLUR_MS);
        void resolveLocation(locationInput.value);
      });
    }
    if (locationClearBtn instanceof HTMLButtonElement) {
      remiPreventLocationSuggestBlur(locationClearBtn);
      locationClearBtn.addEventListener("click", (e) => {
        e.preventDefault();
        clearItineraryLocationChoice();
      });
    }
    const onStopScheduleChange = () => {
      validateDateInTripRange();
      if (latestGooglePlaceId) {
        refreshPlaceChip();
      }
    };
    if (itineraryDateInput) {
      itineraryDateInput.addEventListener("change", onStopScheduleChange);
      validateDateInTripRange();
    }
    if (startAtIso instanceof HTMLInputElement && startAtIso !== itineraryDateInput) {
      startAtIso.addEventListener("change", onStopScheduleChange);
      startAtIso.addEventListener("input", onStopScheduleChange);
    }
    if (endAtIso instanceof HTMLInputElement && endAtIso !== itineraryDateInput) {
      endAtIso.addEventListener("change", onStopScheduleChange);
      endAtIso.addEventListener("input", onStopScheduleChange);
    }

    void (async () => {
      const pid =
        googlePlaceIdInput instanceof HTMLInputElement
          ? String(googlePlaceIdInput.value || "").trim()
          : "";
      if (pid) {
        setChipLoading(true);
        const d = await fetchPlaceDetailsForItinerary(pid);
        setChipLoading(false);
        if (d) {
          latestPlaceDetails = d;
          latestGooglePlaceId = pid;
          refreshPlaceChip();
        }
      } else if (venueHoursJsonInput instanceof HTMLInputElement && (venueHoursJsonInput.value || "").trim()) {
        try {
          const j = JSON.parse(venueHoursJsonInput.value);
          if (chip) {
            let line = "";
            if (j.user_opening_hours) line = String(j.user_opening_hours);
            else if (j.open_mins != null && j.close_mins != null) {
              line = `${remiFormatMinutesClock(j.open_mins)} – ${remiFormatMinutesClock(j.close_mins)}`;
            } else if (j.summary) line = String(j.summary);
            if (line) {
              chip.textContent = line;
              chipWrap?.removeAttribute("hidden");
            }
          }
        } catch (e) {
          /* ignore */
        }
      }
      syncLocationClearBtn();
    })();

    // Submit must stay synchronous for preventDefault + default action; async listeners and
    // requestSubmit() without the original submitter caused first-click no-ops in some cases.
    itineraryForm.addEventListener("submit", (event) => {
      if (allowResolvedSubmit) {
        allowResolvedSubmit = false;
        return;
      }
      if (geocodeSubmitInProgress) {
        event.preventDefault();
        return;
      }
      if (!validateDateInTripRange()) {
        event.preventDefault();
        const vis = itineraryDateVisibleEl();
        if (vis instanceof HTMLElement) vis.focus();
        else itineraryDateInput?.focus?.();
        return;
      }
      syncVenueHoursHiddenField();
      if (!locationInput) return;
      const query = (locationInput.value || "").trim();
      if (!query) return;

      if (!locationLookupEnabled) {
        return;
      }

      const latVal = (latitudeInput?.value || "").trim();
      const lngVal = (longitudeInput?.value || "").trim();
      if (latVal && lngVal) {
        const lat = parseFloat(latVal);
        const lng = parseFloat(lngVal);
        if (
          Number.isFinite(lat) &&
          Number.isFinite(lng) &&
          Math.abs(lat) <= 90 &&
          Math.abs(lng) <= 180
        ) {
          return;
        }
      }

      event.preventDefault();
      const sub =
        event.submitter instanceof HTMLButtonElement || event.submitter instanceof HTMLInputElement
          ? event.submitter
          : undefined;
      geocodeSubmitInProgress = true;
      void resolveLocation(query)
        .then((ok) => {
          geocodeSubmitInProgress = false;
          if (!ok) {
            locationInput.focus();
            return;
          }
          allowResolvedSubmit = true;
          try {
            if (typeof itineraryForm.requestSubmit === "function") {
              if (sub && itineraryForm.contains(sub)) {
                itineraryForm.requestSubmit(sub);
              } else {
                itineraryForm.requestSubmit();
              }
            } else {
              itineraryForm.submit();
            }
          } catch (e) {
            try {
              itineraryForm.requestSubmit();
            } catch (e2) {
              itineraryForm.submit();
            }
          }
        })
        .catch(() => {
          geocodeSubmitInProgress = false;
          locationInput.focus();
        });
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
          const line1 = document.createElement("span");
          line1.className = "location-suggestion-btn__primary";
          line1.textContent = suggestion.placeName || suggestion.shortName || suggestion.displayName;
          const line2 = document.createElement("span");
          line2.className = "location-suggestion-btn__secondary";
          line2.textContent = suggestion.formattedAddress || suggestion.displayName || "";
          btn.appendChild(line1);
          btn.appendChild(line2);
          remiPreventLocationSuggestBlur(btn);
          btn.addEventListener("click", () => {
            void (async () => {
              const pid = String(suggestion.placeId || "").trim();
              if (pid) {
                const detail = await fetchPlaceDetailsForItinerary(pid);
                input.value =
                  (detail && String(detail.formattedAddress || "").trim()) ||
                  String(suggestion.formattedAddress || "").trim() ||
                  suggestion.displayName;
              } else {
                input.value = suggestion.displayName;
              }
              hide();
            })();
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

  const rememberFlightAirportCache = (s) => {
    const csrfMeta = document.querySelector("meta[name='csrf-token']");
    const csrf = csrfMeta && csrfMeta.getAttribute("content") ? csrfMeta.getAttribute("content").trim() : "";
    const iata = String(s.iataCode || s.IATACode || "").trim().toUpperCase();
    if (!csrf || iata.length !== 3) {
      return;
    }
    void fetch("/api/flight-airports/remember", {
      method: "POST",
      credentials: "same-origin",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        "X-CSRF-Token": csrf
      },
      body: JSON.stringify({
        iataCode: iata,
        icaoCode: String(s.icaoCode || "").trim(),
        name: String(s.displayName || "").trim(),
        city: String(s.cityName || s.city || "").trim(),
        country: String(s.country || "").trim(),
        timezone: String(s.timezone || s.timeZone || "").trim()
      })
    }).catch(() => {});
  };

  const wireFlightAirportAutocompleteInput = (input) => {
    if (!(input instanceof HTMLInputElement)) {
      return;
    }
    if (input.dataset.remiFlightAirportBound === "1") {
      return;
    }
    input.dataset.remiFlightAirportBound = "1";
    let timer = null;
    let inflight = 0;
    let suggestionsHost = null;
    const ensureHost = () => {
      if (suggestionsHost) {
        return suggestionsHost;
      }
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
    const iataFieldName = input.name === "depart_airport" ? "depart_airport_iata" : "arrive_airport_iata";
    const setIATA = (code) => {
      const f = input.closest("form");
      if (!f) {
        return;
      }
      const h = f.querySelector(`input[name='${iataFieldName}']`);
      if (h instanceof HTMLInputElement) {
        h.value = code || "";
      }
    };
    /** Visible field value: full airport name + IATA when known (e.g. "Kempegowda International Airport - BLR"). */
    const flightAirportFieldValueFromPick = (s) => {
      const iata = String(s.iataCode || "").trim().toUpperCase();
      if (iata.length === 3) {
        const name = String(s.displayName || s.secondaryLine || s.primaryLine || "").trim();
        if (name) {
          const tail = ` - ${iata}`;
          if (name.toUpperCase().endsWith(tail.toUpperCase())) {
            return name;
          }
          return `${name}${tail}`;
        }
        return iata;
      }
      return String(s.displayName || s.primaryLine || s.formattedAddress || "").trim();
    };
    input.addEventListener("input", () => {
      if (timer) {
        clearTimeout(timer);
      }
      const query = input.value.trim();
      if (query.length < 2) {
        hide();
        return;
      }
      timer = window.setTimeout(() => {
        const req = ++inflight;
        const host = ensureHost();
        void (async () => {
          const res = await fetch(
            `${remiFlightAirportSuggestURL()}?q=${encodeURIComponent(query)}${remiLocationLangQuery()}`,
            {
              credentials: "same-origin",
              cache: "no-store",
              headers: { Accept: "application/json", "Cache-Control": "no-cache" }
            }
          );
          if (req !== inflight) {
            return;
          }
          if (!res.ok) {
            hide();
            return;
          }
          const manualFlightSearch =
            String(res.headers.get("X-Flight-Search-Manual") || "").trim() === "1";
          const data = await res.json();
          if (
            manualFlightSearch &&
            (!Array.isArray(data) || data.length === 0) &&
            typeof window.remiShowToast === "function"
          ) {
            window.remiShowToast(
              "Airport lookup is unavailable (invalid or rate-limited AirLabs key). Enable map location search in Site Settings, or enter airports manually."
            );
          }
          if (req !== inflight) {
            return;
          }
          host.innerHTML = "";
          if (!Array.isArray(data) || data.length === 0) {
            hide();
            return;
          }
          data.forEach((raw) => {
            const s = {
              lat: parseFloat(raw.lat ?? raw.Lat ?? "0") || 0,
              lng: parseFloat(raw.lng ?? raw.Lon ?? "0") || 0,
              displayName: String(raw.displayName || raw.display_name || "").trim(),
              iataCode: String(raw.iataCode || raw.iata_code || "").trim().toUpperCase(),
              placeId: String(raw.placeId || raw.place_id || "").trim(),
              primaryLine: String(raw.primaryLine || raw.primary_line || "").trim(),
              secondaryLine: String(raw.secondaryLine || raw.secondary_line || "").trim(),
              placeName: String(raw.placeName || raw.place_name || "").trim(),
              formattedAddress: String(raw.formattedAddress || raw.formatted_address || "").trim(),
              source: String(raw.source || "").trim(),
              cityName: String(raw.cityName || raw.city_name || "").trim(),
              country: String(raw.country || "").trim(),
              timezone: String(raw.timezone || "").trim(),
              icaoCode: String(raw.icaoCode || raw.icao_code || "").trim()
            };
            if (!s.primaryLine) {
              s.primaryLine = s.placeName || s.displayName;
            }
            if (!s.secondaryLine) {
              s.secondaryLine = s.formattedAddress || s.displayName;
            }
            if (!s.displayName) {
              s.displayName = s.formattedAddress || s.primaryLine;
            }
            const btn = document.createElement("button");
            btn.type = "button";
            btn.className = "location-suggestion-btn";
            const line1 = document.createElement("span");
            line1.className = "location-suggestion-btn__primary";
            line1.textContent = s.primaryLine;
            const line2 = document.createElement("span");
            line2.className = "location-suggestion-btn__secondary muted small";
            line2.textContent = s.secondaryLine;
            btn.appendChild(line1);
            btn.appendChild(line2);
            remiPreventLocationSuggestBlur(btn);
            btn.addEventListener("click", () => {
              input.value = flightAirportFieldValueFromPick(s);
              if (s.iataCode && s.iataCode.length === 3) {
                setIATA(s.iataCode);
                if (s.source === "airlabs" || s.source === "cache") {
                  rememberFlightAirportCache(s);
                }
              } else {
                setIATA("");
              }
              hide();
            });
            host.appendChild(btn);
          });
          host.classList.remove("hidden");
        })();
      }, 300);
    });
    input.addEventListener("blur", () => {
      window.setTimeout(hide, REMI_LOCATION_SUGGEST_BLUR_MS);
    });
  };

  const remiInitFlightAirportAutocompleteIn = (root) => {
    const scope = remiIsDomQueryRoot(root) ? root : document;
    scope.querySelectorAll("input[data-flight-airport-autocomplete]").forEach((el) => {
      wireFlightAirportAutocompleteInput(el);
    });
  };
  window.remiInitFlightAirportAutocompleteIn = remiInitFlightAirportAutocompleteIn;
  remiInitFlightAirportAutocompleteIn(document);

  /** Visible airline field after pick: "American Airlines - AA" (matches server primaryLine). */
  const flightAirlineFieldValueFromPick = (s) => {
    const primary = String(s.primaryLine || "").trim();
    if (primary) {
      return primary;
    }
    const iata = String(s.iataCode || "").trim().toUpperCase();
    const name = String(s.displayName || "").trim();
    if (name && iata.length >= 2) {
      const tail = ` - ${iata}`;
      if (name.toUpperCase().endsWith(tail.toUpperCase())) {
        return name;
      }
      return `${name}${tail}`;
    }
    return String(s.displayName || "").trim();
  };

  const wireFlightAirlineAutocompleteInput = (input) => {
    if (!(input instanceof HTMLInputElement)) {
      return;
    }
    if (input.dataset.remiFlightAirlineBound === "1") {
      return;
    }
    input.dataset.remiFlightAirlineBound = "1";
    let timer = null;
    let inflight = 0;
    let suggestionsHost = null;
    const ensureHost = () => {
      if (suggestionsHost) {
        return suggestionsHost;
      }
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
      if (timer) {
        clearTimeout(timer);
      }
      const query = input.value.trim();
      if (query.length < 2) {
        hide();
        return;
      }
      timer = window.setTimeout(() => {
        const req = ++inflight;
        const host = ensureHost();
        void (async () => {
          const res = await fetch(
            `${remiFlightAirlineSuggestURL()}?q=${encodeURIComponent(query)}`,
            {
              credentials: "same-origin",
              cache: "no-store",
              headers: { Accept: "application/json", "Cache-Control": "no-cache" }
            }
          );
          if (req !== inflight) {
            return;
          }
          if (!res.ok) {
            hide();
            return;
          }
          const manualAirlineSearch =
            String(res.headers.get("X-Flight-Airline-Search-Manual") || "").trim() === "1";
          const data = await res.json();
          if (
            manualAirlineSearch &&
            (!Array.isArray(data) || data.length === 0) &&
            typeof window.remiShowToast === "function"
          ) {
            window.remiShowToast(
              "Airline lookup is unavailable (invalid or rate-limited AirLabs key). Enter the airline manually, e.g. American Airlines - AA."
            );
          }
          if (req !== inflight) {
            return;
          }
          host.innerHTML = "";
          if (!Array.isArray(data) || data.length === 0) {
            hide();
            return;
          }
          data.forEach((raw) => {
            const s = {
              displayName: String(raw.displayName || raw.display_name || "").trim(),
              iataCode: String(raw.iataCode || raw.iata_code || "").trim().toUpperCase(),
              primaryLine: String(raw.primaryLine || raw.primary_line || "").trim(),
              secondaryLine: String(raw.secondaryLine || raw.secondary_line || "").trim(),
              formattedAddress: String(raw.formattedAddress || raw.formatted_address || "").trim()
            };
            if (!s.primaryLine) {
              s.primaryLine = flightAirlineFieldValueFromPick(s);
            }
            if (!s.secondaryLine) {
              s.secondaryLine = s.formattedAddress || "";
            }
            if (!s.displayName && s.primaryLine) {
              const parts = s.primaryLine.split(/\s+-\s+/);
              s.displayName = parts[0] ? parts[0].trim() : s.primaryLine;
            }
            const btn = document.createElement("button");
            btn.type = "button";
            btn.className = "location-suggestion-btn";
            const line1 = document.createElement("span");
            line1.className = "location-suggestion-btn__primary";
            line1.textContent = s.primaryLine;
            const line2 = document.createElement("span");
            line2.className = "location-suggestion-btn__secondary muted small";
            line2.textContent = s.secondaryLine;
            btn.appendChild(line1);
            btn.appendChild(line2);
            remiPreventLocationSuggestBlur(btn);
            btn.addEventListener("click", () => {
              input.value = flightAirlineFieldValueFromPick(s);
              hide();
            });
            host.appendChild(btn);
          });
          host.classList.remove("hidden");
        })();
      }, 300);
    });
    input.addEventListener("blur", () => {
      window.setTimeout(hide, REMI_LOCATION_SUGGEST_BLUR_MS);
    });
  };

  const remiInitFlightAirlineAutocompleteIn = (root) => {
    const scope = remiIsDomQueryRoot(root) ? root : document;
    scope.querySelectorAll("input[data-flight-airline-autocomplete]").forEach((el) => {
      wireFlightAirlineAutocompleteInput(el);
    });
  };
  window.remiInitFlightAirlineAutocompleteIn = remiInitFlightAirlineAutocompleteIn;
  remiInitFlightAirlineAutocompleteIn(document);

  const remiInitStopWeatherIn = (root) => {
    const scope = remiIsDomQueryRoot(root) ? root : document;
    scope.querySelectorAll("[data-stop-weather]").forEach((slot) => {
      if (!(slot instanceof HTMLElement) || slot.dataset.remiStopWeatherWired === "1") return;
      slot.dataset.remiStopWeatherWired = "1";
      const base = slot.getAttribute("data-weather-url");
      const date = (slot.getAttribute("data-weather-date") || "").trim();
      if (!base || !date) {
        slot.remove();
        return;
      }
      const u = new URL(base, window.location.origin);
      u.searchParams.set("date", date);
      void (async () => {
        try {
          const res = await fetch(u.toString(), {
            credentials: "same-origin",
            cache: "no-store",
            headers: { Accept: "application/json" }
          });
          if (!res.ok) {
            slot.remove();
            return;
          }
          const d = await res.json();
          if (!d || !d.ok) {
            const reason = String(d?.reason || "").trim();
            if (reason === "fetch_failed" || reason === "out_of_range") {
              slot.remove();
              return;
            }
            slot.remove();
            return;
          }
          const hi = typeof d.highC === "number" ? Math.round(d.highC) : "—";
          const lo = typeof d.lowC === "number" ? Math.round(d.lowC) : "—";
          const icon = String(d.icon || "☁️").trim() || "☁️";
          slot.textContent = "";
          const row = document.createElement("div");
          row.className = "itinerary-weather-preview";
          const line = document.createElement("div");
          line.className = "itinerary-weather-line";
          const ic = document.createElement("span");
          ic.className = "itinerary-weather-icon";
          ic.setAttribute("aria-hidden", "true");
          ic.textContent = icon;
          const temps = document.createElement("span");
          temps.className = "itinerary-weather-temps";
          temps.textContent = `H${hi}° L${lo}°`;
          line.appendChild(ic);
          line.appendChild(temps);
          row.appendChild(line);
          if (d.hasAlert) {
            const alert = document.createElement("div");
            alert.className = "itinerary-weather-alert";
            alert.textContent = "⚠️ Weather Alert for this day.";
            row.appendChild(alert);
          }
          slot.appendChild(row);
          slot.hidden = false;
        } catch {
          slot.remove();
        }
      })();
    });
  };
  window.remiInitStopWeatherIn = remiInitStopWeatherIn;
  remiInitStopWeatherIn(document);

  const vehicleDropoffSyncByFieldset = new WeakMap();
  const initVehicleDropoffFieldsetsIn = (root) => {
    const scope = root instanceof HTMLElement || root instanceof Document ? root : document;
    scope.querySelectorAll(".vehicle-dropoff-fieldset").forEach((fs) => {
      if (fs.dataset.remiVehicleDropoffBound === "1") return;
      fs.dataset.remiVehicleDropoffBound = "1";
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
  };
  initVehicleDropoffFieldsetsIn(document);
  window.remiInitVehicleDropoffFieldsetsIn = initVehicleDropoffFieldsetsIn;

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
    if (form.matches("[data-remi-unified-expense-add-form]")) {
      const ft = form.querySelector(".remi-unified-expense-from-tab");
      if (ft instanceof HTMLInputElement && !ft.disabled && String(ft.value || "").trim() === "1") {
        return "Group expense saved.";
      }
      return "Expense saved.";
    }
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
    const icon = row.querySelector(".remi-checklist-check-btn .material-symbols-outlined, .check-btn .material-symbols-outlined");
    const text = row.querySelector(".remi-checklist-item-text");
    const btn = row.querySelector(".remi-checklist-check-btn");
    if (btn) btn.classList.toggle("remi-checklist-check-btn--done", done);
    if (btn) btn.classList.toggle("trip-keep-check-btn--done", done);
    if (icon) icon.textContent = done ? "check" : "";
    if (text) text.classList.toggle("done", done);
    if (text) text.classList.toggle("trip-keep-checklist-text--done", done);
    const hiddenDone = form.querySelector("input[name='done']");
    if (hiddenDone) hiddenDone.value = done ? "false" : "true";
    const toggleBtn = row.querySelector(".trip-keep-check-btn");
    if (toggleBtn) toggleBtn.setAttribute("aria-pressed", done ? "true" : "false");
  };

  const updateChecklistEditUI = (form) => {
    const row = form.closest(".reminder-checklist-item");
    if (!row) return;
    const textInput = form.querySelector("input[name='text']");
    const nextText = (textInput?.value || "").trim();
    if (!nextText) return;
    const textNode = row.querySelector(".remi-checklist-item-text");
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
    if (!(form instanceof HTMLFormElement)) return;
    const inFlyoutEditClone = Boolean(form.closest(".remi-trip-flyout-edit-body"));
    if (!form.classList.contains("item-edit") && !(form.classList.contains("remi-trip-widget-form") && inFlyoutEditClone)) {
      return;
    }
    // Cloned flyout DOM may still carry a dead .edit-attachment-ui from the source form (listeners are not cloned).
    let sib = fileInput.nextElementSibling;
    while (sib instanceof HTMLElement && sib.classList.contains("edit-attachment-ui")) {
      const rm = sib;
      sib = sib.nextElementSibling;
      rm.remove();
    }

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
        fileName = meta.kind === "image" ? "No image uploaded" : "No file attached";
        sub = `Max file size: ${maxSizeMsg}`;
      }

      icon.textContent = remiAttachmentIcon(meta.kind, ext);
      nameEl.textContent = fileName;
      subEl.textContent = sub;
      replaceBtn.querySelector("span:last-child").textContent =
        state.pendingFile || existingUrl ? "Replace File" : "Upload File";
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
    scope.querySelectorAll("input[type='file'][name]").forEach((input) => {
      if (!(input instanceof HTMLInputElement)) return;
      if (!FILE_FIELD_META_BY_NAME[input.name]) return;
      const form = input.closest("form");
      if (!(form instanceof HTMLFormElement)) return;
      if (form.classList.contains("item-edit")) {
        initAttachmentEditField(input);
        return;
      }
      if (form.classList.contains("remi-trip-widget-form") && form.closest(".remi-trip-flyout-edit-body")) {
        initAttachmentEditField(input);
      }
    });
  };
  window.remiInitAttachmentEditFieldsIn = initAttachmentEditFieldsIn;
  initAttachmentEditFieldsIn(document);

  /** After replacing innerHTML inside a card view, reattach listeners for Edit / Delete / AJAX (e.g. spend row). */
  const rewireInjectedActionUI = (root) => {
    if (!remiIsDomQueryRoot(root)) return;
    window.remiEnsureCsrfOnPostForms?.(root);
    // Do not clear remiConfirmWired / remiAjaxWired on existing nodes: stacking duplicate
    // capture-phase confirm handlers makes the second handler block submit after OK (confirmPass
    // is consumed by the first). Only wire forms that are new (e.g. from AJAX partial HTML).
    root.querySelectorAll("form[data-app-confirm]").forEach((f) => {
      if (f.dataset.remiConfirmWired === "1") return;
      window.remiWireAppConfirmOnForm?.(f);
    });
    root.querySelectorAll("form[data-ajax-submit]").forEach((f) => {
      if (f.dataset.remiAjaxWired === "1") return;
      window.remiBindAjaxSubmitForm?.(f);
    });
    root.querySelectorAll("form[data-remi-unified-expense-add-form]").forEach((f) => {
      window.remiWireUnifiedExpenseForm?.(f);
    });
    window.remiWireInlineEditOpenButtonsIn?.(root);
    window.remiWireDateFieldsIn?.(root);
    window.remiWireDateTimeFieldsIn?.(root);
    window.remiWireTimeFieldsIn?.(root);
    window.remiInitAttachmentEditFieldsIn?.(root);
    window.remiApplyExpenseActionsDropdownOpen?.();
  };

  /**
   * Applies JSON `tripDocuments` payload from upload/update/delete handlers on the trip documents page.
   * @returns {boolean} true if the DOM was updated.
   */
  window.remiApplyTripDocumentsAjaxResponse = (root, td /* , form */) => {
    if (!root || !td || typeof td !== "object") return false;
    const ul = root.querySelector(".trip-documents-items");
    if (!ul) return false;
    let changed = false;
    if (td.removeId) {
      const id = String(td.removeId);
      const li = ul.querySelector(`[data-trip-doc-id="${CSS.escape(id)}"]`);
      if (li) {
        li.remove();
        changed = true;
      }
    }
    if (td.replaceHtml && td.documentId) {
      const id = String(td.documentId);
      const li = ul.querySelector(`[data-trip-doc-id="${CSS.escape(id)}"]`);
      if (li) {
        const tpl = document.createElement("template");
        tpl.innerHTML = String(td.replaceHtml).trim();
        const neu = tpl.content.firstElementChild;
        if (neu) {
          li.replaceWith(neu);
          changed = true;
        }
      }
    }
    if (td.appendHtml) {
      const html = String(td.appendHtml).trim();
      if (html) {
        const tpl = document.createElement("template");
        tpl.innerHTML = html;
        ul.insertBefore(tpl.content, ul.firstChild);
        changed = true;
      }
    }
    if (changed) {
      rewireInjectedActionUI(ul);
      window.remiRefreshTripDocumentsListFilter?.();
    }
    return changed;
  };

  const fetchNoStore = async (resource, init = {}) => {
    const headers = new Headers(init.headers || {});
    if (!headers.has("Cache-Control")) {
      headers.set("Cache-Control", "no-cache");
    }
    return fetch(resource, {
      cache: "no-store",
      credentials: "same-origin",
      ...init,
      headers
    });
  };

  /** Trip page HTML fetches must bypass the SW cache-first rule (sw.js) or stale DOM fragments replace live UI. */
  const tripPageHtmlFetchURL = () => {
    const url = new URL(window.location.href);
    url.searchParams.set("_remi_cb", String(Date.now()));
    return url.toString();
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
      const response = await fetchNoStore(tripPageHtmlFetchURL(), {
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
    const r = await fetchNoStore(tripPageHtmlFetchURL(), {
      headers: { "X-Requested-With": "XMLHttpRequest", Accept: "text/html" }
    });
    if (!r.ok) return null;
    return new DOMParser().parseFromString(await r.text(), "text/html");
  };

  const rewireLiveRefreshedDom = (root = document) => {
    const scope = root instanceof HTMLElement || root instanceof Document ? root : document;
    scope.querySelectorAll("form[data-app-confirm]").forEach((f) => {
      delete f.dataset.remiConfirmWired;
      window.remiWireAppConfirmOnForm?.(f);
    });
    scope.querySelectorAll("form[data-ajax-submit]").forEach((f) => {
      delete f.dataset.remiAjaxWired;
      window.remiBindAjaxSubmitForm?.(f);
    });
    window.remiWireInlineEditOpenButtonsIn?.(scope);
    window.remiInitAttachmentEditFieldsIn?.(scope);
    window.remiWireDateFieldsIn?.(scope);
    window.remiWireDateTimeFieldsIn?.(scope);
    window.remiWireTimeFieldsIn?.(scope);
    window.remiEnsureCsrfOnPostForms?.(scope);
    window.remiInitTripInviteMethodsIn?.(scope);
    window.remiInitVehicleDropoffFieldsetsIn?.(scope);
    window.remiInitFlightAirportAutocompleteIn?.(scope);
    window.remiInitFlightAirlineAutocompleteIn?.(scope);
    window.remiInitStopWeatherIn?.(scope);
    window.remiSetupMobileEntryCarouselsIn?.(scope);
    window.remiRefreshTripDocumentsListFilter?.();
    window.remiFitBudgetDonutCenterValues?.(scope);
    window.remiWireMobileSheetFormDirtyIn?.(scope);
    scope.querySelectorAll("[data-itinerary-form]").forEach((f) => {
      if (f instanceof HTMLFormElement && !f.dataset.remiItineraryFormBound) {
        initItineraryForm(f);
      }
    });
    /* New <details.trip-inline-actions-dropdown> from server HTML default to closed; without `open`,
     * the UA hides .trip-inline-actions-buttons (Edit/Delete) until reload — same as after inline edit. */
    window.remiApplyExpenseActionsDropdownOpen?.();
    window.remiRewireTripItineraryCalendarAfterLiveRefresh?.();
    window.remiInitTripPageSectionFoldsIn?.(scope);
    window.remiInitUnifiedBookingsIn?.(scope);
  };

  const replaceSelectorFromFreshDoc = (selector, freshDoc) => {
    const currentNodes = Array.from(document.querySelectorAll(selector));
    const freshNodes = Array.from(freshDoc.querySelectorAll(selector));
    if (!currentNodes.length || currentNodes.length !== freshNodes.length) return false;
    currentNodes.forEach((node, idx) => {
      const freshNode = freshNodes[idx];
      if (!freshNode) return;
      node.replaceWith(freshNode.cloneNode(true));
    });
    return true;
  };

  const refreshSharedTripChromeFromServer = async () => {
    const freshDoc = await fetchFreshDoc();
    if (!freshDoc) return false;
    let changed = false;
    [
      ".trip-mobile-header",
      ".trip-desktop-header-tools",
      ".trip-members-mobile-shell",
      ".sidebar-trip-members-wrap"
    ].forEach((selector) => {
      changed = replaceSelectorFromFreshDoc(selector, freshDoc) || changed;
    });
    if (changed) {
      rewireLiveRefreshedDom(document);
    }
    return changed;
  };

  const refreshTripDetailsSupportSectionsFromServer = async () => {
    const freshDoc = await fetchFreshDoc();
    if (!freshDoc) return false;
    let changed = false;
    [
      ".trip-itinerary-shell",
      ".trip-unified-bookings-section",
      ".trip-members-mobile-shell",
      ".sidebar-trip-members-wrap",
      ".trip-mobile-header",
      ".trip-desktop-header-tools",
      ".trip-sidebar-itinerary-composer",
      "#mobile-sheet-stop .trip-mobile-fab-itinerary-refresh",
      "#mobile-sheet-commute .trip-mobile-fab-itinerary-refresh"
    ].forEach((selector) => {
      changed = replaceSelectorFromFreshDoc(selector, freshDoc) || changed;
    });
    if (changed) {
      rewireLiveRefreshedDom(document);
    }
    return changed;
  };

  const hasOptimisticRowRemoveSkip = (form) =>
    Boolean(document.querySelector("main.tab-page")) &&
    Boolean(form.closest("[data-tab-refresh-region], .tab-settle-card, .tab-expenses-section"));

  const beginOptimisticAjaxMutation = (form, formData) => {
    const rollbacks = [];
    const action = (form.action || "").toLowerCase();
    if (form.classList.contains("check-row")) {
      const done = String(formData.get("done")) === "true";
      updateChecklistToggleUI(form, done);
      rollbacks.push(() => updateChecklistToggleUI(form, !done));
    }
    if (action.includes("/delete") && !hasOptimisticRowRemoveSkip(form)) {
        const row = form.closest(
          ".timeline-item, .expense-item, .reminder-checklist-item, .budget-tx-view, .budget-mobile-tx-item, .flight-card, .vehicle-rental-item, .vehicle-rental-card, .accommodation-card-wrap, .trip-unified-booking-row, .trip-document-row"
        );
      if (row?.parentNode) {
        const parent = row.parentNode;
        const nextSibling = row.nextSibling;
        const isTimelineRow = row.classList.contains("timeline-item");
        row.remove();
        if (isTimelineRow) {
          void renderItineraryConnectors(document);
          window.remiRefreshTripMapFromItineraryDOM?.();
        }
        rollbacks.push(() => {
          parent.insertBefore(row, nextSibling);
          if (isTimelineRow) {
            void renderItineraryConnectors(document);
            window.remiRefreshTripMapFromItineraryDOM?.();
          }
          rewireLiveRefreshedDom(parent instanceof HTMLElement ? parent : document);
        });
      }
    }
    return {
      rollback() {
        for (let i = rollbacks.length - 1; i >= 0; i -= 1) {
          try {
            rollbacks[i]();
          } catch (e) {
            /* ignore */
          }
        }
      }
    };
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
    let row = form.closest(".timeline-item");
    if (!row) {
      const v = document.getElementById(viewId);
      if (v) row = v.closest(".timeline-item");
    }
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
      const response = await fetchNoStore(tripPageHtmlFetchURL(), {
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

  const expandAndFocusNewItineraryAfterAddStop = async (opts) => {
    const focusRow = !(opts && opts.skipRowFocus);
    const dateIso = opts && opts.dateIso ? String(opts.dateIso).trim() : "";
    const titleHint = opts && opts.titleHint ? String(opts.titleHint).trim() : "";
    const dayNumberRaw = opts && opts.dayNumber != null ? opts.dayNumber : null;
    const dayNumber =
      typeof dayNumberRaw === "number" && Number.isFinite(dayNumberRaw) && dayNumberRaw >= 1
        ? dayNumberRaw
        : null;
    const shell =
      document.getElementById("trip-page-itinerary") || document.querySelector(".trip-itinerary-shell");
    if (!(shell instanceof HTMLElement)) return;
    const listBtn = shell.querySelector('[data-itinerary-view-toggle="list"]');
    if (listBtn instanceof HTMLElement && !listBtn.classList.contains("is-active")) listBtn.click();
    let dayGroup = null;
    if (dateIso) {
      dayGroup = shell.querySelector(`details.day-group[data-date="${dateIso}"]`);
    }
    if (!dayGroup && dayNumber != null) {
      const hit = shell.querySelector(`ul.day-items li.timeline-item[data-map-day="${dayNumber}"]`);
      if (hit instanceof HTMLElement) dayGroup = hit.closest("details.day-group");
    }
    if (!(dayGroup instanceof HTMLDetailsElement)) return;
    dayGroup.open = true;
    const ul = dayGroup.querySelector(":scope > ul.day-items");
    if (!(ul instanceof HTMLElement)) return;
    const rows = Array.from(ul.querySelectorAll(":scope > li.timeline-item[data-itinerary-item-id]"));
    if (!rows.length) return;
    let target = null;
    if (titleHint) {
      const tNorm = titleHint.toLowerCase();
      const byTitle = rows.filter(
        (li) =>
          (li.getAttribute("data-marker-kind") || "") === "stop" &&
          (li.getAttribute("data-title") || "").trim().toLowerCase() === tNorm
      );
      if (byTitle.length) target = byTitle[byTitle.length - 1];
    }
    if (!target) {
      const stops = rows.filter((li) => (li.getAttribute("data-marker-kind") || "") === "stop");
      target = stops.length ? stops[stops.length - 1] : rows[rows.length - 1];
    }
    await renderItineraryConnectors(document);
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        target.scrollIntoView({ block: "nearest", behavior: "smooth" });
        if (!focusRow) return;
        if (!target.hasAttribute("tabindex")) target.setAttribute("tabindex", "-1");
        try {
          target.focus({ preventScroll: true });
        } catch (e) {
          /* ignore */
        }
      });
    });
  };

  const handleAjaxFormSubmit = (event) => {
    const form = event.currentTarget;
    if (!(form instanceof HTMLFormElement)) return;
    if (event.defaultPrevented) return;
    event.preventDefault();
    const submitterEl = event.submitter;
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
      "Cache-Control": "no-cache",
      Accept: "application/json"
    };
    if (!isMultipartForm) {
      requestHeaders["Content-Type"] = "application/x-www-form-urlencoded;charset=UTF-8";
    }
    const optimisticMutation = beginOptimisticAjaxMutation(form, formData);
    void (async () => {
      try {
        const response = await fetchNoStore(form.action, {
          method,
          body: requestBody,
          headers: requestHeaders
        });
        if (!response.ok) {
          const txt = (await response.text()).trim();
          let errBody = null;
          try {
            errBody = txt ? JSON.parse(txt) : null;
          } catch {
            errBody = null;
          }
          const jsonMsg = errBody && errBody.error ? String(errBody.error) : "";
          const jsonCode = errBody && errBody.code ? String(errBody.code) : "";
          const error = new Error(jsonMsg || txt || "Save failed.");
          error.status = response.status;
          error.code = jsonCode;
          throw error;
        }
        const contentType = (response.headers.get("Content-Type") || "").toLowerCase();
        if (contentType.includes("application/json")) {
          let data;
          try {
            data = await response.json();
          } catch {
            throw new Error("Invalid server response.");
          }
          if (data && data.error) {
            throw new Error(String(data.error));
          }
          if (data != null && data.tripDocuments != null && typeof data.tripDocuments === "object") {
            const docRoot = document.querySelector("[data-trip-documents-root]");
            if (docRoot) {
              window.remiApplyTripDocumentsAjaxResponse?.(docRoot, data.tripDocuments, form);
              showToast(inferToastMessage(form));
              return;
            }
          }
        }
        if (form.getAttribute("data-remi-trip-fab-flight-add") === "1") {
          const saveAnother =
            submitterEl instanceof HTMLElement &&
            submitterEl.getAttribute("data-remi-flight-save-another") === "1";
          await refreshTripDetailsSupportSectionsFromServer();
          showToast(inferToastMessage(form));
          window.remiNotifyMobileSheetFormSaved?.(form);
          if (saveAnother) {
            window.remiResetTripFabFlightForm?.(form);
            const airlineIn = form.querySelector('input[name="flight_name"]');
            if (airlineIn instanceof HTMLElement) airlineIn.focus();
          } else {
            requestDismissMobileBottomSheet();
          }
          return;
        }
        if (form.getAttribute("data-remi-trip-fab-stay-add") === "1") {
          await refreshTripDetailsSupportSectionsFromServer();
          showToast(inferToastMessage(form));
          window.remiNotifyMobileSheetFormSaved?.(form);
          requestDismissMobileBottomSheet();
          return;
        }
        if (form.getAttribute("data-remi-trip-fab-vehicle-add") === "1") {
          await refreshTripDetailsSupportSectionsFromServer();
          showToast(inferToastMessage(form));
          window.remiNotifyMobileSheetFormSaved?.(form);
          requestDismissMobileBottomSheet();
          return;
        }
        if (form.getAttribute("data-remi-trip-fab-add-stop") === "1") {
          const saveAnother =
            submitterEl instanceof HTMLElement &&
            submitterEl.getAttribute("data-remi-itinerary-save-another") === "1";
          const inMobile = Boolean(form.closest("#mobile-sheet-stop"));
          const startH = form.querySelector("input.remi-datetime-iso[name='start_at']");
          const startVal = startH instanceof HTMLInputElement ? startH.value.trim() : "";
          const dateIso = startVal.length >= 10 ? startVal.slice(0, 10) : "";
          const titleInp = form.querySelector('input[name="title"]');
          const titleHint = titleInp instanceof HTMLInputElement ? titleInp.value.trim() : "";
          let dayNumber = null;
          const ts = (form.getAttribute("data-trip-start") || "").trim();
          if (dateIso && ts) {
            const a = new Date(`${ts}T12:00:00`);
            const b = new Date(`${dateIso}T12:00:00`);
            if (!Number.isNaN(a.getTime()) && !Number.isNaN(b.getTime())) {
              const dn = Math.floor((b.getTime() - a.getTime()) / 864e5) + 1;
              if (Number.isFinite(dn) && dn >= 1) dayNumber = dn;
            }
          }
          await refreshTripDetailsSupportSectionsFromServer();
          showToast(inferToastMessage(form));
          await expandAndFocusNewItineraryAfterAddStop({
            dateIso,
            titleHint,
            dayNumber,
            skipRowFocus: saveAnother
          });
          if (saveAnother) {
            const freshForm = inMobile
              ? document.querySelector("#mobile-sheet-stop form[data-remi-trip-fab-add-stop]")
              : document.querySelector(
                  ".trip-sidebar-itinerary-section--stop form[data-remi-trip-fab-add-stop]"
                );
            if (freshForm instanceof HTMLFormElement) {
              window.remiNotifyMobileSheetFormSaved?.(freshForm);
              window.remiResetTripFabAddStopForm?.(freshForm, { preserveStartAt: startVal });
              const loc = freshForm.querySelector("[data-location-input]");
              if (loc instanceof HTMLElement) loc.focus();
            }
          } else {
            window.remiNotifyMobileSheetFormSaved?.(form);
            requestDismissMobileBottomSheet();
          }
          return;
        }
        if (
          form.classList.contains("remi-add-commute-form") &&
          (form.action || "").includes("/itinerary/commute")
        ) {
          const saveAnother =
            submitterEl instanceof HTMLElement &&
            submitterEl.getAttribute("data-remi-commute-save-another") === "1";
          const prevId = form.id || "";
          await refreshTripDetailsSupportSectionsFromServer();
          showToast(inferToastMessage(form));
          if (saveAnother) {
            const freshForm = prevId ? document.getElementById(prevId) : null;
            if (freshForm instanceof HTMLFormElement) {
              window.remiNotifyMobileSheetFormSaved?.(freshForm);
              window.remiResetTripFabCommuteForm?.(freshForm);
              const titleInp = freshForm.querySelector('input[name="title"]');
              if (titleInp instanceof HTMLElement) titleInp.focus();
            }
          } else {
            window.remiNotifyMobileSheetFormSaved?.(form);
            requestDismissMobileBottomSheet();
          }
          return;
        }
        if (form.matches("[data-remi-unified-expense-add-form]")) {
          const saveAnother =
            submitterEl instanceof HTMLElement &&
            (submitterEl.getAttribute("data-remi-save-expense-another") === "1" ||
              submitterEl.getAttribute("data-remi-save-group-expense-another") === "1");
          const msg = inferToastMessage(form);
          if (saveAnother) {
            if (document.body.classList.contains("budget-page")) {
              try {
                sessionStorage.setItem(TOAST_KEY, msg);
                sessionStorage.setItem("remi_focus_budget_expense_title", "1");
              } catch (e) {
                /* ignore */
              }
              window.location.reload();
              return;
            }
            showToast(msg);
            if (document.querySelector(".trip-details-page .budget-tile")) {
              await refreshBudgetTilesFromPage();
            }
            window.remiResetExpenseAddForm?.(form);
            const titleInp = form.querySelector('input[name="title"]');
            if (titleInp instanceof HTMLElement) titleInp.focus();
            return;
          }
          try {
            sessionStorage.setItem(TOAST_KEY, msg);
          } catch (e) {
            /* ignore */
          }
          window.location.reload();
          return;
        }
        if (form.matches("[data-remi-expense-add-form]")) {
          const saveExpenseAnother =
            submitterEl instanceof HTMLElement &&
            submitterEl.getAttribute("data-remi-save-expense-another") === "1";
          const msg = inferToastMessage(form);
          if (saveExpenseAnother) {
            if (document.body.classList.contains("budget-page")) {
              try {
                sessionStorage.setItem(TOAST_KEY, msg);
                sessionStorage.setItem("remi_focus_budget_expense_title", "1");
              } catch (e) {
                /* ignore */
              }
              window.location.reload();
              return;
            }
            showToast(msg);
            if (document.querySelector(".trip-details-page .budget-tile")) {
              await refreshBudgetTilesFromPage();
            }
            window.remiResetExpenseAddForm?.(form);
            const titleInp = form.querySelector('input[name="title"]');
            if (titleInp instanceof HTMLElement) titleInp.focus();
            return;
          }
          try {
            sessionStorage.setItem(TOAST_KEY, msg);
          } catch (e) {
            /* ignore */
          }
          window.location.reload();
          return;
        }
        if (form.matches("[data-remi-trip-tab-add-form]")) {
          const saveGroupAnother =
            submitterEl instanceof HTMLElement &&
            submitterEl.getAttribute("data-remi-save-group-expense-another") === "1";
          const msg = inferToastMessage(form);
          if (saveGroupAnother) {
            showToast(msg);
            if (document.querySelector(".trip-details-page .budget-tile")) {
              await refreshBudgetTilesFromPage();
            }
            window.remiResetExpenseAddForm?.(form);
            const titleInp = form.querySelector('input[name="title"]');
            if (titleInp instanceof HTMLElement) titleInp.focus();
            return;
          }
          try {
            sessionStorage.setItem(TOAST_KEY, msg);
          } catch (e) {
            /* ignore */
          }
          window.location.reload();
          return;
        }
        if (form.classList.contains("check-row")) {
          const done = String(formData.get("done")) === "true";
          updateChecklistToggleUI(form, done);
        }
        if (/\/checklist\/[^/]+\/update$/i.test(form.action || "")) {
          updateChecklistEditUI(form);
        }
        if (
          /\/checklist\/[^/]+\/(toggle|update|delete)/i.test(form.action || "") &&
          (form.closest("[data-trip-keep-board-root]") || form.closest("[data-trip-details-keep-live]"))
        ) {
          void window.remiSyncTripKeepAfterMutation?.();
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
        } else if (
          document.querySelector(".trip-unified-bookings-section") &&
          /\/trips\/[^/]+\/(accommodation|vehicle-rental|flights)\/[^/]+\/update$/i.test(form.action || "")
        ) {
          await refreshTripDetailsSupportSectionsFromServer();
          if (/\/trips\/[^/]+\/(accommodation|vehicle-rental|flights)\/[^/]+\/update$/i.test(form.action || "")) {
            await syncAllRenderedItineraryRows();
          }
          if (form.id && form.id.includes("-edit-")) {
            closeInlineEdit(form.id);
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
              ".timeline-item, .expense-item, .reminder-checklist-item, .budget-tx-view, .budget-mobile-tx-item, .trip-unified-booking-row"
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
        window.remiNotifyMobileSheetFormSaved?.(form);
      } catch (error) {
        optimisticMutation.rollback();
        let dayLabelInput = form.querySelector("input[data-day-label-input]");
        if (!dayLabelInput && form.id) {
          dayLabelInput = document.querySelector(`input[data-day-label-input][form="${CSS.escape(form.id)}"]`);
        }
        if (dayLabelInput) {
          delete dayLabelInput.dataset.lastSubmittedValue;
        }
        const status = Number(error?.status || 0);
        const code = String(error?.code || "").toLowerCase();
        if (status === 409 || code === "conflict") {
          showToast(
            error?.message ||
              "Someone else updated this expense a moment ago. Reopen it to review the latest values, then try again."
          );
          return;
        }
        showToast(error?.message || "Unable to save right now.");
      }
    })();
  };

  const bindAjaxSubmitForm = (form) => {
    if (!(form instanceof HTMLFormElement) || !form.hasAttribute("data-ajax-submit")) return;
    if (form.dataset.remiAjaxWired === "1") return;
    form.dataset.remiAjaxWired = "1";
    form.addEventListener("submit", handleAjaxFormSubmit);
  };
  window.remiBindAjaxSubmitForm = bindAjaxSubmitForm;
  document.querySelectorAll("form[data-ajax-submit]").forEach((form) => bindAjaxSubmitForm(form));
  document.querySelectorAll("form[data-remi-unified-expense-add-form]").forEach((form) => {
    window.remiWireUnifiedExpenseForm?.(form);
  });

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
        if (f.dataset.remiConfirmWired === "1") return;
        window.remiWireAppConfirmOnForm?.(f);
      });
      destTbody?.querySelectorAll("form[data-ajax-submit]").forEach((f) => {
        if (f.dataset.remiAjaxWired === "1") return;
        window.remiBindAjaxSubmitForm?.(f);
      });
      window.remiEnsureCsrfOnPostForms?.(destTbody || document);
      window.setupTabSplitRootsIn?.(destTbody || document);
      window.rewireTabInlineEditOpenButtons?.();
      window.remiWireDateFieldsIn?.(destTbody || document);
      window.remiWireDateTimeFieldsIn?.(destTbody || document);
      window.remiWireTimeFieldsIn?.(destTbody || document);
      window.remiSyncTabExpenseInstantFilter?.();
      btn.textContent = "Showing all transactions";
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
        if (f.dataset.remiConfirmWired === "1") return;
        window.remiWireAppConfirmOnForm?.(f);
      });
      ul.querySelectorAll("form[data-ajax-submit]").forEach((f) => {
        if (f.dataset.remiAjaxWired === "1") return;
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
      if (f.dataset.remiConfirmWired === "1") return;
      window.remiWireAppConfirmOnForm?.(f);
    });
    target.querySelectorAll("form[data-ajax-submit]").forEach((f) => {
      if (f.dataset.remiAjaxWired === "1") return;
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

  document.querySelectorAll("input[data-day-label-input]").forEach((input) => {
    const form = formOwnerForField(input);
    const AUTO_SAVE_DEBOUNCE_MS = 300;
    let autoSaveTimer = 0;
    const clearAutoSaveTimer = () => {
      if (!autoSaveTimer) return;
      window.clearTimeout(autoSaveTimer);
      autoSaveTimer = 0;
    };
    const normalizedInitial = () => (input.dataset.initialValue || "").trim();
    const normalizedCurrent = () => (input.value || "").trim();
    const setDirtyState = () => {
      if (!form) return;
      form.classList.toggle("day-label-dirty", normalizedCurrent() !== normalizedInitial());
    };
    const submitIfDirty = () => {
      if (!form) return;
      const current = normalizedCurrent();
      if (current === normalizedInitial()) return;
      if (input.dataset.lastSubmittedValue === current) return;
      input.dataset.lastSubmittedValue = current;
      if (typeof form.requestSubmit === "function") {
        form.requestSubmit();
        return;
      }
      form.submit();
    };
    input.dataset.initialValue = input.value || "";
    setDirtyState();
    input.addEventListener("input", () => {
      setDirtyState();
      clearAutoSaveTimer();
      if (!form) return;
      autoSaveTimer = window.setTimeout(() => {
        submitIfDirty();
      }, AUTO_SAVE_DEBOUNCE_MS);
    });
    input.addEventListener("blur", () => {
      clearAutoSaveTimer();
      submitIfDirty();
    });
    input.addEventListener("keydown", (event) => {
      event.stopPropagation();
      const key = event.key || "";
      const isEnter = key === "Enter" || key === "NumpadEnter" || event.keyCode === 13 || event.which === 13;
      if (!isEnter) return;
      event.preventDefault();
      clearAutoSaveTimer();
      submitIfDirty();
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
  // Only intercept click on <summary>: preventDefault stops <details> toggle. Do not use capture-phase
  // mousedown/keydown on summary when the target is the day-label input — that runs *before* the event
  // reaches the input and breaks typing (e.g. Space still toggling the section).
  document.querySelectorAll(".day-group > summary").forEach((summaryEl) => {
    summaryEl.addEventListener("click", (event) => {
      if (!(event.target instanceof Element)) return;
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
  let tripFabIconBridgeGeneration = 0;
  const isTripFabFlyoutSheet = (el) =>
    el instanceof HTMLElement && el.classList.contains("mobile-sheet--trip-fab-flyout");
  const isTripFabFlyoutSlideContext = () => !!document.querySelector("main.trip-fab-flyout-root");
  const clearMobileSheetHeaderIconSlots = () => {
    document.querySelectorAll("[data-mobile-sheet-header-icon-slot]").forEach((el) => {
      el.replaceChildren();
    });
  };
  const removeTripFabIconFlyClones = () => {
    document.querySelectorAll(".mobile-fab-icon-fly-clone").forEach((node) => node.remove());
  };
  const placeMaterialIconInHeaderSlot = (slot, glyph) => {
    if (!(slot instanceof HTMLElement) || !glyph) return;
    slot.replaceChildren();
    const perm = document.createElement("span");
    perm.className = "material-symbols-outlined";
    perm.setAttribute("aria-hidden", "true");
    perm.textContent = glyph;
    slot.appendChild(perm);
  };
  /** Target center + end scale for FAB → drawer header bridge (slot may be :empty with 0×0 box). */
  const getTripFabHeaderIconFlyTarget = (slot, sourceIconHeight) => {
    const row = slot?.closest(".mobile-sheet-title-row");
    const h3 = row?.querySelector("h3");
    const targetGlyphPx = 22;
    const gap = 8;
    if (h3 instanceof HTMLElement) {
      const r = h3.getBoundingClientRect();
      const cx = r.left - gap - targetGlyphPx / 2;
      const cy = r.top + r.height / 2;
      const scaleEnd = targetGlyphPx / Math.max(sourceIconHeight, 1);
      return { cx, cy, scaleEnd };
    }
    const sr = slot.getBoundingClientRect();
    return {
      cx: sr.left + sr.width / 2,
      cy: sr.top + sr.height / 2,
      scaleEnd: targetGlyphPx / Math.max(sourceIconHeight, 1)
    };
  };
  /**
   * Spring-physics motion (stiffness / damping) in pixel + scale space.
   * Uses a small coupling factor so k=300, c=30 integrates stably at 60fps; hard-capped at maxMs.
   */
  const springTripFabIconBridge = (from, to, stiffness, damping, maxMs, onFrame, onComplete) => {
    const pos = { x: from.x, y: from.y, s: from.s };
    const vel = { x: 0, y: 0, s: 0 };
    const t0 = performance.now();
    let lastT = t0;
    const k = stiffness;
    const c = damping;
    const coupling = 1 / 900;
    const step = (now) => {
      const dt = Math.min(1 / 24, (now - lastT) / 1000);
      lastT = now;
      const elapsed = now - t0;
      for (const key of ["x", "y", "s"]) {
        const acc = (k * (to[key] - pos[key]) - c * vel[key]) * coupling;
        vel[key] += acc * dt;
        pos[key] += vel[key] * dt;
      }
      onFrame(pos);
      const err =
        Math.abs(to.x - pos.x) + Math.abs(to.y - pos.y) + Math.abs(to.s - pos.s) * 120;
      const spd = Math.abs(vel.x) + Math.abs(vel.y) + Math.abs(vel.s);
      if (elapsed >= maxMs || (err < 2.5 && spd < 12)) {
        onFrame(to);
        onComplete();
        return;
      }
      requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
  };
  const startTripFabMenuIconBridge = (snapshot, targetSheet) => {
    if (!snapshot?.glyph || !(targetSheet instanceof HTMLElement)) return;
    const gen = ++tripFabIconBridgeGeneration;
    const slot = targetSheet.querySelector("[data-mobile-sheet-header-icon-slot]");
    if (!(slot instanceof HTMLElement)) return;
    const fly = getTripFabHeaderIconFlyTarget(slot, snapshot.baseH);
    const clone = document.createElement("span");
    clone.className = "material-symbols-outlined mobile-fab-icon-fly-clone";
    clone.textContent = snapshot.glyph;
    if (snapshot.fontSize) clone.style.fontSize = snapshot.fontSize;
    if (snapshot.fontVariationSettings) clone.style.fontVariationSettings = snapshot.fontVariationSettings;
    document.body.appendChild(clone);
    const apply = (cx, cy, s) => {
      clone.style.left = `${cx}px`;
      clone.style.top = `${cy}px`;
      clone.style.transform = `translate(-50%, -50%) scale(${s})`;
    };
    apply(snapshot.cx, snapshot.cy, 1);
    springTripFabIconBridge(
      { x: snapshot.cx, y: snapshot.cy, s: 1 },
      { x: fly.cx, y: fly.cy, s: fly.scaleEnd },
      300,
      30,
      350,
      (p) => apply(p.x, p.y, p.s),
      () => {
        if (gen !== tripFabIconBridgeGeneration) return;
        clone.remove();
        placeMaterialIconInHeaderSlot(slot, snapshot.glyph);
      }
    );
  };
  const focusMobileCommuteForm = () => {
    const form = document.getElementById("mobile-add-commute-form");
    const focusable =
      form?.querySelector("select[name='commute_from_item_id']") ||
      form?.querySelector("input, textarea, select");
    window.setTimeout(() => {
      if (focusable && typeof focusable.focus === "function") focusable.focus();
    }, 120);
  };
  const tripFlyoutEditMetaBySheetId = {
    "mobile-sheet-stop": { kind: "stop", open: "stop", editTitle: "Edit Stop" },
    "mobile-sheet-commute": { kind: "commute", open: "commute", editTitle: "Edit Commute Leg" },
    "mobile-sheet-flight": { kind: "flight", open: "flight", editTitle: "Edit Flight" },
    "mobile-sheet-accommodation": { kind: "accommodation", open: "accommodation", editTitle: "Edit Accommodation" },
    "mobile-sheet-vehicle": { kind: "vehicle", open: "vehicle", editTitle: "Edit Vehicle Rental" }
  };
  const tripFlyoutSheetByOpenKey = {
    stop: "mobile-sheet-stop",
    commute: "mobile-sheet-commute",
    flight: "mobile-sheet-flight",
    accommodation: "mobile-sheet-accommodation",
    vehicle: "mobile-sheet-vehicle",
    checklist: "mobile-sheet-checklist",
    expense: "mobile-sheet-expense"
  };
  const parseTripFlyoutKindAndIDFromForm = (form, formId = "") => {
    if (!(form instanceof HTMLFormElement)) return null;
    const action = form.getAttribute("action") || "";
    const commuteID = action.match(/\/itinerary\/([^/]+)\/update/i)?.[1] || "";
    if (form.classList.contains("commute-itinerary-form") || formId.startsWith("commute-itinerary-edit-")) {
      return commuteID ? { kind: "commute", id: commuteID, sheetId: "mobile-sheet-commute" } : null;
    }
    const stopID = action.match(/\/itinerary\/([^/]+)\/update/i)?.[1] || "";
    if (formId.startsWith("itinerary-edit-")) {
      return stopID ? { kind: "stop", id: stopID, sheetId: "mobile-sheet-stop" } : null;
    }
    const flightID = action.match(/\/flights\/([^/]+)\/update/i)?.[1] || "";
    if (flightID) return { kind: "flight", id: flightID, sheetId: "mobile-sheet-flight" };
    const accommodationID = action.match(/\/accommodation\/([^/]+)\/update/i)?.[1] || "";
    if (accommodationID) return { kind: "accommodation", id: accommodationID, sheetId: "mobile-sheet-accommodation" };
    const vehicleID = action.match(/\/vehicle-rental\/([^/]+)\/update/i)?.[1] || "";
    if (vehicleID) return { kind: "vehicle", id: vehicleID, sheetId: "mobile-sheet-vehicle" };
    return null;
  };
  const tripFlyoutStateBySheetId = new Map();
  const getTripFlyoutSheetDefaultBody = (sheet) => {
    if (!(sheet instanceof HTMLElement)) return null;
    if (sheet.id === "mobile-sheet-stop" || sheet.id === "mobile-sheet-commute") {
      return sheet.querySelector(":scope > .trip-itinerary-drawer-scroll");
    }
    return sheet.querySelector(":scope > form");
  };
  const ensureTripFlyoutSheetState = (sheet) => {
    if (!(sheet instanceof HTMLElement) || !sheet.id) return null;
    if (tripFlyoutStateBySheetId.has(sheet.id)) return tripFlyoutStateBySheetId.get(sheet.id);
    const titleEl = sheet.querySelector(":scope .mobile-sheet-head h3");
    const defaultBody = getTripFlyoutSheetDefaultBody(sheet);
    if (!(titleEl instanceof HTMLElement) || !(defaultBody instanceof HTMLElement)) return null;
    const state = {
      defaultTitle: titleEl.textContent || "",
      titleEl,
      defaultBody
    };
    tripFlyoutStateBySheetId.set(sheet.id, state);
    return state;
  };
  const resetTripFlyoutSheetAddMode = (sheet) => {
    const state = ensureTripFlyoutSheetState(sheet);
    if (!state) return;
    state.titleEl.textContent = state.defaultTitle;
    state.defaultBody.classList.remove("hidden");
    const editBody = sheet.querySelector(":scope > .remi-trip-flyout-edit-body");
    if (editBody) editBody.remove();
    delete sheet.dataset.remiTripFlyoutEditActive;
  };
  const resetAllTripFlyoutSheets = () => {
    Object.keys(tripFlyoutEditMetaBySheetId).forEach((sheetId) => {
      const sheet = document.getElementById(sheetId);
      if (sheet instanceof HTMLElement) resetTripFlyoutSheetAddMode(sheet);
    });
  };
  /** Cloned DOM (e.g. trip FAB edit flyout) copies `data-remi-*-wired` flags from the live form; strip so pickers re-bind. */
  const stripRemiDomWireMarkersForClone = (root) => {
    if (!(root instanceof HTMLElement)) return;
    const ATTRS = [
      "data-remi-date-wired",
      "data-remi-time-wired",
      "data-remi-datetime-wired",
      "data-remi-inline-open-wired",
      "data-remi-ajax-wired",
      "data-remi-confirm-wired",
      "data-remi-itinerary-form-bound",
      "data-remi-mobile-sheet-dirty-wired",
      "data-remi-attachment-field-wired",
      "data-remi-unified-expense-wired",
      "data-remi-flight-airport-bound",
      "data-remi-flight-airline-bound",
      "data-remi-stop-weather-wired",
      "data-remi-vehicle-dropoff-bound"
    ];
    ATTRS.forEach((attr) => {
      root.querySelectorAll(`[${attr}]`).forEach((el) => el.removeAttribute(attr));
    });
    root.querySelectorAll(".edit-attachment-ui").forEach((el) => {
      if (el instanceof HTMLElement) el.remove();
    });
  };

  const wireDynamicInputsIn = (root) => {
    if (!(root instanceof HTMLElement)) return;
    window.remiBindAjaxSubmitForm &&
      root.querySelectorAll("form[data-ajax-submit]").forEach((f) => window.remiBindAjaxSubmitForm(f));
    window.remiWireDateFieldsIn?.(root);
    window.remiWireDateTimeFieldsIn?.(root);
    window.remiWireTimeFieldsIn?.(root);
    window.remiWireInlineEditOpenButtonsIn?.(root);
    window.remiWireAppConfirmOnForm &&
      root.querySelectorAll("form[data-app-confirm]").forEach((f) => window.remiWireAppConfirmOnForm(f));
    window.remiWireMobileSheetFormDirtyIn?.(root);
    window.remiEnsureCsrfOnPostForms?.(root);
    window.remiInitFlightAirportAutocompleteIn?.(root);
    window.remiInitFlightAirlineAutocompleteIn?.(root);
    window.remiInitStopWeatherIn?.(root);
    window.remiInitAttachmentEditFieldsIn?.(root);
    root.querySelectorAll("[data-itinerary-form]").forEach((f) => {
      if (f instanceof HTMLFormElement && !f.dataset.remiItineraryFormBound) {
        initItineraryForm(f);
      }
    });
  };
  const findTripFlyoutSourceForm = (kind, id) => {
    const normalizedKind = String(kind || "").toLowerCase();
    const normalizedID = String(id || "").trim();
    if (!normalizedKind || !normalizedID) return null;
    const selectorsByKind = {
      stop: `form.item-edit[id^="itinerary-edit-"][action*="/itinerary/${CSS.escape(normalizedID)}/update"]`,
      commute: `form.commute-itinerary-form[action*="/itinerary/${CSS.escape(normalizedID)}/update"]`,
      flight: `form.item-edit[action*="/flights/${CSS.escape(normalizedID)}/update"]`,
      accommodation: `form.item-edit[action*="/accommodation/${CSS.escape(normalizedID)}/update"]`,
      vehicle: `form.item-edit[action*="/vehicle-rental/${CSS.escape(normalizedID)}/update"]`
    };
    const selector = selectorsByKind[normalizedKind];
    if (!selector) return null;
    const form = document.querySelector(selector);
    return form instanceof HTMLFormElement ? form : null;
  };
  const parseTripIDFromAction = (action) => {
    const m = String(action || "").match(/\/trips\/([^/]+)/i);
    return m && m[1] ? m[1] : "";
  };
  window.remiHandleDesktopTripFlyoutEdit = (form, formId = "") => {
    if (!(form instanceof HTMLFormElement)) return false;
    const parsed = parseTripFlyoutKindAndIDFromForm(form, formId || "");
    if (!parsed) return false;
    const tripFabRoot = document.querySelector("main.trip-fab-flyout-root");
    const sheet = document.getElementById(parsed.sheetId);
    if (!(tripFabRoot instanceof HTMLElement) || !(sheet instanceof HTMLElement) || typeof openMobileSheet !== "function") {
      const tripID = parseTripIDFromAction(form.getAttribute("action") || "");
      if (!tripID) return false;
      const meta = tripFlyoutEditMetaBySheetId[parsed.sheetId];
      if (!meta) return false;
      const target = `/trips/${tripID}?open=${encodeURIComponent(meta.open)}&edit=${encodeURIComponent(
        `${parsed.kind}:${parsed.id}`
      )}`;
      window.location.assign(target);
      return true;
    }
    const sourceForm = form instanceof HTMLFormElement ? form : findTripFlyoutSourceForm(parsed.kind, parsed.id);
    if (!(sourceForm instanceof HTMLFormElement)) return false;
    /*
     * Open sheet first so its internal "reset to add mode" lifecycle runs once.
     * Then mount the edit clone; this preserves existing values in edit mode.
     */
    openMobileSheet(parsed.sheetId);
    const openSheet = document.getElementById(parsed.sheetId);
    if (!(openSheet instanceof HTMLElement)) return false;
    const state = ensureTripFlyoutSheetState(openSheet);
    if (!state) return false;
    resetTripFlyoutSheetAddMode(openSheet);
    state.defaultBody.classList.add("hidden");
    const editBody = document.createElement("div");
    editBody.className = "trip-itinerary-drawer-scroll remi-trip-flyout-edit-body";
    const editFormClone = sourceForm.cloneNode(true);
    if (!(editFormClone instanceof HTMLElement)) return false;
    editFormClone.classList.remove("hidden", "item-edit");
    editFormClone.classList.add("remi-trip-widget-form");
    const flyoutFormIds = {
      stop: `itinerary-edit-flyout-${parsed.id}`,
      commute: `commute-itinerary-edit-flyout-${parsed.id}`,
      flight: `flight-itinerary-edit-flyout-${parsed.id}`,
      accommodation: `accommodation-itinerary-edit-flyout-${parsed.id}`,
      vehicle: `vehicle-rental-itinerary-edit-flyout-${parsed.id}`
    };
    const nextFlyoutId = flyoutFormIds[parsed.kind];
    if (nextFlyoutId) {
      editFormClone.id = nextFlyoutId;
    } else {
      editFormClone.removeAttribute("id");
    }
    stripRemiDomWireMarkersForClone(editFormClone);
    editBody.appendChild(editFormClone);
    openSheet.appendChild(editBody);
    const meta = tripFlyoutEditMetaBySheetId[parsed.sheetId];
    if (meta?.editTitle) state.titleEl.textContent = meta.editTitle;
    openSheet.dataset.remiTripFlyoutEditActive = "1";
    wireDynamicInputsIn(editBody);
    window.remiInitVehicleDropoffFieldsetsIn?.(editBody);
    window.setTimeout(() => {
      editBody.querySelector("input:not([type='hidden']), textarea, select")?.focus();
    }, 80);
    return true;
  };
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

  const MOBILE_SHEET_BREAKPOINT = 920;
  const remiMobileSheetFormDirty = new WeakMap();
  const remiPrefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)");

  const clearDirtyStateForSheet = (sheet) => {
    if (!(sheet instanceof HTMLElement)) return;
    sheet.querySelectorAll("form").forEach((f) => {
      if (f instanceof HTMLFormElement) remiMobileSheetFormDirty.set(f, false);
    });
  };

  const wireMobileSheetFormDirty = (form) => {
    if (!(form instanceof HTMLFormElement)) return;
    if (form.dataset.remiMobileSheetDirtyWired === "1") return;
    if (!form.closest(".mobile-sheet")) return;
    form.dataset.remiMobileSheetDirtyWired = "1";
    remiMobileSheetFormDirty.set(form, false);
    try {
      Object.defineProperty(form, "isDirty", {
        configurable: true,
        get: () => remiMobileSheetFormDirty.get(form) === true
      });
    } catch (e) {
      /* ignore */
    }
    const mark = () => remiMobileSheetFormDirty.set(form, true);
    form.addEventListener("input", mark, true);
    form.addEventListener("change", mark, true);
    form.addEventListener(
      "remi-mobile-sheet-saved",
      () => {
        remiMobileSheetFormDirty.set(form, false);
      },
      true
    );
  };

  const wireMobileSheetFormsIn = (root = document) => {
    const scope = root instanceof HTMLElement || root instanceof Document ? root : document;
    scope.querySelectorAll(".mobile-sheet form").forEach((f) => wireMobileSheetFormDirty(f));
  };

  window.remiNotifyMobileSheetFormSaved = (form) => {
    if (!(form instanceof HTMLFormElement)) return;
    if (!form.closest(".mobile-sheet")) return;
    form.dispatchEvent(new CustomEvent("remi-mobile-sheet-saved", { bubbles: false }));
  };
  window.remiWireMobileSheetFormDirtyIn = wireMobileSheetFormsIn;

  const remiResetTripFabFlightForm = (form) => {
    if (!(form instanceof HTMLFormElement)) return;
    form.reset();
    form.querySelectorAll("[data-remi-datetime]").forEach((w) => {
      if (w instanceof HTMLElement) delete w.dataset.remiDatetimeWired;
    });
    window.remiWireDateTimeFieldsIn?.(form);
    window.remiWireMobileSheetFormDirtyIn?.(form);
  };
  window.remiResetTripFabFlightForm = remiResetTripFabFlightForm;

  const remiResetTripFabAddStopForm = (form, opts) => {
    if (!(form instanceof HTMLFormElement)) return;
    const preserveStart =
      opts && typeof opts.preserveStartAt === "string" ? opts.preserveStartAt.trim() : "";
    form.reset();
    form.querySelectorAll("[data-remi-datetime]").forEach((w) => {
      if (w instanceof HTMLElement) delete w.dataset.remiDatetimeWired;
    });
    form.dispatchEvent(new CustomEvent("remi-itinerary-form-reset-save-another", { bubbles: false }));
    window.remiWireDateTimeFieldsIn?.(form);
    if (preserveStart) {
      const startIso = form.querySelector("input.remi-datetime-iso[name='start_at']");
      if (startIso instanceof HTMLInputElement) {
        startIso.value = preserveStart;
        startIso.dispatchEvent(new Event("change", { bubbles: true }));
      }
    }
    window.remiWireMobileSheetFormDirtyIn?.(form);
  };
  window.remiResetTripFabAddStopForm = remiResetTripFabAddStopForm;

  const remiResetTripFabCommuteForm = (form) => {
    if (!(form instanceof HTMLFormElement)) return;
    form.reset();
    form.querySelectorAll("[data-remi-datetime]").forEach((w) => {
      if (w instanceof HTMLElement) delete w.dataset.remiDatetimeWired;
    });
    window.remiWireDateTimeFieldsIn?.(form);
    window.remiWireMobileSheetFormDirtyIn?.(form);
  };
  window.remiResetTripFabCommuteForm = remiResetTripFabCommuteForm;

  const sheetHasDirtyForm = (sheet) => {
    if (!(sheet instanceof HTMLElement)) return false;
    const forms = sheet.querySelectorAll("form");
    for (let i = 0; i < forms.length; i++) {
      const f = forms[i];
      if (f instanceof HTMLFormElement && remiMobileSheetFormDirty.get(f) === true) return true;
    }
    return false;
  };

  const getDismissableTripFabSheet = () => {
    if (window.innerWidth > MOBILE_SHEET_BREAKPOINT) return null;
    if (!document.querySelector("main.trip-fab-flyout-root")) return null;
    return (
      mobileSheets.find(
        (s) =>
          s instanceof HTMLElement &&
          !s.classList.contains("hidden") &&
          isTripFabFlyoutSheet(s) &&
          s.classList.contains("mobile-sheet--trip-fab-flyout-open")
      ) || null
    );
  };

  /**
   * After drag release: ease-out to off-screen in one continuous motion (no spring “settle” pause).
   * Keeps mobile-sheet--dragging so the sheet’s CSS transition does not fight inline transform.
   */
  const animateSheetDismissTranslateY = (sheet, fromY, toY, releaseVyPxMs, onComplete) => {
    if (!(sheet instanceof HTMLElement)) {
      onComplete?.();
      return;
    }
    const f = Number(fromY);
    const t = Number(toY);
    if (!Number.isFinite(f) || !Number.isFinite(t)) {
      onComplete?.();
      return;
    }
    if (remiPrefersReducedMotion.matches) {
      sheet.style.transform = `translateY(${t}px)`;
      sheet.classList.remove("mobile-sheet--dragging");
      onComplete?.();
      return;
    }
    sheet.classList.add("mobile-sheet--dragging");
    const dist = t - f;
    const v = Math.max(0, Math.min(2.5, Number(releaseVyPxMs) || 0));
    const duration = Math.min(400, Math.max(200, 200 + dist * 0.26 - v * 130));
    const t0 = performance.now();
    const step = (now) => {
      const u = Math.min(1, (now - t0) / duration);
      const eased = 1 - (1 - u) ** 3;
      const y = f + dist * eased;
      sheet.style.transform = `translateY(${y}px)`;
      if (u < 1) {
        requestAnimationFrame(step);
      } else {
        sheet.style.transform = `translateY(${t}px)`;
        sheet.classList.remove("mobile-sheet--dragging");
        onComplete?.();
      }
    };
    requestAnimationFrame(step);
  };

  const finalizeCloseMobileSheets = () => {
    tripFabIconBridgeGeneration++;
    removeTripFabIconFlyClones();
    clearMobileSheetHeaderIconSlots();
    mobileSheets.forEach((sheet) => {
      if (!(sheet instanceof HTMLElement)) return;
      /* Hide first so clearing transform/open never flashes the “peek” state (avoids close jerk). */
      sheet.classList.add("hidden");
      sheet.setAttribute("aria-hidden", "true");
      sheet.classList.remove("mobile-sheet--trip-fab-flyout-open", "mobile-sheet--dragging");
      sheet.style.transform = "";
      clearDirtyStateForSheet(sheet);
    });
    if (mobileBackdrop) mobileBackdrop.classList.add("hidden");
    resetAllTripFlyoutSheets();
    /*
     * Desktop itinerary/booking edit uses trip-inline-actions-dropdown with the <details> kept
     * open for CSS (see applyExpenseActionsDropdownOpen). Opening the flyout clears `open`;
     * closing the sheet must restore it or Edit/Delete stay hidden until reload.
     */
    window.remiApplyExpenseActionsDropdownOpen?.();
  };

  const closeMobileSheets = () => finalizeCloseMobileSheets();

  const readSheetTranslateY = (sheet) => {
    if (!(sheet instanceof HTMLElement)) return 0;
    const m = (sheet.style.transform || "").match(/translateY\((-?[\d.]+)px\)/);
    return m ? parseFloat(m[1]) : 0;
  };

  const MOBILE_SHEET_CLOSE_MS = 520;

  /** At rest: CSS transform transition (matches open). Mid-drag: ease-out rAF (no spring stall). */
  const runMobileSheetCloseAnimationThen = (thenFn) => {
    const active = getDismissableTripFabSheet();
    if (!active || remiPrefersReducedMotion.matches) {
      thenFn();
      return;
    }
    const inlineY = readSheetTranslateY(active);
    if (Number.isFinite(inlineY) && inlineY > 8) {
      const h = active.getBoundingClientRect().height;
      const offY = Math.ceil(h + 32);
      const vyRaw = active.dataset.remiDismissVy;
      delete active.dataset.remiDismissVy;
      const vy = vyRaw != null && vyRaw !== "" ? parseFloat(vyRaw, 10) : 0;
      animateSheetDismissTranslateY(
        active,
        inlineY,
        offY,
        Number.isFinite(vy) ? vy : 0,
        thenFn
      );
      return;
    }
    active.style.transform = "";
    active.classList.remove("mobile-sheet--dragging");
    let finished = false;
    const done = () => {
      if (finished) return;
      finished = true;
      active.removeEventListener("transitionend", onEnd);
      thenFn();
    };
    const onEnd = (e) => {
      if (e.target !== active) return;
      if (e.propertyName !== "transform") return;
      done();
    };
    active.addEventListener("transitionend", onEnd);
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        active.classList.remove("mobile-sheet--trip-fab-flyout-open");
      });
    });
    window.setTimeout(done, MOBILE_SHEET_CLOSE_MS);
  };

  const DRAG_CLOSE_PX = 100;
  const DRAG_VEL_PX_MS = 0.55;

  function requestDismissMobileBottomSheet() {
    const active = getDismissableTripFabSheet();
    if (!active) {
      finalizeCloseMobileSheets();
      return;
    }
    if (sheetHasDirtyForm(active) && typeof window.remiOpenAppConfirm === "function") {
      const confirmDialogEl = document.getElementById("dialog-confirm-action");
      let discardConfirmed = false;
      const springSheetBackIfStillOpen = () => {
        if (discardConfirmed) return;
        if (!(active instanceof HTMLElement)) return;
        if (active.classList.contains("hidden")) return;
        if (!active.classList.contains("mobile-sheet--trip-fab-flyout-open")) return;
        const cur = readSheetTranslateY(active);
        if (cur <= 2) return;
        delete active.dataset.remiDismissVy;
        active.classList.remove("mobile-sheet--dragging");
        requestAnimationFrame(() => {
          active.style.transform = "";
        });
      };
      const onDialogClose = () => {
        confirmDialogEl?.removeEventListener("close", onDialogClose);
        requestAnimationFrame(() => {
          if (discardConfirmed) return;
          springSheetBackIfStillOpen();
        });
      };
      confirmDialogEl?.addEventListener("close", onDialogClose);
      window.remiOpenAppConfirm({
        title: "Discard your changes?",
        body: "You have unsaved edits in this sheet.",
        okText: "Discard",
        icon: "edit_off",
        variant: "neutral",
        onConfirm: () => {
          discardConfirmed = true;
          confirmDialogEl?.removeEventListener("close", onDialogClose);
          clearDirtyStateForSheet(active);
          runMobileSheetCloseAnimationThen(() => finalizeCloseMobileSheets());
        }
      });
      return;
    }
    runMobileSheetCloseAnimationThen(() => finalizeCloseMobileSheets());
  }

  /** Gestural / explicit dismiss entry (dirty check on trip FAB mobile sheets). */
  function closeDrawer() {
    requestDismissMobileBottomSheet();
  }
  window.remiCloseDrawer = closeDrawer;

  const initMobileSheetDragDismiss = () => {
    mobileSheets.forEach((sheet) => {
      if (!isTripFabFlyoutSheet(sheet)) return;
      const region = sheet.querySelector("[data-mobile-sheet-drag-region]");
      if (!(region instanceof HTMLElement)) return;
      if (region.dataset.remiSheetDragBound === "1") return;
      region.dataset.remiSheetDragBound = "1";

      let ptrId = null;
      let startClientY = 0;
      let dragging = false;
      let lastY = 0;
      let lastT = 0;
      let prevY = 0;
      let prevT = 0;

      region.addEventListener(
        "pointerdown",
        (e) => {
          if (e.button !== 0) return;
          if (window.innerWidth > MOBILE_SHEET_BREAKPOINT) return;
          if (!isTripFabFlyoutSlideContext()) return;
          if (!sheet.classList.contains("mobile-sheet--trip-fab-flyout-open")) return;
          ptrId = e.pointerId;
          startClientY = e.clientY;
          dragging = false;
          lastY = e.clientY;
          lastT = performance.now();
          prevY = lastY;
          prevT = lastT;
          try {
            region.setPointerCapture(e.pointerId);
          } catch (err) {
            /* ignore */
          }
        },
        { passive: true }
      );

      region.addEventListener(
        "pointermove",
        (e) => {
          if (e.pointerId !== ptrId) return;
          const dy = e.clientY - startClientY;
          if (!dragging) {
            if (dy <= 4) return;
            dragging = true;
            sheet.classList.add("mobile-sheet--dragging");
          }
          prevY = lastY;
          prevT = lastT;
          lastY = e.clientY;
          lastT = performance.now();
          const y = Math.max(0, e.clientY - startClientY);
          sheet.style.transform = `translateY(${y}px)`;
          e.preventDefault();
        },
        { passive: false }
      );

      const endDrag = (e) => {
        if (e.pointerId !== ptrId) return;
        try {
          region.releasePointerCapture(e.pointerId);
        } catch (err) {
          /* ignore */
        }
        ptrId = null;
        if (!dragging) return;
        dragging = false;
        const y = Math.max(0, e.clientY - startClientY);
        let v = 0;
        if (lastT > prevT) v = (lastY - prevY) / (lastT - prevT);
        const shouldClose = y >= DRAG_CLOSE_PX || v > DRAG_VEL_PX_MS;
        if (shouldClose) {
          sheet.dataset.remiDismissVy = String(Math.max(0, v));
          closeDrawer();
        } else {
          sheet.classList.remove("mobile-sheet--dragging");
          const cur = readSheetTranslateY(sheet);
          if (Number.isFinite(cur) && cur > 2) {
            requestAnimationFrame(() => {
              sheet.style.transform = "";
            });
          }
        }
      };

      region.addEventListener("pointerup", endDrag);
      region.addEventListener("pointercancel", endDrag);
    });
  };

  const openMobileSheet = (sheetId, fabSourceBtn = null) => {
    if (!sheetId) return;
    const target = document.getElementById(sheetId);
    if (!target) return;
    const canBridge =
      fabSourceBtn instanceof HTMLElement &&
      fabSourceBtn.closest("[data-mobile-fab-menu]") &&
      isTripFabFlyoutSheet(target) &&
      isTripFabFlyoutSlideContext();
    let bridgeSnapshot = null;
    let bridgeGlyphReduced = null;
    if (canBridge) {
      const iconEl = fabSourceBtn.querySelector(".material-symbols-outlined");
      if (iconEl instanceof HTMLElement) {
        const glyph = (iconEl.textContent || "").trim();
        if (glyph) {
          const r = iconEl.getBoundingClientRect();
          const cs = window.getComputedStyle(iconEl);
          if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
            bridgeGlyphReduced = glyph;
          } else {
            bridgeSnapshot = {
              glyph,
              cx: r.left + r.width / 2,
              cy: r.top + r.height / 2,
              baseH: Math.max(r.height, 1),
              fontSize: cs.fontSize || "",
              fontVariationSettings: cs.fontVariationSettings || ""
            };
          }
        }
      }
    }
    closeMobileSheets();
    resetTripFlyoutSheetAddMode(target);
    target.classList.remove("hidden");
    target.style.transform = "";
    target.setAttribute("aria-hidden", "false");
    clearDirtyStateForSheet(target);
    wireMobileSheetFormsIn(target);
    if (mobileBackdrop) mobileBackdrop.classList.remove("hidden");
    if (bridgeGlyphReduced) {
      const slot = target.querySelector("[data-mobile-sheet-header-icon-slot]");
      if (slot instanceof HTMLElement) placeMaterialIconInHeaderSlot(slot, bridgeGlyphReduced);
    }
    if (isTripFabFlyoutSheet(target) && isTripFabFlyoutSlideContext()) {
      target.classList.remove("mobile-sheet--trip-fab-flyout-open");
      void target.offsetWidth;
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          target.classList.add("mobile-sheet--trip-fab-flyout-open");
          if (bridgeSnapshot) {
            requestAnimationFrame(() => {
              startTripFabMenuIconBridge(bridgeSnapshot, target);
            });
          }
        });
      });
    }
  };
  const tripIdFromPath = () => {
    const m = window.location.pathname.match(/^\/trips\/([^/]+)/i);
    return m && m[1] ? m[1] : "";
  };
  const openDesktopFabFallback = (sheetId) => {
    if (!sheetId || window.innerWidth <= 920) return false;
    if (document.getElementById(sheetId)) {
      return false;
    }
    const tripId = tripIdFromPath();
    if (!tripId) return false;
    const pathBase = `/trips/${tripId}`;
    const navMap = {
      "mobile-sheet-stop": `${pathBase}?open=stop`,
      "mobile-sheet-commute": `${pathBase}?open=commute`,
      "mobile-sheet-expense": `${pathBase}?open=expense`,
      "mobile-sheet-tab": `${pathBase}?open=expense`,
      "mobile-sheet-accommodation": `${pathBase}?open=accommodation`,
      "mobile-sheet-vehicle": `${pathBase}?open=vehicle`,
      "mobile-sheet-flight": `${pathBase}?open=flight`,
      "mobile-sheet-checklist": `${pathBase}?open=checklist`
    };
    const target = navMap[sheetId];
    if (!target) return false;
    window.location.assign(target);
    return true;
  };
  const bindTripQuickSheets = () => {
    document.querySelectorAll("[data-mobile-sheet-open]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const sheetId = btn.getAttribute("data-mobile-sheet-open");
        if (openDesktopFabFallback(sheetId)) {
          closeMobileFab();
          return;
        }
        openMobileSheet(sheetId, btn);
        closeMobileFab();
        if (
          btn.getAttribute("data-fab-prefer-group-split") === "1" &&
          sheetId === "mobile-sheet-expense"
        ) {
          window.setTimeout(() => {
            const sheet = document.getElementById("mobile-sheet-expense");
            const form = sheet?.querySelector("form[data-remi-unified-expense-add-form]");
            const splitCb = form?.querySelector(".remi-unified-expense-split-checkbox");
            if (splitCb instanceof HTMLInputElement && !splitCb.disabled && !splitCb.checked) {
              splitCb.checked = true;
              splitCb.dispatchEvent(new Event("change", { bubbles: true }));
            }
          }, 60);
        }
        const dcf = document.querySelector("[data-desktop-calendar-flyout]");
        if (dcf) {
          dcf.classList.add("hidden");
          dcf.setAttribute("aria-hidden", "true");
          dcf.classList.remove("is-expanded");
        }
      });
    });
    document.querySelectorAll("[data-mobile-sheet-close]").forEach((btn) => {
      btn.addEventListener("click", () => {
        requestDismissMobileBottomSheet();
      });
    });
    if (mobileBackdrop) {
      mobileBackdrop.addEventListener("click", () => {
        requestDismissMobileBottomSheet();
      });
    }
  };
  if (mobileSheets.length > 0 || mobileBackdrop) {
    bindTripQuickSheets();
    wireMobileSheetFormsIn(document);
    initMobileSheetDragDismiss();
  }
  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") return;
    closeMobileFab();
    requestDismissMobileBottomSheet();
  });
  if (mobileFab && mobileFabToggle) {
    closeMobileFab();
    mobileFabToggle.addEventListener("click", () => {
      if (mobileFab.classList.contains("open")) {
        closeMobileFab();
      } else {
        openMobileFab();
      }
    });
    window.addEventListener("resize", closeMobileFab);
  }
  /* Trip details: FAB uses the same links as other trip pages; intercept ?open= on the main trip URL so we open flyouts without a full reload. */
  if (mobileFabMenu && document.querySelector("main.trip-fab-flyout-root")) {
    mobileFabMenu.addEventListener("click", (e) => {
      const a = e.target.closest("a.mobile-fab-option[href]");
      if (!(a instanceof HTMLAnchorElement)) return;
      let url;
      try {
        url = new URL(a.getAttribute("href") || "", window.location.origin);
      } catch {
        return;
      }
      if (url.origin !== window.location.origin) return;
      const pathTrip = url.pathname.match(/^\/trips\/([^/]+)\/?$/);
      if (!pathTrip) return;
      const pageTrip = window.location.pathname.match(/^\/trips\/([^/]+)/)?.[1];
      if (!pageTrip || pathTrip[1] !== pageTrip) return;
      const openKey = (url.searchParams.get("open") || "").toLowerCase();
      if (!openKey) return;
      const sheetId = tripFlyoutSheetByOpenKey[openKey];
      if (!sheetId || typeof openMobileSheet !== "function") return;
      if (!(document.getElementById(sheetId) instanceof HTMLElement)) return;
      e.preventDefault();
      e.stopPropagation();
      openMobileSheet(sheetId, a);
      closeMobileFab();
      const dcf = document.querySelector("[data-desktop-calendar-flyout]");
      if (dcf) {
        dcf.classList.add("hidden");
        dcf.setAttribute("aria-hidden", "true");
        dcf.classList.remove("is-expanded");
      }
    });
  }

  document.querySelectorAll(".mobile-sheet").forEach((sheet) => {
    sheet.addEventListener(
      "focusin",
      (e) => {
        const t = e.target;
        if (!(t instanceof HTMLElement)) return;
        if (!t.matches("input, textarea, select")) return;
        const sheet = t.closest(".mobile-sheet");
        const scrollRoot =
          (sheet instanceof HTMLElement ? sheet.querySelector(".trip-itinerary-drawer-scroll") : null) ||
          t.closest(".mobile-sheet > form");
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

  const sheetSearchParams = new URLSearchParams(window.location.search);
  const openParamForSheet = (sheetSearchParams.get("open") || "").toLowerCase();
  const editParamForSheet = sheetSearchParams.get("edit") || "";
  const parseSheetEditParam = (raw) => {
    const m = String(raw || "").match(/^([a-z_]+):(.+)$/i);
    if (!m) return null;
    return { kind: m[1].toLowerCase(), id: m[2] };
  };
  const stripOpenQueryParam = () => {
    try {
      const u = new URL(window.location.href);
      u.searchParams.delete("open");
      u.searchParams.delete("edit");
      u.searchParams.delete("focus");
      const qs = u.searchParams.toString();
      window.history.replaceState({}, "", u.pathname + (qs ? `?${qs}` : "") + u.hash);
    } catch (e) {
      /* ignore */
    }
  };
  const focusSheetPrimaryField = (sheetId) => {
    const sheet = document.getElementById(sheetId);
    if (!(sheet instanceof HTMLElement)) return;
    if (sheetId === "mobile-sheet-commute") {
      window.setTimeout(() => focusMobileCommuteForm(), 80);
      return;
    }
    window.setTimeout(() => {
      const loc = sheet.querySelector("[data-location-input]");
      const cat = sheet.querySelector('select[name="category"]');
      const amt = sheet.querySelector(".remi-unified-expense-amount-input");
      const focusable = loc || cat || amt || sheet.querySelector("input, textarea, select");
      if (focusable && typeof focusable.focus === "function") focusable.focus();
    }, 80);
  };
  const targetSheetId = tripFlyoutSheetByOpenKey[openParamForSheet] || "";
  if (targetSheetId && typeof openMobileSheet === "function") {
    const sheet = document.getElementById(targetSheetId);
    if (sheet instanceof HTMLElement) {
      const parsedEdit = parseSheetEditParam(editParamForSheet);
      if (parsedEdit && tripFlyoutEditMetaBySheetId[targetSheetId]) {
        const sourceForm = findTripFlyoutSourceForm(parsedEdit.kind, parsedEdit.id);
        if (sourceForm) {
          window.remiHandleDesktopTripFlyoutEdit?.(sourceForm, sourceForm.id || "");
          stripOpenQueryParam();
        } else {
          openMobileSheet(targetSheetId);
          focusSheetPrimaryField(targetSheetId);
          stripOpenQueryParam();
        }
      } else {
        openMobileSheet(targetSheetId);
        focusSheetPrimaryField(targetSheetId);
        stripOpenQueryParam();
      }
    }
  }

  const initTripPageSectionFoldsIn = (root) => {
    const scope = root instanceof Document || root instanceof HTMLElement ? root : document;
    scope.querySelectorAll("[data-trip-page-section-fold]").forEach((el) => {
      if (!(el instanceof HTMLDetailsElement)) return;
      if (el.dataset.remiTripPageSectionFoldWired === "1") return;
      el.dataset.remiTripPageSectionFoldWired = "1";
      const sync = () => {
        const ch = el.querySelector(".trip-page-section-fold__chevron");
        if (ch) ch.textContent = el.open ? "expand_less" : "expand_more";
      };
      el.addEventListener("toggle", sync);
      sync();
    });
  };
  window.remiInitTripPageSectionFoldsIn = initTripPageSectionFoldsIn;

  const initUnifiedBookingsIn = (root) => {
    const scope = root instanceof Document || root instanceof HTMLElement ? root : document;
    scope.querySelectorAll("[data-unified-bookings-panel]").forEach((panel) => {
      if (!(panel instanceof HTMLElement) || panel.dataset.remiUnifiedBookingsInit === "1") return;
      panel.dataset.remiUnifiedBookingsInit = "1";
      const list = panel.querySelector("[data-unified-bookings-list]");
      const pager = panel.querySelector("[data-unified-bookings-pager]");
      const pageLabel = pager?.querySelector("[data-unified-bookings-page-label]");
      const prevBtn = pager?.querySelector('[data-unified-bookings-page="prev"]');
      const nextBtn = pager?.querySelector('[data-unified-bookings-page="next"]');
      const filterMenu = panel.querySelector("#trip-unified-bookings-filter-menu");
      const filterTrigger = panel.querySelector("#trip-unified-bookings-filter-trigger");
      const filterValueEl = panel.querySelector("#trip-unified-bookings-filter-value");
      const stateGroup = panel.querySelector("[data-unified-bookings-state-group]");
      const detailsEl = panel.closest("[data-unified-bookings-details]");

      const kindLabels = { all: "All bookings", flight: "Flights", lodging: "Accommodation", vehicle: "Vehicle rentals" };
      let kindFilter = "all";
      let stateFilter = "all";
      let page = 0;
      const perPage = 10;

      const getRows = () => Array.from(list?.querySelectorAll(".trip-unified-booking-row") || []);

      const applyFilters = () => {
        const rows = getRows();
        rows.forEach((row) => {
          const k = row.getAttribute("data-booking-kind") || "";
          const st = row.getAttribute("data-booking-state") || "";
          let match = true;
          if (kindFilter !== "all" && k !== kindFilter) match = false;
          if (stateFilter === "booked" && st !== "booked") match = false;
          if (stateFilter === "to_book" && st !== "to_book") match = false;
          row.dataset.filterMatch = match ? "1" : "0";
        });
        const visible = rows.filter((r) => r.dataset.filterMatch === "1");
        const n = visible.length;
        const pageCount = Math.max(1, Math.ceil(n / perPage) || 1);
        if (page >= pageCount) page = Math.max(0, pageCount - 1);
        const start = page * perPage;
        const end = start + perPage;
        visible.forEach((row, i) => {
          const onPage = i >= start && i < end;
          row.hidden = !onPage;
        });
        rows.forEach((row) => {
          if (row.dataset.filterMatch === "0") row.hidden = true;
        });
        if (pageLabel) {
          pageLabel.textContent = n ? `Page ${page + 1} of ${pageCount}` : "";
        }
        if (prevBtn) prevBtn.disabled = page <= 0;
        if (nextBtn) nextBtn.disabled = page >= pageCount - 1;
        if (pager) {
          if (n > perPage) {
            pager.classList.remove("hidden");
            pager.removeAttribute("hidden");
          } else {
            pager.classList.add("hidden");
            pager.setAttribute("hidden", "");
          }
        }
      };

      const withFade = (fn) => {
        if (list) {
          list.classList.add("trip-unified-bookings--fading");
          requestAnimationFrame(() => {
            fn();
            requestAnimationFrame(() => list.classList.remove("trip-unified-bookings--fading"));
          });
        } else {
          fn();
        }
      };

      filterMenu?.addEventListener("click", (e) => {
        const li = e.target instanceof Element ? e.target.closest("li[data-unified-bookings-kind]") : null;
        if (!li || !filterMenu.contains(li)) return;
        e.stopPropagation();
        const k = li.getAttribute("data-unified-bookings-kind") || "all";
        kindFilter = k;
        page = 0;
        if (filterValueEl) filterValueEl.textContent = kindLabels[k] || "All bookings";
        filterMenu.querySelectorAll("[data-unified-bookings-kind]").forEach((x) => x.classList.remove("is-active"));
        li.classList.add("is-active");
        filterMenu.setAttribute("hidden", "");
        filterMenu.classList.add("hidden");
        filterTrigger?.setAttribute("aria-expanded", "false");
        withFade(applyFilters);
      });

      filterTrigger?.addEventListener("click", (e) => {
        e.preventDefault();
        const open = filterMenu?.hasAttribute("hidden");
        if (open) {
          filterMenu?.removeAttribute("hidden");
          filterMenu?.classList.remove("hidden");
          filterTrigger.setAttribute("aria-expanded", "true");
        } else {
          filterMenu?.setAttribute("hidden", "");
          filterMenu?.classList.add("hidden");
          filterTrigger.setAttribute("aria-expanded", "false");
        }
      });
      document.addEventListener("click", (e) => {
        if (!filterMenu || !filterTrigger) return;
        const t = e.target;
        if (!(t instanceof Node)) return;
        if (filterMenu.contains(t) || filterTrigger.contains(t)) return;
        filterMenu.setAttribute("hidden", "");
        filterMenu.classList.add("hidden");
        filterTrigger.setAttribute("aria-expanded", "false");
      });

      stateGroup?.querySelectorAll("[data-unified-bookings-state]").forEach((btn) => {
        btn.addEventListener("click", () => {
          const s = btn.getAttribute("data-unified-bookings-state") || "all";
          if (s === "booked" && stateFilter === "booked") {
            stateFilter = "all";
            btn.classList.remove("is-active");
            withFade(applyFilters);
            return;
          }
          if (s === "to_book" && stateFilter === "to_book") {
            stateFilter = "all";
            btn.classList.remove("is-active");
            withFade(applyFilters);
            return;
          }
          stateGroup.querySelectorAll("[data-unified-bookings-state]").forEach((b) => b.classList.remove("is-active"));
          if (s === "booked" || s === "to_book") {
            stateFilter = s;
            btn.classList.add("is-active");
          } else {
            stateFilter = "all";
          }
          page = 0;
          withFade(applyFilters);
        });
      });

      prevBtn?.addEventListener("click", () => {
        if (page > 0) {
          page -= 1;
          withFade(applyFilters);
        }
      });
      nextBtn?.addEventListener("click", () => {
        page += 1;
        withFade(applyFilters);
      });

      if (detailsEl instanceof HTMLDetailsElement) {
        detailsEl.addEventListener("toggle", () => {
          const ch = detailsEl.querySelector(".trip-unified-bookings__summary-chevron");
          if (ch) ch.textContent = detailsEl.open ? "expand_less" : "expand_more";
        });
      }

      applyFilters();
    });
  };
  window.remiInitUnifiedBookingsIn = initUnifiedBookingsIn;
  window.remiRefreshTripBookingsStream = () => {
    if (!document.querySelector(".trip-unified-bookings-section")) return Promise.resolve(false);
    return refreshTripDetailsSupportSectionsFromServer();
  };
  if (document.querySelector("[data-unified-bookings-panel]")) {
    initUnifiedBookingsIn(document);
  }
  if (document.querySelector("[data-trip-page-section-fold]")) {
    initTripPageSectionFoldsIn(document);
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
          /* Keep section in the tree: empty states use data-trip-search-hide-when-query; hiding
             the whole section made the dashboard look blank when the search box had any value. */
          section.hidden = false;
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
    window.requestAnimationFrame(() => applyTripSearch());
  }

  const mapEl = document.getElementById("map");
  if (!mapEl) {
    /* no trip map */
  } else {
  const gMapKey = (mapEl.getAttribute("data-google-maps-key") || "").trim();
  const gMapId = (mapEl.getAttribute("data-google-map-id") || "").trim();
  const useGoogleMaps = /^AIza[0-9A-Za-z_-]{20,}$/.test(gMapKey);
  const wantGoogleAdvancedMarkers = useGoogleMaps && gMapId.length > 0;
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
    commute: "directions_transit",
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
    /* Raster maps: dark mode via JSON styles. Vector maps (Map ID + AdvancedMarkerElement) use map colorScheme. */
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
    let remiGoogleTripUsesAdvancedMarkers = false;

    const googleTripMapDarkNow = () => document.documentElement.classList.contains("theme-dark");

    let syncGoogleTripMapTheme = () => {};

    document.addEventListener("remi:themechange", (ev) => {
      const d =
        ev && ev.detail && typeof ev.detail.dark === "boolean"
          ? ev.detail.dark
          : undefined;
      syncGoogleTripMapTheme(d);
    });

    const initGoogleTripMap = async () => {
      if (!window.google || !google.maps) return;

      let AdvancedMarkerElement = null;
      if (wantGoogleAdvancedMarkers && typeof google.maps.importLibrary === "function") {
        try {
          const markerLib = await google.maps.importLibrary("marker");
          AdvancedMarkerElement = markerLib.AdvancedMarkerElement;
        } catch (e) {
          AdvancedMarkerElement = null;
        }
      }
      const useAdv = Boolean(AdvancedMarkerElement && wantGoogleAdvancedMarkers);
      remiGoogleTripUsesAdvancedMarkers = useAdv;

      const markerContentFromBmp = (bmp) => {
        const root = document.createElement("div");
        root.style.cssText = "width:0;height:0;overflow:visible;position:relative;pointer-events:none;";
        if (!bmp.url) return root;
        const img = document.createElement("img");
        img.src = bmp.url;
        img.width = bmp.size;
        img.height = bmp.size;
        img.alt = "";
        img.draggable = false;
        img.style.cssText = `position:absolute;left:50%;bottom:0;width:${bmp.size}px;height:${bmp.size}px;transform:translate(-50%,0);display:block;`;
        root.appendChild(img);
        return root;
      };

      const createGoogleTripMarker = (p, darkMarkers) => {
        const ring = ringForDay(Math.max(1, p.day));
        const bmp = remiGoogleMapMarkerBitmap(ring, p.kind, darkMarkers);
        if (useAdv) {
          return new AdvancedMarkerElement({
            map: null,
            position: { lat: p.lat, lng: p.lng },
            title: `${p.title} · Day ${p.day}`,
            content: markerContentFromBmp(bmp),
            gmpClickable: true
          });
        }
        const iconOpts =
          bmp.url && typeof google.maps.Size === "function" && typeof google.maps.Point === "function"
            ? {
                url: bmp.url,
                scaledSize: new google.maps.Size(bmp.size, bmp.size),
                anchor: new google.maps.Point(bmp.anchorX, bmp.anchorY)
              }
            : undefined;
        return new google.maps.Marker({
          position: { lat: p.lat, lng: p.lng },
          map: null,
          title: `${p.title} · Day ${p.day}`,
          ...(iconOpts ? { icon: iconOpts } : {})
        });
      };

      const setGoogleTripMarkerMap = (m, map) => {
        if (useAdv) {
          m.map = map;
        } else {
          m.setMap(map);
        }
      };

      const getGoogleTripMarkerLatLng = (m) => {
        if (useAdv) {
          const pos = m.position;
          if (pos == null) return null;
          if (typeof pos.lat === "function") return pos;
          return new google.maps.LatLng(pos.lat, pos.lng);
        }
        return m.getPosition();
      };

      const setGoogleTripMarkerPosition = (m, lat, lng) => {
        if (useAdv) {
          m.position = { lat, lng };
        } else {
          m.setPosition({ lat, lng });
        }
      };

      const setGoogleTripMarkerTitle = (m, t) => {
        if (useAdv) {
          m.title = t;
        } else {
          m.setTitle(t);
        }
      };

      const applyGoogleTripMarkerAppearance = (m, ring, kind, isDark) => {
        const bmp = remiGoogleMapMarkerBitmap(ring, kind, isDark);
        if (useAdv) {
          m.content = markerContentFromBmp(bmp);
          return;
        }
        if (bmp.url && typeof google.maps.Size === "function" && typeof google.maps.Point === "function") {
          m.setIcon({
            url: bmp.url,
            scaledSize: new google.maps.Size(bmp.size, bmp.size),
            anchor: new google.maps.Point(bmp.anchorX, bmp.anchorY)
          });
        }
      };

      syncGoogleTripMapTheme = (darkOverride) => {
        if (!remiGoogleTripMap || !window.google || !google.maps) return;
        const dark = typeof darkOverride === "boolean" ? darkOverride : googleTripMapDarkNow();
        if (remiGoogleTripUsesAdvancedMarkers && google.maps.ColorScheme) {
          remiGoogleTripMap.setOptions({
            colorScheme: dark ? google.maps.ColorScheme.DARK : google.maps.ColorScheme.LIGHT
          });
        } else {
          remiGoogleTripMap.setOptions({
            styles: dark ? remiGmapDarkStyles : []
          });
        }
        remiGoogleTripMarkerEntries.forEach(({ marker, point }) => {
          const ring = ringForDay(Math.max(1, point.day));
          applyGoogleTripMarkerAppearance(marker, ring, point.kind, dark);
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

      const dark = googleTripMapDarkNow();
      const mapOpts = {
        center: { lat: startLat, lng: startLng },
        zoom: startZoom,
        mapTypeControl: true,
        streetViewControl: true,
        fullscreenControl: true
      };
      if (useAdv) {
        mapOpts.mapId = gMapId;
        if (google.maps.ColorScheme) {
          mapOpts.colorScheme = dark ? google.maps.ColorScheme.DARK : google.maps.ColorScheme.LIGHT;
        }
      } else {
        mapOpts.styles = dark ? remiGmapDarkStyles : [];
      }
      remiGoogleTripMap = new google.maps.Map(mapEl, mapOpts);
      const gMap = remiGoogleTripMap;
      remiGoogleTripMarkerEntries.length = 0;
      const googleMarkersByDay = new Map();
      const googleMarkerByItemId = new Map();
      uniqueDays.forEach((d) => googleMarkersByDay.set(d, []));
      const darkMarkers = googleTripMapDarkNow();
      points.forEach((p) => {
        const marker = createGoogleTripMarker(p, darkMarkers);
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
            setGoogleTripMarkerMap(m, gMap);
          });
        });
        googleMarkersByDay.forEach((markers, d) => {
          if (selectedDays.has(d)) return;
          markers.forEach((m) => setGoogleTripMarkerMap(m, null));
        });
        if (vis.length === 0) {
          gMap.setCenter({ lat: startLat, lng: startLng });
          gMap.setZoom(startZoom);
          return;
        }
        if (vis.length === 1) {
          const pos = getGoogleTripMarkerLatLng(vis[0]);
          if (pos) {
            gMap.setCenter(pos);
            gMap.setZoom(Math.max(startZoom, 12));
          }
          return;
        }
        const bounds = new google.maps.LatLngBounds();
        vis.forEach((m) => {
          const pos = getGoogleTripMarkerLatLng(m);
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
            const marker = createGoogleTripMarker(p, darkM);
            const pointRef = { ...p };
            const iw = new google.maps.InfoWindow({
              content: `<div><b>${escapeHtmlMap(p.title)}</b><br><span class="trip-map-popup-day">Day ${p.day}</span><br>${escapeHtmlMap(p.location)}</div>`
            });
            marker.addListener("click", () => iw.open({ anchor: marker, map: gMap }));
            ent = { marker, iw, point: pointRef };
            googleMarkerByItemId.set(p.itemId, ent);
          } else {
            Object.assign(ent.point, p);
            setGoogleTripMarkerPosition(ent.marker, p.lat, p.lng);
            setGoogleTripMarkerTitle(ent.marker, `${p.title} · Day ${p.day}`);
            const ring = ringForDay(Math.max(1, p.day));
            applyGoogleTripMarkerAppearance(ent.marker, ring, p.kind, darkM);
            ent.iw.setContent(
              `<div><b>${escapeHtmlMap(p.title)}</b><br><span class="trip-map-popup-day">Day ${p.day}</span><br>${escapeHtmlMap(p.location)}</div>`
            );
          }
        });
        googleMarkerByItemId.forEach((ent, id) => {
          if (seen.has(id)) return;
          setGoogleTripMarkerMap(ent.marker, null);
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
      googleInitDone = true;
      initGoogleTripMap().catch(() => fallbackToLeaflet("google-init-error"));
    } else {
      window.remiGoogleTripMapInit = () => {
        if (fallbackTried) return;
        googleInitDone = true;
        initGoogleTripMap()
          .then(() => {
            try {
              delete window.remiGoogleTripMapInit;
            } catch (e) {
              window.remiGoogleTripMapInit = undefined;
            }
          })
          .catch(() => fallbackToLeaflet("google-init-error"));
      };
      const gs = document.createElement("script");
      gs.async = true;
      gs.onerror = () => {
        fallbackToLeaflet("google-script-error");
      };
      gs.src = `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(
        gMapKey
      )}&callback=remiGoogleTripMapInit&v=weekly&loading=async`;
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
    ".expense-item, .timeline-item, .reminder-checklist-item, .page-trip-notes .trip-keep-masonry .trip-keep-card--note, .trip-details-keep-preview .trip-keep-card--note, .flight-card, .title-row, .budget-mobile-tx-item, .accommodation-card-wrap, .vehicle-rental-item, .trip-documents-item";

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
      return (
        row.getAttribute("data-title")?.trim() ||
        row.querySelector(".itinerary-item-view .vehicle-main-top h4")?.textContent?.trim() ||
        row.querySelector(".itinerary-item-view .flight-main-head h4")?.textContent?.trim() ||
        row.querySelector(".itinerary-item-view .itinerary-stop-card__title")?.textContent?.trim() ||
        row.querySelector(".itinerary-item-view strong")?.textContent?.trim() ||
        "Stop"
      );
    }
    if (row.matches(".reminder-checklist-item")) {
      const s = row.querySelector(".remi-checklist-item-text, .checklist-view > span");
      return s?.textContent?.trim() || "Checklist item";
    }
    if (row.matches(".trip-details-keep-preview .trip-keep-card--note")) {
      return row.querySelector(".trip-keep-note-title")?.textContent?.trim() || "Note";
    }
    if (row.matches(".page-trip-notes .trip-keep-masonry .trip-keep-card--note")) {
      return row.querySelector(".trip-keep-note-title")?.textContent?.trim() || "Note";
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
    let sourceButtons = actionsRoot
      ? Array.from(
          actionsRoot.querySelectorAll(".trip-inline-actions-buttons button, .trip-inline-actions-buttons a.item-action-btn")
        )
      : [];
    if (
      !sourceButtons.length &&
      (row.classList.contains("reminder-checklist-item") ||
        (row.classList.contains("trip-keep-card--note") && row.closest(".trip-details-keep-preview")))
    ) {
      const editBtn = row.querySelector(
        ".remi-checklist-item-actions .remi-checklist-action-btn--secondary[data-inline-edit-open]"
      );
      const delBtn = row.querySelector(".remi-checklist-item-actions .remi-checklist-delete-form button[type='submit']");
      sourceButtons = [editBtn, delBtn].filter(Boolean);
    }
    if (!sourceButtons.length && row.matches(".page-trip-notes .trip-keep-masonry .trip-keep-card--note")) {
      const noteEdit = row.querySelector("[data-trip-keep-note-open-edit]");
      sourceButtons = noteEdit ? [noteEdit] : [];
    }
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
      if (src.classList.contains("item-action-btn--download")) {
        b.classList.add("trip-long-press-sheet__action--download");
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
      const hasFromDetails = Boolean(
        actionsRoot?.querySelector(".trip-inline-actions-buttons button, .trip-inline-actions-buttons a.item-action-btn")
      );
      const hasFromChecklist =
        row.classList.contains("reminder-checklist-item") &&
        Boolean(
          row.querySelector(".remi-checklist-item-actions .remi-checklist-action-btn--secondary[data-inline-edit-open]")
        );
      const hasFromNote =
        row.classList.contains("trip-keep-card--note") &&
        Boolean(row.closest(".trip-details-keep-preview")) &&
        Boolean(
          row.querySelector(".remi-checklist-item-actions .remi-checklist-action-btn--secondary[data-inline-edit-open]")
        );
      const hasFromNotesPageNote =
        row.matches(".page-trip-notes .trip-keep-masonry .trip-keep-card--note") &&
        Boolean(row.querySelector("[data-trip-keep-note-open-edit]"));
      if (!hasFromDetails && !hasFromChecklist && !hasFromNote && !hasFromNotesPageNote) {
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

  document.addEventListener("remi-keep-dom-updated", syncBind);
})();

(function initTripKeepNotesSearchComposer() {
  const sync = () => {
    const input = document.getElementById("keep-search-q");
    const composer = document.querySelector("[data-trip-keep-composer-row]");
    const clearBtn = document.querySelector("[data-trip-keep-search-clear]");
    if (!(input instanceof HTMLInputElement)) return;
    const q = input.value.trim();
    if (clearBtn instanceof HTMLElement) {
      clearBtn.classList.toggle("hidden", q.length === 0);
    }
    if (composer instanceof HTMLElement) {
      composer.classList.toggle("trip-keep-composer-row--hidden-for-search", q.length > 0);
    }
  };

  const bind = () => {
    if (!document.body.classList.contains("page-trip-notes")) return;
    const form = document.querySelector("[data-trip-keep-search-form]");
    const input = document.getElementById("keep-search-q");
    const clearBtn = document.querySelector("[data-trip-keep-search-clear]");
    if (!(form instanceof HTMLFormElement) || !(input instanceof HTMLInputElement)) return;
    if (form.dataset.tripKeepSearchUiBound === "1") return;
    form.dataset.tripKeepSearchUiBound = "1";
    input.addEventListener("input", sync);
    input.addEventListener("search", sync);
    clearBtn?.addEventListener("click", () => {
      input.value = "";
      sync();
      form.submit();
    });
    sync();
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bind, { once: true });
  } else {
    bind();
  }

  document.addEventListener("remi-keep-dom-updated", () => {
    sync();
  });
})();

(function remiTripKeepLiveSync() {
  const remiKeepChangeRelevant = (entity) => {
    const e = String(entity || "")
      .toLowerCase()
      .trim();
    return e === "trip_note" || e === "checklist_item" || e === "checklist_category_pin";
  };

  const hasActiveKeepBoardEdit = (root) => {
    if (!(root instanceof HTMLElement)) return false;
    const noteDlg = document.getElementById("trip-keep-note-view-dialog");
    if (noteDlg instanceof HTMLDialogElement && noteDlg.open) return true;
    if (root.querySelector("details.trip-keep-details[open]")) return true;
    if (root.querySelector(".remi-checklist-inline-edit-form:not(.hidden)")) return true;
    return false;
  };

  const remiAfterKeepDomSwap = (root) => {
    const scope = root instanceof HTMLElement ? root : document.body;
    scope.querySelectorAll("form[data-app-confirm]").forEach((f) => {
      if (f.dataset.remiConfirmWired === "1") return;
      window.remiWireAppConfirmOnForm?.(f);
    });
    scope.querySelectorAll("form[data-ajax-submit]").forEach((f) => {
      if (f.dataset.remiAjaxWired === "1") return;
      window.remiBindAjaxSubmitForm?.(f);
    });
    window.remiEnsureCsrfOnPostForms?.(document);
    window.remiWireInlineEditOpenButtonsIn?.(scope);
    window.remiWireDateFieldsIn?.(scope);
    document.dispatchEvent(new Event("remi-keep-dom-updated"));
  };

  const fetchKeepBoardFragment = async (tripID, view, q) => {
    const u = new URL(`/api/v1/trips/${encodeURIComponent(tripID)}/keep/board-fragment`, window.location.origin);
    if (view && view !== "notes") u.searchParams.set("view", view);
    const qq = String(q || "").trim();
    if (qq) u.searchParams.set("q", qq);
    const res = await fetch(u.toString(), {
      credentials: "same-origin",
      headers: { Accept: "text/html", "X-Requested-With": "XMLHttpRequest" }
    });
    if (!res.ok) throw new Error("Could not refresh notes board.");
    return await res.text();
  };

  const fetchDetailsPreviewFragment = async (tripID) => {
    const res = await fetch(
      `/api/v1/trips/${encodeURIComponent(tripID)}/keep/details-preview-fragment`,
      {
        credentials: "same-origin",
        headers: { Accept: "text/html", "X-Requested-With": "XMLHttpRequest" }
      }
    );
    if (!res.ok) throw new Error("Could not refresh notes preview.");
    return await res.text();
  };

  window.remiSyncTripKeepAfterMutation = async () => {
    try {
      const notesBody = document.body;
      if (notesBody.classList.contains("page-trip-notes") && notesBody.hasAttribute("data-trip-keep-live")) {
        const tid = notesBody.getAttribute("data-trip-id") || "";
        if (tid) {
          const view = notesBody.getAttribute("data-keep-view") || "notes";
          const input = document.querySelector("#keep-search-q");
          const qFromInput = input instanceof HTMLInputElement ? input.value.trim() : "";
          const qFromUrl = new URLSearchParams(window.location.search).get("q") || "";
          const q = qFromInput || qFromUrl;
          const cur = document.querySelector("[data-trip-keep-board-root]");
          if (cur?.parentNode && !hasActiveKeepBoardEdit(cur)) {
            const html = await fetchKeepBoardFragment(tid, view, q);
            const tpl = document.createElement("template");
            tpl.innerHTML = html.trim();
            const next = tpl.content.firstElementChild;
            if (next) {
              cur.parentNode.replaceChild(next, cur);
              remiAfterKeepDomSwap(next);
            }
          }
        }
      }
      const host = document.querySelector("[data-trip-details-keep-live]");
      if (host instanceof HTMLElement) {
        const tid = host.getAttribute("data-trip-id") || "";
        if (tid && !hasActiveKeepBoardEdit(host)) {
          const html = await fetchDetailsPreviewFragment(tid);
          host.innerHTML = html.trim();
          remiAfterKeepDomSwap(host);
        }
      }
    } catch {
      /* ignore */
    }
  };

  const startNotesPageKeepPolling = () => {
    if (window.remiRealtimeTripSyncEnabled) return;
    const body = document.body;
    if (!body.classList.contains("page-trip-notes") || !body.hasAttribute("data-trip-keep-live")) return;
    const tripID = body.getAttribute("data-trip-id") || "";
    if (!tripID) return;
    let since = "";
    let inflight = false;
    let timer = null;
    const schedule = (ms) => {
      if (timer) window.clearTimeout(timer);
      timer = window.setTimeout(tick, Math.max(500, ms));
    };
    const tick = async () => {
      if (inflight) {
        schedule(2000);
        return;
      }
      if (document.hidden) {
        schedule(20000);
        return;
      }
      const boardMount = document.querySelector("[data-trip-keep-board-root]");
      if (hasActiveKeepBoardEdit(boardMount)) {
        schedule(4000);
        return;
      }
      inflight = true;
      try {
        const url = `/api/v1/trips/${encodeURIComponent(tripID)}/changes${since ? `?since=${encodeURIComponent(since)}` : ""}`;
        const res = await fetch(url, { credentials: "same-origin", headers: { Accept: "application/json" } });
        if (!res.ok) {
          schedule(12000);
          return;
        }
        const payload = await res.json();
        const changes = Array.isArray(payload?.changes) ? payload.changes : [];
        let relevant = false;
        if (changes.length) {
          const last = changes[changes.length - 1];
          if (last?.changed_at) since = String(last.changed_at);
          relevant = changes.some((c) => remiKeepChangeRelevant(c?.entity));
        }
        if (relevant) {
          const view = body.getAttribute("data-keep-view") || "notes";
          const input = document.querySelector("#keep-search-q");
          const qFromInput = input instanceof HTMLInputElement ? input.value.trim() : "";
          const qFromUrl = new URLSearchParams(window.location.search).get("q") || "";
          const q = qFromInput || qFromUrl;
          const html = await fetchKeepBoardFragment(tripID, view, q);
          const cur = document.querySelector("[data-trip-keep-board-root]");
          if (cur?.parentNode && !hasActiveKeepBoardEdit(cur)) {
            const tpl = document.createElement("template");
            tpl.innerHTML = html.trim();
            const next = tpl.content.firstElementChild;
            if (next) {
              cur.parentNode.replaceChild(next, cur);
              remiAfterKeepDomSwap(next);
            }
          }
        }
        schedule(5000);
      } catch {
        schedule(12000);
      } finally {
        inflight = false;
      }
    };
    schedule(800);
    document.addEventListener("visibilitychange", () => {
      if (!document.hidden) schedule(400);
    });
  };

  const startTripDetailsKeepPolling = () => {
    if (window.remiRealtimeTripSyncEnabled) return;
    const host = document.querySelector("[data-trip-details-keep-live]");
    if (!host) return;
    const tripID = host.getAttribute("data-trip-id") || "";
    if (!tripID) return;
    let since = "";
    let inflight = false;
    let timer = null;
    const schedule = (ms) => {
      if (timer) window.clearTimeout(timer);
      timer = window.setTimeout(tick, Math.max(500, ms));
    };
    const tick = async () => {
      if (inflight) {
        schedule(2000);
        return;
      }
      if (document.hidden) {
        schedule(20000);
        return;
      }
      if (hasActiveKeepBoardEdit(host)) {
        schedule(4000);
        return;
      }
      inflight = true;
      try {
        const url = `/api/v1/trips/${encodeURIComponent(tripID)}/changes${since ? `?since=${encodeURIComponent(since)}` : ""}`;
        const res = await fetch(url, { credentials: "same-origin", headers: { Accept: "application/json" } });
        if (!res.ok) {
          schedule(12000);
          return;
        }
        const payload = await res.json();
        const changes = Array.isArray(payload?.changes) ? payload.changes : [];
        let relevant = false;
        if (changes.length) {
          const last = changes[changes.length - 1];
          if (last?.changed_at) since = String(last.changed_at);
          relevant = changes.some((c) => remiKeepChangeRelevant(c?.entity));
        }
        if (relevant) {
          const html = await fetchDetailsPreviewFragment(tripID);
          host.innerHTML = html.trim();
          remiAfterKeepDomSwap(host);
        }
        schedule(5000);
      } catch {
        schedule(12000);
      } finally {
        inflight = false;
      }
    };
    schedule(800);
    document.addEventListener("visibilitychange", () => {
      if (!document.hidden) schedule(400);
    });
  };

  window.remiRealtimeTripSyncEnabled = typeof window.EventSource === "function";

  const startZeroRefreshTripSync = () => {
    if (!window.remiRealtimeTripSyncEnabled) return;
    const tripMatch = window.location.pathname.match(/^\/trips\/([^/]+)/i);
    const tripID = tripMatch ? decodeURIComponent(tripMatch[1]) : "";
    if (!tripID) return;

    let refreshQueued = false;
    let refreshInFlight = false;
    let pendingViewportRefresh = false;
    const softRefreshActiveTripPage = async () => {
      if (document.hidden) {
        refreshQueued = true;
        return;
      }
      if (
        document.querySelector(".item-edit:not(.hidden), .trip-keep-form:not(.hidden), .trip-keep-form.item-edit:not(.hidden)") ||
        hasActiveKeepBoardEdit(document.querySelector("[data-trip-keep-board-root]")) ||
        hasActiveKeepBoardEdit(document.querySelector("[data-trip-details-keep-live]"))
      ) {
        refreshQueued = true;
        window.setTimeout(() => {
          if (!refreshInFlight) void softRefreshActiveTripPage();
        }, 700);
        return;
      }
      refreshInFlight = true;
      refreshQueued = false;
      try {
        if (document.querySelector("main.trip-details-page")) {
          await syncAllRenderedItineraryRows();
          await refreshBudgetTilesFromPage();
          await window.remiSyncTripKeepAfterMutation?.();
          await refreshTripDetailsSupportSectionsFromServer();
          return;
        }
        if (document.querySelector("main.tab-page")) {
          await window.refreshTheTabPageFromServer();
          await refreshSharedTripChromeFromServer();
          return;
        }
        if (document.body.classList.contains("page-trip-notes")) {
          const view = document.body.getAttribute("data-keep-view") || "notes";
          const input = document.querySelector("#keep-search-q");
          const qFromInput = input instanceof HTMLInputElement ? input.value.trim() : "";
          const qFromUrl = new URLSearchParams(window.location.search).get("q") || "";
          const html = await fetchKeepBoardFragment(tripID, view, qFromInput || qFromUrl);
          const cur = document.querySelector("[data-trip-keep-board-root]");
          if (cur?.parentNode) {
            const tpl = document.createElement("template");
            tpl.innerHTML = html.trim();
            const next = tpl.content.firstElementChild;
            if (next) {
              cur.parentNode.replaceChild(next, cur);
              remiAfterKeepDomSwap(next);
            }
          }
          await refreshSharedTripChromeFromServer();
          return;
        }
        window.location.reload();
      } catch (e) {
        window.location.reload();
      } finally {
        refreshInFlight = false;
        if (refreshQueued) {
          refreshQueued = false;
          void softRefreshActiveTripPage();
        }
      }
    };

    const source = new EventSource(`/api/v1/trips/${encodeURIComponent(tripID)}/events?after_id=latest`);
    source.addEventListener("change", () => {
      if (refreshInFlight) {
        refreshQueued = true;
        return;
      }
      void softRefreshActiveTripPage();
    });
    source.onerror = () => {
      refreshQueued = true;
    };
    document.addEventListener("visibilitychange", () => {
      if (!document.hidden && (refreshQueued || pendingViewportRefresh)) {
        pendingViewportRefresh = false;
        void softRefreshActiveTripPage();
      }
    });
    if (typeof window.matchMedia === "function") {
      const mq = window.matchMedia("(max-width: 980px)");
      const onViewportSwap = () => {
        pendingViewportRefresh = true;
        if (!document.hidden) {
          void softRefreshActiveTripPage();
        }
      };
      if (typeof mq.addEventListener === "function") {
        mq.addEventListener("change", onViewportSwap);
      } else if (typeof mq.addListener === "function") {
        mq.addListener(onViewportSwap);
      }
    }
  };

  const boot = () => {
    startZeroRefreshTripSync();
    startNotesPageKeepPolling();
    startTripDetailsKeepPolling();
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
    if (form.dataset.remiGmapsKeyWired !== "1") {
      try {
        initSiteSettingsGoogleMapsKeySection(form);
        form.dataset.remiGmapsKeyWired = "1";
      } catch (err) {
        if (typeof window.remiToast === "function") {
          window.remiToast("Map settings controls are temporarily unavailable.");
        }
      }
    }
    if (form.dataset.remiAirlabsKeyWired !== "1") {
      try {
        initSiteSettingsAirLabsKeySection(form);
        form.dataset.remiAirlabsKeyWired = "1";
      } catch (err) {
        if (typeof window.remiToast === "function") {
          window.remiToast("AirLabs API key controls are temporarily unavailable.");
        }
      }
    }
    if (form.dataset.remiOpenweatherKeyWired !== "1") {
      try {
        initSiteSettingsOpenWeatherKeySection(form);
        form.dataset.remiOpenweatherKeyWired = "1";
      } catch (err) {
        if (typeof window.remiToast === "function") {
          window.remiToast("OpenWeatherMap API key controls are temporarily unavailable.");
        }
      }
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
    if (!(wrap instanceof HTMLElement) || wrap.dataset.remiCarouselBound === "1") return;
    wrap.dataset.remiCarouselBound = "1";
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

  const initMobileEntryCarouselsIn = (root) => {
    const scope = root instanceof HTMLElement || root instanceof Document ? root : document;
    scope.querySelectorAll("[data-mobile-entry-carousel]").forEach(setupCarousel);
  };
  window.remiSetupMobileEntryCarouselsIn = initMobileEntryCarouselsIn;
  const boot = () => {
    initMobileEntryCarouselsIn(document);
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

  function parseMoneyToCentsClient(value) {
    const raw = String(value ?? "").trim().replace(/,/g, "");
    if (!raw) return 0;
    const sign = raw.startsWith("-") ? -1 : 1;
    const normalized = raw.replace(/^[+-]/, "");
    const parts = normalized.split(".");
    if (parts.length > 2) return 0;
    const whole = parts[0] || "0";
    let frac = parts[1] || "";
    if (!/^\d+$/.test(whole) || (frac && !/^\d+$/.test(frac))) return 0;
    frac = (frac + "00").slice(0, 2);
    return sign * ((parseInt(whole, 10) * 100) + parseInt(frac || "0", 10));
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
      sum += parseMoneyToCentsClient(weights[k] || 0);
    });
    return sum;
  }

  function exactSumsMatch(root) {
    const total = parseMoneyToCentsClient(root.querySelector("[data-tab-amount-input]")?.value || "");
    const sum = exactSelectedSum(root);
    return total === sum;
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
    const totalCents = parseMoneyToCentsClient(root.querySelector("[data-tab-amount-input]")?.value || "");
    const sumCents = exactSelectedSum(root);
    const remainingCents = totalCents - sumCents;
    const remaining = remainingCents / 100;
    let remText;
    if (remainingCents < 0) {
      remText = `-${sym}${(-remaining).toFixed(2)}`;
    } else {
      remText = `${sym}${remaining.toFixed(2)}`;
    }
    remainingEl.textContent = remText;
    if (matchEl) {
      const ok = remainingCents === 0;
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
      btn.setAttribute("title", "Enter an amount before configuring the split.");
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
    const amountsCents = {};
    const mode = (root.querySelector("[data-tab-split-mode]")?.value || "equal").toLowerCase();
    keys.forEach((k) => {
      if (mode === "exact") {
        amountsCents[k] = parseMoneyToCentsClient(raw[k] || 0);
      } else {
        weights[k] = raw[k] || 0;
      }
    });
    const payload = mode === "exact"
      ? { participants: keys, amounts_cents: amountsCents }
      : { participants: keys, weights };
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
        equal: "Split equally among selected members.",
        exact: "Enter each person’s share (must total the expense amount).",
        percent: "Percentages must total 100%.",
        shares: "Allocate costs using share units.",
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
      const initEntries = initMode === "exact"
        ? Object.entries(initJson.amounts_cents || {}).map(([k, v]) => [k, Number(v) / 100])
        : Object.entries(initJson.weights || {});
      initEntries.forEach(([k, v]) => {
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
        const defaultEntries = tripMode === "exact"
          ? Object.entries(tripDef.amounts_cents || {}).map(([k, v]) => [k, Number(v) / 100])
          : Object.entries(tripDef.weights || {});
        defaultEntries.forEach(([k, v]) => {
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

  function applySection(section) {
    if (!(section instanceof HTMLElement)) return;
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

  function apply() {
    document.querySelectorAll("[data-tab-expenses-section]").forEach((section) => {
      applySection(section);
    });
  }

  let mqBound = false;
  function bindMqOnce() {
    if (mqBound) return;
    mqBound = true;
    const mq = window.matchMedia?.("(max-width: 920px)");
    if (mq?.addEventListener) {
      mq.addEventListener("change", () => apply());
    } else if (mq?.addListener) {
      mq.addListener(() => apply());
    }
  }

  function wire() {
    document.querySelectorAll("[data-tab-expenses-section]").forEach((section) => {
      if (!(section instanceof HTMLElement)) return;
      if (section.dataset.remiTabExpFilterWired === "1") return;
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
      applySection(section);
    });
    bindMqOnce();
  }

  window.remiSyncTabExpenseInstantFilter = apply;

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", wire, { once: true });
  } else {
    wire();
  }
})();

const TAB_OVER_TIME_PAGE_SIZE = 7;

/** When opening Group Expenses via #add-group-expense (or legacy #add-tab), scroll to Add New Expense after layout. */
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
  const mqDocActionsMobile = window.matchMedia("(max-width: 980px)");
  const mainShell = document.querySelector("main.app-shell[data-max-upload-file-size-mb]");
  const getTripDocMaxBytes = () => {
    const raw = mainShell?.getAttribute("data-max-upload-file-size-mb") || "";
    const mb = Math.max(1, parseInt(raw, 10) || 5);
    return mb * 1024 * 1024;
  };
  const uploadFeedbacks = Array.from(root.querySelectorAll("[data-trip-doc-upload-feedback]"));
  const clearTripDocUploadFeedback = () => {
    uploadFeedbacks.forEach((el) => {
      if (el instanceof HTMLElement) {
        el.textContent = "";
        el.classList.add("hidden");
      }
    });
  };
  const setTripDocUploadFeedback = (msg) => {
    const t = String(msg || "").trim();
    if (!t) {
      clearTripDocUploadFeedback();
      return;
    }
    const cap = t.charAt(0).toUpperCase() + t.slice(1);
    uploadFeedbacks.forEach((el) => {
      if (el instanceof HTMLElement) {
        el.textContent = cap;
        el.classList.remove("hidden");
      }
    });
  };
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

  window.remiRefreshTripDocumentsListFilter = () => {
    const ul = root.querySelector(".trip-documents-items");
    if (!ul) return;
    items.length = 0;
    items.push(...ul.querySelectorAll("[data-trip-doc-item]"));
    apply();
  };

  const dropzone = root.querySelector("[data-trip-doc-dropzone]");
  const input = root.querySelector("[data-trip-doc-file-input]");
  const selectedList = root.querySelector("[data-trip-doc-selected-list]");

  /** @type {File[]} */
  let tripDocStagedFiles = [];

  const tripDocFileKey = (f) => `${f.name}\0${f.size}\0${f.lastModified}`;

  const syncTripDocUploadSubmitLabel = () => {
    const btn = root.querySelector(".trip-documents-upload-submit");
    if (!(btn instanceof HTMLButtonElement)) return;
    const label = "Upload";
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
    let added = 0;
    for (let i = 0; i < fileList.length; i++) {
      const f = fileList[i];
      if (typeof isBlocked === "function" && isBlocked(f.name)) {
        const bm =
          blockedMsg || "This file type is not allowed — executables and scripts cannot be uploaded.";
        setTripDocUploadFeedback(bm);
        window.remiShowToast?.(bm);
        continue;
      }
      const k = tripDocFileKey(f);
      if (!seen.has(k)) {
        seen.add(k);
        tripDocStagedFiles.push(f);
        added += 1;
      }
    }
    syncTripDocStagedToInput();
    refreshTripDocSelectedList();
    if (added > 0) clearTripDocUploadFeedback();
  };

  const selectionStatus = root.querySelector("[data-trip-doc-selection-status]");

  const refreshTripDocSelectedList = () => {
    syncTripDocUploadSubmitLabel();
    if (selectionStatus instanceof HTMLElement) {
      selectionStatus.classList.toggle("hidden", tripDocStagedFiles.length > 0);
    }
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
    input.addEventListener("click", () => {
      clearTripDocUploadFeedback();
    });
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

  const validateFn = window.remiValidateBookingAttachmentFile;
  const uploadForm = root.querySelector(".trip-documents-upload-form");
  const mobileForm = root.querySelector(".trip-documents-mobile-upload");
  const mobileInput = root.querySelector("[data-trip-doc-mobile-file-input]");

  const runTripDocumentsAjaxUpload = async (formEl, fileCountHint) => {
    if (!(formEl instanceof HTMLFormElement)) return;
    let r;
    try {
      const fd = new FormData(formEl);
      r = await fetch(formEl.action, {
        method: "POST",
        body: fd,
        headers: {
          "X-Requested-With": "XMLHttpRequest",
          Accept: "application/json"
        }
      });
    } catch (e) {
      const msg = e?.message || "Upload failed.";
      setTripDocUploadFeedback(msg);
      window.remiShowToast?.(msg);
      return;
    }
    const raw = await r.text();
    let data = null;
    try {
      data = raw ? JSON.parse(raw) : null;
    } catch {
      data = null;
    }
    if (!r.ok) {
      const msg = (data && data.error) || raw.trim() || "Upload failed.";
      setTripDocUploadFeedback(msg);
      window.remiShowToast?.(msg);
      return;
    }
    if (!data || !data.tripDocuments) {
      const msg = "Unexpected server response.";
      setTripDocUploadFeedback(msg);
      window.remiShowToast?.(msg);
      return;
    }
    if (window.remiApplyTripDocumentsAjaxResponse?.(root, data.tripDocuments, formEl)) {
      tripDocStagedFiles = [];
      if (input instanceof HTMLInputElement) {
        input.value = "";
        syncTripDocStagedToInput();
      }
      refreshTripDocSelectedList();
      clearTripDocUploadFeedback();
      if (mobileInput instanceof HTMLInputElement) {
        mobileInput.value = "";
      }
      const n = typeof fileCountHint === "number" && fileCountHint > 0 ? fileCountHint : 1;
      window.remiShowToast?.(n > 1 ? `${n} documents uploaded.` : "Document uploaded.");
    }
  };

  if (uploadForm instanceof HTMLFormElement && typeof validateFn === "function") {
    uploadForm.addEventListener("submit", (e) => {
      e.preventDefault();
      if (tripDocStagedFiles.length === 0) {
        setTripDocUploadFeedback("Please select at least one file.");
        window.remiShowToast?.("Please select at least one file.");
        return;
      }
      const maxB = getTripDocMaxBytes();
      const nFiles = tripDocStagedFiles.length;
      void (async () => {
        for (let i = 0; i < tripDocStagedFiles.length; i++) {
          const f = tripDocStagedFiles[i];
          const r = await validateFn(f, maxB);
          if (!r || !r.ok) {
            const msg = (r && r.message) || "Upload could not be validated.";
            setTripDocUploadFeedback(msg);
            window.remiShowToast?.(msg);
            return;
          }
        }
        clearTripDocUploadFeedback();
        await runTripDocumentsAjaxUpload(uploadForm, nFiles);
      })();
    });
  }

  if (mobileForm instanceof HTMLFormElement && mobileInput instanceof HTMLInputElement) {
    mobileInput.addEventListener("click", () => {
      clearTripDocUploadFeedback();
    });
    if (typeof validateFn === "function") {
      mobileInput.addEventListener("change", () => {
        const files = mobileInput.files;
        if (!files || files.length === 0) return;
        const maxB = getTripDocMaxBytes();
        const nFiles = files.length;
        void (async () => {
          for (let i = 0; i < files.length; i++) {
            const f = files[i];
            if (typeof window.remiIsBlockedUploadFilename === "function" && window.remiIsBlockedUploadFilename(f.name)) {
              const bm =
                window.remiBlockedUploadFilenameMessage ||
                "This file type is not allowed — executables and scripts cannot be uploaded.";
              setTripDocUploadFeedback(bm);
              window.remiShowToast?.(bm);
              mobileInput.value = "";
              return;
            }
            const r = await validateFn(f, maxB);
            if (!r || !r.ok) {
              const msg = (r && r.message) || "Upload could not be validated.";
              setTripDocUploadFeedback(msg);
              window.remiShowToast?.(msg);
              return;
            }
          }
          clearTripDocUploadFeedback();
          await runTripDocumentsAjaxUpload(mobileForm, nFiles);
        })();
      });
    } else {
      mobileInput.addEventListener("change", () => {
        if (mobileInput.files && mobileInput.files.length > 0) {
          void runTripDocumentsAjaxUpload(mobileForm, mobileInput.files.length);
        }
      });
    }
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
      clearTripDocUploadFeedback();
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
      if (mqDocActionsMobile.matches) {
        const actionsDetails = openBtn.closest("details.trip-documents-item-actions");
        if (actionsDetails instanceof HTMLDetailsElement) {
          actionsDetails.removeAttribute("open");
        }
      }
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
      return;
    }
    if (mqDocActionsMobile.matches) {
      const openActions = root.querySelector("details.trip-documents-item-actions[open]");
      if (openActions instanceof HTMLDetailsElement) {
        e.preventDefault();
        openActions.removeAttribute("open");
      }
    }
  });

  const closeOtherDocActionMenus = (keepOpen) => {
    root.querySelectorAll("details.trip-documents-item-actions[open]").forEach((d) => {
      if (d !== keepOpen) d.removeAttribute("open");
    });
  };
  root.addEventListener("toggle", (e) => {
    if (!mqDocActionsMobile.matches) return;
    const t = e.target;
    if (!(t instanceof HTMLDetailsElement) || !t.classList.contains("trip-documents-item-actions")) return;
    if (t.open) {
      closeOtherDocActionMenus(t);
    }
  });
  document.addEventListener(
    "pointerdown",
    (e) => {
      if (!mqDocActionsMobile.matches) return;
      const t = e.target;
      if (!(t instanceof Node)) return;
      const openDetails = root.querySelector("details.trip-documents-item-actions[open]");
      if (!openDetails || !(openDetails instanceof HTMLDetailsElement)) return;
      if (openDetails.contains(t)) return;
      openDetails.removeAttribute("open");
    },
    true
  );
})();

/** Budget donut: scale center amount so it stays inside the ring (same abbrev string as trip budget tile). */
(function remiBudgetDonutCenterFit() {
  const MIN_PX = 11;
  const WIDTH_FRAC = 0.96;

  const fitOne = (valueEl) => {
    const fit = valueEl.closest(".budget-donut-center-value-fit");
    if (!(fit instanceof HTMLElement)) return;
    const maxW = fit.clientWidth * WIDTH_FRAC;
    if (maxW <= 1) return;
    valueEl.style.fontSize = "";
    let hi = parseFloat(window.getComputedStyle(valueEl).fontSize);
    if (!Number.isFinite(hi) || hi < MIN_PX) hi = MIN_PX;
    valueEl.style.fontSize = `${hi}px`;
    if (valueEl.scrollWidth <= maxW) return;
    let lo = MIN_PX;
    let best = lo;
    for (let i = 0; i < 28; i++) {
      const mid = (lo + hi) / 2;
      valueEl.style.fontSize = `${mid}px`;
      if (valueEl.scrollWidth <= maxW) {
        best = mid;
        lo = mid;
      } else {
        hi = mid;
      }
    }
    valueEl.style.fontSize = `${best}px`;
  };

  const fitIn = (root) => {
    const scope = root instanceof HTMLElement ? root : document;
    scope.querySelectorAll(".budget-donut-center-value").forEach((el) => {
      if (el instanceof HTMLElement) fitOne(el);
    });
  };

  let ro;
  const observeWrappers = () => {
    if (typeof ResizeObserver !== "function") return;
    document.querySelectorAll(".budget-donut-wrapper").forEach((w) => {
      if (!(w instanceof HTMLElement)) return;
      if (w.dataset.remiDonutFitObserved === "1") return;
      w.dataset.remiDonutFitObserved = "1";
      if (!ro) {
        ro = new ResizeObserver((entries) => {
          for (const ent of entries) {
            const t = ent.target;
            if (!(t instanceof HTMLElement)) continue;
            t.querySelectorAll(".budget-donut-center-value").forEach((el) => {
              if (el instanceof HTMLElement) fitOne(el);
            });
          }
        });
      }
      ro.observe(w);
    });
  };

  window.remiFitBudgetDonutCenterValues = (root) => {
    observeWrappers();
    fitIn(root ?? document);
  };

  const boot = () => {
    window.remiFitBudgetDonutCenterValues(document);
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }

  document.body?.addEventListener("htmx:afterSwap", (event) => {
    const target = event?.detail?.target;
    if (!(target instanceof HTMLElement)) return;
    const hasDonut =
      target.matches(".budget-donut-wrapper, .budget-donut, .budget-donut-center, .budget-donut-center-value-fit") ||
      target.querySelector(".budget-donut-center-value");
    if (!hasDonut) return;
    window.remiFitBudgetDonutCenterValues(target);
  });

  if (document.fonts?.ready) {
    document.fonts.ready.then(() => window.remiFitBudgetDonutCenterValues(document)).catch(() => {});
  }
})();

(function initFirstTripSetup() {
  const STEP_META = [
    {
      step: "STEP 01 — BASICS",
      k: "01",
      sidebarTitle: "Trip basics",
      sidebarDesc: "Name your adventure and capture the story you want to remember.",
      nextHint: "Next: Cover image",
    },
    {
      step: "STEP 02 — AESTHETICS",
      k: "02",
      sidebarTitle: "Cover image",
      sidebarDesc: "Choose a hero image or a built-in style for your trip header.",
      nextHint: "Next: Locals & currency",
    },
    {
      step: "STEP 03 — LOGISTICS",
      k: "03",
      sidebarTitle: "Logistics",
      sidebarDesc: "Set currency, distance units, and how times and dates display on this trip.",
      nextHint: "Next: Personalization",
    },
    {
      step: "STEP 04 — PERSONALIZATION",
      k: "04",
      sidebarTitle: "Personalization",
      sidebarDesc: "Choose solo or group travel and turn trip sections on or off.",
      nextHint: "Next: Saved notes & checklists",
    },
    {
      step: "STEP 05 — NOTES",
      k: "05",
      sidebarTitle: "Saved notes & checklists",
      sidebarDesc: "Optionally copy global notes and checklist templates into this trip.",
      nextHint: "",
    },
  ];

  function syncCurrencyFields(root, sel) {
    const wrap = root.querySelector("[data-fts-currency-other-wrap]");
    const cust = root.querySelector("[data-fts-currency-custom]");
    const symPost = root.querySelector("[data-fts-currency-symbol-post]");
    const symEdit = root.querySelector("[data-fts-currency-symbol-edit]");
    const isOther = sel.value === "__OTHER__";
    if (wrap) {
      if (isOther) wrap.removeAttribute("hidden");
      else wrap.setAttribute("hidden", "");
    }
    if (cust instanceof HTMLInputElement) {
      cust.required = isOther;
    }
    if (symEdit instanceof HTMLInputElement) {
      symEdit.required = isOther;
    }
    if (symPost instanceof HTMLInputElement) {
      if (isOther) {
        if (symEdit instanceof HTMLInputElement) symPost.value = symEdit.value;
      } else {
        const opt = sel.options[sel.selectedIndex];
        const ds = opt?.dataset?.symbol;
        if (ds) symPost.value = ds;
      }
    }
  }

  function syncCurrencyFromTrip(root, sel) {
    const symPost = root.querySelector("[data-fts-currency-symbol-post]");
    const symEdit = root.querySelector("[data-fts-currency-symbol-edit]");
    const code = (root.getAttribute("data-currency-code") || "").trim();
    if (!code) {
      syncCurrencyFields(root, sel);
      return;
    }
    const u = code.toUpperCase();
    const hasPreset = Array.from(sel.options).some((o) => o.value === u);
    if (hasPreset) {
      sel.value = u;
    } else {
      sel.value = "__OTHER__";
      const wrap = root.querySelector("[data-fts-currency-other-wrap]");
      const cust = root.querySelector("[data-fts-currency-custom]");
      if (wrap) wrap.removeAttribute("hidden");
      if (cust instanceof HTMLInputElement) cust.value = code;
      const sym = (root.getAttribute("data-currency-symbol") || "").trim();
      if (symEdit instanceof HTMLInputElement) symEdit.value = sym || cust.value;
      if (symPost instanceof HTMLInputElement && symEdit instanceof HTMLInputElement) symPost.value = symEdit.value;
    }
    syncCurrencyFields(root, sel);
  }

  function wireCoverUI(root) {
    const modeInput = root.querySelector("[data-fts-cover-mode]");
    const fileInput = root.querySelector("[data-fts-cover-file]");
    const drop = root.querySelector("[data-fts-cover-drop]");
    const filenameEl = root.querySelector("[data-fts-cover-filename]");
    const preview = root.querySelector("[data-fts-cover-preview]");
    const initial = (root.getAttribute("data-trip-cover") || "").trim();
    const presetBtns = root.querySelectorAll("[data-fts-cover-preset]");
    const clearBtn = root.querySelector("[data-fts-cover-clear]");

    function setMode(m) {
      if (modeInput instanceof HTMLInputElement) modeInput.value = m;
    }

    function clearPresetVisual() {
      presetBtns.forEach((b) => b.classList.remove("is-selected"));
    }

    function clearFilePreview() {
      if (fileInput instanceof HTMLInputElement) fileInput.value = "";
      if (filenameEl instanceof HTMLElement) {
        filenameEl.textContent = "";
        filenameEl.hidden = true;
      }
      if (preview instanceof HTMLElement) {
        preview.innerHTML = "";
        preview.hidden = true;
      }
    }

    presetBtns.forEach((btn) => {
      btn.addEventListener("click", () => {
        const v = btn.getAttribute("data-fts-cover-preset") || "";
        setMode(v);
        clearPresetVisual();
        btn.classList.add("is-selected");
        clearFilePreview();
      });
    });

    if (clearBtn) {
      clearBtn.addEventListener("click", () => {
        setMode("clear");
        clearPresetVisual();
        clearFilePreview();
      });
    }

    if (fileInput instanceof HTMLInputElement && drop instanceof HTMLElement) {
      fileInput.addEventListener("change", () => {
        const f = fileInput.files?.[0];
        if (!f) return;
        setMode("upload");
        clearPresetVisual();
        if (filenameEl instanceof HTMLElement) {
          filenameEl.textContent = f.name;
          filenameEl.hidden = false;
        }
        if (preview instanceof HTMLElement) {
          preview.replaceChildren();
          const im = document.createElement("img");
          im.src = URL.createObjectURL(f);
          im.alt = "";
          preview.appendChild(im);
          preview.hidden = false;
        }
      });

      drop.addEventListener("click", () => fileInput.click());
      drop.addEventListener("keydown", (e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          fileInput.click();
        }
      });

      ["dragenter", "dragover"].forEach((ev) => {
        drop.addEventListener(ev, (e) => {
          e.preventDefault();
          drop.classList.add("is-dragover");
        });
      });
      ["dragleave", "drop"].forEach((ev) => {
        drop.addEventListener(ev, (e) => {
          e.preventDefault();
          drop.classList.remove("is-dragover");
        });
      });
      drop.addEventListener("drop", (e) => {
        const f = e.dataTransfer?.files?.[0];
        if (!f || !String(f.type || "").startsWith("image/")) return;
        try {
          const dt = new DataTransfer();
          dt.items.add(f);
          fileInput.files = dt.files;
        } catch {
          return;
        }
        fileInput.dispatchEvent(new Event("change", { bubbles: true }));
      });
    }

    if (initial) {
      if (initial === "default" || initial.startsWith("pattern:")) {
        const match = Array.from(presetBtns).find((b) => (b.getAttribute("data-fts-cover-preset") || "") === initial);
        if (match) {
          clearPresetVisual();
          match.classList.add("is-selected");
        }
      } else if (initial.startsWith("http://") || initial.startsWith("https://") || initial.startsWith("/static/")) {
        if (preview instanceof HTMLElement) {
          const im = document.createElement("img");
          im.src = initial;
          im.alt = "";
          preview.replaceChildren(im);
          preview.hidden = false;
        }
      }
    }
  }

  function syncFtsDisplayPreferences(root) {
    const pairs = [
      { key: "itinerary", checkboxId: "fts-itin" },
      { key: "spends", checkboxId: "fts-spends" },
      { key: "the_tab", checkboxId: "fts-tab" }
    ];
    pairs.forEach(({ key, checkboxId }) => {
      const cb = root.querySelector(`#${checkboxId}`);
      const sel = root.querySelector(`[data-fts-display-select="${key}"]`);
      const lab = root.querySelector(`[data-fts-display-for="${key}"]`);
      if (!(sel instanceof HTMLSelectElement)) return;
      const enabled = cb instanceof HTMLInputElement && cb.checked && !cb.disabled;
      sel.disabled = !enabled;
      if (lab instanceof HTMLElement) {
        lab.classList.toggle("first-trip-setup__field--display-disabled", !enabled);
      }
    });
  }

  function applyTravelMode(root, solo) {
    const pill = root.querySelector("[data-fts-group-pill]");
    const groupCells = root.querySelectorAll("[data-fts-group-only]");
    if (pill) {
      if (solo) pill.setAttribute("hidden", "");
      else pill.removeAttribute("hidden");
    }
    groupCells.forEach((cell) => {
      const cb = cell.querySelector('input[type="checkbox"]');
      if (solo) {
        cell.classList.add("first-trip-setup__toggle-cell--disabled");
        if (cb instanceof HTMLInputElement) {
          cb.checked = false;
          cb.disabled = true;
        }
      } else {
        cell.classList.remove("first-trip-setup__toggle-cell--disabled");
        if (cb instanceof HTMLInputElement) cb.disabled = false;
      }
    });
    syncFtsDisplayPreferences(root);
  }

  function boot() {
    const root = document.querySelector("[data-first-trip-setup]");
    if (!root) return;
    const form = root.querySelector("[data-fts-form]");
    if (!(form instanceof HTMLFormElement)) return;

    document.body.classList.add("first-trip-setup-lock");

    const curSel = root.querySelector("[data-fts-currency-select]");
    const symPost = root.querySelector("[data-fts-currency-symbol-post]");
    const symEdit = root.querySelector("[data-fts-currency-symbol-edit]");
    if (curSel instanceof HTMLSelectElement) {
      syncCurrencyFromTrip(root, curSel);
      curSel.addEventListener("change", () => {
        if (curSel.value === "__OTHER__") {
          if (symPost instanceof HTMLInputElement && symEdit instanceof HTMLInputElement) {
            symEdit.value = symPost.value || symEdit.value;
            symPost.value = symEdit.value;
          }
        }
        syncCurrencyFields(root, curSel);
      });
    }
    if (symEdit instanceof HTMLInputElement && symPost instanceof HTMLInputElement) {
      symEdit.addEventListener("input", () => {
        symPost.value = symEdit.value;
      });
    }

    const steps = Array.from(root.querySelectorAll("[data-fts-step]")).sort(
      (a, b) =>
        Number(a.getAttribute("data-fts-step") || "0") - Number(b.getAttribute("data-fts-step") || "0"),
    );
    const segbar = root.querySelector("[data-fts-segbar]");
    const dots = root.querySelector("[data-fts-dots]");
    // Must be declared before the first rebuildProgressChrome() — that path reads this via maxStepIndex().
    const checklistToggle = root.querySelector("#fts-checklist");
    rebuildProgressChrome();

    wireCoverUI(root);

    let idx = 0;
    const btnCont = root.querySelector("[data-fts-continue]");
    const btnBack = root.querySelector("[data-fts-back]");
    const btnSave = root.querySelector("[data-fts-save]");
    const btnSkip = root.querySelector("[data-fts-skip]");
    const nextHint = root.querySelector("[data-fts-next-hint]");
    const sidebarStep = root.querySelector("[data-fts-sidebar-step]");
    const sidebarTitle = root.querySelector("[data-fts-sidebar-title]");
    const sidebarDesc = root.querySelector("[data-fts-sidebar-desc]");
    const mobileStep = root.querySelector("[data-fts-mobile-step]");
    const progressPct = root.querySelector("[data-fts-progress-pct]");
    const progressFill = root.querySelector("[data-fts-progress-fill]");

    function checklistSectionOn() {
      return checklistToggle instanceof HTMLInputElement && checklistToggle.checked;
    }

    /** Last step index: 4 when Notes & Checklists is on, else 3 (skips global-import step). */
    function maxStepIndex() {
      return checklistSectionOn() ? steps.length - 1 : steps.length - 2;
    }

    function stepCount() {
      return maxStepIndex() + 1;
    }

    function rebuildProgressChrome() {
      const n = stepCount();
      if (segbar) {
        segbar.replaceChildren();
        for (let i = 0; i < n; i++) segbar.appendChild(document.createElement("span"));
      }
      if (dots) {
        dots.replaceChildren();
        for (let i = 0; i < n; i++) dots.appendChild(document.createElement("span"));
      }
    }

    function renderProgress() {
      const n = stepCount();
      const pct = Math.round(((idx + 1) / n) * 100);
      if (progressPct) progressPct.textContent = `${pct}% Complete`;
      if (progressFill instanceof HTMLElement) progressFill.style.width = `${pct}%`;
      const segs = segbar ? Array.from(segbar.children) : [];
      segs.forEach((el, i) => {
        el.classList.toggle("is-on", i <= idx);
      });
      const dchildren = dots ? Array.from(dots.children) : [];
      dchildren.forEach((el, i) => {
        el.classList.toggle("is-active", i === idx);
      });
      const m = STEP_META[idx] || STEP_META[0];
      if (sidebarStep) sidebarStep.textContent = `STEP ${m.k}`;
      if (sidebarTitle) sidebarTitle.textContent = m.sidebarTitle;
      if (sidebarDesc) sidebarDesc.textContent = m.sidebarDesc;
      if (mobileStep) mobileStep.textContent = m.step;
      if (nextHint) nextHint.textContent = idx < maxStepIndex() ? m.nextHint : "";
      if (btnCont instanceof HTMLElement && btnSave instanceof HTMLElement) {
        const last = idx >= maxStepIndex();
        btnCont.hidden = last;
        btnSave.hidden = !last;
      }
      if (btnBack instanceof HTMLElement) btnBack.hidden = idx === 0;
    }

    function showStep(i) {
      const cap = maxStepIndex();
      if (i > cap) i = cap;
      if (i < 0) i = 0;
      idx = i;
      steps.forEach((fs, j) => {
        const on = j === i;
        fs.classList.toggle("first-trip-setup__step--hidden", !on);
        fs.hidden = !on;
      });
      renderProgress();
    }

    const localsStepIndex = 2;
    form.addEventListener("submit", (e) => {
      const sel = root.querySelector("[data-fts-currency-select]");
      if (!(sel instanceof HTMLSelectElement) || sel.value !== "__OTHER__") return;
      const cust = root.querySelector("[data-fts-currency-custom]");
      if (cust instanceof HTMLInputElement && !cust.value.trim()) {
        e.preventDefault();
        showStep(localsStepIndex);
        cust.focus();
        cust.reportValidity();
      }
    });

    function travelSolo() {
      const g = root.querySelector('input[name="trip_travel_mode"][value="group"]');
      return !(g instanceof HTMLInputElement && g.checked);
    }

    root.querySelectorAll('input[name="trip_travel_mode"]').forEach((r) => {
      r.addEventListener("change", () => applyTravelMode(root, travelSolo()));
    });
    applyTravelMode(root, travelSolo());

    form.addEventListener("change", (e) => {
      const t = e.target;
      if (
        t instanceof HTMLInputElement &&
        t.type === "checkbox" &&
        (t.name || "").startsWith("section_")
      ) {
        syncFtsDisplayPreferences(root);
      }
      if (t instanceof HTMLInputElement && t.id === "fts-checklist") {
        rebuildProgressChrome();
        if (!checklistSectionOn() && idx >= steps.length - 1) {
          showStep(steps.length - 2);
        } else {
          renderProgress();
        }
      }
    });

    showStep(0);

    function onEscapeDismiss(e) {
      if (e.key !== "Escape") return;
      e.preventDefault();
      dismissFirstTripSetup();
    }

    function dismissFirstTripSetup() {
      if (root.classList.contains("first-trip-setup--dismissed")) return;
      root.classList.add("first-trip-setup--dismissed");
      document.body.classList.remove("first-trip-setup-lock");
      document.removeEventListener("keydown", onEscapeDismiss, true);
    }

    document.addEventListener("keydown", onEscapeDismiss, true);

    btnCont?.addEventListener("click", () => {
      if (idx === 0) {
        const name = form.querySelector('input[name="name"]');
        if (name instanceof HTMLInputElement && !name.value.trim()) {
          name.reportValidity();
          return;
        }
      }
      if (idx < maxStepIndex()) showStep(idx + 1);
    });
    btnBack?.addEventListener("click", () => {
      if (idx > 0) showStep(idx - 1);
    });
    btnSkip?.addEventListener("click", () => {
      dismissFirstTripSetup();
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();
