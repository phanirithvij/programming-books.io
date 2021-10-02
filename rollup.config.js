import svelte from "rollup-plugin-svelte";
import resolve from "@rollup/plugin-node-resolve";
import commonjs from "@rollup/plugin-commonjs";
import { terser } from "rollup-plugin-terser";
import css from 'rollup-plugin-css-only'

let production = !process.env.ROLLUP_WATCH;
const isDev = (process.env.ISDEV || "").trim();
//console.log(`env.ISDEV: '${isDev}'`);
if (isDev == "true") {
  console.log("is dev build");
  production = false;
}

// https://www.npmjs.com/package/rollup-plugin-svelte
let sveltePlugin = svelte({
  emitCss: true,
  compilerOptions: {
  }
});

// If you have external dependencies installed from
// npm, you'll most likely need these plugins. In
// some cases you'll need additional configuration —
// consult the documentation for details:
// https://github.com/rollup/rollup-plugin-commonjs
let resolvePlugin = resolve({
  browser: true,
  dedupe: importee =>
    importee === "svelte" || importee.startsWith("svelte/")
});

// https://www.npmjs.com/package/rollup-plugin-css-only
// note: file name is relative to outpt.file directory
let cssPlugin = css({ output: 'bundle.css' });

export default {
  input: "fe/main.js",
  output: {
    file: "www_generated/svelte/bundle.js",
    format: "iife",
    name: "app",
  },
  plugins: [
    sveltePlugin,
    cssPlugin,
    resolvePlugin,
    commonjs(),
    // If we're building for production (npm run build
    // instead of npm run dev), minify
    production && terser()
  ],
  watch: {
    clearScreen: false
  }
};
