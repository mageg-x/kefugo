<template>
  <div class="inbox-page">
    <aside class="inbox-sessions">
      <div class="inbox-side-head">
        <div>
          <h2>{{ t("pageInbox.title") }}</h2>
          <p>{{ t("pageInbox.status.assigned") }} {{ activeCount }} · {{ t("pageInbox.status.unassigned") }} {{
            unassignedCount }}</p>
        </div>
        <el-button size="small" @click="reloadSessions">{{ t("action.refresh") }}</el-button>
      </div>

      <div class="inbox-filters">
        <button type="button" class="filter-toggle-btn" @click="toggleFilterPanel">
          <span>{{ filterPanelCollapsed ? t("pageInbox.filter.expand") : t("pageInbox.filter.collapse") }}</span>
        </button>
      </div>

      <div v-show="!filterPanelCollapsed" class="inbox-filters inbox-filters-expand">
        <div class="inbox-filter-row">
          <el-select v-model="statusFilter" size="small" @change="reloadSessions">
            <el-option :label="t('pageInbox.filter.allStatus')" value="" />
            <el-option :label="t('pageInbox.status.unassigned')" value="unassigned" />
            <el-option :label="t('pageInbox.status.assigned')" value="assigned" />
            <el-option :label="t('pageInbox.status.unread')" value="unread" />
            <el-option :label="t('pageInbox.status.unreply')" value="unreply" />
            <el-option :label="t('pageInbox.status.closed')" value="closed" />
          </el-select>
          <el-select v-model="assignedFilter" size="small" @change="reloadSessions">
            <el-option :label="t('pageInbox.filter.allSessions')" value="" />
            <el-option :label="t('pageInbox.filter.mineOnly')" value="mine" />
            <el-option :label="t('pageInbox.filter.unassignedOnly')" value="unassigned" />
          </el-select>
          <el-select v-model="appFilter" size="small" @change="reloadSessions">
            <el-option :label="t('pageInbox.filter.allApps')" value="" />
            <el-option v-for="app in appOptions" :key="app.value" :label="app.label" :value="app.value" />
          </el-select>
        </div>
        <div class="inbox-filter-row">
          <el-date-picker v-model="startTime" size="small" type="datetime"
            :placeholder="t('pageInbox.filter.startTime')" value-format="YYYY-MM-DD HH:mm:ss"
            @change="reloadSessions" />
          <el-date-picker v-model="endTime" size="small" type="datetime" :placeholder="t('pageInbox.filter.endTime')"
            value-format="YYYY-MM-DD HH:mm:ss" @change="reloadSessions" />
        </div>
      </div>

      <div class="session-list" v-loading="sessionLoading">
        <button v-for="session in sessions" :key="session.sid" class="session-row"
          :class="{ active: selectedSession?.sid === session.sid }" @click="selectSession(session)">
          <div class="session-main">
            <p class="visitor">{{ session.visitor_id }}</p>
            <p class="last-msg">{{ session.last_message || t('pageInbox.empty.noMessage') }}</p>
          </div>
          <div class="session-right">
            <el-tag :type="statusTagType(session.status)" size="small">{{ statusLabel(session.status) }}</el-tag>
            <span v-if="session.unread_count > 0" class="unread">{{ session.unread_count }}</span>
          </div>
        </button>
      </div>
    </aside>

    <main ref="inboxChatRef" class="inbox-chat" :class="{ 'chat-wide': isChatWide }">
      <template v-if="selectedSession">
        <header class="chat-head">
          <button type="button" class="visitor-trigger" @click="openVisitorDrawer">
            <img class="visitor-trigger-avatar" :src="visitorAvatarUrl" alt="visitor" />
            <span class="visitor-trigger-main">
              <strong>{{ selectedSession.visitor_id }}</strong>
              <small>{{ selectedSession.sid }}</small>
              <small class="visitor-profile">
                {{ selectedSession.last_client_ip || "-" }} · {{ selectedSession.last_device || "-" }} · {{
                  selectedSession.last_geo || "-" }}
              </small>
            </span>
          </button>
          <div class="chat-actions">
            <el-button size="small" type="primary" plain @click="acceptCurrentSession" :disabled="!canAcceptSession">
              {{ t("pageInbox.action.accept") }}
            </el-button>
            <el-button size="small" type="warning" plain @click="openTransferDialog" :disabled="!canControlSession">
              {{ t("pageInbox.action.transfer") }}
            </el-button>
            <el-button size="small" type="danger" plain @click="closeCurrentSession" :disabled="!canControlSession">
              {{ t("pageInbox.action.close") }}
            </el-button>
            <el-button size="small" type="info" plain @click="markFollowUp" :disabled="!canControlSession">
              {{ t("pageInbox.action.followUp") }}
            </el-button>
          </div>
        </header>

        <section ref="messageContainerRef" class="chat-messages">
          <button v-if="hasMoreHistory" class="load-more" @click="loadMoreMessages" :disabled="historyLoading">
            {{ historyLoading ? t("pageInbox.loading") : t("pageInbox.action.loadMoreHistory") }}
          </button>
          <template v-for="msg in messages" :key="msg.msg_id || msg.local_id">
            <t-chat-message v-if="msg.content_type === 'system'" role="system"
              :content="[{ type: 'text', data: typeof msg.content === 'string' ? msg.content : '' }]"
              :datetime="formatTime(msg.timestamp)" variant="outline" />
            <t-chat-message v-else :role="msg.isSelf ? 'user' : 'assistant'" :avatar="messageAvatar(msg)"
              :name="messageDisplayName(msg)" :datetime="formatTime(msg.timestamp)"
              :variant="msg.isSelf ? 'base' : 'outline'" :status="toTDesignStatus(msg.status)"
              :content="CUSTOM_MESSAGE_SEGMENTS" :allow-content-segment-custom="true">
              <template #content>
                <div class="msg-rich-content" :class="{ self: msg.isSelf }">
                  <div v-if="msg.replyTo" class="msg-reply-quote" :class="{ self: msg.isSelf }">
                    <p class="msg-reply-quote-head">{{ msg.replyTo.sender || t("pageInbox.word.quotedMessage") }}</p>
                    <p class="msg-reply-quote-preview">{{ replyPreviewText(msg.replyTo) }}</p>
                  </div>

                  <div v-if="msg.content_type === BUSINESS_CONTENT_TYPES.TEXT" class="msg-bubble"
                    :class="{ self: msg.isSelf }">
                    <div class="msg-markdown-content" v-html="renderMarkdownContent(msg.content)"></div>
                  </div>

                  <button v-else-if="msg.content_type === BUSINESS_CONTENT_TYPES.IMAGE" type="button"
                    class="msg-image-wrap" :class="{ self: msg.isSelf }"
                    @click="previewImage(msg.url || msg.content)">
                    <img class="msg-image" :src="msg.url || msg.content" alt="image" />
                  </button>

                  <div v-else-if="msg.content_type === BUSINESS_CONTENT_TYPES.AUDIO" class="msg-audio-pill"
                    :class="{ self: msg.isSelf }">
                    <AudioLines :size="16" class="msg-audio-symbol" />
                    <button type="button" class="msg-audio-icon-btn"
                      :title="playingAudioId === (msg.msg_id || msg.local_id) ? 'pause' : 'play'"
                      :aria-label="playingAudioId === (msg.msg_id || msg.local_id) ? 'pause' : 'play'"
                      @click="toggleAudio(msg)">
                      <CirclePause v-if="playingAudioId === (msg.msg_id || msg.local_id)" :size="18" />
                      <CirclePlay v-else :size="18" />
                    </button>
                    <button type="button" class="msg-audio-icon-btn" title="stop" aria-label="stop"
                      :disabled="playingAudioId !== (msg.msg_id || msg.local_id)" @click="stopAudio(msg)">
                      <CircleStop :size="18" />
                    </button>
                    <span class="msg-audio-duration">{{ formatDuration(msg.duration) }}</span>
                    <audio class="msg-hidden-audio" :ref="(el) => bindAudio(msg, el)" :src="msg.url || msg.content"
                      @ended="onAudioEnded(msg)"></audio>
                  </div>

                  <div v-else-if="msg.content_type === BUSINESS_CONTENT_TYPES.FILE" class="kefu-file-bubble">
                    <div class="kefu-file-info">
                      <span class="kefu-file-icon" :style="{ color: getFileIconColor(msg) }">
                        <component :is="getFileIconComponent(msg)" :size="28" />
                      </span>
                      <div class="kefu-file-detail">
                        <span class="kefu-file-name">{{ msg.name || msg.content || t('message.file') }}</span>
                        <span v-if="msg.size" class="kefu-file-size">{{ formatFileSize(msg.size) }}</span>
                      </div>
                    </div>
                    <a v-if="msg.url || msg.content" class="kefu-file-download" :href="msg.url || msg.content"
                      :download="msg.name || t('message.file')" :title="t('action.download')"
                      :aria-label="t('action.download')">
                      <Download :size="18" />
                    </a>
                  </div>

                  <div v-else class="msg-bubble" :class="{ self: msg.isSelf }">
                    <div class="msg-markdown-content" v-html="renderMarkdownContent(msg.content)"></div>
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
                <div class="msg-meta-line" :class="{ self: msg.isSelf }">
                  <template v-if="msg.isSelf">
                    <span class="msg-time">{{ formatTime(msg.timestamp) }}</span>
                    <span class="msg-dot">·</span>
                    <span v-if="msg.status === UI_MESSAGE_STATUS.FAILED" class="msg-status msg-status-failed">
                      <AlertCircle :size="12" />
                      {{ t("pageInbox.msgStatus.failed") }}
                    </span>
                    <span v-else class="msg-status">{{ toMessageStatusText(msg.status) }}</span>
                    <button v-if="msg.status === UI_MESSAGE_STATUS.FAILED" class="retry-btn"
                      @click.stop="retryMessage(msg)">{{
                        t("action.retry") }}</button>
                  </template>
                  <template v-else>
                    <span class="msg-time">{{ formatTime(msg.timestamp) }}</span>
                  </template>
                </div>
              </template>
            </t-chat-message>
          </template>
        </section>

        <footer class="chat-input-wrap">
          <div v-if="isVisitorTyping" class="typing-tip">{{ t("pageInbox.tip.visitorTyping") }}</div>
          <div v-if="isSessionClosed" class="closed-tip">{{ t("pageInbox.tip.sessionClosedNoSend") }}</div>
          <div v-if="showEmojiPicker" class="emoji-mask" @click.self="toggleEmojiPicker">
            <div class="emoji-dialog">
              <header class="emoji-dialog-head">
                <span>{{ t("pageInbox.action.pickEmoji") }}</span>
                <button type="button" class="emoji-close-btn" @click="toggleEmojiPicker">×</button>
              </header>
              <div class="emoji-panel">
                <section v-for="group in emojiGroups" :key="group.label" class="emoji-group">
                  <p class="emoji-group-title">{{ group.label }}</p>
                  <div class="emoji-grid">
                    <button v-for="emoji in group.items" :key="group.label + emoji" type="button" class="emoji-btn"
                      @click="appendEmoji(emoji)">
                      {{ emoji }}
                    </button>
                  </div>
                </section>
              </div>
            </div>
          </div>
          <div v-if="replyTo" class="reply-bar">
            <div class="reply-bar-text">
              <p class="reply-bar-head">{{ t("pageInbox.action.reply") }} {{ replyTo.sender || t("message.text") }}</p>
              <p class="reply-bar-preview">{{ replyPreviewText(replyTo) }}</p>
            </div>
            <button type="button" class="reply-bar-cancel" @click="cancelReply">×</button>
          </div>
          <div class="chat-sender-host" @paste.capture="handleSenderPaste">
            <t-chat-sender v-model="inputMessage" :placeholder="sendHintText" :disabled="!canSend()"
              :send-btn-disabled="!canSend()" @send="(val) => sendText(val)">
              <template #footer-prefix>
                <div class="kefu-input-toolbar">
                  <button type="button" class="kefu-tool-btn" :disabled="!canSend()"
                    :title="t('pageInbox.action.quickReply')" :aria-label="t('pageInbox.action.quickReply')"
                    @click="openSnippetDialog = true">
                    <span class="kefu-tool-icon">
                      <MessageSquare :size="16" />
                    </span>
                    <span>{{ t("pageInbox.action.quickReply") }}</span>
                  </button>
                  <button type="button" class="kefu-tool-btn" :disabled="!canSend() || aiSuggestLoading"
                    :title="t('pageInbox.action.aiSuggest')" :aria-label="t('pageInbox.action.aiSuggest')"
                    @click="applyAISuggest">
                    <span class="kefu-tool-icon">
                      <Brain :size="16" />
                    </span>
                    <span>{{ t("pageInbox.action.aiSuggest") }}</span>
                  </button>
                  <button type="button" class="kefu-tool-btn" :disabled="!canSend()" :title="t('message.image')"
                    :aria-label="t('message.image')" @click="pickFile('image/*')">
                    <span class="kefu-tool-icon">
                      <ImagePlus :size="16" />
                    </span>
                    <span>{{ t("message.image") }}</span>
                  </button>
                  <button type="button" class="kefu-tool-btn" :disabled="!canSend()" :title="t('message.file')"
                    :aria-label="t('message.file')" @click="pickFile('*/*')">
                    <span class="kefu-tool-icon">
                      <FileText :size="16" />
                    </span>
                    <span>{{ t("message.file") }}</span>
                  </button>
                  <button type="button" class="kefu-tool-btn" :disabled="!canSend() || !isRecordSupported"
                    :class="{ active: isRecording }"
                    :title="isRecordSupported ? t('voice.holdToRecord', '按住录音，松开发送，移出取消') : t('voice.notSupported', '此浏览器不支持录音')"
                    :aria-label="t('pageInbox.action.record')" @mousedown.prevent="handlePressRecordStart"
                    @mouseup.prevent="handlePressRecordStop" @mouseleave.prevent="handlePressRecordCancel"
                    @touchstart.prevent="handlePressRecordStart" @touchend.prevent="handlePressRecordStop"
                    @touchcancel.prevent="handlePressRecordCancel">
                    <span class="kefu-tool-icon" v-if="!isRecording">
                      <Mic :size="16" />
                    </span>
                    <span class="kefu-tool-icon kefu-recording-pulse" v-else>
                      <MicOff :size="16" />
                    </span>
                    <span>{{ isRecording ? t("action.send") : t("pageInbox.action.record") }}</span>
                  </button>
                  <button type="button" class="kefu-tool-btn" :disabled="!canSend()" :title="t('pageInbox.action.emoji')"
                    :aria-label="t('pageInbox.action.emoji')" @click="toggleEmojiPicker">
                    <span class="kefu-tool-icon">
                      <SmilePlus :size="16" />
                    </span>
                    <span>{{ t("pageInbox.action.emoji") }}</span>
                  </button>
                </div>
              </template>
            </t-chat-sender>
          </div>
          <div class="send-row">
            <span class="ws-status" :class="wsStatus">{{ wsStatusText }}</span>
          </div>
        </footer>
      </template>

      <div v-else class="empty-chat">
        <el-icon :size="56">
          <ChatDotRound />
        </el-icon>
        <p>{{ t("pageInbox.empty.selectSession") }}</p>
      </div>
    </main>

    <el-dialog v-model="transferDialogVisible" :title="t('pageInbox.dialog.transfer')" width="420px">
      <el-select v-model="transferTargetAgent" :placeholder="t('pageInbox.placeholder.selectTargetAgent')"
        style="width: 100%">
        <el-option v-for="agent in transferAgents" :key="agent.username" :label="agent.username"
          :value="agent.username" />
      </el-select>
      <template #footer>
        <el-button @click="transferDialogVisible = false">{{ t("action.cancel") }}</el-button>
        <el-button type="primary" :disabled="!transferTargetAgent" @click="confirmTransfer">{{
          t("pageInbox.action.confirmTransfer") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="openSnippetDialog" :title="t('pageInbox.dialog.quickReply')" width="680px">
      <div class="snippet-grid" v-loading="snippetLoading">
        <button v-for="snippet in snippets" :key="snippet.id" class="snippet-card" @click="useSnippet(snippet)">
          <div class="snippet-head">
            <el-tag size="small">{{ snippet.category }}</el-tag>
            <span>{{ snippet.usage_count || 0 }}{{ t("pageInbox.word.times") }}</span>
          </div>
          <h4>{{ snippet.title }}</h4>
          <p>{{ snippet.content }}</p>
        </button>
        <div v-if="!snippetLoading && snippets.length === 0" class="snippet-empty">{{ t("pageInbox.empty.noSnippet") }}
        </div>
      </div>
    </el-dialog>

    <el-drawer v-model="visitorDrawerVisible" :title="t('pageInbox.dialog.visitorProfile')" direction="rtl"
      size="380px">
      <div class="visitor-drawer">
        <img class="visitor-drawer-avatar" :src="visitorAvatarUrl" alt="visitor" />
        <div class="visitor-drawer-id">{{ selectedSession?.visitor_id || "-" }}</div>
        <div class="visitor-kv">
          <label>{{ t("pageInbox.field.sessionId") }}</label>
          <span>{{ selectedSession?.sid || "-" }}</span>
        </div>
        <div class="visitor-kv">
          <label>{{ t("pageInbox.field.appId") }}</label>
          <span>{{ selectedSession?.app_id || "-" }}</span>
        </div>
        <div class="visitor-kv">
          <label>{{ t("status.label") }}</label>
          <span>{{ statusLabel(selectedSession?.status || "") }}</span>
        </div>
        <div class="visitor-kv">
          <label>{{ t("pageInbox.field.currentAgent") }}</label>
          <span>{{ selectedSession?.cur_agent_id || t("pageInbox.status.unassigned") }}</span>
        </div>
        <div class="visitor-kv">
          <label>IP</label>
          <span>{{ selectedSession?.last_client_ip || "-" }}</span>
        </div>
        <div class="visitor-kv">
          <label>{{ t("pageInbox.field.device") }}</label>
          <span>{{ selectedSession?.last_device || "-" }}</span>
        </div>
        <div class="visitor-kv">
          <label>{{ t("pageInbox.field.geo") }}</label>
          <span>{{ selectedSession?.last_geo || "-" }}</span>
        </div>
        <div class="visitor-kv">
          <label>{{ t("pageInbox.field.userAgent") }}</label>
          <span class="visitor-ua">{{ selectedSession?.last_user_agent || "-" }}</span>
        </div>
        <div class="visitor-kv">
          <label>{{ t("pageInbox.field.rating") }}</label>
          <span>{{ selectedSession?.rating_score || 0 }} / 5</span>
        </div>
        <div class="visitor-kv">
          <label>{{ t("pageInbox.field.lastMessage") }}</label>
          <span>{{ selectedSession?.last_message || "-" }}</span>
        </div>
      </div>
    </el-drawer>

    <el-dialog v-model="imagePreviewVisible" :title="t('pageInbox.dialog.imagePreview')" width="860px" top="6vh">
      <img class="preview-image" :src="imagePreviewUrl" alt="preview" />
    </el-dialog>

    <input ref="fileInputRef" type="file" class="hidden-input" :accept="fileAccept" @change="onPickedFile" />
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue";
import DOMPurify from "dompurify";
import { ElMessage } from "element-plus";
import { ChatDotRound } from "@element-plus/icons-vue";
import { marked } from "marked";
import {
  ImagePlus,
  Mic,
  MicOff,
  AlertCircle,
  FileText,
  MessageSquare,
  Brain,
  SmilePlus,
  AudioLines,
  CirclePlay,
  CirclePause,
  CircleStop,
  MessageCircleReply,
  Download,
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
import api from "@/script/api";
import { AgentWSClient } from "@/script/ws-agent";
import { useStore } from "@/script/store";
import {
  BUSINESS_CONTENT_TYPES,
  BUSINESS_MESSAGE_TYPES,
  UI_MESSAGE_STATUS,
  buildInboxUiMessageFromBusiness,
  buildInboxUiMessageFromOutgoing,
  createLocalId,
  normalizeIncomingBusinessMessage,
  toInboxDisplayText,
} from "@/script/message-protocol";
import { t } from "@/script/i18n-text";

const store = useStore();

const sessions = ref([]);
const selectedSession = ref(null);
const sessionLoading = ref(false);
const messages = ref([]);
const messageCursor = ref("");
const hasMoreHistory = ref(false);
const historyLoading = ref(false);
const inputMessage = ref("");
const wsStatus = ref("disconnected");
const wsClient = ref(null);
const inboxChatRef = ref(null);
const messageContainerRef = ref(null);
const statusFilter = ref("");
const assignedFilter = ref("");
const appFilter = ref("");
const appOptions = ref([]);
const startTime = ref("");
const endTime = ref("");
const filterPanelCollapsed = ref(true);

const transferDialogVisible = ref(false);
const transferAgents = ref([]);
const transferTargetAgent = ref("");
const CUSTOM_MESSAGE_SEGMENTS = Object.freeze([{ type: "custom", data: " " }]);
const MARKDOWN_SANITIZE_OPTIONS = {
  USE_PROFILES: { html: true },
  ALLOWED_URI_REGEXP: /^(?:(?:https?|mailto|tel):|\/|#)/i,
};
const openSnippetDialog = ref(false);
const snippetLoading = ref(false);
const snippets = ref([]);
const visitorDrawerVisible = ref(false);
const imagePreviewVisible = ref(false);
const imagePreviewUrl = ref("");
const isVisitorTyping = ref(false);
const showEmojiPicker = ref(false);
const emojiGroups = [
  { label: "Common", items: ["😀", "😁", "😂", "🤣", "😊", "🙂", "😉", "😍", "🥰", "😘", "😎", "🤩", "🤔", "😴", "😭", "😡"] },
  { label: "Gestures", items: ["👍", "👎", "👌", "✌️", "🤝", "👏", "🙏", "💪", "👀", "🎉", "🔥", "✨"] },
  { label: "Symbols", items: ["❤️", "💙", "💚", "💛", "💜", "🧡", "💯", "✅", "❌", "⚠️", "⭐", "🌈"] },
];
const typingTimers = new Map();
let lastTypingAt = 0;

const fileInputRef = ref(null);
const fileAccept = ref("*/*");
const playingAudioId = ref("");
const audioMap = new Map();
const retryTimers = new Map();
const mediaRecorderRef = ref(null);
const recordingStreamRef = ref(null);
const audioChunksRef = ref([]);
const recordingStartAtRef = ref(0);
const isRecording = ref(false);
const aiSuggestLoading = ref(false);
const replyTo = ref(null);
const agentSettings = ref({
  soundEnabled: true,
  desktopNotifyEnabled: false,
  typingIndicatorEnabled: true,
  enterToSend: true,
  aiEnabled: false,
});
const lastNotifyAtBySid = new Map();
const isChatWide = ref(false);
let inboxChatResizeObserver = null;

const activeCount = computed(() => sessions.value.filter((s) => s.status === "assigned").length);
const unassignedCount = computed(() => sessions.value.filter((s) => s.status === "unassigned").length);
const currentUserName = computed(() => store.user?.username || store.user?.name || "");
const currentUserDisplayName = computed(() => store.user?.username || store.user?.name || t("role.agent"));
const isMineSession = computed(() => selectedSession.value?.cur_agent_id === currentUserName.value);
const canForceClose = computed(() => store.user?.role === "admin");
const canControlSession = computed(() => isMineSession.value || canForceClose.value);
const canAcceptSession = computed(() => {
  if (!selectedSession.value) return false;
  if (selectedSession.value.status === "closed") return false;
  const owner = String(selectedSession.value.cur_agent_id || "").trim();
  const sessionAppID = String(selectedSession.value.app_id || "").trim();
  if (!owner) {
    if (!sessionAppID) return true;
    if (store.user?.role === "admin") return true;
    return appOptions.value.some((app) => String(app?.value || "").trim() === sessionAppID);
  }
  return owner === currentUserName.value;
});
const isSessionClosed = computed(() => selectedSession.value?.status === "closed");
const visitorAvatarUrl = computed(() => buildAvatarURL(selectedSession.value?.visitor_id || "visitor"));
const isRecordSupported = computed(() => {
  if (typeof window === "undefined") return false;
  return Boolean(window.MediaRecorder && navigator?.mediaDevices?.getUserMedia);
});
const wsStatusText = computed(() => {
  if (wsStatus.value === "connected") return t("pageInbox.ws.connected");
  if (wsStatus.value === "reconnect-failed") return t("pageInbox.ws.reconnectFailed");
  return t("pageInbox.ws.disconnected");
});
const sendHintText = computed(() =>
  isSessionClosed.value
    ? t("pageInbox.tip.sessionClosed")
    : agentSettings.value.enterToSend
      ? t("pageInbox.tip.enterSend")
      : t("pageInbox.tip.ctrlEnterSend")
);

function statusLabel(status) {
  const map = {
    unassigned: t("pageInbox.status.unassigned"),
    unread: t("pageInbox.status.unread"),
    unreply: t("pageInbox.status.unreply"),
    assigned: t("pageInbox.status.assigned"),
    follow: t("pageInbox.status.follow"),
    closed: t("pageInbox.status.closed"),
  };
  return map[status] || status;
}

function statusTagType(status) {
  if (status === "assigned") return "success";
  if (status === "unread" || status === "unreply") return "warning";
  if (status === "closed") return "info";
  return "primary";
}

function toggleFilterPanel() {
  filterPanelCollapsed.value = !filterPanelCollapsed.value;
}

function buildAvatarURL(seed) {
  const text = String(seed || "user").trim() || "user";
  return `https://api.dicebear.com/9.x/adventurer/svg?seed=${encodeURIComponent(text)}`;
}

function toAbsoluteMediaURL(rawURL) {
  const text = String(rawURL || "").trim();
  if (!text) return "";
  if (/^(data:|blob:|https?:\/\/)/i.test(text)) return text;
  if (text.startsWith("//")) {
    const protocol = window.location.protocol === "https:" ? "https:" : "http:";
    return `${protocol}${text}`;
  }
  const base = String(api.baseURL || window.location.origin || "").replace(/\/api\/v1\/?$/, "");
  if (!base) return text;
  if (text.startsWith("/")) return `${base}${text}`;
  return `${base}/${text}`;
}

function normalizeMessageMediaFields(msg) {
  if (!msg || typeof msg !== "object") return msg;
  const normalized = { ...msg };
  if (normalized.content_type === BUSINESS_CONTENT_TYPES.IMAGE) {
    const absolute = toAbsoluteMediaURL(normalized.url || normalized.content);
    normalized.url = absolute;
    normalized.content = absolute;
  } else if (normalized.content_type === BUSINESS_CONTENT_TYPES.AUDIO) {
    const absolute = toAbsoluteMediaURL(normalized.url || normalized.content);
    normalized.url = absolute;
    normalized.content = absolute;
  } else if (normalized.content_type === BUSINESS_CONTENT_TYPES.FILE) {
    const absolute = toAbsoluteMediaURL(normalized.url || normalized.content);
    normalized.url = absolute;
    if (!normalized.content) {
      normalized.content = absolute;
    }
  }
  return normalized;
}

function replyPreviewText(target) {
  if (!target) return "";
  const contentType = String(target.contentType || "").toLowerCase();
  if (contentType === BUSINESS_CONTENT_TYPES.IMAGE) return `[${t("message.image")}]`;
  if (contentType === BUSINESS_CONTENT_TYPES.AUDIO) return `[${t("message.voice")}]`;
  if (contentType === BUSINESS_CONTENT_TYPES.FILE) return `[${t("message.file")}]`;
  return String(target.preview || "").trim();
}

function buildReplyPayloadFromMessage(message) {
  if (!message) {
    return null;
  }
  const msgId = String(message?.msg_id || "").trim();
  const contentType = String(message?.content_type || BUSINESS_CONTENT_TYPES.TEXT).toLowerCase();
  let preview = "";
  if (contentType === BUSINESS_CONTENT_TYPES.TEXT) {
    preview = String(message?.content || "").trim();
  } else if (contentType === BUSINESS_CONTENT_TYPES.IMAGE) {
    preview = `[${t("message.image")}]`;
  } else if (contentType === BUSINESS_CONTENT_TYPES.AUDIO) {
    preview = `[${t("message.voice")}]`;
  } else {
    preview = String(message?.name || `[${t("message.file")}]`).trim();
  }
  const sender = message?.isSelf ? t("pageInbox.word.me") : String(messageDisplayName(message) || t("pageInbox.word.visitor"));
  return {
    msgId,
    contentType,
    preview: preview.slice(0, 120),
    sender,
    timestamp: Number(message?.timestamp || Math.floor(Date.now() / 1000)),
  };
}

function startReply(message) {
  if (!message) {
    return;
  }
  replyTo.value = buildReplyPayloadFromMessage(message);
}

function cancelReply() {
  replyTo.value = null;
}

function messageAvatar(msg) {
  if (msg?.isSelf) {
    return buildAvatarURL(currentUserName.value || "agent");
  }
  return buildAvatarURL(selectedSession.value?.visitor_id || "visitor");
}

function messageDisplayName(msg) {
  if (msg?.isSelf) {
    return currentUserDisplayName.value;
  }
  return selectedSession.value?.visitor_id || t("pageInbox.word.visitor");
}

function toMessageStatusText(status) {
  if (status === UI_MESSAGE_STATUS.SENDING) return t("pageInbox.msgStatus.sending");
  if (status === UI_MESSAGE_STATUS.FAILED) return t("pageInbox.msgStatus.failed");
  return t("pageInbox.msgStatus.sent");
}

function toTDesignStatus(status) {
  if (status === UI_MESSAGE_STATUS.SENDING) return "pending";
  if (status === UI_MESSAGE_STATUS.FAILED) return "error";
  return "complete";
}

function formatTime(ts) {
  if (!ts) return "--";
  const n = Number(ts);
  const ms = n > 1e12 ? n : n * 1000;
  const d = new Date(ms);
  if (Number.isNaN(d.getTime())) return "--";
  return d.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
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

function normalizeSendEventValue(value) {
  if (typeof value === "string") {
    return value.trim();
  }
  if (value && typeof value === "object") {
    return String(value.value || "").trim();
  }
  return "";
}

function compareInboxMessages(a, b) {
  const leftTs = Number(a?.timestamp || 0);
  const rightTs = Number(b?.timestamp || 0);
  if (leftTs !== rightTs) {
    return leftTs - rightTs;
  }
  const leftId = String(a?.msg_id || a?.local_id || "");
  const rightId = String(b?.msg_id || b?.local_id || "");
  return leftId.localeCompare(rightId);
}

function sortInboxMessages(list = []) {
  return [...list].sort(compareInboxMessages);
}

function formatDuration(sec) {
  const n = Number(sec || 0);
  const m = Math.floor(n / 60).toString().padStart(2, "0");
  const s = Math.floor(n % 60).toString().padStart(2, "0");
  return `${m}:${s}`;
}

function formatBytes(size) {
  const n = Number(size || 0);
  if (n < 1024) return `${n}B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)}KB`;
  return `${(n / (1024 * 1024)).toFixed(1)}MB`;
}

function previewImage(url) {
  if (!url) return;
  imagePreviewUrl.value = url;
  imagePreviewVisible.value = true;
}

function openVisitorDrawer() {
  visitorDrawerVisible.value = true;
}

function toMessage(item) {
  const normalized = normalizeIncomingBusinessMessage(item);
  if (!normalized) {
    return null;
  }
  const msg = buildInboxUiMessageFromBusiness(normalized, { localId: createLocalId("hist") });
  return normalizeMessageMediaFields(msg);
}

async function reloadSessions() {
  sessionLoading.value = true;
  try {
    const startTimeVal = startTime.value ? Math.floor(new Date(startTime.value).getTime() / 1000) : undefined;
    const endTimeVal = endTime.value ? Math.floor(new Date(endTime.value).getTime() / 1000) : undefined;
    const resp = await api.listSessions({
      page: 1,
      page_size: 200,
      status: statusFilter.value,
      assigned: assignedFilter.value,
      app_id: appFilter.value || undefined,
      start_time: startTimeVal,
      end_time: endTimeVal,
    });
    sessions.value = resp?.data?.data?.data || [];
  } catch (error) {
    ElMessage.error(error.message || t("pageInbox.toast.loadSessionsFailed"));
  } finally {
    sessionLoading.value = false;
  }
}

async function selectSession(session) {
  selectedSession.value = session;
  messages.value = [];
  messageCursor.value = "";
  hasMoreHistory.value = false;
  const idx = sessions.value.findIndex((s) => s.sid === session.sid);
  if (idx >= 0) {
    sessions.value[idx].unread_count = 0;
    sessions.value[idx].status = sessions.value[idx].status === "unread" ? "assigned" : sessions.value[idx].status;
  }
  await loadHistory();
  if (canControlSession.value) {
    await api.readSession(session.sid).catch(() => { });
  }
}

async function loadHistory(before = "") {
  if (!selectedSession.value) return;
  historyLoading.value = true;
  try {
    const resp = await api.getSessionMessages({
      sid: selectedSession.value.sid,
      limit: 30,
      before,
    });
    const rows = resp?.data?.data?.data || [];
    const mapped = sortInboxMessages(rows.map(toMessage).filter(Boolean));
    if (before) {
      messages.value = sortInboxMessages([...mapped, ...messages.value]);
    } else {
      messages.value = mapped;
      nextTick(scrollToBottom);
    }
    messageCursor.value = resp?.data?.data?.next_cursor || "";
    hasMoreHistory.value = rows.length >= 30;
  } catch (error) {
    ElMessage.error(error.message || t("pageInbox.toast.loadHistoryFailed"));
  } finally {
    historyLoading.value = false;
  }
}

function loadMoreMessages() {
  if (!messageCursor.value || historyLoading.value) return;
  loadHistory(messageCursor.value);
}

function scrollToBottom() {
  const el = messageContainerRef.value;
  if (!el) return;
  el.scrollTop = el.scrollHeight;
}

function appendMessage(msg) {
  messages.value = sortInboxMessages([...messages.value, msg]);
  nextTick(scrollToBottom);
}

function updateLocalMessageStatus(localId, status) {
  const idx = messages.value.findIndex((m) => m.local_id === localId);
  if (idx >= 0) {
    messages.value[idx].status = status;
  }
}

function scheduleMarkFailed(localId, timeoutMs = 10000) {
  clearRetryTimer(localId);
  const t = setTimeout(() => {
    updateLocalMessageStatus(localId, UI_MESSAGE_STATUS.FAILED);
    retryTimers.delete(localId);
  }, timeoutMs);
  retryTimers.set(localId, t);
}

function clearRetryTimer(localId) {
  const t = retryTimers.get(localId);
  if (t) {
    clearTimeout(t);
    retryTimers.delete(localId);
  }
}

function canSend() {
  if (!selectedSession.value) return false;
  if (isSessionClosed.value) return false;
  return canControlSession.value;
}

function sendText(text) {
  const content = normalizeSendEventValue(text) || String(inputMessage.value || "").trim();
  if (!content || !canSend()) return;

  const localId = createLocalId("out");
  const clientId = createLocalId("cid");
  const currentReply = replyTo.value ? { ...replyTo.value } : null;
  appendMessage({
    ...buildInboxUiMessageFromOutgoing(
      BUSINESS_CONTENT_TYPES.TEXT,
      { content, replyTo: currentReply },
      { sid: selectedSession.value.sid, localId }
    ),
    status: UI_MESSAGE_STATUS.SENDING,
    client_id: clientId,
  });
  inputMessage.value = "";

  const result = wsClient.value?.sendPayload(
    selectedSession.value.sid,
    BUSINESS_CONTENT_TYPES.TEXT,
    { content, clientId, replyTo: currentReply }
  );
  replyTo.value = null;
  const ok = Boolean(result?.ok);
  updateLocalMessageStatus(localId, ok ? UI_MESSAGE_STATUS.SENT : UI_MESSAGE_STATUS.SENDING);
  if (!ok) scheduleMarkFailed(localId);
  else clearRetryTimer(localId);
}

function handleInputKeydown(event) {
  if (!event) return;
  if (agentSettings.value.enterToSend) {
    if (event.key === "Enter" && !event.ctrlKey && !event.metaKey && !event.shiftKey) {
      event.preventDefault();
      sendText();
    }
    return;
  }
  if (event.key === "Enter" && event.ctrlKey) {
    event.preventDefault();
    sendText();
  }
}

async function acceptCurrentSession() {
  if (!selectedSession.value) return;
  try {
    const resp = await api.acceptSession(selectedSession.value.sid);
    selectedSession.value = resp?.data?.data?.session || selectedSession.value;
    await reloadSessions();
    const sid = selectedSession.value?.sid;
    if (sid) {
      const latest = sessions.value.find((s) => s.sid === sid);
      if (latest) selectedSession.value = latest;
    }
    ElMessage.success(t("pageInbox.toast.accepted"));
  } catch (error) {
    ElMessage.error(error.message || t("pageInbox.toast.acceptFailed"));
  }
}

async function closeCurrentSession() {
  if (!selectedSession.value) return;
  try {
    const ok = wsClient.value?.sendClose(selectedSession.value.sid);
    if (!ok) {
      await api.closeSession(selectedSession.value.sid);
    }
    selectedSession.value = { ...selectedSession.value, status: "closed" };
    ElMessage.success(t("pageInbox.toast.sessionClosed"));
    await reloadSessions();
  } catch (error) {
    ElMessage.error(error.message || t("pageInbox.toast.closeFailed"));
  }
}

async function markFollowUp() {
  if (!selectedSession.value) return;
  try {
    await api.followUpSession(selectedSession.value.sid);
    await reloadSessions();
    ElMessage.success(t("pageInbox.toast.markedFollow"));
  } catch (error) {
    ElMessage.error(error.message || t("pageInbox.toast.markFailed"));
  }
}

function handleInputTyping() {
  if (!agentSettings.value.typingIndicatorEnabled) return;
  if (!selectedSession.value || !canSend()) return;
  const now = Date.now();
  if (now - lastTypingAt < 1200) return;
  lastTypingAt = now;
  wsClient.value?.sendTyping(selectedSession.value.sid);
}

async function openTransferDialog() {
  if (!selectedSession.value) return;
  try {
    const resp = await api.listSessionAgents({ app_id: selectedSession.value.app_id });
    transferAgents.value = resp?.data?.data?.data || [];
    transferTargetAgent.value = "";
    transferDialogVisible.value = true;
  } catch (error) {
    ElMessage.error(error.message || t("pageInbox.toast.loadAgentsFailed"));
  }
}

async function confirmTransfer() {
  if (!selectedSession.value || !transferTargetAgent.value) return;
  try {
    await api.transferSession(selectedSession.value.sid, transferTargetAgent.value);
    transferDialogVisible.value = false;
    ElMessage.success(t("pageInbox.toast.transferSuccess"));
    await reloadSessions();
    const sid = selectedSession.value?.sid;
    if (sid) {
      const latest = sessions.value.find((s) => s.sid === sid);
      if (latest) selectedSession.value = latest;
    }
  } catch (error) {
    ElMessage.error(error.message || t("pageInbox.toast.transferFailed"));
  }
}

function pickFile(accept) {
  fileAccept.value = accept;
  if (fileInputRef.value) {
    fileInputRef.value.accept = accept;
    fileInputRef.value.value = "";
  }
  fileInputRef.value?.click();
}

function inferFileExtension(mimeType) {
  const normalized = String(mimeType || "").trim().toLowerCase();
  if (!normalized) return "bin";
  const direct = normalized.split("/")[1] || "";
  if (direct.includes("png")) return "png";
  if (direct.includes("jpeg") || direct.includes("jpg")) return "jpg";
  if (direct.includes("gif")) return "gif";
  if (direct.includes("webp")) return "webp";
  if (direct.includes("bmp")) return "bmp";
  if (direct.includes("svg")) return "svg";
  if (direct.includes("pdf")) return "pdf";
  if (direct.includes("plain")) return "txt";
  if (direct.includes("json")) return "json";
  if (direct.includes("zip")) return "zip";
  if (direct.includes("mpeg")) return "mp3";
  if (direct.includes("ogg")) return "ogg";
  if (direct.includes("wav")) return "wav";
  if (direct.includes("webm")) return "webm";
  if (direct.includes("mp4")) return "mp4";
  return direct.replace(/[^a-z0-9]+/g, "") || "bin";
}

function ensureClipboardFileName(file) {
  if (!file) return null;
  if (String(file.name || "").trim()) return file;
  const isImage = String(file.type || "").startsWith("image/");
  const isAudio = String(file.type || "").startsWith("audio/");
  const extension = inferFileExtension(file.type);
  const prefix = isImage ? "pasted_image" : isAudio ? "pasted_audio" : "pasted_file";
  return new window.File([file], `${prefix}_${Date.now()}.${extension}`, {
    type: file.type || "application/octet-stream",
    lastModified: Date.now(),
  });
}

function extractClipboardFiles(event) {
  const clipboardData = event?.clipboardData;
  if (!clipboardData) return [];
  const files = [];
  for (const item of Array.from(clipboardData.items || [])) {
    if (item?.kind !== "file") continue;
    const file = ensureClipboardFileName(item.getAsFile());
    if (file) files.push(file);
  }
  if (files.length > 0) return files;
  return Array.from(clipboardData.files || []).map(ensureClipboardFileName).filter(Boolean);
}

function toggleEmojiPicker() {
  showEmojiPicker.value = !showEmojiPicker.value;
}

function appendEmoji(emoji) {
  inputMessage.value += emoji;
  showEmojiPicker.value = false;
  nextTick(() => {
    const textarea = document.querySelector('.chat-input');
    if (textarea) {
      textarea.focus();
    }
  });
}

function detectContentType(file) {
  if (file.type.startsWith("image/")) return BUSINESS_CONTENT_TYPES.IMAGE;
  if (file.type.startsWith("audio/")) return BUSINESS_CONTENT_TYPES.AUDIO;
  return BUSINESS_CONTENT_TYPES.FILE;
}

async function uploadPickedFile(file, contentType) {
  const resp = await api.uploadFile(selectedSession.value.app_id, file, contentType);
  const data = resp?.data?.data || {};
  if (!data?.url) {
    throw new Error(t("pageInbox.toast.uploadFailed"));
  }
  return data;
}

async function sendFileMessage(file) {
  if (!file || !selectedSession.value || !canSend()) return;
  const previewUrl = URL.createObjectURL(file);
  const localId = createLocalId("out");
  const contentType = detectContentType(file);
  const currentReply = replyTo.value ? { ...replyTo.value } : null;
  appendMessage({
    ...buildInboxUiMessageFromOutgoing(
      contentType,
      {
        url: previewUrl,
        name: file.name,
        size: file.size || 0,
        duration: 0,
        replyTo: currentReply,
      },
      { sid: selectedSession.value.sid, localId }
    ),
    status: UI_MESSAGE_STATUS.SENDING,
  });

  try {
    const uploaded = await uploadPickedFile(file, contentType);
    const remoteUrl = toAbsoluteMediaURL(uploaded.url);
    const idx = messages.value.findIndex((m) => m.local_id === localId);
    if (idx >= 0) {
      messages.value[idx].url = remoteUrl;
      if (contentType === BUSINESS_CONTENT_TYPES.IMAGE || contentType === BUSINESS_CONTENT_TYPES.AUDIO) {
        messages.value[idx].content = remoteUrl;
      }
      if (contentType === BUSINESS_CONTENT_TYPES.FILE) {
        messages.value[idx].name = uploaded.name || file.name;
        messages.value[idx].size = Number(uploaded.size || file.size || 0);
      }
      if (contentType === BUSINESS_CONTENT_TYPES.IMAGE) {
        URL.revokeObjectURL(previewUrl);
      }
    }

    let ok = false;
    const clientId = createLocalId("cid");
    if (contentType === BUSINESS_CONTENT_TYPES.IMAGE) {
      ok = Boolean(
        wsClient.value?.sendPayload(selectedSession.value.sid, BUSINESS_CONTENT_TYPES.IMAGE, {
          url: remoteUrl,
          name: uploaded.name || file.name,
          clientId,
          replyTo: currentReply,
        })?.ok
      );
    } else if (contentType === BUSINESS_CONTENT_TYPES.AUDIO) {
      ok = Boolean(
        wsClient.value?.sendPayload(selectedSession.value.sid, BUSINESS_CONTENT_TYPES.AUDIO, {
          url: remoteUrl,
          duration: 0,
          clientId,
          replyTo: currentReply,
        })?.ok
      );
    } else {
      ok = Boolean(
        wsClient.value?.sendPayload(selectedSession.value.sid, BUSINESS_CONTENT_TYPES.FILE, {
          url: remoteUrl,
          name: uploaded.name || file.name,
          size: Number(uploaded.size || file.size || 0),
          clientId,
          replyTo: currentReply,
        })?.ok
      );
    }

    if (!ok) {
      updateLocalMessageStatus(localId, UI_MESSAGE_STATUS.SENDING);
      scheduleMarkFailed(localId);
    } else {
      updateLocalMessageStatus(localId, UI_MESSAGE_STATUS.SENT);
      clearRetryTimer(localId);
    }
    replyTo.value = null;
  } catch (error) {
    updateLocalMessageStatus(localId, UI_MESSAGE_STATUS.FAILED);
    ElMessage.error(error.message || t("pageInbox.toast.uploadFailed"));
  } finally {
    if (contentType !== BUSINESS_CONTENT_TYPES.IMAGE) {
      URL.revokeObjectURL(previewUrl);
    }
  }
}

async function handlePickedFiles(fileList) {
  const files = Array.from(fileList || []);
  for (const file of files) {
    await sendFileMessage(file);
  }
}

async function onPickedFile(event) {
  const files = event.target.files;
  event.target.value = "";
  if (!files || files.length === 0) return;
  await handlePickedFiles(files);
}

function handleSenderPaste(event) {
  if (!canSend()) return;
  const files = extractClipboardFiles(event);
  if (files.length === 0) return;
  event.preventDefault();
  showEmojiPicker.value = false;
  void handlePickedFiles(files);
}

function retryMessage(msg) {
  if (!selectedSession.value || !msg || !msg.local_id) return;
  updateLocalMessageStatus(msg.local_id, UI_MESSAGE_STATUS.SENDING);
  const retryReplyTo = msg.replyTo || null;

  let ok = false;
  const clientId = createLocalId("cid");
  if (msg.content_type === BUSINESS_CONTENT_TYPES.IMAGE) {
    ok = Boolean(
      wsClient.value?.sendPayload(selectedSession.value.sid, BUSINESS_CONTENT_TYPES.IMAGE, {
        url: msg.url || msg.content,
        name: msg.name || "image.jpg",
        clientId,
        replyTo: retryReplyTo,
      })?.ok
    );
  } else if (msg.content_type === BUSINESS_CONTENT_TYPES.AUDIO) {
    ok = Boolean(
      wsClient.value?.sendPayload(selectedSession.value.sid, BUSINESS_CONTENT_TYPES.AUDIO, {
        url: msg.url || msg.content,
        duration: Number(msg.duration || 0),
        clientId,
        replyTo: retryReplyTo,
      })?.ok
    );
  } else if (msg.content_type === BUSINESS_CONTENT_TYPES.FILE) {
    ok = Boolean(
      wsClient.value?.sendPayload(selectedSession.value.sid, BUSINESS_CONTENT_TYPES.FILE, {
        url: msg.url || msg.content,
        name: msg.name || "file",
        size: Number(msg.size || 0),
        clientId,
        replyTo: retryReplyTo,
      })?.ok
    );
  } else {
    ok = Boolean(
      wsClient.value?.sendPayload(selectedSession.value.sid, BUSINESS_CONTENT_TYPES.TEXT, {
        content: msg.content || "",
        clientId,
        replyTo: retryReplyTo,
      })?.ok
    );
  }

  if (!ok) {
    scheduleMarkFailed(msg.local_id);
  } else {
    updateLocalMessageStatus(msg.local_id, UI_MESSAGE_STATUS.SENT);
    clearRetryTimer(msg.local_id);
  }
}

function bindAudio(msg, el) {
  const id = msg.msg_id || msg.local_id;
  if (!id) return;
  if (el) audioMap.set(id, el);
  else audioMap.delete(id);
}

async function readAudioDurationFromURL(url) {
  return await new Promise((resolve) => {
    const audio = new Audio(url);
    audio.preload = "metadata";
    audio.onloadedmetadata = () => {
      const d = Number(audio.duration || 0);
      resolve(Number.isFinite(d) ? d : 0);
    };
    audio.onerror = () => resolve(0);
  });
}

function blobToFile(blob, fileName, mimeType = "audio/webm") {
  return new window.File([blob], fileName, { type: mimeType });
}

async function uploadVoiceBlob(blob, mimeType) {
  if (!selectedSession.value || !blob) return;
  const fileExt = mimeType.includes("ogg") ? "ogg" : mimeType.includes("mp4") || mimeType.includes("m4a") ? "m4a" : "webm";
  const fileName = `voice_${Date.now()}.${fileExt}`;
  const file = blobToFile(blob, fileName, mimeType);
  const previewURL = URL.createObjectURL(blob);
  const duration = Math.max((Date.now() - recordingStartAtRef.value) / 1000, await readAudioDurationFromURL(previewURL));
  const localId = createLocalId("out");
  const currentReply = replyTo.value ? { ...replyTo.value } : null;
  appendMessage({
    ...normalizeMessageMediaFields(buildInboxUiMessageFromOutgoing(
      BUSINESS_CONTENT_TYPES.AUDIO,
      { url: previewURL, duration, replyTo: currentReply },
      { sid: selectedSession.value.sid, localId }
    )),
    status: UI_MESSAGE_STATUS.SENDING,
  });

  try {
    const uploaded = await uploadPickedFile(file, BUSINESS_CONTENT_TYPES.AUDIO);
    const remoteUrl = toAbsoluteMediaURL(uploaded.url);
    const clientId = createLocalId("cid");
    const idx = messages.value.findIndex((m) => m.local_id === localId);
    if (idx >= 0) {
      messages.value[idx].url = remoteUrl;
      messages.value[idx].content = remoteUrl;
      messages.value[idx].duration = Number(duration || 0);
    }
    const ok = Boolean(
      wsClient.value?.sendPayload(selectedSession.value.sid, BUSINESS_CONTENT_TYPES.AUDIO, {
        url: remoteUrl,
        duration: Number(duration || 0),
        clientId,
        replyTo: currentReply,
      })?.ok
    );
    if (!ok) {
      scheduleMarkFailed(localId);
    } else {
      updateLocalMessageStatus(localId, UI_MESSAGE_STATUS.SENT);
      clearRetryTimer(localId);
    }
    replyTo.value = null;
  } catch (error) {
    updateLocalMessageStatus(localId, UI_MESSAGE_STATUS.FAILED);
    ElMessage.error(error.message || t("pageInbox.toast.audioSendFailed"));
  } finally {
    URL.revokeObjectURL(previewURL);
  }
}

function clearRecordingResources() {
  if (recordingStreamRef.value) {
    for (const track of recordingStreamRef.value.getTracks()) {
      track.stop();
    }
  }
  recordingStreamRef.value = null;
  mediaRecorderRef.value = null;
  audioChunksRef.value = [];
  recordingStartAtRef.value = 0;
  isRecording.value = false;
}

async function startRecording() {
  if (!isRecordSupported.value || isRecording.value || !canSend()) return;
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    const recorder = new MediaRecorder(stream);
    recordingStreamRef.value = stream;
    mediaRecorderRef.value = recorder;
    audioChunksRef.value = [];
    recordingStartAtRef.value = Date.now();
    isRecording.value = true;

    recorder.ondataavailable = (event) => {
      if (event?.data?.size > 0) {
        audioChunksRef.value.push(event.data);
      }
    };
    recorder.onstop = async () => {
      const mimeType = recorder.mimeType || "audio/webm";
      const blob = new Blob(audioChunksRef.value, { type: mimeType });
      if (blob.size > 0) {
        await uploadVoiceBlob(blob, mimeType);
      }
      clearRecordingResources();
    };
    recorder.start();
  } catch (error) {
    clearRecordingResources();
    ElMessage.error(error?.message || t("pageInbox.toast.recordStartFailed"));
  }
}

function stopRecording(send = true) {
  const recorder = mediaRecorderRef.value;
  if (!recorder) return;
  if (recorder.state === "recording") {
    if (send) {
      recorder.stop();
    } else {
      recorder.onstop = null;
      recorder.stop();
      clearRecordingResources();
    }
  }
}

function handlePressRecordStart() {
  showEmojiPicker.value = false;
  void startRecording();
}

function handlePressRecordStop() {
  if (!isRecording.value) return;
  stopRecording(true);
}

function handlePressRecordCancel() {
  if (!isRecording.value) return;
  stopRecording(false);
}

function onAudioEnded(msg) {
  const id = msg.msg_id || msg.local_id;
  if (id === playingAudioId.value) {
    playingAudioId.value = "";
  }
}

async function toggleAudio(msg) {
  const id = msg.msg_id || msg.local_id;
  const el = audioMap.get(id);
  if (!el) return;
  if (playingAudioId.value === id && !el.paused) {
    el.pause();
    playingAudioId.value = "";
    return;
  }
  for (const [key, audio] of audioMap.entries()) {
    if (!audio.paused) {
      audio.pause();
      audio.currentTime = 0;
    }
    if (key !== id && playingAudioId.value === key) {
      playingAudioId.value = "";
    }
  }
  playingAudioId.value = id;
  try {
    await el.play();
  } catch {
    playingAudioId.value = "";
  }
}

function stopAudio(msg) {
  const id = msg?.msg_id || msg?.local_id;
  if (!id) return;
  const el = audioMap.get(id);
  if (!el) return;
  el.pause();
  el.currentTime = 0;
  if (playingAudioId.value === id) {
    playingAudioId.value = "";
  }
}

function getFileExtension(msg) {
  const name = msg.name || msg.content || "";
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
      msg._isGood = true;
      msg._isBad = false;
      break;
    case "bad":
      msg._isGood = false;
      msg._isBad = true;
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

async function loadSnippets() {
  snippetLoading.value = true;
  try {
    const resp = await api.listQuickReplies();
    snippets.value = resp?.data?.data?.data || [];
  } catch (error) {
    snippets.value = [];
    ElMessage.error(error?.message || t("pageInbox.toast.loadSnippetsFailed"));
  } finally {
    snippetLoading.value = false;
  }
}

async function loadAppOptions() {
  try {
    const appSet = new Set();
    for (const session of sessions.value) {
      const appID = String(session?.app_id || "").trim();
      if (appID) {
        appSet.add(appID);
      }
    }
    const userResp = await api.getUserInfo();
    const rawApps = userResp?.data?.data?.user?.apps;
    let apps = [];
    if (Array.isArray(rawApps)) {
      apps = rawApps.map((v) => String(v || "").trim()).filter(Boolean);
    } else if (typeof rawApps === "string" && rawApps.trim()) {
      try {
        const parsed = JSON.parse(rawApps);
        if (Array.isArray(parsed)) {
          apps = parsed.map((v) => String(v || "").trim()).filter(Boolean);
        }
      } catch {
        apps = [];
      }
    }
    const hasAll = apps.some((v) => v.toLowerCase() === "all");
    if (apps.length > 0 && !hasAll) {
      for (const appID of apps) {
        appSet.add(appID);
      }
      appOptions.value = Array.from(appSet).map((appID) => ({
        label: appID,
        value: appID,
      }));
      return;
    }
    if (store.user?.role === "admin") {
      const appResp = await api.listApps({ page: 1, page_size: 500 });
      const rows = appResp?.data?.data?.data || [];
      for (const app of rows) {
        const appID = String(app?.app_id || "").trim();
        if (appID) {
          appSet.add(appID);
        }
      }
      appOptions.value = rows.map((app) => ({
        label: `${app.name} (${app.app_id})`,
        value: app.app_id,
      }));
      if (appOptions.value.length === 0 && appSet.size > 0) {
        appOptions.value = Array.from(appSet).map((appID) => ({ label: appID, value: appID }));
      }
      return;
    }
    if (hasAll && appSet.size > 0) {
      appOptions.value = Array.from(appSet).map((appID) => ({ label: appID, value: appID }));
      return;
    }
    appOptions.value = Array.from(appSet).map((appID) => ({ label: appID, value: appID }));
  } catch (error) {
    const appSet = new Set();
    for (const session of sessions.value) {
      const appID = String(session?.app_id || "").trim();
      if (appID) {
        appSet.add(appID);
      }
    }
    appOptions.value = Array.from(appSet).map((appID) => ({ label: appID, value: appID }));
  }
}

async function useSnippet(snippet) {
  inputMessage.value = snippet.content || "";
  openSnippetDialog.value = false;
  await api.useQuickReply(snippet.id).catch(() => { });
  if (canControlSession.value && String(inputMessage.value || "").trim()) {
    sendText();
  }
}

function onIncomingMessage(message) {
  if (!message || !message.sid) return;
  const sid = message.sid;
  if (message.businessType === BUSINESS_MESSAGE_TYPES.TYPING) {
    if (selectedSession.value?.sid === sid) {
      isVisitorTyping.value = true;
      const oldTimer = typingTimers.get(sid);
      if (oldTimer) clearTimeout(oldTimer);
      const timer = setTimeout(() => {
        isVisitorTyping.value = false;
        typingTimers.delete(sid);
      }, 2200);
      typingTimers.set(sid, timer);
    }
    return;
  }
  const mapped = normalizeMessageMediaFields(buildInboxUiMessageFromBusiness(message, { localId: createLocalId("ws") }));
  const streamEnabled = Boolean(message?.payload?.raw?.stream);
  const streamDelta = Boolean(message?.payload?.raw?.stream_delta);
  const streamFinal = Boolean(message?.payload?.raw?.stream_final);
  const streamKey = String(message?.payload?.raw?.stream_key || "").trim();
  if (streamEnabled && streamKey && selectedSession.value?.sid === sid) {
    const streamLocalID = `stream_${streamKey}`;
    const streamText = String(message?.payload?.content || "").trim();
    if (streamDelta && streamText) {
      const idx = messages.value.findIndex((m) => m.local_id === streamLocalID);
      if (idx >= 0) {
        messages.value[idx] = {
          ...messages.value[idx],
          content: streamText,
          timestamp: Number(message.timestamp || Math.floor(Date.now() / 1000)),
          status: UI_MESSAGE_STATUS.SENT,
        };
      } else {
        appendMessage({
          local_id: streamLocalID,
          msg_id: "",
          business_type: BUSINESS_MESSAGE_TYPES.AGENT,
          content_type: BUSINESS_CONTENT_TYPES.TEXT,
          content: streamText,
          timestamp: Number(message.timestamp || Math.floor(Date.now() / 1000)),
          isSelf: true,
          sid,
          status: UI_MESSAGE_STATUS.SENT,
        });
      }
      return;
    }
    if (streamFinal && mapped) {
      const idx = messages.value.findIndex((m) => m.local_id === streamLocalID);
      if (idx >= 0) {
        messages.value[idx] = {
          ...mapped,
          local_id: streamLocalID,
          status: UI_MESSAGE_STATUS.SENT,
        };
      } else {
        appendMessage({ ...mapped, local_id: streamLocalID, status: UI_MESSAGE_STATUS.SENT });
      }
      return;
    }
  }
  const payloadClientID = String(message?.payload?.clientId || message?.payload?.client_id || "").trim();
  const payloadCode = String(message?.payload?.code || "").trim();
  if (payloadCode === "visitor_offline_blocked") {
    if (payloadClientID) {
      const pending = messages.value.find(
        (m) => m.isSelf && m.client_id === payloadClientID && m.status === UI_MESSAGE_STATUS.SENDING
      );
      if (pending?.local_id) {
        updateLocalMessageStatus(pending.local_id, UI_MESSAGE_STATUS.FAILED);
        clearRetryTimer(pending.local_id);
      }
    }
    ElMessage.warning(t("pageInbox.toast.visitorOffline"));
    return;
  }
  if (payloadCode === "session_closed_blocked") {
    ElMessage.warning(t("pageInbox.toast.sessionClosedNoSend"));
    if (selectedSession.value?.sid === sid) {
      selectedSession.value = { ...selectedSession.value, status: "closed" };
    }
    const target = sessions.value.find((s) => s.sid === sid);
    if (target) {
      target.status = "closed";
    }
    return;
  }
  if (!mapped) {
    return;
  }
  const mappedComparable = String(
    mapped.content_type === BUSINESS_CONTENT_TYPES.FILE ? (mapped.url || mapped.content || "")
      : mapped.content_type === BUSINESS_CONTENT_TYPES.AUDIO ? (mapped.url || mapped.content || "")
        : mapped.content_type === BUSINESS_CONTENT_TYPES.IMAGE ? (mapped.url || mapped.content || "")
          : (mapped.content || "")
  ).trim();
  if (mapped.business_type === BUSINESS_MESSAGE_TYPES.AGENT && mapped.isSelf) {
    for (let i = messages.value.length - 1; i >= 0; i--) {
      const local = messages.value[i];
      if (!local || !local.isSelf || local.msg_id || local.status !== UI_MESSAGE_STATUS.SENDING) {
        continue;
      }
      const localComparable = String(
        local.content_type === BUSINESS_CONTENT_TYPES.FILE ? (local.url || local.content || "")
          : local.content_type === BUSINESS_CONTENT_TYPES.AUDIO ? (local.url || local.content || "")
            : local.content_type === BUSINESS_CONTENT_TYPES.IMAGE ? (local.url || local.content || "")
              : (local.content || "")
      ).trim();
      if (
        (payloadClientID && local.client_id === payloadClientID) ||
        (local.content_type === mapped.content_type && localComparable && localComparable === mappedComparable)
      ) {
        local.status = UI_MESSAGE_STATUS.SENT;
        local.msg_id = mapped.msg_id || local.msg_id;
        clearRetryTimer(local.local_id);
        return;
      }
    }
  }

  const idx = sessions.value.findIndex((s) => s.sid === sid);
  if (idx < 0) {
    // Refresh session queue immediately when message belongs to a session not in current list.
    void reloadSessions();
  }
  if (idx >= 0) {
    sessions.value[idx].last_message = toInboxDisplayText(mapped);
    sessions.value[idx].last_message_type = mapped.content_type;
    if (mapped.business_type === BUSINESS_MESSAGE_TYPES.VISITOR && selectedSession.value?.sid !== sid) {
      sessions.value[idx].unread_count = Number(sessions.value[idx].unread_count || 0) + 1;
    }
  }

  if (selectedSession.value?.sid === sid) {
    appendMessage(mapped);
    if (canControlSession.value) {
      api.readSession(sid).catch(() => { });
    }
    if (idx >= 0) {
      sessions.value[idx].unread_count = 0;
      sessions.value[idx].status = sessions.value[idx].status === "unread" ? "assigned" : sessions.value[idx].status;
    }
  }

  if (mapped.business_type === BUSINESS_MESSAGE_TYPES.VISITOR && selectedSession.value?.sid !== sid) {
    maybePlaySound();
    maybeDesktopNotify(sid, mapped);
  }

  if (mapped.business_type === BUSINESS_MESSAGE_TYPES.AGENT && mapped.isSelf) {
    for (let i = messages.value.length - 1; i >= 0; i--) {
      const local = messages.value[i];
      if (
        !local.msg_id &&
        local.isSelf &&
        local.status === UI_MESSAGE_STATUS.SENDING &&
        (
          (payloadClientID && local.client_id === payloadClientID) ||
          (local.content_type === mapped.content_type && local.content === mapped.content)
        )
      ) {
        local.status = UI_MESSAGE_STATUS.SENT;
        clearRetryTimer(local.local_id);
        break;
      }
    }
  }
}

function maybePlaySound() {
  if (!agentSettings.value.soundEnabled) return;
  const ctx = window.AudioContext ? new AudioContext() : null;
  if (!ctx) return;
  const osc = ctx.createOscillator();
  const gain = ctx.createGain();
  osc.type = "sine";
  osc.frequency.setValueAtTime(880, ctx.currentTime);
  gain.gain.setValueAtTime(0.0001, ctx.currentTime);
  gain.gain.exponentialRampToValueAtTime(0.03, ctx.currentTime + 0.01);
  gain.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + 0.18);
  osc.connect(gain);
  gain.connect(ctx.destination);
  osc.start();
  osc.stop(ctx.currentTime + 0.2);
}

async function maybeDesktopNotify(sid, msg) {
  if (!agentSettings.value.desktopNotifyEnabled) return;
  if (typeof window === "undefined" || !("Notification" in window)) return;
  const now = Date.now();
  const prev = Number(lastNotifyAtBySid.get(sid) || 0);
  if (now - prev < 1500) return;
  lastNotifyAtBySid.set(sid, now);

  if (Notification.permission === "default") {
    await Notification.requestPermission().catch(() => { });
  }
  if (Notification.permission !== "granted") return;

  const title = t("pageInbox.notify.newVisitorTitle");
  const body = msg?.content_type === BUSINESS_CONTENT_TYPES.TEXT ? String(msg.content || "") : toInboxDisplayText(msg);
  const n = new Notification(title, { body: body || t("pageInbox.notify.newMessageBody") });
  setTimeout(() => n.close(), 5000);
}

async function loadAgentSettings() {
  try {
    const resp = await api.getAgentSettings();
    const data = resp?.data?.data || {};
    agentSettings.value = {
      soundEnabled: data.soundEnabled !== false,
      desktopNotifyEnabled: Boolean(data.desktopNotifyEnabled),
      typingIndicatorEnabled: data.typingIndicatorEnabled !== false,
      enterToSend: data.enterToSend !== false,
      aiEnabled: Boolean(data.aiEnabled),
    };
  } catch {
    agentSettings.value = {
      soundEnabled: true,
      desktopNotifyEnabled: false,
      typingIndicatorEnabled: true,
      enterToSend: true,
      aiEnabled: false,
    };
  }
}

async function applyAISuggest() {
  if (!selectedSession.value || !canControlSession.value) return;
  aiSuggestLoading.value = true;
  try {
    const resp = await api.suggestAIReply(selectedSession.value.sid);
    const text = String(resp?.data?.data?.suggestion || "").trim();
    const source = String(resp?.data?.data?.source || "").trim();
    if (!text) {
      ElMessage.warning(t("pageInbox.toast.noSuggestion"));
      return;
    }
    inputMessage.value = text;
    if (source === "rag-vector-eino" || source === "rag-vittoriadb-eino") {
      ElMessage.success(t("pageInbox.toast.aiSuggestionApplied"));
    } else if (source === "rag-local-fallback") {
      ElMessage.warning(t("pageInbox.toast.vectorNotReadyFallback"));
    } else if (source === "rule-fallback") {
      ElMessage.warning(t("pageInbox.toast.knowledgeUnavailableFallback"));
    } else {
      ElMessage.success(t("pageInbox.toast.aiSuggestionApplied"));
    }
  } catch (error) {
    ElMessage.error(error.message || t("pageInbox.toast.aiSuggestFailed"));
  } finally {
    aiSuggestLoading.value = false;
  }
}

function initWS() {
  wsClient.value?.disconnect();
  wsClient.value = new AgentWSClient(store.token, {
    apiBaseUrl: api.baseURL.replace("/api/v1", ""),
    onMessage: onIncomingMessage,
    onStatus: (status) => {
      wsStatus.value = status;
    },
    onError: (error) => {
      console.error(error);
    },
  });
  wsClient.value.connect();
}

function syncChatWideState() {
  const width = Number(inboxChatRef.value?.clientWidth || 0);
  isChatWide.value = width >= 800;
}

onMounted(async () => {
  initWS();
  syncChatWideState();
  if (typeof ResizeObserver !== "undefined") {
    inboxChatResizeObserver = new ResizeObserver(() => {
      syncChatWideState();
    });
    if (inboxChatRef.value) {
      inboxChatResizeObserver.observe(inboxChatRef.value);
    }
  } else {
    window.addEventListener("resize", syncChatWideState);
  }
  await loadAgentSettings();
  await reloadSessions();
  await loadAppOptions();
  await loadSnippets();
  const pendingSnippet = localStorage.getItem("kefu_snippet_to_inbox");
  if (pendingSnippet) {
    inputMessage.value = pendingSnippet;
    localStorage.removeItem("kefu_snippet_to_inbox");
  }
});

onBeforeUnmount(() => {
  wsClient.value?.disconnect();
  if (inboxChatResizeObserver) {
    inboxChatResizeObserver.disconnect();
    inboxChatResizeObserver = null;
  } else {
    window.removeEventListener("resize", syncChatWideState);
  }
  for (const timer of typingTimers.values()) clearTimeout(timer);
  typingTimers.clear();
  for (const t of retryTimers.values()) clearTimeout(t);
  retryTimers.clear();
  clearRecordingResources();
});
</script>

<style scoped>
.inbox-page {
  height: 100%;
  display: grid;
  grid-template-columns: 228px 1fr;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  gap: 1px;
  overflow: hidden;
}

@media (max-width: 1200px) {
  .inbox-page {
    grid-template-columns: 208px 1fr;
  }
}

.inbox-sessions {
  border-right: 1px solid rgba(229, 231, 235, 0.6);
  background: linear-gradient(180deg, #ffffff 0%, #fafbfc 100%);
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.02);
}

.inbox-side-head {
  padding: 1rem 1rem 0.875rem;
  border-bottom: 1px solid rgba(229, 231, 235, 0.5);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: linear-gradient(180deg, #ffffff 0%, #fafbfc 100%);
  flex-shrink: 0;
}

.inbox-side-head h2 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: #1e293b;
}

.inbox-side-head p {
  margin: 0.25rem 0 0;
  font-size: 0.75rem;
  color: #64748b;
}

.inbox-filters {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  padding: 0.65rem 0.75rem;
  border-bottom: 1px solid rgba(229, 231, 235, 0.5);
  background: #fafbfc;
  flex-shrink: 0;
}

.inbox-filters-expand {
  padding-top: 0.35rem;
}

.filter-toggle-btn {
  border: 1px solid #dbeafe;
  border-radius: 8px;
  height: 28px;
  background: #eff6ff;
  color: #1d4ed8;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.filter-toggle-btn:hover {
  background: #dbeafe;
}

.inbox-filter-row {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.4rem;
}

.inbox-filter-row>* {
  min-width: 0;
}

.inbox-filter-row .el-select {
  width: 100%;
}

.inbox-filter-row .el-date-editor {
  width: 100%;
}

.session-list {
  overflow-y: auto;
  overflow-x: hidden;
  flex: 1;
  min-height: 0;
  padding: 0.5rem;
}

.session-row {
  width: 100%;
  border: 1px solid transparent;
  background: transparent;
  display: flex;
  justify-content: space-between;
  padding: 0.875rem 0.75rem;
  border-radius: 0.75rem;
  cursor: pointer;
  text-align: left;
  transition: all 0.15s ease;
  margin-bottom: 0.25rem;
}

.session-row:hover {
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  border-color: rgba(229, 231, 235, 0.6);
}

.session-row.active {
  background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
  border-color: #93c5fd;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.1);
}

.session-main {
  min-width: 0;
}

.session-main .visitor {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: #1e293b;
}

.session-main .last-msg {
  margin: 0.25rem 0 0;
  font-size: 0.75rem;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 126px;
}

.session-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.375rem;
}

.unread {
  min-width: 1.25rem;
  height: 1.25rem;
  border-radius: 9999px;
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  color: #ffffff;
  font-size: 0.625rem;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 0.375rem;
  box-shadow: 0 1px 2px rgba(239, 68, 68, 0.3);
}

.inbox-chat {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  background: #ffffff;
}

.chat-head {
  background: radial-gradient(circle at left top, #ffffff 0%, #f8fbff 58%, #f6f9ff 100%);
  border-bottom: 1px solid rgba(229, 231, 235, 0.5);
  padding: 1rem 1.25rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.visitor-trigger {
  border: 0;
  background: transparent;
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  cursor: pointer;
  text-align: left;
  padding: 0;
}

.visitor-trigger-avatar {
  width: 36px;
  height: 36px;
  border-radius: 999px;
  border: 1px solid #dbeafe;
  background: #f8fafc;
}

.visitor-trigger-main {
  display: inline-flex;
  flex-direction: column;
  gap: 0.125rem;
}

.visitor-trigger-main strong {
  font-size: 14px;
  line-height: 1.2;
  color: #1e293b;
}

.visitor-trigger-main small {
  font-size: 12px;
  line-height: 1.2;
  color: #64748b;
}

.visitor-profile {
  color: #94a3b8 !important;
}

.chat-actions {
  display: flex;
  gap: 0.5rem;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0;
  padding: 1rem 1.25rem;
  background:
    radial-gradient(circle at 14% 8%, rgba(59, 130, 246, 0.07) 0%, rgba(59, 130, 246, 0) 36%),
    radial-gradient(circle at 84% 88%, rgba(16, 185, 129, 0.06) 0%, rgba(16, 185, 129, 0) 40%),
    linear-gradient(180deg, #f8fafc 0%, #ffffff 100%);
}

.load-more {
  width: 100%;
  border: 1px solid #e2e8f0;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
  color: #475569;
  border-radius: 0.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.75rem;
  margin-bottom: 0.75rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.load-more:hover {
  border-color: #cbd5e1;
  background: #f1f5f9;
}

.msg-row {
  display: flex;
  flex-direction: column;
  margin-bottom: 0.75rem;
  align-items: flex-start;
  gap: 0.2rem;
  width: 100%;
}

.msg-row.self {
  align-items: flex-end;
}

.msg-avatar {
  width: 28px;
  height: 28px;
  border-radius: 999px;
  border: 1px solid #dbeafe;
  background: #f8fafc;
  flex-shrink: 0;
}

.msg-head {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.msg-head.self {
  justify-content: flex-end;
}

.msg-main {
  display: flex;
  flex-direction: column;
  max-width: min(82%, 860px);
  min-width: 0;
}

.msg-rich-content {
  display: inline-flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  width: fit-content;
  max-width: 100%;
  min-width: 0;
}

.msg-rich-content.self {
  align-items: flex-start;
}

.msg-name {
  margin: 0;
  font-size: 11px;
  line-height: 1.2;
  color: #64748b;
}

.msg-row.self .msg-main {
  align-items: flex-end;
}

.msg-row.self .msg-name {
  display: none;
}

.msg-bubble {
  display: inline-block;
  width: fit-content;
  max-width: 100%;
  border-radius: 1rem;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  color: #1e293b;
  padding: 0.625rem 0.875rem;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  font-size: 13px;
  line-height: 1.45;
}

.msg-bubble.self {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 68%, #1e40af 100%);
  border-color: transparent;
  color: #ffffff;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.25);
}

.msg-row.self .msg-bubble {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 68%, #1e40af 100%);
  border-color: transparent;
  color: #ffffff;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.25);
}

.msg-reply-quote {
  border-left: 2px solid #93c5fd;
  background: rgba(241, 245, 249, 0.96);
  border-radius: 8px;
  padding: 6px 10px;
  width: 100%;
  box-sizing: border-box;
  margin: 0;
}

.msg-reply-quote.self {
  border-left-color: #60a5fa;
  background: rgba(255, 255, 255, 0.82);
}

.msg-reply-quote-head {
  margin: 0;
  font-size: 11px;
  color: #1d4ed8;
  line-height: 1.35;
}

.msg-rich-content.self .msg-reply-quote-head {
  color: #1d4ed8;
}

.msg-row.self .msg-reply-quote-head {
  color: #dbeafe;
}

.msg-reply-quote-preview {
  margin: 1px 0 0;
  font-size: 11px;
  color: #334155;
  line-height: 1.45;
}

.msg-rich-content.self .msg-reply-quote-preview {
  color: #1e293b;
}

.msg-row.self .msg-reply-quote-preview {
  color: #dbeafe;
}

.msg-image-wrap {
  display: inline-flex;
  border: 0;
  padding: 0;
  background: transparent;
  cursor: zoom-in;
}

.msg-image {
  display: block;
  max-width: 280px;
  border-radius: 0.75rem;
}

.msg-content {
  font-size: 13px;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.markdown-content :deep(p) {
  margin: 0 0 4px;
}

.markdown-content :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-content :deep(pre) {
  margin: 4px 0;
  white-space: pre-wrap;
  background: rgba(15, 23, 42, 0.06);
  border-radius: 8px;
  padding: 8px;
  font-size: 12px;
  overflow-x: hidden;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.markdown-content :deep(code) {
  background: rgba(15, 23, 42, 0.08);
  border-radius: 4px;
  padding: 1px 4px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.markdown-content :deep(a) {
  color: #2563eb;
  text-decoration: underline;
  word-break: break-all;
}

.msg-markdown-content {
  max-width: 100%;
  min-width: 0;
}

.msg-markdown-content p {
  margin: 0 0 4px;
}

.msg-markdown-content p:last-child {
  margin-bottom: 0;
}

.msg-markdown-content pre {
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

.msg-markdown-content code {
  background: rgba(15, 23, 42, 0.08);
  border-radius: 4px;
  padding: 1px 4px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.msg-markdown-content a {
  color: #2563eb;
  text-decoration: underline;
  word-break: break-all;
}

.msg-audio-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid #c7dcff;
  background: #eff6ff;
  border-radius: 12px;
  padding: 7px 9px;
  max-width: 260px;
}

.msg-audio-pill.self {
  background: rgba(219, 234, 254, 0.22);
  border-color: rgba(191, 219, 254, 0.55);
}

.msg-audio-symbol {
  color: #2563eb;
  flex-shrink: 0;
}

.msg-rich-content.self .msg-audio-symbol {
  color: #dbeafe;
}

.msg-row.self .msg-audio-symbol {
  color: #dbeafe;
}

.msg-audio-icon-btn {
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

.msg-rich-content.self .msg-audio-icon-btn {
  color: #dbeafe;
}

.msg-row.self .msg-audio-icon-btn {
  color: #dbeafe;
}

.msg-audio-icon-btn:hover {
  background: rgba(37, 99, 235, 0.12);
}

.msg-audio-icon-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.msg-audio-duration {
  margin-left: auto;
  font-size: 12px;
  color: #4b5563;
}

.msg-rich-content.self .msg-audio-duration {
  color: #dbeafe;
}

.msg-row.self .msg-audio-duration {
  color: #dbeafe;
}

.msg-hidden-audio {
  display: none;
}

.file-link {
  color: inherit;
  text-decoration: underline;
  text-underline-offset: 2px;
  font-size: 13px;
  line-height: 1.45;
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

:deep(.t-chat-message),
:deep(.t-chat__item__inner) {
  width: 100%;
  display: flex !important;
  flex-direction: row !important;
  align-items: flex-start !important;
  justify-content: flex-start !important;
  gap: 10px !important;
  margin-bottom: 12px;
}

:deep(.t-chat-message--user),
:deep(.t-chat__item__role--user) {
  flex-direction: row !important;
  justify-content: flex-start !important;
}

:deep(.t-chat-message__avatar),
:deep(.t-chat__item__avatar) {
  flex-shrink: 0 !important;
  margin: 0 !important;
  padding: 0 !important;
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
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  flex: 0 1 auto;
  width: auto;
  max-width: 100%;
  min-width: 0;
}

:deep(.t-chat-message--user .t-chat-message__main),
:deep(.t-chat__item__role--user .t-chat__item__main) {
  align-items: flex-start;
}

:deep(.t-chat-message--user .t-chat-message__content),
:deep(.t-chat__item__role--user .t-chat__item__content) {
  align-items: flex-start;
}

:deep(.t-chat__item__detail) {
  max-width: 100%;
  overflow-x: auto;
  word-break: break-word;
  overflow-wrap: anywhere;
}

:deep(.t-chat-markdown) {
  max-width: 100%;
  overflow-x: auto;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.inbox-chat.chat-wide :deep(.t-chat-message--user),
.inbox-chat.chat-wide :deep(.t-chat__item__role--user) {
  flex-direction: row-reverse !important;
  justify-content: flex-start !important;
}

.inbox-chat.chat-wide :deep(.t-chat-message__main),
.inbox-chat.chat-wide :deep(.t-chat__item__main) {
  max-width: min(82%, 860px);
}

.inbox-chat.chat-wide :deep(.t-chat-message--user .t-chat-message__main),
.inbox-chat.chat-wide :deep(.t-chat__item__role--user .t-chat__item__main) {
  align-items: flex-end;
  margin-left: auto;
}

.inbox-chat.chat-wide :deep(.t-chat-message--user .t-chat-message__content),
.inbox-chat.chat-wide :deep(.t-chat__item__role--user .t-chat__item__content) {
  align-items: flex-end;
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

.kefu-input-toolbar {
  display: flex;
  gap: 0;
  padding: 4px 0;
  background: transparent;
  border: none;
  flex-wrap: nowrap;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: thin;
}

.kefu-tool-btn {
  flex: 0 0 auto;
  border: none;
  background: transparent;
  color: #5b6b7c;
  border-radius: 8px;
  padding: 8px 8px;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: all 0.2s ease;
  position: relative;
  white-space: nowrap;
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
  flex-shrink: 0;
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

.msg-meta-line {
  margin-top: 3px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  line-height: 1;
  color: #94a3b8;
}

.msg-meta-line.self {
  justify-content: flex-end;
}

.msg-time {
  margin: 0;
  font-size: 11px;
  color: #94a3b8;
}

.msg-dot {
  color: #cbd5e1;
}

.msg-status {
  margin: 0;
  font-size: 11px;
  color: #94a3b8;
}

.msg-status-failed {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: #dc2626;
}

.retry-btn {
  margin-top: 0;
  border: 0;
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
  color: #ffffff;
  border-radius: 0.375rem;
  padding: 0.125rem 0.5rem;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.retry-btn:hover {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
}

.msg-reply-btn {
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

.msg-reply-btn:hover {
  color: #2563eb;
}

.chat-input-wrap {
  border-top: 1px solid rgba(229, 231, 235, 0.5);
  background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
  padding: 0.75rem 1rem;
  flex-shrink: 0;
}

.reply-bar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  min-height: 52px;
  padding: 8px 10px;
  border: 1px solid #dbeafe;
  border-radius: 10px;
  margin-bottom: 0.5rem;
  background: #f8fbff;
}

.reply-bar-text {
  flex: 1;
  min-width: 0;
}

.reply-bar-head {
  margin: 0;
  font-size: 12px;
  line-height: 1.35;
  color: #1e40af;
}

.reply-bar-preview {
  margin: 2px 0 0;
  font-size: 12px;
  line-height: 1.35;
  color: #64748b;
  white-space: normal;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.reply-bar-cancel {
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

.typing-tip {
  margin-bottom: 0.5rem;
  font-size: 0.75rem;
  color: #64748b;
  font-style: italic;
}

.toolbar {
  display: flex;
  gap: 0.25rem;
  margin-bottom: 0.5rem;
  flex-wrap: wrap;
}

.toolbar-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.8125rem;
  font-weight: 500;
  padding: 0.34rem 0.62rem;
  border-radius: 0.5rem;
  transition: all 0.15s ease;
}

.toolbar-btn:hover {
  background: rgba(59, 130, 246, 0.06);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.toolbar-btn.is-recording {
  background: #fee2e2;
  color: #b91c1c;
  border-color: #fecaca;
}

.emoji-panel {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  padding: 0.75rem;
  max-height: min(48vh, 340px);
  overflow-y: auto;
}

.emoji-mask {
  position: fixed;
  inset: 0;
  z-index: 9998;
  background: rgba(15, 23, 42, 0.36);
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding: 10px;
}

.emoji-dialog {
  width: min(560px, 96vw);
  background: #ffffff;
  border: 1px solid #dbe3ef;
  border-radius: 12px;
  box-shadow: 0 22px 48px rgba(15, 23, 42, 0.26);
  padding: 10px;
}

.emoji-dialog-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 13px;
  font-weight: 600;
  color: #334155;
}

.emoji-close-btn {
  border: 0;
  border-radius: 8px;
  width: 28px;
  height: 28px;
  background: #f1f5f9;
  color: #475569;
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
}

.emoji-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.emoji-group-title {
  margin: 0;
  font-size: 11px;
  color: #64748b;
}

.emoji-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.35rem;
}

.emoji-btn {
  width: 100%;
  height: 30px;
  border: 1px solid #dbeafe;
  background: #f8fbff;
  border-radius: 0.45rem;
  font-size: 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.emoji-btn:hover {
  background: rgba(59, 130, 246, 0.1);
  transform: scale(1.1);
}

.chat-input {
  width: 100%;
  min-height: 80px;
  max-height: 140px;
  resize: none;
  overflow-y: auto;
  border: 1px solid #e2e8f0;
  border-radius: 0.75rem;
  padding: 0.625rem 0.875rem;
  outline: none;
  font-size: 13px;
  transition: all 0.15s ease;
  background: #ffffff;
}

.chat-input:focus {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.chat-input:hover:not(:focus) {
  border-color: #cbd5e1;
}

.send-row {
  margin-top: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.closed-tip {
  margin-bottom: 0.5rem;
  border: 1px solid #fed7aa;
  background: #fff7ed;
  color: #9a3412;
  border-radius: 8px;
  padding: 0.45rem 0.6rem;
  font-size: 12px;
}

.ws-status {
  font-size: 0.75rem;
  font-weight: 500;
}

.ws-status.connected {
  color: #16a34a;
}

.ws-status.reconnect-failed,
.ws-status.disconnected {
  color: #dc2626;
}

.empty-chat {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  color: #94a3b8;
  gap: 1rem;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
}

.empty-chat .el-icon {
  opacity: 0.5;
}

.snippet-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
  max-height: 420px;
  overflow: auto;
  padding: 0.25rem;
}

.snippet-empty {
  grid-column: 1 / -1;
  border: 1px dashed #cbd5e1;
  border-radius: 10px;
  padding: 1rem;
  color: #64748b;
  font-size: 13px;
  text-align: center;
  background: #f8fafc;
}

.snippet-card {
  border: 1px solid #e2e8f0;
  background: linear-gradient(180deg, #ffffff 0%, #fafbfc 100%);
  border-radius: 0.75rem;
  padding: 0.75rem;
  text-align: left;
  cursor: pointer;
  transition: all 0.15s ease;
}

.snippet-card:hover {
  border-color: #93c5fd;
  background: linear-gradient(180deg, #eff6ff 0%, #dbeafe 100%);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.1);
}

.snippet-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.75rem;
  color: #64748b;
}

.snippet-card h4 {
  margin: 0.5rem 0 0.25rem;
  font-size: 0.875rem;
  color: #1e293b;
}

.snippet-card p {
  margin: 0;
  font-size: 0.75rem;
  color: #64748b;
}

.hidden-input {
  display: none;
}

.preview-image {
  width: 100%;
  max-height: 72vh;
  object-fit: contain;
  display: block;
  border-radius: 0.5rem;
}

.visitor-drawer {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.visitor-drawer-avatar {
  width: 56px;
  height: 56px;
  border-radius: 999px;
  border: 1px solid #dbeafe;
}

.visitor-drawer-id {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
}

.visitor-kv {
  display: grid;
  grid-template-columns: 84px 1fr;
  gap: 0.5rem;
  align-items: flex-start;
}

.visitor-kv label {
  font-size: 12px;
  color: #64748b;
}

.visitor-kv span {
  font-size: 13px;
  color: #334155;
  word-break: break-word;
}

.visitor-ua {
  white-space: pre-wrap;
}
</style>
