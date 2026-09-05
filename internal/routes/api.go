package routes

import "net/http"

const APIV1 = "/api/v1"

type API struct {
	mux *http.ServeMux
}

func New(mux *http.ServeMux) *API {
	return &API{mux: mux}
}

func (a *API) V1() *http.ServeMux {
	return a.mux
}

func (a *API) HandleV1(pattern string, handler http.Handler) {
	a.mux.Handle(APIV1+pattern, handler)
}

func (a *API) HandleV1Func(pattern string, handler http.HandlerFunc) {
	a.mux.HandleFunc(APIV1+pattern, handler)
}
