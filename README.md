# Bouncer
Validation for go http handlers

## Usage

```go
    
    type Foo struct {
        Id    int64   json:"-" create:"-" patch:"-"`
        Name  string  `json:"name" create:"required"`
    }

    http.Handle('/foo", NewBouncerHandler(Foo{}, fooHandler)
```

using "-" for create or patch tags indicates this field is immutable and will throw an error if it is found
(and not the zero value for that type)
