<script>
  let { title, onclose, children } = $props()

  let dialog

  $effect(() => {
    dialog?.showModal()
  })

  // A click on the backdrop targets the dialog element itself
  function backdropClick(event) {
    if (event.target === dialog) {
      onclose?.()
    }
  }
</script>

<dialog bind:this={dialog} onclose={() => onclose?.()} onclick={backdropClick}>
  <h2>{title}</h2>
  {@render children()}
</dialog>
