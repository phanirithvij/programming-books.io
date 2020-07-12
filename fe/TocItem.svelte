<script>
  import Toc from "./Toc.svelte";
  import { item } from "./item.js";
  import {
    scrollPosSet,
    currentlySelectedIdx,
    tocItemIdxToScroll
  } from "./store.js";
  import { isTocItemExpanded, getLocationLastElementWithHash } from "./util.js";

  export let itemIdx = -1;
  export let level = 0;
  let element;

  const loc = getLocationLastElementWithHash();

  const tocItem = gTocItems[itemIdx];
  const title = item.title(tocItem);
  const url = item.url(tocItem);
  const hasChildren = item.hasChildren(tocItem);

  // let isExpanded = loc.startsWith(url);
  let isExpanded = isTocItemExpanded(itemIdx);

  if (isExpanded) {
    // console.log(`loc: ${loc}, url: ${url}`);
  }

  let isSelected = url === loc;

  currentlySelectedIdx.subscribe(idx => {
    isSelected = idx == itemIdx;
  });

  tocItemIdxToScroll.subscribe(idx => {
    if (idx === itemIdx) {
      // console.log("tocItemIdxToScroll: idx", idx);
      element.scrollIntoView(true);
      tocItemIdxToScroll.set(-2);
    }
  });

  function toggleExpand() {
    isExpanded = !isExpanded;
    // console.log("toogleExpand(): ", isExpanded);
  }

  function linkClicked() {
    var el = document.getElementById("toc");
    scrollPosSet(el.scrollTop);
    currentlySelectedIdx.set(itemIdx);
    // console.log("TocItem.linkClicked, scrollTop:", el.scrollTop);
  }

  // TODO: less level
  /*  
  var uri = tocItemURL(tocItem);
  if (uri.indexOf("#") != -1) {
    var parent = tocItemParent(tocItem);
    var isChapter = tocItemIsRoot(parent);
    var hasChildren = tocItemHasChildren(parent);
    var onlyArticleChildren = tocItemHasArticleChildren(parent);
    if (isChapter && hasChildren && onlyArticleChildren) {
      level += 1;
    }
  }
  */
</script>

<div bind:this={element} class="toc-item lvl{level}">
  {#if !hasChildren}
    <div class="toc-nav-empty-arrow" />
  {:else if isExpanded}
    <svg class="arrow" on:click={toggleExpand}>
      <use xlink:href="#arrow-expanded" />
    </svg>
  {:else}
    <svg class="arrow" on:click={toggleExpand}>
      <use xlink:href="#arrow-not-expanded" />
    </svg>
  {/if}

  {#if isSelected}
    <b>{title}</b>
  {:else}
    <a class="toc-link" {title} href={url} on:click={linkClicked}>{title}</a>
  {/if}
</div>

{#if hasChildren && isExpanded}
  <Toc parentIdx={itemIdx} level={level + 1} />
{/if}
