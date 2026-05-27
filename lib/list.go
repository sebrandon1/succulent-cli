package lib

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

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

func (c *Client) ListEnvironmentsWithInfo(concurrency int) ([]EnvironmentDetail, error) {
	envs, err := c.ListEnvironments()
	if err != nil {
		return nil, err
	}

	details := make([]EnvironmentDetail, len(envs))
	sem := make(chan struct{}, concurrency)

	var wg sync.WaitGroup

	for i, env := range envs {
		details[i] = EnvironmentDetail{
			Name:   env.Name,
			Group:  env.Group,
			Status: "empty",
		}

		wg.Add(1)

		go func(idx int, envName string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			info, infoErr := c.GetInfoPlan(envName)
			if infoErr != nil || len(info.Nodes) == 0 {
				return
			}

			d := &details[idx]
			d.InstallerIP = info.InstallerIP
			d.NodeCount = len(info.Nodes)
			d.CreationDate = info.CreationDate

			if info.CreationDate != "" {
				parts := strings.Fields(info.CreationDate)
				if len(parts) >= 3 {
					d.Owner = parts[len(parts)-1]
				}
			}

			nodesUp := 0

			for _, node := range info.Nodes {
				if node.Status == StatusUp {
					nodesUp++
				}
			}

			d.NodesUp = nodesUp

			if nodesUp == d.NodeCount {
				d.Status = "active"
			} else {
				d.Status = "partial"
			}
		}(i, env.Name)
	}

	wg.Wait()

	return details, nil
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
