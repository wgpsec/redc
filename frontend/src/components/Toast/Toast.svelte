<script>
  import { onMount, onDestroy } from 'svelte';
  import { onToastChange, getToasts, removeToast } from '../../lib/toast.js';

  let toasts = $state(getToasts());
  let unsubscribe = null;

  onMount(() => {
    unsubscribe = onToastChange((list) => { toasts = list; });
  });
  onDestroy(() => { if (unsubscribe) unsubscribe(); });

  const icons = {
    success: 'M4.5 12.75l6 6 9-13.5',
    error: 'M6 18L18 6M6 6l12 12',
    warning: 'M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z',
    info: 'M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z',
  };

  const styles = {
    success: { bg: 'bg-emerald-50', border: 'border-emerald-200', icon: 'text-emerald-500', text: 'text-emerald-800' },
    error: { bg: 'bg-red-50', border: 'border-red-200', icon: 'text-red-500', text: 'text-red-800' },
    warning: { bg: 'bg-amber-50', border: 'border-amber-200', icon: 'text-amber-500', text: 'text-amber-800' },
    info: { bg: 'bg-blue-50', border: 'border-blue-200', icon: 'text-blue-500', text: 'text-blue-800' },
  };
</script>

{#if toasts.length > 0}
  <div class="fixed top-4 right-4 z-[9999] flex flex-col gap-2 pointer-events-none" style="max-width: 380px;">
    {#each toasts as t (t.id)}
      {@const s = styles[t.type] || styles.info}
      <div class="pointer-events-auto flex items-start gap-2.5 px-4 py-3 rounded-xl border shadow-lg backdrop-blur-sm animate-toast-in {s.bg} {s.border}">
        <svg class="w-4 h-4 flex-shrink-0 mt-0.5 {s.icon}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d={icons[t.type] || icons.info} />
        </svg>
        <span class="flex-1 text-[12px] leading-relaxed {s.text}">{t.message}</span>
        <button
          class="flex-shrink-0 p-0.5 rounded hover:bg-black/5 transition-colors cursor-pointer {s.text} opacity-50 hover:opacity-100"
          onclick={() => removeToast(t.id)}
          aria-label="Close"
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    {/each}
  </div>
{/if}

<style>
  @keyframes toastIn {
    from { opacity: 0; transform: translateX(20px); }
    to { opacity: 1; transform: translateX(0); }
  }
  :global(.animate-toast-in) {
    animation: toastIn 0.25s ease-out;
  }
</style>
