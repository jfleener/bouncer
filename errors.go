package bouncer

type (
	// Errors may be generated during deserialization, binding,
	// or validation. This type is mapped to the context so you
	// can inject it into your own handlers and use it in your
	// application if you want all your errors to look the same.
	Errors []Error

	Error struct {
		// An error supports zero or more field names, because an
		// error can morph three ways: (1) it can indicate something
		// wrong with the request as a whole, (2) it can point to a
		// specific problem with a particular input field, or (3) it
		// can span multiple related input fields.
		FieldNames []string `json:"fieldNames,omitempty"`

		// The classification is like an error code, convenient to
		// use when processing or categorizing an error programmatically.
		// It may also be called the "kind" of error.
		Classification string `json:"classification,omitempty"`

		// Message should be human-readable and detailed enough to
		// pinpoint and resolve the problem, but it should be brief. For
		// example, a payload of 100 objects in a JSON array might have
		// an error in the 41st object. The message should help the
		// end user find and fix the error with their request.
		Message string `json:"message,omitempty"`
	}
)

// Add adds an error associated with the fields indicated
// by fieldNames, with the given classification and message.
func (e *Errors) Add(fieldNames []string, classification, message string) {
	*e = append(*e, Error{
		FieldNames:     fieldNames,
		Classification: classification,
		Message:        message,
	})
}

// Len returns the number of errors.
func (e *Errors) Len() int {
	return len(*e)
}

// Has determines whether an Errors slice has an Error with
// a given classification in it; it does not search on messages
// or field names.
func (e *Errors) Has(class string) bool {
	for _, err := range *e {
		if err.Kind() == class {
			return true
		}
	}
	return false
}

// Fields returns the list of field names this error is
// associated with.
func (e Error) Fields() []string {
	return e.FieldNames
}

// Kind returns this error's classification.
func (e Error) Kind() string {
	return e.Classification
}

// Error returns this error's message.
func (e Error) Error() string {
	return e.Message
}

const (
	ImmutableError       = "ImmutableError"
	RequiredError        = "RequiredError"
	ContentTypeError     = "ContentTypeError"
	DeserializationError = "DeserializationError"
	TypeError            = "TypeError"
)
