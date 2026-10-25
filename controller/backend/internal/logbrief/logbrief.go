// Package logbrief picks the key messages out of a test log: test results,
// errors and key NF events, so a failure can be understood without reading
// the whole log.
package logbrief

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/model"
)

const (
	CategoryTest  = "test"
	CategoryError = "error"
	CategoryEvent = "event"

	maxEntries = 200
	maxItems   = 50
)

var (
	ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

	// Compose log: "nrf    | 2026-10-25T06:27:39.512046137Z [INFO][NRF][NFM] NF suspended, ..."
	composeLine = regexp.MustCompile(`^([a-z0-9][a-z0-9_.-]*)\s+\| (.*)$`)
	nfLine      = regexp.MustCompile(`^(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z) \[([A-Z]{4})\]((?:\[[^\]]*\])*) ?(.*)$`)
	bracketTag  = regexp.MustCompile(`\[[^\]]*\]`)

	// Docker build output: "#12 3.456 ..."
	buildkitPrefix = regexp.MustCompile(`^#\d+ \d+\.\d+ `)

	testResult  = regexp.MustCompile(`^--- (FAIL|PASS): \S+`)
	goPanic     = regexp.MustCompile(`^panic: `)
	errorTrace  = regexp.MustCompile(`^\s*Error Trace:\s*(.*)$`)
	assertField = regexp.MustCompile(`^\s*(Error|expected|actual|Test|Messages)\s*:\s*(.*)$`)
	scriptError = regexp.MustCompile(`^Error: `)
	exitCode    = regexp.MustCompile(`^Test completed with exit code: [1-9]`)
	buildError  = regexp.MustCompile(`^(ERROR[: \[]|.*failed to solve|make: \*\*\* |[\w./-]+\.go:\d+:\d+: )`)
	gitFatal    = regexp.MustCompile(`^fatal: `)
	ginPanic    = regexp.MustCompile(`^\[Recovery\].*panic recovered`)

	// NFs retry registering until NRF listens, so this is expected at startup
	startupNoise = regexp.MustCompile(`register to NRF Error.*connection refused`)

	uuidPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	ipPattern   = regexp.MustCompile(`\b\d{1,3}(\.\d{1,3}){3}(:\d+)?\b`)
)

// NF log lines worth showing although they are not errors. A capture group
// marks the part that differs between lines merged into one entry.
var nfEvents = []*regexp.Regexp{
	regexp.MustCompile(`^NF suspended, no heart-beat received: (\w+)`),
	regexp.MustCompile(`^Send Registration Reject`),
	regexp.MustCompile(`^Handle event\[Authentication Error\]`),
}

type builder struct {
	entries []model.LogBriefEntry
	times   []time.Time
	index   map[string]int
}

// Summarize returns the key messages of log, or nil if there are none.
// Test results come first, then errors and events of NFs in time order,
// then other errors in line order.
func Summarize(log string) []model.LogBriefEntry {
	lines := strings.Split(strings.ReplaceAll(log, "\r\n", "\n"), "\n")
	b := &builder{index: map[string]int{}}

	for i := 0; i < len(lines); i++ {
		number := i + 1
		text := clean(lines[i])

		source := ""
		if m := composeLine.FindStringSubmatch(text); m != nil {
			source, text = m[1], m[2]
		}

		if m := nfLine.FindStringSubmatch(text); m != nil {
			b.addNfLine(m[1], m[2], m[3], m[4], number, source)
			continue
		}

		switch {
		case errorTrace.MatchString(text):
			message, end := readAssertion(lines, i)
			b.add(CategoryTest, message, number, source, "", "")
			i = end
		case goPanic.MatchString(text):
			category := CategoryTest
			if source != "" {
				category = CategoryError
			}
			b.add(category, text, number, source, "", "")
		case ginPanic.MatchString(text):
			b.add(CategoryError, text, number, source, "", "")
		case source != "":
			// Only NF output comes with a source; the rules below are for
			// the test, script and build output around it
		case testResult.MatchString(text):
			b.add(CategoryTest, text, number, "", "", "")
		case scriptError.MatchString(text), exitCode.MatchString(text), gitFatal.MatchString(text):
			b.add(CategoryError, text, number, "", "", "")
		default:
			if build := buildkitPrefix.ReplaceAllString(text, ""); buildError.MatchString(build) {
				b.add(CategoryError, build, number, "", "", "")
			}
		}
	}

	return b.sorted()
}

func clean(line string) string {
	return strings.TrimRight(ansiEscape.ReplaceAllString(line, ""), " \t\r")
}

func (b *builder) addNfLine(timestamp, level, tags, message string, line int, source string) {
	if startupNoise.MatchString(message) {
		return
	}

	// Keep the NF and component tags, e.g. [AMF][Gmm], and drop per-UE context
	prefix := strings.Join(firstN(bracketTag.FindAllString(tags, -1), 2), "") + " "

	switch level {
	case "ERRO", "FATA", "PANI":
		b.add(CategoryError, prefix+message, line, source, timestamp, "")
		return
	}

	for _, event := range nfEvents {
		m := event.FindStringSubmatch(message)
		if m == nil {
			continue
		}
		base, item := m[0], ""
		if len(m) > 1 {
			item = m[1]
			base = strings.TrimRight(strings.TrimSuffix(m[0], m[1]), " :")
		}
		b.add(CategoryEvent, prefix+base, line, source, timestamp, item)
		return
	}
}

// readAssertion reads the testify failure starting with "Error Trace:" at
// lines[start] and returns it as one message, with the index of its last line.
func readAssertion(lines []string, start int) (string, int) {
	trace := strings.TrimSpace(errorTrace.FindStringSubmatch(clean(lines[start]))[1])
	fields := map[string]string{}
	end := start
	for i := start + 1; i < len(lines); i++ {
		m := assertField.FindStringSubmatch(clean(lines[i]))
		if m == nil {
			break
		}
		if _, ok := fields[m[1]]; !ok {
			fields[m[1]] = strings.TrimSpace(m[2])
		}
		end = i
		if m[1] == "Messages" {
			break
		}
	}

	message := strings.TrimSuffix(fields["Error"], ":")
	if message == "" {
		message = "Assertion failed"
	}
	if fields["expected"] != "" || fields["actual"] != "" {
		message += fmt.Sprintf(" (expected: %s, actual: %s)", fields["expected"], fields["actual"])
	}
	if fields["Messages"] != "" {
		message += ": " + fields["Messages"]
	}
	if trace != "" {
		message += " [" + path.Base(trace) + "]"
	}

	return message, end
}

func (b *builder) add(category, message string, line int, source, timestamp, item string) {
	key := category + "\x00" + source + "\x00" + ipPattern.ReplaceAllString(uuidPattern.ReplaceAllString(message, "<uuid>"), "<ip>")
	if i, ok := b.index[key]; ok {
		entry := &b.entries[i]
		entry.Count++
		if item != "" && len(entry.Items) < maxItems && !slices.Contains(entry.Items, item) {
			entry.Items = append(entry.Items, item)
		}
		return
	}
	if len(b.entries) >= maxEntries {
		return
	}

	entry := model.LogBriefEntry{
		Category: category,
		Message:  message,
		Line:     line,
		Count:    1,
		Source:   source,
		Time:     timestamp,
	}
	if item != "" {
		entry.Items = []string{item}
	}
	parsed, _ := time.Parse(time.RFC3339Nano, timestamp)

	b.index[key] = len(b.entries)
	b.entries = append(b.entries, entry)
	b.times = append(b.times, parsed)
}

func (b *builder) sorted() []model.LogBriefEntry {
	if len(b.entries) == 0 {
		return nil
	}

	order := make([]int, len(b.entries))
	for i := range order {
		order[i] = i
	}
	rank := func(i int) int {
		switch {
		case b.entries[i].Category == CategoryTest:
			return 0
		case b.entries[i].Time != "":
			return 1
		default:
			return 2
		}
	}
	sort.SliceStable(order, func(x, y int) bool {
		i, j := order[x], order[y]
		if rank(i) != rank(j) {
			return rank(i) < rank(j)
		}
		if rank(i) == 1 && !b.times[i].Equal(b.times[j]) {
			return b.times[i].Before(b.times[j])
		}
		return b.entries[i].Line < b.entries[j].Line
	})

	entries := make([]model.LogBriefEntry, len(order))
	for x, i := range order {
		entries[x] = b.entries[i]
	}
	return entries
}

func firstN(values []string, n int) []string {
	if len(values) > n {
		return values[:n]
	}
	return values
}
