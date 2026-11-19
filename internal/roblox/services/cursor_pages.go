package services

import (
	"net/url"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const maxCursorLength = 4096

// readCursorPages passes each cursor to readPage, starting with an empty one,
// until readPage returns no next cursor. A cursor that is too long or that
// Roblox already returned ends the read as invalid data, so that a looping
// cursor cannot read the same pages again.
func readCursorPages(endpoint string, maxPages int, tooManyPagesMessage string, readPage func(cursor string) (nextCursor string, err error)) error {
	seenCursors := map[string]bool{}
	cursor := ""
	for range maxPages {
		nextCursor, err := readPage(cursor)
		if err != nil {
			return err
		}
		if nextCursor == "" {
			return nil
		}
		if len(nextCursor) > maxCursorLength || seenCursors[nextCursor] {
			return invalidProfileDetails(endpoint)
		}
		seenCursors[nextCursor] = true
		cursor = nextCursor
	}
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: tooManyPagesMessage}
}

// newestFirstPageQuery asks for one page of pageSize entries, newest first,
// after cursor.
func newestFirstPageQuery(pageSize int, cursor string) url.Values {
	query := url.Values{"limit": {strconv.Itoa(pageSize)}, "sortOrder": {"Desc"}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	return query
}

// nextCursorOf returns the cursor of the next page, or false when Roblox
// returned a cursor too long to send back.
func nextCursorOf(nextPageCursor *string) (string, bool) {
	if nextPageCursor == nil {
		return "", true
	}
	return *nextPageCursor, len(*nextPageCursor) <= maxCursorLength
}
