package src

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"sourcegraph.com/sourcegraph/srclib/graph"

	"github.com/peterh/liner"
)

func init() {
	interactiveGroup, err := CLI.AddCommand("interactive",
		"interactive REPL for build data",
		"The interactive (i) command is a readline-like interface for interacting with the build data for a project.",
		&interactiveCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
	interactiveGroup.Aliases = append(interactiveGroup.Aliases, "i")
}

type InteractiveCmd struct{}

var interactiveCmd InteractiveCmd

var historyFile = "/tmp/.srclibi_history"

var activeRepo = "."

var activeContext commandContext

func (c *InteractiveCmd) Execute(args []string) error {
	fmt.Printf("Analyzing project...")
	// Build project concurrently so we can update the UI.
	type maybeContext struct {
		context commandContext
		err     error
	}
	done := make(chan maybeContext)
	go func() {
		context, err := prepareCommandContext(activeRepo)
		done <- maybeContext{context, err}
	}()
OuterLoop:
	for {
		select {
		case <-time.Tick(time.Second):
			fmt.Print(".")
		case m := <-done:
			if m.err != nil {
				fmt.Println()
				return m.err
			}
			activeContext = m.context
			break OuterLoop
		}
	}
	fmt.Println()
	// Invariant: activeContext is the result of prepareCommandContext
	// after the loop above.

	term, err := terminal(historyFile)
	if err != nil {
		return err
	}
	defer persist(term, historyFile)

	for {
		line, err := term.Prompt("src> ")
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return nil
			}
			return err
		}
		term.AppendHistory(line)
		result, err := eval(line)
		if err != nil {
			log.Println(err)
		} else {
			fmt.Println(result)
		}
	}
}

// inputValues holds the fully-parsed input.
type inputValues struct {
	def    []tokValue
	in     []tokValue
	sel    []tokValue
	format []tokValue
	limit  []tokValue
	help   []tokValue
}

// get returns the tokValue slice that tokKeyword 'k' maps to.
func (i inputValues) get(k tokKeyword) ([]tokValue, error) {
	switch k {
	case keyDef:
		return i.def, nil
	case keyIn:
		return i.in, nil
	case keySel:
		return i.sel, nil
	case keyFormat:
		return i.format, nil
	case keyLimit:
		return i.limit, nil
	case keyHelp:
		return i.help, nil
	default:
		return nil, tokKeywordError{k}
	}
}

// append appends 'vs' to the tokValue slice that tokKeyword 'k' maps to.
func (i *inputValues) append(k tokKeyword, vs ...tokValue) error {
	switch k {
	case keyDef:
		i.def = append(i.def, vs...)
	case keyIn:
		i.in = append(i.in, vs...)
	case keySel:
		i.sel = append(i.sel, vs...)
	case keyFormat:
		i.format = append(i.format, vs...)
	case keyLimit:
		i.limit = append(i.limit, vs...)
	case keyHelp:
		i.help = append(i.help, vs...)
	default:
		return tokKeywordError{k}
	}
	return nil
}

// validate returns nil if 'i' is valid for the tokKeyword 'k'.
// Otherwise, validate returns an error.
func (i inputValues) validate(k tokKeyword) error {
	info := keywordInfoMap[k]
	if info.validVals == nil {
		return nil
	}
	s, err := i.get(k)
	if err != nil {
		return err
	}
	for _, val := range s {
		var valid bool
		for _, v := range info.validVals {
			if val == v {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("%s is not a valid value for :%s. The following are valid values for :%s: %v", val, k, k, info.validVals)
		}
	}
	return nil
}

// validateAndSetDefaults validates 'i' and sets default values for
// 'i's empty tokValue slices. validateAndSetDefaults returns 'i'.
func (i *inputValues) validateAndSetDefaults() (*inputValues, error) {
	for keyword, info := range keywordInfoMap {
		if err := i.validate(keyword); err != nil {
			return nil, err
		}
		// Skip error check on get because we already know
		// that keyword is a valid tokKeyword.
		if s, _ := i.get(keyword); len(s) == 0 {
			i.append(keyword, info.defaultVals...)
		}
	}
	return i, nil
}

// A token is a structure that a lexer can emit.
type token interface {
	isToken()
}

// tokError represents a lexing error.
type tokError struct {
	msg   string
	start int
	pos   int
}

func (e tokError) isToken() {}

func (e tokError) Error() string { return fmt.Sprintf("%d:%d: %s", e.start, e.pos-e.start, e.msg) }

// tokEOF is emitted by the lexer when it is out of input.
type tokEOF struct{}

func (e tokEOF) isToken() {}

// A tokKeyword is a keyword for the 'src i' lanugage. Keywords always
// start with ":". Do not cast strings to tokKeywords. Always use
// 'toTokKeyword'.
type tokKeyword string

var (
	keyDef    tokKeyword = "def"
	keyIn     tokKeyword = "in"
	keySel    tokKeyword = "select"
	keyFormat tokKeyword = "format"
	keyLimit  tokKeyword = "limit"
	keyHelp   tokKeyword = "help"
)

// keywordInfo holds a keyword's meta information.
type keywordInfo struct {
	// If validVals is nil, then the keyword is not restricted to
	// any specific values. Otherwise, the keyword is only valid
	// if its values match one of validVals.
	validVals []tokValue
	// defaultVals are the default values for a keyword. They are
	// only set if the user does not specify values for the
	// keyword. If defaultVals is nil, then the keyword has no
	// default vals.
	defaultVals []tokValue
	// typeConstraint constrains the keyword to a type.
	// typeConstraint can be "int" or the empty string. If
	// typeConstraint is non-empty, then validVals must be nil.
	typeConstraint string
}

// keywordInfoMap is a map from tokKeyword to keywordInfo for every
// keyword.
var keywordInfoMap = map[tokKeyword]keywordInfo{
	keyDef: keywordInfo{},
	keyIn:  keywordInfo{},
	keySel: keywordInfo{
		validVals:   []tokValue{"defs", "refs", "docs"},
		defaultVals: []tokValue{"defs", "refs"},
	},
	keyFormat: keywordInfo{
		validVals:   []tokValue{"decl", "methods", "body", "full"},
		defaultVals: []tokValue{"decl", "body"},
	},
	keyLimit: keywordInfo{
		typeConstraint: "int",
	},
	keyHelp: keywordInfo{},
}

func (k tokKeyword) isToken() {}

// tokKeywordError represents a validation error for a keyword.
type tokKeywordError struct {
	k tokKeyword
}

func (e tokKeywordError) Error() string {
	if e.k == "" {
		return "invalid keyword: keyword is empty"
	}
	return fmt.Sprintf("unknown keyword: %s", e.k)
}

// toTokKeyword returns a keyword for 's'. It does not check 's's
// validity.
func toTokKeyword(s string) tokKeyword {
	return tokKeyword(strings.ToLower(s))
}

// A tokValue is any non-keyword value.
type tokValue string

func (v tokValue) isToken() {}

// These character runs are used by the lexer to identify terms.
const (
	horizontalWhitespace = " \t"
	alpha                = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	num                  = "0123456789"
	symbol               = "-()*_\\"
	wordChar             = alpha + num + symbol
)

// lexer holds the state of the lexer.
type lexer struct {
	input  string     // String being scanned.
	start  int        // Start position of token.
	pos    int        // Current position of input.
	width  int        // Width of last rune read.
	tokens chan token // Channel of scanned tokens.
}

const eof = -1

// TODO(samer): This leaks channels/lexers if the parsing step errors out.
func (l *lexer) run() {
	l.input = strings.TrimSpace(l.input)
	for state := lexStart; state != nil; {
		state = state(l)
	}
	l.emitEOF()
}

// next returns the next rune and steps 'width' forward.
func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, s := utf8.DecodeRuneInString(l.input[l.pos:])
	if r == utf8.RuneError && s == 1 {
		log.Fatal("input error")
	}
	l.width = s
	l.pos += l.width
	return r
}

// backup can only be called once after each call to 'next'.
func (l *lexer) backup() {
	l.pos -= l.width
}

// accept returns true and moves forward if the next rune is in
// 'valid'.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) != -1 {
		return true
	}
	l.backup()
	return false
}

// acceptRun moves forward for all runes that match 'valid'.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) != -1 {
	}
	l.backup()
}

// acceptRunAllBut moves forward for all runes that do not match
// 'invalid'.
func (l *lexer) acceptRunAllBut(invalid string) {
	n := l.next()
	for n != eof && strings.IndexRune(invalid, n) == -1 {
		n = l.next()
	}
	l.backup()
}

// peek returns the next rune without moving forward.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// ignore ignores the text that the lexer has eaten.
func (l *lexer) ignore() {
	l.start = l.pos
}

// emitKeyword emits the text the lexer has eaten as a tokKeyword.
func (l *lexer) emitKeyword() {
	v := l.input[l.start:l.pos]
	l.start = l.pos
	l.tokens <- tokKeyword(v)
}

// emitValue emits the text the lexer has eaten as a tokValue. If
// 'quoted' is true, then all backslash escape sequences are replaced
// with the literal of the escaped char. If 'quoted' is false, then
// the value's whitespace is trimmed on both sides.
func (l *lexer) emitValue(quoted bool) {
	v := l.input[l.start:l.pos]
	l.start = l.pos
	if quoted {
		for i := 0; i < len(v); i++ {
			if v[i] == '\\' {
				if i == len(v)-1 {
					log.Println("emitValue: '\\' cannot be last character in quoted string. Please file a bug report.")
				} else {
					v = v[:i] + v[i+1:]
				}
				i++ // Don't process the escaped char.
			}
		}
	} else {
		v = strings.TrimSpace(v)
	}
	l.tokens <- tokValue(v)
}

// emitErrorf emits a formatted tokError.
func (l *lexer) emitErrorf(format string, a ...interface{}) {
	l.tokens <- tokError{msg: fmt.Sprintf(format, a), start: l.start, pos: l.pos}
}

// emitError emits a tokError.
func (l *lexer) emitError(a ...interface{}) {
	l.tokens <- tokError{msg: fmt.Sprint(a), start: l.start, pos: l.pos}
}

// emitEOF emits a tokEOF.
func (l *lexer) emitEOF() {
	l.tokens <- tokEOF{}
}

// A stateFn is a function that represents one of the lexer's states.
type stateFn func(l *lexer) stateFn

// lexStart is the entrypoint for the lexer.
func lexStart(l *lexer) stateFn {
	// if GlobalOpt.Verbose {
	// 	log.Printf("lexStart: on %s", string(l.peek()))
	// }
	l.acceptRun(horizontalWhitespace)
	l.ignore()
	if l.peek() == eof {
		return nil
	}
	if l.accept(":") {
		l.ignore()
		return lexKeyword
	}
	if l.accept("\"") {
		l.ignore()
		return lexDoubleQuote
	}
	if l.accept("'") {
		l.ignore()
		return lexSingleQuote
	}
	if l.accept(wordChar) {
		return lexValue
	}
	l.emitErrorf("unexpected char: '%s'", l.next())
	return nil
}

func lexKeyword(l *lexer) stateFn {
	// if GlobalOpt.Verbose {
	// 	log.Printf("lexKeyword: on %s", string(l.peek()))
	// }
	l.acceptRun(alpha)
	l.emitKeyword()
	return lexStart
}

func lexDoubleQuote(l *lexer) stateFn {
	// if GlobalOpt.Verbose {
	// 	log.Printf("lexDoubleQuote: on %s", string(l.peek()))
	// }
	return lexQuote(l, lexDoubleQuote, '"')
}

func lexSingleQuote(l *lexer) stateFn {
	// if GlobalOpt.Verbose {
	// 	log.Printf("lexSingleQuote: on %s", string(l.peek()))
	// }
	return lexQuote(l, lexSingleQuote, '\'')
}

func lexQuote(l *lexer, fromFn stateFn, quote rune) stateFn {
	// if GlobalOpt.Verbose {
	// 	log.Printf("lexQuote: on %s", string(l.peek()))
	// }
	l.acceptRunAllBut(string(quote) + "\\")
	n := l.next()
	switch n {
	case eof:
		// TODO(samer): Continue input.
		l.emitError("unexpected eof")
		return nil
	case '\\':
		l.next() // eat next char
	case quote:
		l.backup()
		l.emitValue(true)
		l.next()
		l.ignore() // ignore quote
		return lexStart
	}
	l.emitErrorf("unexpected char: '%s'", string(n))
	return nil
}

func lexValue(l *lexer) stateFn {
	// if GlobalOpt.Verbose {
	// 	log.Printf("lexValue: on %s", string(l.peek()))
	// }
	l.acceptRunAllBut(":,")
	if l.accept(":") {
		l.backup()
		l.emitValue(false)
		return lexStart
	}
	if l.accept(",") {
		l.backup()
		l.emitValue(false)
		l.next()
		l.ignore() // ignore ','
		return lexValue
	}
	if l.peek() == eof {
		l.emitValue(false)
		return nil
	}
	panic("unreachable")
	return nil
}

// parse parses the user input and organizes it into inputValues.
func parse(input string) (*inputValues, error) {
	if GlobalOpt.Verbose {
		log.Printf("parsing: input %s\n", input)
	}
	l := &lexer{
		input:  input,
		tokens: make(chan token),
	}
	go l.run()
	// Create the inputValues from the input tokens.
	i := &inputValues{}
	type parseState int
	var on tokKeyword
	// invariant:
	//  - 'on' is empty before the loop starts, and is set on the
	//  first successful iteration of theloop.
loop:
	for t := range l.tokens {
		switch t := t.(type) {
		case tokEOF:
			if GlobalOpt.Verbose {
				log.Printf("parsing: got tokEOF\n")
			}
			break loop
		case tokError:
			if GlobalOpt.Verbose {
				log.Printf("parsing: got tokError %s\n", t)
			}
			return nil, t
		case tokKeyword:
			if GlobalOpt.Verbose {
				log.Printf("parsing: got tokKeyword %s\n", t)
			}
			// if we see a tokKeyword, check that the input
			// for the previously seen tokKeyword (stored as
			// 'on') is valid.
			if on != "" {
				if err := i.validate(on); err != nil {
					return nil, err
				}
			}
			s, err := i.get(t)
			if err != nil {
				return nil, err
			}
			if len(s) > 0 {
				return nil, fmt.Errorf("error: keyword :%s can only appear once.", t)
			}
			// Set 'on' to the new tokKeyword.
			on = t
		case tokValue:
			if GlobalOpt.Verbose {
				log.Printf("parsing: got tokValue %s\n", t)
			}
			// If we haven't seen a tokKeyword ('on' is
			// empty), then we're implicitly on keyDef.
			if on == "" {
				on = keyDef
			}
			i.append(on, t)
		default:
			panic("unknown concrete type for token: " + reflect.TypeOf(t).Name())
		}
	}
	return i.validateAndSetDefaults()
}

func inputToFormat(i *inputValues) format {
	var f format
	for _, s := range i.sel {
		switch s {
		case "defs":
			f.showDefs = true
		case "refs":
			f.showRefs = true
		case "docs":
			f.showDocs = true
		}
	}
	for _, fmt := range i.format {
		switch fmt {
		case "decl":
			f.showDefDecl = true
		case "methods":
			f.showDefMethods = true
		case "body":
			f.showDefBody = true
		case "full":
			f.showDefFull = true
		}
	}
	// TODO: make limit parsing more robust.
	if len(i.limit) == 1 {
		l, err := strconv.Atoi(string(i.limit[0]))
		if err != nil {
			log.Printf("Could not convert limit %s to an int, skipping.\n", i.limit[0])
		}
		f.limit = l
	}
	return f
}

type format struct {
	showDefs    bool
	showRefs    bool
	showDocs    bool
	showDefDecl bool
	showDefBody bool
	limit       int
	// The following are unimplemented:
	showDefMethods bool
	showDefFull    bool
}

// eval evaluates input and returns the results as 'output'.
func eval(input string) (output string, err error) {
	i, err := parse(input)
	if err != nil {
		return "", err
	}
	if GlobalOpt.Verbose {
		log.Printf("parsed: %+v\n", i)
	}
	f := inputToFormat(i)
	var out []string
	for _, input := range i.def {
		c := &StoreDefsCmd{
			Query:    string(input),
			CommitID: activeContext.repo.CommitID,
			Limit:    f.limit,
		}
		defs, err := c.Get()
		if err != nil {
			return "", err
		}
		if f.showRefs {
			outDefRefs := make([]defRefs, 0, len(defs))
			for _, d := range defs {
				c := &StoreRefsCmd{
					DefRepo:     d.Repo,
					DefUnitType: d.UnitType,
					DefUnit:     d.Unit,
					DefPath:     d.Path,
				}
				refs, err := c.Get()
				if err != nil {
					return "", err
				}
				outDefRefs = append(outDefRefs, defRefs{d, refs})
			}
			out = append(out, formatObject(outDefRefs, f))
			continue
		}
		out = append(out, formatObject(defs, f))
	}
	return strings.Join(out, "\n"), nil
}

type byDefKind struct {
	kind string
}

func (h byDefKind) SelectDef(def *graph.Def) bool {
	return def.Kind == "" || def.Kind == h.kind
}

type defRefs struct {
	def  *graph.Def
	refs []*graph.Ref
}

func getFileSegment(file string, start, end uint32, header bool) string {
	f, err := ioutil.ReadFile(file)
	if err != nil {
		return ""
	}
	if header {
		startLine := bytes.Count(f[:start], []byte{'\n'}) + 1
		// Roll 'start' back and 'end' forward to the nearest
		// newline.
		for ; start-1 > 0 && f[start-1] != '\n'; start-- {
		}
		for ; end < uint32(len(f)) && f[end] != '\n'; end++ {
		}
		var out []string
		onLine := startLine
		for _, line := range bytes.Split(f[start:end], []byte{'\n'}) {
			var marker string
			if startLine == onLine {
				marker = ":"
			} else {
				marker = "-"
			}
			out = append(out, fmt.Sprintf("%s:%d%s%s",
				file, onLine, marker, string(line),
			))
			onLine++
		}
		return strings.Join(out, "\n")
	}
	return string(f[start:end])
}

func formatObject(objs interface{}, f format) string {
	switch o := objs.(type) {
	case *graph.Def:
		if o == nil {
			return "def is nil"
		}
		var output []string
		if f.showDefs {
			output = append(output, "---------- def ----------")
			if f.showDefDecl {
				b, err := json.Marshal(o)
				if err != nil {
					return fmt.Sprintf("error unmarshalling: %s", err)
				}
				c := &FmtCmd{
					UnitType:   o.UnitType,
					ObjectType: "def",
					Format:     "decl",
					Object:     string(b),
				}
				out, err := c.Get()
				if err != nil {
					return fmt.Sprintf("error formatting def: %s", err)
				}
				output = append(output, out)
			}
			if f.showDefBody {
				output = append(output, getFileSegment(o.File, o.DefStart, o.DefEnd, true))
			}
		}
		if f.showDocs {
			var data string
			for _, doc := range o.Docs {
				if doc.Format == "text/plain" {
					data = doc.Data
					break
				}
			}
			if data != "" {
				output = append(output, "---------- doc ----------", data)
			}
		}
		return strings.Join(output, "\n")
	case []*graph.Def:
		var out []string
		for _, d := range o {
			out = append(out, formatObject(d, f))
		}
		return strings.Join(out, "\n")
	case *graph.Ref:
		if o == nil {
			return "ref is nil"
		}
		if f.showRefs {
			return getFileSegment(o.File, o.Start, o.End, true)
		}
		return ""
	case []*graph.Ref:
		var out []string
		for _, r := range o {
			if !r.Def {
				out = append(out, formatObject(r, f))
			}
		}
		return strings.Join(out, "\n")
	case []defRefs:
		var out []string
		for _, d := range o {
			out = append(out, formatObject(d, f))
		}
		return strings.Join(out, "\n")
	case defRefs:
		var out []string
		out = append(out,
			formatObject(o.def, f),
			"---------- refs ----------",
			formatObject(o.refs, f),
		)
		return strings.Join(out, "\n")
	default:
		log.Printf("formatObject: no output for %#v\n", o)
		return ""
	}
}

// from google/cayley
func terminal(path string) (*liner.State, error) {
	term := liner.NewLiner()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
		err := persist(term, historyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to properly clean up terminal: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return term, nil
		}
		return term, err
	}
	defer f.Close()
	_, err = term.ReadHistory(f)
	return term, err
}

// from google/cayley
func persist(term *liner.State, path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("could not open %q to append history: %v", path, err)
	}
	defer f.Close()
	_, err = term.WriteHistory(f)
	if err != nil {
		return fmt.Errorf("could not write history to %q: %v", path, err)
	}
	return term.Close()
}
