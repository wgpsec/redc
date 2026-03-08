<script>
  import { onMount, onDestroy } from 'svelte';
  import { StartSSHTerminal, StartSSHTerminalInstance, WriteToTerminal, ResizeTerminal, CloseTerminal, StartPortForward, StopPortForward, ListPortForwards, GetSSHInfoForCase, GetSSHInfosForCase, UploadUserdataScript, ListCases } from '../../../wailsjs/wailsjs/go/main/App.js';
  import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime.js';
  import FileManager from '../Cases/FileManager.svelte';
  import { loadUserdataTemplates, getGroupedTemplates, userdataCategoryNames } from '../../lib/userdataTemplates.js';

  let { t, onTabChange } = $props();

  // --- Session state ---
  let sessions = $state([]);
  let activeSessionIndex = $state(-1);
  let showNewSessionDialog = $state(false);
  let newSessionCaseId = $state('');
  let newSessionCaseName = $state('');
  let availableCases = $state([]);
  let casesLoading = $state(false);
  let showManualInput = $state(false);
  let multiInstances = $state([]);
  let multiInstanceCase = $state(null);

  // --- Right panel state: 'none' | 'portForward' | 'userdata' | 'fileManager' ---
  let rightPanel = $state('none');

  // --- Port forwarding state ---
  let portForwards = $state([]);
  let pfCaseId = $state('');
  let pfLocalPort = $state('');
  let pfRemoteHost = $state('');
  let pfRemotePort = $state('');
  let pfLoading = $state(false);
  let pfError = $state('');

  // --- Userdata state ---
  let userdataTemplates = $state([]);
  let userdataTemplatesLoading = $state(true);
  let selectedUserdataTemplate = $state(null);
  let uploading = $state(false);
  let expandedCategories = $state({});
  let groupedUserdataTemplates = $derived(() => getGroupedTemplates(userdataTemplates));

  // --- File manager modal ---
  let showFileManagerModal = $state(false);

  // --- Terminal selection → AI ---
  let selectedText = $state('');
  let showSendToAI = $state(false);

  // --- xterm modules (loaded once) ---
  let xtermModules = $state(null);

  const TERMINAL_THEME = {
    background: '#111827', foreground: '#d1d5db', cursor: '#60a5fa',
    black: '#1f2937', red: '#f87171', green: '#4ade80', yellow: '#fbbf24',
    blue: '#60a5fa', magenta: '#c084fc', cyan: '#22d3ee', white: '#9ca3af',
    brightBlack: '#4b5563', brightRed: '#fca5a5', brightGreen: '#86efac',
    brightYellow: '#fcd34d', brightBlue: '#93c5fd', brightMagenta: '#d8b4fe',
    brightCyan: '#67e8f9', brightWhite: '#f3f4f6',
  };

  let activeSession = $derived(activeSessionIndex >= 0 && activeSessionIndex < sessions.length ? sessions[activeSessionIndex] : null);

  onMount(async () => {
    try {
      const [xtermMod, fitMod] = await Promise.all([
        import('xterm'),
        import('xterm-addon-fit'),
      ]);
      xtermModules = { Terminal: xtermMod.Terminal, FitAddon: fitMod.FitAddon };
    } catch (err) {
      console.error('Failed to load xterm:', err);
    }

    // Check for pending session from other pages
    try {
      const pending = localStorage.getItem('ssh-pending-session');
      if (pending) {
        localStorage.removeItem('ssh-pending-session');
        const { caseId, caseName } = JSON.parse(pending);
        if (caseId) {
          await createSession(caseId, caseName || caseId);
        }
      }
    } catch (_) {}

    await refreshPortForwards();

    // Load userdata templates
    try {
      userdataTemplates = await loadUserdataTemplates();
    } catch (_) {}
    userdataTemplatesLoading = false;

    EventsOn('port-forward-closed', (id) => {
      portForwards = portForwards.filter(pf => pf.id !== id);
    });
  });

  onDestroy(() => {
    for (const session of sessions) {
      cleanupSession(session);
    }
    EventsOff('port-forward-closed');
  });

  // --- Visibility management ---
  $effect(() => {
    const idx = activeSessionIndex;
    for (let i = 0; i < sessions.length; i++) {
      const s = sessions[i];
      if (s.containerEl) {
        s.containerEl.style.display = i === idx ? 'block' : 'none';
      }
      if (i === idx && s.fitAddon && s.terminal) {
        try { s.fitAddon.fit(); } catch (_) {}
      }
    }
  });

  async function createSession(caseId, caseName, instanceIndex = 0, hostInfo = null) {
    if (!xtermModules) return;

    let host = '', user = '';
    if (hostInfo) {
      host = hostInfo.host || '';
      user = hostInfo.user || '';
    } else {
      try {
        const info = await GetSSHInfoForCase(caseId);
        host = info.host || '';
        user = info.user || '';
      } catch (_) {}
    }

    const session = {
      id: crypto.randomUUID(),
      caseId,
      caseName: caseName || caseId,
      instanceIndex,
      sessionId: null,
      terminal: null,
      fitAddon: null,
      containerEl: null,
      resizeObserver: null,
      connected: false,
      connecting: true,
      error: '',
      host,
      user,
    };

    sessions = [...sessions, session];
    const idx = sessions.length - 1;
    activeSessionIndex = idx;

    await tick();

    const reactiveSession = sessions[idx];
    initSessionTerminal(reactiveSession);
  }

  function tick() {
    return new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
  }

  function initSessionTerminal(session) {
    const { Terminal, FitAddon } = xtermModules;

    const terminal = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: TERMINAL_THEME,
      allowProposedApi: true,
    });

    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);

    session.terminal = terminal;
    session.fitAddon = fitAddon;

    if (session.containerEl) {
      terminal.open(session.containerEl);
      fitAddon.fit();
    } else {
      console.error('SSHManager: containerEl is null');
    }

    terminal.onData((data) => {
      if (session.sessionId && session.connected) {
        WriteToTerminal(session.sessionId, data).catch(() => {});
      }
    });

    // Track text selection for "Send to AI" feature
    terminal.onSelectionChange(() => {
      const sel = terminal.getSelection();
      if (sel && sel.trim().length > 0) {
        selectedText = sel;
        showSendToAI = true;
      } else {
        selectedText = '';
        showSendToAI = false;
      }
    });

    if (session.containerEl) {
      const resizeObserver = new ResizeObserver(() => {
        if (session.fitAddon && session.terminal) {
          try {
            session.fitAddon.fit();
            if (session.sessionId && session.connected) {
              ResizeTerminal(session.sessionId, session.terminal.rows, session.terminal.cols).catch(() => {});
            }
          } catch (_) {}
        }
      });
      resizeObserver.observe(session.containerEl);
      session.resizeObserver = resizeObserver;
    }

    connectSession(session);
  }

  async function connectSession(session) {
    session.connecting = true;
    session.error = '';
    sessions = [...sessions];

    session.terminal?.writeln('\x1b[1;34m' + (t.sshConnecting || '正在连接 SSH...') + '\x1b[0m');

    const rows = session.terminal?.rows || 24;
    const cols = session.terminal?.cols || 80;

    try {
      const sid = await StartSSHTerminalInstance(session.caseId, session.instanceIndex || 0, rows, cols);
      session.sessionId = sid;

      EventsOn(`terminal-output-${sid}`, (data) => {
        session.terminal?.write(data);
      });
      EventsOn(`terminal-error-${sid}`, (err) => {
        session.terminal?.writeln(`\r\n\x1b[1;31m${err}\x1b[0m`);
        session.error = err;
        sessions = [...sessions];
      });
      EventsOn(`terminal-closed-${sid}`, () => {
        session.terminal?.writeln('\r\n\x1b[1;33m' + (t.sshConnectionClosed || '连接已关闭') + '\x1b[0m');
        session.connected = false;
        sessions = [...sessions];
      });

      session.connected = true;
      session.terminal?.writeln('\x1b[1;32m' + (t.sshConnected || '已连接') + '\x1b[0m\r\n');
    } catch (err) {
      session.error = err.message || String(err);
      session.terminal?.writeln(`\r\n\x1b[1;31m${session.error}\x1b[0m`);
    } finally {
      session.connecting = false;
      sessions = [...sessions];
    }
  }

  function cleanupSession(session) {
    if (session.sessionId) {
      EventsOff(`terminal-output-${session.sessionId}`);
      EventsOff(`terminal-error-${session.sessionId}`);
      EventsOff(`terminal-closed-${session.sessionId}`);
      CloseTerminal(session.sessionId).catch(() => {});
    }
    if (session.resizeObserver) session.resizeObserver.disconnect();
    session.terminal?.dispose();
  }

  function closeSession(index) {
    const session = sessions[index];
    if (!session) return;
    cleanupSession(session);
    sessions = sessions.filter((_, i) => i !== index);
    if (activeSessionIndex >= sessions.length) {
      activeSessionIndex = sessions.length - 1;
    } else if (activeSessionIndex > index) {
      activeSessionIndex--;
    }
  }

  let showCloseAllConfirm = $state(false);

  function closeAllSessions() {
    for (const session of sessions) {
      cleanupSession(session);
    }
    sessions = [];
    activeSessionIndex = -1;
    showCloseAllConfirm = false;
  }

  async function openNewSessionDialog() {
    showNewSessionDialog = true;
    showManualInput = false;
    newSessionCaseId = '';
    newSessionCaseName = '';
    multiInstances = [];
    multiInstanceCase = null;
    casesLoading = true;
    try {
      const cases = await ListCases();
      availableCases = (cases || []).filter(c => c.state === 'running');
    } catch {
      availableCases = [];
    }
    casesLoading = false;
  }

  async function selectCase(c) {
    // 检查是否有多个实例
    try {
      const infos = await GetSSHInfosForCase(c.id);
      if (infos && infos.length > 1) {
        multiInstanceCase = c;
        multiInstances = infos;
        return;
      }
    } catch (_) {}
    createSession(c.id, c.name || c.id);
    showNewSessionDialog = false;
  }

  function selectInstance(c, info, index) {
    const label = `${c.name || c.id.substring(0, 12)} #${index + 1} (${info.host})`;
    createSession(c.id, label, index, info);
    multiInstances = [];
    multiInstanceCase = null;
    showNewSessionDialog = false;
  }

  function connectAllInstances(c, infos) {
    for (let i = 0; i < infos.length; i++) {
      const label = `${c.name || c.id.substring(0, 12)} #${i + 1} (${infos[i].host})`;
      createSession(c.id, label, i, infos[i]);
    }
    multiInstances = [];
    multiInstanceCase = null;
    showNewSessionDialog = false;
  }

  function handleNewSessionSubmit() {
    const caseId = newSessionCaseId.trim();
    if (!caseId) return;
    createSession(caseId, newSessionCaseName.trim() || caseId);
    newSessionCaseId = '';
    newSessionCaseName = '';
    showNewSessionDialog = false;
  }

  function sessionLabel(session) {
    if (session.caseName && session.caseName !== session.caseId) return session.caseName;
    if (session.user && session.host) return `${session.user}@${session.host}`;
    return session.caseId?.substring(0, 12) || 'SSH';
  }

  function sendToAIChat() {
    if (!selectedText.trim()) return;
    localStorage.setItem('ai-chat-pending-terminal', selectedText.trim());
    showSendToAI = false;
    // Clear terminal selection
    if (activeSession?.terminal) {
      activeSession.terminal.clearSelection();
    }
    selectedText = '';
    onTabChange?.('aiChat');
  }

  function togglePanel(panel) {
    rightPanel = rightPanel === panel ? 'none' : panel;
    // Refit terminal after panel toggle
    requestAnimationFrame(() => {
      const s = activeSession;
      if (s?.fitAddon && s?.terminal) {
        try { s.fitAddon.fit(); } catch (_) {}
      }
    });
  }

  // --- Userdata: upload script to terminal ---
  async function uploadAndShowCommand() {
    if (!selectedUserdataTemplate?.script || !activeSession?.connected) return;

    uploading = true;
    const fileName = `${selectedUserdataTemplate.name || 'userdata'}.sh`;

    try {
      const result = await UploadUserdataScript(activeSession.caseId, selectedUserdataTemplate.script, fileName);
      if (result.success) {
        activeSession.terminal?.writeln(`\r\n\x1b[32m${t.scriptUploadedTo || '脚本已上传到'}: /tmp/${fileName}\x1b[0m`);
        activeSession.terminal?.writeln(`\x1b[33m${t.execScriptCmd || '执行命令'}: bash /tmp/${fileName}\x1b[0m\r\n`);
      } else {
        activeSession.terminal?.writeln(`\r\n\x1b[31m${t.uploadFailed || '上传失败'}: ${result.error}\x1b[0m\r\n`);
      }
    } catch (err) {
      activeSession.terminal?.writeln(`\r\n\x1b[31m${t.uploadFailed || '上传失败'}: ${err.message}\x1b[0m\r\n`);
    } finally {
      uploading = false;
    }
  }

  // --- Port Forwarding ---
  async function refreshPortForwards() {
    try {
      portForwards = await ListPortForwards() || [];
    } catch (_) {
      portForwards = [];
    }
  }

  async function handleStartPortForward() {
    const caseId = activeSession?.caseId || pfCaseId.trim();
    const lp = parseInt(pfLocalPort, 10);
    const rh = pfRemoteHost.trim() || '127.0.0.1';
    const rp = parseInt(pfRemotePort, 10);

    if (!caseId || !lp || !rp) {
      pfError = t.sshAllFieldsRequired || '请填写所有字段';
      return;
    }

    pfLoading = true;
    pfError = '';
    try {
      const info = await StartPortForward(caseId, lp, rh, rp);
      portForwards = [...portForwards, info];
      pfCaseId = '';
      pfLocalPort = '';
      pfRemoteHost = '';
      pfRemotePort = '';
    } catch (err) {
      pfError = err.message || String(err);
    } finally {
      pfLoading = false;
    }
  }

  async function handleStopPortForward(id) {
    try {
      await StopPortForward(id);
      portForwards = portForwards.filter(pf => pf.id !== id);
    } catch (err) {
      console.error('StopPortForward failed:', err);
    }
  }
</script>

<svelte:head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@5.3.0/css/xterm.css" />
</svelte:head>

<div class="h-full flex flex-col bg-white text-gray-900">
  <!-- Main layout -->
  <div class="flex-1 flex overflow-hidden">
    <!-- Left: Terminal area -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- Tab bar -->
      <div class="flex-shrink-0 flex flex-wrap items-center bg-gray-50 border-b border-gray-200 px-2 min-h-[40px] py-1 gap-1">
        {#each sessions as session, i (session.id)}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-t-lg text-[12px] font-medium cursor-pointer select-none transition-colors max-w-[200px] {i === activeSessionIndex ? 'bg-white text-gray-900 border border-gray-200 border-b-white -mb-px shadow-sm' : 'text-gray-500 hover:text-gray-900 hover:bg-gray-100'}"
            role="tab"
            tabindex="0"
            aria-selected={i === activeSessionIndex}
            onclick={() => activeSessionIndex = i}
            onkeydown={(e) => { if (e.key === 'Enter') activeSessionIndex = i; }}
          >
            {#if session.connected}
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 flex-shrink-0"></span>
            {:else if session.connecting}
              <span class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse flex-shrink-0"></span>
            {:else}
              <span class="w-1.5 h-1.5 rounded-full bg-gray-400 flex-shrink-0"></span>
            {/if}
            <span class="truncate">{sessionLabel(session)}</span>
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <span
              class="ml-1 w-4 h-4 flex items-center justify-center rounded hover:bg-red-50 text-gray-400 hover:text-red-500 cursor-pointer flex-shrink-0"
              role="button"
              tabindex="0"
              aria-label="Close session"
              onclick={(e) => { e.stopPropagation(); closeSession(i); }}
              onkeydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); closeSession(i); } }}
            >
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </span>
          </div>
        {/each}

        <!-- New session button -->
        <button
          class="flex items-center justify-center w-7 h-7 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50 transition-colors flex-shrink-0 cursor-pointer"
          onclick={openNewSessionDialog}
          title={t.sshNewSession || '新建会话'}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
          </svg>
        </button>

        <!-- Close all sessions button -->
        {#if sessions.length > 1}
          <button
            class="flex items-center justify-center w-7 h-7 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50 transition-colors flex-shrink-0 cursor-pointer"
            onclick={() => showCloseAllConfirm = true}
            title={t.sshCloseAll || '关闭所有连接'}
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        {/if}

        <div class="flex-1"></div>

        <!-- Toolbar buttons -->
        {#if activeSession}
          <!-- Userdata toggle -->
          <button
            class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[12px] font-medium transition-colors flex-shrink-0 cursor-pointer {rightPanel === 'userdata' ? 'bg-red-50 text-red-600' : 'text-gray-500 hover:text-gray-900 hover:bg-gray-100'}"
            onclick={() => togglePanel('userdata')}
            title={t.execUserdata || '命令片段'}
          >
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12c0 1.268-.63 2.39-1.593 3.068a3.745 3.745 0 01-1.043 3.296 3.745 3.745 0 01-3.296 1.043A3.745 3.745 0 0112 21c-1.268 0-2.39-.63-3.068-1.593a3.746 3.746 0 01-3.296-1.043 3.745 3.745 0 01-1.043-3.296A3.745 3.745 0 013 12c0-1.268.63-2.39 1.593-3.068a3.745 3.745 0 011.043-3.296 3.746 3.746 0 013.296-1.043A3.746 3.746 0 0112 3c1.268 0 2.39.63 3.068 1.593a3.746 3.746 0 013.296 1.043 3.746 3.746 0 011.043 3.296A3.745 3.745 0 0121 12z" />
            </svg>
            {t.execUserdata || '命令片段'}
          </button>

          <!-- File manager button -->
          <button
            class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[12px] font-medium transition-colors flex-shrink-0 cursor-pointer text-gray-500 hover:text-gray-900 hover:bg-gray-100"
            onclick={() => showFileManagerModal = true}
            title={t.fileManager || '文件管理器'}
          >
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
            </svg>
            {t.fileManager || '文件管理器'}
          </button>
        {/if}

        <!-- Port forwarding toggle -->
        <button
          class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[12px] font-medium transition-colors flex-shrink-0 cursor-pointer {rightPanel === 'portForward' ? 'bg-red-50 text-red-600' : 'text-gray-500 hover:text-gray-900 hover:bg-gray-100'}"
          onclick={() => togglePanel('portForward')}
          title={t.sshPortForward || '端口转发'}
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5" />
          </svg>
          {t.sshPortForward || '端口转发'}
          {#if portForwards.length > 0}
            <span class="px-1.5 py-0.5 bg-red-100 text-red-600 text-[10px] font-medium rounded-full">{portForwards.length}</span>
          {/if}
        </button>
      </div>

      <!-- Terminal containers -->
      <div class="flex-1 relative overflow-hidden bg-gray-900 {sessions.length === 0 ? '!bg-white' : ''} rounded-b-lg m-0">
        {#if sessions.length === 0}
          <div class="absolute inset-0 flex items-center justify-center bg-white">
            <div class="text-center">
              <div class="w-16 h-16 rounded-2xl bg-red-50 flex items-center justify-center mx-auto mb-4">
                <svg class="w-8 h-8 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
                </svg>
              </div>
              <p class="text-gray-900 text-[14px] font-medium mb-1">{t.sshNoSessions || '暂无 SSH 会话'}</p>
              <p class="text-gray-400 text-[12px] mb-4">{t.sshCreateHint || '点击下方按钮或从场景页面打开 SSH 终端'}</p>
              <button
                class="h-10 px-5 bg-red-600 hover:bg-red-700 text-white text-[13px] font-medium rounded-lg transition-colors cursor-pointer"
                onclick={openNewSessionDialog}
              >
                {t.sshNewSession || '新建会话'}
              </button>
            </div>
          </div>
        {/if}

        {#each sessions as session, i (session.id)}
          <div
            class="ssh-terminal-container absolute inset-0"
            bind:this={session.containerEl}
          ></div>
        {/each}

        <!-- Floating "Send to AI" button -->
        {#if showSendToAI && selectedText}
          <div class="absolute bottom-4 right-4 z-10 animate-in fade-in">
            <button
              class="flex items-center gap-2 px-4 py-2.5 bg-red-600 hover:bg-red-700 text-white text-[13px] font-medium rounded-xl shadow-lg shadow-red-600/25 transition-all hover:shadow-xl hover:shadow-red-600/30 cursor-pointer"
              onclick={sendToAIChat}
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z" />
              </svg>
              {t.sendToAI || '发送到 AI 分析'}
              <span class="text-red-200 text-[11px]">({selectedText.length > 100 ? selectedText.length + ' chars' : selectedText.substring(0, 30) + '...'})</span>
            </button>
          </div>
        {/if}
      </div>
    </div>

    <!-- Right panel: Userdata -->
    {#if rightPanel === 'userdata' && activeSession}
      <div class="w-80 flex-shrink-0 bg-white border-l border-gray-200 flex flex-col overflow-hidden">
        <div class="flex items-center justify-between px-4 py-3 border-b border-gray-200">
          <h3 class="text-[13px] font-semibold text-gray-900">{t.execUserdata || '命令片段'}</h3>
          <button
            class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors cursor-pointer"
            onclick={() => rightPanel = 'none'}
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-3 space-y-2">
          {#if userdataTemplatesLoading}
            <div class="text-center py-8 text-gray-400 text-[13px]">{t.loading || '加载中...'}</div>
          {:else if userdataTemplates.length === 0}
            <div class="text-center py-8">
              <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
              </svg>
              <p class="text-gray-400 text-[13px] mb-1">{t.noTemplates || '暂无可用模板'}</p>
              <p class="text-gray-400 text-[12px] mb-3">{t.noTemplatesHint || '请先在模板仓库中下载 userdata 模板'}</p>
              <button
                class="h-8 px-4 text-[12px] font-medium text-red-600 bg-red-50 rounded-lg hover:bg-red-100 transition-colors cursor-pointer"
                onclick={() => onTabChange?.('registry')}
              >
                {t.goToTemplateRepo || '前往模板仓库 →'}
              </button>
            </div>
          {:else}
            <div class="space-y-2">
              {#each Object.entries(groupedUserdataTemplates()) as [category, categoryTemplates]}
                <div class="border border-gray-200 rounded-lg overflow-hidden">
                  <button
                    class="w-full flex items-center justify-between px-3 py-2 bg-gray-50 hover:bg-gray-100 transition-colors cursor-pointer"
                    onclick={() => expandedCategories[category] = !expandedCategories[category]}
                  >
                    <span class="text-[12px] font-medium text-gray-700">
                      {userdataCategoryNames[category] || category}
                      <span class="ml-1 text-gray-400">({categoryTemplates.length})</span>
                    </span>
                    <svg class="w-4 h-4 text-gray-400 transition-transform {expandedCategories[category] ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                    </svg>
                  </button>
                  {#if expandedCategories[category]}
                    <div class="space-y-1 p-2 bg-white">
                      {#each categoryTemplates as template}
                        <button
                          class="w-full text-left px-3 py-2 text-[12px] bg-gray-50 hover:bg-gray-100 border border-gray-200 hover:border-red-300 rounded transition-colors cursor-pointer {selectedUserdataTemplate === template ? 'border-red-400 bg-red-50' : ''}"
                          onclick={() => selectedUserdataTemplate = template}
                        >
                          <span class="font-medium text-gray-900">{template.nameZh || template.name}</span>
                          {#if template.description}
                            <span class="block text-gray-400 text-[10px] truncate">{template.description}</span>
                          {/if}
                        </button>
                      {/each}
                    </div>
                  {/if}
                </div>
              {/each}
            </div>

            {#if selectedUserdataTemplate}
              <div class="mt-3">
                <span class="block text-[12px] font-medium text-gray-700 mb-2">{t.scriptPreview || '脚本预览'}</span>
                <pre class="bg-gray-900 text-gray-300 text-[11px] p-2 rounded-lg overflow-auto max-h-40 font-mono">{selectedUserdataTemplate?.script || ''}</pre>
              </div>
              <button
                class="w-full h-10 mt-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
                onclick={uploadAndShowCommand}
                disabled={!selectedUserdataTemplate?.script || !activeSession?.connected || uploading}
              >
                {#if uploading}
                  <span class="flex items-center justify-center gap-2">
                    <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    {t.uploading || '上传中...'}
                  </span>
                {:else}
                  {t.uploadAndExec || '上传并执行'}
                {/if}
              </button>
            {/if}
          {/if}
        </div>
      </div>
    {/if}

    <!-- Right panel: Port forwarding -->
    {#if rightPanel === 'portForward'}
      <div class="w-80 flex-shrink-0 bg-white border-l border-gray-200 flex flex-col overflow-hidden">
        <div class="flex items-center justify-between px-4 py-3 border-b border-gray-200">
          <h3 class="text-[13px] font-semibold text-gray-900">{t.sshPortForward || '端口转发'}</h3>
          <button
            class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors cursor-pointer"
            onclick={() => rightPanel = 'none'}
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-3 space-y-2">
          {#if portForwards.length === 0}
            <p class="text-[12px] text-gray-400 text-center py-4">{t.sshNoForwards || '暂无活跃的端口转发'}</p>
          {/if}
          {#each portForwards as pf}
            <div class="bg-gray-50 rounded-lg p-3 border border-gray-200">
              <div class="flex items-center justify-between">
                <div class="min-w-0">
                  <p class="text-[12px] font-mono text-red-600 truncate">
                    localhost:{pf.localPort} → {pf.remoteHost}:{pf.remotePort}
                  </p>
                  <p class="text-[10px] text-gray-400 mt-0.5 truncate">Case: {pf.caseId?.substring(0, 12)}</p>
                </div>
                <button
                  class="ml-2 flex-shrink-0 px-2 py-1 text-[11px] text-red-500 hover:text-red-700 hover:bg-red-50 rounded transition-colors cursor-pointer"
                  onclick={() => handleStopPortForward(pf.id)}
                >
                  {t.sshStopForward || '停止'}
                </button>
              </div>
            </div>
          {/each}
        </div>

        <div class="flex-shrink-0 border-t border-gray-200 p-3 space-y-2">
          <p class="text-[12px] font-medium text-gray-700">{t.sshStartForward || '新建转发'}</p>
          {#if activeSession}
            <div class="flex items-center gap-2 px-3 py-1.5 bg-gray-50 border border-gray-200 rounded-lg">
              <span class="text-[11px] text-gray-400 flex-shrink-0">{t.sshCurrentSession || '当前会话'}:</span>
              <span class="text-[12px] text-red-600 font-medium truncate">{sessionLabel(activeSession)}</span>
            </div>
          {:else}
            <input
              type="text"
              class="w-full px-3 py-1.5 bg-gray-50 border border-gray-300 rounded-lg text-[12px] text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-red-500"
              placeholder={t.sshCaseId || '场景/部署 ID'}
              bind:value={pfCaseId}
            />
          {/if}
          <div class="grid grid-cols-2 gap-2">
            <input
              type="number"
              class="px-3 py-1.5 bg-gray-50 border border-gray-300 rounded-lg text-[12px] text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-red-500"
              placeholder={t.sshLocalPort || '本地端口'}
              bind:value={pfLocalPort}
            />
            <input
              type="number"
              class="px-3 py-1.5 bg-gray-50 border border-gray-300 rounded-lg text-[12px] text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-red-500"
              placeholder={t.sshRemotePort || '远程端口'}
              bind:value={pfRemotePort}
            />
          </div>
          <input
            type="text"
            class="w-full px-3 py-1.5 bg-gray-50 border border-gray-300 rounded-lg text-[12px] text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-red-500"
            placeholder={t.sshRemoteHost || '远程主机 (默认 127.0.0.1)'}
            bind:value={pfRemoteHost}
          />
          {#if pfError}
            <p class="text-[11px] text-red-500">{pfError}</p>
          {/if}
          <button
            class="w-full h-10 bg-red-600 hover:bg-red-700 text-white text-[13px] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            onclick={handleStartPortForward}
            disabled={pfLoading}
          >
            {#if pfLoading}
              <span class="flex items-center justify-center gap-2">
                <svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              </span>
            {:else}
              {t.sshStartForward || '开始转发'}
            {/if}
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>

<!-- New session dialog -->
{#if showNewSessionDialog}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
    onclick={(e) => { if (e.target === e.currentTarget) showNewSessionDialog = false; }}
    onkeydown={(e) => { if (e.key === 'Escape') showNewSessionDialog = false; }}
  >
    <div class="bg-white rounded-xl border border-gray-100 w-full max-w-md p-5 shadow-2xl">
      <h3 class="text-[15px] font-semibold text-gray-900 mb-4">{t.sshNewSession || '新建会话'}</h3>

      {#if multiInstances.length > 0 && multiInstanceCase}
        <!-- Multi-instance picker -->
        <div class="mb-4">
          <div class="flex items-center gap-2 mb-3">
            <button
              class="text-[12px] text-gray-500 hover:text-red-600 transition-colors cursor-pointer"
              onclick={() => { multiInstances = []; multiInstanceCase = null; }}
            >
              ← {t.sshBackToCases || '返回场景列表'}
            </button>
          </div>
          <label class="block text-[12px] font-medium text-gray-700 mb-2">
            {multiInstanceCase.name || multiInstanceCase.id.substring(0, 16)} — {t.sshSelectInstance || '选择实例'} ({multiInstances.length})
          </label>
          <div class="space-y-1 max-h-60 overflow-y-auto">
            {#each multiInstances as info, i}
              <button
                class="w-full flex items-center gap-3 px-3 py-2.5 text-left rounded-lg border border-gray-200 hover:border-red-300 hover:bg-red-50 transition-colors cursor-pointer group"
                onclick={() => selectInstance(multiInstanceCase, info, i)}
              >
                <span class="flex-shrink-0 w-6 h-6 rounded-full bg-gray-100 text-gray-600 text-[11px] font-medium flex items-center justify-center">#{i + 1}</span>
                <div class="flex-1 min-w-0">
                  <div class="text-[13px] font-mono text-gray-900">{info.host}</div>
                  <div class="text-[11px] text-gray-400">{info.user}@{info.host}:{info.port || 22}</div>
                </div>
                <svg class="w-4 h-4 text-gray-300 group-hover:text-red-500 flex-shrink-0 transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7"/></svg>
              </button>
            {/each}
          </div>
        </div>
        <div class="flex items-center justify-between border-t border-gray-100 pt-3">
          <span class="text-[11px] text-gray-400">{t.sshMultiInstanceHint || '点击单个实例连接，或一键全部连接'}</span>
          <button
            class="h-9 px-4 text-[12px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors cursor-pointer"
            onclick={() => connectAllInstances(multiInstanceCase, multiInstances)}
          >
            {t.sshConnectAll || '全部连接'}
          </button>
        </div>
      {:else if !showManualInput}
        <!-- Active cases list -->
        <div class="mb-4">
          <label class="block text-[12px] font-medium text-gray-700 mb-2">{t.sshActiveCases || '运行中的场景'}</label>
          {#if casesLoading}
            <div class="flex items-center justify-center py-6 text-gray-400 text-[13px]">
              <svg class="animate-spin h-4 w-4 mr-2" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
              {t.loading || '加载中...'}
            </div>
          {:else if availableCases.length === 0}
            <div class="text-center py-4 text-gray-400 text-[13px]">{t.sshNoCasesRunning || '暂无运行中的场景'}</div>
          {:else}
            <div class="max-h-60 overflow-y-auto space-y-1">
              {#each availableCases as c}
                <button
                  class="w-full flex items-center gap-3 px-3 py-2.5 text-left rounded-lg border border-gray-200 hover:border-red-300 hover:bg-red-50 transition-colors cursor-pointer group"
                  onclick={() => selectCase(c)}
                >
                  <span class="flex-shrink-0 w-2 h-2 rounded-full bg-green-500"></span>
                  <div class="flex-1 min-w-0">
                    <div class="text-[13px] font-medium text-gray-900 truncate">{c.name || c.id.substring(0, 16)}</div>
                    <div class="text-[11px] text-gray-400 truncate font-mono">{c.id.substring(0, 24)}...</div>
                  </div>
                  <span class="text-[11px] text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded flex-shrink-0">{c.type === 'predefined' ? (t.predefined || '预定义') : (t.custom || '自定义')}</span>
                  <svg class="w-4 h-4 text-gray-300 group-hover:text-red-500 flex-shrink-0 transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7"/></svg>
                </button>
              {/each}
            </div>
          {/if}
        </div>
        <div class="flex items-center justify-between border-t border-gray-100 pt-3">
          <button
            class="text-[12px] text-gray-500 hover:text-red-600 transition-colors cursor-pointer"
            onclick={() => showManualInput = true}
          >
            {t.sshManualInput || '手动输入 ID →'}
          </button>
          <button
            class="h-9 px-4 text-[13px] font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
            onclick={() => showNewSessionDialog = false}
          >
            {t.cancel || '取消'}
          </button>
        </div>
      {:else}
        <!-- Manual input fallback -->
        <div class="space-y-3">
          <div>
            <label class="block text-[12px] font-medium text-gray-700 mb-1">{t.sshCaseId || '场景/部署 ID'}</label>
            <input
              type="text"
              class="w-full px-3 py-2 text-[13px] border border-gray-300 rounded-lg text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-red-500"
              placeholder={t.sshCaseIdPlaceholder || '输入场景或部署 ID'}
              bind:value={newSessionCaseId}
              onkeydown={(e) => { if (e.key === 'Enter') handleNewSessionSubmit(); }}
            />
          </div>
          <div>
            <label class="block text-[12px] font-medium text-gray-700 mb-1">{t.sshDisplayName || '显示名称 (可选)'}</label>
            <input
              type="text"
              class="w-full px-3 py-2 text-[13px] border border-gray-300 rounded-lg text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-red-500"
              placeholder={t.sshDisplayNamePlaceholder || '在标签页中显示的名称'}
              bind:value={newSessionCaseName}
              onkeydown={(e) => { if (e.key === 'Enter') handleNewSessionSubmit(); }}
            />
          </div>
        </div>
        <div class="flex items-center justify-between mt-5">
          <button
            class="text-[12px] text-gray-500 hover:text-red-600 transition-colors cursor-pointer"
            onclick={() => showManualInput = false}
          >
            ← {t.sshBackToCases || '返回场景列表'}
          </button>
          <div class="flex gap-2">
            <button
              class="h-10 px-5 text-[13px] font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
              onclick={() => showNewSessionDialog = false}
            >
              {t.cancel || '取消'}
            </button>
            <button
              class="h-10 px-5 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
              onclick={handleNewSessionSubmit}
              disabled={!newSessionCaseId.trim()}
            >
              {t.sshConnect || '连接'}
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- File Manager Modal -->
{#if showFileManagerModal && activeSession}
  <FileManager {t} caseId={activeSession.caseId} caseName={activeSession.caseName} onClose={() => showFileManagerModal = false} />
{/if}

<!-- Close All Confirmation -->
{#if showCloseAllConfirm}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
    onclick={(e) => { if (e.target === e.currentTarget) showCloseAllConfirm = false; }}
    onkeydown={(e) => { if (e.key === 'Escape') showCloseAllConfirm = false; }}
  >
    <div class="bg-white rounded-xl border border-gray-100 w-full max-w-xs p-5 shadow-2xl">
      <h3 class="text-[15px] font-semibold text-gray-900 mb-2">{t.sshCloseAll || '关闭所有连接'}</h3>
      <p class="text-[13px] text-gray-500 mb-5">{t.sshCloseAllConfirm || `确定关闭所有 ${sessions.length} 个 SSH 连接吗？`}</p>
      <div class="flex justify-end gap-2">
        <button
          class="h-9 px-4 text-[13px] font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
          onclick={() => showCloseAllConfirm = false}
        >{t.cancel || '取消'}</button>
        <button
          class="h-9 px-4 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors cursor-pointer"
          onclick={closeAllSessions}
        >{t.confirm || '确定'}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .ssh-terminal-container :global(.xterm) {
    height: 100%;
    padding: 8px;
  }

  .ssh-terminal-container :global(.xterm-viewport) {
    overflow-y: auto;
  }
</style>
