package lib

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// ReadFile locates and reads the file. Either it's an actual file name in which case
// we'll read it right away, or if not we'll try to locate it in the appropriate
// directory with one of the configured extensions.
func ReadFile(fileName *string, fileType string) (string, string, error) {
	filePath := *fileName
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// fileName is not an actual file. Try to find it in the right subdirectory.
		fileFound := false
		fileDirectory := viper.GetString(fileType + ".directory")
		// First, try the bare name in the directory (handles names that already include an extension).
		filePath = filepath.Join(fileDirectory, *fileName)
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			fileFound = true
		}
		if !fileFound {
			for _, extension := range viper.GetStringSlice(fileType + ".extensions") {
				filePath = filepath.Join(fileDirectory, *fileName+extension)
				if _, err := os.Stat(filePath); !os.IsNotExist(err) {
					fileFound = true
					break
				}
			}
		}
		if !fileFound {
			errMsg := fmt.Sprintf("no file found for '%s' matching '%s'", fileType, *fileName)
			return "", "", errors.New(errMsg)
		}
	}
	dat, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}
	return string(dat), filePath, nil
}

// ReadTemplate reads a template file using the configured templates directory and extensions.
func ReadTemplate(templateName *string) (string, string, error) {
	return ReadFile(templateName, "templates")
}

// ReadTagsfile reads a tags file using the configured tags directory and extensions.
func ReadTagsfile(tagsName string) (string, string, error) {
	return ReadFile(&tagsName, "tags")
}

// ReadParametersfile reads a parameters file using the configured parameters directory and extensions.
func ReadParametersfile(parametersName string) (string, string, error) {
	return ReadFile(&parametersName, "parameters")
}

// ReadDeploymentFile reads a deployment file using the configured deployments directory and extensions.
func ReadDeploymentFile(deploymentmentFileName string) (string, string, error) {
	return ReadFile(&deploymentmentFileName, "deployments")
}

// UploadTemplate uploads a CloudFormation template to S3 with a timestamped name and returns the generated key.
func UploadTemplate(ctx context.Context, templateName *string, template string, bucketName *string, svc S3UploadAPI) (string, error) {
	// use the template name with a timestamp that should be unique
	// prefix with fog to make it easier to set up specific lifecycle rules
	generatedname := fmt.Sprintf("fog/%v-%v", *templateName, time.Now().UnixNano())
	input := s3.PutObjectInput{
		Bucket: bucketName,
		Key:    aws.String(generatedname),
		Body:   strings.NewReader(template),
	}
	_, err := svc.PutObject(ctx, &input)
	if err != nil {
		return generatedname, err
	}
	return generatedname, nil
}

// splitShellArgs splits a command string into arguments, respecting single
// and double-quoted substrings. Quotes are stripped from the result and spaces
// inside quotes are preserved as part of the argument. Backslash-escaped
// quotes inside double-quoted strings are handled (e.g., "arg with \"escaped\" quotes").
// Backslash-escaped spaces outside quotes are treated as literal spaces within
// the current argument (e.g., path\ with\ spaces becomes "path with spaces").
// Returns an error if the input contains unbalanced quotes.
func splitShellArgs(s string) ([]string, error) {
	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && inDouble && i+1 < len(s) && s[i+1] == '"':
			// Escaped double quote inside a double-quoted string
			current.WriteByte('"')
			i++ // skip the escaped quote
		case c == '\\' && !inSingle && !inDouble && i+1 < len(s):
			// Backslash escape outside quotes: treat next character literally
			i++
			current.WriteByte(s[i])
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == ' ' && !inSingle && !inDouble:
			if current.Len() > 0 || (i > 0 && (s[i-1] == '\'' || s[i-1] == '"')) {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if inSingle {
		return nil, fmt.Errorf("unbalanced single quote in command: %s", s)
	}
	if inDouble {
		return nil, fmt.Errorf("unbalanced double quote in command: %s", s)
	}
	if current.Len() > 0 || (len(s) > 0 && (s[len(s)-1] == '\'' || s[len(s)-1] == '"')) {
		args = append(args, current.String())
	}
	return args, nil
}

// blockedPrecheckCommands are never allowed to run as prechecks, even when a
// wrapper binary delegates execution to them. Wrapper inspection is
// intentionally conservative: command strings are only unwrapped when they can
// be reduced to a single argv vector without shell-style control operators.
var blockedPrecheckCommands = []string{"rm", "del", "kill"}

func normalizePrecheckCommandName(command string) string {
	name := strings.ToLower(filepath.Base(command))
	switch filepath.Ext(name) {
	case ".exe", ".bat", ".cmd":
		return strings.TrimSuffix(name, filepath.Ext(name))
	default:
		return name
	}
}

func findUnsafeWrappedPrecheck(args []string) (string, error) {
	for len(args) > 0 {
		command := args[0]
		commandName := normalizePrecheckCommandName(command)
		if stringInSlice(commandName, blockedPrecheckCommands) {
			return command, nil
		}

		var (
			nextArgs []string
			err      error
		)
		switch commandName {
		case "env":
			nextArgs, err = unwrapEnvCommand(args[1:])
		case "sh", "bash", "zsh", "dash", "ksh", "ash":
			nextArgs, err = unwrapShellCommand(args[1:])
		case "cmd":
			nextArgs, err = unwrapCmdCommand(args[1:])
		case "powershell", "pwsh":
			nextArgs, err = unwrapPowerShellCommand(args[1:])
		default:
			return "", nil
		}
		if err != nil {
			return "", err
		}
		if len(nextArgs) == 0 {
			return "", nil
		}
		args = nextArgs
	}

	return "", nil
}

func unwrapEnvCommand(args []string) ([]string, error) {
	for i := 0; i < len(args); {
		arg := args[i]
		switch {
		case arg == "--":
			if i+1 >= len(args) {
				return nil, nil
			}
			return args[i+1:], nil
		case arg == "-S" || arg == "--split-string":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("env wrapper missing command after %q", arg)
			}
			return splitShellArgs(args[i+1])
		case arg == "-u" || arg == "--unset" || arg == "-C" || arg == "--chdir" || arg == "-a" || arg == "--argv0":
			i += 2
		case strings.HasPrefix(arg, "--unset=") || strings.HasPrefix(arg, "--chdir=") || strings.HasPrefix(arg, "--argv0="):
			i++
		case arg == "-i" || arg == "--ignore-environment" || arg == "-0" || arg == "--null" || arg == "-v" || arg == "--debug":
			i++
		case strings.Contains(arg, "=") && !strings.HasPrefix(arg, "="):
			i++
		default:
			return args[i:], nil
		}
	}

	return nil, nil
}

func parseWrappedCommandString(commandString string, wrapper string) ([]string, error) {
	if hasUnquotedCommandOperators(commandString) {
		return nil, fmt.Errorf("%s command strings cannot be safely unwrapped for precheck", wrapper)
	}
	return splitShellArgs(commandString)
}

func hasUnquotedCommandOperators(commandString string) bool {
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(commandString); i++ {
		c := commandString[i]
		switch {
		case escaped:
			escaped = false
		case c == '\\' && !inSingle:
			escaped = true
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case !inSingle && !inDouble:
			switch c {
			case ';', '|', '&', '\n', '\r', '<', '>', '(', ')':
				return true
			}
		}
	}

	return false
}

func unwrapShellShortOptions(arg string, args []string, index *int) ([]string, error) {
	shortOpts := strings.TrimPrefix(arg, "-")
	for pos := 0; pos < len(shortOpts); pos++ {
		switch shortOpts[pos] {
		case 'c':
			if pos != len(shortOpts)-1 {
				return nil, nil
			}
			if *index+1 >= len(args) {
				return nil, nil
			}
			return parseWrappedCommandString(args[*index+1], "shell -c")
		case 'o', 'O':
			if pos+1 < len(shortOpts) {
				return nil, nil
			}
			*index++
			return nil, nil
		}
	}

	return nil, nil
}

func unwrapShellCommand(args []string) ([]string, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-c":
			if i+1 >= len(args) {
				return nil, nil
			}
			return parseWrappedCommandString(args[i+1], "shell -c")
		case arg == "-o" || arg == "-O" || arg == "--rcfile" || arg == "--init-file":
			i++
		case strings.HasPrefix(arg, "--rcfile=") || strings.HasPrefix(arg, "--init-file="):
			continue
		case arg == "--":
			return nil, nil
		case strings.HasPrefix(arg, "--"):
			continue
		case strings.HasPrefix(arg, "-"):
			nextArgs, err := unwrapShellShortOptions(arg, args, &i)
			if err != nil {
				return nil, err
			}
			if len(nextArgs) > 0 {
				return nextArgs, nil
			}
			continue
		default:
			return nil, nil
		}
	}

	return nil, nil
}

func unwrapCmdCommand(args []string) ([]string, error) {
	for i, arg := range args {
		if !strings.EqualFold(arg, "/c") && !strings.EqualFold(arg, "/k") {
			continue
		}
		if i+1 >= len(args) {
			return nil, nil
		}
		rest := args[i+1:]
		if len(rest) == 1 {
			return splitShellArgs(rest[0])
		}
		return rest, nil
	}

	return nil, nil
}

func matchesPowerShellFlagPrefix(arg string, fullFlag string, minPrefix string) bool {
	return len(arg) >= len(minPrefix) && len(arg) <= len(fullFlag) && strings.HasPrefix(fullFlag, arg)
}

func isPowerShellCommandFlag(arg string) bool {
	lowerArg := strings.ToLower(arg)
	return lowerArg == "-c" || matchesPowerShellFlagPrefix(lowerArg, "-command", "-com")
}

func isPowerShellEncodedCommandFlag(arg string) bool {
	lowerArg := strings.ToLower(arg)
	return lowerArg == "-enc" || matchesPowerShellFlagPrefix(lowerArg, "-encodedcommand", "-enc")
}

func isPowerShellFileFlag(arg string) bool {
	lowerArg := strings.ToLower(arg)
	return lowerArg == "-f" || matchesPowerShellFlagPrefix(lowerArg, "-file", "-fi")
}

func decodePowerShellEncodedCommand(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("invalid PowerShell encoded command: %w", err)
	}
	if len(decoded)%2 != 0 {
		return "", fmt.Errorf("invalid PowerShell encoded command length")
	}

	utf16Words := make([]uint16, 0, len(decoded)/2)
	for i := 0; i < len(decoded); i += 2 {
		utf16Words = append(utf16Words, uint16(decoded[i])|uint16(decoded[i+1])<<8)
	}

	return string(utf16.Decode(utf16Words)), nil
}

func unwrapPowerShellCommand(args []string) ([]string, error) {
	for i, arg := range args {
		switch {
		case isPowerShellEncodedCommandFlag(arg):
			if i+1 >= len(args) {
				return nil, nil
			}
			commandString, err := decodePowerShellEncodedCommand(args[i+1])
			if err != nil {
				return nil, err
			}
			return parseWrappedCommandString(commandString, "PowerShell encoded")
		case isPowerShellFileFlag(arg):
			return nil, fmt.Errorf("PowerShell -File commands cannot be safely unwrapped for precheck")
		case isPowerShellCommandFlag(arg):
			if i+1 >= len(args) {
				return nil, nil
			}
			rest := args[i+1:]
			if len(rest) == 1 {
				return parseWrappedCommandString(rest[0], "PowerShell")
			}
			return rest, nil
		}
	}

	return nil, nil
}

// RunPrechecks executes configured template validation commands and returns results for each check.
func RunPrechecks(deployment *DeployInfo) (map[string]string, error) {
	results := make(map[string]string)
	for _, precheck := range viper.GetStringSlice("templates.prechecks") {
		precheck := strings.ReplaceAll(precheck, "$TEMPLATEPATH", deployment.TemplateRelativePath)
		separated, err := splitShellArgs(precheck)
		if err != nil {
			return results, err
		}
		if len(separated) == 0 {
			return results, fmt.Errorf("precheck command is empty or only whitespace: %q", precheck)
		}
		command, args := separated[0], separated[1:]
		if unsafeCommand, err := findUnsafeWrappedPrecheck(separated); err != nil {
			return results, err
		} else if unsafeCommand != "" {
			return results, fmt.Errorf("unsafe command '%v' detected", unsafeCommand)
		}
		binary, lookErr := exec.LookPath(command)
		if lookErr != nil {
			return results, fmt.Errorf("command '%v' cannot be found", command)
		}
		cmd := exec.Command(binary, args...)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			results[precheck] = stderr.String() + out.String()
			deployment.PrechecksFailed = true
		}
	}
	return results, nil
}

// YamlToJson converts a YAML byte array to a JSON byte array
func YamlToJson(input []byte) ([]byte, error) {
	var unmarshalled any
	if err := yaml.Unmarshal(input, &unmarshalled); err != nil {
		return nil, fmt.Errorf("invalid YAML: %s", err)
	}
	unmarshalled = convertMapInterfaceToMapString(unmarshalled)
	result, err := json.Marshal(unmarshalled)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON: %s", err)
	}
	return result, nil
}

// convertMapInterfaceToMapString converts a map[interface{}]interface{} to a map[string]interface{}
// This is required for the YAML to JSON conversion as the JSON library does not support interface{} keys
func convertMapInterfaceToMapString(i any) any {
	switch x := i.(type) {
	case map[any]any:
		m2 := map[string]any{}
		for key, value := range x {
			m2[fmt.Sprint(key)] = convertMapInterfaceToMapString(value)
		}
		return m2
	case []any:
		for i, v := range x {
			x[i] = convertMapInterfaceToMapString(v)
		}
	}
	return i
}
