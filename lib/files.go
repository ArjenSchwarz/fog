package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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
		for _, extension := range viper.GetStringSlice(fileType + ".extensions") {
			filePath = fileDirectory + "/" + *fileName + extension
			if _, err := os.Stat(filePath); !os.IsNotExist(err) {
				fileFound = true
				break
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
func UploadTemplate(templateName *string, template string, bucketName *string, svc S3UploadAPI) (string, error) {
	// use the template name with a timestamp that should be unique
	// prefix with fog to make it easier to set up specific lifecycle rules
	generatedname := fmt.Sprintf("fog/%v-%v", *templateName, time.Now().UnixNano())
	input := s3.PutObjectInput{
		Bucket: bucketName,
		Key:    aws.String(generatedname),
		Body:   strings.NewReader(template),
	}
	_, err := svc.PutObject(context.TODO(), &input)
	if err != nil {
		return generatedname, err
	}
	return generatedname, nil
}

// RunPrechecks executes configured template validation commands and returns results for each check.
func RunPrechecks(deployment *DeployInfo) (map[string]string, error) {
	results := make(map[string]string)
	for _, precheck := range viper.GetStringSlice("templates.prechecks") {
		precheck := strings.ReplaceAll(precheck, "$TEMPLATEPATH", deployment.TemplateRelativePath)
		separated := strings.Split(precheck, " ")
		command, args := separated[0], separated[1:]
		//TODO: improve on this list or find a better solution to keep it safe
		if stringInSlice(command, []string{"rm", "del", "kill"}) {
			return results, fmt.Errorf("unsafe command '%v' detected", command)
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
		err := cmd.Run()
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
