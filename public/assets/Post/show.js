import {
  Link
} from "/assets/chunk-HZJBEF3K.js";
import "/assets/chunk-WMQEQ65G.js";
import {
  append,
  append_styles,
  child,
  derived,
  each,
  first_child,
  get,
  index,
  next,
  pop,
  push,
  reset,
  set_text,
  sibling,
  template,
  template_effect,
  text
} from "/assets/chunk-2IRIOZ5M.js";

// ../view/Post/show.svelte
var root_1 = template(`<th> </th>`);
var root_2 = template(`<td> </td>`);
var root = template(`<h1>Show Post</h1> <table border="1" cellpadding="10" class="svelte-dfg4j1"><thead></thead><tbody><tr></tr></tbody></table> <br> <!> <span>|</span> <!>`, 1);
var $$css = {
  hash: "svelte-dfg4j1",
  code: "table.svelte-dfg4j1 {border-collapse:collapse;}"
};
function Show($$anchor, $$props) {
  push($$props, true);
  append_styles($$anchor, $$css);
  var fragment = root();
  var table = sibling(first_child(fragment), 2);
  var thead = child(table);
  each(thead, 21, () => Object.keys($$props.post), index, ($$anchor2, key) => {
    var th = root_1();
    var text2 = child(th, true);
    reset(th);
    template_effect(() => set_text(text2, get(key)));
    append($$anchor2, th);
  });
  reset(thead);
  var tbody = sibling(thead);
  var tr = child(tbody);
  each(tr, 21, () => Object.keys($$props.post), index, ($$anchor2, key) => {
    var td = root_2();
    var text_1 = child(td, true);
    reset(td);
    template_effect(() => set_text(text_1, $$props.post[get(key)]));
    append($$anchor2, td);
  });
  reset(tr);
  reset(tbody);
  reset(table);
  var node = sibling(table, 4);
  Link(node, {
    href: `/`,
    children: ($$anchor2, $$slotProps) => {
      next();
      var text_2 = text("Back");
      append($$anchor2, text_2);
    },
    $$slots: { default: true }
  });
  var node_1 = sibling(node, 4);
  var href = derived(() => `/${$$props.post.id || 0}/edit`);
  Link(node_1, {
    get href() {
      return get(href);
    },
    children: ($$anchor2, $$slotProps) => {
      next();
      var text_3 = text("Edit");
      append($$anchor2, text_3);
    },
    $$slots: { default: true }
  });
  append($$anchor, fragment);
  pop();
}
export {
  Show as default
};
