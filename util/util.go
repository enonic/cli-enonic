package util

import (
	"bytes"
	"encoding/json"
	"bufio"
	"os"
	"fmt"
)

func PrettyPrintJSONBytes(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "    ")
	return out.Bytes(), err
}

func PrettyPrintJSON(data interface{}) (string, error) {
	var out = new(bytes.Buffer)
	enc := json.NewEncoder(out)
	enc.SetIndent("", "    ")
	err := enc.Encode(data)
	return out.String(), err
}

func PromptUntilTrue(val string, assessFunc func(val string, i byte) string) string {
	index := byte(0)
	text := assessFunc(val, index)
	for text != "" {
		reader := bufio.NewScanner(os.Stdin)
		fmt.Fprint(os.Stderr, text)
		reader.Scan()
		val = reader.Text()
		index += 1
		text = assessFunc(val, index)
	}
	return val
}
