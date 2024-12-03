import { mount } from 'svelte'
import App from './view/About.svelte'

const app = mount(App, {
  target: document.getElementById('app'),
})

export default app
