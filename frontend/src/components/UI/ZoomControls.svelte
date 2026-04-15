<script>
  let {
    zoom = $bindable(1),
    min = 0.3,
    max = 3,
    step = 0.15,
    t = {},
  } = $props();
</script>

<div class="absolute top-2 right-2 z-10 flex items-center gap-1 bg-white/90 backdrop-blur rounded-lg border border-gray-200 shadow-sm px-1 py-0.5">
  <button class="w-7 h-7 flex items-center justify-center text-gray-500 hover:text-gray-800 hover:bg-gray-100 rounded transition-colors cursor-pointer" onclick={() => zoom = Math.max(min, zoom - step)} aria-label={t.zoomOut || '缩小'} title={t.zoomOut || '缩小'}>
    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" d="M5 12h14" /></svg>
  </button>
  <span class="text-[11px] text-gray-400 min-w-[36px] text-center">{Math.round(zoom * 100)}%</span>
  <button class="w-7 h-7 flex items-center justify-center text-gray-500 hover:text-gray-800 hover:bg-gray-100 rounded transition-colors cursor-pointer" onclick={() => zoom = Math.min(max, zoom + step)} aria-label={t.zoomIn || '放大'} title={t.zoomIn || '放大'}>
    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" d="M12 5v14m-7-7h14" /></svg>
  </button>
  <button class="w-7 h-7 flex items-center justify-center text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded transition-colors cursor-pointer" onclick={() => zoom = 1} aria-label={t.zoomReset || '重置'} title={t.zoomReset || '重置'}>
    <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 9V4.5M9 9H4.5M9 9L3.75 3.75M9 15v4.5M9 15H4.5M9 15l-5.25 5.25M15 9h4.5M15 9V4.5M15 9l5.25-5.25M15 15h4.5M15 15v4.5m0-4.5l5.25 5.25" /></svg>
  </button>
</div>
