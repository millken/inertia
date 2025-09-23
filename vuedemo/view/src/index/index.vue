<template>
  <div>
    <h1 class="text-4xl font-bold tracking-tight text-gray-900 sm:text-6xl">{{ text }}!</h1>

    <h2>Post Index</h2>
    <input type="text" placeholder="Search" :value="text" @input="updateText($event.target.value)" />
    <table border="1" cellpadding="10">
      <thead v-if="posts.length > 0">
        <tr>
          <th v-for="key in Object.keys(posts[0])" :key="key">{{ key }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="post in posts" :key="post.id">
          <td v-for="key in Object.keys(post)" :key="key">
            <Link v-if="key.toLowerCase() === 'id'" :href="`/post/${post.id || 0}`">{{ post[key] }}</Link>
            <span v-else>{{ post[key] }}</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import Link from '../Link.vue';

const props = defineProps({
  text: {
    type: String,
    default: "JS 系统"
  },
  posts: Array
});

const emit = defineEmits(['update:text']);

const updateText = (value) => {
  emit('update:text', value);
};
</script>

<style>
  table {
    border-collapse: collapse;
  }
  .text-4xl {
    font-size: 2.25rem;
    line-height: 2.5rem;
  }
  .font-bold {
    font-weight: 700;
  }
  .tracking-tight {
    letter-spacing: -0.025em;
  }
  .text-gray-900 {
    --text-opacity: 1;
    color: #1a202c;
    color: rgba(26, 32, 44, var(--text-opacity));
  }
</style>