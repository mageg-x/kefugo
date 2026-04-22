<template>
  <div class="kefu-liao-chat-core">
    <div v-if="typingText" class="kefu-typing-tip">{{ typingText }}</div>
    <div class="kefu-top-actions">
      <button
        v-if="sessionId"
        type="button"
        class="kefu-rate-entry"
        :disabled="disabled"
        @click="openRateDialog"
      >
        <Star :size="13" />
        <span>Rate</span>
      </button>
    </div>
    <LiaoMessageList
      ref="messageListRef"
      :messages="messages"
      :skip-user-message-adapter="true"
      :show-avatar="false"
      :show-avatar-self="false"
      :show-name="false"
      :show-time="false"
      :show-date-divider="false"
      :scroll-to-bottom="true"
      :empty-text="emptyText"
      class="kefu-liao-list"
      @file-retry="handleFileRetry"
    >
      <template #message="{ message }">
        <div v-if="message.type === 'system'" class="kefu-system-message">{{ message.content }}</div>
        <div v-else class="kefu-msg-row" :class="{ 'is-self': message.isSelf }">
          <div class="kefu-msg-head" :class="{ 'is-self': message.isSelf }">
            <img class="kefu-msg-avatar" :src="resolveMessageAvatar(message)" :alt="resolveMessageName(message)" />
            <p v-if="!message.isSelf" class="kefu-msg-name">{{ resolveMessageName(message) }}</p>
          </div>
          <div class="kefu-msg-main" :class="{ 'is-self': message.isSelf }">
            <div class="kefu-msg-bubble" :class="{ 'is-self': message.isSelf }">
              <div
                v-if="message.replyTo"
                class="kefu-reply-quote"
                :class="{ 'is-self': message.isSelf }"
              >
                <p class="kefu-reply-quote-head">
                  {{ message.replyTo.sender || "Quoted message" }}
                </p>
                <p class="kefu-reply-quote-preview">{{ replyPreviewText(message.replyTo) }}</p>
              </div>
              <template v-if="message.type === 'image'">
                <div class="kefu-image-wrap" @click="openImage(message.content)">
                  <img :src="message.content" alt="image" />
                </div>
              </template>
              <template v-else-if="message.type === 'audio'">
                <div class="kefu-audio-bubble" :class="{ 'is-self': message.isSelf }">
                  <AudioLines :size="16" class="kefu-audio-symbol" />
                  <button
                    class="kefu-audio-icon-btn"
                    type="button"
                    :title="playingMessageId === message.id ? 'pause' : 'play'"
                    :aria-label="playingMessageId === message.id ? 'pause' : 'play'"
                    @click="toggleAudioPlay(message)"
                  >
                    <CirclePause v-if="playingMessageId === message.id" :size="18" />
                    <CirclePlay v-else :size="18" />
                  </button>
                  <button
                    class="kefu-audio-icon-btn"
                    type="button"
                    title="stop"
                    aria-label="stop"
                    :disabled="playingMessageId !== message.id"
                    @click="stopAudio(message.id)"
                  >
                    <CircleStop :size="18" />
                  </button>
                  <span class="kefu-audio-duration">{{ formatDuration(message.duration) }}</span>
                  <audio
                    class="kefu-hidden-audio"
                    :ref="(el) => bindAudioElement(message.id, el)"
                    :src="message.audioUrl || message.content"
                    preload="metadata"
                    @ended="handleAudioEnded(message.id)"
                  ></audio>
                </div>
              </template>
              <template v-else-if="message.type === 'file'">
                <div class="kefu-file-wrap" :class="{ 'is-self': message.isSelf }">
                  <a class="kefu-file-link" :href="message.fileUrl || message.content" target="_blank" rel="noreferrer">
                    {{ message.fileName || "File" }}
                  </a>
                  <div class="kefu-file-meta">
                    <span>{{ formatBytes(message.fileSize) }}</span>
                    <span class="kefu-file-dot">·</span>
                    <a
                      class="kefu-file-download"
                      :href="message.fileUrl || message.content"
                      :download="message.fileName || 'file'"
                      @click.stop
                    >
                      Download
                    </a>
                  </div>
                </div>
              </template>
              <template v-else>
                <div class="kefu-markdown" v-html="renderMarkdown(message.content)"></div>
              </template>
            </div>
            <div class="kefu-msg-meta-line" :class="{ 'is-self': message.isSelf }">
              <template v-if="message.isSelf">
                <span class="kefu-msg-time">{{ formatMessageTime(message.time) }}</span>
                <span class="kefu-msg-dot">·</span>
                <span v-if="message.status === SEND_STATUS.FAILED" class="kefu-msg-status kefu-msg-failed">
                  <AlertCircle :size="12" />
                  Send failed
                </span>
                <span v-else class="kefu-msg-status">{{ toStatusText(message.status) }}</span>
                <button v-if="message.status === SEND_STATUS.FAILED" type="button" class="kefu-retry-btn" @click.stop="retryMessageById(message.id)">
                  Retry
                </button>
              </template>
              <template v-else>
                <span class="kefu-msg-time">{{ formatMessageTime(message.time) }}</span>
              </template>
              <button
                type="button"
                class="kefu-msg-reply-btn"
                title="reply"
                aria-label="reply"
                @click.stop="startReply(message)"
              >
                <MessageCircleReply :size="14" />
              </button>
            </div>
          </div>
        </div>
      </template>
    </LiaoMessageList>

    <div class="kefu-input-toolbar">
      <button
        type="button"
        class="kefu-tool-btn"
        :disabled="disabled"
        title="Send image"
        aria-label="Send image"
        @click="pickImageFiles"
      >
        <span class="kefu-tool-icon"><ImageIcon :size="18" /></span>
        <span>Image</span>
      </button>
      <button
        type="button"
        class="kefu-tool-btn"
        :disabled="disabled"
        title="Send file"
        aria-label="Send file"
        @click="pickAnyFiles"
      >
        <span class="kefu-tool-icon"><FileText :size="18" /></span>
        <span>File</span>
      </button>
      <button
        type="button"
        class="kefu-tool-btn"
        :disabled="disabled || !isVoiceSupported"
        :class="{ active: isRecording }"
        :title="isVoiceSupported ? 'Hold to record, release to send, move out to cancel' : 'This browser does not support recording'"
        aria-label="Send voice"
        @mousedown.prevent="handlePressRecordStart"
        @mouseup.prevent="handlePressRecordStop"
        @mouseleave.prevent="handlePressRecordCancel"
        @touchstart.prevent="handlePressRecordStart"
        @touchend.prevent="handlePressRecordStop"
        @touchcancel.prevent="handlePressRecordCancel"
      >
        <span class="kefu-tool-icon" v-if="!isRecording"><Mic :size="18" /></span>
        <span class="kefu-tool-icon kefu-recording-pulse" v-else><MicOff :size="18" /></span>
        <span>{{ isRecording ? "Send" : "Record" }}</span>
      </button>
      <button
        type="button"
        class="kefu-tool-btn"
        :disabled="disabled"
        title="Insert emoji"
        aria-label="Insert emoji"
        @click="toggleEmojiPicker"
      >
        <span class="kefu-tool-icon"><SmilePlus :size="18" /></span>
        <span>Emoji</span>
      </button>
    </div>

    <div v-if="showEmojiPicker" class="kefu-emoji-mask" @click.self="toggleEmojiPicker">
      <div class="kefu-emoji-dialog">
        <header class="kefu-emoji-head">
          <span>Pick Emoji</span>
          <button type="button" class="kefu-emoji-close" @click="toggleEmojiPicker">×</button>
        </header>
        <div class="kefu-emoji-panel">
          <section v-for="group in emojiGroups" :key="group.label" class="kefu-emoji-group">
            <p class="kefu-emoji-group-title">{{ group.label }}</p>
            <div class="kefu-emoji-grid">
              <button
                v-for="emoji in group.items"
                :key="group.label + emoji"
                type="button"
                class="kefu-emoji-btn"
                @click="appendEmoji(emoji)"
              >
                {{ emoji }}
              </button>
            </div>
          </section>
        </div>
      </div>
    </div>

    <div v-if="replyTo" class="kefu-reply-bar">
      <div class="kefu-reply-bar-text">
        <p class="kefu-reply-bar-head">Reply {{ replyTo.sender || "Message" }}</p>
        <p class="kefu-reply-bar-preview">{{ replyPreviewText(replyTo) }}</p>
      </div>
      <button type="button" class="kefu-reply-bar-cancel" @click="cancelReply">×</button>
    </div>

    <LiaoInputArea
      v-model="inputValue"
      :placeholder="inputPlaceholder"
      :disabled="disabled"
      :accept="accept"
      :multiple="true"
      :show-voice="true"
      :enable-voice-input="false"
      :enable-emoji-input="false"
      :enable-file-upload="false"
      :enable-camera="false"
      @send="handleSend"
      @file-upload="handleFileUpload"
      @voice-record="handleVoiceRecord"
    />

    <input
      ref="filePickerRef"
      type="file"
      class="kefu-hidden-file"
      :accept="filePickerAccept"
      :multiple="true"
      @change="onPickerChanged"
    />

    <div v-if="rateDialogVisible" class="kefu-rate-dialog-mask" @click.self="closeRateDialog">
      <div class="kefu-rate-dialog">
        <h4>Rate</h4>
        <div class="kefu-rate-stars">
          <button
            v-for="star in [1,2,3,4,5]"
            :key="star"
            type="button"
            class="kefu-rate-star"
            :class="{ active: rateScore >= star }"
            @click="rateScore = star"
          >
            ★
          </button>
        </div>
        <textarea v-model="rateComment" class="kefu-rate-input" maxlength="200" placeholder="Optional: leave a comment"></textarea>
        <div class="kefu-rate-actions">
          <button type="button" class="kefu-rate-cancel" @click="closeRateDialog">Cancel</button>
          <button type="button" class="kefu-rate-submit" :disabled="rateSaving || rateScore < 1" @click="submitRate">
            {{ rateSaving ? "Submitting..." : "Submit" }}
          </button>
        </div>
      </div>
    </div>
    <div v-if="imagePreviewVisible" class="kefu-image-preview-mask" @click.self="closeImagePreview">
      <img class="kefu-image-preview" :src="imagePreviewUrl" alt="image-preview" />
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import {
  Mic,
  AlertCircle,
  MessageCircleReply,
  Image as ImageIcon,
  FileText,
  MicOff,
  SmilePlus,
  AudioLines,
  CirclePlay,
  CirclePause,
  CircleStop,
  Star,
} from "lucide-vue-next";
import { LiaoInputArea, LiaoMessageList } from "@yuandezuohua/liaokit";
import { marked } from "marked";
import DOMPurify from "dompurify";
import api from "../script/api.js";
import { isDomainForbiddenCode } from "../script/error-codes.js";
import { WSClient } from "../script/wscli.js";
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
  serviceName: { type: String, default: "Agent" },
  serviceAvatar: { type: String, default: "" },
  apiBaseUrl: {
    type: String,
    default: () =>
      typeof window !== "undefined" && window.location?.origin
        ? window.location.origin
        : "http://localhost:5300",
  },
  wsUrl: { type: String, default: "" },
  emptyText: { type: String, default: "No messages" },
  inputPlaceholder: { type: String, default: "Type your message..." },
  disabled: { type: Boolean, default: false },
});

const emit = defineEmits(["config-loaded", "config-error", "ws-status", "ws-error", "unread-change"]);

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
    if (messageListRef.value?.scrollToBottom) {
      messageListRef.value.scrollToBottom(true);
    }
  });
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
    name: "System",
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
  const sender = message?.isSelf ? "Me" : String(resolveMessageName(message) || "Agent");
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

function cancelReply() {
  replyTo.value = null;
}

function toStatusText(status) {
  if (status === SEND_STATUS.SENDING) return "Sending";
  if (status === SEND_STATUS.FAILED) return "Send failed";
  if (status === SEND_STATUS.READ) return "Read";
  return "Sent";
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
        status: SEND_STATUS.SENT,
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
        pushMessage({ ...uiStreamMessage, status: SEND_STATUS.SENT, _streamFinal: false });
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
      });
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
    if (messageListRef.value?.scrollToBottom) {
      messageListRef.value.scrollToBottom(true);
    }
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
    throw new Error(uploadResp?.msg || "Upload failed");
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
          ? "[Voice message]"
          : contentType === BUSINESS_CONTENT_TYPES.FILE
            ? (retryData.name || "File")
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

function blobUrlToFile(blobUrl, filename, mimeType = "audio/webm") {
  return fetch(blobUrl)
    .then((resp) => resp.blob())
    .then((blob) => new File([blob], filename, { type: mimeType }));
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
  return String(resolvedServiceName.value || "Agent");
}

function escapeHTML(value) {
  return String(value || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function renderMarkdown(content) {
  const text = String(content || "").trim();
  if (!text) {
    return "";
  }
  const safe = escapeHTML(text);
  const html = String(marked.parse(safe, { gfm: true, breaks: true }));
  return DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true },
    ALLOWED_URI_REGEXP: /^(?:(?:https?|mailto|tel):|\/|#)/i,
  });
}

function handleSend(value) {
  if (inputValue.value) {
    inputValue.value = "";
  }
  sendText(value);
}

function pickImageFiles() {
  showEmojiPicker.value = false;
  filePickerAccept.value = "image/*";
  filePickerRef.value?.click();
}

function pickAnyFiles() {
  showEmojiPicker.value = false;
  filePickerAccept.value = accept.value;
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
:deep(.liao-input-area) {
  padding-top: 12px !important;
}

:deep(.liao-input-area-textarea) {
  max-height: 120px !important;
  overflow-y: auto !important;
  overscroll-behavior: contain;
}

:deep(.liao-input-area-textarea::-webkit-scrollbar) {
  width: 4px;
}

:deep(.liao-input-area-textarea::-webkit-scrollbar-thumb) {
  background: linear-gradient(180deg, #60a5fa 0%, #2563eb 100%);
  border-radius: 9999px;
}

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
  max-height: 100%;
  overflow: hidden;
}

:deep(.liao-message-list-wrapper) {
  min-height: 0 !important;
  height: 100% !important;
  overflow: hidden !important;
}

:deep(.liao-message-list-container) {
  min-height: 0 !important;
  height: 100% !important;
}

:deep(.liao-message-list) {
  min-height: 0 !important;
  height: 100% !important;
  overflow-y: auto !important;
}

:deep(.liao-message-item-content),
:deep(.liao-message-item-content p),
:deep(.liao-message-item-content span),
:deep(.liao-message-item-content div),
:deep(.liao-message-item-content code),
:deep(.liao-message-item-content pre) {
  font-size: 13px !important;
  line-height: 1.45 !important;
  word-break: break-word !important;
  overflow-wrap: anywhere !important;
}

:deep(.liao-message-list-item) {
  display: flex;
  padding: 6px 8px;
}

:deep(.liao-message-item-avatar),
:deep(.liao-message-list-item .liao-message-item-avatar) {
  display: none !important;
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

.kefu-msg-reply-btn {
  border: 0;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
}

.kefu-msg-reply-btn:hover {
  color: #2563eb;
}

.kefu-reply-quote {
  border-left: 2px solid #93c5fd;
  background: rgba(219, 234, 254, 0.35);
  border-radius: 8px;
  padding: 4px 7px;
  margin-bottom: 6px;
}

.kefu-reply-quote.is-self {
  border-left-color: #60a5fa;
}

.kefu-reply-quote-head {
  margin: 0;
  font-size: 11px;
  color: #2563eb;
}

.kefu-reply-quote-preview {
  margin: 1px 0 0;
  font-size: 11px;
  color: #475569;
}

.kefu-reply-bar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  min-height: 52px;
  padding: 2px 12px;
  border-top: 1px solid #e2e8f0;
  background: #f8fbff;
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
  margin-left: auto;
}

.kefu-markdown :deep(p) {
  margin: 0 0 4px;
}

.kefu-markdown :deep(p:last-child) {
  margin-bottom: 0;
}

.kefu-markdown :deep(pre) {
  background: #f1f5f9;
  border-radius: 6px;
  padding: 8px 10px;
  margin: 4px 0;
  overflow-x: hidden;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.kefu-markdown :deep(code) {
  background: #f1f5f9;
  border-radius: 3px;
  padding: 1px 4px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.kefu-markdown :deep(a) {
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

.kefu-image-wrap img {
  max-width: 220px;
  border-radius: 8px;
  cursor: zoom-in;
}

.kefu-hidden-audio,
.kefu-hidden-file {
  display: none;
}

.kefu-input-toolbar {
  display: flex;
  gap: 0;
  padding: 6px 12px;
  border-top: 1px solid #eef2f7;
  background: linear-gradient(180deg, #fafbfc 0%, #f5f7fa 100%);
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
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}

.kefu-emoji-panel {
  max-height: min(46vh, 340px);
  overflow: auto;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.kefu-emoji-mask {
  position: fixed;
  inset: 0;
  z-index: 9997;
  background: rgba(15, 23, 42, 0.36);
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding: 12px;
}

.kefu-emoji-dialog {
  width: min(520px, 96vw);
  border-radius: 14px;
  border: 1px solid #dbe3ef;
  background: #ffffff;
  box-shadow: 0 22px 48px rgba(15, 23, 42, 0.26);
  padding: 10px;
}

.kefu-emoji-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  font-weight: 600;
  color: #334155;
  margin-bottom: 8px;
}

.kefu-emoji-close {
  border: 0;
  border-radius: 8px;
  width: 28px;
  height: 28px;
  cursor: pointer;
  background: #f1f5f9;
  color: #475569;
  font-size: 18px;
  line-height: 1;
}

.kefu-emoji-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.kefu-emoji-group-title {
  margin: 0;
  font-size: 11px;
  color: #64748b;
}

.kefu-emoji-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
}

.kefu-emoji-btn {
  border: 1px solid #dbeafe;
  background: #f8fbff;
  border-radius: 10px;
  height: 30px;
  font-size: 16px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.kefu-emoji-btn:hover {
  background: #f0f4ff;
  border-color: #93c5fd;
  transform: scale(1.08);
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
  :deep(.liao-message-list-item) {
    padding-left: 6px;
    padding-right: 6px;
  }

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
</style>
