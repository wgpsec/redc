<script>
  let { t, inputText = $bindable(''), isStreaming = false, mode = 'free', onSend = () => {}, onStop = () => {} } = $props();
</script>

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
          onSend();
        }
      }}
      disabled={isStreaming}
    ></textarea>
    <button
      class="px-4 h-10 bg-gray-900 text-white text-[12px] font-medium rounded-xl hover:bg-gray-800 transition-colors disabled:opacity-50 flex items-center gap-2 cursor-pointer flex-shrink-0"
      onclick={onSend}
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
    {#if isStreaming}
      <button
        class="px-3 h-10 bg-red-600 text-white text-[12px] font-medium rounded-xl hover:bg-red-700 transition-colors flex items-center gap-1.5 cursor-pointer flex-shrink-0"
        onclick={onStop}
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <rect x="6" y="6" width="12" height="12" rx="1" />
        </svg>
        {t.aiChatStop || '停止'}
      </button>
    {/if}
  </div>
</div>
