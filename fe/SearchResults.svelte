<script>
  import Overlay from "./Overlay.svelte";
  import { afterUpdate, beforeUpdate, onMount, onDestroy } from "svelte";
  import { isEnter, isNavUp, isNavDown } from "./keys.js";
  import { item } from "./item.js";
  import { scrollintoview } from "./actions/scrollintoview.js";

  export let ondismiss = null; // function
  if (!ondismiss) {
    throw new Error("ondimiss property is requred");
  }
  /* results is array of items:
    {
      tocItem: [],
      term: "",
      match: [[idx, len], ...],
      id: int,
    }
  */
  export let results = [];
  export let selectedIdx = 0;
  export let searchTerm = "";

  let ignoreNextMouseEnter = false;
  let prevResulutsCount = 0;

  // TODO: I don't quite understand when beforeUpdate / afterUpdate
  // are called. Maybe just clamp selectedIdx to be within
  // results in afterUpdate
  beforeUpdate(() => {
    //console.log("before:", prevResulutsCount);
    if (results.length !== prevResulutsCount) {
      selectedIdx = 0;
      prevResulutsCount = results.length;
    }
  });

  afterUpdate(() => {
    // reset which item is selected when the number
    // of search results changes
    // console.log("after:", results.length);
    if (results.length !== prevResulutsCount) {
      selectedIdx = 0;
      prevResulutsCount = results.length;
    }
  });

  // must add them globally to be called even when search
  // input field has focus
  onMount(() => {
    // console.log("SearchResults term:", searchTerm, "results:", results.length);
    document.addEventListener("keydown", keyDown);
  });

  onDestroy(() => {
    document.removeEventListener("keydown", keyDown);
  });

  function getParentTitle(tocItem) {
    var res = "";
    var parent = item.parent(tocItem);
    while (parent) {
      var s = item.title(parent);
      if (res) {
        s = s + " / ";
      }
      res = s + res;
      parent = item.parent(parent);
    }
    return res;
  }

  // return true if term is a search synonym inside tocItem
  function isMatchSynonym(tocItem, term) {
    term = term.toLowerCase();
    var title = item.title(tocItem).toLowerCase();
    return title != term;
  }

  // if search matched synonym returns "${chapterTitle} / ${articleTitle}"
  // otherwise empty string
  function getArticlePath(tocItem, term) {
    if (!isMatchSynonym(tocItem, term)) {
      return null;
    }
    var title = item.title(tocItem);
    var parentTitle = getParentTitle(tocItem);
    if (parentTitle == "") {
      return title;
    }
    return parentTitle + " / " + title;
  }

  function getWhere(idx) {
    const r = results[idx];
    var tocItem = r.tocItem;
    var term = r.term;

    // TODO: get multi-level path (e.g. for 'json' where in Refelection / Uses for reflection chapter)
    const inTxt = getArticlePath(tocItem, term);
    if (inTxt) {
      return inTxt;
    }
    return getParentTitle(tocItem);
  }

  // create HTML to highlight part of s starting at idx and with length len
  function hilightSearchResult(txt, matches) {
    var prevIdx = 0;
    var n = matches.length;
    var res = "";
    var s = "";
    // alternate non-higlighted and highlihted strings
    for (var i = 0; i < n; i++) {
      var el = matches[i];
      var idx = el[0];
      var len = el[1];

      var nonHilightLen = idx - prevIdx;
      if (nonHilightLen > 0) {
        s = txt.substring(prevIdx, prevIdx + nonHilightLen);
        res += `<span>${s}</span>`;
      }
      s = txt.substring(idx, idx + len);
      res += `<span class="hili">${s}</span>`;
      prevIdx = idx + len;
    }
    var txtLen = txt.length;
    nonHilightLen = txtLen - prevIdx;
    if (nonHilightLen > 0) {
      s = txt.substring(prevIdx, prevIdx + nonHilightLen);
      res += `<span>${s}</span>`;
    }
    return res;
  }

  function hiliHTML(idx) {
    const r = results[idx];
    // console.log("hili: idx:", idx, "r:", r);
    return hilightSearchResult(r.term, r.match);
  }

  function isChapterOrArticleURL(s) {
    var isChapterOrArticle = s.indexOf("#") === -1;
    return isChapterOrArticle;
  }

  function navigateToSearchResult(idx) {
    // console.log("navigateToSearchResult:", idx);
    var loc = window.location.pathname;
    var parts = loc.split("/");
    var lastIdx = parts.length - 1;
    var lastURL = parts[lastIdx];
    var selected = results[idx];
    var tocItem = selected.tocItem;

    // either replace chapter/article url or append to book url
    var uri = item.url(tocItem);
    if (isChapterOrArticleURL(lastURL)) {
      parts[lastIdx] = uri;
    } else {
      parts.push(uri);
    }
    loc = parts.join("/");
    window.location = loc;
    ondismiss();
  }

  function dir(ev) {
    if (isNavUp(ev)) {
      return -1;
    }
    if (isNavDown(ev)) {
      return 1;
    }
    return 0;
  }

  function keyDown(ev) {
    // console.log("SearchResults.keyDown:", ev);
    if (isEnter(ev)) {
      navigateToSearchResult(selectedIdx);
      ev.stopPropagation();
      return;
    }
    const n = dir(ev);
    if (n === 0) {
      return;
    }

    ev.stopPropagation();
    ev.preventDefault();

    selectedIdx += n;
    if (selectedIdx < 0) {
      selectedIdx = 0;
    }
    const lastIdx = results.length - 1;
    if (selectedIdx > lastIdx) {
      selectedIdx = lastIdx;
    }
    // console.log("newSelected", selectedIdx);
    // changing selected element triggers mouseenter
    // on the element so we have to supress it
    ignoreNextMouseEnter = true;
  }

  function clicked(idx) {
    // console.log("clicked:", idx);
    navigateToSearchResult(idx);
  }

  function mouseEnter(idx) {
    if (ignoreNextMouseEnter) {
      ignoreNextMouseEnter = false;
      return;
    }
    selectedIdx = idx;
  }
</script>

<style>
  .wrapper {
    position: fixed;
    top: 28px;
    width: 74vw;
    left: 13vw; /* (100 - 74) / 2 */
    right: 13vw;

    border: 1px solid #aaaaaa;
    z-index: 25;

    /* min-height: 320px; */
    background-color: white;
  }

  .results {
    max-height: 70vh;
    padding: 4px 8px;
    line-height: 1.3em;
    cursor: pointer;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .help {
    color: #717274;
    background-color: #f9f9f9;
    padding: 8px 8px;
    font-size: 0.7em;
    padding-left: 10px;
  }

  .selected {
    background-color: #eeeeee;
  }

  .no-results {
    padding-top: 48px;
    padding-bottom: 48px;
    margin-left: auto;
    margin-right: auto;
    /* background-color: #dddddd; */
    text-align: center;
  }

  .in {
    color: gray;
    font-size: 0.8em;
    float: right;
  }

  @media screen and (max-width: 500px) {
    .wrapper {
      width: 90vw;
      left: 5vw; /* (100 - 90) / 2 */
      right: 5vw;
    }

    /* leave space for a on-screen keyboard. Tested on
     Android Pixel device */
    .results {
      max-height: 70vh;
    }
  }

  /* higlight search results with yellow-ish background */
  :global(.hili) {
    /* font-weight: bold; */
    /*padding: 1px 2px; */
    /* background: #ffeb3b; */
    background: rgba(255, 235, 59, 0.6);
    /* border-radius: 2px; */
    /* font-weight: bold; */
    /* background-color: lightskyblue; */
  }
</style>

<Overlay {ondismiss} dismissWithEsc={true}>
  <div class="wrapper">
    <div class="results">
      {#if results.length === 0}
        <div class="no-results">No search results for '{searchTerm}'</div>
      {:else}
        {#each results as r, idx (r.id)}
          {#if idx === selectedIdx}
            <div
              use:scrollintoview
              on:click={() => clicked(idx)}
              class="selected">
              {@html hiliHTML(idx)}
              <span class="in">{getWhere(idx)}</span>
            </div>
          {:else}
            <div
              on:click={() => clicked(idx)}
              on:mouseenter={() => mouseEnter(idx)}>
              {@html hiliHTML(idx)}
              <span class="in">{getWhere(idx)}</span>
            </div>
          {/if}
        {/each}
      {/if}
    </div>
    <div class="help">
      &uarr; &darr; to navigate &nbsp;&nbsp;&nbsp; &crarr; to select
      &nbsp;&nbsp;&nbsp; Esc to close
    </div>
  </div>
</Overlay>
