# Bouncer
Validation for go http handlers

## Usage

```go
    
    type Foo struct {
        Id       int64   `json:"-" create:"-" patch:"-"`
        Name     string  `json:"name" create:"required"`
        Password string  `json:"password" notrim:"true"`
    }

    http.Handle('/foo", NewBouncerHandler(Foo{}, fooHandler)
```

using "-" for create or patch tags indicates this field is immutable and will throw an error if it is found
(and not the zero value for that type)

By default, the leading and trailing spaces are trimmed. You can, however, use the struct tag `notrim:"true"` to keep those spaces.

## Caveat

If you make a field `create:"required"` or `patch:"required"`, then you could not pass trivial/default value of that type. For example, if you set

```go
    
    type Foo struct {
        Id       int64   `json:"id" create:"required"`
        Name     string  `json:"name" create:"required"`
        Active   bool    `json:"password" create:"required"`
    }
```

Then you could not pass the following json on POST request

```json
{
    "id": 0,
    "name": "",
    "active": false
}
```