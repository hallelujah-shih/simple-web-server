package service

import (
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (sw *SimpleWeb) regHandlers() {
	sw.httpMux.HandleFunc("/ca", sw.caHandler)
	sw.httpMux.HandleFunc("/", sw.rootHandler)
	sw.httpMux.HandleFunc("/hello", sw.helloHandler)
}

func (sw *SimpleWeb) caHandler(writer http.ResponseWriter, request *http.Request) {
	ca := sw.pm.GetRoot()

	var buf strings.Builder
	for _, data := range ca.Certificate {
		buf.Write(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: data,
		}))
		buf.WriteString("\n")
	}
	fmt.Fprintf(writer, buf.String())
}

func (sw *SimpleWeb) rootHandler(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, `help:
curl /ca # get root cert
curl /hello # rsp world
`)
}

func (sw *SimpleWeb) helloHandler(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "rsp world time: %v\n", time.Now().Format(time.RFC3339))
}
