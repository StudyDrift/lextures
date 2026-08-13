package render

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	mermaidFenceRE  = regexp.MustCompile(`(?m)^` + "```" + `[ \t]*mermaid[ \t]*\r?\n([\s\S]*?)^` + "```" + `[ \t]*$`)
	mermaidHeaderRE = regexp.MustCompile(`(?i)^(graph|flowchart)\s+(TD|TB|BT|LR|RL)?\s*$`)
	mermaidStyleRE  = regexp.MustCompile(`(?i)^style\s+(\S+)\s+(.+)$`)
	mermaidFillRE   = regexp.MustCompile(`(?i)fill\s*:\s*#([0-9a-fA-F]{3,8})`)
	mermaidSubRE    = regexp.MustCompile(`(?i)^subgraph\s+(.+)$`)
)

type mermaidNode struct {
	id, label, tone string
}

type mermaidSubgraph struct {
	title string
	ids   []string
}

func extractMermaidFences(source string) (string, map[string]string) {
	replacements := map[string]string{}
	sequence := 0
	converted := mermaidFenceRE.ReplaceAllStringFunc(source, func(block string) string {
		m := mermaidFenceRE.FindStringSubmatch(block)
		body := ""
		if len(m) > 1 {
			body = m[1]
		}
		token := "CONTENTMERMAID" + strconv.Itoa(sequence) + "TOKEN"
		sequence++
		html := renderMermaidBlock(body)
		replacements[token] = html
		return token
	})
	return converted, replacements
}

func renderMermaidBlock(source string) string {
	direction, nodes, subgraphs, ok := parseMermaidGraph(source)
	if !ok {
		return mermaidCodeBlock(source)
	}
	return renderMermaidHTML(direction, nodes, subgraphs)
}

func parseMermaidGraph(source string) (string, []mermaidNode, []mermaidSubgraph, bool) {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	if start >= len(lines) {
		return "", nil, nil, false
	}
	header := mermaidHeaderRE.FindStringSubmatch(strings.TrimSpace(lines[start]))
	if header == nil {
		return "", nil, nil, false
	}
	direction := "td"
	if strings.EqualFold(header[2], "LR") || strings.EqualFold(header[2], "RL") {
		direction = "lr"
	}
	nodeMap := map[string]*mermaidNode{}
	var order []string
	var subgraphs []mermaidSubgraph
	current := -1

	ensure := func(id, label string) *mermaidNode {
		node, exists := nodeMap[id]
		if !exists {
			node = &mermaidNode{id: id, label: firstNonEmpty(label, id), tone: "neutral"}
			nodeMap[id] = node
			order = append(order, id)
		} else if label != "" && label != id {
			node.label = label
		}
		if current >= 0 && !containsID(subgraphs[current].ids, id) {
			subgraphs[current].ids = append(subgraphs[current].ids, id)
		}
		return node
	}

	for i := start + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		if strings.EqualFold(line, "end") {
			current = -1
			continue
		}
		if sub := mermaidSubRE.FindStringSubmatch(line); sub != nil {
			subgraphs = append(subgraphs, mermaidSubgraph{title: mermaidSubgraphTitle(sub[1])})
			current = len(subgraphs) - 1
			continue
		}
		if style := mermaidStyleRE.FindStringSubmatch(line); style != nil {
			ensure(style[1], "").tone = toneFromFill(style[2])
			continue
		}
		if hasAnyPrefixFold(line, "classDef", "class", "linkStyle", "click") {
			continue
		}
		ids, labels, ok := parseFlowLine(line)
		if !ok {
			continue
		}
		for n, id := range ids {
			ensure(id, labels[n])
		}
	}

	if len(nodeMap) == 0 {
		return "", nil, nil, false
	}
	if len(subgraphs) == 0 {
		subgraphs = []mermaidSubgraph{{ids: append([]string{}, order...)}}
	} else {
		claimed := map[string]bool{}
		for _, group := range subgraphs {
			for _, id := range group.ids {
				claimed[id] = true
			}
		}
		var leftover []string
		for _, id := range order {
			if !claimed[id] {
				leftover = append(leftover, id)
			}
		}
		if len(leftover) > 0 {
			subgraphs = append(subgraphs, mermaidSubgraph{ids: leftover})
		}
	}
	nodes := make([]mermaidNode, 0, len(order))
	for _, id := range order {
		nodes = append(nodes, *nodeMap[id])
	}
	return direction, nodes, subgraphs, true
}

func mermaidSubgraphTitle(value string) string {
	title := strings.TrimSpace(value)
	if open := strings.Index(title, "["); open >= 0 && strings.HasSuffix(title, "]") {
		return strings.Trim(strings.TrimSpace(title[open+1:len(title)-1]), `"'`)
	}
	return strings.Trim(title, `"'`)
}

func parseFlowLine(line string) ([]string, []string, bool) {
	var ids, labels []string
	i := 0
	s := line
	for i < len(s) {
		for i < len(s) && unicode.IsSpace(rune(s[i])) {
			i++
		}
		if i >= len(s) {
			break
		}
		if strings.HasPrefix(s[i:], "-.->") || strings.HasPrefix(s[i:], "-->") || strings.HasPrefix(s[i:], "==>") || strings.HasPrefix(s[i:], "---") || strings.HasPrefix(s[i:], "--") {
			if strings.HasPrefix(s[i:], "-.->") {
				i += 4
			} else if strings.HasPrefix(s[i:], "-->") || strings.HasPrefix(s[i:], "==>") || strings.HasPrefix(s[i:], "---") {
				i += 3
			} else {
				i += 2
			}
			if i < len(s) && s[i] == '|' {
				i++
				for i < len(s) && s[i] != '|' {
					i++
				}
				if i < len(s) && s[i] == '|' {
					i++
				}
			}
			continue
		}
		if i >= len(s) || !isIdentStart(rune(s[i])) {
			if len(ids) > 0 {
				return ids, labels, true
			}
			return nil, nil, false
		}
		start := i
		i++
		for i < len(s) && isIdent(rune(s[i])) {
			i++
		}
		id := s[start:i]
		label := id
		if i < len(s) {
			opener := s[i]
			if opener == '[' || opener == '(' || opener == '{' {
				closer := byte(']')
				if opener == '(' {
					closer = ')'
				} else if opener == '{' {
					closer = '}'
				}
				i++
				if i < len(s) && (s[i] == '"' || s[i] == '\'') {
					quote := s[i]
					i++
					labelStart := i
					for i < len(s) && s[i] != quote {
						i++
					}
					label = s[labelStart:i]
					if i < len(s) && s[i] == quote {
						i++
					}
					if i < len(s) && s[i] == closer {
						i++
					}
				} else {
					labelStart := i
					for i < len(s) && s[i] != closer {
						i++
					}
					label = s[labelStart:i]
					if i < len(s) && s[i] == closer {
						i++
					}
				}
			}
		}
		ids = append(ids, id)
		labels = append(labels, label)
	}
	if len(ids) == 0 {
		return nil, nil, false
	}
	return ids, labels, true
}

func toneFromFill(attrs string) string {
	match := mermaidFillRE.FindStringSubmatch(attrs)
	if match == nil {
		return "neutral"
	}
	fill := strings.ToLower(match[1])
	switch {
	case strings.HasPrefix(fill, "ffe"), strings.HasPrefix(fill, "ef6"), strings.HasPrefix(fill, "ff98"), strings.HasPrefix(fill, "f57"):
		return "warm"
	case strings.HasPrefix(fill, "ffc"), strings.HasPrefix(fill, "c62"), strings.HasPrefix(fill, "e57"), strings.HasPrefix(fill, "ef9"), strings.HasPrefix(fill, "f44"):
		return "hot"
	case strings.HasPrefix(fill, "e3"), strings.HasPrefix(fill, "bb"), strings.HasPrefix(fill, "90"), strings.HasPrefix(fill, "15"), strings.HasPrefix(fill, "19"), strings.HasPrefix(fill, "21"), strings.HasPrefix(fill, "42"):
		return "cool"
	default:
		return "neutral"
	}
}

func renderMermaidHTML(direction string, nodes []mermaidNode, subgraphs []mermaidSubgraph) string {
	byID := map[string]mermaidNode{}
	for _, node := range nodes {
		byID[node.id] = node
	}
	var titles []string
	for _, group := range subgraphs {
		if group.title != "" {
			titles = append(titles, group.title)
		}
	}
	label := strings.Join(titles, "; ")
	if label == "" {
		label = "Diagram"
	}
	var groups strings.Builder
	var description strings.Builder
	for _, group := range subgraphs {
		groups.WriteString(`<div class="content-mermaid-subgraph">`)
		if group.title != "" {
			groups.WriteString(`<p class="content-mermaid-title">` + mermaidEscape(group.title) + `</p>`)
		}
		groups.WriteString(`<ol class="content-mermaid-nodes">`)
		var labels []string
		for _, id := range group.ids {
			node, ok := byID[id]
			if !ok {
				continue
			}
			groups.WriteString(`<li class="content-mermaid-node content-mermaid-node-` + node.tone + `">` + mermaidEscape(node.label) + `</li>`)
			labels = append(labels, mermaidEscape(node.label))
		}
		groups.WriteString(`</ol></div>`)
		description.WriteString(`<p>`)
		if group.title != "" {
			description.WriteString(mermaidEscape(group.title) + `: `)
		}
		description.WriteString(strings.Join(labels, " → "))
		description.WriteString(`.</p>`)
	}
	return `<figure class="content-figure content-diagram content-mermaid"><div class="content-diagram-scroll" role="img" aria-label="` + mermaidEscape(label) + `"><div class="content-mermaid-graph content-mermaid-` + direction + `">` + groups.String() + `</div></div><details><summary>Diagram description</summary><div class="content-media-description">` + description.String() + `</div></details></figure>`
}

func mermaidCodeBlock(source string) string {
	body := strings.ReplaceAll(source, "\r\n", "\n")
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return `<pre><code class="language-mermaid">` + mermaidEscape(body) + `</code></pre>`
}

func mermaidEscape(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, `"`, "&quot;")
	return value
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdent(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

func containsID(ids []string, id string) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func hasAnyPrefixFold(line string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if len(line) >= len(prefix) && strings.EqualFold(line[:len(prefix)], prefix) {
			return true
		}
	}
	return false
}
