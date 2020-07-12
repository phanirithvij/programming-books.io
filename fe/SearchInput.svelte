<script>
  import { onMount, onDestroy } from "svelte";
  import { search } from "./search.js";
  import { makeDebouncer } from "./util.js";
  import SearchResults from "./SearchResults.svelte";
  import { isEsc } from "./keys.js";

  export let bookTitle = "";
  let searchTerm = "";
  let input;
  let results = [];
  let showResults = false;

  // Maybe: use debouncer from https://gist.github.com/nmsdvid/8807205
  const debouncer = makeDebouncer(250);

  $: searchTermChanged(searchTerm);
  $: showResults = searchTerm !== "";

  function keyDown(ev) {
    if (ev.key == "/") {
      input.focus();
      ev.preventDefault();
      return;
    }

    if (isEsc(ev)) {
      searchTerm = "";
      input.blur();
      results = [];
      return;
    }
  }

  function searchTermChanged(s) {
    const fn = doSearch.bind(this, s);
    debouncer(fn);
  }

  onMount(() => {
    document.addEventListener("keydown", keyDown);
  });

  onDestroy(() => {
    document.removeEventListener("keydown", keyDown);
  });

  function doSearch(s) {
    s = s.trim().toLowerCase();
    if (s.length == 0) {
      results = [];
      return;
    }
    // console.log(`doSearch: '${s}'`);
    const res = search(s);
    results = res;
    // console.log("results:", results);
  }

  function ondismiss() {
    // console.log("didDismiss");
    searchTerm = "";
    results = [];
  }
</script>

<style>
  input {
    width: 100%;
    font-size: 16px;
    padding: 2px 8px;
    background-color: white;
    /* filter: opacity(1); */
    /* border-color: #717274; */
    /* box-shadow: inset 0 1px 1px rgba(0,0,0,.075); */
    border: 1px solid silver;
    /* box-shadow: inset 0 0 0 0 transparent; */
    outline: 0;
    z-index: 25;
  }

  input:hover {
    border-color: #a0a0a0;
  }

  input::placeholder {
    color: #aaaaaa;
  }

  /* trick to make placeholder invisible when input field is focused */
  input:focus::placeholder {
    color: white;
  }

  /* no blue border when focused */
  input:focus {
    /* border: 1px solid lightskyblue; */
    /* border: 1px solid #aaaaaa; */
    border-color: #a0a0a0;
    box-shadow: inset 0 1px 1px rgba(0, 0, 0, 0.075);
  }
</style>

<input
  placeholder="Search '{bookTitle}' Tip: press '/'."
  bind:value={searchTerm}
  bind:this={input} />

{#if showResults}
  <SearchResults {ondismiss} {searchTerm} {results} selectedIdx={0} />
{/if}
