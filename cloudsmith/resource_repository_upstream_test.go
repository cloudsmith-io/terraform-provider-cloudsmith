//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccRepositoryUpstreamDart_basic(t *testing.T) {
	t.Parallel()

	const dartUpstreamResourceName = "cloudsmith_repository_upstream.pub_dev"

	testAccRepositoryPythonUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-dart"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "pub_dev" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "dart"
    upstream_url  = "https://pub.dev"
}
`, namespace)

	testAccRepositoryPythonUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-dart"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "pub_dev" {
		extra_header_1 = "Cross-Origin-Resource-Policy"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "cross-origin"
	    extra_value_2  = "*"
	    is_active      = true
	    mode           = "Proxy Only"
		name           = cloudsmith_repository.test.name
	    namespace      = cloudsmith_repository.test.namespace
	    priority       = 12345
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "dart"
	    upstream_url   = "https://pub.dev"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(dartUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryPythonUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dartUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(dartUpstreamResourceName, AuthSecret, ""),
					resource.TestCheckResourceAttr(dartUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(dartUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(dartUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(dartUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(dartUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(dartUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(dartUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(dartUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(dartUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(dartUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(dartUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(dartUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(dartUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(dartUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(dartUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(dartUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(dartUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryPythonUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(dartUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(dartUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(dartUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(dartUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(dartUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(dartUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(dartUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(dartUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: dartUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[dartUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRepositoryUpstreamDeb_basic(t *testing.T) {
	t.Parallel()

	const debUpstreamResourceName = "cloudsmith_repository_upstream.ubuntu"

	var testAccRepositoryDebUpstreamConfigBasic = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-deb"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "ubuntu" {
    distro_versions = ["ubuntu/trusty", "ubuntu/xenial"]
    namespace       = cloudsmith_repository.test.namespace
    repository      = cloudsmith_repository.test.slug
	name            = cloudsmith_repository.test.name
    upstream_type   = "deb"
    upstream_url    = "http://archive.ubuntu.com/ubuntu"
}
`, namespace)

	var testAccRepositoryDebUpstreamConfigUpdate = fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-deb"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "ubuntu" {
	    component             = "main"
	    distro_versions       = ["ubuntu/focal"]
		extra_header_1        = "Cross-Origin-Resource-Policy"
	    extra_header_2        = "Access-Control-Allow-Origin"
	    extra_value_1         = "cross-origin"
	    extra_value_2         = "*"
	    include_sources       = true
	    is_active             = true
	    mode                  = "Cache and Proxy"
		name                  = cloudsmith_repository.test.name
	    namespace             = cloudsmith_repository.test.namespace
	    priority              = 12345
	    repository            = cloudsmith_repository.test.slug
		upstream_distribution = "focal"
	    upstream_type         = "deb"
	    upstream_url          = "http://archive.ubuntu.com/ubuntu"
	    verify_ssl            = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(debUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryDebUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(debUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(debUpstreamResourceName, AuthSecret, ""),
					resource.TestCheckResourceAttr(debUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckResourceAttrSet(debUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(debUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(debUpstreamResourceName, DistroVersion),
					resource.TestCheckResourceAttr(debUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(debUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(debUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(debUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckResourceAttr(debUpstreamResourceName, IncludeSources, "false"),
					resource.TestCheckResourceAttr(debUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(debUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(debUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(debUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(debUpstreamResourceName, UpdatedAt),
					resource.TestCheckResourceAttr(debUpstreamResourceName, UpstreamDistribution, ""),
					resource.TestCheckResourceAttr(debUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryDebUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(debUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(debUpstreamResourceName, DistroVersion),
					resource.TestCheckResourceAttrSet(debUpstreamResourceName, UpdatedAt),
					resource.TestCheckResourceAttr(debUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: debUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[debUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRepositoryUpstreamDocker_basic(t *testing.T) {
	t.Parallel()

	const dockerUpstreamResourceName = "cloudsmith_repository_upstream.dockerhub"

	testAccRepositoryPythonUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-docker"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "dockerhub" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "docker"
    upstream_url  = "https://index.docker.io"
}
`, namespace)

	testAccRepositoryPythonUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-docker"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "dockerhub" {
		auth_mode      = "Username and Password"
	    auth_secret    = "SuperSecretPassword123!"
	    auth_username  = "jonny.tables"
		extra_header_1 = "Cross-Origin-Resource-Policy"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "cross-origin"
	    extra_value_2  = "*"
	    is_active      = false
	    mode           = "Cache and Proxy"
		name           = cloudsmith_repository.test.name
	    namespace      = cloudsmith_repository.test.namespace
	    priority       = 12345
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "docker"
	    upstream_url   = "https://index.docker.io"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(dockerUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryPythonUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, AuthSecret, ""),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(dockerUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(dockerUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(dockerUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(dockerUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryPythonUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(dockerUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(dockerUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, IsActive, "false"),
				),
			},
			{
				ResourceName: dockerUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[dockerUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRepositoryUpstreamHelm_basic(t *testing.T) {
	t.Parallel()

	const helmUpstreamResourceName = "cloudsmith_repository_upstream.helm"

	testAccRepositoryPythonUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-helm"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "helm" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "helm"
    upstream_url  = "https://charts.helm.sh/stable"
}
`, namespace)

	testAccRepositoryPythonUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-helm"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "helm" {
		auth_mode      = "Username and Password"
	    auth_secret    = "SuperSecretPassword123!"
	    auth_username  = "jonny.tables"
		extra_header_1 = "Cross-Origin-Resource-Policy"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "cross-origin"
	    extra_value_2  = "*"
	    is_active      = true
	    mode           = "Cache and Proxy"
		name           = cloudsmith_repository.test.name
	    namespace      = cloudsmith_repository.test.namespace
	    priority       = 12345
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "helm"
	    upstream_url   = "https://charts.helm.sh/stable"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(helmUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryPythonUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(helmUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(helmUpstreamResourceName, AuthSecret, ""),
					resource.TestCheckResourceAttr(helmUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(helmUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(helmUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(helmUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(helmUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(helmUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(helmUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(helmUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(helmUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(helmUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(helmUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(helmUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(helmUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(helmUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(helmUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(helmUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(helmUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryPythonUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(helmUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(helmUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(helmUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(helmUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(helmUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(helmUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(helmUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(helmUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: helmUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[helmUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRepositoryUpstreamMaven_basic(t *testing.T) {
	t.Parallel()

	const mavenUpstreamResourceName = "cloudsmith_repository_upstream.maven_central"

	testAccRepositoryPythonUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-maven"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "maven_central" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "maven"
    upstream_url  = "https://repo1.maven.org/maven2"
}
`, namespace)

	testAccRepositoryPythonUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-python"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "maven_central" {
		extra_header_1 = "Cross-Origin-Resource-Policy"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "cross-origin"
	    extra_value_2  = "*"
	    is_active      = true
	    mode           = "Cache and Proxy"
		name           = cloudsmith_repository.test.name
	    namespace      = cloudsmith_repository.test.namespace
	    priority       = 12345
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "maven"
	    upstream_url   = "https://repo1.maven.org/maven2"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(mavenUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryPythonUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, AuthSecret, ""),
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(mavenUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(mavenUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(mavenUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(mavenUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(mavenUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(mavenUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(mavenUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(mavenUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(mavenUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryPythonUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(mavenUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(mavenUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(mavenUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(mavenUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(mavenUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(mavenUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(mavenUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(mavenUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: mavenUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[mavenUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRepositoryUpstreamNpm_basic(t *testing.T) {
	t.Parallel()

	const npmUpstreamResourceName = "cloudsmith_repository_upstream.npmjs"

	testAccRepositoryPythonUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-npm"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "npmjs" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "npm"
    upstream_url  = "https://registry.npmjs.org"
    is_active      = true

}
`, namespace)

	testAccRepositoryPythonUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-npm"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "npmjs" {
		extra_header_1 = "Cross-Origin-Resource-Policy"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "cross-origin"
	    extra_value_2  = "*"
	    is_active      = true
	    mode           = "Cache and Proxy"
		name           = cloudsmith_repository.test.name
	    namespace      = cloudsmith_repository.test.namespace
	    priority       = 12345
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "npm"
	    upstream_url   = "https://registry.npmjs.org"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(npmUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryPythonUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(npmUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(npmUpstreamResourceName, AuthSecret, ""),
					resource.TestCheckResourceAttr(npmUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(npmUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(npmUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(npmUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(npmUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(npmUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(npmUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(npmUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(npmUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(npmUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(npmUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(npmUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(npmUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(npmUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(npmUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(npmUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(npmUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryPythonUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(npmUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(npmUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(npmUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(npmUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(npmUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(npmUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(npmUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(npmUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: npmUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[npmUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRepositoryUpstreamNuget_basic(t *testing.T) {
	t.Parallel()

	const nugetUpstreamResourceName = "cloudsmith_repository_upstream.nuget"

	testAccRepositoryPythonUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-nuget"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "nuget" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "nuget"
    upstream_url  = "https://api.nuget.org/v3/index.json"
}
`, namespace)

	testAccRepositoryPythonUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-nuget"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "nuget" {
		extra_header_1 = "Cross-Origin-Resource-Policy"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "cross-origin"
	    extra_value_2  = "*"
	    is_active      = true
	    mode           = "Cache and Proxy"
		name           = cloudsmith_repository.test.name
	    namespace      = cloudsmith_repository.test.namespace
	    priority       = 12345
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "nuget"
	    upstream_url   = "https://api.nuget.org/v3/index.json"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(nugetUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryPythonUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, AuthSecret, ""),
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(nugetUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(nugetUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(nugetUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(nugetUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(nugetUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(nugetUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(nugetUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(nugetUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(nugetUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryPythonUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(nugetUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(nugetUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(nugetUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(nugetUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(nugetUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(nugetUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(nugetUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(nugetUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: nugetUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[nugetUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRepositoryUpstreamPython_basic(t *testing.T) {
	t.Parallel()

	const pythonUpstreamResourceName = "cloudsmith_repository_upstream.pypi"

	testAccRepositoryPythonUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-python"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "pypi" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "python"
    upstream_url  = "https://pypi.org"
}
`, namespace)

	testAccRepositoryPythonUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-python"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "pypi" {
		auth_mode      = "Username and Password"
	    auth_secret    = "SuperSecretPassword123!"
	    auth_username  = "jonny.tables"
		extra_header_1 = "Cross-Origin-Resource-Policy"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "cross-origin"
	    extra_value_2  = "*"
	    is_active      = false
	    mode           = "Cache and Proxy"
		name           = cloudsmith_repository.test.name
	    namespace      = cloudsmith_repository.test.namespace
	    priority       = 12345
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "python"
	    upstream_url   = "https://pypi.org"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(pythonUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryPythonUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, AuthSecret, ""),
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(pythonUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(pythonUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(pythonUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(pythonUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(pythonUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(pythonUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(pythonUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(pythonUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(pythonUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryPythonUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(pythonUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(pythonUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(pythonUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(pythonUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(pythonUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(pythonUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(pythonUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(pythonUpstreamResourceName, IsActive, "false"),
				),
			},
			{
				ResourceName: pythonUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[pythonUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRepositoryUpstreamRpm_basic(t *testing.T) {
	t.Parallel()

	const rpmUpstreamResourceName = "cloudsmith_repository_upstream.rpm_fusion"

	var testAccRepositoryRpmUpstreamConfigBasic = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-rpm"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "rpm_fusion" {
    distro_version = "fedora/35"
    namespace      = cloudsmith_repository.test.namespace
    repository     = cloudsmith_repository.test.slug
	name           = cloudsmith_repository.test.name
    upstream_type  = "rpm"
    upstream_url   = "https://download1.rpmfusion.org/free/fedora/releases/35/Everything/x86_64/os"
	is_active      = false
}
`, namespace)

	var testAccRepositoryRpmUpstreamConfigUpdate = fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-rpm"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "rpm_fusion" {
		auth_mode       = "Username and Password"
	    auth_secret     = "SuperSecretPassword123!"
	    auth_username   = "jonny.tables"
	    distro_version  = "fedora/35"
		extra_header_1  = "Cross-Origin-Resource-Policy"
	    extra_header_2  = "Access-Control-Allow-Origin"
	    extra_value_1   = "cross-origin"
	    extra_value_2   = "*"
	    include_sources = true
	    is_active       = false
	    mode            = "Cache and Proxy"
		name            = cloudsmith_repository.test.name
	    namespace       = cloudsmith_repository.test.namespace
	    priority        = 12345
	    repository      = cloudsmith_repository.test.slug
	    upstream_type   = "rpm"
	    upstream_url    = "https://download1.rpmfusion.org/free/fedora/releases/35/Everything/x86_64/os"
	    verify_ssl      = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(rpmUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryRpmUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, AuthSecret, ""),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(rpmUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(rpmUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(rpmUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, IncludeSources, "false"),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, IsActive, "false"),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(rpmUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(rpmUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(rpmUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(rpmUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryRpmUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(rpmUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(rpmUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(rpmUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttrSet(rpmUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(rpmUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(rpmUpstreamResourceName, IsActive, "false"),
				),
			},
			{
				ResourceName: rpmUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[rpmUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRepositoryUpstreamRuby_basic(t *testing.T) {
	t.Parallel()

	const rubyUpstreamResourceName = "cloudsmith_repository_upstream.rubygems"

	testAccRepositoryPythonUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-ruby"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "rubygems" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "ruby"
    upstream_url  = "https://rubygems.org"
}
`, namespace)

	testAccRepositoryPythonUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-ruby"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "rubygems" {
		auth_mode      = "Username and Password"
	    auth_secret    = "SuperSecretPassword123!"
	    auth_username  = "jonny.tables"
		extra_header_1 = "Cross-Origin-Resource-Policy"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "cross-origin"
	    extra_value_2  = "*"
	    is_active      = true
	    mode           = "Cache and Proxy"
		name           = cloudsmith_repository.test.name
	    namespace      = cloudsmith_repository.test.namespace
	    priority       = 12345
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "ruby"
	    upstream_url   = "https://rubygems.org"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(rubyUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryPythonUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, AuthSecret, ""),
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(rubyUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(rubyUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(rubyUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(rubyUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(rubyUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(rubyUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(rubyUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(rubyUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(rubyUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryPythonUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(rubyUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(rubyUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(rubyUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(rubyUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(rubyUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(rubyUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(rubyUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(rubyUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: rubyUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[rubyUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRepositoryUpstreamCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		namespace := resourceState.Primary.Attributes[Namespace]
		repository := resourceState.Primary.Attributes[Repository]
		upstreamType := resourceState.Primary.Attributes[UpstreamType]
		slugPerm := resourceState.Primary.Attributes[SlugPerm]

		var resp *http.Response
		var err error

		switch upstreamType {
		case Cran:
			req := pc.APIClient.ReposApi.ReposUpstreamCranRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamCranReadExecute(req)
		case Dart:
			req := pc.APIClient.ReposApi.ReposUpstreamDartRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamDartReadExecute(req)
		case Deb:
			req := pc.APIClient.ReposApi.ReposUpstreamDebRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamDebReadExecute(req)
		case Docker:
			req := pc.APIClient.ReposApi.ReposUpstreamDockerRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamDockerReadExecute(req)
		case Helm:
			req := pc.APIClient.ReposApi.ReposUpstreamHelmRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamHelmReadExecute(req)
		case Maven:
			req := pc.APIClient.ReposApi.ReposUpstreamMavenRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamMavenReadExecute(req)
		case Npm:
			req := pc.APIClient.ReposApi.ReposUpstreamNpmRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamNpmReadExecute(req)
		case NuGet:
			req := pc.APIClient.ReposApi.ReposUpstreamNugetRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamNugetReadExecute(req)
		case Python:
			req := pc.APIClient.ReposApi.ReposUpstreamPythonRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamPythonReadExecute(req)
		case Rpm:
			req := pc.APIClient.ReposApi.ReposUpstreamRpmRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamRpmReadExecute(req)
		case Ruby:
			req := pc.APIClient.ReposApi.ReposUpstreamRubyRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamRubyReadExecute(req)
		case Swift:
			req := pc.APIClient.ReposApi.ReposUpstreamSwiftRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamSwiftReadExecute(req)
		default:
			err = fmt.Errorf("invalid upstream_type: '%s'", upstreamType)
		}

		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify upstream deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify upstream deletion: still exists: %s", resourceName)
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		return nil
	}
}
