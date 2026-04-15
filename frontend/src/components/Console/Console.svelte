<script>
  import { onMount } from 'svelte';
  import { ExportConsoleLogs } from '../../../wailsjs/go/main/App.js';
  
  let { logs = $bindable([]), t = {} } = $props();
  
  let searchQuery = $state('');
  let levelFilter = $state('all'); // 'all' | 'error' | 'warn' | 'success'
  let autoScroll = $state(true);
  let showScrollBtn = $state(false);
  let copiedIdx = $state(-1);
  let logContainer;

  function stripAnsi(value) {
    if (!value) return '';
    return value.replace(/\x1B\[[0-9;]*m/g, '');
  }

  // Detect log level from message content
  function getLogLevel(message) {
    if (!message) return 'info';
    const msg = stripAnsi(message).toLowerCase();
    if (msg.includes('error') || msg.includes('failed') || msg.includes('failure') || msg.includes('fatal') || msg.includes('错误') || msg.includes('失败')) return 'error';
    if (msg.includes('warn') || msg.includes('warning') || msg.includes('警告')) return 'warn';
    if (msg.includes('success') || msg.includes('completed') || msg.includes('created') || msg.includes('成功') || msg.includes('完成') || msg.includes('apply complete')) return 'success';
    return 'info';
  }

  function getLevelColor(level) {
    switch (level) {
      case 'error': return 'text-red-400';
      case 'warn': return 'text-yellow-400';
      case 'success': return 'text-emerald-400';
      default: return 'text-gray-300';
    }
  }

  function getLevelDot(level) {
    switch (level) {
      case 'error': return 'bg-red-500';
      case 'warn': return 'bg-yellow-500';
      case 'success': return 'bg-emerald-500';
      default: return '';
    }
  }

  // Stats
  let errorCount = $derived(logs.filter(l => getLogLevel(l.message) === 'error').length);
  let warnCount = $derived(logs.filter(l => getLogLevel(l.message) === 'warn').length);

  // Filtered logs
  let filteredLogs = $derived.by(() => {
    let result = logs.map((log, idx) => ({ ...log, _idx: idx }));
    if (levelFilter !== 'all') {
      result = result.filter(l => getLogLevel(l.message) === levelFilter);
    }
    if (searchQuery.trim()) {
      const q = searchQuery.trim().toLowerCase();
      result = result.filter(l => stripAnsi(l.message).toLowerCase().includes(q));
    }
    return result;
  });

  // Time gap detection: insert separator if gap > 5 minutes
  function shouldShowTimeSep(currentLog, prevLog) {
    if (!prevLog || !currentLog) return false;
    try {
      const parseTime = (t) => {
        const parts = t.split(':');
        if (parts.length < 3) return 0;
        return parseInt(parts[0]) * 3600 + parseInt(parts[1]) * 60 + parseInt(parts[2]);
      };
      const diff = parseTime(currentLog.time) - parseTime(prevLog.time);
      return diff > 300 || diff < -300; // > 5 min or day wrap
    } catch { return false; }
  }

  // Auto-scroll
  $effect(() => {
    if (logs.length && autoScroll && logContainer) {
      requestAnimationFrame(() => {
        logContainer.scrollTop = logContainer.scrollHeight;
      });
    }
  });

  function handleScroll() {
    if (!logContainer) return;
    const { scrollTop, scrollHeight, clientHeight } = logContainer;
    const atBottom = scrollHeight - scrollTop - clientHeight < 40;
    autoScroll = atBottom;
    showScrollBtn = !atBottom && logs.length > 0;
  }

  function scrollToBottom() {
    if (logContainer) {
      logContainer.scrollTop = logContainer.scrollHeight;
      autoScroll = true;
      showScrollBtn = false;
    }
  }

  // Copy single log line
  function copyLog(log, idx) {
    const text = `[${log.time}] ${stripAnsi(log.message)}`;
    navigator.clipboard.writeText(text);
    copiedIdx = idx;
    setTimeout(() => { copiedIdx = -1; }, 1500);
  }

  // Export all logs via native save dialog
  async function exportLogs() {
    const text = logs.map(l => `[${l.time}] ${stripAnsi(l.message)}`).join('\n');
    try {
      await ExportConsoleLogs(text);
    } catch (e) {
      console.error('Export failed:', e);
    }
  }

  export function clearLogs() {
    logs = [];
  }
</script>

<div class="h-full flex flex-col bg-[#1e1e1e] rounded-xl overflow-hidden relative">
  <!-- Header bar -->
  <div class="flex items-center justify-between px-4 py-2 bg-[#252526] border-b border-[#3c3c3c]">
    <div class="flex items-center gap-2">
      <div class="flex gap-1.5">
        <span class="w-3 h-3 rounded-full bg-[#ff5f56]"></span>
        <span class="w-3 h-3 rounded-full bg-[#ffbd2e]"></span>
        <span class="w-3 h-3 rounded-full bg-[#27ca40]"></span>
      </div>
      <span class="text-[12px] text-gray-500 ml-2">{t.terminal}</span>
      <!-- Log counts -->
      {#if logs.length > 0}
        <span class="text-[10px] text-gray-600 ml-1 tabular-nums">{logs.length}</span>
        {#if errorCount > 0}
          <span class="text-[10px] text-red-400 tabular-nums">{errorCount} {t.consoleErrors || 'err'}</span>
        {/if}
        {#if warnCount > 0}
          <span class="text-[10px] text-yellow-400 tabular-nums">{warnCount} {t.consoleWarns || 'warn'}</span>
        {/if}
      {/if}
    </div>
    <div class="flex items-center gap-2">
      <!-- Search -->
      <div class="relative">
        <input 
          type="text"
          bind:value={searchQuery}
          placeholder={t.consoleSearch || '搜索日志...'}
          class="h-6 w-36 pl-6 pr-2 text-[11px] bg-[#3c3c3c] text-gray-300 rounded border border-[#4c4c4c] focus:border-gray-500 focus:outline-none placeholder-gray-600"
        />
        <svg class="w-3 h-3 text-gray-600 absolute left-2 top-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" /></svg>
      </div>
      <!-- Level filter pills -->
      <div class="flex items-center gap-0.5 bg-[#1e1e1e] rounded p-0.5">
        <button class="h-5 px-1.5 text-[10px] rounded transition-colors cursor-pointer {levelFilter === 'all' ? 'bg-[#3c3c3c] text-gray-300' : 'text-gray-600 hover:text-gray-400'}" onclick={() => levelFilter = 'all'}>{t.filterAll || '全部'}</button>
        <button class="h-5 px-1.5 text-[10px] rounded transition-colors cursor-pointer {levelFilter === 'error' ? 'bg-[#3c3c3c] text-red-400' : 'text-gray-600 hover:text-red-400'}" onclick={() => levelFilter = levelFilter === 'error' ? 'all' : 'error'}>ERR</button>
        <button class="h-5 px-1.5 text-[10px] rounded transition-colors cursor-pointer {levelFilter === 'warn' ? 'bg-[#3c3c3c] text-yellow-400' : 'text-gray-600 hover:text-yellow-400'}" onclick={() => levelFilter = levelFilter === 'warn' ? 'all' : 'warn'}>WARN</button>
        <button class="h-5 px-1.5 text-[10px] rounded transition-colors cursor-pointer {levelFilter === 'success' ? 'bg-[#3c3c3c] text-emerald-400' : 'text-gray-600 hover:text-emerald-400'}" onclick={() => levelFilter = levelFilter === 'success' ? 'all' : 'success'}>OK</button>
      </div>
      <!-- Export -->
      <button 
        class="h-6 w-6 flex items-center justify-center text-gray-600 hover:text-gray-300 transition-colors cursor-pointer rounded hover:bg-[#3c3c3c] disabled:opacity-30"
        onclick={exportLogs}
        disabled={logs.length === 0}
        title={t.consoleExport || '导出日志'}
      >
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
      </button>
      <!-- Clear -->
      <button 
        class="h-6 w-6 flex items-center justify-center text-gray-600 hover:text-gray-300 transition-colors cursor-pointer rounded hover:bg-[#3c3c3c] disabled:opacity-30"
        onclick={clearLogs}
        disabled={logs.length === 0}
        title={t.clear}
      >
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" /></svg>
      </button>
    </div>
  </div>

  <!-- Log area -->
  <div 
    class="flex-1 px-4 py-3 overflow-auto font-mono text-[12px] leading-5 relative"
    bind:this={logContainer}
    onscroll={handleScroll}
  >
    {#if filteredLogs.length === 0 && logs.length === 0}
      <!-- Empty state -->
      <div class="flex flex-col items-center justify-center h-full text-gray-400 gap-3">
        <svg class="w-10 h-10 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
        </svg>
        <div class="text-[12px]">$ {t.waitOutput}</div>
        <div class="text-[10px] text-gray-500">{t.consoleHint || '部署、启动、停止等操作的日志将在此显示'}</div>
      </div>
    {:else if filteredLogs.length === 0 && logs.length > 0}
      <!-- Filter empty -->
      <div class="flex flex-col items-center justify-center h-full text-gray-600 gap-2">
        <div class="text-[12px]">{t.consoleNoMatch || '无匹配日志'}</div>
        <button class="text-[11px] text-gray-500 hover:text-gray-300 cursor-pointer" onclick={() => { searchQuery = ''; levelFilter = 'all'; }}>{t.consoleClearFilter || '清除筛选'}</button>
      </div>
    {:else}
      {#each filteredLogs as log, i}
        <!-- Time separator -->
        {#if i > 0 && shouldShowTimeSep(log, filteredLogs[i - 1])}
          <div class="flex items-center gap-2 my-2">
            <div class="flex-1 border-t border-[#3c3c3c]"></div>
            <span class="text-[9px] text-gray-700 flex-shrink-0">{log.time}</span>
            <div class="flex-1 border-t border-[#3c3c3c]"></div>
          </div>
        {/if}
        {@const level = getLogLevel(log.message)}
        {@const dotClass = getLevelDot(level)}
        <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
        <div 
          class="flex items-start group hover:bg-[#2a2a2a] rounded px-1 -mx-1 cursor-pointer transition-colors"
          onclick={() => copyLog(log, log._idx)}
          title={t.consoleCopyLine || '点击复制'}
        >
          {#if dotClass}
            <span class="w-1.5 h-1.5 rounded-full {dotClass} mt-[7px] mr-1.5 flex-shrink-0"></span>
          {:else}
            <span class="w-1.5 mr-1.5 flex-shrink-0"></span>
          {/if}
          <span class="text-gray-600 select-none flex-shrink-0">[{log.time}]</span>
          <span class="{getLevelColor(level)} ml-2 break-all">{stripAnsi(log.message)}</span>
          <!-- Copy indicator -->
          <span class="ml-auto pl-2 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity text-gray-600">
            {#if copiedIdx === log._idx}
              <svg class="w-3 h-3 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" /></svg>
            {:else}
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0 0 13.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 0 1-.75.75H9.75a.75.75 0 0 1-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 0 1-2.25 2.25H6.75A2.25 2.25 0 0 1 4.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 0 1 1.927-.184" /></svg>
            {/if}
          </span>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Scroll to bottom button -->
  {#if showScrollBtn}
    <button 
      class="absolute bottom-4 right-6 w-8 h-8 bg-[#3c3c3c] hover:bg-[#4c4c4c] text-gray-400 hover:text-gray-200 rounded-full flex items-center justify-center shadow-lg transition-colors cursor-pointer z-10"
      onclick={scrollToBottom}
      title={t.consoleScrollBottom || '跳到底部'}
    >
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" /></svg>
    </button>
  {/if}
</div>
