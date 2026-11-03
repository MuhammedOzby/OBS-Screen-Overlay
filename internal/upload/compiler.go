package upload

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// CompileHTML validates a complete document and returns its rendered HTML and
// distinct field names in first source appearance order. Values are plain text;
// missing values are empty and unused map entries are ignored. SSE policy has
// no effect on this initial rendering. No runtime is injected at this stage.
// The source is never executed as a Go template or modified in place.
// Placeholders must appear literally in source text, not through HTML entities.
// Unsupported or ambiguous insertion contexts are rejected conservatively.
func CompileHTML(source []byte, values map[string]string) ([]byte, []string, error) {
	return compileHTML(source, values)
}

func compileHTML(source []byte, values map[string]string) ([]byte, []string, error) {
	original, err := validatePlaceholderTree(source)
	if err != nil {
		return nil, nil, err
	}
	// A private attribute marks only the synthetic nodes in the verification
	// parse; ensure it cannot collide with an attribute in the source.
	marker := "data-overlay-compiler-check"
	for bytes.Contains(bytes.ToLower(source), []byte(marker)) {
		marker += "x"
	}
	var output, check bytes.Buffer
	var fields []string
	seen := make(map[string]bool)
	count, tableDepth := 0, 0
	z := html.NewTokenizer(bytes.NewReader(source))
	for {
		typ := z.Next()
		if typ == html.ErrorToken {
			if z.Err() != io.EOF {
				return nil, nil, fmt.Errorf("HTML ayrıştırılamadı: %w", z.Err())
			}
			break
		}
		raw := string(z.Raw())
		tok := z.Token()
		if typ == html.StartTagToken && tok.Data == "table" {
			tableDepth++
		}
		if typ == html.EndTagToken && tok.Data == "table" && tableDepth > 0 {
			tableDepth--
		}
		if typ == html.CommentToken && placeholderRE.MatchString(tok.Data) {
			return nil, nil, errPlaceholderComment
		}
		if typ == html.StartTagToken || typ == html.SelfClosingTagToken {
			for _, a := range tok.Attr {
				if placeholderRE.MatchString(a.Val) {
					return nil, nil, errPlaceholderAttr
				}
			}
		}
		if typ != html.TextToken {
			output.WriteString(raw)
			check.WriteString(raw)
			continue
		}
		matches := placeholderRE.FindAllStringIndex(raw, -1)
		// Entity-encoded placeholders cannot be mapped back to source bytes
		// unambiguously without changing unrelated markup; reject them.
		decoded := placeholderRE.FindAllString(tok.Data, -1)
		if len(decoded) != len(matches) {
			return nil, nil, errPlaceholderText
		}
		last := 0
		for i, m := range matches {
			literal := raw[m[0]:m[1]]
			if decoded[i] != literal || tableDepth > 0 {
				return nil, nil, errPlaceholderText
			}
			name := literal[2 : len(literal)-2]
			if !seen[name] {
				seen[name] = true
				fields = append(fields, name)
			}
			output.WriteString(raw[last:m[0]])
			check.WriteString(raw[last:m[0]])
			fmt.Fprintf(&output, `<span data-overlay-field="%s">%s</span>`, name, stdhtml.EscapeString(values[name]))
			fmt.Fprintf(&check, `<span %s="">%s</span>`, marker, literal)
			count++
			last = m[1]
		}
		output.WriteString(raw[last:])
		check.WriteString(raw[last:])
	}
	modified, err := html.Parse(bytes.NewReader(check.Bytes()))
	if err != nil {
		return nil, nil, fmt.Errorf("HTML ayrıştırılamadı: %w", err)
	}
	var restore func(*html.Node) error
	restore = func(n *html.Node) error {
		for child := n.FirstChild; child != nil; {
			next := child.NextSibling
			if child.Type == html.ElementNode && len(child.Attr) == 1 && child.Attr[0].Key == marker {
				if child.Namespace != "" || child.Data != "span" || child.FirstChild == nil || child.FirstChild != child.LastChild || child.FirstChild.Type != html.TextNode || !placeholderTextAllowed(child.FirstChild) {
					return errPlaceholderText
				}
				text := &html.Node{Type: html.TextNode, Data: child.FirstChild.Data}
				n.InsertBefore(text, child)
				n.RemoveChild(child)
				count--
			} else if err := restore(child); err != nil {
				return err
			}
			child = next
		}
		return nil
	}
	if err := restore(modified); err != nil || count != 0 {
		return nil, nil, errPlaceholderText
	}
	var before, after strings.Builder
	if err := html.Render(&before, original); err != nil {
		return nil, nil, err
	}
	if err := html.Render(&after, modified); err != nil {
		return nil, nil, err
	}
	if before.String() != after.String() {
		return nil, nil, errPlaceholderText
	}
	return output.Bytes(), fields, nil
}
