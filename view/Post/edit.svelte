<script>
    import Link from "./Link.svelte";

  let {post} = $props();
</script>

<h1>Edit Post</h1>

<form method="post" action={`/${post.id || 0}`}>
  <input type="hidden" name="_method" value="patch" />
  <!-- Add input fields here -->
  <input type="submit" value="Update Post" />
</form>

<br />

<Link href={`/`}>Back</Link>
<span> | </span>
<Link href={`/${post.id || 0}`}>Show Post</Link>
