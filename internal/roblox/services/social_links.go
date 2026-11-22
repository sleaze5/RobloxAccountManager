package services

import "net/url"

const maxSocialLinkTextLength = 100

// SocialLink is a link that a group or a game lists on its page, such as its
// Discord server.
type SocialLink struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

// listedSocialLink is a social link as the groups and games APIs list it.
type listedSocialLink struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

func socialLinksFromListed(listedLinks []listedSocialLink) []SocialLink {
	links := make([]SocialLink, 0, len(listedLinks))
	for _, listedLink := range listedLinks {
		// Only web links open, so that a link never starts another program.
		if parsedLink, err := url.Parse(listedLink.URL); err == nil && parsedLink.Scheme == "https" && parsedLink.Host != "" {
			links = append(links, SocialLink{Type: boundedText(listedLink.Type, maxSocialLinkTextLength), URL: parsedLink.String(), Title: boundedText(listedLink.Title, maxSocialLinkTextLength)})
		}
	}
	return links
}
