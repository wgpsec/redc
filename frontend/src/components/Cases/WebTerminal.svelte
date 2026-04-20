<script>
  import { onMount, onDestroy } from 'svelte';
  import { StartSSHTerminal, WriteToTerminal, ResizeTerminal, CloseTerminal, UploadUserdataScript } from '../../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime.js';
  import { loadUserdataTemplates, getGroupedTemplates, userdataCategoryNames } from '../../lib/userdataTemplates.js';
  import Modal from '../UI/Modal.svelte';

  let { t, caseId, caseName, onClose } = $props();

  let terminalContainer = $state(null);
  let terminal = $state(null);
  let fitAddon = $state(null);
  let sessionId = $state(null);
  let connected = $state(false);
  let connecting = $state(false);
  let error = $state('');

  // Userdata 面板状态
  let showUserdataPanel = $state(false);
  let userdataTemplates = $state([]);
  let userdataTemplatesLoading = $state(true);
  let selectedUserdataTemplate = $state(null);
  let uploadResult = $state(null);
  let uploading = $state(false);
  let expandedCategories = $state({});

  // Grouped templates by category
  let groupedUserdataTemplates = $derived(() => {
    return getGroupedTemplates(userdataTemplates);
  });

  onMount(() => {
    initTerminal();
  });

  async function initTerminal() {
    // 动态导入 xterm
    try {
      const { Terminal } = await import('xterm');
      const { FitAddon } = await import('xterm-addon-fit');
      
      // 创建终端实例
      terminal = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        theme: {
          background: '#111827',
          foreground: '#d1d5db',
          cursor: '#60a5fa',
          black: '#1f2937',
          red: '#f87171',
          green: '#4ade80',
          yellow: '#fbbf24',
          blue: '#60a5fa',
          magenta: '#c084fc',
          cyan: '#22d3ee',
          white: '#9ca3af',
          brightBlack: '#4b5563',
          brightRed: '#fca5a5',
          brightGreen: '#86efac',
          brightYellow: '#fcd34d',
          brightBlue: '#93c5fd',
          brightMagenta: '#d8b4fe',
          brightCyan: '#67e8f9',
          brightWhite: '#f3f4f6',
        },
        allowProposedApi: true,
      });

      // 添加 fit addon
      fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);

      // 挂载到 DOM
      terminal.open(terminalContainer);
      fitAddon.fit();

      // 监听用户输入
      terminal.onData((data) => {
        if (sessionId && connected) {
          WriteToTerminal(sessionId, data).catch(err => {
            console.error('写入终端失败:', err);
          });
        }
      });

      // 监听窗口大小变化
      const resizeObserver = new ResizeObserver(() => {
        if (fitAddon && terminal) {
          fitAddon.fit();
          if (sessionId && connected) {
            ResizeTerminal(sessionId, terminal.rows, terminal.cols).catch(err => {
              console.error('调整终端大小失败:', err);
            });
          }
        }
      });
      resizeObserver.observe(terminalContainer);

      // 连接到 SSH
      await connectSSH();

      // 加载 userdata 模板
      userdataTemplates = await loadUserdataTemplates();
      userdataTemplatesLoading = false;
    } catch (err) {
      error = `${t.loadTerminalFailed || '加载终端失败'}: ${err.message}`;
      console.error('加载终端失败:', err);
    }
  }

  onDestroy(() => {
    cleanup();
    if (terminalContainer) {
      // 清理 resize observer
    }
  });

  async function uploadAndShowCommand() {
    if (!selectedUserdataTemplate?.script || !connected) return;

    uploading = true;
    uploadResult = null;

    const fileName = `${selectedUserdataTemplate.name || 'userdata'}.sh`;
    
    try {
      const result = await UploadUserdataScript(caseId, selectedUserdataTemplate.script, fileName);
      uploadResult = result;
      
      if (result.success) {
        terminal?.writeln(`\r\n\x1b[32m${t.scriptUploadedTo || '脚本已上传到'}: /tmp/${fileName}\x1b[0m`);
        terminal?.writeln(`\x1b[33m${t.execScriptCmd || '执行命令'}: bash /tmp/${fileName}\x1b[0m\r\n`);
      } else {
        terminal?.writeln(`\r\n\x1b[31m${t.uploadFailed || '上传失败'}: ${result.error}\x1b[0m\r\n`);
      }
    } catch (err) {
      uploadResult = { success: false, error: err.message };
      terminal?.writeln(`\r\n\x1b[31m${t.uploadFailed || '上传失败'}: ${err.message}\x1b[0m\r\n`);
    } finally {
      uploading = false;
    }
  }

  async function connectSSH() {
    if (connecting || connected) return;

    connecting = true;
    error = '';
    terminal?.writeln('\x1b[1;34m' + (t.sshConnecting || '正在连接 SSH...') + '\x1b[0m');

    try {
      // 启动 SSH 终端会话
      sessionId = await StartSSHTerminal(caseId, terminal.rows, terminal.cols);
      
      // 监听终端输出
      EventsOn(`terminal-output-${sessionId}`, (data) => {
        terminal?.write(data);
      });

      // 监听终端错误
      EventsOn(`terminal-error-${sessionId}`, (err) => {
        terminal?.writeln(`\r\n\x1b[1;31m${t.terminalError || '错误'}: ${err}\x1b[0m`);
        error = err;
      });

      // 监听终端关闭
      EventsOn(`terminal-closed-${sessionId}`, () => {
        terminal?.writeln('\r\n\x1b[1;33m' + (t.sshConnectionClosed || '连接已关闭') + '\x1b[0m');
        connected = false;
      });

      connected = true;
      terminal?.writeln('\x1b[1;32m' + (t.sshConnected || '已连接') + '\x1b[0m\r\n');
    } catch (err) {
      error = err.message || String(err);
      terminal?.writeln(`\r\n\x1b[1;31m${t.connectFailed || '连接失败'}: ${error}\x1b[0m`);
    } finally {
      connecting = false;
    }
  }

  function cleanup() {
    if (sessionId) {
      EventsOff(`terminal-output-${sessionId}`);
      EventsOff(`terminal-error-${sessionId}`);
      EventsOff(`terminal-closed-${sessionId}`);
      CloseTerminal(sessionId).catch(err => {
        console.error('关闭终端失败:', err);
      });
    }
    terminal?.dispose();
  }

  function handleClose() {
    cleanup();
    onClose();
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      handleClose();
    }
  }

  function handleBackdropClick(e) {
    if (e.target === e.currentTarget) {
      handleClose();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<!-- 导入 xterm 样式 -->
<svelte:head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@5.3.0/css/xterm.css" />
</svelte:head>

<Modal show={true} onclose={handleClose} class="p-4">
  <div class="bg-gray-900 rounded-xl border border-gray-700 w-[calc(100vw-2rem)] max-w-6xl h-[85vh] overflow-hidden flex flex-col" onclick={(e) => e.stopPropagation()}>
    <!-- Header -->
    <div class="px-5 py-4 border-b border-gray-700 flex items-center justify-between flex-shrink-0">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-lg bg-emerald-600 flex items-center justify-center">
          <svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
          </svg>
        </div>
        <div>
          <h3 class="text-[15px] font-semibold text-white">{t.webTerminal || 'Web 终端'}</h3>
          <p class="text-[12px] text-gray-400 mt-0.5">{caseName} <span class="text-gray-500">({caseId?.substring(0, 8)})</span></p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <button
          class="px-3 py-1.5 text-[12px] font-medium text-gray-300 bg-gray-800 rounded-lg hover:bg-gray-700 transition-colors flex items-center gap-2"
          onclick={() => showUserdataPanel = !showUserdataPanel}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12c0 1.268-.63 2.39-1.593 3.068a3.745 3.745 0 01-1.043 3.296 3.745 3.745 0 01-3.296 1.043A3.745 3.745 0 0112 21c-1.268 0-2.39-.63-3.068-1.593a3.746 3.746 0 01-3.296-1.043 3.745 3.745 0 01-1.043-3.296A3.745 3.745 0 013 12c0-1.268.63-2.39 1.593-3.068a3.745 3.745 0 011.043-3.296 3.746 3.746 0 013.296-1.043A3.746 3.746 0 0112 3c1.268 0 2.39.63 3.068 1.593a3.746 3.746 0 013.296 1.043 3.746 3.746 0 011.043 3.296A3.745 3.745 0 0121 12z" />
          </svg>
          {t.execUserdata || 'Userdata'}
        </button>
        {#if connected}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-emerald-500/20 text-emerald-400 text-[11px] font-medium rounded-full">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
            {t.sshConnected || '已连接'}
          </span>
        {:else if connecting}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-amber-500/20 text-amber-400 text-[11px] font-medium rounded-full">
            <svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {t.connecting || '连接中...'}
          </span>
        {:else}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-gray-700 text-gray-400 text-[11px] font-medium rounded-full">
            <span class="w-1.5 h-1.5 rounded-full bg-gray-500"></span>
            {t.disconnected || '未连接'}
          </span>
        {/if}
        <button
          class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-800 text-gray-400 hover:text-gray-200 transition-colors"
          onclick={handleClose}
          aria-label={t.close}
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Terminal Container -->
    <div class="flex-1 overflow-hidden p-4">
      <div bind:this={terminalContainer} class="w-full h-full rounded-lg overflow-hidden"></div>
    </div>

    <!-- Userdata Panel -->
    {#if showUserdataPanel}
      <div class="border-t border-gray-700 bg-gray-800 p-4 max-h-64 overflow-auto">
        {#if userdataTemplatesLoading}
          <div class="text-center py-4 text-gray-400 text-[13px]">
            {t.loading || '加载中...'}
          </div>
        {:else if userdataTemplates.length === 0}
          <div class="text-center py-4 text-gray-400 text-[13px]">
            {t.noTemplates || '暂无可用模板'}
          </div>
        {:else}
          <div class="space-y-3">
            <div class="space-y-2 max-h-48 overflow-y-auto">
              {#each Object.entries(groupedUserdataTemplates()) as [category, categoryTemplates]}
                <div class="border border-gray-600 rounded-lg overflow-hidden">
                  <button
                    class="w-full flex items-center justify-between px-3 py-2 bg-gray-700 hover:bg-gray-600 transition-colors"
                    onclick={() => expandedCategories[category] = !expandedCategories[category]}
                  >
                    <span class="text-[12px] font-medium text-gray-300">
                      {userdataCategoryNames[category] || category}
                      <span class="ml-1 text-gray-500">({categoryTemplates.length})</span>
                    </span>
                    <svg class="w-4 h-4 text-gray-500 transition-transform {expandedCategories[category] ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                    </svg>
                  </button>
                  {#if expandedCategories[category]}
                    <div class="space-y-1 p-2 bg-gray-800">
                      {#each categoryTemplates as template}
                        <button
                          class="w-full text-left px-3 py-2 text-[12px] bg-gray-700 hover:bg-gray-600 border border-gray-600 hover:border-emerald-500 rounded transition-colors {selectedUserdataTemplate === template ? 'border-emerald-500 bg-emerald-900/30' : ''}"
                          onclick={() => selectedUserdataTemplate = template}
                        >
                          <span class="font-medium text-gray-200">{template.nameZh || template.name}</span>
                          {#if template.description}
                            <span class="block text-gray-500 text-[10px] truncate">{template.description}</span>
                          {/if}
                        </button>
                      {/each}
                    </div>
                  {/if}
                </div>
              {/each}
            </div>

            {#if selectedUserdataTemplate}
              <div>
                <span class="block text-[12px] font-medium text-gray-300 mb-2">{t.scriptPreview || '脚本预览'}</span>
                <pre class="bg-gray-900 text-gray-300 text-[11px] p-2 rounded-lg overflow-auto max-h-32 font-mono">{selectedUserdataTemplate?.script || ''}</pre>
              </div>
              <button
                class="w-full px-4 py-2.5 text-[13px] font-medium text-white bg-emerald-600 rounded-lg hover:bg-emerald-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                onclick={uploadAndShowCommand}
                disabled={!selectedUserdataTemplate?.script || !connected || uploading}
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
          </div>
        {/if}
      </div>
    {/if}

    <!-- Error Display -->
    {#if error}
      <div class="px-5 py-3 border-t border-gray-700 bg-red-500/10">
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4 text-red-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <span class="text-[12px] text-red-400">{error}</span>
        </div>
      </div>
    {/if}
  </div>
</Modal>

<style>
  :global(.xterm) {
    height: 100%;
    padding: 8px;
  }
  
  :global(.xterm-viewport) {
    overflow-y: auto;
  }
</style>
