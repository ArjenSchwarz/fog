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

// ReadTemplate parses the provided string and attempts to read the template that it points to.
// In doing so it traverses the defined templates.directory config setting and tries to find a file
// with a name matching the provided string and an extension from the templates.extensions setting.
// Returns the contents of the template, the relative path of the template, and an error
func ReadTemplate(templateName *string) (string, string, error) {
	templateDirectory := viper.GetString("templates.directory")
	for _, extension := range viper.GetStringSlice("templates.extensions") {
		templatePath := templateDirectory + "/" + *templateName + extension
		if _, err := os.Stat(templatePath); !os.IsNotExist(err) {
			dat, err := os.ReadFile(templatePath)
			if err != nil {
				return "", "", err
			}
			return string(dat), templatePath, nil
		}
	}
	return "", "", errors.New("no template found")
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

func ReadTagsfile(tagsName string) (string, error) {
	tagsDirectory := viper.GetString("tags.directory")
	for _, extension := range viper.GetStringSlice("tags.extensions") {
		tagsPath := tagsDirectory + "/" + tagsName + extension
		if _, err := os.Stat(tagsPath); !os.IsNotExist(err) {
			dat, err := os.ReadFile(tagsPath)
			if err != nil {
				return "", err
			}
			return string(dat), nil
		}
	}
	return "", errors.New("no tags file found")
}

func ReadParametersfile(parametersName string) (string, error) {
	parametersDirectory := viper.GetString("parameters.directory")
	for _, extension := range viper.GetStringSlice("parameters.extensions") {
		parametersPath := parametersDirectory + "/" + parametersName + extension
		if _, err := os.Stat(parametersPath); !os.IsNotExist(err) {
			dat, err := os.ReadFile(parametersPath)
			if err != nil {
				return "", err
			}
			return string(dat), nil
		}
	}
	return "", errors.New("no parameters file found")
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
