package processor_test

import (
	"testing"

	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/model"

	"github.com/free-ran-ue/util"
)

func registerRunner(t *testing.T, p interface {
	RegisterRunner(*model.RequestRegisterRunner) (*model.ResponseRegisterRunner, *model.ErrorDetail)
}, name string) string {
	t.Helper()

	resp, errDetail := p.RegisterRunner(&model.RequestRegisterRunner{Name: name, IP: "10.8.0.11"})
	if errDetail != nil {
		t.Fatalf("RegisterRunner(%s): %s", name, errDetail.Detail)
	}

	claims, err := util.ValidateJWT(resp.Token, "runner-secret")
	if err != nil {
		t.Fatalf("runner token: %v", err)
	}
	rid, _ := claims[constant.RUNNER_REGISTRATION_CLAIM_TAG].(string)
	if len(rid) != 32 {
		t.Fatalf("registration id claim = %q, want 32 hex characters", rid)
	}

	return rid
}

func TestRunnerTokenCurrent(t *testing.T) {
	p := newTestProcessor(t)

	first := registerRunner(t, p, "runner-1")
	other := registerRunner(t, p, "runner-2")

	if !p.RunnerTokenCurrent("runner-1", first) {
		t.Error("the token of the current registration is refused")
	}
	if p.RunnerTokenCurrent("runner-1", other) {
		t.Error("runner-2's registration id works for runner-1")
	}
	if p.RunnerTokenCurrent("runner-1", "") {
		t.Error("a token without a registration id is accepted")
	}

	if _, errDetail := p.DeleteRunner("runner-1"); errDetail != nil {
		t.Fatal(errDetail.Detail)
	}
	if p.RunnerTokenCurrent("runner-1", first) {
		t.Error("the token of a deleted runner is accepted")
	}

	second := registerRunner(t, p, "runner-1")
	if second == first {
		t.Fatal("registering again reused the registration id")
	}
	if p.RunnerTokenCurrent("runner-1", first) {
		t.Error("the token from before registering again is accepted")
	}
	if !p.RunnerTokenCurrent("runner-1", second) {
		t.Error("the token of the new registration is refused")
	}
}
