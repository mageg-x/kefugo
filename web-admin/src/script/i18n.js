import { ref } from "vue";
import zhCN from "../locales/zh-CN.json";
import enUS from "../locales/en-US.json";

const STORAGE_KEY = "kefu_locale";
const DEFAULT_LOCALE = "zh-CN";
const SUPPORTED = new Set(["zh-CN", "en-US"]);

const LOCALES = {
  "zh-CN": zhCN,
  "en-US": enUS,
};

function flattenLocaleEntries(tree, prefix = "", output = {}) {
  if (!tree || typeof tree !== "object" || Array.isArray(tree)) {
    return output;
  }
  for (const [key, value] of Object.entries(tree)) {
    const nextKey = prefix ? `${prefix}.${key}` : key;
    if (typeof value === "string") {
      output[nextKey] = value;
    } else if (value && typeof value === "object" && !Array.isArray(value)) {
      flattenLocaleEntries(value, nextKey, output);
    }
  }
  return output;
}

function getLocaleSections(localeObj) {
  if (!localeObj || typeof localeObj !== "object") {
    return {};
  }
  const tree = { ...localeObj };
  delete tree.name;
  delete tree.languageOptions;
  return tree;
}

const zhMessages = flattenLocaleEntries(getLocaleSections(LOCALES["zh-CN"]));
const enMessages = flattenLocaleEntries(getLocaleSections(LOCALES["en-US"]));
const EN_DICT = new Map();
for (const [key, zhText] of Object.entries(zhMessages)) {
  const enText = enMessages[key];
  if (typeof zhText === "string" && typeof enText === "string" && zhText && enText) {
    EN_DICT.set(zhText, enText);
  }
}

const languageOptions = LOCALES["zh-CN"]?.languageOptions || {
  "zh-CN": "中文",
  "en-US": "English",
};

export const localeRef = ref(DEFAULT_LOCALE);
export const localeOptions = [
  { value: "zh-CN", label: languageOptions["zh-CN"] || "中文" },
  { value: "en-US", label: languageOptions["en-US"] || "English" },
];

const textOriginalMap = new Map();
const attrOriginalMap = new Map();
let observer = null;
let applying = false;

function hasChinese(value) {
  return /[\u3400-\u9FFF]/.test(String(value || ""));
}

function normalizeLocale(raw) {
  const value = String(raw || "").trim();
  if (SUPPORTED.has(value)) {
    return value;
  }
  if (/^en/i.test(value)) {
    return "en-US";
  }
  if (/^zh/i.test(value)) {
    return "zh-CN";
  }
  return DEFAULT_LOCALE;
}

function queryLocale() {
  try {
    const params = new URLSearchParams(window.location.search);
    const fromQuery = params.get("lang") || params.get("locale");
    return fromQuery ? normalizeLocale(fromQuery) : "";
  } catch {
    return "";
  }
}

function getStoredLocale() {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    return value ? normalizeLocale(value) : "";
  } catch {
    return "";
  }
}

function setStoredLocale(locale) {
  try {
    localStorage.setItem(STORAGE_KEY, locale);
  } catch {
    // noop
  }
}

function shouldSkipElement(el) {
  if (!el || el.nodeType !== Node.ELEMENT_NODE) {
    return true;
  }
  if (el.classList?.contains("i18n-ignore")) {
    return true;
  }
  const tag = String(el.tagName || "").toUpperCase();
  return ["SCRIPT", "STYLE", "NOSCRIPT", "TEXTAREA"].includes(tag);
}

function translateZhToEn(raw) {
  const input = String(raw || "");
  if (!input) {
    return input;
  }
  if (!hasChinese(input)) {
    return input;
  }
  if (EN_DICT.has(input)) {
    return EN_DICT.get(input);
  }
  return input;
}

function translateByLocale(raw, locale) {
  if (locale === "en-US") {
    return translateZhToEn(raw);
  }
  return String(raw || "");
}

function convertTextNode(node, forceOriginal = false) {
  if (!node || node.nodeType !== Node.TEXT_NODE) {
    return;
  }
  const current = String(node.nodeValue || "");
  if (!hasChinese(current)) {
    return;
  }
  if (forceOriginal || !textOriginalMap.has(node)) {
    textOriginalMap.set(node, current);
  }
  const source = String(textOriginalMap.get(node) || current);
  const converted = translateByLocale(source, localeRef.value);
  if (converted !== current) {
    applying = true;
    node.nodeValue = converted;
    applying = false;
  }
}

function rememberAttr(el, key, value, forceOriginal) {
  let inner = attrOriginalMap.get(el);
  if (!inner) {
    inner = new Map();
    attrOriginalMap.set(el, inner);
  }
  if (forceOriginal || !inner.has(key)) {
    inner.set(key, value);
  }
  return inner.get(key);
}

function convertElementAttrs(el, forceOriginal = false) {
  if (!el || el.nodeType !== Node.ELEMENT_NODE || shouldSkipElement(el)) {
    return;
  }

  const attrs = ["title", "placeholder", "aria-label", "alt"];
  for (const attr of attrs) {
    if (!el.hasAttribute(attr)) {
      continue;
    }
    const raw = String(el.getAttribute(attr) || "");
    if (!hasChinese(raw)) {
      continue;
    }
    const source = rememberAttr(el, attr, raw, forceOriginal);
    const converted = translateByLocale(source || raw, localeRef.value);
    if (converted !== raw) {
      applying = true;
      el.setAttribute(attr, converted);
      applying = false;
    }
  }

  const tag = String(el.tagName || "").toUpperCase();
  if (tag === "META" && el.hasAttribute("content")) {
    const name = String(el.getAttribute("name") || "").toLowerCase();
    if (name === "description" || name === "keywords") {
      const raw = String(el.getAttribute("content") || "");
      if (hasChinese(raw)) {
        const source = rememberAttr(el, "content", raw, forceOriginal);
        const converted = translateByLocale(source || raw, localeRef.value);
        if (converted !== raw) {
          applying = true;
          el.setAttribute("content", converted);
          applying = false;
        }
      }
    }
  }

  if (tag === "INPUT") {
    const type = String(el.getAttribute("type") || "text").toLowerCase();
    if (["button", "submit", "reset"].includes(type)) {
      const raw = String(el.value || "");
      if (hasChinese(raw)) {
        const source = rememberAttr(el, "__value__", raw, forceOriginal);
        const converted = translateByLocale(source || raw, localeRef.value);
        if (converted !== raw) {
          applying = true;
          el.value = converted;
          applying = false;
        }
      }
    }
  }
}

function convertTree(node, forceOriginal = false) {
  if (!node) {
    return;
  }
  if (node.nodeType === Node.TEXT_NODE) {
    convertTextNode(node, forceOriginal);
    return;
  }
  if (node.nodeType !== Node.ELEMENT_NODE && node.nodeType !== Node.DOCUMENT_NODE) {
    return;
  }
  const element = node.nodeType === Node.ELEMENT_NODE ? node : null;
  if (element && shouldSkipElement(element)) {
    return;
  }
  if (element) {
    convertElementAttrs(element, forceOriginal);
  }
  const children = Array.from(node.childNodes || []);
  for (const child of children) {
    convertTree(child, forceOriginal);
  }
}

function restoreAll() {
  applying = true;
  for (const [node, original] of textOriginalMap.entries()) {
    if (node?.isConnected) {
      node.nodeValue = original;
    }
  }
  for (const [el, attrs] of attrOriginalMap.entries()) {
    if (!el?.isConnected) {
      continue;
    }
    for (const [name, original] of attrs.entries()) {
      if (name === "__value__") {
        el.value = original;
      } else {
        el.setAttribute(name, original);
      }
    }
  }
  textOriginalMap.clear();
  attrOriginalMap.clear();
  applying = false;
}

function stopObserver() {
  if (observer) {
    observer.disconnect();
    observer = null;
  }
}

function startObserver() {
  stopObserver();
  observer = new MutationObserver((mutations) => {
    if (applying || localeRef.value !== "en-US") {
      return;
    }
    for (const mutation of mutations) {
      if (mutation.type === "characterData") {
        convertTextNode(mutation.target, true);
      } else if (mutation.type === "attributes") {
        convertElementAttrs(mutation.target, true);
      } else if (mutation.type === "childList") {
        for (const node of mutation.addedNodes) {
          convertTree(node, true);
        }
      }
    }
  });
  observer.observe(document.documentElement, {
    subtree: true,
    childList: true,
    characterData: true,
    attributes: true,
    attributeFilter: ["title", "placeholder", "aria-label", "alt", "content", "value"],
  });
}

function applyLocale() {
  if (typeof document === "undefined") {
    return;
  }
  document.documentElement.setAttribute("lang", localeRef.value);
  if (localeRef.value === "en-US") {
    convertTree(document.documentElement, false);
    startObserver();
  } else {
    stopObserver();
    restoreAll();
  }
}

export function setLocale(locale, options = {}) {
  const normalized = normalizeLocale(locale);
  if (localeRef.value === normalized) {
    return;
  }
  localeRef.value = normalized;
  if (options.persist !== false) {
    setStoredLocale(normalized);
  }
  applyLocale();
}

export function initRuntimeI18n() {
  const locale = queryLocale() || getStoredLocale() || DEFAULT_LOCALE;
  localeRef.value = normalizeLocale(locale);
  setStoredLocale(localeRef.value);
  if (typeof document !== "undefined") {
    applyLocale();
  }
}
