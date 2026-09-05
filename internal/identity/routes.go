package identity
import "net/http"
func (h *Handler) Routes(mux *http.ServeMux){mux.HandleFunc("POST /api/v1/auth/register",h.Register);mux.HandleFunc("POST /api/v1/auth/login",h.Login);mux.HandleFunc("POST /api/v1/auth/refresh",h.Refresh);mux.HandleFunc("POST /api/v1/auth/logout",h.Logout)}
