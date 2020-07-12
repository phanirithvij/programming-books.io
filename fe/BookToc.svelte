<script>
  import { getLocationLastElement } from "./util.js";
  import { item } from "./item.js";

  let tocItems = [];

  let no = 0;

  const pageURL = getLocationLastElement();
  function makeItem(it, indent) {
    const url = item.url(it);
    const isSelected = url == pageURL;
    no++;
    return {
      title: item.title(it),
      url: url,
      indent: indent,
      isSelected: isSelected,
      no: no
    };
  }

  function getChildrenForIdx(idx) {
    const it = gTocItems[idx];
    const allChildren = item.childrenForParentIdx(idx);
    const res = [];
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
    tocItems = getChildrenForIdx(-1);
    // console.log("calcTocItems:", tocItems);
  }

  calcTocItems();
</script>

<style>
  .toc {
    /* font-size: 0.8em; */
    columns: 6; /* for sizes over 1200px */
  }

  /* TODO: possibly tweak those breakpoints e.g. add a column every 240px? */
  @media (max-width: 1600px) {
    .toc {
      columns: 5;
    }
  }

  @media (max-width: 1280px) {
    .toc {
      columns: 4;
    }
  }

  @media (max-width: 960px) {
    .toc {
      columns: 3;
    }
  }

  @media (max-width: 640px) {
    .toc {
      columns: 2;
    }
  }

  @media (max-width: 320px) {
    .toc {
      columns: 1;
    }
  }

  .no {
    color: lightslategray;
    display: inline-block;
    width: 2em;
  }

  .current {
    font-weight: bold;
  }
</style>

<div class="toc">
  {#each tocItems as item}
    <div class="chapters-toc-item">
      <span class="no">{item.no}</span>
      {#if item.isSelected}
        <span class="current">{item.title}</span>
      {:else}
        <a href={item.url}>{item.title}</a>
      {/if}
    </div>
  {/each}
</div>
