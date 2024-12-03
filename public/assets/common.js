import {
  mount
} from "/assets/chunk-2IRIOZ5M.js";

// ../view/common.ts
var el = document.getElementById("app") || document.body;
var { View = void 0, ...props } = JSON.parse(el?.dataset.page || "{}");
if (!View) {
  console.error("No view found");
} else {
  import(`/assets/${View}.js`).then((module) => {
    const component = module.default;
    mount(component, {
      target: el,
      props
    });
  }, (error) => {
    console.error("Error loading view", error);
  });
}
