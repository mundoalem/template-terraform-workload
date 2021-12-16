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

module "echo" {
  source  = "mundoalem/echo/null"
  version = "1.0.2"
  message = "Hello, ${title(var.environment)} environment!"
}
