package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// upstream types
const (
	Cran   = "cran"
	Dart   = "dart"
	Deb    = "deb"
	Docker = "docker"
	Helm   = "helm"
	Maven  = "maven"
	Npm    = "npm"
	NuGet  = "nuget"
	Python = "python"
	Rpm    = "rpm"
	Ruby   = "ruby"
	Swift  = "swift"
)

// tf state prop names
const (
	AuthMode             = "auth_mode"
	AuthSecret           = "auth_secret"
	AuthUsername         = "auth_username"
	Component            = "component"
	DistroVersion        = "distro_version"
	DistroVersions       = "distro_versions"
	ExtraHeader1         = "extra_header_1"
	ExtraHeader2         = "extra_header_2"
	ExtraValue1          = "extra_value_1"
	ExtraValue2          = "extra_value_2"
	IsActive             = "is_active"
	IncludeSources       = "include_sources"
	Mode                 = "mode"
	Priority             = "priority"
	UpstreamDistribution = "upstream_distribution"
	UpstreamType         = "upstream_type"
	UpstreamUrl          = "upstream_url"
	VerifySsl            = "verify_ssl"
)

var (
	authModes = []string{
		"None",
		"Username and Password",
		"Token",
	}
	upstreamModes = []string{
		"Proxy Only",
		"Cache and Proxy",
		"Cache Only",
	}
	upstreamTypes = []string{
		Cran,
		Dart,
		Deb,
		Docker,
		Helm,
		Maven,
		Npm,
		NuGet,
		Python,
		Rpm,
		Ruby,
		Swift,
	}
)

type Upstream interface {
	GetAuthMode() string
	GetAuthSecret() string
	GetAuthUsername() string
	GetExtraHeader1() string
	GetExtraHeader2() string
	GetExtraValue1() string
	GetExtraValue2() string
	GetCreatedAt() time.Time
	GetIsActive() bool
	GetMode() string
	GetName() string
	GetPriority() int64
	GetSlugPerm() string
	GetUpdatedAt() time.Time
	GetUpstreamUrl() string
	GetVerifySsl() bool
}

func importUpstream(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 4 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <namespace_slug>.<repository_slug>.<upstream_type>.<upsteam_slug_perm>, got: %s", d.Id(),
		)
	}

	_ = d.Set(Namespace, idParts[0])
	_ = d.Set(Repository, idParts[1])
	_ = d.Set(UpstreamType, idParts[2])
	d.SetId(idParts[3])
	return []*schema.ResourceData{d}, nil
}

func resourceRepositoryUpstreamCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)
	upstreamType := requiredString(d, UpstreamType)

	authMode := optionalString(d, AuthMode)
	authSecret := nullableString(d, AuthSecret)
	authUsername := nullableString(d, AuthUsername)
	extraHeader1 := nullableString(d, ExtraHeader1)
	extraHeader2 := nullableString(d, ExtraHeader2)
	extraValue1 := nullableString(d, ExtraValue1)
	extraValue2 := nullableString(d, ExtraValue2)
	isActive := optionalBool(d, IsActive)
	mode := optionalString(d, Mode)
	name := requiredString(d, Name)
	priority := optionalInt64(d, Priority)
	upstreamUrl := requiredString(d, UpstreamUrl)
	verifySsl := optionalBool(d, VerifySsl)

	var upstream Upstream
	var resp *http.Response
	var err error

	switch upstreamType {
	case Cran:
		req := pc.APIClient.ReposApi.ReposUpstreamCranCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.CranUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamCranCreateExecute(req)
	case Dart:
		req := pc.APIClient.ReposApi.ReposUpstreamDartCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.DartUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamDartCreateExecute(req)
	case Deb:
		req := pc.APIClient.ReposApi.ReposUpstreamDebCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.DebUpstreamRequest{
			AuthMode:             authMode,
			AuthSecret:           authSecret,
			AuthUsername:         authUsername,
			Component:            optionalString(d, Component),
			DistroVersions:       expandStrings(d, DistroVersions),
			ExtraHeader1:         extraHeader1,
			ExtraHeader2:         extraHeader2,
			ExtraValue1:          extraValue1,
			ExtraValue2:          extraValue2,
			IncludeSources:       optionalBool(d, IncludeSources),
			IsActive:             isActive,
			Mode:                 mode,
			Name:                 name,
			Priority:             priority,
			UpstreamDistribution: nullableString(d, UpstreamDistribution),
			UpstreamUrl:          upstreamUrl,
			VerifySsl:            verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamDebCreateExecute(req)
	case Docker:
		req := pc.APIClient.ReposApi.ReposUpstreamDockerCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.DockerUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamDockerCreateExecute(req)
	case Helm:
		req := pc.APIClient.ReposApi.ReposUpstreamHelmCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.HelmUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamHelmCreateExecute(req)
	case Maven:
		req := pc.APIClient.ReposApi.ReposUpstreamMavenCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.MavenUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamMavenCreateExecute(req)
	case Npm:
		req := pc.APIClient.ReposApi.ReposUpstreamNpmCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.NpmUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamNpmCreateExecute(req)
	case NuGet:
		req := pc.APIClient.ReposApi.ReposUpstreamNugetCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.NugetUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamNugetCreateExecute(req)
	case Python:
		req := pc.APIClient.ReposApi.ReposUpstreamPythonCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.PythonUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamPythonCreateExecute(req)
	case Rpm:
		req := pc.APIClient.ReposApi.ReposUpstreamRpmCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.RpmUpstreamRequest{
			AuthMode:       authMode,
			AuthSecret:     authSecret,
			AuthUsername:   authUsername,
			DistroVersion:  requiredString(d, DistroVersion),
			ExtraHeader1:   extraHeader1,
			ExtraHeader2:   extraHeader2,
			ExtraValue1:    extraValue1,
			ExtraValue2:    extraValue2,
			IncludeSources: optionalBool(d, IncludeSources),
			IsActive:       isActive,
			Mode:           mode,
			Name:           name,
			Priority:       priority,
			UpstreamUrl:    upstreamUrl,
			VerifySsl:      verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamRpmCreateExecute(req)
	case Ruby:
		req := pc.APIClient.ReposApi.ReposUpstreamRubyCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.RubyUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamRubyCreateExecute(req)
	case Swift:
		req := pc.APIClient.ReposApi.ReposUpstreamSwiftCreate(pc.Auth, namespace, repository)
		req = req.Data(cloudsmith.SwiftUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamSwiftCreateExecute(req)
	default:
		err = fmt.Errorf("invalid upstream type: '%s'", upstreamType)
	}

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusInternalServerError {
			// Until we handle this better in API response we have to assume that this is the issue
			return fmt.Errorf("this `upstream_url` might be already configured for this repository. %w", err)
		}
		return err
	}

	d.SetId(upstream.GetSlugPerm())

	checkerFunc := func() error {
		if upstream, resp, err = getUpstream(d, m); err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return err
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for upstream (%s) to be created: %w", d.Id(), err)
	}

	return resourceRepositoryUpstreamRead(d, m)
}

func getUpstream(d *schema.ResourceData, m interface{}) (Upstream, *http.Response, error) {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)
	upstreamType := requiredString(d, UpstreamType)

	var err error
	var resp *http.Response
	var upstream Upstream

	switch upstreamType {
	case Cran:
		req := pc.APIClient.ReposApi.ReposUpstreamCranRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamCranReadExecute(req)
	case Dart:
		req := pc.APIClient.ReposApi.ReposUpstreamDartRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamDartReadExecute(req)
	case Deb:
		req := pc.APIClient.ReposApi.ReposUpstreamDebRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamDebReadExecute(req)
	case Docker:
		req := pc.APIClient.ReposApi.ReposUpstreamDockerRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamDockerReadExecute(req)
	case Helm:
		req := pc.APIClient.ReposApi.ReposUpstreamHelmRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamHelmReadExecute(req)
	case Maven:
		req := pc.APIClient.ReposApi.ReposUpstreamMavenRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamMavenReadExecute(req)
	case Npm:
		req := pc.APIClient.ReposApi.ReposUpstreamNpmRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamNpmReadExecute(req)
	case NuGet:
		req := pc.APIClient.ReposApi.ReposUpstreamNugetRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamNugetReadExecute(req)
	case Python:
		req := pc.APIClient.ReposApi.ReposUpstreamPythonRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamPythonReadExecute(req)
	case Rpm:
		req := pc.APIClient.ReposApi.ReposUpstreamRpmRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamRpmReadExecute(req)
	case Ruby:
		req := pc.APIClient.ReposApi.ReposUpstreamRubyRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamRubyReadExecute(req)
	case Swift:
		req := pc.APIClient.ReposApi.ReposUpstreamSwiftRead(pc.Auth, namespace, repository, d.Id())
		upstream, resp, err = pc.APIClient.ReposApi.ReposUpstreamSwiftReadExecute(req)
	default:
		err = fmt.Errorf("invalid upstream_type '%s'", upstreamType)
	}

	return upstream, resp, err
}

func resourceRepositoryUpstreamRead(d *schema.ResourceData, m interface{}) error {

	upstream, resp, err := getUpstream(d, m)

	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return err
	}

	_ = d.Set(AuthMode, upstream.GetAuthMode())
	_ = d.Set(AuthSecret, upstream.GetAuthSecret())
	_ = d.Set(AuthUsername, upstream.GetAuthUsername())
	_ = d.Set(CreatedAt, timeToString(upstream.GetCreatedAt()))
	_ = d.Set(ExtraHeader1, upstream.GetExtraHeader1())
	_ = d.Set(ExtraHeader2, upstream.GetExtraHeader2())
	_ = d.Set(ExtraValue1, upstream.GetExtraValue1())
	_ = d.Set(ExtraValue2, upstream.GetExtraValue2())
	_ = d.Set(IsActive, upstream.GetIsActive())
	_ = d.Set(Mode, upstream.GetMode())
	_ = d.Set(Name, upstream.GetName())
	_ = d.Set(Priority, upstream.GetPriority())
	_ = d.Set(SlugPerm, upstream.GetSlugPerm())
	_ = d.Set(UpdatedAt, timeToString(upstream.GetUpdatedAt()))
	_ = d.Set(UpstreamUrl, upstream.GetUpstreamUrl())
	_ = d.Set(VerifySsl, upstream.GetVerifySsl())

	switch u := upstream.(type) {
	case *cloudsmith.DebUpstream:
		_ = d.Set(Component, u.GetComponent())
		_ = d.Set(DistroVersions, flattenStrings(u.GetDistroVersions()))
		_ = d.Set(IncludeSources, u.GetIncludeSources())
		_ = d.Set(UpstreamDistribution, u.GetUpstreamDistribution())
	case *cloudsmith.RpmUpstream:
		_ = d.Set(DistroVersion, u.GetDistroVersion())
		_ = d.Set(IncludeSources, u.GetIncludeSources())
	}

	// namespace, repository and upstream_type are not returned from the read
	// endpoint, so we can use the values stored in resource state. We rely on
	// ForceNew to ensure that if any of these change then a new resource is created.
	_ = d.Set(Namespace, requiredString(d, Namespace))
	_ = d.Set(Repository, requiredString(d, Repository))
	_ = d.Set(UpstreamType, requiredString(d, UpstreamType))

	return nil
}

func resourceRepositoryUpstreamUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)
	upstreamType := requiredString(d, UpstreamType)
	slugPerm := d.Id()

	authMode := optionalString(d, AuthMode)
	authSecret := nullableString(d, AuthSecret)
	authUsername := nullableString(d, AuthUsername)
	extraHeader1 := nullableString(d, ExtraHeader1)
	extraHeader2 := nullableString(d, ExtraHeader2)
	extraValue1 := nullableString(d, ExtraValue1)
	extraValue2 := nullableString(d, ExtraValue2)
	isActive := optionalBool(d, IsActive)
	mode := optionalString(d, Mode)
	name := requiredString(d, Name)
	priority := optionalInt64(d, Priority)
	upstreamUrl := requiredString(d, UpstreamUrl)
	verifySsl := optionalBool(d, VerifySsl)

	var upstream Upstream
	var err error

	switch upstreamType {
	case Cran:
		req := pc.APIClient.ReposApi.ReposUpstreamCranUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.CranUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamCranUpdateExecute(req)
	case Dart:
		req := pc.APIClient.ReposApi.ReposUpstreamDartUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.DartUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamDartUpdateExecute(req)
	case Deb:
		req := pc.APIClient.ReposApi.ReposUpstreamDebUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.DebUpstreamRequest{
			AuthMode:             authMode,
			AuthSecret:           authSecret,
			AuthUsername:         authUsername,
			Component:            optionalString(d, Component),
			DistroVersions:       expandStrings(d, DistroVersions),
			ExtraHeader1:         extraHeader1,
			ExtraHeader2:         extraHeader2,
			ExtraValue1:          extraValue1,
			ExtraValue2:          extraValue2,
			IncludeSources:       optionalBool(d, IncludeSources),
			IsActive:             isActive,
			Mode:                 mode,
			Name:                 name,
			Priority:             priority,
			UpstreamDistribution: nullableString(d, UpstreamDistribution),
			UpstreamUrl:          upstreamUrl,
			VerifySsl:            verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamDebUpdateExecute(req)
	case Docker:
		req := pc.APIClient.ReposApi.ReposUpstreamDockerUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.DockerUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamDockerUpdateExecute(req)
	case Helm:
		req := pc.APIClient.ReposApi.ReposUpstreamHelmUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.HelmUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamHelmUpdateExecute(req)
	case Maven:
		req := pc.APIClient.ReposApi.ReposUpstreamMavenUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.MavenUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamMavenUpdateExecute(req)
	case Npm:
		req := pc.APIClient.ReposApi.ReposUpstreamNpmUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.NpmUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamNpmUpdateExecute(req)
	case NuGet:
		req := pc.APIClient.ReposApi.ReposUpstreamNugetUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.NugetUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamNugetUpdateExecute(req)
	case Python:
		req := pc.APIClient.ReposApi.ReposUpstreamPythonUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.PythonUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamPythonUpdateExecute(req)
	case Rpm:
		req := pc.APIClient.ReposApi.ReposUpstreamRpmUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.RpmUpstreamRequest{
			AuthMode:       authMode,
			AuthSecret:     authSecret,
			AuthUsername:   authUsername,
			DistroVersion:  requiredString(d, DistroVersion),
			ExtraHeader1:   extraHeader1,
			ExtraHeader2:   extraHeader2,
			ExtraValue1:    extraValue1,
			ExtraValue2:    extraValue2,
			IncludeSources: optionalBool(d, IncludeSources),
			IsActive:       isActive,
			Mode:           mode,
			Name:           name,
			Priority:       priority,
			UpstreamUrl:    upstreamUrl,
			VerifySsl:      verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamRpmUpdateExecute(req)
	case Ruby:
		req := pc.APIClient.ReposApi.ReposUpstreamRubyUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.RubyUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamRubyUpdateExecute(req)
	case Swift:
		req := pc.APIClient.ReposApi.ReposUpstreamSwiftUpdate(pc.Auth, namespace, repository, slugPerm)
		req = req.Data(cloudsmith.SwiftUpstreamRequest{
			AuthMode:     authMode,
			AuthSecret:   authSecret,
			AuthUsername: authUsername,
			ExtraHeader1: extraHeader1,
			ExtraHeader2: extraHeader2,
			ExtraValue1:  extraValue1,
			ExtraValue2:  extraValue2,
			IsActive:     isActive,
			Mode:         mode,
			Name:         name,
			Priority:     priority,
			UpstreamUrl:  upstreamUrl,
			VerifySsl:    verifySsl,
		})
		upstream, _, err = pc.APIClient.ReposApi.ReposUpstreamSwiftUpdateExecute(req)
	default:
		err = fmt.Errorf("invalid upstream type: '%s'", upstreamType)
	}

	if err != nil {
		return err
	}

	d.SetId(upstream.GetSlugPerm())

	checkerFunc := func() error {
		if upstream, _, err = getUpstream(d, m); err != nil {
			return err
		}
		if !stringToTime(d.Get(UpdatedAt).(string)).Before(upstream.GetUpdatedAt()) {
			return errKeepWaiting
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for upstream (%s) to be updated: %w", d.Id(), err)
	}

	return resourceRepositoryUpstreamRead(d, m)
}

func resourceRepositoryUpstreamDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)
	upstreamType := requiredString(d, UpstreamType)

	var err error

	switch upstreamType {
	case Cran:
		req := pc.APIClient.ReposApi.ReposUpstreamCranDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamCranDeleteExecute(req)
	case Dart:
		req := pc.APIClient.ReposApi.ReposUpstreamDartDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamDartDeleteExecute(req)
	case Deb:
		req := pc.APIClient.ReposApi.ReposUpstreamDebDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamDebDeleteExecute(req)
	case Docker:
		req := pc.APIClient.ReposApi.ReposUpstreamDockerDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamDockerDeleteExecute(req)
	case Helm:
		req := pc.APIClient.ReposApi.ReposUpstreamHelmDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamHelmDeleteExecute(req)
	case Maven:
		req := pc.APIClient.ReposApi.ReposUpstreamMavenDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamMavenDeleteExecute(req)
	case Npm:
		req := pc.APIClient.ReposApi.ReposUpstreamNpmDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamNpmDeleteExecute(req)
	case NuGet:
		req := pc.APIClient.ReposApi.ReposUpstreamNugetDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamNugetDeleteExecute(req)
	case Python:
		req := pc.APIClient.ReposApi.ReposUpstreamPythonDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamPythonDeleteExecute(req)
	case Rpm:
		req := pc.APIClient.ReposApi.ReposUpstreamRpmDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamRpmDeleteExecute(req)
	case Ruby:
		req := pc.APIClient.ReposApi.ReposUpstreamRubyDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamRubyDeleteExecute(req)
	case Swift:
		req := pc.APIClient.ReposApi.ReposUpstreamSwiftDelete(pc.Auth, namespace, repository, d.Id())
		_, err = pc.APIClient.ReposApi.ReposUpstreamSwiftDeleteExecute(req)
	default:
		err = fmt.Errorf("invalid upstream_type: '%s'", upstreamType)
	}

	if err != nil {
		return err
	}

	checkerFunc := func() error {
		if _, resp, err := getUpstream(d, m); err != nil {
			if is404(resp) {
				return nil
			}
			return err
		}
		return errKeepWaiting
	}
	if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
		return fmt.Errorf("error waiting for upstream (%s) to be deleted: %w", d.Id(), err)
	}

	return nil
}

func validateUpstreamUrl(v interface{}, k string) (warnings []string, errors []error) {
	valueStr := v.(string)
	if len(valueStr) > 0 && valueStr[len(valueStr)-1] == '/' {
		errors = append(errors, fmt.Errorf("%q cannot end with a trailing slash", k))
	}
	return
}

func resourceRepositoryUpstream() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryUpstreamCreate,
		Read:   resourceRepositoryUpstreamRead,
		Update: resourceRepositoryUpstreamUpdate,
		Delete: resourceRepositoryUpstreamDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importUpstream,
		},

		Schema: map[string]*schema.Schema{
			AuthMode: {
				Type:         schema.TypeString,
				Description:  "The authentication mode to use when accessing this upstream.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(authModes, false),
			},
			AuthSecret: {
				Type:         schema.TypeString,
				Description:  "Secret to provide with requests to upstream.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			AuthUsername: {
				Type:         schema.TypeString,
				Description:  "Username to provide with requests to upstream.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			Component: {
				Type:         schema.TypeString,
				Description:  "(deb only) The component to fetch from the upstream.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			CreatedAt: {
				Type:        schema.TypeString,
				Description: "ISO 8601 timestamp at which the Upstream was created.",
				Computed:    true,
			},
			DistroVersion: {
				Type:         schema.TypeString,
				Description:  "(rpm only) The distribution version that packages found on this upstream will be associated with.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			DistroVersions: {
				Type:        schema.TypeSet,
				Description: "(deb only) The distribution versions that packages found on this upstream will be associated with.",
				Optional:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
			ExtraHeader1: {
				Type:         schema.TypeString,
				Description:  "The key for extra header #1 to send to upstream.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			ExtraHeader2: {
				Type:         schema.TypeString,
				Description:  "The key for extra header #2 to send to upstream.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			ExtraValue1: {
				Type:         schema.TypeString,
				Description:  "The value for extra header #1 to send to upstream. This is stored as plaintext, and is NOT encrypted.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			ExtraValue2: {
				Type:         schema.TypeString,
				Description:  "The value for extra header #2 to send to upstream. This is stored as plaintext, and is NOT encrypted.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			IncludeSources: {
				Type:        schema.TypeBool,
				Description: "(deb/rpm only) When true, source packages will be available from this upstream.",
				Optional:    true,
				Computed:    true,
			},
			IsActive: {
				Type:        schema.TypeBool,
				Description: "Whether or not this upstream is active and ready for requests.",
				Optional:    true,
				Computed:    true,
			},
			Mode: {
				Type:         schema.TypeString,
				Description:  "The mode that this upstream should operate in. Upstream sources can be used to proxy resolved packages, as well as operate in a proxy/cache or cache only mode.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(upstreamModes, false),
			},
			Name: {
				Type:         schema.TypeString,
				Description:  "A descriptive name for this upstream source. A shortened version of this name will be used for tagging cached packages retrieved from this upstream.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			Namespace: {
				Type:         schema.TypeString,
				Description:  "The Organization to which the Upstream belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			Priority: {
				Type:         schema.TypeInt,
				Description:  "Upstream sources are selected for resolving requests by sequential order (1..n), followed by creation date.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntBetween(1, 32767),
			},
			Repository: {
				Type:         schema.TypeString,
				Description:  "The Repository to which the Upstream belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			SlugPerm: {
				Type:        schema.TypeString,
				Description: "The unique identifier for this Upstream.",
				Computed:    true,
			},
			UpdatedAt: {
				Type:        schema.TypeString,
				Description: "ISO 8601 timestamp at which the Upstream was updated.",
				Computed:    true,
			},
			UpstreamDistribution: {
				Type:         schema.TypeString,
				Description:  "(deb only) The distribution to fetch from the upstream.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			UpstreamType: {
				Type:         schema.TypeString,
				Description:  "The type of Upstream (docker, nuget, python, ...)",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(upstreamTypes, false),
			},
			UpstreamUrl: {
				Type:        schema.TypeString,
				Description: "The URL for this upstream source. This must be a fully qualified URL including any path elements required to reach the root of the repository.",
				Required:    true,
				ValidateFunc: validation.All(
					validation.StringIsNotEmpty,
					validateUpstreamUrl,
				),
			},
			VerifySsl: {
				Type:        schema.TypeBool,
				Description: "If enabled, SSL certificates are verified when requests are made to this upstream. It's recommended to leave this enabled for all public sources to help mitigate Man-In-The-Middle (MITM) attacks. Please note this only applies to HTTPS upstreams.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}
