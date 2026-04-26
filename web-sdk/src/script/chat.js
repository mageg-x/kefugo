import { createApp } from "vue";
import TDesign from "tdesign-vue-next";
import "tdesign-vue-next/es/style/index.css";
import Chat from "@tdesign-vue-next/chat";
import "@tdesign-vue-next/chat/es/style/index.css";
import "../style.css";
import ChatPage from "../components/Chat.vue";
import { initRuntimeI18n } from "./i18n.js";

initRuntimeI18n();
const app = createApp(ChatPage);
app.use(TDesign);
app.use(Chat);
app.mount("#app");
