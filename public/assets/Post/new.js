import {
  Link
} from "/assets/chunk-HZJBEF3K.js";
import "/assets/chunk-WMQEQ65G.js";
import {
  append,
  first_child,
  next,
  set_attribute,
  sibling,
  template,
  text
} from "/assets/chunk-2IRIOZ5M.js";

// ../view/Post/new.svelte
var root = template(`<h1>New Post</h1> <form method="post"><input type="submit" value="Create Post"></form> <br> <!>`, 1);
function New($$anchor) {
  var fragment = root();
  var form = sibling(first_child(fragment), 2);
  set_attribute(form, "action", `/`);
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
  append($$anchor, fragment);
}
export {
  New as default
};
