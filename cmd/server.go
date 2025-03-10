package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.szostok.io/version"
	"gopkg.in/yaml.v3"
)

//go:embed template.html
var htmlTemplate string

// Microservice represents configuration for each microservice
type Microservice struct {
	Endpoint   string `mapstructure:"endpoint"`
	URL        string `mapstructure:"url"`
	SwaggerURL string `mapstructure:"swagger_url"`
}

type MicroserviceList struct {
	Name     string
	Endpoint string
	Selected bool
}

// OpenAPISpec represents the structure of the OpenAPI specification for extracting info fields.
type OpenAPISpec struct {
	Info struct {
		ServiceTitle   string `json:"title"`
		ServiceSummary string `json:"summary"`
	} `json:"info"`
}

// GetOASSpec retrieves the OpenAPI specification from the specified microservice URL
func GetOASSpec(url string) ([]byte, string, string, error) {
	if !strings.HasSuffix(url, "/") {
		return nil, "", "", fmt.Errorf("microservice URL doesn't have a trailing '/'")
	}
	requestURL := fmt.Sprintf("%s%s", url, apiSpecsPath)
	log.Debug("Requesting ", requestURL)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, "", "", err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("failed to retrieve OpenAPI spec: received status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", err
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(body, &spec); err != nil {
		return nil, "", "", err
	}

	return body, spec.Info.ServiceTitle, spec.Info.ServiceSummary, nil
}

// GenerateHTML generates the HTML for viewing the OpenAPI spec using Swagger UI
func GenerateHTML(spec []byte, microserviceList []MicroserviceList, serviceURL, selectedService, serviceSummary, message string) (string, error) {

	servicesYaml, err := yaml.Marshal(&services)
	checkError(err)

	headersYaml, err := yaml.Marshal(&headers)
	checkError(err)

	oasBinderConfiguration := `
config = "` + cfgFile + `"
proxyAddress = "` + proxyAddress + `"
listenPort = ` + strconv.Itoa(listenPort) + `
listenAddress = "` + listenAddress + `"
apiSpecsPath = "` + apiSpecsPath + `"
services =
` + string(servicesYaml) + `headers = ` + string(headersYaml)

	versionInfo := version.Get()

	aboutOasBinder := `
Version: ` + versionInfo.Version + `
Git Commit: ` + versionInfo.GitCommit + `
Build Date: ` + versionInfo.BuildDate + `
Commit Date: ` + versionInfo.CommitDate + `
Go Version: ` + versionInfo.GoVersion + `
Compiler: ` + versionInfo.Compiler + `
Platform: ` + versionInfo.Platform

	type SwaggerUIParams struct {
		Spec                   string
		Host                   string
		ProxyAddress           string
		Headers                map[string]string
		MicroserviceList       []MicroserviceList
		SelectedService        string
		ServiceSummary         string
		DisplaySwaggerUI       bool
		Message                string
		OASBinderConfiguration string
		AboutOASBinder         string
	}

	params := SwaggerUIParams{
		Spec:                   string(spec),
		Host:                   serviceURL,
		ProxyAddress:           proxyAddress + "/",
		Headers:                headers,
		MicroserviceList:       microserviceList,
		SelectedService:        selectedService,
		ServiceSummary:         serviceSummary,
		DisplaySwaggerUI:       message == "",
		Message:                message,
		OASBinderConfiguration: oasBinderConfiguration,
		AboutOASBinder:         aboutOasBinder,
	}

	t, err := template.New("swaggerui").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var htmlBuilder strings.Builder
	err = t.Execute(&htmlBuilder, params)
	if err != nil {
		return "", err
	}

	return htmlBuilder.String(), nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	log.Debug("path = ", path)

	message := ""
	var spec []byte
	var selectedSpec []byte
	var err error
	var serviceName, serviceSummary, microserviceURL string

	microserviceList := []MicroserviceList{}
	for _, service := range services {
		// Retrieve name and summary of each microservice to construct a drop-down list.
		spec, serviceName, serviceSummary, err = GetOASSpec(service.URL)
		log.Debug("serviceEndpoint = ", service.Endpoint)
		if service.Endpoint == path {
			microserviceURL = service.SwaggerURL
			log.Debug("microserviceURL = ", microserviceURL)
			// selectedSpec is the one which will be rendered by SwaggerUIBundle
			selectedSpec = spec
			if err != nil {
				message = "Could not retrieve OpenAPI spec: " + err.Error()
			}
		}
		if err != nil {
			log.Error("Could not retrieve OpenAPI spec: " + err.Error())
			continue
		}
		microserviceList = append(microserviceList, MicroserviceList{
			Name:     serviceName + " — " + serviceSummary,
			Endpoint: service.Endpoint,
			Selected: service.Endpoint == path,
		})
	}

	log.Debug("microserviceURL = ", microserviceURL)

	if microserviceURL == "" {
		message = "Please select a service from the list."
	}

	html, err := GenerateHTML(selectedSpec, microserviceList, microserviceURL, serviceName, serviceSummary, message)
	if err != nil {
		http.Error(w, "Could not generate HTML", http.StatusInternalServerError)
		log.Error(err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		log.Error("Failed to write response: ", err)
	}
}

func serve() {
	http.HandleFunc("/", handler)

	addr := listenAddress + ":" + strconv.Itoa(listenPort)
	log.Info("Listening on ", addr)

	s := &http.Server{
		Addr:           addr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
