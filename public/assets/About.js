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
  set_attribute,
  sibling,
  template
} from "/assets/chunk-2IRIOZ5M.js";

// ../view/About.svelte
var root = template(`<!> <h1>New Post</h1> <form method="post"><input type="submit" value="Create Post"></form> <br> <a>Back</a> <!>`, 1);
function About($$anchor) {
  var fragment = root();
  var node = first_child(fragment);
  Head(node, {});
  var form = sibling(node, 4);
  set_attribute(form, "action", `/`);
  var a = sibling(form, 4);
  set_attribute(a, "href", `/`);
  var node_1 = sibling(a, 2);
  Foot(node_1, {});
  append($$anchor, fragment);
}
export {
  About as default
};
