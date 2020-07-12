<script>
  import { onMount } from "svelte";
  import TocItem from "./TocItem.svelte";
  import { item } from "./item.js";
  import {
    getLocationLastElementWithHash,
    setTocExpandedForCurrentURL
  } from "./util.js";
  import { scrollPosGet, scrollPosClear, tocItemIdxToScroll } from "./store.js";

  export let parentIdx = -1;
  export let level = 0;

  let currentlySelectedIdx = -1;

  const children = item.childrenForParentIdx(parentIdx);

  onMount(() => {
    // when we mount top-level Toc, scroll toc item into view
    if (parentIdx !== -1) {
      return;
    }

    // if we explicitly remembered toc scroll position, restore it
    const scrollTop = scrollPosGet() || -1;
    if (scrollTop >= 0) {
      // console.log("scrollTop:", scrollTop);
      const el = document.getElementById("toc");
      el.scrollTop = scrollTop;
      scrollPosClear();
      tocItemIdxToScroll.set(-2);
      return;
    }

    // otherwise tell currently selected TocItem to scroll
    // itself into view.
    tocItemIdxToScroll.set(currentlySelectedIdx);
  });

  if (parentIdx === -1) {
    // initial setup
    const loc = getLocationLastElementWithHash();
    // console.log(`loc: ${loc}`);
    currentlySelectedIdx = setTocExpandedForCurrentURL();
    window.onhashchange = setTocExpandedForCurrentURL;
  }
</script>

{#each children as itemIdx}
  <TocItem {itemIdx} {level} />
{/each}
