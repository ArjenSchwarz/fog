/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/ArjenSchwarz/fog/lib"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// terraformCmd represents the terraform command
var terraformCmd = &cobra.Command{
	Use:   "terraform",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: terraform,
}

func init() {
	rootCmd.AddCommand(terraformCmd)

}

func terraform(cmd *cobra.Command, args []string) {
	binary := "/opt/Homebrew/bin/terraform"
	planName := placeholderParser(viper.GetString("changeset.name-format"), nil)
	planPath := fmt.Sprintf("/tmp/%s.plan", planName)
	args = []string{"plan", fmt.Sprintf("--out=%s", planPath)}
	cmd1 := exec.Command(binary, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd1.Stdout = &out
	cmd1.Stderr = &stderr
	err := cmd1.Run()
	if err != nil {
		log.Fatal(err)
	}

	args = []string{"show", "--json", planPath}
	cmd2 := exec.Command(binary, args...)
	var out2 bytes.Buffer
	cmd2.Stdout = &out2
	cmd2.Stderr = &stderr
	err = cmd2.Run()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println("Testing testing testing --")
	plan := lib.TerraformPlan{}
	err = json.Unmarshal(out2.Bytes(), &plan)
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Printf("%s", plan)
	title := fmt.Sprintf("Terraform plan summary")
	keys := []string{"Action", "Name in Terraform", "Type", "Resource Name"}
	if settings.GetBool("verbose") {
		keys = append(keys, []string{"ProviderName", "Mode"}...)
	}
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = title
	output.Settings.SortKey = "Type"
	bold := color.New(color.Bold).SprintFunc()
	for _, resource := range plan.ResourceChanges {
		if resource.HasChange() {
			content := make(map[string]interface{})
			content["Type"] = resource.Type
			content["Name in Terraform"] = resource.Name
			actions := strings.Join(resource.Change.Actions, ", ")
			if strings.Contains(actions, "delete") {
				actions = bold(actions)
			}
			content["Action"] = actions
			content["Resource Name"] = resource.GetName()
			if settings.GetBool("verbose") {
				content["ProviderName"] = resource.ProviderName
				content["Mode"] = resource.Mode
			}
			holder := format.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
	}
	output.Write()

}
