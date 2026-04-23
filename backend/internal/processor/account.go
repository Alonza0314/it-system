package processor

import (
	"backend/model"
	"net/http"

	"github.com/free-ran-ue/util"
)

func (p *Processor) Login(req *model.RequestLogin) (*model.ResponseLogin, *model.ErrorDetail) {
	p.ProcLog.Debugf("Processing login for username: %s", req.Username)

	if req.Username != p.username || req.Password != p.password {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusUnauthorized,
			Detail:     "Invalid username or incorrect password",
		}
	}

	token, err := util.CreateJWT(p.jwtSecret, req.Username, p.jwtExpiresIn, nil)
	if err != nil {
		p.ProcLog.Errorf("Failed to create JWT for username %s: %v", req.Username, err)
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     "Failed to create JWT",
		}
	}

	return &model.ResponseLogin{
		Message: "Login successful",
		Token:   token,
	}, nil
}
