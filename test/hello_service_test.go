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

package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestHelloService(t *testing.T) {
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../infrastructure/test",
		Targets: []string{
			"module.hello_service",
		},
	})

	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	outputsHelloService := terraform.OutputListOfObjects(t, terraformOptions, "hello_service_outputs")
	outputMessage := outputsHelloService[0]["message"]

	assert.Equal(t, "Hello, Test environment!", outputMessage)
}
