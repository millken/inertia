import {
  Link
} from "/assets/chunk-HZJBEF3K.js";
import "/assets/chunk-WMQEQ65G.js";
import {
  $document,
  append,
  append_styles,
  bind_value,
  child,
  comment,
  derived,
  each,
  first_child,
  get,
  head,
  if_block,
  index,
  next,
  pop,
  prop,
  push,
  remove_input_defaults,
  reset,
  set_text,
  sibling,
  template,
  template_effect,
  text
} from "/assets/chunk-2IRIOZ5M.js";

// ../view/Post/Index.svelte
var root_3 = template(`<th> </th>`);
var root_2 = template(`<thead></thead>`);
var root_6 = template(`<td><!></td>`);
var root_8 = template(`<td> </td>`);
var root_4 = template(`<tr></tr>`);
var root = template(`<h1 class="text-4xl font-bold tracking-tight text-gray-900 sm:text-6xl svelte-19o8zy9"> </h1> <h2>Post Index</h2> <input type="text" placeholder="Search"> <table border="1" cellpadding="10" class="svelte-19o8zy9"><!><!></table>`, 1);
var $$css = {
  hash: "svelte-19o8zy9",
  code: "table.svelte-19o8zy9 {border-collapse:collapse;}.text-4xl.svelte-19o8zy9 {font-size:2.25rem;line-height:2.5rem;}.font-bold.svelte-19o8zy9 {font-weight:700;}.tracking-tight.svelte-19o8zy9 {letter-spacing:-0.025em;}.text-gray-900.svelte-19o8zy9 {--text-opacity: 1;color:#1a202c;color:rgba(26, 32, 44, var(--text-opacity));}"
};
function Index($$anchor, $$props) {
  push($$props, true);
  append_styles($$anchor, $$css);
  let text2 = prop($$props, "text", 7, "JS \u7CFB\u7EDF");
  var fragment = root();
  head(($$anchor2) => {
    template_effect(() => $document.title = text2());
  });
  var h1 = first_child(fragment);
  var text_1 = child(h1);
  reset(h1);
  var input = sibling(h1, 4);
  remove_input_defaults(input);
  var table = sibling(input, 2);
  var node = child(table);
  if_block(node, () => $$props.posts.length > 0, ($$anchor2) => {
    var thead = root_2();
    each(thead, 21, () => Object.keys($$props.posts[0]), index, ($$anchor3, key) => {
      var th = root_3();
      var text_2 = child(th, true);
      reset(th);
      template_effect(() => set_text(text_2, get(key)));
      append($$anchor3, th);
    });
    reset(thead);
    append($$anchor2, thead);
  });
  var node_1 = sibling(node);
  each(node_1, 17, () => $$props.posts, index, ($$anchor2, post) => {
    var tr = root_4();
    each(tr, 21, () => Object.keys(get(post)), index, ($$anchor3, key) => {
      var fragment_1 = comment();
      var node_2 = first_child(fragment_1);
      if_block(
        node_2,
        () => get(key).toLowerCase() === "id",
        ($$anchor4) => {
          var td = root_6();
          var node_3 = child(td);
          var href = derived(() => `/${get(post).id || 0}`);
          Link(node_3, {
            get href() {
              return get(href);
            },
            children: ($$anchor5, $$slotProps) => {
              next();
              var text_3 = text();
              template_effect(() => set_text(text_3, get(post)[get(key)]));
              append($$anchor5, text_3);
            },
            $$slots: { default: true }
          });
          reset(td);
          append($$anchor4, td);
        },
        ($$anchor4) => {
          var td_1 = root_8();
          var text_4 = child(td_1, true);
          reset(td_1);
          template_effect(() => set_text(text_4, get(post)[get(key)]));
          append($$anchor4, td_1);
        }
      );
      append($$anchor3, fragment_1);
    });
    reset(tr);
    append($$anchor2, tr);
  });
  reset(table);
  template_effect(() => set_text(text_1, `${text2() ?? ""}!`));
  bind_value(input, text2);
  append($$anchor, fragment);
  pop();
}
export {
  Index as default
};
