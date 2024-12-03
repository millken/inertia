import { mount } from "svelte";
let el = document.getElementById("app") || document.body;

const { View = undefined, ...props } = JSON.parse(el?.dataset.page || '{}');

if (!View) {
    console.error("No view found");
} else {
    import(`/assets/${View}.js`).then((module) => {
        const component = module.default;
        mount(component, {
            target: el,
            props: props,
        });
    }, (error) => {
        console.error("Error loading view", error);
    });
}
