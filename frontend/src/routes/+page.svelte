<script lang="ts">
	import { client } from "$lib/graphqlClient";

    const response = await client.query(`
        query {
            beans {
                id
                title
            }
        }
    `, {})

    const beans = response.data;
</script>

<h1>Welcome to SvelteKit</h1>
<p>Visit <a href="https://svelte.dev/docs/kit">svelte.dev/docs/kit</a> to read the documentation</p>

{JSON.stringify(beans, null, 2)}
