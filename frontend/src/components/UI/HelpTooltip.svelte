<script>
  let {
    text = '',
    position = 'right',
    maxWidth = 240,
    inline = false,
    children,
  } = $props();

  let show = $state(false);
  let triggerEl = $state(null);
  let tooltipEl = $state(null);
  let hoverTimer = $state(null);
  let tooltipStyle = $state('');

  function handleEnter() {
    hoverTimer = setTimeout(() => {
      show = true;
      // Wait for tooltip to render, then calculate position
      requestAnimationFrame(() => {
        requestAnimationFrame(updatePosition);
      });
    }, 100);
  }

  function handleLeave() {
    clearTimeout(hoverTimer);
    show = false;
  }

  function updatePosition() {
    if (!tooltipEl || !triggerEl) return;
    const tr = triggerEl.getBoundingClientRect();
    const tp = tooltipEl.getBoundingClientRect();
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    const gap = 8;

    let pos = position;
    // Flip if no room
    if (pos === 'top' && tr.top - tp.height - gap < 0) pos = 'bottom';
    else if (pos === 'bottom' && tr.bottom + tp.height + gap > vh) pos = 'top';
    else if (pos === 'left' && tr.left - tp.width - gap < 0) pos = 'right';
    else if (pos === 'right' && tr.right + tp.width + gap > vw) pos = 'left';

    let top, left;
    if (pos === 'top') {
      top = tr.top - tp.height - gap;
      left = tr.left + tr.width / 2 - tp.width / 2;
    } else if (pos === 'bottom') {
      top = tr.bottom + gap;
      left = tr.left + tr.width / 2 - tp.width / 2;
    } else if (pos === 'left') {
      top = tr.top + tr.height / 2 - tp.height / 2;
      left = tr.left - tp.width - gap;
    } else {
      top = tr.top + tr.height / 2 - tp.height / 2;
      left = tr.right + gap;
    }

    // Clamp to viewport with padding
    const pad = 8;
    if (left < pad) left = pad;
    if (left + tp.width > vw - pad) left = vw - pad - tp.width;
    if (top < pad) top = pad;
    if (top + tp.height > vh - pad) top = vh - pad - tp.height;

    tooltipStyle = `position:fixed;top:${top}px;left:${left}px;max-width:${maxWidth}px;`;

    // Update arrow position class
    tooltipEl.dataset.pos = pos;
  }
</script>

{#if inline}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <span
    bind:this={triggerEl}
    class="inline-flex items-center gap-0.5 border-b border-dashed border-gray-400 cursor-help"
    onmouseenter={handleEnter}
    onmouseleave={handleLeave}
  >
    {#if children}{@render children()}{/if}
    <span class="inline-flex items-center justify-center w-3.5 h-3.5 rounded-full bg-gray-200 flex-shrink-0">
      <span class="text-[8px] font-semibold text-gray-500 leading-none">?</span>
    </span>
  </span>
{:else}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <span
    bind:this={triggerEl}
    class="inline-flex items-center justify-center w-4 h-4 rounded-full bg-gray-200 cursor-help flex-shrink-0"
    onmouseenter={handleEnter}
    onmouseleave={handleLeave}
  >
    <span class="text-[10px] font-semibold text-gray-500 leading-none">?</span>
  </span>
{/if}

{#if show && text}
  <div
    bind:this={tooltipEl}
    class="tooltip-bubble"
    style={tooltipStyle}
  >
    {text}
  </div>
{/if}

<style>
  .tooltip-bubble {
    position: fixed;
    background: #1e293b;
    color: white;
    font-size: 12px;
    line-height: 1.5;
    padding: 8px 12px;
    border-radius: 6px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    z-index: 9999;
    pointer-events: none;
    white-space: normal;
    font-weight: 400;
    animation: tooltipIn 150ms ease-out;
    width: max-content;
  }

  @keyframes tooltipIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
