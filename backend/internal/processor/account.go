package processor

import (
	"backend/constant"
	"backend/model"
	"fmt"
	"net/http"

	"github.com/free-ran-ue/util"
)

func (p *Processor) Login(req *model.RequestLogin) (*model.ResponseLogin, *model.ErrorDetail) {
	p.ProcLog.Debugf("Processing login for username: %s", req.Username)

	// TODO: Replace with real authentication logic, e.g., check against a database
	if req.Username != p.username || req.Password != p.password {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusUnauthorized,
			Detail:     "Invalid username or incorrect password",
		}
	}

	claims := map[string]interface{}{
		"user": req.Username,
	}
	if req.Username == constant.USER_LEVEL_ADMIN {
		claims[constant.USER_LEVEL_CLAIM_TAG] = constant.USER_LEVEL_ADMIN
	} else {
		claims[constant.USER_LEVEL_CLAIM_TAG] = constant.USER_LEVEL_DEFAULT
	}

	token, err := util.CreateJWT(p.jwtSecret, req.Username, p.jwtExpiresIn, claims)
	if err != nil {
		p.ProcLog.Errorf("Failed to create JWT for username %s: %v", req.Username, err)
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to create JWT for username %s: %v", req.Username, err),
		}
	}

	return &model.ResponseLogin{
		Message: "Login successful",
		Token:   token,
	}, nil
}
