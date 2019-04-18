package bouncer

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/gorilla/context"
)

const (
	jsonContentType           = "application/json; charset=utf-8"
	StatusUnprocessableEntity = 422
)

type BouncerHandler struct {
	iface interface{}
	f     http.Handler
}

type BouncerPatchHandler struct {
	iface         interface{}
	maxBodyLength int64
	f             http.Handler
}

func NewBouncerHandler(obj interface{}, f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := BouncerHandler{
			f:     f,
			iface: obj,
		}
		h.ServeHTTP(w, r)
	})
}

func NewBouncerPatchHandler(obj interface{}, maxBodyLength int64, f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := BouncerPatchHandler{
			f:             f,
			maxBodyLength: maxBodyLength,
			iface:         obj,
		}
		h.ServeHTTP(w, r)
	})
}

func (h BouncerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	errs := Validate(h.iface, r)

	if len(errs) > 0 {
		ErrorHandler(errs, w)
		return
	}

	h.f.ServeHTTP(w, r)

}

func (h BouncerPatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var errors Errors

	mr := http.MaxBytesReader(w, r.Body, h.maxBodyLength)
	defer mr.Close() //also closes r.Body
	jsonData, err := ioutil.ReadAll(mr)
	if err != nil {
		errors.Add([]string{}, DeserializationError, err.Error())
		ErrorHandler(errors, w)
		return
	}

	// validate json, potentially modify it
	mergeObject, errs := ValidateJson(h.iface, jsonData, r.Method)
	if len(errs) > 0 {
		ErrorHandler(errs, w)
		return
	}

	// marshall the json object back to a string
	mergeJson, err := json.Marshal(mergeObject)
	if err != nil {
		errors.Add([]string{}, DeserializationError, err.Error())
		ErrorHandler(errors, w)
		return
	}

	// ensure the final object only contains keys that it started with
	finalJson, err := CreateEncodedInterfaceFromOriginal(jsonData, mergeJson)
	if err != nil {
		errors.Add([]string{}, DeserializationError, err.Error())
		ErrorHandler(errors, w)
		return
	}

	context.Set(r, "requestBody", finalJson)

	h.f.ServeHTTP(w, r)

}

func CreateEncodedInterfaceFromOriginal(originalJson []byte, latestJson []byte) ([]byte, error) {
	var originalInterface interface{}
	var latestInterface interface{}

	// unmarshal originalJson patch input to a generic interface
	err := json.Unmarshal(originalJson, &originalInterface)
	if err != nil {
		return nil, err
	}

	// unmarshal modified patch input to a generic interface
	err = json.Unmarshal(latestJson, &latestInterface)
	if err != nil {
		return nil, err
	}

	// Merge the modified input into the originalJson, ignoring fields that didn't exist in the originalJson (these were added when unmarshalled)
	finalInterface, err := MergeInterface(originalInterface, latestInterface)
	if err != nil {
		return nil, err
	}

	finalJson, err := json.Marshal(finalInterface)
	if err != nil {
		return nil, err
	}

	return finalJson, nil
}

func MergeInterface(dest interface{}, src interface{}) (interface{}, error) {
	var err error

	switch destMap := dest.(type) {
	case map[string]interface{}:
		if srcMap, ok := src.(map[string]interface{}); ok {
			// somehow the types are different. This shouldn't be possible
			for key := range destMap {
				// if src interface doesn't have the key, skip (struct is ignoring it, so we can too)
				if _, ok := srcMap[key]; !ok {
					continue
				}

				// potentially update the value of destMap for the current key
				destMap[key], err = MergeInterface(destMap[key], srcMap[key])
				if err != nil {
					return nil, err
				}
			}

			// all keys updated or omitted, return the new interface
			return destMap, nil
		}
		return nil, errors.New("fatal issue merging sanitized patch data")
	case []interface{}:
		// no naive way to know which items in an unordered list are associated
		// so arrays will have to be passed through unmodified
		return dest, nil
	default:
		// if we get here it shouldn't be from a top level call, so must be recusrive from a range over the original patch fields
		return src, nil
	}
}

// ErrorHandler simply counts the number of errors in the
// context and, if more than 0, writes a response with an
// error code and a JSON payload describing the errors.
// The response will have a JSON content-type.
// Middleware remaining on the stack will not even see the request
// if, by this point, there are any errors.
// This is a "default" handler, of sorts, and you are
// welcome to use your own instead. The Bind middleware
// invokes this automatically for convenience.
func ErrorHandler(errs Errors, resp http.ResponseWriter) {
	if len(errs) > 0 {
		resp.Header().Set("Content-Type", jsonContentType)
		if errs.Has(DeserializationError) {
			resp.WriteHeader(http.StatusBadRequest)
		} else if errs.Has(ContentTypeError) {
			resp.WriteHeader(http.StatusUnsupportedMediaType)
		} else {
			resp.WriteHeader(StatusUnprocessableEntity)
		}
		errOutput, _ := json.Marshal(errs)
		resp.Write(errOutput)
		return
	}
}

func Validate(obj interface{}, req *http.Request) Errors {
	contentType := req.Header.Get("Content-Type")
	if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" || contentType != "" {

		if strings.Contains(contentType, "json") {
			return Json(obj, req)
		}
		return Json(obj, req)
	}
	return nil
}

func Json(jsonStruct interface{}, req *http.Request) Errors {
	body, errors := validateJsonFromReader(jsonStruct, req.Body, req.Method)
	context.Set(req, "decodedBody", body)
	return errors

}

func ValidateJson(jsonStruct interface{}, jsonData []byte, method string) (interface{}, Errors) {
	return validateJsonFromReader(jsonStruct, bytes.NewReader(jsonData), method)
}

func validateJsonFromReader(jsonStruct interface{}, reader io.Reader, method string) (interface{}, Errors) {
	var errors Errors
	ensureNotPointer(jsonStruct)
	obj := reflect.New(reflect.TypeOf(jsonStruct))

	if reader != nil {
		err := json.NewDecoder(reader).Decode(obj.Interface())
		if err != nil && err != io.EOF {
			errors.Add([]string{}, DeserializationError, err.Error())
		}
	}

	if method == "PATCH" {
		errors = validatePatchStruct(errors, obj.Interface())
	} else if method == "POST" || method == "PUT" {
		errors = validateCreateStruct(errors, obj.Interface())
	}

	return obj.Interface(), errors

}

func validateCreateStruct(errors Errors, obj interface{}) Errors {
	typ := reflect.TypeOf(obj)
	val := reflect.ValueOf(obj)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip ignored and unexported fields in the struct
		if field.Tag.Get("form") == "-" || !val.Field(i).CanInterface() {
			continue
		}

		fieldValue := val.Field(i).Interface()
		zero := reflect.Zero(field.Type).Interface()

		// If the field Value is a string, then trim the leading spaces
		if field.Tag.Get("notrim") != "true" {
			fieldActualValue := val.Field(i)
			if fieldActualValue.IsValid() {
				if fieldActualValue.CanSet() {
					if fieldActualValue.Kind() == reflect.String {
						fieldActualValue.SetString(strings.TrimSpace(fieldValue.(string)))
					}
				}
			}
		}

		// Validate nested and embedded structs (if pointer, only do so if not nil)
		if field.Type.Kind() == reflect.Struct ||
			(field.Type.Kind() == reflect.Ptr && !reflect.DeepEqual(zero, fieldValue) &&
				field.Type.Elem().Kind() == reflect.Struct) {
			errors = validateCreateStruct(errors, fieldValue)
		}

		if field.Tag.Get("create") == "-" {
			//this is immutable - make sure it's zero
			if !reflect.DeepEqual(zero, fieldValue) {
				name := field.Name
				if j := field.Tag.Get("json"); j != "" {
					name = j
				} else if f := field.Tag.Get("form"); f != "" {
					name = f
				}
				errors.Add([]string{name}, ImmutableError, "Immutable")
			}
		}

		if strings.Index(field.Tag.Get("create"), "required") > -1 {
			if reflect.DeepEqual(zero, fieldValue) {
				name := field.Name
				if j := field.Tag.Get("json"); j != "" {
					name = j
				} else if f := field.Tag.Get("form"); f != "" {
					name = f
				}
				errors.Add([]string{name}, RequiredError, "Required")
			}
		}
	}
	return errors

}

func validatePatchStruct(errors Errors, obj interface{}) Errors {
	typ := reflect.TypeOf(obj)
	val := reflect.ValueOf(obj)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip ignored and unexported fields in the struct
		if field.Tag.Get("form") == "-" || !val.Field(i).CanInterface() {
			continue
		}

		fieldValue := val.Field(i).Interface()
		zero := reflect.Zero(field.Type).Interface()

		// If the field Value is a string, then trim the leading spaces
		if field.Tag.Get("notrim") != "true" {
			fieldActualValue := val.Field(i)
			if fieldActualValue.IsValid() {
				if fieldActualValue.CanSet() {
					if fieldActualValue.Kind() == reflect.String {
						fieldActualValue.SetString(strings.TrimSpace(fieldValue.(string)))
					}
				}
			}
		}

		// Validate nested and embedded structs (if pointer, only do so if not nil)
		if field.Type.Kind() == reflect.Struct ||
			(field.Type.Kind() == reflect.Ptr && !reflect.DeepEqual(zero, fieldValue) &&
				field.Type.Elem().Kind() == reflect.Struct) {
			errors = validatePatchStruct(errors, fieldValue)
		}

		if field.Tag.Get("patch") == "-" {
			//this is immutable - make sure it's zero
			if !reflect.DeepEqual(zero, fieldValue) {
				name := field.Name
				if j := field.Tag.Get("json"); j != "" {
					name = j
				} else if f := field.Tag.Get("form"); f != "" {
					name = f
				}
				errors.Add([]string{name}, ImmutableError, "Immutable")
			}
		}

		if strings.Index(field.Tag.Get("patch"), "required") > -1 {
			if reflect.DeepEqual(zero, fieldValue) {
				name := field.Name
				if j := field.Tag.Get("json"); j != "" {
					name = j
				} else if f := field.Tag.Get("form"); f != "" {
					name = f
				}
				errors.Add([]string{name}, RequiredError, "Required")
			}
		}
	}
	return errors

}

// Don't pass in pointers to bouncer
func ensureNotPointer(obj interface{}) {
	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		panic("Pointers are not accepted as binding models")
	}
}
