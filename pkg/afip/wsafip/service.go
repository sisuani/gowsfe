package wsafip

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/sisuani/gowsfe/pkg/certs"

	"github.com/hooklift/gowsdl/soap"
)

// URLWSAATesting ... wsdl de wsaa en environment de homolagación de afip
const URLWSAATesting string = "https://wsaahomo.afip.gov.ar/ws/services/LoginCms?WSDL"

// URLWSAAProduction ... wsdl de wsaa en environment de producción de afip
const URLWSAAProduction string = "https://wsaa.afip.gov.ar/ws/services/LoginCms?WSDL"

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
	key         string
	cert        string
	urlWsaa     string
	tickets     map[string]*LoginTicketResponse
}

// LoginTicket es una estructura que representa un ticket de un servicio de afip
type LoginTicket struct {
	ServiceName    string
	Token          string
	Sign           string
	ExpirationTime time.Time
}

// HeaderLoginTicket es la cabecera de la estructura de request y response
type HeaderLoginTicket struct {
	Source         string `xml:"source,omitempty"`
	Destination    string `xml:"destination,omitempty"`
	UniqueID       uint32 `xml:"uniqueId,omitempty"`
	GenerationTime string `xml:"generationTime,omitempty"`
	ExpirationTime string `xml:"expirationTime,omitempty"`
}

// LoginTicketRequest es la estructura general del request
type LoginTicketRequest struct {
	XMLName xml.Name           `xml:"loginTicketRequest"`
	Version string             `xml:"version,attr"`
	Header  *HeaderLoginTicket `xml:"header,omitempty"`
	Service string             `xml:"service,omitempty"`
}

// Credentials es la estructura que devuelve el response con la info principal
type Credentials struct {
	Token string `xml:"token,omitempty"`
	Sign  string `xml:"sign,omitempty"`
}

// LoginTicketResponse ...
type LoginTicketResponse struct {
	XMLName     xml.Name           `xml:"loginTicketResponse"`
	Header      *HeaderLoginTicket `xml:"header,omitempty"`
	Credentials *Credentials       `xml:"credentials,omitempty"`
}

// Create crea un objeto cliente para acceder a los servicios web de afip
func NewService(environment Environment, cert, key string) *Service {
	var url string
	if environment == PRODUCTION {
		url = URLWSAAProduction
	} else {
		url = URLWSAATesting
	}

	return &Service{environment: environment, urlWsaa: url, cert: cert, key: key, tickets: make(map[string]*LoginTicketResponse)}
}

// GetLoginTicket devuelve el ticket de acceso afip correspondiente al servicio pasado por parámetro.
func (s *Service) GetLoginTicket(serviceName string) (token string, sign string, expiration string, err error) {
	expired := true

	ticket, _ := s.tickets[serviceName]
	if ticket != nil {
		expTime, err := time.Parse(time.RFC3339, ticket.Header.ExpirationTime)
		if err != nil {
			return "", "", "", fmt.Errorf("GetLoginTicket: Error parseando fecha de expiración del ticket. %s", err)
		}

		expired = time.Now().After(expTime)
	}

	if expired {
		expiration := time.Now().Add(10 * time.Minute)
		generationTime := fmt.Sprintf("%s", time.Now().Add(-10*time.Minute).Format(time.RFC3339))
		expirationTime := fmt.Sprintf("%s", expiration.Format(time.RFC3339))

		// Armo estructura request
		loginTicketRequest := LoginTicketRequest{
			Version: "1.0",
			Header: &HeaderLoginTicket{
				UniqueID:       1,
				GenerationTime: generationTime,
				ExpirationTime: expirationTime,
			},
			Service: serviceName,
		}

		// Armo XML
		loginTicketRequestXML, err := xml.MarshalIndent(loginTicketRequest, " ", "  ")
		if err != nil {
			return "", "", "", fmt.Errorf("GetLoginTicket: Error armando login ticket request XML. %s", err)
		}
		content := []byte(string(loginTicketRequestXML))

		// Creo CMS (Cryptographic Message Syntax)
		certificate, privateKey, err := certs.LoadX509KeyPair(s.cert, s.key)
		if err != nil {
			return "", "", "", fmt.Errorf("GetLoginTicket: %s", err)
		}
		cms, err := certs.EncodeCMS(content, certificate, privateKey)
		if err != nil {
			return "", "", "", fmt.Errorf("GetLoginTicket: %s", err)
		}

		// Convierto CMS a base64
		cmsBase64 := base64.StdEncoding.EncodeToString(cms)

		// Armo conexión SOAP y solicitud
		soapClient := soap.NewClient(s.urlWsaa)
		login := NewLoginCMS(soapClient)

		request := LoginCms{In0: cmsBase64}

		// Logeo solicitud
		if s.environment == TESTING {
			requestXML, _ := xml.MarshalIndent(request, " ", "  ")
			fmt.Printf("REQUEST XML:\n%s\n\n", xml.Header+string(requestXML))
		}

		// Llamo al servicio de autenticación afip wssa
		responseXML, err := login.LoginCms(&request)
		if err != nil {
			return "", "", "", fmt.Errorf("GetLoginTicket: %s", err)
		}

		// Logeo respuesta
		if s.environment == TESTING {
			fmt.Printf("RESPONSE XML:\n%s\n\n", responseXML)
		}

		// Desarmo respuesta XML
		response := LoginTicketResponse{}
		if err := xml.Unmarshal([]byte(responseXML.LoginCmsReturn), &response); err != nil {
			return "", "", "", fmt.Errorf("GetLoginTicket: Error desarmando respuesta XML. %s", err)
		}

		// Almaceno ticket de respuesta (porque no se puede llamar nuevamente al servicio hasta dentro de 10 minutos,
		// hay que seguir usando el ticket actual. El vencimiento de los ticket de afip suele ser de 12 horas)
		s.tickets[serviceName] = &response
	}

	return s.tickets[serviceName].Credentials.Token, s.tickets[serviceName].Credentials.Sign, s.tickets[serviceName].Header.ExpirationTime, nil
}
