package bouncer

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type (
	jsonTestCase struct {
		description         string
		method              string
		withInterface       bool
		shouldSucceedOnJson bool
		shouldFailOnBind    bool
		payload             string
		contentType         string
		testType            string
		ifaceType           interface{}
	}
)

var jsonTestCases = []jsonTestCase{

	{
		description:         "Create Foo",
		method:              "POST",
		shouldSucceedOnJson: true,
		payload:             `{"title":"Foo Title", "content": "Foo Content"}`,
		contentType:         jsonContentType,
		ifaceType:           Foo{},
	},
	{
		description:         "Create Foo with missing fields",
		method:              "POST",
		shouldSucceedOnJson: false,
		payload:             `{"content": "Foo Content"}`,
		contentType:         jsonContentType,
		ifaceType:           Foo{},
	},
	{
		description:         "Create Foo with immutable fields",
		method:              "POST",
		shouldSucceedOnJson: false,
		payload:             `{"title":"Foo Title", "content": "Foo Content", "create_ignored":"bar"}`,
		contentType:         jsonContentType,
		ifaceType:           Foo{},
	},
	{
		description:         "Patch Foo",
		method:              "PATCH",
		shouldSucceedOnJson: true,
		payload:             `{"content": "New Foo Content"}`,
		contentType:         jsonContentType,
		ifaceType:           Foo{},
	},
	{
		description:         "Patch Foo with immutable fields",
		method:              "PATCH",
		shouldSucceedOnJson: false,
		payload:             `{"title":"Foo Title", "content": "Foo Content"}`,
		contentType:         jsonContentType,
		ifaceType:           Foo{},
	},
	{
		description:         "Create Foo [DIRECT]",
		method:              "POST",
		shouldSucceedOnJson: true,
		payload:             `{"title":"Foo Title", "content": "Foo Content"}`,
		contentType:         jsonContentType,
		testType:            "direct",
		ifaceType:           Foo{},
	},
	{
		description:         "Create Foo with missing fields [DIRECT]",
		method:              "POST",
		shouldSucceedOnJson: false,
		payload:             `{"content": "Foo Content"}`,
		contentType:         jsonContentType,
		testType:            "direct",
		ifaceType:           Foo{},
	},
	{
		description:         "Create Foo with immutable fields [DIRECT]",
		method:              "POST",
		shouldSucceedOnJson: false,
		payload:             `{"title":"Foo Title", "content": "Foo Content", "create_ignored":"bar"}`,
		contentType:         jsonContentType,
		testType:            "direct",
		ifaceType:           Foo{},
	},
	{
		description:         "Patch Foo [DIRECT]",
		method:              "PATCH",
		shouldSucceedOnJson: true,
		payload:             `{"content": "New Foo Content"}`,
		contentType:         jsonContentType,
		testType:            "direct",
		ifaceType:           Foo{},
	},
	{
		description:         "Patch Foo with immutable fields [DIRECT]",
		method:              "PATCH",
		shouldSucceedOnJson: false,
		payload:             `{"title":"Foo Title", "content": "Foo Content"}`,
		contentType:         jsonContentType,
		testType:            "direct",
		ifaceType:           Foo{},
	},
}

func TestJson(t *testing.T) {
	for _, testCase := range jsonTestCases {
		performJsonTest(t, testCase)
	}
}

func performJsonTest(t *testing.T, testCase jsonTestCase) {
	if testCase.testType == "direct" {
		//use the validator directly
		switch testCase.ifaceType.(type) {
		case Foo:
			errs := ValidateJson(Foo{}, []byte(testCase.payload), testCase.method)
			if testCase.shouldSucceedOnJson &&
				errs.Len() > 0 {
				t.Errorf("'%s' should have succeeded, but returned errors '%+v'",
					testCase.description, errs)
			} else if !testCase.shouldSucceedOnJson && errs.Len() <= 0 {
				t.Errorf("'%s' should have failed, but returned NO errors", testCase.description)
			}
		}

		return
	}
	var payload io.Reader
	httpRecorder := httptest.NewRecorder()

	jsonTestHandler := func(w http.ResponseWriter, r *http.Request) {
		if !testCase.shouldSucceedOnJson {
			t.Errorf("'%s' should NOT have succeeded, but there were NO errors", testCase.description)
		}
	}

	var testHandler http.Handler
	switch testCase.ifaceType.(type) {
	case Foo:
		testHandler = NewBouncerHandler(Foo{}, http.HandlerFunc(jsonTestHandler))
	}

	if testCase.payload == "-nil-" {
		payload = nil
	} else {
		payload = strings.NewReader(testCase.payload)
	}

	req, err := http.NewRequest(testCase.method, testRoute, payload)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", testCase.contentType)

	testHandler.ServeHTTP(httpRecorder, req)

	switch httpRecorder.Code {
	case http.StatusNotFound:
		panic("Routing is messed up in test fixture (got 404): check method and path")
	case http.StatusInternalServerError:
		panic("Something bad happened on '" + testCase.description + "'")
	default:
		if testCase.shouldSucceedOnJson &&
			httpRecorder.Code != http.StatusOK {
			t.Errorf("'%s' should have succeeded, but returned HTTP status %d with body '%s'",
				testCase.description, httpRecorder.Code, httpRecorder.Body.String())
		}
	}
}
