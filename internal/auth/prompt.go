package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

func ReadAuthorizationCode(input io.Reader, output io.Writer) (string, error) {
	fmt.Fprint(output, "> ")

	reader := bufio.NewReader(input)
	code, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	code = strings.TrimSpace(code)
	if code == "" {
		return "", errors.New("authorization code is empty")
	}

	return code, nil
}
