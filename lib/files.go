package lib

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"
)

func ReadTemplate(templateName *string) (string, error) {
	templateDirectory := viper.GetString("templates.directory")
	for _, extension := range viper.GetStringSlice("templates.extensions") {
		templatePath := templateDirectory + "/" + *templateName + extension
		if _, err := os.Stat(templatePath); !os.IsNotExist(err) {
			dat, err := ioutil.ReadFile(templatePath)
			if err != nil {
				return "", err
			}
			return string(dat), nil
		}
	}
	return "", errors.New("no template found")
}

func ReadTagsfile(tagsName *string) (string, error) {
	tagsDirectory := viper.GetString("tags.directory")
	for _, extension := range viper.GetStringSlice("tags.extensions") {
		tagsPath := tagsDirectory + "/" + *tagsName + extension
		if _, err := os.Stat(tagsPath); !os.IsNotExist(err) {
			dat, err := ioutil.ReadFile(tagsPath)
			if err != nil {
				return "", err
			}
			return string(dat), nil
		}
	}
	return "", errors.New("no tags file found")
}

func ReadParametersfile(parametersName *string) (string, error) {
	parametersDirectory := viper.GetString("parameters.directory")
	for _, extension := range viper.GetStringSlice("parameters.extensions") {
		parametersPath := parametersDirectory + "/" + *parametersName + extension
		if _, err := os.Stat(parametersPath); !os.IsNotExist(err) {
			dat, err := ioutil.ReadFile(parametersPath)
			if err != nil {
				return "", err
			}
			return string(dat), nil
		}
	}
	return "", errors.New("no parameters file found")
}
