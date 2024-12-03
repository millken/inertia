

<script>
    import Head from "./layouts/Head.svelte";
    import Foot from "./layouts/Foot.svelte";
</script>
<Head />
<h1>New Post</h1>

<form method="post" action={`/`}>
  <!-- Add input fields here -->
  <input type="submit" value="Create Post" />
</form>

<br />

<a href={`/`}>Back</a>
<Foot />