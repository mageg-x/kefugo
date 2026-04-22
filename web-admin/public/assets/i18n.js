(function () {
  var STORAGE_KEY = "kefu_locale";
  var DEFAULT_LOCALE = "zh-CN";
  var SUPPORTED = { "zh-CN": true, "en-US": true };

  function normalizeLocale(raw) {
    var value = String(raw || "").trim();
    if (SUPPORTED[value]) return value;
    if (/^en/i.test(value)) return "en-US";
    if (/^zh/i.test(value)) return "zh-CN";
    return DEFAULT_LOCALE;
  }

  function getQueryLocale() {
    try {
      var params = new URLSearchParams(window.location.search);
      return params.get("lang") || params.get("locale") || "";
    } catch (_err) {
      return "";
    }
  }

  function getStoredLocale() {
    try {
      return localStorage.getItem(STORAGE_KEY) || "";
    } catch (_err) {
      return "";
    }
  }

  function setStoredLocale(locale) {
    try {
      localStorage.setItem(STORAGE_KEY, locale);
    } catch (_err) {
      // noop
    }
  }

  function flattenLocaleEntries(tree, prefix, output) {
    if (!tree || typeof tree !== "object" || Array.isArray(tree)) return output;
    Object.keys(tree).forEach(function (key) {
      var value = tree[key];
      var nextKey = prefix ? prefix + "." + key : key;
      if (typeof value === "string") {
        output[nextKey] = value;
      } else if (value && typeof value === "object" && !Array.isArray(value)) {
        flattenLocaleEntries(value, nextKey, output);
      }
    });
    return output;
  }

  function hasChinese(value) {
    return /[\u3400-\u9FFF]/.test(String(value || ""));
  }

  function getLocaleBaseURL() {
    try {
      if (document.currentScript && document.currentScript.src) {
        return new URL("./locales/", document.currentScript.src).toString();
      }
    } catch (_err) {
      // noop
    }
    return "./locales/";
  }

  var locale = normalizeLocale(getQueryLocale() || getStoredLocale() || DEFAULT_LOCALE);
  setStoredLocale(locale);
  document.documentElement.setAttribute("lang", locale);
  window.__kefuLocale = locale;
  window.__kefuSetLocale = function (next) {
    var normalized = normalizeLocale(next);
    setStoredLocale(normalized);
    if (normalized !== locale) {
      window.location.reload();
    }
  };

  async function fetchJson(url) {
    try {
      var resp = await fetch(url, { credentials: "same-origin" });
      if (!resp.ok) return null;
      return await resp.json();
    } catch (_err) {
      return null;
    }
  }

  function translateTextNodes(map) {
    var walker = document.createTreeWalker(document.body || document.documentElement, NodeFilter.SHOW_TEXT);
    var node;
    while ((node = walker.nextNode())) {
      if (!node || !node.nodeValue) continue;
      var raw = String(node.nodeValue);
      var trimmed = raw.trim();
      if (!trimmed || !hasChinese(trimmed)) continue;
      var translated = map.get(trimmed);
      if (!translated) continue;
      node.nodeValue = raw.replace(trimmed, translated);
    }
  }

  function translateAttributes(map) {
    var attrs = ["title", "placeholder", "aria-label", "alt"];
    var all = document.querySelectorAll("*");
    all.forEach(function (el) {
      attrs.forEach(function (attr) {
        if (!el.hasAttribute(attr)) return;
        var raw = String(el.getAttribute(attr) || "");
        var trimmed = raw.trim();
        if (!trimmed || !hasChinese(trimmed)) return;
        var translated = map.get(trimmed);
        if (!translated) return;
        el.setAttribute(attr, raw.replace(trimmed, translated));
      });
    });
  }

  async function applyStaticI18n() {
    if (locale !== "en-US") return;
    var baseURL = getLocaleBaseURL();
    var zh = await fetchJson(baseURL + "zh-CN.json");
    var en = await fetchJson(baseURL + "en-US.json");
    if (!zh || !en) return;

    var zhTree = Object.assign({}, zh);
    var enTree = Object.assign({}, en);
    delete zhTree.name;
    delete zhTree.languageOptions;
    delete enTree.name;
    delete enTree.languageOptions;

    var zhFlat = flattenLocaleEntries(zhTree, "", {});
    var enFlat = flattenLocaleEntries(enTree, "", {});
    var map = new Map();

    Object.keys(zhFlat).forEach(function (key) {
      var zhText = zhFlat[key];
      var enText = enFlat[key];
      if (typeof zhText === "string" && typeof enText === "string" && zhText && enText) {
        map.set(zhText, enText);
      }
    });

    translateTextNodes(map);
    translateAttributes(map);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", applyStaticI18n, { once: true });
  } else {
    applyStaticI18n();
  }
})();
