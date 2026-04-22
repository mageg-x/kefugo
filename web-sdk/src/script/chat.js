import { createApp } from "vue";
import "../style.css";
import Chat from "../components/Chat.vue";
import { initRuntimeI18n } from "./i18n.js";

initRuntimeI18n();
createApp(Chat).mount("#app");
