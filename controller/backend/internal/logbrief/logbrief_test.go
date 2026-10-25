package logbrief_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Alonza0314/it-system/controller/backend/internal/logbrief"
	"github.com/Alonza0314/it-system/controller/backend/model"
)

// Excerpt of a failed IT run: only nrf ran a heart-beat PR, so NRF suspended
// the other NFs and AMF could not select an AUSF
const failedTestLog = "Running test... TestPDUSessionReleaseRequest\n" +
	"=== RUN   TestPDUSessionReleaseRequest\n" +
	"    it_pdusessionreleaserequest_test.go:65: \n" +
	"        \tError Trace:\t/root/test/goTest/it_pdusessionreleaserequest_test.go:65\n" +
	"        \tError:      \tNot equal: \n" +
	"        \t            \texpected: 0x56\n" +
	"        \t            \tactual  : 0x44\n" +
	"        \tTest:       \tTestPDUSessionReleaseRequest\n" +
	"        \tMessages:   \tReceived wrong GMM message. Expected Authentication Request.\n" +
	"--- FAIL: TestPDUSessionReleaseRequest (0.16s)\n" +
	"FAIL\n" +
	"exit status 1\n" +
	"pcf  | 2026-10-25T06:27:14.056380554Z [ERRO][PCF][Consumer] PCF register to NRF Error[Put \"http://nrf.free5gc.org:8000/nnrf-nfm/v1/nf-instances/f9727c08-2787-4a36-8f4f-5997c56cf2df\": dial tcp 10.100.200.3:8000: connect: connection refused]\n" +
	"amf    | 2026-10-25T06:28:05.030575253Z [ERRO][AMF][Gmm][amf_ue_ngap_id:RU:1,AU:1(3GPP)][supi:SUPI:] AMF can not select an AUSF by NRF\n" +
	"amf    | 2026-10-25T06:28:05.030604550Z [INFO][AMF][Gmm][amf_ue_ngap_id:RU:1,AU:1(3GPP)][supi:SUPI:] Send Registration Reject\n" +
	"amf    | 2026-10-25T06:28:05.074173931Z [WARN][AMF][Ngap] RAN context has been removed[addr: <nil>]\n" +
	"nrf    | 2026-10-25T06:27:14.503532766Z [INFO][NRF][SBI] SBI server started\n" +
	"nrf    | 2026-10-25T06:27:39.512046137Z [INFO][NRF][NFM] NF suspended, no heart-beat received: CHF [0a2fa372-5a16-457d-879e-7436aa7fe396]\n" +
	"nrf    | 2026-10-25T06:27:39.5Z [INFO][NRF][NFM] NF suspended, no heart-beat received: AUSF [94c5fc63-a560-4754-84fd-7db59f81f2b8]\n" +
	"nrf    | 2026-10-25T06:27:39.527235692Z [INFO][NRF][NFM] NF suspended, no heart-beat received: AMF [c8315d56-e078-4e77-9430-8d1b67c8ee97]\n" +
	"nrf    | 2026-10-25T06:27:39.528Z [INFO][NRF][NFM] NF suspended, no heart-beat received: AMF [41af18ca-4885-43cf-85fa-b40124068799]\n" +
	"Test completed with exit code: 1\n"

func TestSummarizeFailedTest(t *testing.T) {
	got := logbrief.Summarize(failedTestLog)

	expected := []model.LogBriefEntry{
		{
			Category: logbrief.CategoryTest,
			Message:  "Not equal (expected: 0x56, actual: 0x44): Received wrong GMM message. Expected Authentication Request. [it_pdusessionreleaserequest_test.go:65]",
			Line:     4,
			Count:    1,
		},
		{Category: logbrief.CategoryTest, Message: "--- FAIL: TestPDUSessionReleaseRequest (0.16s)", Line: 10, Count: 1},
		// NF lines come in time order, not in the order containers were dumped
		{
			Category: logbrief.CategoryEvent,
			Message:  "[NRF][NFM] NF suspended, no heart-beat received",
			Line:     18,
			Count:    4,
			Source:   "nrf",
			Time:     "2026-10-25T06:27:39.512046137Z",
			Items:    []string{"CHF", "AUSF", "AMF"},
		},
		{
			Category: logbrief.CategoryError,
			Message:  "[AMF][Gmm] AMF can not select an AUSF by NRF",
			Line:     14,
			Count:    1,
			Source:   "amf",
			Time:     "2026-10-25T06:28:05.030575253Z",
		},
		{
			Category: logbrief.CategoryEvent,
			Message:  "[AMF][Gmm] Send Registration Reject",
			Line:     15,
			Count:    1,
			Source:   "amf",
			Time:     "2026-10-25T06:28:05.030604550Z",
		},
		{Category: logbrief.CategoryError, Message: "Test completed with exit code: 1", Line: 22, Count: 1},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Unexpected brief:\n%s", format(got))
	}
}

func TestSummarizePassedTest(t *testing.T) {
	log := "=== RUN   TestRegistration\n" +
		"--- PASS: TestRegistration (1.44s)\n" +
		"PASS\n" +
		"ok  \ttest\t1.445s\n" +
		"Test completed with exit code: 0\n"

	expected := []model.LogBriefEntry{
		{Category: logbrief.CategoryTest, Message: "--- PASS: TestRegistration (1.44s)", Line: 2, Count: 1},
	}
	if got := logbrief.Summarize(log); !reflect.DeepEqual(got, expected) {
		t.Errorf("Unexpected brief:\n%s", format(got))
	}
}

func TestSummarizeBuildAndFetchErrors(t *testing.T) {
	log := "\x1b[0m\x1b[91mgo: downloading github.com/pkg/errors v0.9.1\r\n" +
		"➤ YN0007: │ esbuild@npm:0.25.0 must be built because it never has been before or the last one failed\n" +
		"#25 41.20 internal/sbi/processor/nf_management.go:120:3: undefined: nfheartbeat\n" +
		"ERROR: failed to solve: process \"/bin/sh -c make -j$(nproc) all\" did not complete successfully: exit code: 2\n" +
		"make: *** [Makefile:20: all-base] Error 1\n" +
		"fatal: couldn't find remote ref pull/9999/head\n"

	expected := []model.LogBriefEntry{
		{Category: logbrief.CategoryError, Message: "internal/sbi/processor/nf_management.go:120:3: undefined: nfheartbeat", Line: 3, Count: 1},
		{Category: logbrief.CategoryError, Message: "ERROR: failed to solve: process \"/bin/sh -c make -j$(nproc) all\" did not complete successfully: exit code: 2", Line: 4, Count: 1},
		{Category: logbrief.CategoryError, Message: "make: *** [Makefile:20: all-base] Error 1", Line: 5, Count: 1},
		{Category: logbrief.CategoryError, Message: "fatal: couldn't find remote ref pull/9999/head", Line: 6, Count: 1},
	}
	if got := logbrief.Summarize(log); !reflect.DeepEqual(got, expected) {
		t.Errorf("Unexpected brief:\n%s", format(got))
	}
}

func TestSummarizeNothing(t *testing.T) {
	if got := logbrief.Summarize("Cloning into 'ci-test'...\nSwitched to branch 'pr-242'\n"); got != nil {
		t.Errorf("Expected no brief, got:\n%s", format(got))
	}
}

func format(entries []model.LogBriefEntry) string {
	b := strings.Builder{}
	for _, e := range entries {
		b.WriteString(strings.Join([]string{e.Category, e.Source, e.Time, e.Message}, " | "))
		b.WriteString("\n")
	}
	return b.String()
}
