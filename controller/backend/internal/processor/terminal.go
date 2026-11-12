package processor

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"

	"github.com/Alonza0314/it-system/controller/backend/constant"
)

// RunnerExists reports whether a runner with this name is registered.
func (p *Processor) RunnerExists(name string) bool {
	return p.itContext.RunnerExists(name)
}

// RunnerTokenCurrent reports whether registrationID (the "rid" claim of a
// runner token) belongs to the runner's current registration. Tokens issued
// before registration ids existed, or before the runner was deleted or
// registered again, do not.
func (p *Processor) RunnerTokenCurrent(name, registrationID string) bool {
	if registrationID == "" || !p.itContext.RunnerExists(name) {
		return false
	}

	stored, err := p.itContext.LoadFromDb(constant.BUCKET_RUNNER_REGISTRATION, name)
	if err != nil || stored == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(stored), []byte(registrationID)) == 1
}

func newRegistrationID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// UserIsAdmin reports whether a login as user would still get the admin
// level, following the same rules as Login. The terminal calls it again
// while a session is open, so deleting a tenant or taking away its admin
// role ends that tenant's sessions.
func (p *Processor) UserIsAdmin(user string) bool {
	if user == "" {
		return false
	}

	if user == p.username {
		return user == constant.USER_LEVEL_ADMIN
	}

	exists, err := p.itContext.ExistsInDb(constant.BUCKET_TENANT, user)
	if err != nil || !exists {
		return false
	}

	if user == constant.USER_LEVEL_ADMIN {
		return true
	}

	role, err := p.itContext.LoadFromDb(constant.BUCKET_TENANT, user)
	if err != nil {
		return false
	}

	return role == constant.USER_LEVEL_ADMIN
}
