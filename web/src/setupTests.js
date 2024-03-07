// jsdom does not implement the <dialog> element's modal API; this minimal
// polyfill mirrors the open/close behavior the components rely on.
if (!window.HTMLDialogElement.prototype.showModal) {
  window.HTMLDialogElement.prototype.showModal = function () {
    this.open = true
  }
  window.HTMLDialogElement.prototype.close = function () {
    this.open = false
    this.dispatchEvent(new Event('close'))
  }
}

// localStorage is not functional in the jsdom test environment, so tests use
// a deterministic in-memory implementation.
const store = new Map()

Object.defineProperty(window, 'localStorage', {
  configurable: true,
  value: {
    getItem: (key) => (store.has(key) ? store.get(key) : null),
    setItem: (key, value) => store.set(key, String(value)),
    removeItem: (key) => store.delete(key),
    clear: () => store.clear(),
  },
})
