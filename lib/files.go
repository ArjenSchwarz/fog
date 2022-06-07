package lib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

// Locate and read the file. Either it's an actual file name in which case
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
			errMsg := fmt.Sprintf("No file found for '%s' matching '%s'", fileType, *fileName)
			return "", "", errors.New(errMsg)
		}
	}
	msg := fmt.Sprintf("Using %s: %s", fileType, filePath)
	fmt.Fprintln(os.Stderr, msg)
	dat, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}
	return string(dat), filePath, nil
}

func ReadTemplate(templateName *string) (string, string, error) {
	return ReadFile(templateName, "templates")
}

func ReadTagsfile(tagsName string) (string, string, error) {
	return ReadFile(&tagsName, "tags")
}

func ReadParametersfile(parametersName string) (string, string, error) {
	return ReadFile(&parametersName, "parameters")
}

func UploadTemplate(templateName *string, template string, bucketName *string, svc *s3.Client) (string, error) {
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

func RunPrechecks(deployment *DeployInfo) (map[string]string, error) {
	results := make(map[string]string)
	for _, precheck := range viper.GetStringSlice("templates.prechecks") {
		precheck := strings.Replace(precheck, "$TEMPLATEPATH", deployment.TemplateRelativePath, -1)
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
