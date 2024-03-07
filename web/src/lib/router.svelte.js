// Minimal hash-based router: routes look like #/domains and survive being
// served from any base path.

function currentPath() {
  return location.hash.slice(1) || '/'
}

export const router = $state({ path: currentPath() })

window.addEventListener('hashchange', () => {
  router.path = currentPath()
})

export function navigate(path) {
  location.hash = path
}
