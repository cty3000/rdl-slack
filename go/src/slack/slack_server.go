//
// This file generated by rdl 1.5.0
//

package slack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	rdl "github.com/ardielle/ardielle-go/rdl"
	"github.com/dimfeld/httptreemux"
)

var _ = json.Marshal
var _ = ioutil.Discard

//
// Init initializes the Slack server with a service identity and an
// implementation (SlackHandler), and returns an http.Handler to serve it.
//
func Init(impl SlackHandler, baseURL string, authz rdl.Authorizer, authns ...rdl.Authenticator) http.Handler {
	for strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL[0 : len(baseURL)-1]
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		log.Fatal(err)
	}
	b := u.Path
	router := httptreemux.New()
	adaptor := SlackAdaptor{impl, authz, authns, b}

	router.POST(b+"/event", func(w http.ResponseWriter, m *http.Request, ps map[string]string) {
		adaptor.postSlackEventHandler(w, m, ps)
	})
	router.GET(b+"/api/tunnels/command_line", func(w http.ResponseWriter, m *http.Request, ps map[string]string) {
		adaptor.getNgrokInterfaceHandler(w, m, ps)
	})
	router.GET(b+"/services/:T/:B/:X", func(w http.ResponseWriter, m *http.Request, ps map[string]string) {
		adaptor.getSlackWebhookResponseHandler(w, m, ps)
	})
	router.POST(b+"/services/:T/:B/:X", func(w http.ResponseWriter, m *http.Request, ps map[string]string) {
		adaptor.postSlackWebhookRequestHandler(w, m, ps)
	})
	router.NotFoundHandler = func(w http.ResponseWriter, m *http.Request) {
		rdl.JSONResponse(w, 404, rdl.ResourceError{Code: http.StatusNotFound, Message: "Not Found"})
	}
	log.Printf("Initialized Slack service at '%s'\n", baseURL)
	return router
}

//
// SlackHandler is the interface that the service implementation must conform to
//
type SlackHandler interface {
	PostSlackEvent(context *rdl.ResourceContext, request *SlackEvent) (*SlackEvent, error)
	GetNgrokInterface(context *rdl.ResourceContext) (*NgrokInterface, error)
	GetSlackWebhookResponse(context *rdl.ResourceContext, T string, B string, X string) (SlackWebhookResponse, error)
	PostSlackWebhookRequest(context *rdl.ResourceContext, T string, B string, X string, request *SlackWebhookRequest) (SlackWebhookResponse, error)
	Authenticate(context *rdl.ResourceContext) bool
}

//
// SlackAdaptor - this adapts the http-oriented router calls to the non-http service handler.
//
type SlackAdaptor struct {
	impl           SlackHandler
	authorizer     rdl.Authorizer
	authenticators []rdl.Authenticator
	endpoint       string
}

func (adaptor SlackAdaptor) authenticate(context *rdl.ResourceContext) bool {
	if adaptor.authenticators != nil {
		for _, authn := range adaptor.authenticators {
			var creds []string
			var ok bool
			header := authn.HTTPHeader()
			if strings.HasPrefix(header, "Cookie.") {
				if cookies, ok2 := context.Request.Header["Cookie"]; ok2 {
					prefix := header[7:] + "="
					for _, c := range cookies {
						if strings.HasPrefix(c, prefix) {
							creds = append(creds, c[len(prefix):])
							ok = true
							break
						}
					}
				}
			} else {
				creds, ok = context.Request.Header[header]
			}
			if ok && len(creds) > 0 {
				principal := authn.Authenticate(creds[0])
				if principal != nil {
					context.Principal = principal
					return true
				}
			}
		}
	}
	if adaptor.impl.Authenticate(context) {
		return true
	}
	log.Println("*** Authentication failed against all authenticator(s)")
	return false
}

func (adaptor SlackAdaptor) authorize(context *rdl.ResourceContext, action string, resource string) bool {
	if adaptor.authorizer == nil {
		return true
	}
	if !adaptor.authenticate(context) {
		return false
	}
	ok, err := adaptor.authorizer.Authorize(action, resource, context.Principal)
	if err == nil {
		return ok
	}
	log.Println("*** Error when trying to authorize:", err)
	return false
}

func intFromString(s string) int64 {
	var n int64 = 0
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

func floatFromString(s string) float64 {
	var n float64 = 0
	_, _ = fmt.Sscanf(s, "%g", &n)
	return n
}

func (adaptor SlackAdaptor) postSlackEventHandler(writer http.ResponseWriter, request *http.Request, params map[string]string) {
	context := &rdl.ResourceContext{Writer: writer, Request: request, Params: params, Principal: nil}
	body, oserr := ioutil.ReadAll(request.Body)
	if oserr != nil {
		rdl.JSONResponse(writer, http.StatusBadRequest, rdl.ResourceError{Code: http.StatusBadRequest, Message: "Bad request: " + oserr.Error()})
		return
	}
	var argRequest *SlackEvent
	oserr = json.Unmarshal(body, &argRequest)
	if oserr != nil {
		rdl.JSONResponse(writer, http.StatusBadRequest, rdl.ResourceError{Code: http.StatusBadRequest, Message: "Bad request: " + oserr.Error()})
		return
	}
	data, err := adaptor.impl.PostSlackEvent(context, argRequest)
	if err != nil {
		switch e := err.(type) {
		case *rdl.ResourceError:
			rdl.JSONResponse(writer, e.Code, err)
		default:
			rdl.JSONResponse(writer, 500, &rdl.ResourceError{Code: 500, Message: e.Error()})
		}
	} else {
		rdl.JSONResponse(writer, 200, data)
	}

}

func (adaptor SlackAdaptor) getNgrokInterfaceHandler(writer http.ResponseWriter, request *http.Request, params map[string]string) {
	context := &rdl.ResourceContext{Writer: writer, Request: request, Params: params, Principal: nil}
	data, err := adaptor.impl.GetNgrokInterface(context)
	if err != nil {
		switch e := err.(type) {
		case *rdl.ResourceError:
			rdl.JSONResponse(writer, e.Code, err)
		default:
			rdl.JSONResponse(writer, 500, &rdl.ResourceError{Code: 500, Message: e.Error()})
		}
	} else {
		rdl.JSONResponse(writer, 200, data)
	}

}

func (adaptor SlackAdaptor) getSlackWebhookResponseHandler(writer http.ResponseWriter, request *http.Request, params map[string]string) {
	context := &rdl.ResourceContext{Writer: writer, Request: request, Params: params, Principal: nil}
	argT := context.Params["T"]
	argB := context.Params["B"]
	argX := context.Params["X"]
	data, err := adaptor.impl.GetSlackWebhookResponse(context, argT, argB, argX)
	if err != nil {
		switch e := err.(type) {
		case *rdl.ResourceError:
			rdl.JSONResponse(writer, e.Code, err)
		default:
			rdl.JSONResponse(writer, 500, &rdl.ResourceError{Code: 500, Message: e.Error()})
		}
	} else {
		rdl.JSONResponse(writer, 200, data)
	}

}

func (adaptor SlackAdaptor) postSlackWebhookRequestHandler(writer http.ResponseWriter, request *http.Request, params map[string]string) {
	context := &rdl.ResourceContext{Writer: writer, Request: request, Params: params, Principal: nil}
	argT := context.Params["T"]
	argB := context.Params["B"]
	argX := context.Params["X"]
	body, oserr := ioutil.ReadAll(request.Body)
	if oserr != nil {
		rdl.JSONResponse(writer, http.StatusBadRequest, rdl.ResourceError{Code: http.StatusBadRequest, Message: "Bad request: " + oserr.Error()})
		return
	}
	var argRequest *SlackWebhookRequest
	oserr = json.Unmarshal(body, &argRequest)
	if oserr != nil {
		rdl.JSONResponse(writer, http.StatusBadRequest, rdl.ResourceError{Code: http.StatusBadRequest, Message: "Bad request: " + oserr.Error()})
		return
	}
	data, err := adaptor.impl.PostSlackWebhookRequest(context, argT, argB, argX, argRequest)
	if err != nil {
		switch e := err.(type) {
		case *rdl.ResourceError:
			rdl.JSONResponse(writer, e.Code, err)
		default:
			rdl.JSONResponse(writer, 500, &rdl.ResourceError{Code: 500, Message: e.Error()})
		}
	} else {
		rdl.JSONResponse(writer, 200, data)
	}

}
