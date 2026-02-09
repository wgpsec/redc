<script>
  let { logs = $bindable([]), t = {} } = $props();
  
  function stripAnsi(value) {
    if (!value) return '';
    return value.replace(/\x1B\[[0-9;]*m/g, '');
  }

  export function clearLogs() {
    logs = [];
  }
</script>

<div class="h-full flex flex-col bg-[#1e1e1e] rounded-xl overflow-hidden">
  <div class="flex items-center justify-between px-4 py-2.5 bg-[#252526] border-b border-[#3c3c3c]">
    <div class="flex items-center gap-2">
      <div class="flex gap-1.5">
        <span class="w-3 h-3 rounded-full bg-[#ff5f56]"></span>
        <span class="w-3 h-3 rounded-full bg-[#ffbd2e]"></span>
        <span class="w-3 h-3 rounded-full bg-[#27ca40]"></span>
      </div>
      <span class="text-[12px] text-gray-500 ml-2">{t.terminal}</span>
    </div>
    <button 
      class="text-[11px] text-gray-500 hover:text-gray-300 transition-colors"
      onclick={clearLogs}
    >{t.clear}</button>
  </div>
  <div class="flex-1 p-4 overflow-auto font-mono text-[12px] leading-5">
    {#each logs as log}
      <div class="flex">
        <span class="text-gray-600 select-none">[{log.time}]</span>
        <span class="text-gray-300 ml-2">{stripAnsi(log.message)}</span>
      </div>
    {:else}
      <div class="text-gray-600">$ {t.waitOutput}</div>
    {/each}
  </div>
</div>
