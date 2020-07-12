<script>
  import { findTocIdxForCurrentURL, getLocationLastElement } from "./util.js";
  import { item } from "./item.js";

  let tocItems = [];

  const pageURL = getLocationLastElement();

  function makeItem(it, indent) {
    const url = item.url(it);
    const isSelected = url == pageURL;
    return {
      title: item.title(it),
      url: url,
      indent: indent,
      isSelected: isSelected
    };
  }

  function getChildrenForIdx(idx) {
    const it = gTocItems[idx];
    const allChildren = item.childrenForParentIdx(idx);
    const res = [makeItem(it, 0)];
    res[0].title = res[0].title + "/";
    for (let idx of allChildren) {
      const c = gTocItems[idx];
      const uri = item.url(c);
      if (!uri.includes("#")) {
        res.push(makeItem(c, 1));
      }
    }
    return res;
  }

  function calcTocItems() {
    let idx = findTocIdxForCurrentURL();
    const it = gTocItems[idx];
    tocItems = getChildrenForIdx(idx);
    if (tocItems.length === 1) {
      idx = item.parentIdx(gTocItems[idx]);
      if (idx != -1) {
        tocItems = getChildrenForIdx(idx);
      }
    }
    if (tocItems.length === 1) {
      tocItems = [];
    }
  }

  calcTocItems();
</script>

<style>
  .chapter-toc {
    font-size: 0.9em;
  }
</style>

<div class="chapter-toc">
  {#each tocItems as item}
    <div class="mtoc-{item.indent}">
      {#if item.isSelected}
        <b>{item.title}</b>
      {:else}
        <a href={item.url}>{item.title}</a>
      {/if}
    </div>
  {/each}
</div>
