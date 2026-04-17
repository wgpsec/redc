<script>
  import ToolCallCard from './ToolCallCard.svelte';

  let { msg, t, renderMarkdown = () => '', formatTime = () => '', getToolDisplayName = () => '', formatToolArgs = () => '', onCopy = () => {}, onSaveTemplate = () => {} } = $props();
</script>

{#if msg.role === 'system-notice'}
  <div class="flex justify-center">
    <div class="px-3 py-1.5 rounded-full bg-blue-50 border border-blue-100 flex items-center gap-1.5">
      <svg class="w-3 h-3 text-blue-500 shrink-0" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M4 2v12M12 2v12M1 5l3-3M1 11l3 3M15 5l-3-3M15 11l-3 3M4 8h8"/>
      </svg>
      <p class="text-[11px] text-blue-600">{msg.content}</p>
    </div>
  </div>
{:else if msg.role === 'user'}
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
  <div class="flex justify-start">
    <div class="max-w-[85%]">
      <div class="flex items-start gap-2.5">
        <div class="w-7 h-7 rounded-lg bg-rose-600 flex items-center justify-center flex-shrink-0 mt-0.5">
          <svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
          </svg>
        </div>
        <div class="flex-1 min-w-0">
          <!-- Saved plan card -->
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
          <!-- Saved tool call cards -->
          {#if msg.toolCalls && msg.toolCalls.length > 0}
            <div class="mb-2 space-y-1.5">
              {#each msg.toolCalls as tc}
                <ToolCallCard {tc} {t} {getToolDisplayName} {formatToolArgs} />
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
                onclick={() => onCopy(msg.content)}
              >
                <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9.75a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                </svg>
                {t.aiChatCopyContent || '复制'}
              </button>
              {#if msg.content && (msg.content.includes('main.tf') || msg.content.includes('case.json') || msg.content.includes('```hcl'))}
                <button
                  class="flex items-center gap-1 px-2 py-1 rounded text-[11px] text-gray-400 hover:text-rose-600 hover:bg-rose-50 transition-colors cursor-pointer"
                  onclick={() => onSaveTemplate(msg.content)}
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
