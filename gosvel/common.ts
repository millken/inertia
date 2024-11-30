import { mount } from "svelte";
export function mountApp(component: any) {
    let el = document.getElementById("app");
    if (!el) {
        el = document.createElement("div");
        el.id = "app";
        document.body.appendChild(el);
    }
    mount(component, {
        target: el,
    });
}