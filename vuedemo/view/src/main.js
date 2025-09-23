import { createApp } from 'vue'
import { modules, hasView, loadView } from './viewLoader.js'

var el = document.getElementById("app") || document.body;
var { _ViEW_ = void 0, ...props } = JSON.parse(el?.dataset.page || "{}");
if (!_ViEW_) {
  _ViEW_ = "App"; // Default view
}

if (hasView(_ViEW_)) {
  loadView(_ViEW_).then((component) => {
    createApp(component, props).mount(el);
  }).catch((error) => {
    console.error(`Error loading view ${_ViEW_}`, error);
  });
} else {
  console.error(`View ${_ViEW_} not found`);
}

