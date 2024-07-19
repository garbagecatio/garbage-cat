package errors

import (
	"fmt"
	"strings"

	"github.com/getsentry/sentry-go"
)

// Op is a unique string describing a method or a function
// Multiple operations can construct a friendly stack tace.
type Op string

// Pkg is the package name our error originates from
type Pkg string

// Sn is a server name field
// each microservice in our network is given a unique name
type Sn string

// Kind is a string categorizing the error
type Kind string

const (
	Database               Kind = "Database"
	DatabaseResultNotFound Kind = "Database Result Not Found"
	Network                Kind = "Network"
	Type                   Kind = "Type"
)

// IsKind unwraps an *Error and checks if its top level kind matches
func IsKind(err error, kind Kind) bool {
	unwrapped, ok := err.(*Error)
	if !ok {
		panic("bad call to IsKind")
	}

	if unwrapped.Kind == kind {
		return true
	}
	return false
}

// Error is a custom Error struct for quickly diagnosing issues with our app
type Error struct {
	Sn   Sn     `json:"server,omitempty"`    // server name
	Pkg  Pkg    `json:"package,omitempty"`   // package module
	Op   Op     `json:"operation,omitempty"` // operation
	Kind Kind   `json:"kind,omitempty"`      // category of errors
	Err  error  `json:"err,omitempty"`       // the wrapped error
	Msg  string `json:"message,omitempty"`
}

func Sentry(hub *sentry.Hub, err error) {
	if hub != nil {
		e, ok := err.(*Error)
		if !ok {
			sentry.CaptureException(err)
			return
		}

		hub.WithScope(func(scope *sentry.Scope) {
			scope.AddEventProcessor(func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
				if e.Kind != "" && e.Msg != "" {
					event.Exception[0].Type = fmt.Sprintf("%v: %v", e.Kind, e.Msg)
					return event
				} else if e.Kind != "" {
					event.Exception[0].Type = fmt.Sprintf("%v", e.Kind)
					return event
				} else if e.Msg != "" {
					event.Exception[0].Type = fmt.Sprintf("%v", e.Msg)
					return event
				}
				return event
			})
			hub.CaptureException(e)
		})
	}
}

func (e *Error) Error() string {
	payload := "------------------------------------------------------------------------\n"

	if e.Sn != "" {
		payload += fmt.Sprintf("\t   server: %v\n", e.Sn)
	}

	if e.Pkg != "" {
		payload += fmt.Sprintf("\t  package: %v\n", e.Pkg)
	}

	if e.Op != "" {
		payload += fmt.Sprintf("\toperation: %v\n", e.Op)
	}

	if e.Kind != "" {
		payload += fmt.Sprintf("\t     kind: %v\n", e.Kind)
	}

	if e.Msg != "" {
		payload += fmt.Sprintf("\t  message: %v\n", e.Msg)
	}

	res := []string{payload}
	subErr, ok := e.Err.(*Error)
	if !ok {
		payload += fmt.Sprintf("\t    error: %v\n", e.Err)
		return payload
	}
	res = append(res, subErr.Error())
	return fmt.Sprintf("\nTrace\n%v------------------------------------------------------------------------\n", strings.Join(res, ""))
}

// E is a helper method for filling our Error Struct, it panics if you send anything other than an error, string, Op, or Kind
func E(args ...interface{}) error {
	e := &Error{}
	for _, arg := range args {
		switch arg := arg.(type) {
		case Sn:
			e.Sn = arg
		case Pkg:
			e.Pkg = arg
		case Op:
			e.Op = arg
		case Kind:
			e.Kind = arg
		case error:
			e.Err = arg
		case Error:
			e.Err = &arg
		case string:
			e.Msg = arg
		default:
			panic("bad call to E")
		}
	}
	return e
}

// Servers is a recursive function for building a trace of servers
func Servers(err error) []Sn {
	unwrapped, ok := err.(*Error)
	if !ok {
		panic("bad call to Servers")
	}

	res := []Sn{unwrapped.Sn}
	subErr, ok := unwrapped.Err.(*Error)
	if !ok {
		return res
	}
	res = append(res, Servers(subErr)...)
	return res
}

// Packages is a recursive function for building a trace of packages
func Packages(err error) []Pkg {
	unwrapped, ok := err.(*Error)
	if !ok {
		panic("bad call to Packages")
	}

	res := []Pkg{unwrapped.Pkg}
	subErr, ok := unwrapped.Err.(*Error)
	if !ok {
		return res
	}
	res = append(res, Packages(subErr)...)
	return res
}

// Operations is a recursive function for building a trace of operations
func Operations(err error) []Op {
	unwrapped, ok := err.(*Error)
	if !ok {
		panic("bad call to Operations")
	}

	res := []Op{unwrapped.Op}
	subErr, ok := unwrapped.Err.(*Error)
	if !ok {
		return res
	}
	res = append(res, Operations(subErr)...)
	return res
}

// Kinds is a recursive function for building a trace of kinds
func Kinds(err error) []Kind {
	unwrapped, ok := err.(*Error)
	if !ok {
		panic("bad call to Kind")
	}

	res := []Kind{unwrapped.Kind}
	subErr, ok := unwrapped.Err.(*Error)
	if !ok {
		return res
	}
	res = append(res, Kinds(subErr)...)
	return res
}

// Messages is a recursive function for building a trace of Messages
func Messages(err error) []string {
	unwrapped, ok := err.(*Error)
	if !ok {
		panic("bad call to Messages")
	}

	res := []string{unwrapped.Msg}
	subErr, ok := unwrapped.Err.(*Error)
	if !ok {
		return res
	}
	res = append(res, Messages(subErr)...)
	return res
}

// LastMessage retrieves the last message on an error chain
func LastMessage(err error) string {
	msgs := Messages(err)
	return msgs[len(msgs)-1]
}
