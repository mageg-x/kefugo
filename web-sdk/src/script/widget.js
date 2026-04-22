import { mountKefuWidget } from "../main.js";
import { initRuntimeI18n, setLocale } from "./i18n.js";

initRuntimeI18n();

function loadCSS() {
  return new Promise((resolve, reject) => {
    const existingLink = document.getElementById("kefu-sdk-styles");
    if (existingLink) {
      resolve();
      return;
    }

    const script =
      document.currentScript ||
      document.querySelector("script[data-kefu-appid]");
    if (!script) {
      console.error("Cannot find script tag with data-kefu-appid attribute");
      resolve();
      return;
    }

    const scriptSrc = script.src;
    const cssUrl = scriptSrc.replace("widget.min.js", "style.css");

    const link = document.createElement("link");
    link.id = "kefu-sdk-styles";
    link.rel = "stylesheet";
    link.href = cssUrl;

    link.onload = () => resolve();
    link.onerror = () => {
      console.error("Failed to load KefuChat stylesheet");
      resolve();
    };

    document.head.appendChild(link);
  });
}

window.KefuChat = {
  init: async (options = {}) => {
    if (options?.locale) {
      setLocale(options.locale);
    }
    await loadCSS();
    return mountKefuWidget(options);
  },
};

const scriptTags = document.querySelectorAll("script");
let appId = null;
let apiBaseUrl = null;
let wsUrl = null;
let userId = null;

for (const script of scriptTags) {
  if (script.hasAttribute("data-kefu-appid")) {
    appId = script.getAttribute("data-kefu-appid");
    apiBaseUrl = script.getAttribute("data-kefu-api-base-url");
    wsUrl = script.getAttribute("data-kefu-ws-url");
    userId = script.getAttribute("data-kefu-user-id");
    break;
  }
}

if (appId) {
  window.KefuChat.init({
    appId,
    ...(apiBaseUrl ? { apiBaseUrl } : {}),
    ...(wsUrl ? { wsUrl } : {}),
    ...(userId ? { userId } : {}),
  });
}
