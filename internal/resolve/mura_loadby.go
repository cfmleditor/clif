package resolve

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// Mura documents a bean's loadBy() as loading the bean and handing it back,
// and declares it nowhere: contentBean's returns what contentManager.read()
// does, which is the bean it was handed or one from a cache, through six
// branches no return can be followed down. It is called on the bean
// getBean() built, `$.getBean( "content" ).loadBy( contentID = id )`, and on
// MasaCMS was the last assignment of 315 variables reported untyped.

// muraLoadBy is the component at path when call is its loadBy() and path is a
// Mura bean, one whose chain reaches mura.bean.bean. call is the method's name
// or the call itself, with its arguments. A bean whose loadBy takes a
// returnFormat (beanORM's, which returns a query or an iterator when asked)
// is itself only when the call spells "self" or names no format, which needs
// the arguments.
func (r *Resolver) muraLoadBy(path, call string) string {
	name, args, hasArgs := strings.Cut(call, "(")
	if path == "" || !strings.EqualFold(strings.TrimSpace(name), "loadBy") || !r.extendsReaches(path, "mura.bean.bean", "/mura/bean/bean.cfc") {
		return ""
	}

	fd := r.ResolveFunc(path, "loadBy", filepath.Dir(path))
	if fd == nil || !takesReturnFormat(fd) {
		return path
	}

	if !hasArgs || !selfFormat(strings.TrimSuffix(strings.TrimSpace(args), ")")) {
		return ""
	}

	return path
}

func takesReturnFormat(fd *parser.FunctionDef) bool {
	for i := range fd.Arguments {
		if strings.EqualFold(fd.Arguments[i].Name, "returnFormat") {
			return true
		}
	}

	return false
}

var (
	returnFormatRe = regexp.MustCompile(`(?i)\breturnFormat\s*=`)
	selfFormatRe   = regexp.MustCompile(`(?i)\breturnFormat\s*=\s*["']self["']`)
	positionalRe   = regexp.MustCompile(`^\s*[^\s=]+\s*(?:,|$)`)
)

// selfFormat reports whether a loadBy call's arguments leave the format the
// bean itself: they name it "self", or name every argument and none of them
// the format. A positional argument or an argumentCollection could be one.
func selfFormat(args string) bool {
	if selfFormatRe.MatchString(args) {
		return true
	}

	if returnFormatRe.MatchString(args) || parser.IndexFold(args, "argumentCollection") >= 0 {
		return false
	}

	return strings.TrimSpace(args) == "" || !positionalRe.MatchString(args)
}
