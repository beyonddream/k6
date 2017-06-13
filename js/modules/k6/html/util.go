package html

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dop251/goja"
	gohtml "golang.org/x/net/html"
)

var (
	lowerRe = regexp.MustCompile("-[[:lower:]]")
	upperRe = regexp.MustCompile("[[:upper:]]")
)

func toAttrName(dataName string) string {
	return upperRe.ReplaceAllStringFunc(dataName, dataNameCb)
}

func toDataName(attrName string) string {
	return lowerRe.ReplaceAllStringFunc(attrName, attrNameCb)
}

// Receives a single upper case char
func dataNameCb(s string) string {
	return "-" + strings.ToLower(s)
}

// Receives a single lower case letter with a hyphen prefix
func attrNameCb(s string) string {
	return strings.ToUpper(s[1:])
}

func namespaceURI(prefix string) string {
	switch prefix {
	case "svg":
		return "http://www.w3.org/2000/svg"
	case "math":
		return "http://www.w3.org/1998/Math/MathML"
	default:
		return "http://www.w3.org/1999/xhtml"
	}
}

func valueOrHTML(s *goquery.Selection) string {
	if val, exists := s.Attr("value"); exists {
		return val
	}

	if val, err := s.Html(); err == nil {
		return val
	}

	return ""
}

func getHtmlAttr(node *gohtml.Node, name string) *gohtml.Attribute {
	for i := 0; i < len(node.Attr); i++ {
		if node.Attr[i].Key == name {
			return &node.Attr[i]
		}
	}

	return nil
}

func elemList(s Selection) (items []goja.Value) {
	for i := 0; i < s.Size(); i++ {
		items = append(items, selToElement(s.Eq(i)))
	}
	return items
}

func nodeToElement(e Element, node *gohtml.Node) goja.Value {
	// Goquery does not expose a way to build a goquery.Selection with an arbitraty html.Node
	// so workaround by making an empty Selection and directly adding the node
	emptySel := e.sel.emptySelection()
	emptySel.sel.Nodes = append(emptySel.sel.Nodes, node)

	elem := Element{node, &emptySel}

	return emptySel.rt.ToValue(elem)
}

func selToElement(sel Selection) goja.Value {
	if sel.sel.Length() == 0 {
		return goja.Undefined()
	}

	elem := Element{sel.sel.Nodes[0], &sel}

	return sel.rt.ToValue(elem)
}

// Some Selection methods use an interface{} to handle an argument which may be a Element/Selection/string or a goja wrapper of those types
// This function unwraps the goja value into it's native go type to be used in a type switch
func exportIfGojaVal(arg interface{}) interface{} {
	if gojaArg, ok := arg.(goja.Value); ok {
		return gojaArg.Export()
	}

	return arg
}

// Try to read numeric values in data- attributes.
// Return numeric value when the representation is unchanged by conversion to float and back.
// Other potentially numeric values (ie "101.00" "1E02") remain as strings.
func toNumeric(val string) (float64, bool) {
	if fltVal, err := strconv.ParseFloat(val, 64); err != nil {
		return 0, false
	} else if repr := strconv.FormatFloat(fltVal, 'f', -1, 64); repr == val {
		return fltVal, true
	} else {
		return 0, false
	}
}

func convertDataAttrVal(val string) interface{} {
	if len(val) == 0 {
		return goja.Undefined()
	} else if val[0] == '{' || val[0] == '[' {
		var subdata interface{}

		err := json.Unmarshal([]byte(val), &subdata)
		if err == nil {
			return subdata
		} else {
			return val
		}
	} else {
		switch val {
		case "true":
			return true

		case "false":
			return false

		case "null":
			return goja.Undefined()

		case "undefined":
			return goja.Undefined()

		default:
			if fltVal, isOk := toNumeric(val); isOk {
				return fltVal
			} else {
				return val
			}
		}
	}
}
