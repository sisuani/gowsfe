## linux compile

```sh
go build -o gowsfe.so -buildmode=c-shared cmd/lib/main.go
```

## windows compile
```sh
go build -o gowsfe.dll -buildmode=c-shared cmd/lib/main.go
```

## gowsdl

https://github.com/hooklift/gowsdl

```sh
go get github.com/hooklift/gowsdl/...
```

## WSDL AFIP

```sh
gowsdl -p pkg/afip/wsfe -o wsfep.go https://servicios1.afip.gov.ar/wsfev1/service.asmx?WSDL
```
or
```sh
gowsdl -p pkg/afip/wsfe -o wsfeh.go https://wswhomo.afip.gov.ar/wsfev1/service.asmx?WSDL
```

<b>Note: If this results in malformed code, fall back to the alternative below and rename manually.</b>

```sh
gowsdl https://servicios1.afip.gov.ar/wsfev1/service.asmx?WSDL
```
or
```sh
gowsdl https://wswhomo.afip.gov.ar/wsfev1/service.asmx?WSDL
```

Move the output from the previous step to `pkg/afip/wsfe/wsfe.go`
