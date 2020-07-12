import { item } from "./item.js";

const maxSearchResults = 25;

// el is [idx, len]
// sort by idx.
// if idx is the same, sort by reverse len
// (i.e. bigger len is first)
function sortSearchByIdx(el1, el2) {
	var res = el1[0] - el2[0];
	if (res == 0) {
		res = el2[1] - el1[1];
	}
	return res;
}

// [[idx, len], ...]
// sort by idx, if there is an overlap, drop overlapped elements
function sortSearchMatches(a) {
	if (a.length < 2) {
		return a;
	}
	a.sort(sortSearchByIdx);
	var lastIdx = a[0][0] + a[0][1]; // start + len
	var n = a.length;
	var res = [a[0]];
	for (var i = 1; i < n; i++) {
		var el = a[i];
		var idx = el[0];
		var len = el[1];
		if (idx >= lastIdx) {
			res.push(el);
			lastIdx = idx + len;
		}
	}
	return a;
}

// searches s for toFind and toFindArr.
// returns null if no match
// returns array of [idx, len] position in $s where $toFind or $toFindArr matches
function searchMatch(s, toFind, toFindArr) {
	s = s.toLowerCase();

	// try exact match
	var idx = s.indexOf(toFind);
	if (idx != -1) {
		return [[idx, toFind.length]];
	}

	// now see if matches for search for AND of components in toFindArr
	if (!toFindArr) {
		return null;
	}

	var n = toFindArr.length;
	var res = Array(n);
	for (var i = 0; i < n; i++) {
		toFind = toFindArr[i];
		idx = s.indexOf(toFind);
		if (idx == -1) {
			return null;
		}
		res[i] = [idx, toFind.length];
	}
	return sortSearchMatches(res);
}

function notEmptyString(s) {
	return s.length > 0;
}

/*
returns null if no match
returns: {
  term: "",
  match: [[idx, len], ...]
}
*/
function searchMatchMulti(toSearchArr, toFind) {
	var toFindArr = toFind.split(" ").filter(notEmptyString);
	var n = toSearchArr.length;
	for (var i = 0; i < n; i++) {
		var toSearch = toSearchArr[i];
		var match = searchMatch(toSearch, toFind, toFindArr);
		if (match) {
			return {
				term: toSearch,
				match: match,
				tocItem: null // will be filled later
			};
		}
	}
	return null;
}

// if search term is multiple words like "blank id",
// we search for both the exact match and if we match all
// terms ("blank", "id") separately
export function search(searchTerm) {
	// console.log("search for:", searchTerm);
	const a = gTocItems; // loaded via toc_search.js, generated in gen_book_toc_search.go
	const n = a.length;
	const res = [];
	for (let i = 0; i < n && res.length < maxSearchResults; i++) {
		var tocItem = a[i];
		var searchable = item.searchable(tocItem);
		var match = searchMatchMulti(searchable, searchTerm);
		if (!match) {
			continue;
		}
		match.tocItem = tocItem;
		match.id = i;
		res.push(match);
	}
	return res;
}
