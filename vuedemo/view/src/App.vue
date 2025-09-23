<script setup>
import { ref } from 'vue'

const counter = ref(0)

const incrementCounter = () => {
  counter.value++
}
</script>

<template>
  <main id="app">
    <h1 class="title">Hello, World!</h1>
    <button @click="incrementCounter">Count: {{ counter }}</button>
  </main>
</template>



<style>
#app {
  padding: 10px;
}

.title {
  color: #5C6AC4;
}
</style>