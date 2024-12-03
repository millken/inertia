<script>
    let text;
    let posts = [];
    </script>
    
    <svelte:head>
    <title>Golang + svelte + SSR</title>
    </svelte:head>
    <h1>{text}!</h1>
    
    <h2>Post Index</h2>
    
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
              <td><a href={`/${post.id || 0}`}>{post[key]}</a></td>
            {:else}
              <td>{post[key]}</td>
            {/if}
          {/each}
        </tr>
      {/each}
    </table>