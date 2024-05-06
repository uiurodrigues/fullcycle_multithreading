package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	log.Println("Starting server...")
	defer log.Println("Server finished...")

	r := mux.NewRouter()
	r.HandleFunc("/ceps/{cep}", handler)
	http.ListenAndServe(":8080", r)
}

func handler(w http.ResponseWriter, r *http.Request) {
	defer log.Println("Request finished...")

	log.Println("Request received on server...")

	vars := mux.Vars(r)
	cep, ok := vars["cep"]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ch := make(chan string)

	go getAddressFromViaCEP(cep, ch)
	go getAddressFromBrasilAPI(cep, ch)

	address := <-ch
	log.Println("Request processed successfully")
	log.Printf("Resultado >>> %s", address)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(address))
}

func getAddressFromBrasilAPI(cep string, ch chan string) {
	ctxHttp, ctxHttpCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer ctxHttpCancel()

	url := fmt.Sprintf("https://brasilapi.com.br/api/cep/v1/%s", cep)
	req, err := http.NewRequestWithContext(ctxHttp, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("Error creating BrasilAPI request. Err:%s", err.Error())
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error while executing request to BrasilAPI. Err:%s", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error while reading result from BrasilAPI. Err:%s", err.Error())
		return
	}

	var address BrasilApiResponse
	if err = json.Unmarshal(body, &address); err != nil {
		log.Printf("Error while converting result from BrasilAPI. Err:%s", err.Error())
		return
	}

	ch <- address.ToString()
}

func getAddressFromViaCEP(cep string, ch chan string) {
	ctxHttp, ctxHttpCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer ctxHttpCancel()

	url := fmt.Sprintf("http://viacep.com.br/ws/%s/json/", cep)
	req, err := http.NewRequestWithContext(ctxHttp, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("Error creating ViaCEP request. Err:%s", err.Error())
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error executing ViaCEP request. Err:%s", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error while reading ViaCEP result. Err:%s", err.Error())
		return
	}

	var address ViaCEPResponse
	if err = json.Unmarshal(body, &address); err != nil {
		log.Printf("Error while converting ViaCEP result. Err:%s", err.Error())
		return
	}

	ch <- address.ToString()
}

type BrasilApiResponse struct {
	CEP          string `json:"cep"`
	State        string `json:"state"`
	City         string `json:"city"`
	Neighborhood string `json:"neighborhood"`
	Street       string `json:"street"`
	Service      string `json:"service"`
}

func (r *BrasilApiResponse) ToString() string {
	return fmt.Sprintf("Fonte BrasilAPI >>> CEP:%s, Cidade:%s-%s, Logradouro:%s - Bairro:%s", r.CEP, r.City, r.State, r.Street, r.Neighborhood)
}

type ViaCEPResponse struct {
	CEP         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Localidade  string `json:"localidade"`
	UF          string `json:"uf"`
	DDD         string `json:"ddd"`
}

func (r *ViaCEPResponse) ToString() string {
	return fmt.Sprintf("Fonte ViaCEP >>> CEP:%s, Cidade:%s-%s, Logradouro:%s - Bairro:%s", r.CEP, r.Localidade, r.UF, r.Logradouro, r.Bairro)
}
