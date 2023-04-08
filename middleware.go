package srv

import "net/http"

// Redirect routes any http requests to an https equivalent.
func Redirect(HTTP, HTTPS string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Host == "localhost"+HTTP {
			req.Host = "localhost" + HTTPS
		}
		// Reconstruct the path with a TLS base.
		target := "https://" + req.Host + req.URL.Path
		// Add querys if present.
		if len(req.URL.RawQuery) > 0 {
			target += "?" + req.URL.RawQuery
		}
		http.Redirect(res, req, target, http.StatusTemporaryRedirect)
	}
}
