package srv

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

const pkg = "srv"

// Mware
type Mware func(http.HandlerFunc) http.HandlerFunc
type Routes []Route

// Route contains a path and an http.HandlerFunc and is the fundamental 'unit'
// or 'object' of the srv package.
type Route struct {
	pattern string
	fn      http.HandlerFunc
}

// Handle takes a pattern and either an http.Handler or a http.HandlerFunc or a
// function that meets the http.HandlerFunc type requirments returning a Route
// which contins the given object wrappend with any Mware that have been passed
// in after the Handler or the HandlerFunc object.
func Handle(pattern string, h any, mw ...Mware) Route {
	var fn http.HandlerFunc
	switch t := h.(type) {
	case http.Handler:
		fn = func(res http.ResponseWriter, req *http.Request) {
			t.ServeHTTP(res, req)
		}
	case http.HandlerFunc:
		fn = t
	case func(http.ResponseWriter, *http.Request):
		fn = http.HandlerFunc(t)
	default:
		msg := fmt.Sprintf("require either http.Handler or http.HandlerFunc got: %T", h)
		log.Output(2, msg)
		os.Exit(1)
	}
	route := Route{pattern, fn}
	for _, fn := range mw {
		route.fn = fn(route.fn)
	}
	return route
}

// Wrap wraps the Route with the given Mware's.
func (r Route) Wrap(mw ...Mware) Route {
	for _, fn := range mw {
		r.fn = fn(r.fn)
	}
	return r
}

// Group is an intermedary object which may contain any one of, a slice of
// Groups, Routes or Mwares, The Groups will wrap all of the its sub Groups and
// Routes with any Mwares that are applied to it using Wrap.
type Group struct {
	groups []Group
	routes []Route
	wrap   []Mware
}

// Wrap wraps all sub groups and routes withing the group with the give Mware.
func (g Group) Wrap(mw ...Mware) Group {
	g.wrap = append(g.wrap, mw...)
	return g
}

// Add takes either Group as sub groups or Routes and adds them to this Group.
func (g Group) Add(v ...any) Group {
	for _, v := range v {
		switch t := v.(type) {
		case []Group:
			g.groups = append(g.groups, t...)
		case Group:
			g.groups = append(g.groups, t)
		case []Route:
			g.routes = append(g.routes, t...)
		case Route:
			g.routes = append(g.routes, t)
		case string:
			log.Fatal("use " + pkg + ".Handle() to add an endpoint")
		default:
			log.Fatalf("unknown type: %T", t)
		}
	}
	return g
}

// compose compiles the groups sub groups into routes and wraps them with the
// groups Mware functions.
func (g Group) compose() []Route {
	for _, group := range g.groups {
		g.routes = append(g.routes, group.compose()...)
	}
	for j := range g.routes {
		for i := range g.wrap {
			g.routes[j].fn = g.wrap[i](g.routes[j].fn)
		}
	}
	return g.routes
}

// Router contains and compiles your applications endpoints, middle ware that
// wraps the router will be run both first and last in the ordering of the
// nested function chain upon all of the routes that it contains.
type Router struct {
	mux    *http.ServeMux
	groups []Group
	routes []Route
	wrap   []Mware
}

// NewRouter returns a Router with a new *http.ServeMux server alreasy set
// inside.
func NewRouter() Router {
	return Router{
		mux: http.NewServeMux(),
	}
}

// Set sets the given *http.ServeMux server into the router.
func (r *Router) Set(mux *http.ServeMux) *Router {
	r.mux = mux
	return r
}

// Wrap adds the given Mware to the Router, to be latter applied to evey route
// and group that the router contains, upon composing.
func (r Router) Wrap(mw ...Mware) Router {
	r.wrap = append(r.wrap, mw...)
	return r
}

// Add adds any given Groups or Routes to the router. Handlers and
// HanderlerFuncs should be added using Handle.
func (r Router) Add(v ...any) Router {
	for _, in := range v {
		switch t := in.(type) {
		case []Group:
			r.groups = append(r.groups, t...)
		case Group:
			r.groups = append(r.groups, t)
		case []Route:
			r.routes = append(r.routes, t...)
		case Route:
			r.routes = append(r.routes, t)
		case http.HandlerFunc:
			log.Fatal("use " + pkg + ".Handle() to add endpoint")
		case string:
			log.Fatal("use " + pkg + ".Handle() to add endpoint")
		default:
			log.Fatalf("unknown type: %T %v", t, t)
		}
	}
	return r
}

// Wrap adds the given Mware to all of these Routes.
func (r Routes) Wrap(mw ...Mware) Routes {
	for j := range r {
		for i := range mw {
			r[j].fn = mw[i](r[j].fn)
		}
	}
	return r
}

// Serve returns a new *http.ServeMux with all of these Routes added.
func (r Routes) Serve() *http.ServeMux {
	server := http.NewServeMux()
	for i := range r {
		server.HandleFunc(r[i].pattern, r[i].fn)
	}
	return server
}

// Compose adds any given Routes or Groups to the server and then recursivly
// composes all groups into routes wrapping them with any group specific
// middleware then finaly it wraps all of its Routes with any Mware that the
// Router contains.
func (r Router) Compose(v ...any) *http.ServeMux {
	r = r.Add(v...)
	for _, group := range r.groups {
		r.routes = append(r.routes, group.compose()...)
	}
	for j := range r.routes {
		for _, fn := range r.wrap {
			r.routes[j].fn = fn(r.routes[j].fn)
		}
		r.mux.HandleFunc(r.routes[j].pattern, r.routes[j].fn)
	}
	return r.mux
}
