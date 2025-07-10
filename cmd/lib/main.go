package main

import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sisuani/gowsfe/pkg/afip/wsafip"
	"github.com/sisuani/gowsfe/pkg/afip/wsfe"
)

var lastError string
var wsafipService *wsafip.Service
var wsfeService *wsfe.Service

func writeToLog(message string) error {
	file, err := os.OpenFile("gowsfe.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir/crear el archivo: %v", err)
	}
	defer file.Close()

	logEntry := fmt.Sprintf("[%s] %s\r\n", time.Now().Format("2006-01-02 15:04:05"), message)

	if _, err := file.WriteString(logEntry); err != nil {
		return fmt.Errorf("error al escribir en el archivo: %v", err)
	}

	return nil
}

//export CreateWSFEService
func CreateWSFEService(certsPath string, cuit int64) bool {
	crt := certsPath + "/" + "cert.crt"
	key := certsPath + "/" + "cert.key"

	writeToLog("CreateWSFEService()")
	writeToLog(fmt.Sprintf("  |_ crt: %s", crt))
	writeToLog(fmt.Sprintf("  |_ key: %s", key))

	wsafipService = wsafip.NewService(wsafip.PRODUCTION, crt, key)
	var err error
	token, sign, _, err := wsafipService.GetLoginTicket("wsfe")
	if err != nil {
		lastError = err.Error()
		return false
	}

	wsfeService = wsfe.NewService(wsfe.PRODUCTION, token, sign)
	return true
}

//export GetUltimoComp
func GetUltimoComp(requestStrCchar *C.char) int64 {
	lastError = ""
	requestStr := C.GoString(requestStrCchar)

	writeToLog("GetUltimoComp()")
	writeToLog(fmt.Sprintf("  |_ Request: %s", requestStr))

	cabRequest := wsfe.CabRequest{}
	err := json.Unmarshal([]byte(requestStr), &cabRequest)
	cbteNro, err := wsfeService.GetUltimoComp(&cabRequest)
	writeToLog(fmt.Sprintf("  |_ Ultimo Comprobante (AFIP): %d", cbteNro))
	if err != nil {
		lastError = err.Error()
		cbteNro = -1
	}
	writeToLog(fmt.Sprintf("  |_ Ultimo Comprobante: %d", cbteNro))
	return int64(cbteNro)
}

//export CaeRequest
func CaeRequest(cabRequestCchar, detRequestCchar *C.char) (*C.char, *C.char) {
	lastError = ""
	cabRequestStr := C.GoString(cabRequestCchar)
	detRequestStr := C.GoString(detRequestCchar)

	writeToLog("CaeRequest()")
	writeToLog(fmt.Sprintf("  |_ CAB: %s", cabRequestStr))
	writeToLog(fmt.Sprintf("  |_ DET: %s", detRequestStr))

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
		writeToLog(fmt.Sprintf("  |_ error: %s", lastError))
		return C.CString(""), C.CString("")
	}

	cae, caeFchVto, err := wsfeService.CaeRequest(&cabRequest, &caeRequest)
	if err != nil {
		lastError = err.Error()
		writeToLog(fmt.Sprintf("  |_ error: %s", lastError))
	}

	writeToLog(fmt.Sprintf("  |_ cae: %s", cae))
	writeToLog(fmt.Sprintf("  |_ vto: %s", caeFchVto))
	return C.CString(cae), C.CString(caeFchVto)
}

//export LastError
func LastError() *C.char {
	return C.CString(lastError)
}

func main() {
	ret := CreateWSFEService("certs", 20285142084)
	if !ret {
		log.Println(lastError)
		return
	}

	request := C.CString(`{"cbteTipo":1,"cuit":20285142084,"pos":6}`)
	nroUltimoComp := GetUltimoComp(request)
	log.Println(nroUltimoComp)

	/*
		pos := int32(6)

		nroUltimoComp := GetUltimoComp(20285142084, pos, CBTE_TIPO_CN_A)
		log.Println(nroUltimoComp)

		for i := 1; i < 100; i++ {
			nroUltimoComp := GetUltimoComp(20285142084, pos, CBTE_TIPO_CN_A)
			today := time.Now().Format("20060102")

			ivasMap := make(map[string]float64)
			ivasMap["210"] = 0.21
			cae, caeFchVto := CaeRequest(20285142084, pos, CBTE_TIPO_CN_A, 80, 20277342562, nroUltimoComp+1, nroUltimoComp+1, today, 2.21, 0, 1, 0, 1, 0.21, ivasMap, CBTE_TIPO_INV_A, 36)

			log.Println(nroUltimoComp, cae, caeFchVto, lastError)

			time.Sleep(time.Minute * 60)
		}
	*/
}
