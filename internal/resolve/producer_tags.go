package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

type producerTag struct{ name, body string }

func producerTags(source string) []producerTag {
	tags := []producerTag{}

	for i := 0; i < len(source); {
		if strings.HasPrefix(source[i:], "<!---") {
			end, closed := parser.CFMLCommentEnd(source, i)
			if !closed {
				return nil
			}

			i = end

			continue
		}

		if source[i] != '<' || (!strings.HasPrefix(strings.ToLower(source[i:min(i+4, len(source))]), "<cf") && !strings.HasPrefix(strings.ToLower(source[i:min(i+5, len(source))]), "</cf")) {
			i++

			continue
		}

		start := i
		i++
		quote := byte(0)

	tagEnd:
		for i < len(source) {
			ch := source[i]
			switch {
			case quote != 0:
				if ch == quote {
					if i+1 < len(source) && source[i+1] == quote {
						i += 2

						continue
					}

					quote = 0
				}
			case ch == '\'' || ch == '"':
				quote = ch
			case ch == '>':
				break tagEnd
			}

			i++
		}

		if i >= len(source) {
			return nil
		}

		text := strings.TrimSpace(source[start+1 : i])
		i++

		end := 0
		for end < len(text) && text[end] != ' ' && text[end] != '\t' && text[end] != '\r' && text[end] != '\n' {
			end++
		}

		name := strings.ToLower(text[:end])
		if !strings.HasPrefix(strings.TrimPrefix(name, "/"), "cf") {
			continue
		}

		tags = append(tags, producerTag{name: name, body: strings.TrimSpace(strings.TrimSuffix(text[end:], "/"))})
	}

	return tags
}

func producerAttr(body, key string) (string, bool) {
	tokens := producerTokens(body)
	for i := 0; i+2 < len(tokens); i++ {
		if !strings.EqualFold(tokens[i].Value, key) || tokens[i+1].Kind != parser.TokEquals {
			continue
		}

		t := tokens[i+2]
		if t.Kind == parser.TokString && len(t.Value) >= 2 {
			quote := t.Value[:1]

			return strings.ReplaceAll(t.Value[1:len(t.Value)-1], quote+quote, quote), true
		}

		if t.Kind == parser.TokIdent || t.Kind == parser.TokNumber {
			return t.Value, true
		}
	}

	return "", false
}

// tagProducerSpans locates each tag function a plan can be made for: its
// lowercased name → the indexes in tags of its <cffunction> and </cffunction>.
// A name declared twice is its last declaration, as a map of plans filled in
// order kept it.
func tagProducerSpans(tags []producerTag) map[string][2]int {
	spans := map[string][2]int{}

	for i := 0; i < len(tags); i++ {
		if tags[i].name != "cffunction" {
			continue
		}

		name, ok := producerAttr(tags[i].body, "name")
		if !ok {
			continue
		}

		end := i + 1
		for end < len(tags) && tags[end].name != "/cffunction" {
			if tags[end].name == "cffunction" {
				break
			}

			end++
		}

		if end >= len(tags) || tags[end].name != "/cffunction" {
			continue
		}

		spans[strings.ToLower(name)] = [2]int{i, end}

		i = end
	}

	return spans
}

// tagProducerPlan is the plan of the tag function whose tags, between its
// <cffunction> and </cffunction>, are body.
func tagProducerPlan(body []producerTag) *producerMethod {
	method := &producerMethod{defaults: map[string]string{}}

	for _, tag := range body {
		if tag.name != "cfargument" {
			continue
		}

		arg, found := producerAttr(tag.body, "name")
		if !found {
			continue
		}

		arg = strings.ToLower(arg)
		method.parameters = append(method.parameters, arg)

		if def, found := producerAttr(tag.body, "default"); found {
			// A computed default is present even though its value is unknown.
			method.defaults[arg] = "'" + strings.ReplaceAll(def, "'", "''") + "'"
		}
	}

	p := producerTagParser{tags: body}

	if len(body)+1 <= 4096 {
		method.body = p.block()
	} else {
		p.failed = true
	}

	if !p.failed && p.position == len(p.tags) {
		method.sensitive = (producerSensitive(method.body) || producerCallBinding(method.body)) && !producerReturnsThis(method.body)
		method.wanted = producerWanted(method.body)
	} else {
		var source strings.Builder
		for _, tag := range body {
			source.WriteString(tag.body)
			source.WriteByte('\n')

			if (tag.name == "cfif" || tag.name == "cfscript") && producerSensitive([]producerNode{{kind: "if", expression: tag.body}}) {
				method.sensitive = true
			}
		}

		method.body = []producerNode{{kind: "unsafe", expression: source.String()}}
	}

	return method
}

type producerTagParser struct {
	tags     []producerTag
	position int
	failed   bool
}

func (p *producerTagParser) block() []producerNode {
	nodes := []producerNode{}

	for !p.failed && p.position < len(p.tags) {
		tag := p.tags[p.position]
		if strings.HasPrefix(tag.name, "/") || tag.name == "cfelse" || tag.name == "cfelseif" || tag.name == "cfcatch" {
			break
		}

		p.position++

		switch tag.name {
		case "cfscript":
			tokens := producerTokens(tag.body)
			if tokens == nil {
				nodes = append(nodes, producerNode{kind: "unsafe", expression: tag.body})

				continue
			}

			script := producerScript{tokens: tokens}

			body := script.block()
			if !script.valid() {
				nodes = append(nodes, producerNode{kind: "unsafe", expression: tag.body})

				continue
			}

			nodes = append(nodes, producerNode{kind: "block", body: body})
		case "cfsetting":
			if strings.ContainsAny(tag.body, "#(") {
				nodes = append(nodes, producerNode{kind: "unsafe"})
			}
		case "cfargument", "cfqueryparam", "cfthrow", "cfabort", "cfexit":
			if tag.name == "cfthrow" || tag.name == "cfabort" || tag.name == "cfexit" {
				nodes = append(nodes, producerNode{kind: "stop"})
			}
		case "cfset":
			nodes = append(nodes, producerAssignment(producerTokens(tag.body)))
		case "cfreturn":
			nodes = append(nodes, producerNode{kind: "return", expression: tag.body})
		case "cfif":
			nodes = append(nodes, p.branch(tag.body))
		case "cfloop":
			body := p.block()
			p.close("/cfloop")

			target, _ := producerAttr(tag.body, "index")
			if target == "" {
				target, _ = producerAttr(tag.body, "item")
			}

			nodes = append(nodes, producerNode{kind: "loop", target: strings.ToLower(target), body: body, header: tag.body})
		case "cftry":
			body := p.block()

			var alternate []producerNode

			if p.position < len(p.tags) && p.tags[p.position].name == "cfcatch" {
				p.position++
				alternate = p.block()
				p.close("/cfcatch")
			}

			p.close("/cftry")

			nodes = append(nodes, producerNode{kind: "try", body: body, alternative: alternate})
		case "cfquery":
			body := p.block()
			p.close("/cfquery")

			target, _ := producerAttr(tag.body, "name")
			result, _ := producerAttr(tag.body, "result")
			attrs, _ := producerAttr(tag.body, "attributeCollection")

			if strings.Contains(target+result, "#") {
				p.failed = true
			}

			nodes = append(nodes, producerNode{kind: "query", target: target, expression: strings.Trim(attrs, "#"), body: body, alternative: []producerNode{{target: result}}})
		case "cflock", "cftransaction", "cfsavecontent":
			body := p.block()
			p.close("/" + tag.name)

			target, _ := producerAttr(tag.body, "name")
			if tag.name == "cfsavecontent" {
				target, _ = producerAttr(tag.body, "variable")
			}

			if target != "" {
				nodes = append(nodes, producerNode{kind: "set", target: strings.ToLower(target), expression: "''"})
			}

			nodes = append(nodes, producerNode{kind: "block", body: body})
		case "cfparam":
			target, _ := producerAttr(tag.body, "name")
			nodes = append(nodes, producerNode{kind: "param", target: strings.ToLower(target)})
		default:
			p.failed = true
		}
	}

	return nodes
}

func (p *producerTagParser) close(name string) {
	if p.position >= len(p.tags) || p.tags[p.position].name != name {
		p.failed = true

		return
	}

	p.position++
}

func (p *producerTagParser) branch(condition string) producerNode {
	body := p.block()

	var alternate []producerNode

	if p.position < len(p.tags) {
		switch p.tags[p.position].name {
		case "cfelse":
			p.position++
			alternate = p.block()
			p.close("/cfif")
		case "cfelseif":
			next := p.tags[p.position].body
			p.position++
			alternate = []producerNode{p.branch(next)}
		default:
			p.close("/cfif")
		}
	} else {
		p.failed = true
	}

	return producerNode{kind: "if", expression: condition, body: body, alternative: alternate}
}
