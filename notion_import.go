package main

import "github.com/kjk/notionapi"

// convert 2131b10c-ebf6-4938-a127-7089ff02dbe4 to 2131b10cebf64938a1277089ff02dbe4
// TODO: replace with direct use of notionapi.ToNoDashID
func toNoDashID(id string) string {
	return notionapi.ToNoDashID(id)
}
