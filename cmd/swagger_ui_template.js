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
};
