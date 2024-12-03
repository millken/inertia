<script>
  import Link from './Link.svelte';
let {text = "JS 系统",posts} = $props();
</script>

<svelte:head>
<title>{text}</title>
</svelte:head>
<h1 class="text-4xl font-bold tracking-tight text-gray-900 sm:text-6xl">{text}!</h1>

<h2>Post Index</h2>
<input type="text" placeholder="Search" bind:value={text} />
<table border="1" cellpadding="10">
  {#if posts.length > 0}
    <thead>
      {#each Object.keys(posts[0]) as key}
        <th>{key}</th>
      {/each}
    </thead>
  {/if}
  {#each posts as post}
    <tr>
      {#each Object.keys(post) as key}
        {#if key.toLowerCase() === "id"}
          <td><Link href={`/${post.id || 0}`}>{post[key]}</Link></td>
        {:else}
          <td>{post[key]}</td>
        {/if}
      {/each}
    </tr>
  {/each}
</table>
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