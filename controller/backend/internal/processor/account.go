package processor

import (
	"fmt"
	"net/http"

	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/model"

	"github.com/free-ran-ue/util"
)

func (p *Processor) Login(req *model.RequestLogin) (*model.ResponseLogin, *model.ErrorDetail) {
	p.ProcLog.Debugf("Processing login for username: %s", req.Username)

	exist := false
	if req.Username == p.username {
		if req.Password == p.password {
			exist = true
		}
	} else {
		var err error
		exist, err = p.itContext.ExistsInDb(constant.BUCKET_TENANT, req.Username)
		if err != nil {
			p.ProcLog.Errorf("Failed to check if tenant %s exists in database: %v", req.Username, err)
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to check if tenant %s exists in database: %v", req.Username, err),
			}
		}

		if req.Username != req.Password {
			exist = false
		}
	}

	if !exist {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusUnauthorized,
			Detail:     "Invalid username or incorrect password",
		}
	}

	claims := map[string]interface{}{
		"user": req.Username,
	}

	role, err := p.itContext.LoadFromDb(constant.BUCKET_TENANT, req.Username)
	if err != nil {
		p.ProcLog.Errorf("Failed to load role for tenant %s from database: %v", req.Username, err)
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to load role for tenant %s from database: %v", req.Username, err),
		}
	}
	if req.Username == constant.USER_LEVEL_ADMIN {
		claims[constant.USER_LEVEL_CLAIM_TAG] = constant.USER_LEVEL_ADMIN
	} else {
		claims[constant.USER_LEVEL_CLAIM_TAG] = role
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
