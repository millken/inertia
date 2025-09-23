<template>
  <component :is="tag" ref="linkEl" v-bind="rest">
    <slot></slot>
  </component>
</template>

<script>
import { pjaxClick } from './pjax.action.js'

export default {
  name: 'Link',
  inheritAttrs: false,
  props: {
    tag: {
      type: String,
      default: 'a'
    }
  },
  computed: {
    rest() {
      return { ...this.$attrs, ...this.$props, tag: undefined }
    }
  },
  mounted() {
    if (this.$refs.linkEl) {
      this.cleanup = pjaxClick(this.$refs.linkEl);
    }
  },
  beforeUnmount() {
    if (this.cleanup && this.cleanup.destroy) {
      this.cleanup.destroy();
    }
  }
}
</script>