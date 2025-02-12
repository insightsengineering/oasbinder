package cmd

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"errors"
)

// Microservice represents configuration for each microservice
type Microservice struct {
	Name     string `yaml:"name"`
	Endpoint string `yaml:"endpoint"`
	URL      string `yaml:"url"`
}

// GetOASSpec retrieves the OpenAPI specification from the specified microservice URL
func GetOASSpec(url string) ([]byte, error) {
	if !strings.HasSuffix(url, "/") {
		return nil, errors.New("Microservice URL doesn't have a trailing '/'.")
	}
	requestUrl := fmt.Sprintf("%s%s", url, apiSpecsPath)
	log.Debug("Requesting ", requestUrl)
	req, err := http.NewRequest("GET", requestUrl, nil)
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

	return ioutil.ReadAll(resp.Body)
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

	tmpl := `<!DOCTYPE html>
<html>
<head>
  <title>Swagger UI</title>
  <!-- Load the latest Swagger UI code and style from npm using unpkg.com -->
  <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.17.14/swagger-ui-bundle.min.js" integrity="sha512-7ihPQv5ibiTr0DW6onbl2MIKegdT6vjpPySyIb4Ftp68kER6Z7Yiub0tFoMmCHzZfQE9+M+KSjQndv6NhYxDgg==" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
  <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.17.14/swagger-ui-standalone-preset.min.js" integrity="sha512-UrYi+60Ci3WWWcoDXbMmzpoi1xpERbwjPGij6wTh8fXl81qNdioNNHExr9ttnBebKF0ZbVnPlTPlw+zECUK1Xw==" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
  <link rel="stylesheet" type="text/css" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.17.14/swagger-ui.min.css" integrity="sha512-+9UD8YSD9GF7FzOH38L9S6y56aYNx3R4dYbOCgvTJ2ZHpJScsahNdaMQJU/8osUiz9FPu0YZ8wdKf4evUbsGSg==" crossorigin="anonymous" referrerpolicy="no-referrer" />
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/bootstrap/5.3.3/css/bootstrap.min.css" integrity="sha512-jnSuA4Ss2PkkikSOLtYs8BlYIeeIK1h99ty4YfvRPAlzr377vr3CXDb7sb7eEEBYjDtcYj+AjBH3FLv5uSJuXg==" crossorigin="anonymous" referrerpolicy="no-referrer" />

  <style>
    .toolbar {
      margin: 20px 0;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="toolbar">
      <div class="form-group">
        <label for="microservice-select" class="font-weight-bold">Select Microservice:</label>
        <select id="microservice-select" class="form-control">
          <option value=""></option>
          {{range .MicroserviceList}}
            <option value="{{.Name}}" {{if .Selected}}selected{{end}}>{{.Name}}</option>
          {{end}}
        </select>
      </div>
    <div id="swagger-ui"></div>
  </div>
  <script>
  document.getElementById('microservice-select').addEventListener('change', function() {
      var newEndpoint = this.value;
      if (newEndpoint !== "{{.SelectedService}}") {
        window.location.href = newEndpoint;
      }
    });`
	if message == "" {
		tmpl += `
    window.onload = function() {
      const ui = SwaggerUIBundle({
        spec: JSON.parse({{.Spec}}),
        dom_id: "#swagger-ui",
        requestInterceptor: (req) => {
          req.url = req.url.replace({{.OasbinderAddress}}, {{.Host}});
          {{range $key, $value := .Headers}}
          req.headers["{{$key}}"] = "{{$value}}";
          {{end}}
          return req;
        }
      });
    };`
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
	w.Write([]byte(html))
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
