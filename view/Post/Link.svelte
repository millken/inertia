<script module>
    let cacheComponent = {};
</script>
<script>
    import {pjaxClick} from './pjax.action'
    let { tag = "a", children, ...rest } = $props();

</script>
<svelte:element use:pjaxClick this={tag} {...rest}>{@render children?.()}</svelte:element>