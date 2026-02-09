<script>
  import { onMount } from 'svelte';
  import { GetMCPStatus, StartMCPServer, StopMCPServer } from '../../../wailsjs/go/main/App.js';

  // MCP state
  let { t } = $props();
  let mcpStatus = $state({ running: false, mode: '', address: '', protocolVersion: '' });
  let mcpForm = $state({ mode: 'sse', address: 'localhost:8080' });
  let mcpLoading = $state(false);
  let error = $state('');

  onMount(async () => {
    await loadMCPStatus();
  });

  async function loadMCPStatus() {
    try {
      mcpStatus = await GetMCPStatus();
    } catch (e) {
      console.error('Failed to load MCP status:', e);
    }
  }

  async function handleStartMCP() {
    mcpLoading = true;
    try {
      mcpForm.mode = 'sse';
      await StartMCPServer(mcpForm.mode, mcpForm.address);
      await loadMCPStatus();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      mcpLoading = false;
    }
  }

  async function handleStopMCP() {
    mcpLoading = true;
    try {
      await StopMCPServer();
      await loadMCPStatus();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      mcpLoading = false;
    }
  }

</script>

<div class="w-full max-w-2xl mx-auto space-y-4 sm:space-y-5">
  <!-- Error display -->
  {#if error}
    <div class="flex items-center gap-3 px-3 sm:px-4 py-2.5 sm:py-3 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[12px] sm:text-[13px] text-red-700 flex-1">{error}</span>
      <button class="text-red-400 hover:text-red-600" onclick={() => error = ''}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

<!-- MCP Status Card -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-0 mb-4">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 sm:w-10 sm:h-10 rounded-lg bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center">
          <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
          </svg>
        </div>
        <div>
          <h2 class="text-[13px] sm:text-[14px] font-semibold text-gray-900">{t.mcpServer}</h2>
          <p class="text-[11px] sm:text-[12px] text-gray-500">{t.mcpDesc}</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        {#if mcpStatus.running}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-emerald-50 text-emerald-600 text-[11px] sm:text-[12px] font-medium rounded-full">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
            {t.running}
          </span>
        {:else}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-gray-50 text-gray-500 text-[11px] sm:text-[12px] font-medium rounded-full">
            <span class="w-1.5 h-1.5 rounded-full bg-gray-400"></span>
            {t.stopped}
          </span>
        {/if}
      </div>
    </div>

    {#if mcpStatus.running}
      <!-- Running status info -->
      <div class="bg-gray-50 rounded-lg p-3 sm:p-4 mb-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4 text-[11px] sm:text-[12px]">
          <div>
            <span class="text-gray-500">{t.transportMode}</span>
            <p class="font-medium text-gray-900 mt-0.5">SSE (HTTP)</p>
          </div>
          <div>
            <span class="text-gray-500">{t.listenAddr}</span>
            <p class="font-mono font-medium text-gray-900 mt-0.5 break-all">{mcpStatus.address || '-'}</p>
          </div>
          <div>
            <span class="text-gray-500">{t.protocolVersion}</span>
            <p class="font-medium text-gray-900 mt-0.5">{mcpStatus.protocolVersion}</p>
          </div>
          <div>
            <span class="text-gray-500">{t.msgEndpoint}</span>
            <p class="font-mono font-medium text-gray-900 mt-0.5 text-[10px] sm:text-[11px] break-all">http://{mcpStatus.address}/message</p>
          </div>
        </div>
      </div>
      <button 
        class="w-full h-9 sm:h-10 bg-red-500 text-white text-[12px] sm:text-[13px] font-medium rounded-lg hover:bg-red-600 transition-colors disabled:opacity-50"
        onclick={handleStopMCP}
        disabled={mcpLoading}
      >
        {mcpLoading ? t.stoppingServer : t.stopServer}
      </button>
    {:else}
      <!-- Configuration form -->
      <div class="space-y-3 sm:space-y-4 mb-4">
        <div>
          <label class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.transportMode}</label>
          <div class="inline-flex items-center h-9 sm:h-10 px-3 sm:px-4 text-[12px] sm:text-[13px] font-medium rounded-lg border bg-gray-900 text-white border-gray-900">
            SSE (HTTP)
          </div>
        </div>
        <div>
          <label class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.listenAddr}</label>
          <input 
            type="text" 
            placeholder="localhost:8080" 
            class="w-full h-9 sm:h-10 px-3 text-[12px] sm:text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={mcpForm.address} 
          />
        </div>
      </div>
      <button 
        class="w-full h-9 sm:h-10 bg-gray-900 text-white text-[12px] sm:text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
        onclick={handleStartMCP}
        disabled={mcpLoading}
      >
        {mcpLoading ? t.startingServer : t.startServer}
      </button>
    {/if}
  </div>

  <!-- MCP Info Card -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <h3 class="text-[13px] sm:text-[14px] font-semibold text-gray-900 mb-3">{t.aboutMcp}</h3>
    <p class="text-[11px] sm:text-[12px] text-gray-600 leading-relaxed mb-4">
      {t.mcpInfo}
    </p>
    <div class="bg-gray-50 rounded-lg p-3 sm:p-4">
      <div class="text-[10px] sm:text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-2">{t.availableTools}</div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[11px] sm:text-[12px]">
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          list_templates
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          search_templates
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          pull_template
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          list_cases
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          plan_case
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          start_case
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          stop_case
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          kill_case
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          get_case_status
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          exec_command
        </div>
      </div>
    </div>
  </div>
</div>
