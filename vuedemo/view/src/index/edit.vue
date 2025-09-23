<template>
  <div>
    <h1>Edit Post Page</h1>

    <form method="post" :action="`/post/${props.post.id || 0}/edit`">
      <input type="hidden" name="_method" value="patch" />

      <!-- 示例输入字段，使用 props 作为初始值 -->
      <div>
        <label>Title</label>
        <input type="text" name="title" :value="props.post.title || ''" />
      </div>

      <div>
        <label>Body</label>
        <textarea name="body" :value="props.post.body || ''"></textarea>
      </div>

      <input type="submit" value="Update Post" />
    </form>

    <br />

    <component :is="Link" href="/">Back</component>
    <span> | </span>
    <component :is="Link" :href="`/post/${props.post.id || 0}`">Show Post</component>
  </div>
</template>

<script setup>
import Link from '../Link.vue';
const props = defineProps({
  post: Object
});
</script>
