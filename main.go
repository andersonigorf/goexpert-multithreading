package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const timeoutRequest = 1

func main() {
	waitGroup := sync.WaitGroup{}
	for _, cep := range os.Args[1:] {
		processCEP(cep, &waitGroup)
		waitGroup.Wait()
	}
}

func processCEP(cep string, waitGroup *sync.WaitGroup) {
	waitGroup.Add(1)
	parsedCep, err := parseCEP(cep)
	if err != nil {
		log.Fatalf("CEP inválido: %v", err)
	}
	apiURLs := createApiUrls(parsedCep)
	result := callApis(apiURLs, waitGroup)
	fmt.Printf("CEP informado: %s\n %s", parsedCep, result)
}

func createApiUrls(cep string) map[string]string {
	return map[string]string{
		"API Cep":    fmt.Sprintf("https://cdn.apicep.com/file/apicep/%s.json", cep),
		"Via Cep":    fmt.Sprintf("http://viacep.com.br/ws/%s/json/", cep),
		"Brasil API": fmt.Sprintf("https://brasilapi.com.br/api/cep/v1/%s", cep),
	}
}

type ApiResult struct {
	name    string
	content string
}

func callApis(apiURLs map[string]string, waitGroup *sync.WaitGroup) string {
	chResult := make(chan ApiResult)
	for name, url := range apiURLs {
		go func(name, url string) {
			chResult <- ApiResult{name, callAPICep(url)}
		}(name, url)
	}

	select {
	case result := <-chResult:
		waitGroup.Done()
		return fmt.Sprintf("URL: %s\n %s: %s\n", apiURLs[result.name], result.name, result.content)
	case <-time.After(time.Second * timeoutRequest):
		waitGroup.Done()
		return "Timeout"
	}
}

func callAPICep(url string) string {
	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Sprintf("status retornado pelo servidor: %v", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	return string(body)
}

func parseCEP(cep string) (string, error) {
	cepCleaned := strings.ReplaceAll(cep, ".", "")
	cepCleaned = strings.ReplaceAll(cepCleaned, "-", "")

	cepRegex := "^[0-9]{8}$"
	isCepValid, err := regexp.MatchString(cepRegex, cepCleaned)
	if err != nil {
		panic(err)
	}

	if !isCepValid {
		return "", fmt.Errorf("%s - O CEP deve conter oito dígitos. Ex.: 12345-123 ou 12345123.\n", cep)
	}

	cep = fmt.Sprintf("%s-%s", cepCleaned[0:5], cepCleaned[5:])

	return cep, nil
}
