//export NODE_ENV=production
import esbuild from "esbuild";
import esbuildSvelte from "esbuild-svelte";
import { sveltePreprocess } from "svelte-preprocess";
esbuild
  .build({
    entryPoints: ["index.js"],
    bundle: true,
    outdir: `./dist`,
    mainFields: ["svelte", "browser", "module"],
    conditions: ["svelte", "browser","production"],
    // logLevel: `info`,
    minify: true, //so the resulting code is easier to understand
    // sourcemap: "inline",
    splitting: true,
    write: true,
    format: `esm`,
    plugins: [
        esbuildSvelte({
            preprocess: sveltePreprocess(),
        }),
    ],
})
.catch((error, location) => {
    console.warn(`Errors: `, error, location);
    process.exit(1);
});