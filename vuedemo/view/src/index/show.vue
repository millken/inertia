<script setup>
import Link from '../Link.vue';
const props = defineProps({
  post: Object
});
</script>

<template>
  <div>
    <h1>Show Post</h1>

    <table border="1" cellpadding="10">
      <thead>
        <tr>
          <th v-for="key in Object.keys(props.post)" :key="key">{{ key }}</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td v-for="key in Object.keys(props.post)" :key="key">{{ props.post[key] }}</td>
        </tr>
      </tbody>
    </table>

    <component :is="Link" href="/">Back</component>
    <span> | </span>
    <component :is="Link" :href="`/post/${props.post.id || 0}/edit`">Edit</component>
  </div>
</template>

<style>
table {
  border-collapse: collapse;
}
</style>
