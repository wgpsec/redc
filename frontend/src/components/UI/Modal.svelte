<script>
  let {
    show = false,
    onclose = () => {},
    zIndex = 50,
    class: className = '',
    children,
  } = $props();

  let visible = $state(false);
  let animating = $state(false);

  $effect(() => {
    if (show && !visible) {
      visible = true;
      requestAnimationFrame(() => { animating = true; });
    } else if (!show && visible) {
      animating = false;
      setTimeout(() => { visible = false; }, 150);
    }
  });

  function handleBackdrop(e) {
    if (e.target === e.currentTarget) onclose();
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') onclose();
  }
</script>

{#if visible}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 flex items-center justify-center overflow-visible modal-backdrop {animating ? 'modal-enter' : 'modal-exit'} {className}"
    style="z-index: {zIndex};"
    role="dialog"
    aria-modal="true"
    onclick={handleBackdrop}
    onkeydown={handleKeydown}
  >
    <div class="modal-content {animating ? 'modal-content-enter' : 'modal-content-exit'}">
      {@render children()}
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    background: rgba(0, 0, 0, 0);
    transition: background 0.15s ease-out;
  }
  .modal-backdrop.modal-enter {
    background: rgba(0, 0, 0, 0.5);
  }
  .modal-backdrop.modal-exit {
    background: rgba(0, 0, 0, 0);
  }
  .modal-content {
    transition: opacity 0.15s ease-out, transform 0.15s ease-out;
  }
  .modal-content-enter {
    opacity: 1;
    transform: scale(1);
  }
  .modal-content-exit {
    opacity: 0;
    transform: scale(0.97);
  }
</style>
