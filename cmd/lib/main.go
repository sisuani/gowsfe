package main

import "C"

import (
	"encoding/json"
	"log"
	//"strconv"
	//"time"

	"gowsfe/pkg/afip/wsafip"
	"gowsfe/pkg/afip/wsfe"
)

var lastError string
var wsafipService *wsafip.Service
var wsfeService *wsfe.Service

//export CreateWSFEService
func CreateWSFEService(certsPath string, cuit int64) bool {
	crt := certsPath + "/" + "cert.crt"
	key := certsPath + "/" + "cert.key"

	wsafipService = wsafip.NewService(wsafip.TESTING, crt, key)
	var err error
	token, sign, _, err := wsafipService.GetLoginTicket("wsfe")
	if err != nil {
		lastError = err.Error()
		return false
	}

	wsfeService = wsfe.NewService(wsfe.TESTING, token, sign)
	return true
}

//export GetUltimoComp
func GetUltimoComp(requestStrCchar *C.char) int64 {
	lastError = ""
	requestStr := C.GoString(requestStrCchar)
	cabRequest := wsfe.CabRequest{}
	err := json.Unmarshal([]byte(requestStr), &cabRequest)
	cbteNro, err := wsfeService.GetUltimoComp(&cabRequest)
	if err != nil {
		lastError = err.Error()
		return 0
	}
	return int64(cbteNro)
}

//export CaeRequest
func CaeRequest(cabRequestCchar, detRequestCchar *C.char) (*C.char, *C.char) {
	lastError = ""
	cabRequestStr := C.GoString(cabRequestCchar)
	detRequestStr := C.GoString(detRequestCchar)
	log.Println("CAB: ", cabRequestStr)
	log.Println("DET: ", detRequestStr)

	cabRequest := wsfe.CabRequest{}
	err := json.Unmarshal([]byte(cabRequestStr), &cabRequest)
	if err != nil {
		lastError = err.Error()
		return C.CString(""), C.CString("")
	}

	caeRequest := wsfe.CaeRequest{}
	err = json.Unmarshal([]byte(detRequestStr), &caeRequest)
	if err != nil {
		lastError = err.Error()
		return C.CString(""), C.CString("")
	}

	cae, caeFchVto, err := wsfeService.CaeRequest(&cabRequest, &caeRequest)
	if err != nil {
		lastError = err.Error()
	}
	return C.CString(cae), C.CString(caeFchVto)
}

//export LastError
func LastError() *C.char {
	return C.CString(lastError)
}
