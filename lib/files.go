package lib

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

func ReadTemplate(templateName *string) (string, error) {
	templateDirectory := viper.GetString("templates.directory")
	for _, extension := range viper.GetStringSlice("templates.extensions") {
		templatePath := templateDirectory + "/" + *templateName + extension
		if _, err := os.Stat(templatePath); !os.IsNotExist(err) {
			dat, err := os.ReadFile(templatePath)
			if err != nil {
				return "", err
			}
			return string(dat), nil
		}
	}
	return "", errors.New("no template found")
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
