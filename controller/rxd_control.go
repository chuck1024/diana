/**
 * Copyright 2018 Author. All rights reserved.
 * Author: Chuck1024
 */

package controller

import (
	"diana/model"
	"diana/service"
	"github.com/chuck1024/godog"
	de "github.com/chuck1024/godog/error"
	"github.com/chuck1024/godog/net/httplib"
	"net/http"
	"time"
)

// TODO: of course, http is one method, you choose kafka etc.
// only support http post
func RxdControl(rsp http.ResponseWriter, req *http.Request) {
	rsp.Header().Add("Access-Control-Allow-Origin", httplib.CONTENT_ALL)
	rsp.Header().Add("Content-Type", httplib.CONTENT_JSON)

	if req.Method == http.MethodOptions {
		rsp.WriteHeader(http.StatusOK)
		return
	} else if req.Method != http.MethodPost {
		rsp.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var dogErr *de.CodeError
	request := &model.RxdReq{}
	response := &model.RxdRsp{}
	currentTs := time.Now().UnixNano()

	defer func() {
		if dogErr != nil {
			godog.Error("[RxdControl], errorCode: %d, errMsg: %s", dogErr.Code(), dogErr.Detail())
		}
		rsp.Write(httplib.LogGetResponseInfo(req, dogErr, response))
	}()

	err := httplib.GetRequestBody(req, &request)
	if err != nil {
		dogErr = de.MakeCodeError(de.ParameterError, err)
		return
	}

	godog.Info("[RxdControl] received request: %v", *request)

	if err = service.Rxd(request, currentTs); err != nil {
		godog.Error("[RxdControl] service.Rxd occur error: %s", err)
		dogErr = de.MakeCodeError(de.SystemError, err)
		return
	}

	return
}
