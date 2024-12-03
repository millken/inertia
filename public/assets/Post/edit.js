import {
  Link
} from "/assets/chunk-HZJBEF3K.js";
import "/assets/chunk-WMQEQ65G.js";
import {
  append,
  derived,
  first_child,
  get,
  next,
  pop,
  push,
  set_attribute,
  sibling,
  template,
  template_effect,
  text
} from "/assets/chunk-2IRIOZ5M.js";

// ../view/Post/edit.svelte
var root = template(`<h1>Edit Post</h1> <form method="post"><input type="hidden" name="_method" value="patch"> <input type="submit" value="Update Post"></form> <br> <!> <span>|</span> <!>`, 1);
function Edit($$anchor, $$props) {
  push($$props, true);
  var fragment = root();
  var form = sibling(first_child(fragment), 2);
  var node = sibling(form, 4);
  Link(node, {
    href: `/`,
    children: ($$anchor2, $$slotProps) => {
      next();
      var text2 = text("Back");
      append($$anchor2, text2);
    },
    $$slots: { default: true }
  });
  var node_1 = sibling(node, 4);
  var href = derived(() => `/${$$props.post.id || 0}`);
  Link(node_1, {
    get href() {
      return get(href);
    },
    children: ($$anchor2, $$slotProps) => {
      next();
      var text_1 = text("Show Post");
      append($$anchor2, text_1);
    },
    $$slots: { default: true }
  });
  template_effect(() => set_attribute(form, "action", `/${$$props.post.id || 0}`));
  append($$anchor, fragment);
  pop();
}
export {
  Edit as default
};
