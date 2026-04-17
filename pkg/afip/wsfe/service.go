package wsfe

import (
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/hooklift/gowsdl/soap"
)

const RequestTimeout = 60 * time.Second

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

// CaeaRequest es el request para FECAEASolicitar / FECAEAConsultar
type CaeaRequest struct {
	Cuit    int64 `json:"cuit"`
	Periodo int32 `json:"periodo"`
	Orden   int16 `json:"orden"`
}

// CaeaRegRequest es el detalle para FECAEARegInformativo (igual que CaeRequest + CAEA y CbteFchHsGen)
type CaeaRegRequest struct {
	CAEA          string  `json:"caea"`
	CbteFchHsGen  string  `json:"cbteFchHsGen"`
	DocTipo       int32   `json:"docTipo"`
	DocNro        int64   `json:"docNro"`
	CbteDesde     int64   `json:"cbteDesde"`
	CbteHasta     int64   `json:"cbteHasta"`
	CbteFch       string  `json:"cbteFch"`
	ImpNeto       float64 `json:"impNeto"`
	ImpOpEx       float64 `json:"impOpEx"`
	ImpTotConc    float64 `json:"impTotConc"`
	ImpTotal      float64 `json:"impTotal"`
	ImpTrib       float64 `json:"impTrib"`
	ImpIVA        float64 `json:"impIVA"`
	IvasArray     []struct {
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
	CbteTipoRef            int32  `json:"cbteTipoRef"`
	CbteNroRef             int64  `json:"cbteNroRef"`
	CondicionIVAReceptorId int32  `json:"condicionIVAReceptorId"`
}

// CaeaSinMovRequest es el request para FECAEASinMovimientoInformar / FECAEASinMovimientoConsultar
type CaeaSinMovRequest struct {
	Cuit   int64  `json:"cuit"`
	PtoVta int32  `json:"ptoVta"`
	CAEA   string `json:"caea"`
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

func BankersRounding(f float64) float64 {
    str := fmt.Sprintf("%.3f", f)
    f, _ = strconv.ParseFloat(str, 64)
    return math.Round(f*100) / 100
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

func isTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

func (s *Service) GetUltimoComp(cabRequest *CabRequest) (int32, error) {
	feCompUltimoAutorizado := FECompUltimoAutorizado{
		Auth:     s.getAuth(cabRequest.Cuit),
		PtoVta:   cabRequest.PtoVta,
		CbteTipo: cabRequest.CbteTipo,
	}

	feCompUltimoAutorizadoResponse, err := s.serviceSoap.FECompUltimoAutorizado(&feCompUltimoAutorizado)
	if err != nil {
		if isTimeoutError(err) {
			return -1, fmt.Errorf("timeout: el servicio AFIP no respondió en %s", RequestTimeout)
		}
		return -1, err
	}

	result := feCompUltimoAutorizadoResponse.FECompUltimoAutorizadoResult
	if result.Errors != nil && len(result.Errors.Err) > 0 {
		return -1, fmt.Errorf("error AFIP (código %d): %s", result.Errors.Err[0].Code, result.Errors.Err[0].Msg)
	}

	return result.CbteNro, nil
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
			BaseImp: BankersRounding(iva.BaseImp),
			Importe: BankersRounding(iva.Importe),
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
		ImpTotal:               BankersRounding(caeRequest.ImpTotal),
		ImpTotConc:             BankersRounding(caeRequest.ImpTotConc),
		ImpNeto:                BankersRounding(caeRequest.ImpNeto),
		ImpOpEx:                BankersRounding(caeRequest.ImpOpEx),
		ImpTrib:                BankersRounding(caeRequest.ImpTrib),
		ImpIVA:                 BankersRounding(caeRequest.ImpIVA),
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
			BaseImp: BankersRounding(tributo.BaseImp),
			Desc:    tributo.Desc,
			Alic:    tributo.Alic,
			Importe: BankersRounding(tributo.Importe),
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
		if isTimeoutError(err) {
			return "", "", fmt.Errorf("timeout: el servicio AFIP no respondió en %s", RequestTimeout)
		}
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

// CaeaSolicitar solicita un CAEA para el periodo y orden indicados.
// Retorna los datos del CAEA obtenido.
func (s *Service) CaeaSolicitar(req *CaeaRequest) (*FECAEAGet, error) {
	feCAEASolicitar := FECAEASolicitar{
		Auth:    s.getAuth(req.Cuit),
		Periodo: req.Periodo,
		Orden:   req.Orden,
	}

	resp, err := s.serviceSoap.FECAEASolicitar(&feCAEASolicitar)
	if err != nil {
		return nil, err
	}

	result := resp.FECAEASolicitarResult
	if result.Errors != nil && len(result.Errors.Err) > 0 {
		return nil, fmt.Errorf(result.Errors.Err[0].Msg)
	}

	return result.ResultGet, nil
}

// CaeaConsultar consulta un CAEA ya emitido para el periodo y orden indicados.
// Retorna los datos del CAEA.
func (s *Service) CaeaConsultar(req *CaeaRequest) (*FECAEAGet, error) {
	feCAEAConsultar := FECAEAConsultar{
		Auth:    s.getAuth(req.Cuit),
		Periodo: req.Periodo,
		Orden:   req.Orden,
	}

	resp, err := s.serviceSoap.FECAEAConsultar(&feCAEAConsultar)
	if err != nil {
		return nil, err
	}

	result := resp.FECAEAConsultarResult
	if result.Errors != nil && len(result.Errors.Err) > 0 {
		return nil, fmt.Errorf(result.Errors.Err[0].Msg)
	}

	return result.ResultGet, nil
}

// CaeaRegInformativo informa comprobantes asociados a un CAEA (rendición informativa).
// Retorna el CAEA y la fecha de vencimiento del comprobante procesado.
func (s *Service) CaeaRegInformativo(cabRequest *CabRequest, caeaReq *CaeaRegRequest) (string, string, error) {
	feCAEACabRequest := FECAEACabRequest{
		FECabRequest: &FECabRequest{
			CantReg:  1,
			PtoVta:   cabRequest.PtoVta,
			CbteTipo: cabRequest.CbteTipo,
		},
	}

	ivas := make([]*AlicIva, 0)
	for _, iva := range caeaReq.IvasArray {
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

	feDetRequest := FEDetRequest{
		Concepto:               1,
		DocTipo:                caeaReq.DocTipo,
		DocNro:                 caeaReq.DocNro,
		CbteDesde:              caeaReq.CbteDesde,
		CbteHasta:              caeaReq.CbteHasta,
		CbteFch:                caeaReq.CbteFch,
		ImpTotal:               caeaReq.ImpTotal,
		ImpTotConc:             caeaReq.ImpTotConc,
		ImpNeto:                caeaReq.ImpNeto,
		ImpOpEx:                caeaReq.ImpOpEx,
		ImpTrib:                caeaReq.ImpTrib,
		ImpIVA:                 caeaReq.ImpIVA,
		MonId:                  "PES",
		CanMisMonExt:           "N",
		CondicionIVAReceptorId: caeaReq.CondicionIVAReceptorId,
		MonCotiz:               1,
	}

	if cabRequest.CbteTipo != 11 &&
		(caeaReq.ImpIVA > 0 || caeaReq.ImpNeto > 0) {
		feDetRequest.Iva = &arrayOfAlicIvas
	}

	if caeaReq.CbteNroRef > 0 && caeaReq.CbteTipoRef > 0 {
		cuit := ""
		if caeaReq.DocNro > 0 {
			docStr := strconv.FormatInt(caeaReq.DocNro, 10)
			if len(docStr) == 11 {
				cuit = docStr
			}
		}
		cbteAsoc := CbteAsoc{
			Tipo:    caeaReq.CbteTipoRef,
			PtoVta:  cabRequest.PtoVta,
			Nro:     caeaReq.CbteNroRef,
			Cuit:    cuit,
			CbteFch: caeaReq.CbteFch,
		}
		arrayOfCbteAsoc := ArrayOfCbteAsoc{
			CbteAsoc: []*CbteAsoc{&cbteAsoc},
		}
		feDetRequest.CbtesAsoc = &arrayOfCbteAsoc
	}

	tributos := make([]*Tributo, 0)
	for _, tributo := range caeaReq.TributosArray {
		t := Tributo{
			Id:      tributo.ID,
			BaseImp: tributo.BaseImp,
			Desc:    tributo.Desc,
			Alic:    tributo.Alic,
			Importe: tributo.Importe,
		}
		tributos = append(tributos, &t)
	}
	if len(tributos) > 0 {
		feDetRequest.Tributos = &ArrayOfTributo{Tributo: tributos}
	}

	feCAEADetRequest := FECAEADetRequest{
		FEDetRequest: &feDetRequest,
		CAEA:         caeaReq.CAEA,
		CbteFchHsGen: caeaReq.CbteFchHsGen,
	}

	feCAEARequest := FECAEARequest{
		FeCabReq: &feCAEACabRequest,
		FeDetReq: &ArrayOfFECAEADetRequest{
			FECAEADetRequest: []*FECAEADetRequest{&feCAEADetRequest},
		},
	}

	feCAEARegInformativo := FECAEARegInformativo{
		Auth:            s.getAuth(cabRequest.Cuit),
		FeCAEARegInfReq: &feCAEARequest,
	}

	resp, err := s.serviceSoap.FECAEARegInformativo(&feCAEARegInformativo)
	if err != nil {
		return "", "", err
	}

	result := resp.FECAEARegInformativoResult
	if result.Errors != nil && len(result.Errors.Err) > 0 {
		return "", "", fmt.Errorf(result.Errors.Err[0].Msg)
	}

	detResp := result.FeDetResp
	if detResp == nil || len(detResp.FECAEADetResponse) == 0 {
		return "", "", fmt.Errorf("respuesta vacía de FECAEARegInformativo")
	}

	det := detResp.FECAEADetResponse[0]
	if det.Observaciones != nil && len(det.Observaciones.Obs) > 0 {
		return det.CAEA, "", fmt.Errorf(det.Observaciones.Obs[0].Msg)
	}

	return det.CAEA, det.CbteFch, nil
}

// CaeaSinMovimientoInformar informa un punto de venta sin movimientos para un CAEA.
// Retorna el resultado ("A" aprobado / "R" rechazado).
func (s *Service) CaeaSinMovimientoInformar(req *CaeaSinMovRequest) (string, error) {
	feCAEASinMovInformar := FECAEASinMovimientoInformar{
		Auth:   s.getAuth(req.Cuit),
		PtoVta: req.PtoVta,
		CAEA:   req.CAEA,
	}

	resp, err := s.serviceSoap.FECAEASinMovimientoInformar(&feCAEASinMovInformar)
	if err != nil {
		return "", err
	}

	result := resp.FECAEASinMovimientoInformarResult
	if result.Errors != nil && len(result.Errors.Err) > 0 {
		return "", fmt.Errorf(result.Errors.Err[0].Msg)
	}

	return result.Resultado, nil
}

// CaeaSinMovimientoConsultar consulta los puntos de venta informados sin movimientos para un CAEA.
// Retorna el listado de registros sin movimiento.
func (s *Service) CaeaSinMovimientoConsultar(req *CaeaSinMovRequest) ([]*FECAEASinMov, error) {
	feCAEASinMovConsultar := FECAEASinMovimientoConsultar{
		Auth:   s.getAuth(req.Cuit),
		CAEA:   req.CAEA,
		PtoVta: req.PtoVta,
	}

	resp, err := s.serviceSoap.FECAEASinMovimientoConsultar(&feCAEASinMovConsultar)
	if err != nil {
		return nil, err
	}

	result := resp.FECAEASinMovimientoConsultarResult
	if result.Errors != nil && len(result.Errors.Err) > 0 {
		return nil, fmt.Errorf(result.Errors.Err[0].Msg)
	}

	if result.ResultGet == nil {
		return []*FECAEASinMov{}, nil
	}

	return result.ResultGet.FECAEASinMov, nil
}

