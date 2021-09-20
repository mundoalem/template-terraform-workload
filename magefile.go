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
	"path"
	"path/filepath"
	"strconv"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	AllEnvironments   []string = []string{"test", "live"}
	InfrastructureDir string   = path.Join(".", "infrastructure")
	LockTimeout       int      = 5
	TestDir           string   = path.Join(".", "test")
	VendorDir         string   = path.Join(".", "vendor")
)

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// FUNCTIONS
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// constains returns whether a string is inside a slice or not
func contains(s []string, el string) bool {
	for _, v := range s {
		if v == el {
			return true
		}
	}

	return false
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// MAGE TARGETS
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Build runs plan for a given environment
func Build(environ string) error {
	var environmentsToBuild []string

	if environ == "all" {
		environmentsToBuild = make([]string, len(AllEnvironments))
		copy(environmentsToBuild, AllEnvironments)
	} else {
		if contains(AllEnvironments, environ) {
			environmentsToBuild = append(environmentsToBuild, environ)
		} else {
			return errors.New("Environment " + environ + " not found")
		}
	}

	for _, env := range environmentsToBuild {
		envPath := path.Join(InfrastructureDir, env)

		args := []string{
			"-chdir=" + envPath,
			"init",
			"-reconfigure",
		}

		if os.Getenv("CI") != "" {
			args = append(args, "-input=false", "-no-color")
		}

		err := sh.RunV("terraform", args...)

		if err != nil {
			return err
		}

		args = []string{
			"-chdir=" + envPath,
			"plan",
			"-lock-timeout=" + strconv.Itoa(LockTimeout) + "s",
		}

		if os.Getenv("CI") != "" {
			args = append(args, "-input=false", "-no-color")
		}

		err = sh.RunV("terraform", args...)

		if err != nil {
			return err
		}
	}

	return nil
}

// Clean removes temporary and build files
func Clean() error {
	filesAndDirsToRemove := []string{
		".terraform",
	}

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if contains(filesAndDirsToRemove, info.Name()) {
			if err := sh.Rm(path); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// Lint checks the project's code for style and syntax issues
func Lint() error {
	pathsToLint := []string{
		InfrastructureDir,
	}

	args := []string{
		"fmt",
		"-recursive",
	}

	if os.Getenv("CI") != "" {
		args = append(args, "-check", "-write=false", "-no-color")
	}

	for _, path := range pathsToLint {
		args = append(args, path)

		if err := sh.RunV("terraform", args...); err != nil {
			return err
		}

		args = args[:len(args)-1]
	}

	return nil
}

// Build runs apply for a given environment
func Release(environ string) error {
	var environmentsToBuild []string

	if environ == "all" {
		environmentsToBuild = make([]string, len(AllEnvironments))
		copy(environmentsToBuild, AllEnvironments)
	} else {
		if contains(AllEnvironments, environ) {
			environmentsToBuild = append(environmentsToBuild, environ)
		} else {
			return errors.New("Environment " + environ + " not found")
		}
	}

	for _, env := range environmentsToBuild {
		envPath := path.Join(InfrastructureDir, env)

		args := []string{
			"-chdir=" + envPath,
			"apply",
			"-lock-timeout=" + strconv.Itoa(LockTimeout) + "s",
		}

		if os.Getenv("CI") != "" {
			args = append(
				args,
				"-auto-approve",
				"-input=false",
				"-no-color",
			)
		}

		err := sh.RunV("terraform", args...)

		if err != nil {
			return err
		}
	}

	return nil
}

// Reset removes all files that Clean does plus the vendor directory
func Reset() error {
	mg.Deps(Clean)

	if err := sh.Rm(VendorDir); err != nil {
		return err
	}

	if err := os.Mkdir(VendorDir, 0755); err != nil {
		return err
	}

	return nil
}

// Scan runs a security check to search for known vulnerabilities in project
func Scan() error {
	_, err := exec.LookPath("tfsec")

	if err != nil {
		return err
	}

	args := []string{
		InfrastructureDir,
		"--verbose",
		"--no-color",
	}

	return sh.RunV("tfsec", args...)
}

// Test runs the unit test for the project
func Test() error {
	args := []string{
		"test",
		"-v",
		"-count=1",
		"./" + TestDir + "/...",
	}

	return sh.RunV("go", args...)
}
