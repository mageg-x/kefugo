<template>
  <div class="kefu-liao-chat-core">
    <div v-if="typingText" class="kefu-typing-tip">{{ typingText }}</div>
    <div class="kefu-top-actions">
      <button v-if="sessionId" type="button" class="kefu-rate-entry" :disabled="disabled" @click="openRateDialog">
        <Star :size="13" />
        <span>{{ t('pageInbox.action.rate', '评分') }}</span>
      </button>
    </div>

    <div ref="messageListRef" class="kefu-liao-list">
      <template v-for="msg in messages" :key="msg.id">
        <t-chat-message v-if="msg.type === 'system'" role="system"
          :content="[{ type: 'text', data: typeof msg.content === 'string' ? msg.content : '' }]"
          :datetime="formatMessageTime(msg.time)" variant="outline" />
        <t-chat-message v-else :role="msg.isSelf ? 'user' : 'assistant'" :avatar="resolveMessageAvatar(msg)"
          :name="resolveMessageName(msg)" :datetime="formatMessageTime(msg.time)"
          :variant="msg.isSelf ? 'base' : 'outline'" :status="toTDesignStatus(msg.status)"
          :content="CUSTOM_MESSAGE_SEGMENTS" :allow-content-segment-custom="true">
          <template #content>
            <div class="kefu-rich-content" :class="{ 'is-self': msg.isSelf }">
              <div v-if="msg.replyTo" class="kefu-reply-quote" :class="{ 'is-self': msg.isSelf }">
                <p class="kefu-reply-quote-head">{{ msg.replyTo.sender || t('reply.message', '消息') }}</p>
                <p class="kefu-reply-quote-preview">{{ replyPreviewText(msg.replyTo) }}</p>
              </div>

              <div v-if="msg.type === 'text'" class="kefu-msg-bubble" :class="{ 'is-self': msg.isSelf }">
                <div class="kefu-markdown-content" v-html="renderMarkdownContent(msg.content)"></div>
              </div>

              <button v-else-if="msg.type === 'image'" type="button" class="kefu-inline-image-wrap"
                :class="{ 'is-self': msg.isSelf }" @click="openImage(msg.content || msg.fileUrl || msg.url)">
                <img class="kefu-inline-image" :src="msg.content || msg.fileUrl || msg.url"
                  :alt="msg.fileName || t('message.image', '图片')" />
              </button>

              <div v-else-if="msg.type === 'audio'" class="kefu-audio-bubble" :class="{ 'is-self': msg.isSelf }">
                <AudioLines :size="16" class="kefu-audio-symbol" />
                <button class="kefu-audio-icon-btn" type="button"
                  :title="playingMessageId === msg.id ? t('audio.pause', '暂停') : t('audio.play', '播放')"
                  :aria-label="playingMessageId === msg.id ? t('audio.pause', '暂停') : t('audio.play', '播放')"
                  @click="toggleAudioPlay(msg)">
                  <CirclePause v-if="playingMessageId === msg.id" :size="18" />
                  <CirclePlay v-else :size="18" />
                </button>
                <button class="kefu-audio-icon-btn" type="button" :title="t('audio.stop', '停止')"
                  :aria-label="t('audio.stop', '停止')" :disabled="playingMessageId !== msg.id" @click="stopAudio(msg.id)">
                  <CircleStop :size="18" />
                </button>
                <span class="kefu-audio-duration">{{ formatDuration(msg.duration) }}</span>
                <audio class="kefu-hidden-audio" :ref="(el) => bindAudioElement(msg.id, el)"
                  :src="msg.audioUrl || msg.content" preload="metadata" @ended="handleAudioEnded(msg.id)"></audio>
              </div>

              <div v-else-if="msg.type === 'file'" class="kefu-file-bubble">
                <div class="kefu-file-info">
                  <span class="kefu-file-icon" :style="{ color: getFileIconColor(msg) }">
                    <component :is="getFileIconComponent(msg)" :size="28" />
                  </span>
                  <div class="kefu-file-detail">
                    <span class="kefu-file-name">{{ msg.fileName || msg.content || t('message.file', '文件') }}</span>
                    <span v-if="msg.fileSize" class="kefu-file-size">{{ formatFileSize(msg.fileSize) }}</span>
                  </div>
                </div>
                <a v-if="msg.fileUrl || msg.url" class="kefu-file-download" :href="msg.fileUrl || msg.url" target="_blank"
                  rel="noopener noreferrer" :title="t('action.download', '下载')" :aria-label="t('action.download', '下载')">
                  <Download :size="18" />
                </a>
              </div>

              <div v-else class="kefu-msg-bubble" :class="{ 'is-self': msg.isSelf }">
                <div class="kefu-markdown-content" v-html="renderMarkdownContent(msg.content)"></div>
              </div>
            </div>
          </template>

          <template #actionbar>
            <div class="kefu-custom-actionbar">
              <button type="button" class="kefu-action-btn" :title="t('action.copy', '复制')"
                :aria-label="t('action.copy', '复制')" @click="handleChatAction('copy', msg)">
                <Copy :size="14" />
              </button>
              <template v-if="!msg.isSelf">
                <button type="button" class="kefu-action-btn" :class="{ active: msg._isGood }"
                  :title="t('action.like', '点赞')" :aria-label="t('action.like', '点赞')"
                  @click="handleChatAction('good', msg)">
                  <ThumbsUp :size="14" />
                </button>
                <button type="button" class="kefu-action-btn" :class="{ active: msg._isBad }"
                  :title="t('action.dislike', '点踩')" :aria-label="t('action.dislike', '点踩')"
                  @click="handleChatAction('bad', msg)">
                  <ThumbsDown :size="14" />
                </button>
              </template>
              <button type="button" class="kefu-action-btn" :title="t('action.reply', '引用回复')"
                :aria-label="t('action.reply', '引用回复')" @click="handleChatAction('replay', msg)">
                <Reply :size="14" />
              </button>
            </div>
          </template>

          <template #actions>
            <div class="kefu-msg-meta-line" :class="{ 'is-self': msg.isSelf }">
              <template v-if="msg.isSelf">
                <span class="kefu-msg-time">{{ formatMessageTime(msg.time) }}</span>
                <span class="kefu-msg-dot">·</span>
                <span v-if="msg.status === SEND_STATUS.FAILED" class="kefu-msg-status kefu-msg-failed">
                  <AlertCircle :size="12" />
                  {{ t('message.sendFailed', '发送失败') }}
                </span>
                <span v-else class="kefu-msg-status">{{ toStatusText(msg.status) }}</span>
                <button v-if="msg.status === SEND_STATUS.FAILED" type="button" class="kefu-retry-btn"
                  @click.stop="retryMessageById(msg.id)">
                  {{ t('action.retry', '重试') }}
                </button>
              </template>
              <template v-else>
                <span class="kefu-msg-time">{{ formatMessageTime(msg.time) }}</span>
              </template>
            </div>
          </template>
        </t-chat-message>
      </template>
    </div>

    <div v-if="replyTo" class="kefu-reply-bar">
      <div class="kefu-reply-bar-text">
        <p class="kefu-reply-bar-head">{{ t('reply.replyTo', '回复') }} {{ replyTo.sender || t('reply.message', '消息')
        }}</p>
        <p class="kefu-reply-bar-preview">{{ replyPreviewText(replyTo) }}</p>
      </div>
      <button type="button" class="kefu-reply-bar-cancel" @click="cancelReply">×</button>
    </div>

    <t-chat-sender v-model="inputValue" :placeholder="inputPlaceholder || t('input.placeholder', '输入消息...')"
      :disabled="disabled" :loading="isRecording" :send-btn-disabled="disabled" @send="handleSend">
      <template #footer-prefix>
        <div class="kefu-input-toolbar">
          <button type="button" class="kefu-tool-btn" :disabled="disabled" :title="t('message.image', '图片')"
            :aria-label="t('message.image', '图片')" @click="pickImageFiles">
            <span class="kefu-tool-icon">
              <ImageIcon :size="18" />
            </span>
            <span>{{ t('message.image', '图片') }}</span>
          </button>
          <button type="button" class="kefu-tool-btn" :disabled="disabled" :title="t('message.file', '文件')"
            :aria-label="t('message.file', '文件')" @click="pickAnyFiles">
            <span class="kefu-tool-icon">
              <FileText :size="18" />
            </span>
            <span>{{ t('message.file', '文件') }}</span>
          </button>
          <button type="button" class="kefu-tool-btn" :disabled="disabled || !isVoiceSupported"
            :class="{ active: isRecording }"
            :title="isVoiceSupported ? t('voice.holdToRecord', '按住录音，松开发送，移出取消') : t('voice.notSupported', '此浏览器不支持录音')"
            :aria-label="t('pageInbox.action.record', '录音')" @mousedown.prevent="handlePressRecordStart"
            @mouseup.prevent="handlePressRecordStop" @mouseleave.prevent="handlePressRecordCancel"
            @touchstart.prevent="handlePressRecordStart" @touchend.prevent="handlePressRecordStop"
            @touchcancel.prevent="handlePressRecordCancel">
            <span class="kefu-tool-icon" v-if="!isRecording">
              <Mic :size="18" />
            </span>
            <span class="kefu-tool-icon kefu-recording-pulse" v-else>
              <MicOff :size="18" />
            </span>
            <span>{{ isRecording ? t('action.send', '发送') : t('pageInbox.action.record', '录音') }}</span>
          </button>
          <button type="button" class="kefu-tool-btn" :disabled="disabled" :title="t('pageInbox.action.emoji', '表情')"
            :aria-label="t('pageInbox.action.emoji', '表情')" @click="toggleEmojiPicker">
            <span class="kefu-tool-icon">
              <SmilePlus :size="18" />
            </span>
            <span>{{ t('pageInbox.action.emoji', '表情') }}</span>
          </button>
        </div>
      </template>
    </t-chat-sender>

    <div v-if="showEmojiPicker" class="kefu-emoji-mask" @click.self="toggleEmojiPicker">
      <div class="kefu-emoji-dialog" @click.stop>
        <div class="kefu-emoji-header">
          <h3 class="kefu-emoji-title">{{ t('pageInbox.action.pickEmoji', '选择表情') }}</h3>
          <button type="button" class="kefu-emoji-close" @click="toggleEmojiPicker">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"
              stroke-linecap="round" stroke-linejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>
        <div class="kefu-emoji-tabs">
          <button v-for="(group, idx) in emojiGroups" :key="group.label" type="button" class="kefu-emoji-tab"
            :class="{ active: activeEmojiTab === idx }" @click="activeEmojiTab = idx">
            {{ t('emoji.' + group.label, group.label) }}
          </button>
        </div>
        <div class="kefu-emoji-panel">
          <div class="kefu-emoji-grid">
            <button v-for="emoji in emojiGroups[activeEmojiTab]?.items || []"
              :key="emojiGroups[activeEmojiTab]?.label + emoji" type="button" class="kefu-emoji-btn"
              @click="appendEmoji(emoji)">
              {{ emoji }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <input ref="filePickerRef" type="file" class="kefu-hidden-file" :accept="filePickerAccept" :multiple="true"
      @change="onPickerChanged" />

    <div v-if="rateDialogVisible" class="kefu-rate-dialog-mask" @click.self="closeRateDialog">
      <div class="kefu-rate-dialog">
        <h4>{{ t('pageInbox.action.rate', '评分') }}</h4>
        <div class="kefu-rate-stars">
          <button v-for="star in [1, 2, 3, 4, 5]" :key="star" type="button" class="kefu-rate-star"
            :class="{ active: rateScore >= star }" @click="rateScore = star">
            ★
          </button>
        </div>
        <textarea v-model="rateComment" class="kefu-rate-input" maxlength="200"
          :placeholder="t('rate.commentPlaceholder', '可选：留下评论')"></textarea>
        <div class="kefu-rate-actions">
          <button type="button" class="kefu-rate-cancel" @click="closeRateDialog">{{ t('action.cancel', '取消')
            }}</button>
          <button type="button" class="kefu-rate-submit" :disabled="rateSaving || rateScore < 1" @click="submitRate">
            {{ rateSaving ? t('action.submitting', '提交中...') : t('action.submit', '提交') }}
          </button>
        </div>
      </div>
    </div>
    <div v-if="imagePreviewVisible" class="kefu-image-preview-mask" @click.self="closeImagePreview">
      <img class="kefu-image-preview" :src="imagePreviewUrl" :alt="t('pageInbox.dialog.imagePreview', '图片预览')" />
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import {
  Mic,
  AlertCircle,
  Image as ImageIcon,
  FileText,
  MicOff,
  SmilePlus,
  AudioLines,
  CirclePlay,
  CirclePause,
  CircleStop,
  Star,
  Download,
  File,
  FileSpreadsheet,
  FileImage,
  FileCode,
  Music,
  Video,
  FileArchive,
  FileQuestion,
  Copy,
  ThumbsUp,
  ThumbsDown,
  Reply,
} from "lucide-vue-next";
import DOMPurify from "dompurify";
import { marked } from "marked";
import api from "../script/api.js";
import { isDomainForbiddenCode } from "../script/error-codes.js";
import { WSClient } from "../script/wscli.js";
import { t } from "../script/i18n.js";
import {
  BUSINESS_CONTENT_TYPES,
  BUSINESS_MESSAGE_TYPES,
  buildChatUiMessageFromBusiness,
  buildChatUiMessageFromOutgoingPayload,
  buildOutgoingBusinessPayload,
  normalizeIncomingBusinessMessage,
} from "../script/message-protocol.js";
import {
  buildWsUrlFromApiBase,
  createLocalMessageId,
  getOrCreateVisitorId,
  normalizeAppUserId,
  normalizeSdkVisitorId,
} from "../script/chat-runtime.js";

const SEND_STATUS = {
  SENDING: "sending",
  SENT: "sent",
  FAILED: "failed",
  READ: "read",
};

const props = defineProps({
  appId: { type: String, required: true },
  userId: { type: String, default: "" },
  serviceName: { type: String, default: "" },
  serviceAvatar: { type: String, default: "" },
  apiBaseUrl: {
    type: String,
    default: () =>
      typeof window !== "undefined" && window.location?.origin
        ? window.location.origin
        : "http://localhost:5300",
  },
  wsUrl: { type: String, default: "" },
  emptyText: { type: String, default: "" },
  inputPlaceholder: { type: String, default: "" },
  disabled: { type: Boolean, default: false },
});

const emit = defineEmits(["config-loaded", "config-error", "ws-status", "ws-error", "unread-change", "feedback"]);

const visitorId = ref(normalizeAppUserId(props.userId) || getOrCreateVisitorId());
const inputValue = ref("");
const messages = ref([]);
const config = ref(null);
const wsClient = ref(null);
const messageListRef = ref(null);
const sessionId = ref("");
const typingText = ref("");
const activeAgentName = ref("");
const activeAgentAvatar = ref("");
const typingTimer = ref(null);
let lastTypingAt = 0;

const mediaRecorder = ref(null);
const recordingStream = ref(null);
const audioChunks = ref([]);
const recordingStartedAt = ref(0);
const isRecording = ref(false);

const playingMessageId = ref("");
const audioElementMap = new Map();
const settleTimers = new Map();
let wsConnectTimeoutTimer = null;
const replyTo = ref(null);
const rateDialogVisible = ref(false);
const rateScore = ref(5);
const rateComment = ref("");
const rateSaving = ref(false);
const imagePreviewVisible = ref(false);
const imagePreviewUrl = ref("");

const receivedMsgIds = new Set();
const unreadCount = ref(0);

const showEmojiPicker = ref(false);
const activeEmojiTab = ref(0);
const CUSTOM_MESSAGE_SEGMENTS = Object.freeze([{ type: "custom", data: " " }]);
const MARKDOWN_SANITIZE_OPTIONS = {
  USE_PROFILES: { html: true },
  ALLOWED_URI_REGEXP: /^(?:(?:https?|mailto|tel):|\/|#)/i,
};
const emojiGroups = [
  { label: "Common", items: ["😀", "😁", "😂", "🤣", "😊", "🙂", "😉", "😍", "🥰", "😘", "😎", "🤩", "😴", "😭", "😡", "🤔"] },
  { label: "Gestures", items: ["👍", "👎", "👌", "✌️", "🤝", "👏", "🙏", "💪", "👀", "🎉", "🔥", "✨"] },
  { label: "Symbols", items: ["❤️", "💙", "💚", "💛", "💜", "🧡", "💯", "✅", "❌", "⚠️", "⭐", "🌈"] },
];
const isPressRecording = ref(false);
const filePickerRef = ref(null);
const filePickerAccept = ref("*/*");

const accept = computed(() => "image/*,audio/*,.pdf,.doc,.docx,.xls,.xlsx,.txt,.zip,.rar");
const wsEndpoint = computed(() => props.wsUrl || buildWsUrlFromApiBase(props.apiBaseUrl));
const resolvedServiceName = computed(() => activeAgentName.value || config.value?.name || props.serviceName);
const resolvedServiceAvatar = computed(
  () => activeAgentAvatar.value || config.value?.logo || props.serviceAvatar || avatarBySeed(resolvedServiceName.value || "agent")
);
const selfAvatar = computed(() => avatarBySeed(visitorId.value || "visitor"));
const isVoiceSupported = computed(() => {
  if (typeof window === "undefined") {
    return false;
  }
  return Boolean(window.MediaRecorder && navigator?.mediaDevices?.getUserMedia);
});

function avatarBySeed(seed) {
  return `https://api.dicebear.com/9.x/adventurer/svg?seed=${encodeURIComponent(String(seed || "agent"))}`;
}

function extractAgentNameFromSystemText(text) {
  const normalized = String(text || "").trim();
  if (!normalized) {
    return "";
  }
  const matched = normalized.match(/(.+?)\s*\u4e3a\u60a8\u670d\u52a1[。.!！]?/);
  if (!matched || !matched[1]) {
    return "";
  }
  const raw = String(matched[1]).trim();
  if (!raw) {
    return "";
  }
  const parts = raw.split(/[，,:：]/).map((item) => String(item || "").trim()).filter(Boolean);
  if (parts.length > 0) {
    return parts[parts.length - 1];
  }
  return raw;
}

function refreshAgentMetaInMessages() {
  const serviceName = resolvedServiceName.value;
  const serviceAvatar = resolvedServiceAvatar.value;
  const myAvatar = selfAvatar.value;
  messages.value = messages.value.map((item) => {
    if (!item || item.type === "system") {
      return item;
    }
    if (item.isSelf) {
      return {
        ...item,
        name: "",
        avatar: myAvatar,
      };
    }
    const metaRaw = item?.business?.payload?.raw || {};
    const serverNamed = String(
      metaRaw.agent_name || metaRaw.from_name || metaRaw.sender_name || ""
    ).trim();
    const displayName = serverNamed || serviceName;
    return {
      ...item,
      name: displayName,
      avatar: serviceAvatar,
    };
  });
}

function applyAgentServiceFromSystemText(text) {
  const name = extractAgentNameFromSystemText(text);
  if (!name) {
    return;
  }
  activeAgentName.value = name;
  activeAgentAvatar.value = avatarBySeed(name);
  refreshAgentMetaInMessages();
}

function pushMessage(message) {
  messages.value.push(message);
  void nextTick(() => {
    scrollToBottom();
  });
}

function scrollToBottom() {
  const el = messageListRef.value;
  if (el) {
    el.scrollTop = el.scrollHeight;
  }
}

function pushSystemTip(content) {
  const text = String(content || "").trim();
  if (!text) {
    return;
  }
  pushMessage({
    id: createLocalMessageId("sys"),
    type: "system",
    isSelf: false,
    name: t('word.system', '系统'),
    content: text,
    time: new Date().toISOString(),
    contentType: BUSINESS_CONTENT_TYPES.TEXT,
  });
}

function syncUnreadCount(nextValue) {
  const normalized = Math.max(0, Number(nextValue || 0));
  if (normalized === unreadCount.value) {
    return;
  }
  unreadCount.value = normalized;
  emit("unread-change", normalized);
}

function updateMessageById(messageId, updater) {
  const idx = messages.value.findIndex((m) => m.id === messageId);
  if (idx < 0) {
    return;
  }
  const next = updater({ ...messages.value[idx] });
  messages.value[idx] = next || messages.value[idx];
}

function clearSettleTimer(messageId) {
  const timer = settleTimers.get(messageId);
  if (timer) {
    clearTimeout(timer);
    settleTimers.delete(messageId);
  }
}

function scheduleAutoSent(messageId, timeout = 500) {
  clearSettleTimer(messageId);
  const timer = setTimeout(() => {
    updateMessageById(messageId, (msg) => {
      if (msg.status !== SEND_STATUS.SENDING) {
        return msg;
      }
      return { ...msg, status: SEND_STATUS.SENT };
    });
    settleTimers.delete(messageId);
  }, timeout);
  settleTimers.set(messageId, timer);
}

function toOutboundUiMessage(contentType, data, localId = "") {
  const payload = buildOutgoingBusinessPayload(contentType, data);
  return buildChatUiMessageFromOutgoingPayload(payload, {
    id: localId || createLocalMessageId("ui"),
    selfName: "",
    selfAvatar: selfAvatar.value,
    time: Date.now(),
  });
}

function replyPreviewText(target) {
  if (!target) {
    return "";
  }
  const contentType = String(target.contentType || "").toLowerCase();
  if (contentType === BUSINESS_CONTENT_TYPES.IMAGE) {
    return "[Image]";
  }
  if (contentType === BUSINESS_CONTENT_TYPES.AUDIO) {
    return "[Voice]";
  }
  if (contentType === BUSINESS_CONTENT_TYPES.FILE) {
    return "[File]";
  }
  return String(target.preview || "").trim();
}

function buildReplyPayloadFromMessage(message) {
  if (!message) {
    return null;
  }
  const msgId = String(message?._serverMsgId || message?.business?.msgId || "").trim();
  const contentType = String(message?.contentType || message?.type || BUSINESS_CONTENT_TYPES.TEXT).toLowerCase();
  let preview = "";
  if (contentType === BUSINESS_CONTENT_TYPES.TEXT) {
    preview = String(message?.content || "").trim();
  } else if (contentType === BUSINESS_CONTENT_TYPES.IMAGE) {
    preview = "[Image]";
  } else if (contentType === BUSINESS_CONTENT_TYPES.AUDIO) {
    preview = "[Voice]";
  } else {
    preview = String(message?.fileName || "[File]").trim();
  }
  const sender = message?.isSelf ? t('word.me', '我') : String(resolveMessageName(message) || t('word.agent', '客服'));
  return {
    msgId,
    contentType,
    preview: preview.slice(0, 120),
    sender,
    timestamp: Number(new Date(message?.time || Date.now()).getTime()) || Date.now(),
  };
}

function startReply(message) {
  if (!message || message.type === "system") {
    return;
  }
  replyTo.value = buildReplyPayloadFromMessage(message);
}

function getFileExtension(msg) {
  const name = msg.fileName || msg.content || "";
  if (!name.includes(".")) return "";
  return name.split(".").pop()?.toLowerCase() || "";
}

function getFileIconComponent(msg) {
  const ext = getFileExtension(msg);
  if (["xlsx", "xls", "csv"].includes(ext)) return FileSpreadsheet;
  if (["jpg", "jpeg", "png", "gif", "webp", "bmp", "svg"].includes(ext)) return FileImage;
  if (["md", "mdx", "json", "xml", "html", "css", "js", "ts"].includes(ext)) return FileCode;
  if (["mp3", "wav", "flac", "aac", "ogg", "webm"].includes(ext)) return Music;
  if (["mp4", "avi", "mov", "mkv", "wmv", "flv"].includes(ext)) return Video;
  if (["zip", "rar", "7z", "tar", "gz"].includes(ext)) return FileArchive;
  if (["doc", "docx", "pdf", "ppt", "pptx", "txt"].includes(ext)) return FileText;
  return FileQuestion;
}

function getFileIconColor(msg) {
  const ext = getFileExtension(msg);
  if (["xlsx", "xls", "csv"].includes(ext)) return "#2BA471";
  if (["doc", "docx"].includes(ext)) return "#0052D9";
  if (["pdf"].includes(ext)) return "#D54941";
  if (["ppt", "pptx"].includes(ext)) return "#E37318";
  if (["mp3", "wav", "flac", "aac", "ogg"].includes(ext)) return "#D54941";
  if (["mp4", "avi", "mov", "mkv", "wmv"].includes(ext)) return "#D54941";
  if (["zip", "rar", "7z", "tar", "gz"].includes(ext)) return "#E37318";
  if (["jpg", "jpeg", "png", "gif", "webp", "svg"].includes(ext)) return "#8c8c8c";
  if (["md", "mdx", "json", "xml", "html", "css", "js", "ts"].includes(ext)) return "#8c8c8c";
  return "#8c8c8c";
}

function formatFileSize(bytes) {
  if (!bytes || bytes <= 0) return "";
  const units = ["B", "KB", "MB", "GB"];
  let size = bytes;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  return size.toFixed(unitIndex === 0 ? 0 : 1) + " " + units[unitIndex];
}

function handleChatAction(type, msg) {
  switch (type) {
    case "replay":
      startReply(msg);
      break;
    case "copy":
      copyTextToClipboard(typeof msg.content === "string" ? msg.content : "");
      break;
    case "good":
      updateMessageById(msg.id, (m) => ({ ...m, _isGood: true, _isBad: false }));
      emit("feedback", { messageId: msg.id, type: "good" });
      break;
    case "bad":
      updateMessageById(msg.id, (m) => ({ ...m, _isGood: false, _isBad: true }));
      emit("feedback", { messageId: msg.id, type: "bad" });
      break;
    default:
      break;
  }
}

function copyTextToClipboard(text) {
  if (!text) return;
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).catch(() => fallbackCopy(text));
  } else {
    fallbackCopy(text);
  }
}

function fallbackCopy(text) {
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand("copy");
  document.body.removeChild(textarea);
}

function cancelReply() {
  replyTo.value = null;
}

function escapeMarkdownText(value) {
  return String(value || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function renderMarkdownContent(value) {
  const text = String(value || "");
  if (!text.trim()) {
    return "";
  }
  const safe = escapeMarkdownText(text);
  const html = String(marked.parse(safe, { gfm: true, breaks: true }));
  return DOMPurify.sanitize(html, MARKDOWN_SANITIZE_OPTIONS);
}

function toStatusText(status) {
  if (status === SEND_STATUS.SENDING) return t('status.sending', '发送中');
  if (status === SEND_STATUS.FAILED) return t('message.sendFailed', '发送失败');
  if (status === SEND_STATUS.READ) return t('status.read', '已读');
  return t('status.sent', '已发送');
}

function toTDesignStatus(status) {
  if (status === SEND_STATUS.SENDING) return "pending";
  if (status === SEND_STATUS.FAILED) return "error";
  if (status === "streaming") return "streaming";
  return "complete";
}

function toAbsoluteMediaURL(rawURL) {
  const text = String(rawURL || "").trim();
  if (!text) return "";
  if (/^(data:|blob:|https?:\/\/)/i.test(text)) return text;
  if (text.startsWith("//")) {
    const protocol = window.location.protocol === "https:" ? "https:" : "http:";
    return `${protocol}${text}`;
  }
  const base = String(props.apiBaseUrl || window.location.origin || "").replace(/\/api\/v1\/?$/, "");
  if (!base) return text;
  if (text.startsWith("/")) return `${base}${text}`;
  return `${base}/${text}`;
}

function formatBytes(value) {
  const size = Number(value || 0);
  if (size < 1024) return `${size}B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)}KB`;
  return `${(size / (1024 * 1024)).toFixed(1)}MB`;
}

function openImage(url) {
  if (!url) return;
  imagePreviewUrl.value = url;
  imagePreviewVisible.value = true;
}

function closeImagePreview() {
  imagePreviewVisible.value = false;
  imagePreviewUrl.value = "";
}

function markOutgoingAckByIncoming(message) {
  if (!message || message.businessType !== BUSINESS_MESSAGE_TYPES.VISITOR) {
    return false;
  }
  const incomingClientId = String(message?.clientId || "").trim();
  if (incomingClientId) {
    const msg = messages.value.find((item) => item?.isSelf && String(item?._clientId || "").trim() === incomingClientId);
    if (msg) {
      updateMessageById(msg.id, (oldMsg) => ({
        ...oldMsg,
        status: SEND_STATUS.SENT,
        _echoAcked: true,
        _serverMsgId: message.msgId || oldMsg._serverMsgId || "",
      }));
      clearSettleTimer(msg.id);
      return true;
    }
  }
  const payload = message.payload || {};
  const contentType = payload.contentType;
  const content = String(payload.content || payload.url || "").trim();
  if (!contentType || !content) {
    return false;
  }

  for (let i = 0; i < messages.value.length; i += 1) {
    const msg = messages.value[i];
    if (!msg?.isSelf || msg._echoAcked) {
      continue;
    }
    if (String(msg.contentType || "").trim() !== String(contentType).trim()) {
      continue;
    }
    let localComparable = "";
    if (contentType === BUSINESS_CONTENT_TYPES.TEXT) {
      localComparable = String(msg.content || "").trim();
    } else if (contentType === BUSINESS_CONTENT_TYPES.IMAGE) {
      localComparable = String(msg.content || "").trim();
    } else if (contentType === BUSINESS_CONTENT_TYPES.AUDIO) {
      localComparable = String(msg.audioUrl || msg.content || "").trim();
    } else if (contentType === BUSINESS_CONTENT_TYPES.FILE) {
      localComparable = String(msg.fileUrl || msg.content || "").trim();
    } else {
      localComparable = String(msg.content || msg.audioUrl || msg.fileUrl || "").trim();
    }
    if (localComparable !== content) {
      continue;
    }
    updateMessageById(msg.id, (oldMsg) => ({
      ...oldMsg,
      status: SEND_STATUS.SENT,
      _echoAcked: true,
      _serverMsgId: message.msgId || oldMsg._serverMsgId || "",
    }));
    clearSettleTimer(msg.id);
    return true;
  }
  return false;
}

function handleIncomingBusinessMessage(message) {
  if (!message) {
    return;
  }
  if (message.sid) {
    sessionId.value = message.sid;
  }

  const dedupeKey = message.msgId || `${message.businessType}:${message.payload?.contentType}:${message.payload?.content || message.payload?.url}:${message.timestamp}`;
  if (dedupeKey && receivedMsgIds.has(dedupeKey)) {
    return;
  }
  if (dedupeKey) {
    receivedMsgIds.add(dedupeKey);
  }

  if (message.businessType === BUSINESS_MESSAGE_TYPES.TYPING) {
    typingText.value = "Agent is typing...";
    if (typingTimer.value) clearTimeout(typingTimer.value);
    typingTimer.value = setTimeout(() => {
      typingText.value = "";
      typingTimer.value = null;
    }, 2200);
    return;
  }

  if (
    message.msgId &&
    (message.businessType === BUSINESS_MESSAGE_TYPES.AGENT || message.businessType === BUSINESS_MESSAGE_TYPES.SYSTEM)
  ) {
    wsClient.value?.sendAck(message.msgId, "received");
  }

  if (
    message.businessType === BUSINESS_MESSAGE_TYPES.SYSTEM &&
    String(message?.payload?.content || "").includes("\u5ba2\u670d\u5df2\u8bfb")
  ) {
    if (message.msgId && receivedMsgIds.has(`read:${message.msgId}`)) {
      return;
    }
    if (message.msgId) {
      receivedMsgIds.add(`read:${message.msgId}`);
    }
    for (let i = messages.value.length - 1; i >= 0; i -= 1) {
      const msg = messages.value[i];
      if (msg.isSelf && (msg.status === SEND_STATUS.SENT || msg.status === SEND_STATUS.SENDING)) {
        updateMessageById(msg.id, (oldMsg) => ({ ...oldMsg, status: SEND_STATUS.READ }));
      }
    }
    syncUnreadCount(0);
    return;
  }

  if (message.businessType === BUSINESS_MESSAGE_TYPES.SYSTEM) {
    applyAgentServiceFromSystemText(String(message?.payload?.content || ""));
  }

  const streamEnabled = Boolean(message?.payload?.raw?.stream);
  const streamDelta = Boolean(message?.payload?.raw?.stream_delta);
  const streamFinal = Boolean(message?.payload?.raw?.stream_final);
  const streamKey = String(message?.payload?.raw?.stream_key || "").trim();
  if (
    streamEnabled &&
    message.businessType === BUSINESS_MESSAGE_TYPES.AGENT &&
    !message?.payload?.from?.includes("visitor")
  ) {
    const deltaText = String(message?.payload?.content || "").trim();
    if (!streamKey || !deltaText) {
      return;
    }
    const streamId = `stream_${streamKey}`;
    const existingIdx = messages.value.findIndex((m) => m.id === streamId);
    if (streamDelta && existingIdx >= 0) {
      const existing = messages.value[existingIdx];
      messages.value[existingIdx] = {
        ...existing,
        content: deltaText,
        time: new Date(message.timestamp || Date.now()).toISOString(),
        status: "streaming",
        _streamFinal: false,
      };
      return;
    }
    if (streamDelta && existingIdx < 0) {
      const uiStreamMessage = buildChatUiMessageFromBusiness(message, {
        id: streamId,
        serviceName: resolvedServiceName.value,
        serviceAvatar: resolvedServiceAvatar.value,
        selfAvatar: selfAvatar.value,
      });
      if (uiStreamMessage) {
        pushMessage({ ...uiStreamMessage, status: "streaming", _streamFinal: false });
      }
      return;
    }
    if (streamFinal) {
      const finalUi = buildChatUiMessageFromBusiness(message, {
        id: createLocalMessageId("in"),
        serviceName: resolvedServiceName.value,
        serviceAvatar: resolvedServiceAvatar.value,
        selfAvatar: selfAvatar.value,
      });
      if (!finalUi) {
        return;
      }
      if (existingIdx >= 0) {
        messages.value[existingIdx] = {
          ...finalUi,
          id: streamId,
          status: SEND_STATUS.SENT,
          _streamFinal: true,
          _serverMsgId: String(message.msgId || "").trim(),
        };
      } else {
        pushMessage({
          ...finalUi,
          id: streamId,
          status: SEND_STATUS.SENT,
          _streamFinal: true,
          _serverMsgId: String(message.msgId || "").trim(),
        });
      }
      if (!finalUi.isSelf) {
        syncUnreadCount(unreadCount.value + 1);
      }
      return;
    }
  }

  const matchedLocalEcho = markOutgoingAckByIncoming(message);
  if (matchedLocalEcho) {
    return;
  }

  // Self-message echo: dedupe by msg_id to avoid duplicate sent bubbles.
  if (message.businessType === BUSINESS_MESSAGE_TYPES.VISITOR && message.msgId) {
    const hasLocal = messages.value.some((m) => m.isSelf && m._serverMsgId === message.msgId);
    if (hasLocal) {
      return;
    }
  }

  const uiMessage = buildChatUiMessageFromBusiness(message, {
    id: createLocalMessageId("in"),
    serviceName: resolvedServiceName.value,
    serviceAvatar: resolvedServiceAvatar.value,
    selfAvatar: selfAvatar.value,
  });

  if (uiMessage) {
    if (uiMessage.type === "image" || uiMessage.type === "audio") {
      uiMessage.content = toAbsoluteMediaURL(uiMessage.content);
      if (uiMessage.type === "audio") {
        uiMessage.audioUrl = toAbsoluteMediaURL(uiMessage.audioUrl || uiMessage.content);
      }
    } else if (uiMessage.type === "file") {
      uiMessage.fileUrl = toAbsoluteMediaURL(uiMessage.fileUrl || uiMessage.content);
    }
    pushMessage({ ...uiMessage, status: SEND_STATUS.SENT });
    if (!uiMessage.isSelf && message.businessType === BUSINESS_MESSAGE_TYPES.AGENT) {
      syncUnreadCount(unreadCount.value + 1);
    }
  }
}

function clearAndRebuildReceivedMsgIDs() {
  receivedMsgIds.clear();
  for (const m of messages.value) {
    const msgId = String(m?._serverMsgId || "").trim();
    if (msgId) {
      receivedMsgIds.add(msgId);
    }
  }
}

function compareChatMessages(a, b) {
  const leftTs = Number(new Date(a?.time || 0).getTime()) || 0;
  const rightTs = Number(new Date(b?.time || 0).getTime()) || 0;
  if (leftTs !== rightTs) {
    return leftTs - rightTs;
  }
  const leftId = String(a?._serverMsgId || a?.id || "");
  const rightId = String(b?._serverMsgId || b?.id || "");
  return leftId.localeCompare(rightId);
}

async function loadInitialHistory() {
  try {
    const resp = await api.getVisitorHistory({
      appId: props.appId,
      visitorId: visitorId.value,
      limit: 50,
    });
    if (resp?.code !== 0) {
      return;
    }
    const sid = String(resp?.data?.sid || "").trim();
    if (sid) {
      sessionId.value = sid;
    }
    const rows = Array.isArray(resp?.data?.messages) ? resp.data.messages : [];
    if (rows.length === 0) {
      return;
    }
    const mapped = rows
      .map((row) => normalizeIncomingBusinessMessage(row))
      .filter(Boolean)
      .map((row) =>
        buildChatUiMessageFromBusiness(row, {
          id: createLocalMessageId("hist"),
          serviceName: resolvedServiceName.value,
          serviceAvatar: resolvedServiceAvatar.value,
          selfAvatar: selfAvatar.value,
        })
      )
      .filter(Boolean)
      .map((msg) => ({
        ...msg,
        status: SEND_STATUS.SENT,
        _serverMsgId: String(msg?.business?.msgId || "").trim(),
      }))
      .map((msg) => {
        if (msg.type === "image" || msg.type === "audio") {
          const absolute = toAbsoluteMediaURL(msg.content);
          msg.content = absolute;
          if (msg.type === "audio") {
            msg.audioUrl = toAbsoluteMediaURL(msg.audioUrl || absolute);
          }
        } else if (msg.type === "file") {
          msg.fileUrl = toAbsoluteMediaURL(msg.fileUrl || msg.content);
        }
        return msg;
      })
      .sort(compareChatMessages);
    messages.value = mapped;
    for (const item of rows) {
      const normalized = normalizeIncomingBusinessMessage(item);
      if (normalized?.businessType === BUSINESS_MESSAGE_TYPES.SYSTEM) {
        applyAgentServiceFromSystemText(String(normalized?.payload?.content || ""));
      }
    }
    refreshAgentMetaInMessages();
    clearAndRebuildReceivedMsgIDs();
    await nextTick();
    scrollToBottom();
  } catch (error) {
    emit("ws-error", error);
  }
}

function ensureWsConnected() {
  const client = new WSClient(props.appId, visitorId.value, {
    wsUrl: wsEndpoint.value,
    onMessage: handleIncomingBusinessMessage,
    onStatusChange: (status) => {
      emit("ws-status", status);
      if (status === "reconnect-failed" || status === "disconnected" || status === "replaced") {
        for (const msg of messages.value) {
          if (msg.isSelf && msg.status === SEND_STATUS.SENDING) {
            updateMessageById(msg.id, (oldMsg) => ({ ...oldMsg, status: SEND_STATUS.FAILED }));
            clearSettleTimer(msg.id);
          }
        }
      }
      if (status === "replaced") {
        pushSystemTip("This account is active in another window. This window was disconnected.");
      }
      if (status === "connected" && wsConnectTimeoutTimer) {
        clearTimeout(wsConnectTimeoutTimer);
        wsConnectTimeoutTimer = null;
      }
    },
    onError: (error) => emit("ws-error", error),
  });

  wsClient.value = client;
  client.connect();
}

async function loadConfigAndConnect() {
  let loadingTooSlowTip = null;
  try {
    loadingTooSlowTip = setTimeout(() => {
      emit("ws-error", new Error("Connection is slow, please wait"));
    }, 3000);
    api.setBaseURL(props.apiBaseUrl);
    api.setUserId(visitorId.value);
    const response = await api.getConfig(props.appId);

    if (response?.code !== 0 || !response?.data) {
      throw new Error(response?.msg || "Failed to load config");
    }

    config.value = response.data;
    emit("config-loaded", response.data);
    await loadInitialHistory();
    ensureWsConnected();
    wsConnectTimeoutTimer = setTimeout(() => {
      if (wsClient.value && !wsClient.value.isConnected) {
        emit("ws-error", new Error("Connection timeout, please retry later"));
      }
    }, 3000);
  } catch (error) {
    if (isDomainForbiddenCode(error?.code, error?.httpStatus)) {
      emit("config-error", new Error("Domain is not in allowlist. Add current domain in admin app settings."));
      return;
    }
    emit("config-error", error);
  } finally {
    if (loadingTooSlowTip) {
      clearTimeout(loadingTooSlowTip);
    }
  }
}

function dispatchWithStatus(messageId, sendFn) {
  if (!wsClient.value || !wsClient.value.isConnected) {
    updateMessageById(messageId, (msg) => ({ ...msg, status: SEND_STATUS.FAILED }));
    return;
  }
  const clientId = sendFn();
  if (clientId) {
    updateMessageById(messageId, (msg) => ({ ...msg, _clientId: clientId }));
  }
  scheduleAutoSent(messageId);
}

function sendTyping() {
  const now = Date.now();
  if (now - lastTypingAt < 1200) return;
  lastTypingAt = now;
  wsClient.value?.sendTyping();
}

function sendText(content, localId = "") {
  const text = String(content || "").trim();
  if (!text) {
    return;
  }

  const currentReply = replyTo.value ? { ...replyTo.value } : null;
  const messageId = localId || createLocalMessageId("out");
  const uiMessage = {
    ...toOutboundUiMessage(
      BUSINESS_CONTENT_TYPES.TEXT,
      { content: text, replyTo: currentReply },
      messageId
    ),
    status: SEND_STATUS.SENDING,
    _retryPayload: { contentType: BUSINESS_CONTENT_TYPES.TEXT, data: { content: text, replyTo: currentReply } },
  };
  const exists = messages.value.some((m) => m.id === messageId);
  if (exists) {
    updateMessageById(messageId, () => uiMessage);
  } else {
    pushMessage(uiMessage);
  }
  dispatchWithStatus(messageId, () => wsClient.value?.sendText(text, { replyTo: currentReply }));
  replyTo.value = null;
}

function readAudioDuration(audioUrl) {
  return new Promise((resolve) => {
    const audio = new Audio(audioUrl);
    audio.onloadedmetadata = () => {
      resolve(Number.isFinite(audio.duration) ? audio.duration : 0);
    };
    audio.onerror = () => resolve(0);
  });
}

async function uploadByContentType(file, contentType) {
  const uploadResp = await api.uploadFile({
    appId: props.appId,
    file,
    contentType,
  });
  if (uploadResp?.code !== 0 || !uploadResp?.data?.url) {
    throw new Error(uploadResp?.msg || t('upload.failed', '上传失败'));
  }
  return uploadResp.data;
}

async function sendFileLike(file, presetDuration = 0) {
  const isImage = file.type.startsWith("image/");
  const isAudio = file.type.startsWith("audio/");
  const contentType = isAudio
    ? BUSINESS_CONTENT_TYPES.AUDIO
    : isImage
      ? BUSINESS_CONTENT_TYPES.IMAGE
      : BUSINESS_CONTENT_TYPES.FILE;

  const messageId = createLocalMessageId("out");
  const temporaryUrl = URL.createObjectURL(file);
  let duration = 0;
  if (contentType === BUSINESS_CONTENT_TYPES.AUDIO) {
    const measured = presetDuration > 0 ? presetDuration : await readAudioDuration(temporaryUrl);
    duration = Math.max(1, Math.round(Number(measured || 0)));
  }

  const currentReply = replyTo.value ? { ...replyTo.value } : null;
  const localUi = {
    ...toOutboundUiMessage(
      contentType,
      {
        url: temporaryUrl,
        name: file.name || "file",
        size: file.size || 0,
        duration,
        replyTo: currentReply,
      },
      messageId
    ),
    status: SEND_STATUS.SENDING,
  };
  pushMessage(localUi);

  try {
    const uploaded = await uploadByContentType(file, contentType);
    const remoteUrl = toAbsoluteMediaURL(uploaded.url);
    const retryData = {
      url: remoteUrl,
      name: uploaded.name || file.name || "file",
      size: Number(uploaded.size || file.size || 0),
      duration,
      replyTo: currentReply,
    };

    updateMessageById(messageId, (msg) => ({
      ...msg,
      content:
        contentType === BUSINESS_CONTENT_TYPES.AUDIO
          ? `[${t('message.voice', '语音消息')}]`
          : contentType === BUSINESS_CONTENT_TYPES.FILE
            ? (retryData.name || t('message.file', '文件'))
            : remoteUrl,
      audioUrl: contentType === BUSINESS_CONTENT_TYPES.AUDIO ? remoteUrl : msg.audioUrl,
      fileUrl: contentType === BUSINESS_CONTENT_TYPES.FILE ? remoteUrl : msg.fileUrl,
      fileName: contentType === BUSINESS_CONTENT_TYPES.FILE ? retryData.name : msg.fileName,
      fileSize: contentType === BUSINESS_CONTENT_TYPES.FILE ? retryData.size : msg.fileSize,
      duration: contentType === BUSINESS_CONTENT_TYPES.AUDIO ? duration : msg.duration,
      status: SEND_STATUS.SENDING,
      _retryPayload: { contentType, data: retryData },
    }));

    dispatchWithStatus(messageId, () => {
      if (contentType === BUSINESS_CONTENT_TYPES.AUDIO) {
        return wsClient.value?.sendAudio(remoteUrl, Math.max(1, Math.round(Number(duration || 0))), { replyTo: currentReply });
      }
      if (contentType === BUSINESS_CONTENT_TYPES.IMAGE) {
        return wsClient.value?.sendImage(remoteUrl, retryData.name || "image.jpg", { replyTo: currentReply });
      }
      return wsClient.value?.sendFile(remoteUrl, retryData.name || "file", retryData.size || 0, { replyTo: currentReply });
    });
    replyTo.value = null;
  } catch (error) {
    updateMessageById(messageId, (msg) => ({ ...msg, status: SEND_STATUS.FAILED }));
    emit("ws-error", error);
  } finally {
    URL.revokeObjectURL(temporaryUrl);
  }
}

async function handleFileUpload(fileList) {
  const files = Array.from(fileList || []);
  for (const file of files) {
    try {
      await sendFileLike(file);
    } catch (error) {
      emit("ws-error", error);
    }
  }
}

async function blobUrlToFile(blobUrl, filename, mimeType = "audio/webm") {
  const resp = await fetch(blobUrl);
  const blob = await resp.blob();
  return new window.File([blob], filename, { type: mimeType });
}

async function startRecording() {
  if (isRecording.value) {
    return;
  }

  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    const recorder = new MediaRecorder(stream);

    recordingStream.value = stream;
    mediaRecorder.value = recorder;
    audioChunks.value = [];
    recordingStartedAt.value = Date.now();

    recorder.ondataavailable = (event) => {
      if (event.data && event.data.size > 0) {
        audioChunks.value.push(event.data);
      }
    };

    recorder.onstop = async () => {
      const mimeType = recorder.mimeType || "audio/webm";
      const blob = new Blob(audioChunks.value, { type: mimeType });
      const audioUrl = URL.createObjectURL(blob);
      const duration = Math.max(
        1,
        Math.round(Math.max((Date.now() - recordingStartedAt.value) / 1000, await readAudioDuration(audioUrl)))
      );
      const audioFile = await blobUrlToFile(audioUrl, `voice_${Date.now()}.webm`, mimeType);

      try {
        await sendFileLike(audioFile, duration);
      } catch (error) {
        emit("ws-error", error);
      }

      cleanupRecordingStream();
      URL.revokeObjectURL(audioUrl);
    };

    recorder.start();
    isRecording.value = true;
  } catch (error) {
    cleanupRecordingStream();
    isPressRecording.value = false;
    isRecording.value = false;
    emit("ws-error", error);
  }
}

function stopRecording(send = true) {
  const recorder = mediaRecorder.value;
  if (!recorder) {
    return;
  }

  if (recorder.state === "recording") {
    if (send) {
      recorder.stop();
    } else {
      recorder.onstop = null;
      recorder.stop();
      cleanupRecordingStream();
    }
  } else {
    cleanupRecordingStream();
  }

  isRecording.value = false;
}

function cleanupRecordingStream() {
  if (recordingStream.value) {
    recordingStream.value.getTracks().forEach((track) => track.stop());
    recordingStream.value = null;
  }
  mediaRecorder.value = null;
  audioChunks.value = [];
  recordingStartedAt.value = 0;
  isRecording.value = false;
}

function handleVoiceRecord(payload) {
  if (!payload || !payload.status) {
    return;
  }

  if (payload.status === "start") {
    void startRecording();
    return;
  }

  if (payload.status === "stop") {
    stopRecording(true);
    return;
  }

  if (payload.status === "cancel") {
    stopRecording(false);
  }
}

function retryMessageById(messageId) {
  const msg = messages.value.find((m) => m.id === messageId);
  if (!msg || !msg._retryPayload) {
    return;
  }
  updateMessageById(messageId, (oldMsg) => ({ ...oldMsg, status: SEND_STATUS.SENDING }));
  const { contentType, data } = msg._retryPayload;
  dispatchWithStatus(messageId, () => {
    if (contentType === BUSINESS_CONTENT_TYPES.TEXT) {
      return wsClient.value?.sendText(String(data.content || ""), { replyTo: data.replyTo || null });
    }
    if (contentType === BUSINESS_CONTENT_TYPES.IMAGE) {
      return wsClient.value?.sendImage(String(data.url || ""), String(data.name || "image.jpg"), { replyTo: data.replyTo || null });
    }
    if (contentType === BUSINESS_CONTENT_TYPES.AUDIO) {
      return wsClient.value?.sendAudio(String(data.url || ""), Math.max(1, Math.round(Number(data.duration || 0))), { replyTo: data.replyTo || null });
    }
    return wsClient.value?.sendFile(String(data.url || ""), String(data.name || "file"), Number(data.size || 0), { replyTo: data.replyTo || null });
  });
}

function handleFileRetry(payload) {
  if (!payload?.message?.id) {
    return;
  }
  retryMessageById(payload.message.id);
}

function bindAudioElement(messageId, element) {
  if (!messageId) {
    return;
  }

  if (element) {
    audioElementMap.set(messageId, element);
  } else {
    audioElementMap.delete(messageId);
  }
}

function stopAllPlayingAudio() {
  for (const [id, el] of audioElementMap.entries()) {
    if (!el.paused) {
      el.pause();
      el.currentTime = 0;
    }
    if (playingMessageId.value === id) {
      playingMessageId.value = "";
    }
  }
}

async function toggleAudioPlay(message) {
  if (!message?.id) {
    return;
  }

  const audioEl = audioElementMap.get(message.id);
  if (!audioEl) {
    return;
  }

  if (playingMessageId.value === message.id && !audioEl.paused) {
    audioEl.pause();
    playingMessageId.value = "";
    return;
  }

  stopAllPlayingAudio();
  playingMessageId.value = message.id;

  try {
    await audioEl.play();
  } catch {
    playingMessageId.value = "";
  }
}

function stopAudio(messageId) {
  if (!messageId) {
    return;
  }
  const audioEl = audioElementMap.get(messageId);
  if (!audioEl) {
    return;
  }
  audioEl.pause();
  audioEl.currentTime = 0;
  if (playingMessageId.value === messageId) {
    playingMessageId.value = "";
  }
}

function handleAudioEnded(messageId) {
  if (playingMessageId.value === messageId) {
    playingMessageId.value = "";
  }
}

function resolveMessageAvatar(message) {
  if (message?.isSelf) {
    return selfAvatar.value;
  }
  return String(message?.avatar || resolvedServiceAvatar.value || avatarBySeed(resolvedServiceName.value || "agent"));
}

function resolveMessageName(message) {
  if (message?.isSelf) {
    return "";
  }
  const payload = message?.business?.raw?.payload || {};
  const payloadName = String(
    payload.agent_name || payload.from_name || payload.sender_name || payload.agent || ""
  ).trim();
  if (payloadName && payloadName !== "agent" && payloadName !== "system" && payloadName !== "visitor") {
    return payloadName;
  }
  const activeName = String(activeAgentName.value || "").trim();
  if (activeName) {
    return activeName;
  }
  const messageName = String(message?.name || "").trim();
  const appName = String(config.value?.name || "").trim();
  if (messageName && appName && messageName === appName && activeName) {
    return activeName;
  }
  if (messageName) {
    return messageName;
  }
  return String(resolvedServiceName.value || t('word.agent', '客服'));
}

function handleSend(value) {
  if (inputValue.value) {
    inputValue.value = "";
  }
  sendText(normalizeSendEventValue(value));
}

function pickImageFiles() {
  showEmojiPicker.value = false;
  filePickerAccept.value = "image/*";
  if (filePickerRef.value) {
    filePickerRef.value.accept = "image/*";
    filePickerRef.value.value = "";
  }
  filePickerRef.value?.click();
}

function pickAnyFiles() {
  showEmojiPicker.value = false;
  filePickerAccept.value = accept.value;
  if (filePickerRef.value) {
    filePickerRef.value.accept = accept.value;
    filePickerRef.value.value = "";
  }
  filePickerRef.value?.click();
}

function onPickerChanged(event) {
  const files = event?.target?.files;
  if (!files || files.length === 0) {
    return;
  }
  void handleFileUpload(files);
  event.target.value = "";
}

function toggleEmojiPicker() {
  showEmojiPicker.value = !showEmojiPicker.value;
}

function appendEmoji(emoji) {
  inputValue.value = `${inputValue.value || ""}${emoji}`;
  showEmojiPicker.value = false;
}

function toggleVoiceRecord() {
  showEmojiPicker.value = false;
  if (!isVoiceSupported.value) {
    emit("ws-error", new Error("This browser does not support voice recording"));
    return;
  }
  if (isRecording.value) {
    stopRecording(true);
    return;
  }
  void startRecording();
}

function handlePressRecordStart() {
  showEmojiPicker.value = false;
  if (!isVoiceSupported.value || props.disabled) {
    return;
  }
  if (isPressRecording.value || isRecording.value) {
    return;
  }
  isPressRecording.value = true;
  void startRecording();
}

function handlePressRecordStop() {
  if (!isPressRecording.value) {
    return;
  }
  isPressRecording.value = false;
  stopRecording(true);
}

function handlePressRecordCancel() {
  if (!isPressRecording.value) {
    return;
  }
  isPressRecording.value = false;
  stopRecording(false);
}

watch(
  () => inputValue.value,
  () => {
    sendTyping();
  }
);

function openRateDialog() {
  rateScore.value = 5;
  rateComment.value = "";
  rateDialogVisible.value = true;
}

function closeRateDialog() {
  rateDialogVisible.value = false;
}

async function submitRate() {
  if (!sessionId.value || rateScore.value < 1) {
    emit("ws-error", new Error("No rateable session yet"));
    return;
  }
  rateSaving.value = true;
  try {
    const resp = await api.rateSession({
      sid: sessionId.value,
      appId: props.appId,
      visitorId: visitorId.value,
      score: Number(rateScore.value || 0),
      comment: String(rateComment.value || ""),
    });
    if (resp?.data?.code !== 0) {
      throw new Error(resp?.data?.msg || "Failed to submit rating");
    }
    closeRateDialog();
  } catch (error) {
    emit("ws-error", error);
  } finally {
    rateSaving.value = false;
  }
}

function formatDuration(duration) {
  const totalSeconds = Math.max(0, Math.floor(Number(duration || 0)));
  const min = Math.floor(totalSeconds / 60)
    .toString()
    .padStart(2, "0");
  const sec = (totalSeconds % 60).toString().padStart(2, "0");
  return `${min}:${sec}`;
}

function formatMessageTime(value) {
  const date = new Date(value || Date.now());
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const hh = String(date.getHours()).padStart(2, "0");
  const mm = String(date.getMinutes()).padStart(2, "0");
  return `${hh}:${mm}`;
}

function normalizeSendEventValue(value) {
  if (typeof value === "string") {
    return value;
  }
  if (value && typeof value === "object") {
    return String(value.value || "");
  }
  return "";
}

watch(
  () => props.userId,
  (newValue) => {
    const normalized = normalizeAppUserId(newValue);
    if (normalized) {
      visitorId.value = normalized;
      return;
    }
    const fallback = normalizeSdkVisitorId(visitorId.value) || getOrCreateVisitorId();
    visitorId.value = fallback;
  }
);

onMounted(() => {
  void loadConfigAndConnect();
});

watch(
  () => props.disabled,
  (disabled) => {
    if (!disabled) {
      syncUnreadCount(0);
    }
  }
);

onBeforeUnmount(() => {
  stopAllPlayingAudio();
  stopRecording(false);
  wsClient.value?.disconnect();
  if (wsConnectTimeoutTimer) {
    clearTimeout(wsConnectTimeoutTimer);
    wsConnectTimeoutTimer = null;
  }
  if (typingTimer.value) {
    clearTimeout(typingTimer.value);
    typingTimer.value = null;
  }
  for (const timer of settleTimers.values()) {
    clearTimeout(timer);
  }
  settleTimers.clear();
  isPressRecording.value = false;
  replyTo.value = null;
  closeImagePreview();
});
</script>

<style scoped>
.kefu-liao-chat-core {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
}

.kefu-top-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  padding: 6px 8px 4px;
}

.kefu-rate-entry {
  border: 1px solid #dbe5f4;
  background: #f8fbff;
  color: #475569;
  border-radius: 999px;
  height: 24px;
  padding: 0 8px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  cursor: pointer;
}

.kefu-rate-entry:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.kefu-rate-entry:hover:not(:disabled) {
  color: #334155;
  border-color: #cfdcf1;
  background: #f1f6ff;
}

.kefu-liao-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px 16px;
}

:deep(.t-chat__footer__content) {
  padding: 8px 12px;
}

:deep(.t-chat__footer__textarea .t-textarea__inner) {
  max-height: 120px !important;
  overflow-y: auto !important;
  overscroll-behavior: contain;
}

.kefu-msg-row {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  width: 100%;
  max-width: 100%;
}

.kefu-msg-row.is-self {
  align-items: flex-end;
}

.kefu-msg-head {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.kefu-msg-head.is-self {
  justify-content: flex-end;
}

.kefu-msg-avatar {
  width: 22px;
  height: 22px;
  border-radius: 9999px;
  border: 1px solid #dbe3ef;
  background: #f8fafc;
  flex-shrink: 0;
  object-fit: cover;
}

.kefu-msg-main {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  width: 100%;
  max-width: min(100%, 82vw);
  min-width: 0;
}

.kefu-msg-main.is-self {
  align-items: flex-end;
}

.kefu-msg-name {
  margin: 0 0 1px;
  font-size: 11px;
  color: #64748b;
}

.kefu-msg-meta-line {
  margin-top: 3px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  line-height: 1;
  color: #94a3b8;
}

.kefu-msg-meta-line.is-self {
  justify-content: flex-end;
}

.kefu-msg-time {
  color: #94a3b8;
}

.kefu-msg-dot {
  color: #cbd5e1;
}

:deep(.t-chat-message__actionbar) {
  margin-top: 6px;
}

.kefu-custom-actionbar {
  display: flex;
  align-items: center;
  gap: 2px;
}

.kefu-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.2s ease;
  padding: 0;
}

.kefu-action-btn:hover {
  background: #f1f5f9;
  color: #475569;
}

.kefu-action-btn.active {
  color: #2563eb;
  background: #eff6ff;
}

:deep(.t-chat-sender) {
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

:deep(.t-chat-sender__textarea) {
  font-size: 14px;
  line-height: 1.5;
}

:deep(.t-chat-sender__textarea textarea) {
  font-size: 14px;
  line-height: 1.5;
}

:deep(.t-chat-sender__footer) {
  padding: 4px 8px;
}

:deep(.t-chat-action) {
  display: flex;
  align-items: center;
  gap: 4px;
}

:deep(.t-chat-action .t-button) {
  padding: 4px 8px;
  font-size: 12px;
  height: auto;
  min-width: auto;
}

:deep(.t-chat-action .t-icon) {
  font-size: 14px;
}

.kefu-file-bubble {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.02);
  min-width: 220px;
  max-width: 320px;
}

.kefu-file-info {
  display: flex;
  align-items: center;
  gap: 10px;
  overflow: hidden;
  flex: 1;
  min-width: 0;
}

.kefu-file-icon {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.04);
}

.kefu-file-detail {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow: hidden;
  min-width: 0;
}

.kefu-file-name {
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.kefu-file-size {
  font-size: 11px;
  color: #94a3b8;
}

.kefu-file-download {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.2s ease;
  text-decoration: none;
  border: none;
  background: transparent;
  margin-left: 8px;
}

.kefu-file-download:hover {
  color: #2563eb;
  background: #eef3fa;
}

.kefu-reply-quote {
  border-left: 2px solid #93c5fd;
  background: rgba(241, 245, 249, 0.96);
  border-radius: 8px;
  padding: 6px 10px;
  width: 100%;
  box-sizing: border-box;
  margin: 0;
}

.kefu-reply-quote.is-self {
  border-left-color: #60a5fa;
  background: rgba(255, 255, 255, 0.82);
}

.kefu-reply-quote-head {
  margin: 0;
  font-size: 11px;
  color: #1d4ed8;
  line-height: 1.35;
}

.kefu-reply-quote-preview {
  margin: 1px 0 0;
  font-size: 11px;
  color: #334155;
  line-height: 1.45;
}

.kefu-rich-content {
  display: inline-flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  width: fit-content;
  max-width: 100%;
  min-width: 0;
}

.kefu-rich-content.is-self {
  align-items: flex-start;
}

.kefu-rich-content.is-self .kefu-reply-quote-head,
.kefu-rich-content.is-self .kefu-reply-quote-preview {
  color: #1e293b;
}

.kefu-reply-bar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 8px;
  margin: 4px 0;
}

.kefu-reply-bar-text {
  flex: 1;
  min-width: 0;
}

.kefu-reply-bar-head {
  margin: 0;
  font-size: 12px;
  line-height: 1.35;
  color: #1e40af;
}

.kefu-reply-bar-preview {
  margin: 2px 0 0;
  font-size: 12px;
  line-height: 1.35;
  color: #64748b;
  white-space: normal;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.kefu-reply-bar-cancel {
  border: 0;
  width: 20px;
  height: 20px;
  margin-top: 1px;
  border-radius: 9999px;
  background: #e2e8f0;
  color: #334155;
  cursor: pointer;
  line-height: 1;
}

.kefu-msg-bubble {
  display: inline-block;
  width: fit-content;
  max-width: 100%;
  border-radius: 12px;
  border: 1px solid #dbe3ef;
  background: #ffffff;
  padding: 8px 10px;
  font-size: 13px;
  line-height: 1.55;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.kefu-msg-bubble.is-self {
  background: #dbeafe;
  border-color: #bfdbfe;
}

.kefu-markdown-content {
  max-width: 100%;
  min-width: 0;
}

.kefu-markdown-content p {
  margin: 0 0 4px;
}

.kefu-markdown-content p:last-child {
  margin-bottom: 0;
}

.kefu-markdown-content pre {
  margin: 4px 0;
  white-space: pre-wrap;
  background: rgba(15, 23, 42, 0.06);
  border-radius: 8px;
  padding: 8px;
  font-size: 12px;
  overflow-x: auto;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.kefu-markdown-content code {
  background: rgba(15, 23, 42, 0.08);
  border-radius: 4px;
  padding: 1px 4px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.kefu-markdown-content a {
  color: #2563eb;
  text-decoration: underline;
  word-break: break-all;
}

.kefu-typing-tip {
  font-size: 12px;
  color: #6b7280;
  padding: 6px 8px;
}

.kefu-system-message {
  margin: 8px auto;
  max-width: 80%;
  text-align: center;
  color: #6b7280;
  font-size: 12px;
  background: #f3f4f6;
  border-radius: 9999px;
  padding: 6px 10px;
}

.kefu-audio-bubble {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid #c7dcff;
  background: #eff6ff;
  border-radius: 12px;
  padding: 7px 9px;
  max-width: 260px;
}

.kefu-audio-bubble.is-self {
  background: #dbeafe;
  border-color: #93c5fd;
}

.kefu-rich-content.is-self .kefu-audio-symbol,
.kefu-rich-content.is-self .kefu-audio-icon-btn,
.kefu-rich-content.is-self .kefu-audio-duration {
  color: #1e3a8a;
}

.kefu-audio-symbol {
  color: #2563eb;
  flex-shrink: 0;
}

.kefu-audio-icon-btn {
  border: none;
  background: transparent;
  color: #1e40af;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 1px;
  border-radius: 9999px;
}

.kefu-audio-icon-btn:hover {
  background: rgba(37, 99, 235, 0.12);
}

.kefu-audio-icon-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.kefu-audio-duration {
  margin-left: auto;
  font-size: 12px;
  color: #4b5563;
}

.kefu-file-wrap,
.kefu-text-wrap,
.kefu-image-wrap {
  display: inline-flex;
  flex-direction: column;
  gap: 4px;
}

.kefu-file-meta,
.kefu-msg-meta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #6b7280;
}

.kefu-file-dot {
  color: #94a3b8;
}

.kefu-file-download {
  color: #1e40af;
  text-decoration: underline;
}

.kefu-msg-failed {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: #dc2626;
  font-weight: 500;
}

.kefu-msg-failed :deep(svg) {
  flex-shrink: 0;
}

.kefu-file-link {
  color: #1d4ed8;
  text-decoration: none;
}

.kefu-file-link:hover {
  text-decoration: underline;
}

.kefu-retry-btn {
  border: 0;
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
  color: #fff;
  border-radius: 6px;
  padding: 2px 10px;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.kefu-retry-btn:hover {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  box-shadow: 0 1px 4px rgba(220, 38, 38, 0.3);
}

.kefu-inline-image-wrap {
  display: inline-flex;
  border: 0;
  padding: 0;
  background: transparent;
  cursor: zoom-in;
}

.kefu-inline-image {
  max-width: 220px;
  border-radius: 8px;
  display: block;
}

.kefu-hidden-audio,
.kefu-hidden-file {
  display: none;
}

.kefu-input-toolbar {
  display: flex;
  gap: 0;
  padding: 4px 0;
  background: transparent;
  border: none;
}

.kefu-tool-btn {
  flex: 1;
  border: none;
  background: transparent;
  color: #5b6b7c;
  border-radius: 8px;
  padding: 8px 4px;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: all 0.2s ease;
  position: relative;
}

.kefu-tool-btn:hover:not(:disabled) {
  background: #eef3fa;
  color: #2563eb;
}

.kefu-tool-btn:active:not(:disabled) {
  background: #dbeafe;
  transform: scale(0.96);
}

.kefu-tool-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.kefu-tool-btn.active {
  background: linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%);
  color: #dc2626;
  box-shadow: inset 0 0 0 1px rgba(220, 38, 38, 0.15);
}

.kefu-tool-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.kefu-tool-btn:hover:not(:disabled) .kefu-tool-icon {
  background: rgba(37, 99, 235, 0.08);
  color: #2563eb;
}

.kefu-recording-pulse {
  animation: kefu-pulse-record 1.4s ease-in-out infinite;
  color: #dc2626 !important;
  background: rgba(220, 38, 38, 0.1) !important;
}

@keyframes kefu-pulse-record {

  0%,
  100% {
    opacity: 1;
  }

  50% {
    opacity: 0.35;
  }
}

.kefu-emoji-mask {
  position: fixed;
  inset: 0;
  z-index: 9997;
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  animation: kefu-fade-in 0.2s ease-out;
}

@keyframes kefu-fade-in {
  from {
    opacity: 0;
  }

  to {
    opacity: 1;
  }
}

@keyframes kefu-slide-up {
  from {
    transform: translateY(20px);
    opacity: 0;
  }

  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.kefu-emoji-dialog {
  width: min(400px, 92vw);
  border-radius: 20px;
  background: #ffffff;
  box-shadow:
    0 25px 50px -12px rgba(0, 0, 0, 0.25),
    0 12px 24px -8px rgba(0, 0, 0, 0.15);
  overflow: hidden;
  animation: kefu-slide-up 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.kefu-emoji-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px 2px;
  border-bottom: 1px solid #f0f2f5;
}

.kefu-emoji-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #1a1a2e;
  letter-spacing: -0.01em;
}

.kefu-emoji-close {
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  background: transparent;
  color: #94a3b8;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.kefu-emoji-close:hover {
  background: #f1f5f9;
  color: #475569;
  transform: rotate(90deg);
}

.kefu-emoji-tabs {
  display: flex;
  gap: 6px;
  padding: 10px 18px 0;
  overflow-x: auto;
}

.kefu-emoji-tab {
  flex-shrink: 0;
  padding: 7px 16px;
  border: none;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
  color: #64748b;
  background: #f1f5f9;
  cursor: pointer;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  white-space: nowrap;
}

.kefu-emoji-tab:hover {
  background: #e2e8f0;
  color: #334155;
}

.kefu-emoji-tab.active {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #ffffff;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.35);
}

.kefu-emoji-panel {
  max-height: min(42vh, 320px);
  overflow-y: auto;
  padding: 14px 18px 18px;
}

.kefu-emoji-panel::-webkit-scrollbar {
  width: 5px;
}

.kefu-emoji-panel::-webkit-scrollbar-track {
  background: transparent;
}

.kefu-emoji-panel::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 3px;
}

.kefu-emoji-panel::-webkit-scrollbar-thumb:hover {
  background: #94a3b8;
}

.kefu-emoji-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 6px;
}

.kefu-emoji-btn {
  aspect-ratio: 1;
  border: none;
  background: transparent;
  border-radius: 12px;
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 0;
}

.kefu-emoji-btn:hover {
  transform: scale(1.15) translateY(-2px);
  z-index: 10;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}

.kefu-emoji-btn:active {
  transform: scale(0.95);
}

.kefu-rate-dialog-mask {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.kefu-rate-dialog {
  width: min(360px, 90vw);
  background: #ffffff;
  border-radius: 12px;
  padding: 14px;
}

.kefu-rate-dialog h4 {
  margin: 0 0 10px;
  font-size: 16px;
  color: #111827;
}

.kefu-rate-stars {
  display: flex;
  gap: 6px;
  margin-bottom: 10px;
}

.kefu-rate-star {
  border: 0;
  background: transparent;
  font-size: 22px;
  color: #d1d5db;
  cursor: pointer;
}

.kefu-rate-star.active {
  color: #f59e0b;
}

.kefu-rate-input {
  width: 100%;
  min-height: 72px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  padding: 8px;
  resize: vertical;
}

.kefu-rate-actions {
  margin-top: 10px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.kefu-rate-cancel,
.kefu-rate-submit {
  border: 0;
  border-radius: 8px;
  padding: 6px 12px;
  cursor: pointer;
}

.kefu-rate-cancel {
  background: #e5e7eb;
  color: #111827;
}

.kefu-rate-submit {
  background: #2563eb;
  color: #ffffff;
}

.kefu-rate-submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.kefu-image-preview-mask {
  position: fixed;
  inset: 0;
  z-index: 9998;
  background: rgba(0, 0, 0, 0.72);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.kefu-image-preview {
  max-width: min(92vw, 980px);
  max-height: 88vh;
  border-radius: 8px;
  object-fit: contain;
}

@media (max-width: 640px) {
  .kefu-msg-avatar {
    width: 20px;
    height: 20px;
  }

  .kefu-msg-main {
    max-width: calc(100vw - 48px);
  }

  .kefu-msg-bubble {
    max-width: calc(100vw - 48px);
  }
}

:deep(.t-chat-message) {
  display: flex !important;
  flex-direction: row !important;
  align-items: flex-start !important;
  gap: 8px !important;
}

:deep(.t-chat-message__avatar) {
  flex-shrink: 0 !important;
  margin: 0 !important;
  padding: 0 !important;
}

:deep(.t-chat-message__header) {
  display: inline-flex !important;
  align-items: center !important;
  gap: 4px;
}

:deep(.t-chat-message__name) {
  margin-right: 4px;
}

:deep(.t-chat-message__main),
:deep(.t-chat__item__main) {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  flex: 0 1 auto;
  min-width: 0;
  max-width: min(100%, calc(100% - 56px));
}

:deep(.t-chat-message__content),
:deep(.t-chat__item__content) {
  font-size: 13px;
  line-height: 1.5;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  flex: 0 1 auto;
  width: auto;
  max-width: 100%;
  min-width: 0;
}

:deep(.t-chat-message),
:deep(.t-chat__item__inner) {
  width: 100%;
  display: flex !important;
  flex-direction: row !important;
  align-items: flex-start !important;
  justify-content: flex-start !important;
  gap: 10px !important;
}

:deep(.t-chat-message--user),
:deep(.t-chat__item__role--user) {
  flex-direction: row !important;
  justify-content: flex-start !important;
}

:deep(.t-chat-message--user .t-chat-message__main),
:deep(.t-chat__item__role--user .t-chat__item__main) {
  align-items: flex-start;
}

:deep(.t-chat-message--user .t-chat-message__avatar),
:deep(.t-chat__item__role--user .t-chat__item__avatar),
:deep(.t-chat-message__avatar),
:deep(.t-chat__item__avatar) {
  margin: 0 !important;
}

:deep(.t-chat-message__header),
:deep(.t-chat__item__header) {
  display: inline-flex !important;
  align-items: center !important;
  gap: 4px;
}

:deep(.t-chat-message__name),
:deep(.t-chat__item__name) {
  margin-right: 4px;
}

:deep(.t-chat-markdown) {
  max-width: 100%;
  overflow-x: auto;
  word-break: break-word;
  overflow-wrap: anywhere;
}

:deep(.t-chat__item__detail) {
  max-width: 100%;
  overflow-x: auto;
  word-break: break-word;
  overflow-wrap: anywhere;
}

@media (min-width: 1024px) {
  :deep(.t-chat-message--user),
  :deep(.t-chat__item__role--user) {
    flex-direction: row-reverse !important;
    justify-content: flex-start !important;
  }

  :deep(.t-chat-message--user .t-chat-message__main),
  :deep(.t-chat__item__role--user .t-chat__item__main) {
    align-items: flex-end;
    margin-left: auto;
  }
}

:deep(.t-chat__item__detail .cherry-markdown) {
  background: transparent;
  font-size: 14px;
  line-height: 1.6;
}

:deep(.t-chat__item__detail .cherry-markdown p) {
  margin: 0 0 8px;
}

:deep(.t-chat__item__detail .cherry-markdown p:last-child) {
  margin-bottom: 0;
}

:deep(.t-chat__item__detail .cherry-markdown pre) {
  max-width: 100%;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-word;
  border-radius: 8px;
  padding: 12px;
  background: #f6f8fa;
  font-size: 12px;
  margin: 8px 0;
}

:deep(.t-chat__item__detail .cherry-markdown code) {
  word-break: break-word;
  font-size: 12px;
}

:deep(.t-chat__item__detail .cherry-markdown code:not(pre code)) {
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  color: #e11d48;
}

:deep(.t-chat__item__detail .cherry-markdown table) {
  display: block;
  max-width: 100%;
  overflow-x: auto;
  border-collapse: collapse;
}

:deep(.t-chat__item__detail .cherry-markdown th),
:deep(.t-chat__item__detail .cherry-markdown td) {
  padding: 6px 10px;
  border: 1px solid #e5e7eb;
  font-size: 12px;
}

:deep(.t-chat__item__detail .cherry-markdown img) {
  max-width: 100%;
  height: auto;
}

:deep(.t-chat__item__detail .cherry-markdown blockquote) {
  border-left: 3px solid #d1d5db;
  padding-left: 12px;
  margin: 8px 0;
  color: #6b7280;
}

:deep(.t-chat__item__detail .cherry-markdown ul),
:deep(.t-chat__item__detail .cherry-markdown ol) {
  padding-left: 20px;
  margin: 4px 0;
}

:deep(.t-chat__item__detail .cherry-markdown li) {
  margin: 2px 0;
}

:deep(.t-chat__item__detail .cherry-markdown h1),
:deep(.t-chat__item__detail .cherry-markdown h2),
:deep(.t-chat__item__detail .cherry-markdown h3),
:deep(.t-chat__item__detail .cherry-markdown h4),
:deep(.t-chat__item__detail .cherry-markdown h5),
:deep(.t-chat__item__detail .cherry-markdown h6) {
  margin: 8px 0 4px;
  font-weight: 600;
  line-height: 1.4;
}

:deep(.t-chat__item__detail .cherry-markdown a) {
  color: #2563eb;
  text-decoration: none;
}

:deep(.t-chat__item__detail .cherry-markdown a:hover) {
  text-decoration: underline;
}

:deep(.t-chat__item__detail .cherry-markdown hr) {
  border: none;
  border-top: 1px solid #e5e7eb;
  margin: 8px 0;
}

:deep(.t-chat__item__attachments) {
  margin-top: 4px;
}

:deep(.t-attachment-list) {
  gap: 8px;
}

:deep(.t-filecard) {
  border-radius: 8px;
}

:deep(.t-filecard-image) {
  border-radius: 8px;
  overflow: hidden;
}

:deep(.t-chat__item__image) {
  margin-top: 4px;
  border-radius: 8px;
  overflow: hidden;
}

:deep(.t-chat__item__image img) {
  max-width: 200px;
  max-height: 200px;
  object-fit: cover;
  border-radius: 8px;
}
</style>
