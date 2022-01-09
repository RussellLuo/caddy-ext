package requestbodyvar

import (
	"bytes"
	"fmt"
	"mime"
	"strings"

	"github.com/basgys/goxml2json"
	"github.com/tidwall/gjson"
)

type Querier interface {
	Query(string) string
}

type JSON struct {
	buf *bytes.Buffer
}

func (j JSON) Query(key string) string {
	return getJSONField(j.buf, key)
}

type XML struct {
	buf *bytes.Buffer
}

func (x XML) Query(key string) string {
	json, err := xml2json.Convert(x.buf)
	if err != nil {
		return ""
	}
	return getJSONField(json, key)
}

func newQuerier(buf *bytes.Buffer, contentType string) (Querier, error) {
	mediaType := "application/json"
	if contentType != "" {
		var err error
		mediaType, _, err = mime.ParseMediaType(contentType)
		if err != nil {
			return nil, err
		}
	}

	switch {
	case mediaType == "application/json":
		return JSON{buf: buf}, nil
	case strings.HasSuffix(mediaType, "/xml"):
		// application/xml
		// text/xml
		return XML{buf: buf}, nil
	default:
		return nil, fmt.Errorf("unsupported Media Type: %q", mediaType)
	}
}

// getJSONField gets the value of the given field from the JSON body,
// which is buffered in buf.
func getJSONField(buf *bytes.Buffer, key string) string {
	if buf == nil {
		return ""
	}
	value := gjson.GetBytes(buf.Bytes(), key)
	return value.String()
}
