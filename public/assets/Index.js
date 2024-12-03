import "/assets/chunk-WMQEQ65G.js";
import {
  $document,
  append,
  child,
  comment,
  each,
  first_child,
  get,
  head,
  if_block,
  index,
  reset,
  set_attribute,
  set_text,
  sibling,
  template,
  template_effect
} from "/assets/chunk-2IRIOZ5M.js";

// ../view/Index.svelte
var root_3 = template(`<th> </th>`);
var root_2 = template(`<thead></thead>`);
var root_6 = template(`<td><a> </a></td>`);
var root_7 = template(`<td> </td>`);
var root_4 = template(`<tr></tr>`);
var root = template(`<h1></h1> <h2>Post Index</h2> <table border="1" cellpadding="10"><!><!></table>`, 1);
function Index($$anchor) {
  let text;
  let posts = [];
  var fragment = root();
  head(($$anchor2) => {
    $document.title = `Golang + svelte + SSR`;
  });
  var h1 = first_child(fragment);
  h1.textContent = `${text ?? ""}!`;
  var table = sibling(h1, 4);
  var node = child(table);
  if_block(node, () => posts.length > 0, ($$anchor2) => {
    var thead = root_2();
    each(thead, 21, () => Object.keys(posts[0]), index, ($$anchor3, key) => {
      var th = root_3();
      var text_1 = child(th, true);
      reset(th);
      template_effect(() => set_text(text_1, get(key)));
      append($$anchor3, th);
    });
    reset(thead);
    append($$anchor2, thead);
  });
  var node_1 = sibling(node);
  each(node_1, 17, () => posts, index, ($$anchor2, post) => {
    var tr = root_4();
    each(tr, 21, () => Object.keys(get(post)), index, ($$anchor3, key) => {
      var fragment_1 = comment();
      var node_2 = first_child(fragment_1);
      if_block(
        node_2,
        () => get(key).toLowerCase() === "id",
        ($$anchor4) => {
          var td = root_6();
          var a = child(td);
          var text_2 = child(a, true);
          reset(a);
          reset(td);
          template_effect(() => {
            set_attribute(a, "href", `/${get(post).id || 0}`);
            set_text(text_2, get(post)[get(key)]);
          });
          append($$anchor4, td);
        },
        ($$anchor4) => {
          var td_1 = root_7();
          var text_3 = child(td_1, true);
          reset(td_1);
          template_effect(() => set_text(text_3, get(post)[get(key)]));
          append($$anchor4, td_1);
        }
      );
      append($$anchor3, fragment_1);
    });
    reset(tr);
    append($$anchor2, tr);
  });
  reset(table);
  append($$anchor, fragment);
}
export {
  Index as default
};
