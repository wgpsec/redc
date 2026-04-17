<script>
  let { tc, t, live = false, getToolDisplayName = () => '', formatToolArgs = () => '' } = $props();
</script>

{#if tc.toolName === 'ask_user'}
  <div class="flex items-start gap-2 px-3 py-2 rounded-lg border-2 {tc.status === 'success' ? 'bg-gray-50 border-gray-200' : 'bg-gray-50 border-gray-300'}">
    <span class="mt-0.5">
      {#if live && tc.status === 'calling'}
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
  <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg border {live ? (tc.status === 'success' ? 'bg-blue-50 border-blue-200' : tc.status === 'calling' ? 'bg-blue-50 border-blue-300' : 'bg-red-50 border-red-200') : 'bg-blue-50 border-blue-200'}">
    <span class="text-xs">
      {#if live && tc.status === 'calling'}
        <svg class="inline w-3 h-3 text-gray-400 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
      {:else}
        <svg class="inline w-3 h-3 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15a2.25 2.25 0 012.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25z" /></svg>
      {/if}
    </span>
    <span class="text-[11px] text-blue-700">{getToolDisplayName(tc.toolName)}</span>
    {#if tc.toolArgs?.title}
      <span class="text-[11px] text-blue-500">— {tc.toolArgs.title}</span>
    {/if}
    {#if tc.toolArgs?.steps}
      <span class="text-[10px] text-blue-400 ml-auto font-mono">{tc.toolArgs.steps.filter(s => s.status === 'done').length}/{tc.toolArgs.steps.length}</span>
    {/if}
  </div>
{:else}
  <div class="flex items-start gap-2 px-3 py-2 rounded-lg border {tc.status === 'success' ? 'bg-emerald-50 border-emerald-200' : tc.status === 'error' ? 'bg-red-50 border-red-200' : live ? 'bg-amber-50 border-amber-200' : 'bg-gray-50 border-gray-200'}">
    <span class="mt-0.5">
      {#if live && tc.status === 'calling'}
        <svg class="w-3.5 h-3.5 animate-spin text-amber-500" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
      {:else if tc.status === 'success'}
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
