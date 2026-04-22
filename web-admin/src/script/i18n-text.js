import zhCN from "@/locales/zh-CN.json";
import enUS from "@/locales/en-US.json";
import { localeRef } from "@/script/i18n";

const LOCALES = {
  "zh-CN": zhCN,
  "en-US": enUS,
};

function getByPath(obj, key) {
  return String(key || "")
    .split(".")
    .reduce((acc, part) => (acc && typeof acc === "object" ? acc[part] : undefined), obj);
}

export function t(key, fallback = "") {
  const currentLocale = LOCALES[localeRef.value] || LOCALES["zh-CN"];
  const zhLocale = LOCALES["zh-CN"];
  return getByPath(currentLocale, key) ?? getByPath(zhLocale, key) ?? (fallback || key);
}
