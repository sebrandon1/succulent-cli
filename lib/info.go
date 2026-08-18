package lib

import (
	"context"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	ipRegex = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)

	errorStatuses = map[string]bool{
		StatusError:       true,
		StatusFailed:      true,
		StatusUnreachable: true,
	}
)

func (c *Client) GetInfoPlan(ctx context.Context, env string) (*ClusterInfo, error) {
	requestURL := fmt.Sprintf("%s"+endpointInfoPlan, c.BaseURL, env)

	resp, err := c.getRaw(ctx, requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch infoplan for %s: %w", env, err)
	}
	defer resp.Body.Close()

	return parseInfoPlan(env, limitedBody(resp.Body, MaxResponseSize))
}

func parseInfoPlan(env string, r io.Reader) (*ClusterInfo, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	info := &ClusterInfo{
		Environment: env,
		Nodes:       []NodeInfo{},
	}

	var rows [][]string
	extractTableRows(doc, &rows)

	headerParsed := false

	for _, cells := range rows {
		if len(cells) < 2 {
			continue
		}

		col0 := strings.TrimSpace(cells[0])
		col1 := strings.TrimSpace(cells[1])

		if !headerParsed && strings.EqualFold(col0, "plan name") {
			headerParsed = true

			continue
		}

		if headerParsed && info.PlanName == "" && !strings.EqualFold(col0, "vm name") {
			info.PlanName = col0
			info.Client = col1

			if len(cells) > 2 {
				info.CreationDate = strings.TrimSpace(cells[2])
			}

			continue
		}

		if strings.EqualFold(col0, "vm name") {
			continue
		}

		node := parseNodeRow(col0, col1, cells)
		if node == nil {
			continue
		}

		info.Nodes = append(info.Nodes, *node)

		if strings.Contains(strings.ToLower(col0), "installer") && node.IP != "" {
			info.InstallerIP = node.IP
		}
	}

	return info, nil
}

func parseNodeRow(name, status string, cells []string) *NodeInfo {
	statusLower := strings.ToLower(status)
	if statusLower != StatusUp && statusLower != StatusDown && !errorStatuses[statusLower] {
		return nil
	}

	node := &NodeInfo{
		Name:     name,
		Status:   statusLower,
		NodeType: classifyNodeType(name),
	}

	if len(cells) > 2 {
		if match := ipRegex.FindString(cells[2]); match != "" && net.ParseIP(match) != nil {
			node.IP = match
		}
	}

	return node
}

func classifyNodeType(name string) string {
	nameLower := strings.ToLower(name)

	switch {
	case strings.Contains(nameLower, "installer"):
		return NodeTypeInstaller
	case strings.Contains(nameLower, "master"):
		return NodeTypeMaster
	case strings.Contains(nameLower, "bootstrap"):
		return NodeTypeBootstrap
	default:
		return NodeTypeWorker
	}
}

func extractTableRows(n *html.Node, rows *[][]string) {
	if n.Type == html.ElementNode && n.Data == "tr" {
		var cells []string

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
				cells = append(cells, extractText(c))
			}
		}

		if len(cells) > 0 {
			*rows = append(*rows, cells)
		}

		return
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractTableRows(c, rows)
	}
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var sb strings.Builder

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}

	return sb.String()
}
