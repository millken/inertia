<script>
  import Link from './Link.svelte';
  let {post} = $props();
</script>

<h1>Show Post</h1>

<table border="1" cellpadding="10">
  <thead>
    {#each Object.keys(post) as key}
      <th>{key}</th>
    {/each}
  </thead>
  <tbody>
    <tr>
      {#each Object.keys(post) as key}
        <td>{post[key]}</td>
      {/each}
    </tr>
  </tbody>
</table>

<br />

<Link href={`/`}>Back</Link>
<span> | </span>
<Link href={`/${post.id || 0}/edit`}>Edit</Link>

<style>
  table {
    border-collapse: collapse;
  }
</style>
