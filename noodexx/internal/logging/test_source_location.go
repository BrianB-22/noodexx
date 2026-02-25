package logging

import (
	"bytes"
	"fmt"
)

func TestSourceLocation() {
	var buf bytes.Buffer
	logger := NewLogger("test", DEBUG, &buf)
	
	logger.Info("test message")
	logger.WithContext("key", "value").Warn("warning with context")
	
	fmt.Println("Output:")
	fmt.Println(buf.String())
}
