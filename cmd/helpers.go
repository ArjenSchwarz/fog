package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// askForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user.
func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("")
		fmt.Printf("🔔 %s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			// Continue to ask again
		}
	}
}

func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, ok := keys[entry]; !ok {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// stringInSlice checks if a string exists in a slice
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func stringValueInMap(a string, list map[string]string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// addToField increases the integer value of the field by the provided value
func addToField(field *map[string]interface{}, key string, value int) {
	(*field)[key] = (*field)[key].(int) + value
}

func failWithError(err error) {
	fmt.Print(settings.NewOutputSettings().StringFailure(fmt.Sprintf("Error: %v", err)))
	if viper.GetBool("debug") {
		panic(err)
	}
	os.Exit(1)
}
