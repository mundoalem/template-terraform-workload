//go:build mage
// +build mage

// This file is part of template-terraform-infrastructure.
//
// template-terraform-infrastructure is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// template-terraform-infrastructure is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with template-terraform-infrastructure. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"text/template"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pterm/pterm"
)

var (
	AllEnvironments   []string = []string{"test", "live"}
	AssetsDir         string   = path.Join(".", "assets")
	BuildDir          string   = path.Join(".", "build")
	FinanceDir        string   = path.Join(AssetsDir, "finance")
	InfrastructureDir string   = path.Join(".", "infrastructure")
	LockTimeout       int      = 5
	TemplatesDir      string   = path.Join(AssetsDir, "templates")
	TestDir           string   = path.Join(".", "test")
	VendorDir         string   = path.Join(".", "vendor")
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// FUNCTIONS
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// calculateInfrastructureCost runs infracost to calculate the cost of the infrastructure to be created by terraform
func calculateInfrastructureCost(planPath string) error {
	args := []string{
		"breakdown",
		"--sync-usage-file",
		"--usage-file=" + path.Join(FinanceDir, "infracost-usage.yml"),
		"--path=" + planPath,
	}

	err := sh.RunV("infracost", args...)

	return err
}

// contains returns whether a string is inside a slice or not
func contains(s []string, el string) bool {
	for _, v := range s {
		if v == el {
			return true
		}
	}

	return false
}

// isCi returns whether we are running in the pipeline or not
func isCi() bool {
	return os.Getenv("CI") != ""
}

// selectEnvironments returns a slice of environment names based on a choice of a selected environment
func selectEnvironments(choice string) ([]string, error) {
	var environments []string

	if choice == "all" {
		environments = make([]string, len(AllEnvironments))
		copy(environments, AllEnvironments)
	} else {
		if contains(AllEnvironments, choice) {
			environments = append(environments, choice)
		} else {
			return nil, errors.New("Environment " + choice + " is not valid")
		}
	}

	return environments, nil
}

// tfApply runs terraform apply in the specified path
func tfApply(path string) error {
	args := []string{
		"-chdir=" + path,
		"apply",
		"-lock-timeout=" + strconv.Itoa(LockTimeout) + "s",
	}

	if isCi() {
		args = append(
			args,
			"-auto-approve",
			"-input=false",
		)
	}

	err := sh.RunV("terraform", args...)

	return err
}

// tfInit runs terraform init inside the specified path
func tfInit(path string) error {
	args := []string{
		"-chdir=" + path,
		"init",
		"-reconfigure",
	}

	if isCi() {
		args = append(args, "-input=false")
	}

	err := sh.RunV("terraform", args...)

	return err
}

// tfLint lints the terraform code in the specified path
func tfLint(path string) error {
	args := []string{
		"fmt",
		"-recursive",
	}

	if isCi() {
		args = append(args, "-check", "-write=false", "-no-color")
	}

	args = append(args, path)
	err := sh.RunV("terraform", args...)

	return err
}

// tfPlan runs terraform plan in the specified path
func tfPlan(path string) error {
	args := []string{
		"-chdir=" + path,
		"plan",
		"-lock-timeout=" + strconv.Itoa(LockTimeout) + "s",
	}

	if isCi() {
		args = append(args, "-input=false")
	}

	err := sh.RunV("terraform", args...)

	return err
}

// tfSavePlan exports the plan as a JSON file and save it locally
func tfSavePlan(path string, planPath string) error {
	args := []string{
		"-chdir=" + path,
		"show",
		"-json",
		"-no-color",
	}

	plan, err := sh.Output("terraform", args...)

	if err != nil {
		return err
	}

	f, err := os.Create(planPath)

	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteString(plan)

	if err != nil {
		return err
	}

	f.Sync()

	return nil
}

// tfSec scans the terraform files in the specified path for security vulnerabilities
func tfSec(path string) error {
	args := []string{
		path,
		"--verbose",
		"--no-color",
	}

	err := sh.RunV("tfsec", args...)

	return err
}

// tfTest runs the terratest files in the specified test path
func tfTest(path string) error {
	args := []string{
		"test",
		"-v",
		"-count=1",
		"./" + path + "/...",
	}

	err := sh.RunV("go", args...)

	return err
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// MAGE TARGETS
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Build plans the release for a given environment
func Build(environ string) error {
	pterm.Info.Println("Build process started")

	_, err := exec.LookPath("infracost")

	if err != nil {
		pterm.Error.Printfln("Could not find 'infracost' command: ", err)
		return err
	}

	environments, err := selectEnvironments(environ)

	if err != nil {
		pterm.Error.Printfln("Could not determine environments to build: ", err)
		return err
	}

	buildReport := make([]pterm.BulletListItem, 0)
	flagError := false

	for _, env := range environments {
		envPath := path.Join(InfrastructureDir, env)
		buildReportItem := pterm.NewBulletListItemFromString(env, "")

		err = tfInit(envPath)

		if err != nil {
			buildReportItem.TextStyle = pterm.NewStyle(pterm.FgRed)
			buildReportItem.BulletStyle = pterm.NewStyle(pterm.FgRed)

			pterm.Error.Printfln("Could not initialize environment '", env, "': ", err)

			flagError = true
		}

		err = tfPlan(envPath)

		if err != nil {
			buildReportItem.TextStyle = pterm.NewStyle(pterm.FgRed)
			buildReportItem.BulletStyle = pterm.NewStyle(pterm.FgRed)

			pterm.Error.Printfln("Could not plan environment '", env, "': ", err)

			flagError = true
		}

		planPath := path.Join(BuildDir, env+".plan")

		err = tfSavePlan(envPath, planPath)

		if err != nil {
			buildReportItem.TextStyle = pterm.NewStyle(pterm.FgRed)
			buildReportItem.BulletStyle = pterm.NewStyle(pterm.FgRed)

			pterm.Error.Printfln("Could not save plan for environment '", env, "': ", err)

			flagError = true
		}

		err = calculateInfrastructureCost(planPath)

		if err != nil {
			buildReportItem.TextStyle = pterm.NewStyle(pterm.FgRed)
			buildReportItem.BulletStyle = pterm.NewStyle(pterm.FgRed)

			pterm.Error.Printfln("Could not calculate cost of environment '", env,"': ", err)

			flagError = true
		}

		buildReport = append(buildReport, buildReportItem)
		
		pterm.Info.Println("Build process completed for environment '", env, "'")
	}

	pterm.DefaultSection.Println("Build")
	pterm.DefaultBulletList.WithItems(buildReport).Render()

	if flagError {
		pterm.Error.Println("Build process completed with errors")
		return errors.New("Process failed")
	}

	pterm.Success.Println("Build process completed")

	return nil
}

// Clean removes temporary files created by other processes
func Clean() error {
	pterm.Info.Println("Clean process started")

	cleanReport := make([]pterm.BulletListItem, 0)
	environments, err := selectEnvironments("all")

	if err != nil {
		pterm.Error.Println("Could not determine the environments: ", err)
		return errors.New("Process failed")
	}

	for _, env := range environments {
		planFile := path.Join(BuildDir, env+".plan")

		if _, err := os.Stat(planFile); err == nil {
			cleanReport = append(
				cleanReport,
				pterm.NewBulletListItemFromString(planFile, ""),
			)
		}

		terraformDir := path.Join(InfrastructureDir, env, ".terraform")

		if _, err := os.Stat(terraformDir); err == nil {
			cleanReport = append(
				cleanReport,
				pterm.NewBulletListItemFromString(terraformDir, ""),
			)
		}
	}

	flagError := false

	if len(cleanReport) <= 0 {
		pterm.Info.Println("Nothing to clean")
	} else {
		pterm.DefaultSection.Println("Removed")

		for i := range cleanReport {
			path := cleanReport[i].Text
			err := sh.Rm(path)

			if err != nil {
				flagError = true

				cleanReport[i].TextStyle = pterm.NewStyle(pterm.FgRed)
				cleanReport[i].BulletStyle = pterm.NewStyle(pterm.FgRed)

				pterm.Error.Printfln("Could not remove '", path, "': ", err)
			}
		}

		pterm.DefaultBulletList.WithItems(cleanReport).Render()
	}

	if flagError {
		pterm.Error.Println("Clean process completed with errors")
		return errors.New("Process failed")
	}

	pterm.Success.Println("Clean process completed")

	return nil
}

// Config sets up the required configuration files to run the pipeline
func Config() error {
	pterm.Info.Println("Config process started")

	currentUser, err := user.Current()

	if err != nil {
		pterm.Error.Println("Could not determine current user: ", err)
		return errors.New("Process failed")
	}

	terraformConfigDir := path.Join(currentUser.HomeDir, ".terraform.d")

	pterm.Info.Println("Creating directory: ", terraformConfigDir)

	err = os.MkdirAll(terraformConfigDir, os.ModePerm)

	if err != nil {
		pterm.Error.Println("Could not create directory '", terraformConfigDir, "': %s", err)
		return errors.New("Process failed")
	}

	terraformConfigPath := path.Join(terraformConfigDir, "credentials.tfrc.json")

	pterm.Info.Println("Creating configuration file: ", terraformConfigPath)	

	if _, err := os.Stat(terraformConfigPath); err == nil {
		pterm.Error.Println("Terraform configuration file '", terraformConfigPath, "' already exists: ", err)
		return errors.New("Process failed")
	}

	f, err := os.Create(terraformConfigPath)

	if err != nil {
		pterm.Error.Println("Could not create file '", terraformConfigPath,"': ", err)
		return errors.New("Process failed")
	}

	defer f.Close()
	token := os.Getenv("TF_CREDENTIALS")

	if token == "" {
		pterm.Error.Println("Terraform remote backend token not found in environment")
		return errors.New("Process failed")
	}

	tmpl := template.Must(
		template.New("credentials.tfrc.json.tmpl").ParseFiles(
			path.Join(TemplatesDir, "credentials.tfrc.json.tmpl"),
		),
	)

	err = tmpl.Execute(f, struct {
		Token string
	}{
		token,
	})

	if err != nil {
		pterm.Error.Println("Could not configure terraform backend from template: ", err)
		return errors.New("Process failed")
	}

	pterm.Success.Println("Config process completed")

	return nil
}

// Lint checks the source code for style and syntax issues
func Lint() error {
	pterm.Info.Println("Lint process started")

	pathsToLint := []string{
		InfrastructureDir,
	}

	lintReport := make([]pterm.BulletListItem, 0)
	flagError := false

	for _, path := range pathsToLint {
		lintReportItem := pterm.NewBulletListItemFromString(path, "")
		err := tfLint(path)

		if err != nil {
			lintReportItem.TextStyle = pterm.NewStyle(pterm.FgRed)
			lintReportItem.BulletStyle = pterm.NewStyle(pterm.FgRed)

			pterm.Error.Printfln("Could not lint environment '", path, "': ", err)

			flagError = true
		}

		lintReport = append(lintReport, lintReportItem)
	}

	pterm.DefaultSection.Println("Lint")
	pterm.DefaultBulletList.WithItems(lintReport).Render()

	if flagError {
		pterm.Error.Println("Lint process completed with errors")
		return errors.New("Process failed")
	}

	pterm.Success.Println("Lint process completed")

	return nil
}

// Release applies the configuration for a given environment
func Release(environ string) error {
	pterm.Info.Println("Release process started")

	environments, err := selectEnvironments(environ)

	if err != nil {
		pterm.Error.Printfln("Could not determine environments to build: ", err)
		return err
	}

	releaseReport := make([]pterm.BulletListItem, 0)
	flagError := false

	for _, env := range environments {
		envPath := path.Join(InfrastructureDir, env)
		releaseReportItem := pterm.NewBulletListItemFromString(env, "")

		err = tfInit(envPath)

		if err != nil {
			releaseReportItem.TextStyle = pterm.NewStyle(pterm.FgRed)
			releaseReportItem.BulletStyle = pterm.NewStyle(pterm.FgRed)

			pterm.Error.Printfln("Could not initialize environment '", env, "': ", err)

			flagError = true
		}

		err = tfApply(envPath)

		if err != nil {
			releaseReportItem.TextStyle = pterm.NewStyle(pterm.FgRed)
			releaseReportItem.BulletStyle = pterm.NewStyle(pterm.FgRed)

			pterm.Error.Printfln("Could not release environment '", env, "': ", err)

			flagError = true
		}

		releaseReport = append(releaseReport, releaseReportItem)
	}

	pterm.DefaultSection.Println("Release")
	pterm.DefaultBulletList.WithItems(releaseReport).Render()

	if flagError {
		pterm.Error.Println("Release process completed with errors")
		return errors.New("Process failed")
	}

	pterm.Success.Println("Release process completed")

	return nil
}

// Reset removes all files that Clean does plus the vendor directory
func Reset() error {
	mg.Deps(Clean)

	pterm.Info.Println("Reset process started")
	pterm.Info.Println("Removing installed dependencies in directory: ", VendorDir)

	if err := sh.Rm(VendorDir); err != nil {
		pterm.Error.Println("Could not remove directory '", VendorDir, "': ", err)
		return errors.New("Process failed")
	}

	pterm.Info.Println("Creating empty dependencies directory: ", VendorDir)

	if err := os.Mkdir(VendorDir, 0755); err != nil {
		pterm.Error.Println("Could not create directory '", VendorDir, "': ", err)
		return errors.New("Process failed")
	}

	pterm.Success.Println("Reset process completed")

	return nil
}

// Scan runs a security check to search for known vulnerabilities in the project
func Scan() error {
	pterm.Info.Println("Scan process started")

	_, err := exec.LookPath("tfsec")

	if err != nil {
		pterm.Error.Println("Could not find tfsec command: %s", err)
		return errors.New("Process failed")
	}

	pterm.Info.Println("Scanning directory: ", InfrastructureDir)

	err = tfSec(InfrastructureDir)
	
	if err != nil {
		pterm.Error.Println("Security vulnerabilities found: ", err)
		return errors.New("Process failed")
	}

	pterm.Success.Println("Scan process completed")

	return nil
}

// Test runs the unit test for the project
func Test() error {
	pterm.Info.Println("Test process started")

	err := tfTest(TestDir)

	if err != nil {
		pterm.Error.Println("Test process failed: ", err)
		return errors.New("Process failed")
	}

	pterm.Success.Println("Test process completed")

	return nil
}
