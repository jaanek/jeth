package eth

import (
	"bufio"
	"os"
	"strings"
)

func StringsToInterfaces(arr []string) []interface{} {
	var result = make([]interface{}, 0, len(arr))
	for _, i := range arr {
		result = append(result, i)
	}
	return result
}

func StdInReadAll() string {
	arr := make([]string, 0)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		text := scanner.Text()
		if len(text) > 0 {
			arr = append(arr, text)
		} else {
			break
		}
	}
	return strings.Join(arr, "")
}
