<script>
  let {
    text = '',
    dismissKey = '',
  } = $props();

  let dismissed = $state(false);

  $effect(() => {
    if (dismissKey) {
      dismissed = localStorage.getItem(`pageGuide_${dismissKey}`) === 'dismissed';
    }
  });

  function handleDismiss() {
    dismissed = true;
    if (dismissKey) {
      localStorage.setItem(`pageGuide_${dismissKey}`, 'dismissed');
    }
  }
</script>

{#if !dismissed && text}
  <div class="bg-[#f8fafc] border border-[#e2e8f0] rounded-lg px-4 py-3 flex items-center gap-3 mb-4">
    <svg class="w-4 h-4 text-gray-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
      <path stroke-linecap="round" stroke-linejoin="round" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" />
    </svg>
    <span class="text-[12.5px] text-gray-500 leading-relaxed flex-1">{@html text}</span>
    <button
      class="text-gray-300 hover:text-gray-500 transition-colors cursor-pointer flex-shrink-0 p-0.5"
      onclick={handleDismiss}
      aria-label="关闭引导"
    >
      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
      </svg>
    </button>
  </div>
{/if}
