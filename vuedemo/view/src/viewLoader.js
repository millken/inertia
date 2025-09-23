// Shared view loader with caching
export const modules = import.meta.glob('./**/*.vue')

const cache = new Map()

export function hasView(name) {
  return !!modules[`./${name}.vue`]
}

export async function loadView(name) {
  const path = `./${name}.vue`
  if (!modules[path]) {
    throw new Error(`View ${name} not found`)
  }
  if (cache.has(path)) {
    return cache.get(path)
  }
  const mod = await modules[path]()
  const component = mod.default || mod
  cache.set(path, component)
  return component
}

export function clearCache(name) {
  if (name) {
    const path = `./${name}.vue`
    cache.delete(path)
    return
  }
  cache.clear()
}

export async function preloadView(name) {
  return await loadView(name)
}
