//nolint:testpackage
package cloudsmith

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccRepositoryUpstreamCargo_basic(t *testing.T) {
	t.Parallel()

	const cargoUpstreamResourceName = "cloudsmith_repository_upstream.crates_io"

	testAccRepositoryCargoUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-cargo"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "crates_io" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "cargo"
    upstream_url  = "https://index.crates.io"
}
`, namespace)

	testAccRepositoryCargoUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-cargo"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "crates_io" {
		extra_header_1 = "X-Custom-Header"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "custom-value"
	    extra_value_2  = "*"
	    is_active      = true
	    mode           = "Proxy Only"
		name           = cloudsmith_repository.test.name
	    namespace      = cloudsmith_repository.test.namespace
	    priority       = 12345
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "cargo"
	    upstream_url   = "https://index.crates.io"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(cargoUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryCargoUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cargoUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(cargoUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(cargoUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(cargoUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(cargoUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(cargoUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(cargoUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(cargoUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(cargoUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(cargoUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(cargoUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(cargoUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(cargoUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(cargoUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(cargoUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(cargoUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(cargoUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(cargoUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryCargoUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(cargoUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(cargoUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(cargoUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(cargoUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(cargoUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(cargoUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(cargoUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(cargoUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: cargoUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[cargoUpstreamResourceName]
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

func TestAccRepositoryUpstreamConda_basic(t *testing.T) {
	t.Parallel()

	const condaUpstreamResourceName = "cloudsmith_repository_upstream.conda_forge"

	testAccRepositoryCondaUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-conda"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "conda_forge" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "conda"
    upstream_url  = "https://conda.anaconda.org/conda-forge"
}
`, namespace)

	testAccRepositoryCondaUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-conda"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "conda_forge" {
		extra_header_1 = "X-Custom-Header"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "custom-value"
	    extra_value_2  = "*"
	    namespace      = cloudsmith_repository.test.namespace
	    repository     = cloudsmith_repository.test.slug
		name           = cloudsmith_repository.test.name
	    upstream_type  = "conda"
	    upstream_url   = "https://conda.anaconda.org/conda-forge"
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(condaUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryCondaUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(condaUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckResourceAttrSet(condaUpstreamResourceName, CreatedAt),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, Name, "terraform-acc-test-upstream-conda"),
					resource.TestCheckResourceAttrSet(condaUpstreamResourceName, Priority),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, UpstreamType, "conda"),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, UpstreamUrl, "https://conda.anaconda.org/conda-forge"),
					resource.TestCheckResourceAttrSet(condaUpstreamResourceName, UpdatedAt),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryCondaUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(condaUpstreamResourceName, ExtraHeader1, "X-Custom-Header"),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, ExtraHeader2, "Access-Control-Allow-Origin"),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, ExtraValue1, "custom-value"),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, ExtraValue2, "*"),
					resource.TestCheckResourceAttr(condaUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: condaUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[condaUpstreamResourceName]
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
					waitForIsActiveTrue(debUpstreamResourceName),
					resource.TestCheckResourceAttr(debUpstreamResourceName, AuthMode, "None"),
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

func generateTestCertificateAndKeyFiles() (certPath string, keyPath string, cleanup func(), err error) {
	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create a temporary file for the private key
	keyFile, err := os.CreateTemp("", "test-key-*.pem")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to create temp key file: %w", err)
	}
	keyPath = keyFile.Name()

	// Encode and write the private key
	keyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	if err := pem.Encode(keyFile, keyPEM); err != nil {
		os.Remove(keyPath)
		return "", "", nil, fmt.Errorf("failed to write private key: %w", err)
	}
	keyFile.Close()

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "test.cloudsmith.io",
			Organization: []string{"Cloudsmith Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24), // 24 hour validity
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Create the certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		os.Remove(keyPath)
		return "", "", nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Create a temporary file for the certificate
	certFile, err := os.CreateTemp("", "test-cert-*.pem")
	if err != nil {
		os.Remove(keyPath)
		return "", "", nil, fmt.Errorf("failed to create temp cert file: %w", err)
	}
	certPath = certFile.Name()

	// Encode and write the certificate
	certPEM := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}
	if err := pem.Encode(certFile, certPEM); err != nil {
		os.Remove(keyPath)
		os.Remove(certPath)
		return "", "", nil, fmt.Errorf("failed to write certificate: %w", err)
	}
	certFile.Close()

	cleanup = func() {
		os.Remove(certPath)
		os.Remove(keyPath)
	}

	return certPath, keyPath, cleanup, nil
}

func TestAccRepositoryUpstreamDocker_basic(t *testing.T) {
	t.Parallel()

	const dockerUpstreamResourceName = "cloudsmith_repository_upstream.fakedocker"

	// Generate test certificate files for mTLS authentication
	certPath, keyPath, cleanup, err := generateTestCertificateAndKeyFiles()
	if err != nil {
		t.Fatalf("Failed to generate test certificates: %v", err)
	}
	defer cleanup()

	testAccRepositoryPythonUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-docker"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "fakedocker" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "docker"
    upstream_url  = "https://index.docker.io"
	auth_mode      = "Username and Password"
	auth_secret    = "SuperSecretPassword123!"
	auth_username  = "jonny.tables"
}
`, namespace)

	testAccRepositoryPythonUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-docker"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "fakedocker" {
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
	    priority       = 4
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "docker"
	    upstream_url   = "https://index.docker.io"
	    verify_ssl     = false
	}
	`, namespace)

	testAccRepositoryPythonUpstreamConfigCert := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-docker"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "fakedocker" {
		auth_mode            = "Certificate and Key"
		auth_certificate     = file("%s")
		auth_certificate_key = file("%s")
		is_active           = false
		mode                = "Cache and Proxy"
		name                = cloudsmith_repository.test.name
		namespace           = cloudsmith_repository.test.namespace
		priority            = 5
		repository          = cloudsmith_repository.test.slug
		upstream_type       = "docker"
		upstream_url        = "https://fake.docker.io"
		verify_ssl          = true
	}
	`, namespace, certPath, keyPath)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(dockerUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryPythonUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, AuthMode, "Username and Password"),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, AuthUsername, "jonny.tables"),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(dockerUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(dockerUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, IsActive, "false"),
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
				Config: testAccRepositoryPythonUpstreamConfigCert,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, AuthMode, "Certificate and Key"),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, IsActive, "false"),
					resource.TestCheckResourceAttr(dockerUpstreamResourceName, Priority, "5"),
				),
			},
			{
				ResourceName:      dockerUpstreamResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"auth_certificate",
					"auth_certificate_key",
					"auth_secret",
				},
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
			},
		},
	})
}

func TestAccRepositoryUpstreamGeneric_basic(t *testing.T) {
	t.Parallel()

	const genericUpstreamResourceName = "cloudsmith_repository_upstream.gradle_distributions"

	testAccRepositoryGenericUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-generic"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "gradle_distributions" {
    namespace       = cloudsmith_repository.test.namespace
    repository      = cloudsmith_repository.test.slug
	name            = cloudsmith_repository.test.name
    upstream_type   = "generic"
    upstream_url    = "https://services.gradle.org"
    upstream_prefix = "distributions"
}
`, namespace)

	testAccRepositoryGenericUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-generic"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "gradle_distributions" {
		extra_header_1  = "X-Custom-Header"
	    extra_header_2  = "Access-Control-Allow-Origin"
	    extra_value_1   = "custom-value"
	    extra_value_2   = "*"
	    is_active       = true
	    mode            = "Proxy Only"
		name            = cloudsmith_repository.test.name
	    namespace       = cloudsmith_repository.test.namespace
	    priority        = 12345
	    repository      = cloudsmith_repository.test.slug
	    upstream_type   = "generic"
	    upstream_url    = "https://services.gradle.org"
	    upstream_prefix = "distributions"
	    verify_ssl      = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(genericUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryGenericUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(genericUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(genericUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(genericUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(genericUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(genericUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(genericUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(genericUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(genericUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(genericUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(genericUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, "upstream_prefix", "distributions"),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryGenericUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(genericUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(genericUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(genericUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(genericUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(genericUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(genericUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(genericUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(genericUpstreamResourceName, "upstream_prefix", "distributions"),
				),
			},
			{
				ResourceName: genericUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[genericUpstreamResourceName]
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

func TestAccRepositoryUpstreamGo_basic(t *testing.T) {
	t.Parallel()

	const goUpstreamResourceName = "cloudsmith_repository_upstream.go_proxy"

	testAccRepositoryGoUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-go"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "go_proxy" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "go"
    upstream_url  = "https://proxy.golang.org"
}
`, namespace)

	testAccRepositoryGoUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-go"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "go_proxy" {
		extra_header_1 = "X-Custom-Header"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "custom-value"
	    extra_value_2  = "*"
	    is_active      = true
	    mode           = "Proxy Only"
		name           = cloudsmith_repository.test.name
	    namespace      = cloudsmith_repository.test.namespace
	    priority       = 12345
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "go"
	    upstream_url   = "https://proxy.golang.org"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(goUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryGoUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(goUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(goUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(goUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(goUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(goUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(goUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(goUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(goUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(goUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(goUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(goUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(goUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(goUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(goUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(goUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(goUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(goUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(goUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryGoUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(goUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(goUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(goUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(goUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(goUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(goUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(goUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(goUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: goUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[goUpstreamResourceName]
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
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_secret"},
			},
		},
	})
}

func TestAccRepositoryUpstreamHuggingface_basic(t *testing.T) {
	t.Parallel()

	const huggingfaceUpstreamResourceName = "cloudsmith_repository_upstream.hugging_face"

	testAccRepositoryHuggingfaceUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-huggingface"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "hugging_face" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "huggingface"
    upstream_url  = "https://huggingface.co"
}
`, namespace)

	testAccRepositoryHuggingfaceUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-huggingface"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "hugging_face" {
		extra_header_1 = "X-Custom-Header"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "custom-value"
	    extra_value_2  = "*"
	    namespace      = cloudsmith_repository.test.namespace
	    repository     = cloudsmith_repository.test.slug
		name           = cloudsmith_repository.test.name
	    upstream_type  = "huggingface"
	    upstream_url   = "https://huggingface.co"
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(huggingfaceUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryHuggingfaceUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckResourceAttrSet(huggingfaceUpstreamResourceName, CreatedAt),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, Name, "terraform-acc-test-upstream-huggingface"),
					resource.TestCheckResourceAttrSet(huggingfaceUpstreamResourceName, Priority),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, UpstreamType, "huggingface"),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, UpstreamUrl, "https://huggingface.co"),
					resource.TestCheckResourceAttrSet(huggingfaceUpstreamResourceName, UpdatedAt),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryHuggingfaceUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, ExtraHeader1, "X-Custom-Header"),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, ExtraHeader2, "Access-Control-Allow-Origin"),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, ExtraValue1, "custom-value"),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, ExtraValue2, "*"),
					resource.TestCheckResourceAttr(huggingfaceUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: huggingfaceUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[huggingfaceUpstreamResourceName]
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

func TestAccRepositoryUpstreamHex_basic(t *testing.T) {
	t.Parallel()

	const hexUpstreamResourceName = "cloudsmith_repository_upstream.hex"

	testAccRepositoryHexUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-hex"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "hex" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "hex"
    upstream_url  = "https://repo.hex.pm"
}
`, namespace)

	testAccRepositoryHexUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-hex"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "hex" {
		extra_header_1 = "X-Custom-Header"
	    extra_header_2 = "Access-Control-Allow-Origin"
	    extra_value_1  = "custom-value"
	    extra_value_2  = "*"
	    namespace      = cloudsmith_repository.test.namespace
	    repository     = cloudsmith_repository.test.slug
		name           = cloudsmith_repository.test.name
	    upstream_type  = "hex"
	    upstream_url   = "https://repo.hex.pm"
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(hexUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryHexUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(hexUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckResourceAttrSet(hexUpstreamResourceName, CreatedAt),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, Name, "terraform-acc-test-upstream-hex"),
					resource.TestCheckResourceAttrSet(hexUpstreamResourceName, Priority),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, UpstreamType, "hex"),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, UpstreamUrl, "https://repo.hex.pm"),
					resource.TestCheckResourceAttrSet(hexUpstreamResourceName, UpdatedAt),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryHexUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(hexUpstreamResourceName, ExtraHeader1, "X-Custom-Header"),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, ExtraHeader2, "Access-Control-Allow-Origin"),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, ExtraValue1, "custom-value"),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, ExtraValue2, "*"),
					resource.TestCheckResourceAttr(hexUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: hexUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[hexUpstreamResourceName]
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
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_secret"},
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
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_secret"},
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
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_secret"},
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
	    verify_ssl        = false
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
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_secret"},
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
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_secret"},
			},
		},
	})
}

func TestAccRepositoryUpstreamComposer_basic(t *testing.T) {
	t.Parallel()

	const composerUpstreamResourceName = "cloudsmith_repository_upstream.packagist"

	testAccRepositoryComposerUpstreamConfigBasic := fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-upstream-composer"
	namespace = "%s"
}

resource "cloudsmith_repository_upstream" "packagist" {
    namespace     = cloudsmith_repository.test.namespace
    repository    = cloudsmith_repository.test.slug
	name          = cloudsmith_repository.test.name
    upstream_type = "composer"
    upstream_url  = "https://packagist.org"
}
`, namespace)

	testAccRepositoryComposerUpstreamConfigUpdate := fmt.Sprintf(`
	resource "cloudsmith_repository" "test" {
		name      = "terraform-acc-test-upstream-composer"
		namespace = "%s"
	}

	resource "cloudsmith_repository_upstream" "packagist" {
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
	    priority       = 1
	    repository     = cloudsmith_repository.test.slug
	    upstream_type  = "composer"
	    upstream_url   = "https://packagist.org"
	    verify_ssl     = false
	}
	`, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryUpstreamCheckDestroy(composerUpstreamResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryComposerUpstreamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(composerUpstreamResourceName, AuthMode, "None"),
					resource.TestCheckResourceAttr(composerUpstreamResourceName, AuthUsername, ""),
					resource.TestCheckNoResourceAttr(composerUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(composerUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(composerUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(composerUpstreamResourceName, DistroVersions),
					resource.TestCheckResourceAttr(composerUpstreamResourceName, ExtraHeader1, ""),
					resource.TestCheckResourceAttr(composerUpstreamResourceName, ExtraHeader2, ""),
					resource.TestCheckResourceAttr(composerUpstreamResourceName, ExtraValue1, ""),
					resource.TestCheckResourceAttr(composerUpstreamResourceName, ExtraValue2, ""),
					resource.TestCheckNoResourceAttr(composerUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttr(composerUpstreamResourceName, IsActive, "true"),
					resource.TestCheckResourceAttr(composerUpstreamResourceName, Mode, "Proxy Only"),
					resource.TestCheckResourceAttrSet(composerUpstreamResourceName, Priority),
					resource.TestCheckResourceAttrSet(composerUpstreamResourceName, SlugPerm),
					resource.TestCheckResourceAttrSet(composerUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(composerUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(composerUpstreamResourceName, VerifySsl, "true"),
				),
			},
			{
				Config: testAccRepositoryComposerUpstreamConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(composerUpstreamResourceName, Component),
					resource.TestCheckResourceAttrSet(composerUpstreamResourceName, CreatedAt),
					resource.TestCheckNoResourceAttr(composerUpstreamResourceName, DistroVersion),
					resource.TestCheckNoResourceAttr(composerUpstreamResourceName, DistroVersions),
					resource.TestCheckNoResourceAttr(composerUpstreamResourceName, IncludeSources),
					resource.TestCheckResourceAttrSet(composerUpstreamResourceName, UpdatedAt),
					resource.TestCheckNoResourceAttr(composerUpstreamResourceName, UpstreamDistribution),
					resource.TestCheckResourceAttr(composerUpstreamResourceName, IsActive, "true"),
				),
			},
			{
				ResourceName: composerUpstreamResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[composerUpstreamResourceName]
					return fmt.Sprintf(
						"%s.%s.%s.%s",
						resourceState.Primary.Attributes[Namespace],
						resourceState.Primary.Attributes[Repository],
						resourceState.Primary.Attributes[UpstreamType],
						resourceState.Primary.Attributes[SlugPerm],
					), nil
				},
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_secret"},
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
		case Cargo:
			req := pc.APIClient.ReposApi.ReposUpstreamCargoRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamCargoReadExecute(req)
		case Composer:
			req := pc.APIClient.ReposApi.ReposUpstreamComposerRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamComposerReadExecute(req)
		case Conda:
			req := pc.APIClient.ReposApi.ReposUpstreamCondaRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamCondaReadExecute(req)
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
		case Generic:
			req := pc.APIClient.ReposApi.ReposUpstreamGenericRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamGenericReadExecute(req)
		case Go:
			req := pc.APIClient.ReposApi.ReposUpstreamGoRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamGoReadExecute(req)
		case Helm:
			req := pc.APIClient.ReposApi.ReposUpstreamHelmRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamHelmReadExecute(req)
		case Hex:
			req := pc.APIClient.ReposApi.ReposUpstreamHexRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamHexReadExecute(req)
		case HuggingFace:
			req := pc.APIClient.ReposApi.ReposUpstreamHuggingfaceRead(pc.Auth, namespace, repository, slugPerm)
			_, resp, err = pc.APIClient.ReposApi.ReposUpstreamHuggingfaceReadExecute(req)
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
			return fmt.Errorf("invalid upstream_type: '%s'", upstreamType)
		}

		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify upstream deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify upstream deletion: still exists: %s", resourceName)
		}
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}

		return nil
	}
}

// waitForIsActiveTrue waits up to 4 minutes for the resource's is_active attribute to become "true", checking every 10 seconds.
func waitForIsActiveTrue(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const (
			maxWait  = 4 * time.Minute
			interval = 10 * time.Second
		)
		start := time.Now()
		for {
			resourceState, ok := s.RootModule().Resources[resourceName]
			if !ok {
				return fmt.Errorf("resource %s not found in state", resourceName)
			}
			isActive := resourceState.Primary.Attributes[IsActive]
			if isActive == "true" {
				return nil
			}
			if time.Since(start) > maxWait {
				return fmt.Errorf("timed out waiting for %s is_active to become true", resourceName)
			}
			time.Sleep(interval)
		}
	}
}
