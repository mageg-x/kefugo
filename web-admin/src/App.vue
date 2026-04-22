<template>
  <div id="app" ref="appRef">
    <router-view />
  </div>
</template>

<script setup>

</script>

<style>
/* Global styles can be added here */
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
  margin: 0;
  padding: 0;
  background-color: #f3f4f6;
}


/* 全局样式（main.css 或 App.vue） */
:root {
  /* 字体大小从 14px 到 18px，随视口平滑变化 */
  font-size: clamp(14px, 1.2vw, 24px);
}

#app {
  /* 内容区最大宽度随屏幕增大，但不超过 1920px */
  max-width: 1920px;
}
</style>