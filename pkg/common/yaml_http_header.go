package common

import (
	"net/http"

	"gopkg.in/yaml.v3"
)

// YAMLHTTPHeader is a type that can unmarshal YAML maps into http.Header.
// It allows fixture files to use map[string]string format while being
// unmarshaled directly into http.Header.
type YAMLHTTPHeader http.Header

// UnmarshalYAML implements yaml.Unmarshaler for YAMLHTTPHeader.
// It expects a YAML map with string keys and string values, and converts
// it into an http.Header.
func (h *YAMLHTTPHeader) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		*h = YAMLHTTPHeader(make(http.Header))
		return nil
	}

	header := make(http.Header)
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind == yaml.ScalarNode && valueNode.Kind == yaml.ScalarNode {
			key := keyNode.Value
			value := valueNode.Value
			if key != "" && value != "" {
				header.Set(key, value)
			}
		}
	}

	*h = YAMLHTTPHeader(header)
	return nil
}

// Header returns the underlying http.Header.
func (h YAMLHTTPHeader) Header() http.Header {
	return http.Header(h)
}
