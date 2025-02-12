package cmd

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed template.html
var html_template string

//go:embed swagger_ui_template.js
var swagger_ui_template string

// Microservice represents configuration for each microservice
type Microservice struct {
	Name     string `yaml:"name"`
	Endpoint string `yaml:"endpoint"`
	URL      string `yaml:"url"`
}

// GetOASSpec retrieves the OpenAPI specification from the specified microservice URL
func GetOASSpec(url string) ([]byte, error) {
	if !strings.HasSuffix(url, "/") {
		return nil, fmt.Errorf("microservice URL doesn't have a trailing '/'")
	}
	requestURL := fmt.Sprintf("%s%s", url, apiSpecsPath)
	log.Debug("Requesting ", requestURL)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve OpenAPI spec: received status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// GenerateHTML generates the HTML for viewing the OpenAPI spec using Swagger UI
func GenerateHTML(spec []byte, serviceURL string, selectedService string, message string) (string, error) {
	type MicroserviceOption struct {
		Name     string
		Selected bool
	}

	type SwaggerUIParams struct {
		Spec             string
		Host             string
		OasbinderAddress string
		Headers          map[string]string
		MicroserviceList []MicroserviceOption
		SelectedService  string
	}

	microserviceOptions := []MicroserviceOption{}
	for _, ms := range services {
		microserviceOptions = append(microserviceOptions, MicroserviceOption{
			Name:     ms.Name,
			Selected: ms.Name == selectedService,
		})
	}

	params := SwaggerUIParams{
		Spec:             string(spec),
		Host:             serviceURL,
		OasbinderAddress: oasbinderAddress + "/",
		Headers:          headers,
		MicroserviceList: microserviceOptions,
		SelectedService:  selectedService,
	}

	tmpl := html_template

	// Only include the SwaggerUIBundle if a service is selected from drop-down list
	// and the OAS specs have been successfully retrieved from the service.
	if message == "" {
		tmpl += swagger_ui_template
	}
	tmpl += `</script><br />` + message + `</body></html>`

	t, err := template.New("swaggerui").Parse(tmpl)
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
	log.Debug("path = ", r.URL.Path)
	serviceName := strings.TrimPrefix(r.URL.Path, "/")
	log.Debug("serviceName = ", serviceName)
	microserviceURL := ""
	for _, service := range services {
		if service.Name == serviceName {
			microserviceURL = service.URL
			break
		}
	}

	log.Debug("microserviceURL = ", microserviceURL)

	message := ""
	var spec []byte
	var err error

	if microserviceURL == "" {
		message = "Please select a service from the list."
	} else {
		spec, err = GetOASSpec(microserviceURL)
		if err != nil {
			message = "Could not retrieve OpenAPI spec: " + err.Error()
		}
	}

	html, err := GenerateHTML(spec, microserviceURL, serviceName, message)
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

	addr := listenAddress + ":" + strconv.Itoa(oasbinderPortNumber)
	log.Info("Listening on ", addr)

	s := &http.Server{
		Addr:           addr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
