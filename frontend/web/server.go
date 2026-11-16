package web

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
)

//go:embed static/app.css templates/page.html
var assets embed.FS

var pageTemplate = template.Must(template.ParseFS(assets, "templates/page.html"))

type pageData struct {
	Customers        []Customer
	ErrorMessage     string
	Form             CustomerForm
	FormErrorMessage string
}

type webServer struct {
	api CustomerAPI
}

// NewHandler serves the server-rendered CRM page, its form posts, and proxies /api/* to the REST API
// (like the Vite dev proxy in the React frontend).
func NewHandler(api CustomerAPI) (http.Handler, error) {
	apiURL, err := url.Parse(api.BaseURL())
	if err != nil {
		return nil, err
	}
	server := &webServer{api: api}
	mux := http.NewServeMux()
	mux.Handle("/api/", newAPIProxy(apiURL))
	mux.HandleFunc("GET /static/app.css", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFileFS(writer, request, assets, "static/app.css")
	})
	mux.HandleFunc("GET /{$}", server.home)
	mux.HandleFunc("POST /customers", server.createCustomer)
	mux.HandleFunc("POST /customers/{customerId}/delete", server.deleteCustomer)
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Not found", http.StatusNotFound)
	})
	return mux, nil
}

func newAPIProxy(apiURL *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(proxyRequest *httputil.ProxyRequest) {
			proxyRequest.SetURL(apiURL)
		},
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			log.Printf("API proxy failed: %v", err)
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusBadGateway)
			writer.Write([]byte(`{"error": "API is unreachable"}`))
		},
	}
}

func (server *webServer) home(writer http.ResponseWriter, request *http.Request) {
	server.renderHome(writer, http.StatusOK, pageData{})
}

func (server *webServer) renderHome(writer http.ResponseWriter, status int, page pageData) {
	customers, err := server.api.ListCustomers()
	if err != nil && page.ErrorMessage == "" {
		page.ErrorMessage = err.Error()
	}
	page.Customers = customers
	var html bytes.Buffer
	if err := pageTemplate.Execute(&html, page); err != nil {
		log.Printf("Could not render the page: %v", err)
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(status)
	writer.Write(html.Bytes())
}

func (server *webServer) createCustomer(writer http.ResponseWriter, request *http.Request) {
	form := CustomerForm{
		FirstName: request.PostFormValue("firstName"),
		LastName:  request.PostFormValue("lastName"),
		Email:     request.PostFormValue("email"),
	}
	if _, err := server.api.CreateCustomer(form); err != nil {
		server.renderHome(writer, http.StatusUnprocessableEntity, pageData{Form: form, FormErrorMessage: err.Error()})
		return
	}
	http.Redirect(writer, request, "/", http.StatusSeeOther)
}

func (server *webServer) deleteCustomer(writer http.ResponseWriter, request *http.Request) {
	customerID, err := strconv.ParseInt(request.PathValue("customerId"), 10, 64)
	if err != nil || customerID < 0 {
		http.Error(writer, "Not found", http.StatusNotFound)
		return
	}
	if err := server.api.DeleteCustomer(customerID); err != nil {
		server.renderHome(writer, http.StatusUnprocessableEntity, pageData{ErrorMessage: err.Error()})
		return
	}
	http.Redirect(writer, request, "/", http.StatusSeeOther)
}
