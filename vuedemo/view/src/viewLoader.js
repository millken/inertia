import { createApp } from 'vue'

export const modules = import.meta.glob('./**/*.vue')

const cache = new Map()
let currentApp = null // 共享的 Vue 实例

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

export async function mountView(viewName, props = {}, targetElement = null) {
  try {
    if (!hasView(viewName)) {
      throw new Error(`View ${viewName} not found`)
    }
    
    const component = await loadView(viewName)
    const target = targetElement || document.getElementById('app') || document.body
    
    // 卸载之前的实例
    if (currentApp) {
      currentApp.unmount()
    }
    
    // 清空内容
    target.innerHTML = ''
    
    // 创建新实例并挂载
    currentApp = createApp(component, props)
    currentApp.mount(target)
    
    return currentApp
  } catch (error) {
    console.error(`Error mounting view ${viewName}:`, error)
    throw error
  }
}

export function getCurrentApp() {
  return currentApp
}

export function unmountCurrentApp() {
  if (currentApp) {
    currentApp.unmount()
    currentApp = null
  }
}