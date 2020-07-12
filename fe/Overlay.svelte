<script>
  export let ondismiss = null; // function
  export let dismissWithEsc = false;

  if (!ondismiss) {
    throw new Error("ondimiss property is requred");
  }

  let overlay;

  function handleKeyDown(ev) {
    if (dismissWithEsc && ev.which === 27) {
      ondismiss();
    }
  }

  function handleClick(ev) {
    if (ev.target !== overlay) {
      // clicked outside of the overlay
      return;
    }
    ondismiss();
  }
</script>

<style>
  .overlay {
    position: fixed;
    z-index: 9;
    top: 0;
    bottom: 0;
    left: 0;
    right: 0;

    background-color: rgba(0, 0, 0, 0.4);
  }
</style>

<div
  bind:this={overlay}
  class="overlay"
  on:click={handleClick}
  on:keydown={handleKeyDown}>
  <slot />
</div>
