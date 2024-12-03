<script>

    import Link from "./Link.svelte";

</script>
<h1>New Post</h1>

<form method="post" action={`/`}>
  <!-- Add input fields here -->
  <input type="submit" value="Create Post" />
</form>

<br />

<Link href={`/`}>Back</Link>
