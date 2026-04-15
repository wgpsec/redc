<script>
  let {
    show = false,
    onclose = () => {},
    zIndex = 50,
    class: className = '',
    children,
  } = $props();

  function handleBackdrop(e) {
    if (e.target === e.currentTarget) onclose();
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') onclose();
  }
</script>

{#if show}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center overflow-visible {className}"
    style="z-index: {zIndex};"
    role="dialog"
    aria-modal="true"
    onclick={handleBackdrop}
    onkeydown={handleKeydown}
  >
    {@render children()}
  </div>
{/if}
