package wsfe

import (
	"fmt"
	"strconv"

	"github.com/hooklift/gowsdl/soap"
)

const (
	FacturaA     = 1
	NotaCreditoA = 3 
	FacturaB     = 6
	NotaCreditoB = 8
	FacturaC     = 11
	NotaCreditoC = 13
)

type CabRequest struct {
	Cuit     int64 `json:"cuit"`
	PtoVta   int32 `json:"ptoVta"`
	CbteTipo int32 `json:"cbteTipo"`
}

type CaeRequest struct {
	DocTipo    int32   `json:"docTipo"`
	DocNro     int64   `json:"docNro"`
	CbteDesde  int64   `json:"cbteDesde"`
	CbteHasta  int64   `json:"cbteHasta"`
	CbteFch    string  `json:"cbteFch"`
	ImpNeto    float64 `json:"impNeto"`
	ImpOpEx    float64 `json:"impOpEx"`
	ImpTotConc float64 `json:"impTotConc"`
	ImpTotal   float64 `json:"impTotal"`
	ImpTrib    float64 `json:"impTrib"`
	ImpIVA     float64 `json:"impIVA"`
	IvasArray  []struct {
		ID      int32   `json:"id"`
		BaseImp float64 `json:"baseImp"`
		Importe float64 `json:"importe"`
	} `json:"ivasArray"`
	TributosArray []struct {
		ID      int16   `json:"id"`
		BaseImp float64 `json:"baseImp"`
		Desc    string  `json:"desc"`
		Alic    float64 `json:"Alic"`
		Importe float64 `json:"importe"`
	} `json:"tributosArray"`
	CbteTipoRef int32 `json:"cbteTipoRef"`
	CbteNroRef  int64 `json:"cbteNroRef"`
	CanMisMonExt string `json:"canMisMonExt"`
	CondicionIVAReceptorId int32 `json:"condicionIVAReceptorId"`
}

const URLWSAATesting string = "https://wswhomo.afip.gov.ar/wsfev1/service.asmx?wsdl"
const URLWSAAProduction string = "https://servicios1.afip.gov.ar/wsfev1/service.asmx?wsdl"

// Environment es un tipo de dato
type Environment int

// Constantes de environment
const (
	TESTING Environment = iota
	PRODUCTION
)

// Service es la estructura global del paquete
type Service struct {
	environment Environment
	serviceSoap ServiceSoap
	token       string
	sign        string
}

func NewService(environment Environment, token, sign string) *Service {
	var url string
	if environment == PRODUCTION {
		url = URLWSAAProduction
	} else {
		url = URLWSAATesting
	}

	soapClient := soap.NewClient(url)
	serviceSoap := NewServiceSoap(soapClient)

	return &Service{environment: environment, serviceSoap: serviceSoap, token: token, sign: sign}
}

func (s *Service) getAuth(cuit int64) *FEAuthRequest {
	feAuthRequest := FEAuthRequest{
		Token: s.token,
		Sign:  s.sign,
		Cuit:  cuit,
	}
	return &feAuthRequest
}

func (s *Service) GetUltimoComp(cabRequest *CabRequest) (int32, error) {
	feCompUltimoAutorizado := FECompUltimoAutorizado{
		Auth:     s.getAuth(cabRequest.Cuit),
		PtoVta:   cabRequest.PtoVta,
		CbteTipo: cabRequest.CbteTipo,
	}

	feCompUltimoAutorizadoResponse, err := s.serviceSoap.FECompUltimoAutorizado(&feCompUltimoAutorizado)
	if err != nil {
		return 0, err
	}

	return feCompUltimoAutorizadoResponse.FECompUltimoAutorizadoResult.CbteNro, nil
}

func (s *Service) CaeRequest(cabRequest *CabRequest, caeRequest *CaeRequest) (string, string, error) {
	feCAECabRequest := FECAECabRequest{
		FECabRequest: &FECabRequest{
			CantReg:  1,
			PtoVta:   cabRequest.PtoVta,
			CbteTipo: cabRequest.CbteTipo,
		},
	}

	ivas := make([]*AlicIva, 0)
	for _, iva := range caeRequest.IvasArray {
		alicIva := AlicIva{
			Id:      iva.ID,
			BaseImp: iva.BaseImp,
			Importe: iva.Importe,
		}
		ivas = append(ivas, &alicIva)
	}

	arrayOfAlicIvas := ArrayOfAlicIva{
		AlicIva: ivas,
	}

	//body request
	feDetRequest := FEDetRequest{
		Concepto:               1,
		DocTipo:                caeRequest.DocTipo,
		DocNro:                 caeRequest.DocNro,
		CbteDesde:              caeRequest.CbteDesde,
		CbteHasta:              caeRequest.CbteHasta,
		CbteFch:                caeRequest.CbteFch,
		ImpTotal:               caeRequest.ImpTotal,
		ImpTotConc:             caeRequest.ImpTotConc,
		ImpNeto:                caeRequest.ImpNeto,
		ImpOpEx:                caeRequest.ImpOpEx,
		ImpTrib:                caeRequest.ImpTrib,
		ImpIVA:                 caeRequest.ImpIVA,
		MonId:                  "PES",
		CanMisMonExt:           "N",  // Si informa MonId = PES, el campo CanMisMonExt no debe informarse.
		CondicionIVAReceptorId: caeRequest.CondicionIVAReceptorId,
		MonCotiz:               1,
		FchVtoPago:             "",
		FchServDesde:           "",
		FchServHasta:           "",
	}

	if cabRequest.CbteTipo != FacturaC && cabRequest.CbteTipo != NotaCreditoC &&
		(caeRequest.ImpIVA > 0 || caeRequest.ImpNeto > 0) {
		feDetRequest.Iva = &arrayOfAlicIvas
	}

	if caeRequest.CbteNroRef > 0 && caeRequest.CbteTipoRef > 0 {
		cbtesAsoc := make([]*CbteAsoc, 0)
		cuit := ""
		if caeRequest.DocNro > 0 {
			docStr := strconv.FormatInt(caeRequest.DocNro, 10)
			if len(docStr) == 11 {
				cuit = docStr
			}
		}
		cbteAsoc := CbteAsoc{
			Tipo:    caeRequest.CbteTipoRef,
			PtoVta:  cabRequest.PtoVta,
			Nro:     caeRequest.CbteNroRef,
			Cuit:    cuit,
			CbteFch: caeRequest.CbteFch,
		}
		cbtesAsoc = append(cbtesAsoc, &cbteAsoc)
		arrayOfCbteAsoc := ArrayOfCbteAsoc{
			CbteAsoc: cbtesAsoc,
		}
		feDetRequest.CbtesAsoc = &arrayOfCbteAsoc
	}

	tributos := make([]*Tributo, 0)
	for _, tributo := range caeRequest.TributosArray {
		tributo := Tributo{
			Id:      tributo.ID,
			BaseImp: tributo.BaseImp,
			Desc:    tributo.Desc,
			Alic:    tributo.Alic,
			Importe: tributo.Importe,
		}
		tributos = append(tributos, &tributo)
	}

	if len(tributos) > 0 {
		arrayOfTributo := ArrayOfTributo{
			Tributo: tributos,
		}
		feDetRequest.Tributos = &arrayOfTributo
	}

	feCAEDetRequest := FECAEDetRequest{
		&feDetRequest,
	}
	arrayOfFECAEDetRequest := ArrayOfFECAEDetRequest{
		FECAEDetRequest: []*FECAEDetRequest{&feCAEDetRequest},
	}

	feCAERequest := FECAERequest{
		FeCabReq: &feCAECabRequest,
		FeDetReq: &arrayOfFECAEDetRequest,
	}

	// solicitar cae request
	feCaeSolicitar := FECAESolicitar{
		Auth:     s.getAuth(cabRequest.Cuit),
		FeCAEReq: &feCAERequest,
	}

	feCAESolicitarResponse, err := s.serviceSoap.FECAESolicitar(&feCaeSolicitar)
	if err != nil {
		return "", "", err
	}

	feCAESolicitarResult := feCAESolicitarResponse.FECAESolicitarResult
	if feCAESolicitarResult.Errors != nil {
		return "", "", fmt.Errorf(feCAESolicitarResult.Errors.Err[0].Msg)
	}

	feDetResponse := feCAESolicitarResult.FeDetResp
	cae := feDetResponse.FECAEDetResponse[0].CAE
	caeFchVto := feDetResponse.FECAEDetResponse[0].CAEFchVto

	if len(feDetResponse.FECAEDetResponse) > 0 {
		if feDetResponse.FECAEDetResponse[0].Observaciones != nil {
			return cae, caeFchVto, fmt.Errorf(feDetResponse.FECAEDetResponse[0].Observaciones.Obs[0].Msg)
		}
	}

	return cae, caeFchVto, nil
}
