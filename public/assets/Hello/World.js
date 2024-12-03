import {
  Foot
} from "/assets/chunk-HHVLFR2V.js";
import {
  Head
} from "/assets/chunk-VVYUQ63Q.js";
import "/assets/chunk-WMQEQ65G.js";
import {
  append,
  first_child,
  sibling,
  template
} from "/assets/chunk-2IRIOZ5M.js";

// ../view/Hello/World.svelte
var root = template(`<!> <h1>Hello World</h1> <!>`, 1);
function World($$anchor) {
  var fragment = root();
  var node = first_child(fragment);
  Head(node, {});
  var node_1 = sibling(node, 4);
  Foot(node_1, {});
  append($$anchor, fragment);
}
export {
  World as default
};
