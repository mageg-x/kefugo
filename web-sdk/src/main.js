import { createApp } from "vue";
import TDesign from "tdesign-vue-next";
import "tdesign-vue-next/es/style/index.css";
import Chat from "@tdesign-vue-next/chat";
import "@tdesign-vue-next/chat/es/style/index.css";
import "./style.css";
import Widget from "./components/Widget.vue";

export function mountKefuWidget(props) {
  const div = document.createElement("div");
  div.id = "kefu-widget-root";
  div.style.position = "fixed";
  div.style.bottom = "5px";
  div.style.right = "5px";
  div.style.zIndex = "50";
  div.style.opacity = "0";
  div.style.transition = "opacity 0.3s ease";
  document.body.appendChild(div);

  const app = createApp(Widget, props);
  app.use(TDesign);
  app.use(Chat);
  app.mount("#kefu-widget-root");

  setTimeout(() => {
    div.style.opacity = "1";
  }, 0);

  return app;
}
