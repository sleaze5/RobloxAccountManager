package services

import "github.com/sleaze5/RobloxAccountManager/internal/roblox"

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
