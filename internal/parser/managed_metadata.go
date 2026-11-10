package parser

import (
	"strings"
	"sync"
)

// Read declarations without a second signature, reference or body parse.
//
// The answer depends on the file's text and how it was split, and every parse
// that has both a setter and a property-bean lookup asks: a batch scan's index,
// scan and handler parses of each DI/1-managed file. It is kept by those, and
// is shared: prepareManagedSetterLookup only reads it.
func (pr *ParseResult) managedPropertyMetadata() *ParseResult {
	key := managedKey{uri: string(pr.URI), content: pr.Content, interpolateAll: pr.interpolateAll}

	managedMu.Lock()
	held, ok := managedMemo[key]
	managedMu.Unlock()

	if ok && pr.regionsOf == pr.Content {
		return held
	}

	metadata := pr.readManagedPropertyMetadata()

	if pr.regionsOf == pr.Content {
		managedMu.Lock()
		if len(managedMemo) >= maxManagedMemo {
			clear(managedMemo)
		}

		managedMemo[key] = metadata
		managedMu.Unlock()
	}

	return metadata
}

// managedKey is what managedPropertyMetadata's answer depends on: the regions
// come from the URI, the text and whether all text is interpolated.
type managedKey struct {
	uri, content   string
	interpolateAll bool
}

// maxManagedMemo bounds managedMemo, a count of files cleared when reached.
// A key holds its text alive.
const maxManagedMemo = 512

var (
	managedMu   sync.Mutex
	managedMemo = map[managedKey]*ParseResult{}
)

func (pr *ParseResult) readManagedPropertyMetadata() *ParseResult {
	metadata := &ParseResult{}

	for _, region := range pr.Regions {
		switch region.Kind {
		case RegionScript:
			pr.managedScriptMetadata(region, metadata)
		case RegionTag:
			pr.managedTagMetadata(region, metadata)
		default:
		}
	}

	return metadata
}

func (pr *ParseResult) managedScriptMetadata(region Region, metadata *ParseResult) {
	sp := newScriptParser(region.Text, string(pr.URI), region.StartLine, nil)
	depth, top := 0, 0

	for {
		tok := sp.sc.NextSkipComments()
		if tok.Kind == TokEOF {
			break
		}

		switch tok.Kind {
		case TokLBrace:
			depth++
		case TokRBrace:
			depth--
		case TokIdent:
			if depth != top {
				continue
			}

			if strings.EqualFold(tok.Value, "component") {
				sp.parseComponentAttrs()

				top = depth + 1
			}

			if strings.EqualFold(tok.Value, "property") {
				sp.parseProperty(tok)
			}
		default:
		}
	}

	metadata.Properties = append(metadata.Properties, sp.properties...)
	metadata.Accessors = metadata.Accessors || sp.accessors
	metadata.Persistent = metadata.Persistent || sp.persistent
}

func (pr *ParseResult) managedTagMetadata(region Region, metadata *ParseResult) {
	tp := newTagParser(region.Text, string(pr.URI))

	ends := newTagEnds(region.Text)

	scanner := NewScanner(region.Text)
	for {
		tok := scanner.NextSkipComments()
		if tok.Kind == TokEOF {
			break
		}

		if tok.Kind != TokLT {
			continue
		}

		word := scanner.NextSkipComments()
		if word.Kind != TokIdent {
			continue
		}

		if !strings.EqualFold(word.Value, "cfcomponent") && !strings.EqualFold(word.Value, "cfproperty") {
			continue
		}

		end := ends.at(tok.Offset)
		if end < 0 {
			continue
		}

		tag := region.Text[tok.Offset : tok.Offset+end+1]
		if strings.EqualFold(word.Value, "cfproperty") {
			tp.parseCFProperty(tag, region.StartLine+int(tok.Line))
		} else {
			metadata.Accessors = metadata.Accessors || isTruthy(getAttr(tag, "accessors"))
			metadata.Persistent = metadata.Persistent || isTruthy(getAttr(tag, "persistent"))
		}
	}

	metadata.Properties = append(metadata.Properties, tp.properties...)
}
