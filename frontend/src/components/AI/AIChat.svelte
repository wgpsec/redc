<script>
  import { onMount, onDestroy } from 'svelte';
  import { marked } from 'marked';
  import { AIChatStream, SmartAgentChatStream, StopAgentStream, ResumeAgentStream, SaveTemplateFiles, ExportChatLog, SubmitAskUserResponse, OrchestratorStream } from '../../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff, BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import { toast } from '../../lib/toast.js';

  let { t, lang, onTabChange = () => {}, visible = true } = $props();

  // Configure marked
  marked.setOptions({ breaks: true, gfm: true });

  // Render markdown to HTML (sanitize basic XSS)
  function renderMarkdown(content) {
    if (!content) return '';
    const html = marked.parse(content);
    return html;
  }

  // Intercept <a> clicks inside chat content → open in system browser
  function handleContentClick(e) {
    const a = e.target.closest('a[href]');
    if (a) {
      e.preventDefault();
      BrowserOpenURL(a.href);
    }
  }

  // State
  let mode = $state('free');
  let messages = $state([]);
  let inputText = $state('');
  let isStreaming = $state(false);
  let currentConversationId = $state('');
  let streamingContent = $state('');
  let error = $state('');
  let successMessage = $state('');
  let messagesContainer = $state(null);
  let agentToolCalls = $state([]);  // { id, toolName, toolArgs, status: 'calling'|'success'|'error', content }
  let askUserPending = $state(null); // { conversationId, toolCallId, question, choices, allowFreeform }
  let askUserInput = $state('');
  let agentPlan = $state(null); // { title, steps: [{name, status, detail}], currentStep }
  let orchestratorStatus = $state(null); // { round, maxRounds, phase, detail }
  let lastInterruptedConvId = $state(''); // Track last interrupted conversation for resume
  let lastUsage = $state(null); // { prompt_tokens, completion_tokens, total_tokens }
  // Conversation history state
  let conversations = $state([]);   // Array of { id, title, mode, messages, updatedAt }
  let activeConvId = $state('');     // Currently active conversation id
  let showHistory = $state(false);   // Toggle history panel

  const STORAGE_KEY = 'redc-ai-chat-conversations';
  const MAX_CONVERSATIONS = 50;

  const modes = [
    { id: 'free', labelKey: 'aiChatFreeChat', icon: 'M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z' },
    { id: 'agent', labelKey: 'aiChatAgent', icon: 'M11.42 15.17l-5.1-5.1a1.5 1.5 0 010-2.12l.88-.88a1.5 1.5 0 012.12 0L12 9.75l5.3-5.3a1.5 1.5 0 012.12 0l.88.88a1.5 1.5 0 010 2.12l-7.18 7.18a1.5 1.5 0 01-2.12 0zM3.75 21h16.5' },
    { id: 'orchestrator', labelKey: 'aiChatOrchestrator', icon: 'M10.5 6h9.75M10.5 6a1.5 1.5 0 11-3 0m3 0a1.5 1.5 0 10-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-9.75 0h9.75' },
  ];

  const modeLabels = { free: 'aiChatFreeChat', agent: 'aiChatAgent', orchestrator: 'aiChatOrchestrator' };
  const welcomeMessages = { free: 'aiChatWelcomeFree', agent: 'aiChatWelcomeAgent', orchestrator: 'aiChatWelcomeOrchestrator' };

  function generateId() {
    return Date.now().toString(36) + Math.random().toString(36).substr(2, 9);
  }

  function getWelcomeMessage(m) {
    return { id: generateId(), role: 'assistant', content: t[welcomeMessages[m]] || '', timestamp: Date.now(), mode: m };
  }

  // Derive conversation title from first user message
  function deriveTitle(msgs) {
    const firstUser = msgs.find(m => m.role === 'user');
    if (firstUser) {
      const text = firstUser.content.trim();
      return text.length > 30 ? text.slice(0, 30) + '...' : text;
    }
    return t.aiChatNewConversation || '新对话';
  }

  // Load all conversations from localStorage
  function loadConversations() {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        const parsed = JSON.parse(saved);
        if (Array.isArray(parsed)) {
          // Deduplicate message IDs to prevent Svelte keyed-each errors
          for (const conv of parsed) {
            if (conv.messages) {
              const seen = new Set();
              for (const m of conv.messages) {
                if (!m.id || seen.has(m.id)) {
                  m.id = (m.id || 'msg') + '-' + generateId();
                }
                seen.add(m.id);
              }
            }
          }
          conversations = parsed;
          return;
        }
      }
      // Migrate from old single-conversation format
      const oldSaved = localStorage.getItem('redc-ai-chat-state');
      if (oldSaved) {
        const parsed = JSON.parse(oldSaved);
        if (parsed.messages && parsed.messages.length > 0) {
          const conv = {
            id: generateId(),
            title: deriveTitle(parsed.messages),
            mode: parsed.mode || 'free',
            messages: parsed.messages,
            updatedAt: Date.now()
          };
          conversations = [conv];
          activeConvId = conv.id;
          mode = conv.mode;
          messages = conv.messages;
          saveConversations();
          localStorage.removeItem('redc-ai-chat-state');
          return;
        }
      }
      conversations = [];
    } catch {
      conversations = [];
    }
  }

  function saveConversations() {
    try {
      // Keep only recent conversations
      const toSave = conversations.slice(0, MAX_CONVERSATIONS);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(toSave));
    } catch {}
  }

  // Save current conversation state into conversations array
  function syncCurrentConversation() {
    if (!activeConvId) return;
    const idx = conversations.findIndex(c => c.id === activeConvId);
    const conv = {
      id: activeConvId,
      title: deriveTitle(messages),
      mode,
      messages,
      updatedAt: Date.now()
    };
    if (idx >= 0) {
      conversations[idx] = conv;
    } else {
      conversations = [conv, ...conversations];
    }
    conversations = [...conversations].sort((a, b) => b.updatedAt - a.updatedAt);
    saveConversations();
  }

  // Switch to a conversation from history
  function switchConversation(convId) {
    if (convId === activeConvId) {
      showHistory = false;
      return;
    }
    // Save current first
    syncCurrentConversation();
    const conv = conversations.find(c => c.id === convId);
    if (conv) {
      activeConvId = conv.id;
      mode = conv.mode;
      messages = [...conv.messages];
      streamingContent = '';
      isStreaming = false;
      error = '';
      currentConversationId = '';
    }
    showHistory = false;
  }

  // Delete a conversation
  function deleteConversation(convId, event) {
    event.stopPropagation();
    conversations = conversations.filter(c => c.id !== convId);
    saveConversations();
    if (convId === activeConvId) {
      createNewConversation();
    }
  }

  // Create a brand new conversation
  function createNewConversation() {
    // Save current if it has meaningful content
    if (activeConvId && messages.length > 1) {
      syncCurrentConversation();
    }
    const newId = generateId();
    activeConvId = newId;
    messages = [getWelcomeMessage(mode)];
    streamingContent = '';
    isStreaming = false;
    error = '';
    currentConversationId = '';
    inputText = '';
    showHistory = false;
    // Don't save empty conversation to list yet — will save on first message
  }

  // Storage event handler (kept at module level for cleanup)
  function handleStorage(e) {
    if (e.key === 'ai-chat-pending-terminal' && e.newValue) {
      checkPendingTerminalText();
    }
    if (e.key === 'ai-chat-pending-error' && e.newValue) {
      checkPendingErrorAnalysis();
    }
  }

  onMount(() => {
    loadConversations();

    // If we have conversations, load the most recent one
    if (conversations.length > 0 && !activeConvId) {
      const latest = conversations[0];
      activeConvId = latest.id;
      mode = latest.mode;
      messages = [...latest.messages];
    }

    // If still no conversation, create a fresh one
    if (!activeConvId) {
      activeConvId = generateId();
      messages = [getWelcomeMessage(mode)];
    }

    EventsOn('ai-chat-chunk', (data) => {
      if (data.conversationId === currentConversationId) {
        streamingContent += data.chunk;
      }
    });

    EventsOn('ai-chat-complete', (data) => {
      if (data.conversationId === currentConversationId) {
        // Capture token usage
        const usage = data.usage && data.usage.total_tokens > 0 ? data.usage : null;

        if (data.success && streamingContent) {
          // Auto-complete plan steps when conversation ends successfully
          if (agentPlan && agentPlan.steps) {
            agentPlan = {
              ...agentPlan,
              steps: agentPlan.steps.map(s =>
                s.status === 'running' || s.status === 'pending'
                  ? { ...s, status: 'done' }
                  : s
              )
            };
          }
          // For agent mode, include tool call cards in the message
          const toolCards = agentToolCalls.length > 0 ? [...agentToolCalls] : undefined;
          const planSnapshot = agentPlan ? { ...agentPlan } : undefined;
          messages = [...messages, {
            id: generateId(),
            role: 'assistant',
            content: streamingContent,
            timestamp: Date.now(),
            mode,
            toolCalls: toolCards,
            plan: planSnapshot,
            usage
          }];
          // Track timeout interruptions (sent as success=true with timeout emoji)
          if (mode !== 'free' && streamingContent.includes('⏱️')) {
            lastInterruptedConvId = currentConversationId;
          }
        } else if (!data.success) {
          // Preserve partial streaming content on error (e.g. timeout)
          if (streamingContent) {
            const toolCards = agentToolCalls.length > 0 ? [...agentToolCalls] : undefined;
            const planSnapshot = agentPlan ? { ...agentPlan } : undefined;
            messages = [...messages, {
              id: generateId(),
              role: 'assistant',
              content: streamingContent + '\n\n⚠ ' + (t.aiChatStreamInterrupted || '响应中断，以上为已接收的部分内容'),
              timestamp: Date.now(),
              mode,
              toolCalls: toolCards,
              plan: planSnapshot,
              usage
            }];
          }
          // Track interrupted conversation for potential resume
          if (mode !== 'free') {
            lastInterruptedConvId = currentConversationId;
          }
          error = t.aiChatStreamError || 'AI 响应失败，请重试';
        }
        // Notify user if they may not be watching
        if (!visible) {
          toast.info(data.success
            ? (t.aiChatCompleteNotify || 'AI 对话已完成')
            : (t.aiChatFailedNotify || 'AI 对话执行失败'));
        }
        streamingContent = '';
        isStreaming = false;
        currentConversationId = '';
        agentToolCalls = [];
        askUserPending = null;
        agentPlan = null;
        orchestratorStatus = null;
        syncCurrentConversation();
      }
    });

    EventsOn('ai-agent-tool-call', (data) => {
      if (data.conversationId === currentConversationId) {
        agentToolCalls = [...agentToolCalls, {
          id: data.toolCallId,
          toolName: data.toolName,
          toolArgs: data.toolArgs,
          status: 'calling',
          content: ''
        }];
        scrollToBottom();
      }
    });

    EventsOn('ai-agent-tool-result', (data) => {
      if (data.conversationId === currentConversationId) {
        agentToolCalls = agentToolCalls.map(tc =>
          tc.id === data.toolCallId
            ? { ...tc, status: data.success ? 'success' : 'error', content: data.content }
            : tc
        );
        scrollToBottom();
      }
    });

    EventsOn('ai-agent-ask-user', (data) => {
      if (data.conversationId === currentConversationId) {
        askUserPending = {
          conversationId: data.conversationId,
          toolCallId: data.toolCallId,
          question: data.question,
          choices: data.choices || [],
          allowFreeform: data.allowFreeform !== false,
        };
        askUserInput = '';
        scrollToBottom();
      }
    });

    EventsOn('ai-agent-plan', (data) => {
      if (data.conversationId === currentConversationId) {
        agentPlan = {
          title: data.title || '',
          steps: data.steps || [],
          currentStep: data.currentStep || 0,
        };
        scrollToBottom();
      }
    });

    EventsOn('ai-chat-failover', (data) => {
      if (data.conversationId === currentConversationId) {
        const msg = (t.aiChatFailoverNotice || 'Provider failover: {provider} ({model})').replace('{provider}', data.provider).replace('{model}', data.model);
        toast.warning(msg);
      }
    });

    EventsOn('ai-chat-compact', (data) => {
      if (data.conversationId === currentConversationId) {
        const beforeK = Math.round(data.before / 1000);
        const afterK = Math.round(data.after / 1000);
        const budgetK = Math.round(data.budget / 1000);
        const msg = t.aiChatCompactNotice
          .replace('{before}', beforeK).replace('{after}', afterK).replace('{budget}', budgetK);
        messages = [...messages, {
          id: 'compact-' + generateId(),
          role: 'system-notice',
          content: msg,
          timestamp: Date.now()
        }];
        toast.info(msg);
        scrollToBottom();
      }
    });

    EventsOn('ai-orchestrator-status', (data) => {
      if (data.conversationId === currentConversationId) {
        orchestratorStatus = {
          round: data.round,
          phase: data.phase,
          detail: data.detail,
        };
        scrollToBottom();
      }
    });

    EventsOn('ai-orchestrator-judge', (data) => {
      if (data.conversationId === currentConversationId) {
        const eval_ = data.evaluation || {};
        orchestratorStatus = {
          ...orchestratorStatus,
          phase: 'judged',
          judge: {
            round: data.round,
            confidence: Math.round((eval_.confidence || 0) * 100),
            feedback: eval_.feedback || '',
            missing: eval_.missing_areas || [],
            nextSteps: eval_.next_steps || [],
            complete: eval_.complete || false,
          }
        };
        scrollToBottom();
      }
    });

    // Check for pending terminal text on initial mount
    checkPendingTerminalText();
    checkPendingErrorAnalysis();

    // Listen for cross-tab storage events
    window.addEventListener('storage', handleStorage);
  });

  onDestroy(() => {
    EventsOff('ai-chat-chunk');
    EventsOff('ai-chat-complete');
    EventsOff('ai-chat-failover');
    EventsOff('ai-chat-compact');
    EventsOff('ai-agent-tool-call');
    EventsOff('ai-agent-tool-result');
    EventsOff('ai-agent-ask-user');
    EventsOff('ai-agent-plan');
    EventsOff('ai-orchestrator-status');
    EventsOff('ai-orchestrator-judge');
    window.removeEventListener('storage', handleStorage);
  });

  // Check for pending terminal text when tab becomes visible
  $effect(() => {
    if (visible) {
      checkPendingTerminalText();
      checkPendingErrorAnalysis();
    }
  });

  // Auto-scroll
  $effect(() => {
    if (streamingContent || messages.length) {
      scrollToBottom();
    }
  });

  function scrollToBottom(force = false) {
    if (messagesContainer) {
      requestAnimationFrame(() => {
        if (!messagesContainer) return;
        // Only auto-scroll if user is near bottom (within 150px) or forced
        const { scrollTop, scrollHeight, clientHeight } = messagesContainer;
        const isNearBottom = scrollHeight - scrollTop - clientHeight < 150;
        if (force || isNearBottom) {
          messagesContainer.scrollTop = messagesContainer.scrollHeight;
        }
      });
    }
  }

  // Switch mode — creates a new conversation with the new mode
  function switchMode(newMode) {
    if (newMode === mode && !isStreaming) return;
    // Save current if meaningful
    if (activeConvId && messages.length > 1) {
      syncCurrentConversation();
    }
    mode = newMode;
    activeConvId = generateId();
    messages = [getWelcomeMessage(newMode)];
    streamingContent = '';
    isStreaming = false;
    error = '';
    currentConversationId = '';
  }

  // Check for pending terminal text from SSH Manager
  function checkPendingTerminalText() {
    try {
      const pending = localStorage.getItem('ai-chat-pending-terminal');
      if (pending) {
        localStorage.removeItem('ai-chat-pending-terminal');
        // Switch to free mode for terminal analysis
        if (mode !== 'free') {
          mode = 'free';
          activeConvId = generateId();
          messages = [getWelcomeMessage('free')];
        }
        // Pre-fill input with the terminal content wrapped in a prompt
        const prompt = (t.analyzeTerminalPrompt || '请帮我分析以下终端输出内容') + ':\n```\n' + pending + '\n```';
        inputText = prompt;
      }
    } catch (_) {}
  }

  // Check for pending error analysis from Cases/CustomDeployment pages
  function checkPendingErrorAnalysis() {
    try {
      const raw = localStorage.getItem('ai-chat-pending-error');
      if (raw) {
        localStorage.removeItem('ai-chat-pending-error');
        const data = JSON.parse(raw);
        // Create new conversation in agent mode for error analysis
        mode = 'agent';
        activeConvId = generateId();
        messages = [getWelcomeMessage('agent')];
        streamingContent = '';
        isStreaming = false;
        error = '';
        currentConversationId = '';

        let prompt = '';
        if (data.templateName) prompt += `模板: ${data.templateName}\n`;
        if (data.provider) prompt += `云厂商: ${data.provider}\n`;
        prompt += '错误信息:\n```\n' + (data.error || '') + '\n```';
        inputText = prompt;
        // Auto-send the error analysis message
        setTimeout(() => sendMessage(), 100);
      }
    } catch (_) {}
  }

  // Stop a running agent
  async function stopAgent() {
    if (!isStreaming || !currentConversationId) return;
    try {
      await StopAgentStream(currentConversationId);
    } catch (e) {
      console.error('Failed to stop agent:', e);
    }
  }

  // Resume an interrupted agent conversation from checkpoint
  async function resumeAgent() {
    if (!lastInterruptedConvId || isStreaming) return;
    const convId = lastInterruptedConvId;
    lastInterruptedConvId = '';
    error = '';
    isStreaming = true;
    currentConversationId = convId;
    streamingContent = '';
    agentToolCalls = [];
    try {
      await ResumeAgentStream(convId);
    } catch (e) {
      error = `恢复失败: ${e}`;
      isStreaming = false;
      currentConversationId = '';
    }
  }

  // Submit answer for ask_user tool
  function submitAskUserAnswer(answer) {
    if (!askUserPending) return;
    const { conversationId, toolCallId } = askUserPending;
    // Update the tool call card to show user's answer
    agentToolCalls = agentToolCalls.map(tc =>
      tc.id === toolCallId
        ? { ...tc, status: 'success', content: answer }
        : tc
    );
    askUserPending = null;
    askUserInput = '';
    SubmitAskUserResponse(conversationId, answer);
    scrollToBottom();
  }

  // Send message
  async function sendMessage() {
    const text = inputText.trim();
    if (!text || isStreaming) return;

    error = '';
    const userMessage = { id: generateId(), role: 'user', content: text, timestamp: Date.now(), mode };
    messages = [...messages, userMessage];
    inputText = '';

    isStreaming = true;
    streamingContent = '';
    agentToolCalls = [];
    agentPlan = null;
    orchestratorStatus = null;
    const convId = generateId();
    currentConversationId = convId;

    // Build messages for backend (only role + content)
    const chatMessages = messages
      .filter(m => m.role === 'user' || m.role === 'assistant')
      .filter(m => m.content)
      .map(m => ({ role: m.role, content: m.content }));

    try {
      if (mode === 'orchestrator') {
        await OrchestratorStream(convId, { maxRounds: 5, objective: text, autoApprove: false }, chatMessages);
      } else if (mode === 'agent') {
        await SmartAgentChatStream(convId, chatMessages);
      } else {
        await AIChatStream(convId, 'free', chatMessages);
      }
    } catch (e) {
      // Preserve partial streaming content on error
      if (streamingContent) {
        const toolCards = agentToolCalls.length > 0 ? [...agentToolCalls] : undefined;
        const planSnapshot = agentPlan ? { ...agentPlan } : undefined;
        messages = [...messages, {
          id: generateId(),
          role: 'assistant',
          content: streamingContent + '\n\n⚠ ' + (t.aiChatStreamInterrupted || '响应中断，以上为已接收的部分内容'),
          timestamp: Date.now(),
          mode,
          toolCalls: toolCards,
          plan: planSnapshot
        }];
      }
      error = e.message || String(e);
      isStreaming = false;
      streamingContent = '';
      currentConversationId = '';
      agentToolCalls = [];
      askUserPending = null;
      agentPlan = null;
    }

    syncCurrentConversation();
  }

  // Parse Markdown template content and extract individual files (operates on raw content)
  /** @param {string} markdown */
  /** @returns {Record<string, string>} */
  function parseTemplateMarkdown(markdown) {
    const files = /** @type {Record<string, string>} */ ({});
    const fileBlocks = markdown.split(/^###\s+/m);
    for (const block of fileBlocks) {
      if (!block.trim()) continue;
      const lines = block.split('\n');
      const filename = lines[0].trim();
      if (!filename.match(/\.(json|tfvars|tf|md|sh|yaml|yml)$/i)) continue;
      const content = lines.slice(1).join('\n').trim();
      let fileContent = content.replace(/^```[\w]*\n?/g, '').replace(/```$/g, '').trim();
      files[filename] = fileContent;
    }
    return files;
  }

  async function handleSaveTemplate(content) {
    const files = parseTemplateMarkdown(content);
    if (Object.keys(files).length === 0) {
      error = t.noTemplateFound || '未检测到有效的模板文件';
      return;
    }
    let templateName = 'ai-generated-' + Date.now();
    if (files['case.json']) {
      try {
        const caseJson = JSON.parse(files['case.json']);
        templateName = caseJson.name || caseJson.Name || templateName;
      } catch {}
    }
    if (!templateName.toLowerCase().startsWith('ai-')) {
      templateName = 'ai-' + templateName;
    }
    try {
      const savedPath = await SaveTemplateFiles(templateName, files);
      error = '';
      successMessage = `${t.templateSaved || '模板已保存'}：${savedPath}`;
      setTimeout(() => { successMessage = ''; }, 3000);
    } catch (e) {
      error = e.message || String(e);
    }
  }

  async function handleCopyContent(content) {
    try {
      await navigator.clipboard.writeText(content);
      successMessage = t.aiChatCopied || '已复制';
      setTimeout(() => { successMessage = ''; }, 2000);
    } catch (e) {
      console.error('Failed to copy:', e);
    }
  }

  async function exportChatLog() {
    if (messages.length === 0) return;
    const modeLabel = modeLabels[mode] ? (t[modeLabels[mode]] || mode) : mode;
    let md = `# RedC AI Chat Log\n\n`;
    md += `- **Mode**: ${modeLabel}\n`;
    md += `- **Time**: ${new Date().toLocaleString()}\n`;
    md += `- **Messages**: ${messages.length}\n\n---\n\n`;

    for (const msg of messages) {
      const time = msg.timestamp ? new Date(msg.timestamp).toLocaleTimeString() : '';
      if (msg.role === 'user') {
        md += `## 🧑 User ${time ? `(${time})` : ''}\n\n${msg.content}\n\n`;
      } else {
        // Tool calls first
        if (msg.toolCalls && msg.toolCalls.length > 0) {
          md += `### 🔧 Tool Calls\n\n`;
          for (const tc of msg.toolCalls) {
            const status = tc.status === 'success' ? '✅' : tc.status === 'error' ? '❌' : '⏳';
            md += `${status} **${tc.toolName}**`;
            if (tc.toolArgs && Object.keys(tc.toolArgs).length > 0) {
              md += ` \`${formatToolArgs(tc.toolArgs)}\``;
            }
            md += `\n`;
            if (tc.content) {
              md += `\n<details><summary>Result</summary>\n\n\`\`\`\n${tc.content}\n\`\`\`\n\n</details>\n\n`;
            }
          }
        }
        md += `## 🤖 Assistant ${time ? `(${time})` : ''}\n\n${msg.content}\n\n`;
      }
      md += `---\n\n`;
    }

    try {
      await ExportChatLog(md);
      successMessage = t.aiChatExported || '对话日志已导出';
      setTimeout(() => { successMessage = ''; }, 2000);
    } catch (e) {
      if (e) error = e.message || String(e);
    }
  }

  function formatTime(ts) {
    if (!ts) return '';
    const d = new Date(ts);
    const now = new Date();
    const isToday = d.toDateString() === now.toDateString();
    if (isToday) return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    return d.toLocaleDateString([], { month: 'short', day: 'numeric' }) + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  // Tool name display mapping
  const toolNameMap = {
    list_templates: '列出模板', search_templates: '搜索模板', pull_template: '下载模板',
    list_cases: '列出场景', plan_case: '规划场景', start_case: '启动场景',
    stop_case: '停止场景', kill_case: '销毁场景', get_case_status: '查看状态',
    exec_command: '执行命令', get_ssh_info: '获取 SSH 信息',
    upload_file: '上传文件', download_file: '下载文件',
    get_template_info: '模板详情', delete_template: '删除模板',
    get_case_outputs: '获取输出', get_config: '获取配置', validate_config: '验证配置',
    list_userdata_templates: '列出部署脚本', exec_userdata: '执行部署脚本',
    save_compose_file: '保存编排文件', compose_preview: '预览编排', compose_up: '启动编排', compose_down: '销毁编排',
    save_template_files: '保存模板文件',
    ask_user: '用户决策',
    update_plan: '更新计划',
    get_template_files: '读取模板文件',
    get_cost_estimate: '成本估算', get_balances: '余额查询', get_resource_summary: '资源汇总', get_predicted_monthly_cost: '月度预测',
    schedule_task: '定时任务', list_scheduled_tasks: '列出定时任务', cancel_scheduled_task: '取消定时任务',
  };

  function getToolDisplayName(name) {
    return toolNameMap[name] || name;
  }

  function formatToolArgs(args) {
    if (!args || typeof args !== 'object') return '';
    return Object.entries(args).map(([k, v]) => `${k}: ${typeof v === 'string' ? v : JSON.stringify(v)}`).join(', ');
  }

  // Quick prompt suggestions per mode
  function getQuickPrompts(m) {
    const promptsZh = {
      free: [
        { label: '红队基础设施规划', text: '帮我规划一个完整的红队基础设施方案，包括 C2、重定向器和钓鱼平台' },
        { label: '云服务商对比', text: '对比各个云服务商在红队基础设施方面的优缺点' },
        { label: '安全加固建议', text: '如何对部署的红队基础设施进行安全加固？' },
      ],
      agent: [
        { label: '部署 nginx 服务器', text: '帮我部署一个 nginx 服务器' },
        { label: '查看当前场景', text: '列出所有当前运行中的场景和状态' },
        { label: '生成 AWS EC2 模板', text: '帮我生成一个 AWS EC2 实例的 Terraform 模板' },
        { label: '分析当前成本', text: '分析我当前所有运行中场景的成本，并给出优化建议' },
        { label: '推荐 C2 场景', text: '推荐适合长期渗透的 C2 基础设施部署方案' },
      ],
      orchestrator: [
        { label: '搭建高可用集群', text: '在 AWS 上搭建一个高可用 Web 服务集群' },
        { label: '安全审计', text: '对所有运行中的场景进行安全审计并生成报告' },
      ],
    };
    const promptsEn = {
      free: [
        { label: 'Red Team Infra Planning', text: 'Help me plan a complete red team infrastructure including C2, redirectors and phishing platform' },
        { label: 'Cloud Provider Comparison', text: 'Compare cloud providers for red team infrastructure in terms of pros and cons' },
        { label: 'Security Hardening', text: 'How to harden deployed red team infrastructure for better security?' },
      ],
      agent: [
        { label: 'Deploy nginx server', text: 'Help me deploy an nginx server' },
        { label: 'List running scenes', text: 'List all currently running scenes and their status' },
        { label: 'Generate AWS EC2 template', text: 'Help me generate a Terraform template for an AWS EC2 instance' },
        { label: 'Analyze current costs', text: 'Analyze the costs of all my running scenes and suggest optimizations' },
        { label: 'Recommend C2 scenario', text: 'Recommend a C2 infrastructure deployment plan suitable for long-term operations' },
      ],
      orchestrator: [
        { label: 'Set up HA cluster', text: 'Set up a highly available web cluster on AWS' },
        { label: 'Security audit', text: 'Audit all running scenarios for security and generate a report' },
      ],
    };
    const prompts = lang === 'en' ? promptsEn : promptsZh;
    return prompts[m] || prompts.free;
  }

  // Add copy buttons to rendered code blocks
  function addCodeCopyButtons(container) {
    if (!container) return;
    container.querySelectorAll('pre').forEach(pre => {
      if (pre.querySelector('.code-copy-btn')) return;
      const btn = document.createElement('button');
      btn.className = 'code-copy-btn';
      btn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9.75a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184"/></svg>';
      btn.onclick = async () => {
        const code = pre.querySelector('code')?.textContent || pre.textContent;
        try {
          await navigator.clipboard.writeText(code);
          btn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5"/></svg>';
          setTimeout(() => { btn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9.75a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184"/></svg>'; }, 1500);
        } catch {}
      };
      pre.style.position = 'relative';
      pre.appendChild(btn);
    });
  }

  // Effect to add copy buttons after render
  $effect(() => {
    if (messages.length || streamingContent) {
      requestAnimationFrame(() => addCodeCopyButtons(messagesContainer));
    }
  });
</script>

<div class="flex flex-col h-full px-6 pt-6 pb-4">
  <!-- Toolbar: mode tabs + actions -->
  <div class="flex items-center gap-3 mb-3 flex-shrink-0">
    <div class="flex gap-0.5 bg-gray-100 rounded-lg p-0.5 flex-wrap">
      {#each modes as m}
        <button
          class="px-2.5 py-1 text-[11px] rounded-md transition-colors cursor-pointer whitespace-nowrap {mode === m.id ? 'bg-white text-gray-900 shadow-sm font-medium' : 'text-gray-500 hover:text-gray-700'}"
          onclick={() => switchMode(m.id)}
        >{t[m.labelKey] || m.id}</button>
      {/each}
    </div>
    <div class="flex-1"></div>
    <button
      class="p-1.5 rounded-lg transition-colors cursor-pointer {showHistory ? 'bg-gray-900 text-white' : 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'}"
      onclick={() => showHistory = !showHistory}
      title={t.aiChatHistory || '对话历史'}
    >
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
    </button>
    <button
      class="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors cursor-pointer"
      onclick={createNewConversation}
      title={t.aiChatNewConversation || '新对话'}
    >
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
    </button>
    <button
      class="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
      onclick={exportChatLog}
      disabled={messages.length <= 1}
      title={t.aiChatExport || '导出对话日志'}
    >
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
    </button>
  </div>

  <!-- Error / Success -->
  {#if error}
    <div class="mb-3 flex items-center gap-2 px-3 py-2 bg-red-50 border border-red-100 rounded-lg flex-shrink-0">
      <svg class="w-3.5 h-3.5 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[12px] text-red-700 flex-1">{error}</span>
      <button class="text-red-400 hover:text-red-600 cursor-pointer" onclick={() => error = ''}>
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}
  {#if lastInterruptedConvId && !isStreaming}
    <div class="mb-3 flex items-center gap-2 px-3 py-2 bg-amber-50 border border-amber-200 rounded-lg flex-shrink-0">
      <svg class="w-3.5 h-3.5 text-amber-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182M2.985 19.644l3.181-3.182" />
      </svg>
      <span class="text-[12px] text-amber-700 flex-1">{t.aiChatResumeHint || '上次 Agent 任务中断，可从检查点恢复继续执行'}</span>
      <button class="text-[11px] px-2 py-0.5 bg-amber-500 text-white rounded hover:bg-amber-600 cursor-pointer" onclick={resumeAgent}>
        {t.aiChatResume || '恢复执行'}
      </button>
      <button class="text-amber-400 hover:text-amber-600 cursor-pointer" onclick={() => lastInterruptedConvId = ''}>
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}
  {#if successMessage}
    <div class="mb-3 flex items-center gap-2 px-3 py-2 bg-emerald-50 border border-emerald-100 rounded-lg flex-shrink-0">
      <svg class="w-3.5 h-3.5 text-emerald-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <span class="text-[12px] text-emerald-700">{successMessage}</span>
    </div>
  {/if}

  <!-- Main content area with optional history panel -->
  <div class="flex-1 flex gap-4 min-h-0 overflow-hidden">
    <!-- History panel -->
    {#if showHistory}
      <div class="w-56 flex-shrink-0 bg-white border border-gray-100 rounded-xl flex flex-col overflow-hidden">
        <div class="px-3 py-2.5 border-b border-gray-100 flex items-center justify-between">
          <span class="text-[12px] font-medium text-gray-700">{t.aiChatHistory || '对话历史'}</span>
          <span class="text-[10px] text-gray-400">{conversations.length}</span>
        </div>
        <div class="flex-1 overflow-y-auto">
          {#if conversations.length === 0}
            <div class="px-3 py-6 text-center text-[11px] text-gray-400">
              {t.aiChatNoHistory || '暂无对话历史'}
            </div>
          {:else}
            {#each conversations as conv (conv.id)}
              <div
                class="w-full text-left px-3 py-2.5 border-b border-gray-50 hover:bg-gray-50 transition-colors cursor-pointer group
                  {conv.id === activeConvId ? 'bg-gray-50' : ''}"
                onclick={() => switchConversation(conv.id)}
                role="button"
                onkeydown={(e) => e.key === 'Enter' && switchConversation(conv.id)}
                tabindex="0"
              >
                <div class="flex items-start justify-between gap-1">
                  <div class="min-w-0 flex-1">
                    <p class="text-[12px] font-medium text-gray-800 truncate {conv.id === activeConvId ? 'text-gray-900 font-semibold' : ''}">{conv.title}</p>
                    <div class="flex items-center gap-1.5 mt-0.5">
                      <span class="text-[10px] px-1.5 py-0.5 rounded bg-gray-100 text-gray-500">{t[modeLabels[conv.mode]] || conv.mode}</span>
                      <span class="text-[10px] text-gray-400">{formatTime(conv.updatedAt)}</span>
                    </div>
                  </div>
                  <button
                    class="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-red-50 hover:text-red-500 text-gray-300 transition-all cursor-pointer flex-shrink-0"
                    onclick={(e) => deleteConversation(conv.id, e)}
                    title={t.delete || '删除'}
                  >
                    <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                    </svg>
                  </button>
                </div>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    {/if}

    <!-- Chat area -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- Messages -->
      <div class="flex-1 overflow-y-auto space-y-4 pb-4" bind:this={messagesContainer} onclick={handleContentClick}>
        {#each messages as msg (msg.id)}
          {#if msg.role === 'system-notice'}
            <!-- System notice (compaction, etc.) -->
            <div class="flex justify-center">
              <div class="px-3 py-1.5 rounded-full bg-blue-50 border border-blue-100 flex items-center gap-1.5">
                <svg class="w-3 h-3 text-blue-500 shrink-0" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M4 2v12M12 2v12M1 5l3-3M1 11l3 3M15 5l-3-3M15 11l-3 3M4 8h8"/>
                </svg>
                <p class="text-[11px] text-blue-600">{msg.content}</p>
              </div>
            </div>
          {:else if msg.role === 'user'}
            <!-- User message -->
            <div class="flex justify-end">
              <div class="max-w-[75%]">
                <div class="px-4 py-2.5 rounded-2xl rounded-br-md bg-gray-900 text-white">
                  <p class="text-[13px] whitespace-pre-wrap leading-relaxed">{msg.content}</p>
                </div>
                {#if msg.timestamp}
                  <div class="text-[10px] text-gray-300 mt-1 text-right pr-1">{formatTime(msg.timestamp)}</div>
                {/if}
              </div>
            </div>
          {:else}
            <!-- Assistant message with markdown -->
            <div class="flex justify-start">
              <div class="max-w-[85%]">
                <div class="flex items-start gap-2.5">
                  <div class="w-7 h-7 rounded-lg bg-rose-600 flex items-center justify-center flex-shrink-0 mt-0.5">
                    <svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
                    </svg>
                  </div>
                  <div class="flex-1 min-w-0">
                    <!-- Saved plan card (agent mode history) -->
                    {#if msg.plan && msg.plan.steps && msg.plan.steps.length > 0}
                      <div class="mb-2 p-2.5 bg-blue-50 border border-blue-200 rounded-lg">
                        <div class="flex items-center gap-1.5 mb-1.5">
                          <svg class="w-3.5 h-3.5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15a2.25 2.25 0 012.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25z" /></svg>
                          <span class="text-[11px] font-semibold text-blue-800">{msg.plan.title || t.agentPlanTitle || '执行计划'}</span>
                          <span class="text-[10px] text-blue-500 ml-auto font-mono">{msg.plan.steps.filter(s => s.status === 'done').length}/{msg.plan.steps.length}</span>
                        </div>
                        <div class="space-y-0.5">
                          {#each msg.plan.steps as step, i}
                            <div class="text-[10px] {step.status === 'done' ? 'text-gray-400' : step.status === 'failed' ? 'text-red-500' : 'text-gray-600'}">
                              {#if step.status === 'done'}<svg class="inline w-3 h-3 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" /></svg>{:else if step.status === 'failed'}<svg class="inline w-3 h-3 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>{:else if step.status === 'skipped'}<svg class="inline w-3 h-3 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 8.689c0-.864.933-1.406 1.683-.977l7.108 4.061a1.125 1.125 0 010 1.954l-7.108 4.061A1.125 1.125 0 013 16.811V8.69zM12.75 8.689c0-.864.933-1.406 1.683-.977l7.108 4.061a1.125 1.125 0 010 1.954l-7.108 4.061a1.125 1.125 0 01-1.683-.977V8.69z" /></svg>{:else}<svg class="inline w-3 h-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2" /></svg>{/if} {i + 1}. {step.name || step.content}
                            </div>
                          {/each}
                        </div>
                      </div>
                    {/if}
                    <!-- Saved tool call cards (agent mode history) -->
                    {#if msg.toolCalls && msg.toolCalls.length > 0}
                      <div class="mb-2 space-y-1.5">
                        {#each msg.toolCalls as tc}
                          {#if tc.toolName === 'ask_user'}
                            <div class="flex items-start gap-2 px-3 py-2 rounded-lg border-2 {tc.status === 'success' ? 'bg-gray-50 border-gray-200' : 'bg-gray-50 border-gray-300'}">
                              <span class="mt-0.5">
                                <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                              </span>
                              <div class="flex-1 min-w-0">
                                <div class="text-[12px] font-medium text-gray-700">{t.askUserTitle || 'AI 需要你的决策'}</div>
                                {#if tc.toolArgs?.question}
                                  <div class="text-[12px] text-gray-600 mt-0.5">{tc.toolArgs.question}</div>
                                {/if}
                                {#if tc.content}
                                  <div class="mt-1 text-[12px] font-medium text-gray-800 bg-white rounded px-2 py-1 border border-gray-200">↩ {tc.content}</div>
                                {/if}
                              </div>
                            </div>
                          {:else if tc.toolName === 'update_plan'}
                            <!-- update_plan: compact card, plan details shown in plan card above -->
                            <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg border bg-blue-50 border-blue-200">
                              <svg class="w-3.5 h-3.5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15a2.25 2.25 0 012.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25z" /></svg>
                              <span class="text-[11px] text-blue-700">{getToolDisplayName(tc.toolName)}</span>
                              {#if tc.toolArgs?.title}
                                <span class="text-[11px] text-blue-500">— {tc.toolArgs.title}</span>
                              {/if}
                              {#if tc.toolArgs?.steps}
                                <span class="text-[10px] text-blue-400 ml-auto font-mono">{tc.toolArgs.steps.filter(s => s.status === 'done').length}/{tc.toolArgs.steps.length}</span>
                              {/if}
                            </div>
                          {:else}
                          <div class="flex items-start gap-2 px-3 py-2 rounded-lg border {tc.status === 'success' ? 'bg-emerald-50 border-emerald-200' : tc.status === 'error' ? 'bg-red-50 border-red-200' : 'bg-gray-50 border-gray-200'}">
                            <span class="mt-0.5">
                              {#if tc.status === 'success'}
                                <svg class="w-3.5 h-3.5 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                              {:else if tc.status === 'error'}
                                <svg class="w-3.5 h-3.5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9.75 9.75l4.5 4.5m0-4.5l-4.5 4.5M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                              {:else}
                                <svg class="w-3.5 h-3.5 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
                              {/if}
                            </span>
                            <div class="flex-1 min-w-0">
                              <div class="text-[12px] font-medium text-gray-700">
                                <svg class="w-3 h-3 inline -mt-0.5 mr-0.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M11.42 15.17l-5.1-5.1a1.5 1.5 0 010-2.12l.88-.88a1.5 1.5 0 012.12 0L12 9.75l2.88-2.88a1.5 1.5 0 012.12 0l.88.88a1.5 1.5 0 010 2.12l-5.1 5.1a1.5 1.5 0 01-2.12 0z" /></svg>
                                {getToolDisplayName(tc.toolName)}</div>
                              {#if tc.toolArgs && Object.keys(tc.toolArgs).length > 0}
                                <div class="text-[11px] text-gray-500 font-mono truncate">{formatToolArgs(tc.toolArgs)}</div>
                              {/if}
                              {#if tc.content}
                                <details class="mt-1">
                                  <summary class="text-[11px] text-gray-400 cursor-pointer hover:text-gray-600">{t.agentViewResult || '查看结果'}</summary>
                                  <pre class="mt-1 text-[11px] text-gray-600 bg-white rounded p-2 max-h-32 overflow-auto whitespace-pre-wrap">{tc.content}</pre>
                                </details>
                              {/if}
                            </div>
                          </div>
                          {/if}
                        {/each}
                      </div>
                    {/if}
                    <div class="px-4 py-2.5 rounded-2xl rounded-tl-md bg-white border border-gray-100">
                      <div class="md-content text-[13px] text-gray-900 leading-relaxed">
                        {@html renderMarkdown(msg.content)}
                      </div>
                    </div>
                    <!-- Action buttons -->
                    {#if msg.content}
                      <div class="flex items-center gap-1 mt-1.5 ml-1">
                        <button
                          class="flex items-center gap-1 px-2 py-1 rounded text-[11px] text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors cursor-pointer"
                          onclick={() => handleCopyContent(msg.content)}
                        >
                          <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9.75a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                          </svg>
                          {t.aiChatCopyContent || '复制'}
                        </button>
                        {#if msg.content && (msg.content.includes('main.tf') || msg.content.includes('case.json') || msg.content.includes('```hcl'))}
                          <button
                            class="flex items-center gap-1 px-2 py-1 rounded text-[11px] text-gray-400 hover:text-rose-600 hover:bg-rose-50 transition-colors cursor-pointer"
                            onclick={() => handleSaveTemplate(msg.content)}
                          >
                            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                              <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
                            </svg>
                            {t.aiChatSaveTemplate || '保存模板'}
                          </button>
                        {/if}
                      </div>
                    {/if}
                    {#if msg.timestamp}
                      <div class="flex items-center gap-2 mt-1 ml-1">
                        <span class="text-[10px] text-gray-300">{formatTime(msg.timestamp)}</span>
                        {#if msg.usage && msg.usage.total_tokens > 0}
                          <span class="text-[10px] text-gray-300">·</span>
                          <span class="text-[10px] text-gray-300" title={`${t.tokenInput} ${msg.usage.prompt_tokens} + ${t.tokenOutput} ${msg.usage.completion_tokens} = ${t.tokenTotal} ${msg.usage.total_tokens} tokens`}>
                            <svg class="inline w-3 h-3 text-gray-400 -mt-px" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" /></svg> {msg.usage.prompt_tokens.toLocaleString()} → {msg.usage.completion_tokens.toLocaleString()} ({msg.usage.total_tokens.toLocaleString()} tokens)
                          </span>
                        {/if}
                      </div>
                    {/if}
                  </div>
                </div>
              </div>
            </div>
          {/if}
        {/each}

        <!-- Quick prompt suggestions for empty conversations -->
        {#if messages.length === 1 && !isStreaming}
          <div class="max-w-lg mx-auto mt-2">
            <div class="grid grid-cols-2 gap-2">
              {#each getQuickPrompts(mode) as prompt}
                <button
                  class="text-left px-3 py-2.5 text-[12px] text-gray-600 bg-white border border-gray-100 rounded-xl hover:border-gray-300 hover:shadow-sm transition-all cursor-pointer leading-relaxed"
                  onclick={() => { inputText = prompt.text; sendMessage(); }}
                >
                  <span class="text-gray-400 mr-1">→</span> {prompt.label}
                </button>
              {/each}
            </div>
          </div>
        {/if}

        <!-- Streaming indicator -->
        {#if isStreaming}
          <div class="flex justify-start">
            <div class="max-w-[85%]">
              <div class="flex items-start gap-2.5">
                <div class="w-7 h-7 rounded-lg bg-rose-600 flex items-center justify-center flex-shrink-0 mt-0.5">
                  <svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
                  </svg>
                </div>
                <div class="flex-1 min-w-0">
                  <!-- Live agent tool call cards -->
                  {#if agentToolCalls.length > 0}
                    <div class="mb-2 space-y-1.5">
                      {#each agentToolCalls as tc (tc.id)}
                        {#if tc.toolName === 'ask_user'}
                          <div class="flex items-start gap-2 px-3 py-2 rounded-lg border-2 {tc.status === 'success' ? 'bg-gray-50 border-gray-200' : 'bg-gray-50 border-gray-300'}">
                            <span class="mt-0.5">
                              {#if tc.status === 'calling'}
                                <svg class="w-3.5 h-3.5 animate-pulse text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                              {:else}
                                <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                              {/if}
                            </span>
                            <div class="flex-1 min-w-0">
                              <div class="text-[12px] font-medium text-gray-700">{t.askUserTitle || 'AI 需要你的决策'}</div>
                              {#if tc.toolArgs?.question}
                                <div class="text-[12px] text-gray-600 mt-0.5">{tc.toolArgs.question}</div>
                              {/if}
                              {#if tc.content}
                                <div class="mt-1 text-[12px] font-medium text-gray-800 bg-white rounded px-2 py-1 border border-gray-200">↩ {tc.content}</div>
                              {/if}
                            </div>
                          </div>
                        {:else if tc.toolName === 'update_plan'}
                          <!-- update_plan: compact card, plan details shown in plan card above -->
                          <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg border {tc.status === 'success' ? 'bg-blue-50 border-blue-200' : tc.status === 'calling' ? 'bg-blue-50 border-blue-300' : 'bg-red-50 border-red-200'}">
                            <span class="text-xs">{#if tc.status === 'calling'}<svg class="inline w-3 h-3 text-gray-400 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>{:else}<svg class="inline w-3 h-3 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15a2.25 2.25 0 012.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25z" /></svg>{/if}</span>
                            <span class="text-[11px] text-blue-700">{getToolDisplayName(tc.toolName)}</span>
                            {#if tc.toolArgs?.title}
                              <span class="text-[11px] text-blue-500">— {tc.toolArgs.title}</span>
                            {/if}
                            {#if tc.toolArgs?.steps}
                              <span class="text-[10px] text-blue-400 ml-auto font-mono">{tc.toolArgs.steps.filter(s => s.status === 'done').length}/{tc.toolArgs.steps.length}</span>
                            {/if}
                          </div>
                        {:else}
                        <div class="flex items-start gap-2 px-3 py-2 rounded-lg border {tc.status === 'success' ? 'bg-emerald-50 border-emerald-200' : tc.status === 'error' ? 'bg-red-50 border-red-200' : 'bg-amber-50 border-amber-200'}">
                          <span class="mt-0.5">
                            {#if tc.status === 'calling'}
                              <svg class="w-3.5 h-3.5 animate-spin text-amber-500" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
                            {:else if tc.status === 'success'}
                              <svg class="w-3.5 h-3.5 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                            {:else}
                              <svg class="w-3.5 h-3.5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9.75 9.75l4.5 4.5m0-4.5l-4.5 4.5M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                            {/if}
                          </span>
                          <div class="flex-1 min-w-0">
                            <div class="text-[12px] font-medium text-gray-700">
                              <svg class="w-3 h-3 inline -mt-0.5 mr-0.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M11.42 15.17l-5.1-5.1a1.5 1.5 0 010-2.12l.88-.88a1.5 1.5 0 012.12 0L12 9.75l2.88-2.88a1.5 1.5 0 012.12 0l.88.88a1.5 1.5 0 010 2.12l-5.1 5.1a1.5 1.5 0 01-2.12 0z" /></svg>
                              {getToolDisplayName(tc.toolName)}</div>
                            {#if tc.toolArgs && Object.keys(tc.toolArgs).length > 0}
                              <div class="text-[11px] text-gray-500 font-mono truncate">{formatToolArgs(tc.toolArgs)}</div>
                            {/if}
                            {#if tc.content}
                              <details class="mt-1">
                                <summary class="text-[11px] text-gray-400 cursor-pointer hover:text-gray-600">{t.agentViewResult || '查看结果'}</summary>
                                <pre class="mt-1 text-[11px] text-gray-600 bg-white rounded p-2 max-h-32 overflow-auto whitespace-pre-wrap">{tc.content}</pre>
                              </details>
                            {/if}
                          </div>
                        </div>
                        {/if}
                      {/each}
                    </div>
                  {/if}
                  <!-- ask_user interactive card -->
                  {#if askUserPending}
                    <div class="mb-2 px-4 py-3 rounded-xl border-2 border-gray-300 bg-gray-50/80">
                      <div class="flex items-center gap-2 mb-2">
                        <svg class="w-4 h-4 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        <span class="text-[13px] font-semibold text-gray-900">{t.askUserTitle || 'AI 需要你的决策'}</span>
                      </div>
                      <p class="text-[13px] text-gray-700 mb-3 leading-relaxed">{askUserPending.question}</p>
                      {#if askUserPending.choices.length > 0}
                        <div class="flex flex-col gap-1.5 mb-3">
                          {#each askUserPending.choices as choice, i}
                            <button
                              class="w-full text-left px-3 py-2 text-[13px] bg-white border border-gray-200 rounded-lg hover:bg-gray-100 hover:border-gray-400 transition-colors cursor-pointer"
                              onclick={() => submitAskUserAnswer(choice)}
                            >
                              <span class="inline-flex items-center justify-center w-5 h-5 text-[11px] font-semibold text-gray-600 bg-gray-100 rounded-full mr-2">{i + 1}</span>
                              {choice}
                            </button>
                          {/each}
                        </div>
                      {/if}
                      {#if askUserPending.allowFreeform}
                        <div class="flex items-center gap-2">
                          <input
                            type="text"
                            class="flex-1 px-3 py-2 text-[13px] bg-white border border-gray-200 rounded-lg focus:ring-1 focus:ring-gray-900 focus:border-gray-900 transition-shadow placeholder-gray-400"
                            placeholder={t.askUserInputPlaceholder || '或输入你的想法...'}
                            bind:value={askUserInput}
                            onkeydown={(e) => { if (e.key === 'Enter' && askUserInput.trim()) { submitAskUserAnswer(askUserInput.trim()); } }}
                          />
                          <button
                            class="px-3 py-2 text-[13px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 cursor-pointer"
                            disabled={!askUserInput.trim()}
                            onclick={() => submitAskUserAnswer(askUserInput.trim())}
                          >
                            {t.send || '发送'}
                          </button>
                        </div>
                      {/if}
                    </div>
                  {/if}
                  <div class="px-4 py-2.5 rounded-2xl rounded-tl-md bg-white border border-gray-100">
                    {#if streamingContent}
                      <div class="md-content text-[13px] text-gray-900 leading-relaxed">
                        {@html renderMarkdown(streamingContent)}
                        <span class="inline-block w-1.5 h-4 bg-rose-500 animate-pulse ml-0.5 align-middle"></span>
                      </div>
                    {:else}
                      <div class="flex items-center gap-2">
                        <svg class="w-4 h-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
                          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        <span class="text-[12px] text-gray-400">
                          {#if mode === 'agent' && agentToolCalls.length > 0}
                            {t.agentProcessing || 'Agent 执行中...'}
                          {:else}
                            {t.aiChatStreaming || 'AI 思考中...'}
                          {/if}
                        </span>
                      </div>
                    {/if}
                  </div>
                </div>
              </div>
            </div>
          </div>
        {/if}

        <div class="h-1"></div>
      </div>

      <!-- Sticky progress panel (plan + orchestrator status) above input -->
      {#if isStreaming && (orchestratorStatus || (agentPlan && agentPlan.steps && agentPlan.steps.length > 0))}
        <div class="flex-shrink-0 px-0.5 pb-2">
          <!-- Orchestrator status + judge -->
          {#if orchestratorStatus}
            <div class="mb-1.5 p-2 bg-gradient-to-r from-purple-50 to-indigo-50 border border-purple-200 rounded-lg">
              <div class="flex items-center gap-2">
                <svg class="w-3.5 h-3.5 text-purple-600 {orchestratorStatus.phase === 'executing' || orchestratorStatus.phase === 'judging' ? 'animate-pulse' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M10.5 6h9.75M10.5 6a1.5 1.5 0 11-3 0m3 0a1.5 1.5 0 10-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-9.75 0h9.75" /></svg>
                <span class="text-[11px] font-semibold text-purple-800">{t.aiChatOrchestrator || 'Orchestrator'}</span>
                <span class="text-[10px] text-purple-500 font-mono ml-auto">
                  {orchestratorStatus.phase === 'complete' ? '✓' : orchestratorStatus.detail || ''}
                </span>
              </div>
              <!-- Judge evaluation result -->
              {#if orchestratorStatus.judge}
                <div class="mt-1.5 pt-1.5 border-t border-purple-200/60 space-y-0.5">
                  <div class="flex items-center gap-1.5 text-[10px]">
                    <span class="font-semibold text-purple-700">{t.aiOrchestratorJudgeResult || 'Judge'} R{orchestratorStatus.judge.round}</span>
                    <span class="px-1.5 py-0.5 rounded-full text-[9px] font-mono {orchestratorStatus.judge.confidence >= 80 ? 'bg-emerald-100 text-emerald-700' : orchestratorStatus.judge.confidence >= 50 ? 'bg-amber-100 text-amber-700' : 'bg-red-100 text-red-700'}">
                      {orchestratorStatus.judge.confidence}%
                    </span>
                  </div>
                  {#if orchestratorStatus.judge.feedback}
                    <p class="text-[10px] text-purple-600 leading-snug line-clamp-2">{orchestratorStatus.judge.feedback}</p>
                  {/if}
                </div>
              {/if}
            </div>
          {/if}
          <!-- Agent plan -->
          {#if agentPlan && agentPlan.steps && agentPlan.steps.length > 0}
            <div class="p-2.5 bg-gradient-to-r from-blue-50 to-indigo-50 border border-blue-200 rounded-lg">
              <div class="flex items-center gap-2 mb-1.5">
                <svg class="w-3.5 h-3.5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15a2.25 2.25 0 012.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25z" /></svg>
                <span class="text-[11px] font-semibold text-blue-900">{agentPlan.title || t.agentPlanTitle || '执行计划'}</span>
                <span class="text-[10px] text-blue-500 ml-auto font-mono">
                  {agentPlan.steps.filter(s => s.status === 'done').length}/{agentPlan.steps.length}
                </span>
              </div>
              <div class="space-y-0.5">
                {#each agentPlan.steps as step, i}
                  <div class="flex items-center gap-1.5 text-[11px] {step.status === 'done' ? 'text-gray-400' : step.status === 'running' ? 'text-blue-700 font-medium' : step.status === 'failed' ? 'text-red-600' : step.status === 'skipped' ? 'text-gray-400 line-through' : 'text-gray-600'}">
                    <span class="flex-shrink-0">
                      {#if step.status === 'done'}<svg class="w-3 h-3 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" /></svg>
                      {:else if step.status === 'running'}<svg class="w-3 h-3 text-blue-600 animate-pulse" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5.14v14l11-7-11-7z" /></svg>
                      {:else if step.status === 'failed'}<svg class="w-3 h-3 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
                      {:else}<svg class="w-3 h-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2" /></svg>
                      {/if}
                    </span>
                    <span class="truncate">{i + 1}. {step.name || step.content}</span>
                  </div>
                {/each}
              </div>
              {#if agentPlan.steps.length > 0}
                {@const doneCount = agentPlan.steps.filter(s => s.status === 'done').length}
                {@const totalCount = agentPlan.steps.length}
                <div class="mt-1.5 h-1 bg-blue-100 rounded-full overflow-hidden">
                  <div class="h-full bg-blue-500 rounded-full transition-all duration-300" style="width: {totalCount > 0 ? (doneCount / totalCount * 100) : 0}%"></div>
                </div>
              {/if}
            </div>
          {/if}
        </div>
      {/if}

      <!-- Input area -->
      <div class="flex-shrink-0 border-t border-gray-100 pt-3 pb-1 px-0.5">
        <div class="flex items-end gap-2">
          <textarea
            class="flex-1 px-4 py-2.5 text-[13px] bg-white border border-gray-200 rounded-xl text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent transition-shadow resize-none"
            rows="2"
            placeholder={t.aiChatPlaceholder || '输入消息... Ctrl/Cmd+Enter 发送'}
            bind:value={inputText}
            onkeydown={(e) => {
              if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                e.preventDefault();
                sendMessage();
              }
            }}
            disabled={isStreaming}
          ></textarea>
          <button
            class="px-4 h-10 bg-gray-900 text-white text-[12px] font-medium rounded-xl hover:bg-gray-800 transition-colors disabled:opacity-50 flex items-center gap-2 cursor-pointer flex-shrink-0"
            onclick={sendMessage}
            disabled={isStreaming || !inputText.trim()}
          >
            {#if isStreaming}
              <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            {:else}
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5" />
              </svg>
            {/if}
            {t.aiChatSend || '发送'}
          </button>
          {#if isStreaming && (mode === 'agent' || mode === 'orchestrator')}
            <button
              class="px-3 h-10 bg-red-600 text-white text-[12px] font-medium rounded-xl hover:bg-red-700 transition-colors flex items-center gap-1.5 cursor-pointer flex-shrink-0"
              onclick={stopAgent}
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <rect x="6" y="6" width="12" height="12" rx="1" />
              </svg>
              {t.aiChatStop || '停止'}
            </button>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>

<style>
  /* Markdown content styles */
  .md-content :global(h1) { font-size: 1.25em; font-weight: 700; margin: 0.8em 0 0.4em; }
  .md-content :global(h2) { font-size: 1.1em; font-weight: 600; margin: 0.7em 0 0.3em; }
  .md-content :global(h3) { font-size: 1em; font-weight: 600; margin: 0.6em 0 0.3em; }
  .md-content :global(p) { margin: 0.4em 0; }
  .md-content :global(ul), .md-content :global(ol) { margin: 0.4em 0; padding-left: 1.5em; }
  .md-content :global(li) { margin: 0.2em 0; }
  .md-content :global(code) {
    background: #f3f4f6; padding: 0.15em 0.4em; border-radius: 4px;
    font-size: 0.9em; font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, monospace;
  }
  .md-content :global(pre) {
    background: #1f2937; color: #e5e7eb; padding: 0.8em 1em; border-radius: 8px;
    overflow-x: auto; margin: 0.5em 0; font-size: 0.85em; line-height: 1.6;
  }
  .md-content :global(pre code) {
    background: none; padding: 0; color: inherit; font-size: inherit;
  }
  .md-content :global(blockquote) {
    border-left: 3px solid #d1d5db; padding-left: 0.8em; margin: 0.5em 0;
    color: #6b7280; font-style: italic;
  }
  .md-content :global(table) { width: 100%; border-collapse: collapse; margin: 0.5em 0; font-size: 0.9em; }
  .md-content :global(th), .md-content :global(td) { border: 1px solid #e5e7eb; padding: 0.4em 0.6em; text-align: left; }
  .md-content :global(th) { background: #f9fafb; font-weight: 600; }
  .md-content :global(hr) { border: none; border-top: 1px solid #e5e7eb; margin: 0.8em 0; }
  .md-content :global(a) { color: #2563eb; text-decoration: underline; }
  .md-content :global(strong) { font-weight: 600; }
  .md-content :global(> *:first-child) { margin-top: 0; }
  .md-content :global(> *:last-child) { margin-bottom: 0; }

  /* Code block copy button */
  :global(.code-copy-btn) {
    position: absolute;
    top: 6px;
    right: 6px;
    padding: 4px;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.1);
    color: #9ca3af;
    border: none;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s, background 0.15s;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  :global(pre:hover .code-copy-btn) {
    opacity: 1;
  }
  :global(.code-copy-btn:hover) {
    background: rgba(255, 255, 255, 0.2);
    color: #e5e7eb;
  }
</style>
