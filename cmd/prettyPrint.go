//go:build noretty

package cmd

func PrettyPrinttFile(filepath string, content string) (string, error) {
	return content, nil
}
