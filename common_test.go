package bouncer

import (
	"net/http"
)

// These types are mostly contrived examples, but they're used
// across many test cases. The idea is to cover all the scenarios
// that this binding package might encounter in actual use.
type (

	// For basic test cases with a required field
	Foo struct {
		Title         string `json:"title" json:"title" create:"required" patch:"-"`
		Content       string `json:"content" json:"content"`
		Ignored       string `json:"-"`
		CreateIgnored string `json:"create_ignored" create:"-"`
	}

	// To be used as a nested struct (with a required field)
	Person struct {
		Name  string `json:"name" json:"name" create:"required"`
		Email string `json:"email" json:"email"`
	}

	// The common function signature of the handlers going under test.
	handlerFunc    func(interface{}, http.Handler) http.Handler
	validationFunc func(interface{}, *http.Request) Errors
)

const (
	testRoute       = "/test"
	formContentType = "application/x-www-form-urlencoded"
)
