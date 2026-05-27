package lib

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var infoplanLinkRegex = regexp.MustCompile(`/infoplan/([a-zA-Z0-9_-]+)`)

func (c *Client) ListEnvironments() ([]EnvironmentInfo, error) {
	resp, err := c.getRaw(c.BaseURL + "/")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch environment list: %w", err)
	}
	defer resp.Body.Close()

	return parseEnvironmentList(resp.Body)
}

func parseEnvironmentList(r io.Reader) ([]EnvironmentInfo, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var envs []EnvironmentInfo
	seen := make(map[string]bool)
	currentGroup := ""

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "th" {
			text := strings.TrimSpace(extractText(n))
			if hasAttr(n, "colspan") && strings.HasPrefix(text, "Hosts ") {
				currentGroup = strings.TrimPrefix(text, "Hosts ")
			}
		}

		if n.Type == html.ElementNode && n.Data == "button" {
			onclick := getAttr(n, "onclick")
			if matches := infoplanLinkRegex.FindStringSubmatch(onclick); len(matches) > 1 {
				name := matches[1]
				if !seen[name] {
					seen[name] = true
					envs = append(envs, EnvironmentInfo{
						Name:  name,
						Group: currentGroup,
					})
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	return envs, nil
}

func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}

	return false
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}

	return ""
}
