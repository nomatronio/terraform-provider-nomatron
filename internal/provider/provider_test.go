package provider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"testing"
	"time"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

func TestNomatronProvider_Metadata(t *testing.T) {
	t.Parallel()

	p := &NomatronProvider{version: "test-version"}

	req := fwprovider.MetadataRequest{}
	resp := &fwprovider.MetadataResponse{}

	p.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron" {
		t.Fatalf("expected type name %q, got %q", "nomatron", resp.TypeName)
	}

	if resp.Version != "test-version" {
		t.Fatalf("expected version %q, got %q", "test-version", resp.Version)
	}
}

func TestNomatronProvider_Schema(t *testing.T) {
	t.Parallel()

	p := &NomatronProvider{}

	req := fwprovider.SchemaRequest{}
	resp := &fwprovider.SchemaResponse{}

	p.Schema(context.Background(), req, resp)

	attrs := resp.Schema.Attributes

	assertStringAttribute(t, attrs, "address", true, false)
	assertStringAttribute(t, attrs, "token", true, false)
	assertStringAttribute(t, attrs, "tls_cert", false, true)
	assertStringAttribute(t, attrs, "tls_key", false, true)
	assertStringAttribute(t, attrs, "tls_ca_cert", false, true)
}

func TestNomatronProvider_Resources(t *testing.T) {
	t.Parallel()

	p := &NomatronProvider{}

	resources := p.Resources(context.Background())

	if len(resources) != 20 {
		t.Fatalf("expected 20 resources, got %d", len(resources))
	}

	agentResource := resources[0]()
	if agentResource == nil {
		t.Fatal("expected first resource factory to return a non-nil resource")
	}

	if _, ok := agentResource.(*AgentResource); !ok {
		t.Fatalf("expected first resource to be *AgentResource, got %T", agentResource)
	}

	applicationResource := resources[1]()
	if applicationResource == nil {
		t.Fatal("expected second resource factory to return a non-nil resource")
	}

	if _, ok := applicationResource.(*ApplicationResource); !ok {
		t.Fatalf("expected second resource to be *ApplicationResource, got %T", applicationResource)
	}

	environmentResource := resources[2]()
	if environmentResource == nil {
		t.Fatal("expected third resource factory to return a non-nil resource")
	}

	if _, ok := environmentResource.(*EnvironmentResource); !ok {
		t.Fatalf("expected third resource to be *EnvironmentResource, got %T", environmentResource)
	}

	githubAppIntegrationResource := resources[3]()
	if githubAppIntegrationResource == nil {
		t.Fatal("expected fourth resource factory to return a non-nil resource")
	}

	if _, ok := githubAppIntegrationResource.(*GitHubAppIntegrationResource); !ok {
		t.Fatalf("expected fourth resource to be *GitHubAppIntegrationResource, got %T", githubAppIntegrationResource)
	}

	organizationGitHubAppIntegrationResource := resources[4]()
	if organizationGitHubAppIntegrationResource == nil {
		t.Fatal("expected fifth resource factory to return a non-nil resource")
	}

	if _, ok := organizationGitHubAppIntegrationResource.(*OrganizationGitHubAppIntegrationResource); !ok {
		t.Fatalf("expected fifth resource to be *OrganizationGitHubAppIntegrationResource, got %T", organizationGitHubAppIntegrationResource)
	}

	jobResource := resources[5]()
	if jobResource == nil {
		t.Fatal("expected sixth resource factory to return a non-nil resource")
	}

	if _, ok := jobResource.(*JobResource); !ok {
		t.Fatalf("expected sixth resource to be *JobResource, got %T", jobResource)
	}

	nomadClusterResource := resources[6]()
	if nomadClusterResource == nil {
		t.Fatal("expected seventh resource factory to return a non-nil resource")
	}

	if _, ok := nomadClusterResource.(*NomadClusterResource); !ok {
		t.Fatalf("expected seventh resource to be *NomadClusterResource, got %T", nomadClusterResource)
	}

	oidcGroupMappingResource := resources[7]()
	if oidcGroupMappingResource == nil {
		t.Fatal("expected eighth resource factory to return a non-nil resource")
	}

	if _, ok := oidcGroupMappingResource.(*OIDCGroupMappingResource); !ok {
		t.Fatalf("expected eighth resource to be *OIDCGroupMappingResource, got %T", oidcGroupMappingResource)
	}

	oidcProviderResource := resources[8]()
	if oidcProviderResource == nil {
		t.Fatal("expected ninth resource factory to return a non-nil resource")
	}

	if _, ok := oidcProviderResource.(*OIDCProviderResource); !ok {
		t.Fatalf("expected ninth resource to be *OIDCProviderResource, got %T", oidcProviderResource)
	}

	organizationNomadClusterResource := resources[9]()
	if organizationNomadClusterResource == nil {
		t.Fatal("expected tenth resource factory to return a non-nil resource")
	}

	if _, ok := organizationNomadClusterResource.(*OrganizationNomadClusterResource); !ok {
		t.Fatalf("expected tenth resource to be *OrganizationNomadClusterResource, got %T", organizationNomadClusterResource)
	}

	groupMemberResource := resources[10]()
	if groupMemberResource == nil {
		t.Fatal("expected eleventh resource factory to return a non-nil resource")
	}

	if _, ok := groupMemberResource.(*GroupMemberResource); !ok {
		t.Fatalf("expected eleventh resource to be *GroupMemberResource, got %T", groupMemberResource)
	}

	groupResource := resources[11]()
	if groupResource == nil {
		t.Fatal("expected twelfth resource factory to return a non-nil resource")
	}

	if _, ok := groupResource.(*GroupResource); !ok {
		t.Fatalf("expected twelfth resource to be *GroupResource, got %T", groupResource)
	}

	organizationMemberResource := resources[12]()
	if organizationMemberResource == nil {
		t.Fatal("expected thirteenth resource factory to return a non-nil resource")
	}

	if _, ok := organizationMemberResource.(*OrganizationMemberResource); !ok {
		t.Fatalf("expected thirteenth resource to be *OrganizationMemberResource, got %T", organizationMemberResource)
	}

	organizationResource := resources[13]()
	if organizationResource == nil {
		t.Fatal("expected fourteenth resource factory to return a non-nil resource")
	}

	if _, ok := organizationResource.(*OrganizationResource); !ok {
		t.Fatalf("expected fourteenth resource to be *OrganizationResource, got %T", organizationResource)
	}

	roleResource := resources[14]()
	if roleResource == nil {
		t.Fatal("expected fifteenth resource factory to return a non-nil resource")
	}

	if _, ok := roleResource.(*RoleResource); !ok {
		t.Fatalf("expected fifteenth resource to be *RoleResource, got %T", roleResource)
	}

	roleAssignmentResource := resources[15]()
	if roleAssignmentResource == nil {
		t.Fatal("expected sixteenth resource factory to return a non-nil resource")
	}

	if _, ok := roleAssignmentResource.(*RoleAssignmentResource); !ok {
		t.Fatalf("expected sixteenth resource to be *RoleAssignmentResource, got %T", roleAssignmentResource)
	}

	serviceAccountResource := resources[16]()
	if serviceAccountResource == nil {
		t.Fatal("expected seventeenth resource factory to return a non-nil resource")
	}

	if _, ok := serviceAccountResource.(*ServiceAccountResource); !ok {
		t.Fatalf("expected seventeenth resource to be *ServiceAccountResource, got %T", serviceAccountResource)
	}

	variableResource := resources[17]()
	if variableResource == nil {
		t.Fatal("expected eighteenth resource factory to return a non-nil resource")
	}

	if _, ok := variableResource.(*VariableResource); !ok {
		t.Fatalf("expected eighteenth resource to be *VariableResource, got %T", variableResource)
	}

	userResource := resources[18]()
	if userResource == nil {
		t.Fatal("expected nineteenth resource factory to return a non-nil resource")
	}

	if _, ok := userResource.(*UserResource); !ok {
		t.Fatalf("expected nineteenth resource to be *UserResource, got %T", userResource)
	}

	jobApprovalPolicyResource := resources[19]()
	if jobApprovalPolicyResource == nil {
		t.Fatal("expected twentieth resource factory to return a non-nil resource")
	}

	if _, ok := jobApprovalPolicyResource.(*JobApprovalPolicyResource); !ok {
		t.Fatalf("expected twentieth resource to be *JobApprovalPolicyResource, got %T", jobApprovalPolicyResource)
	}
}

func TestNomatronProvider_DataSources(t *testing.T) {
	t.Parallel()

	p := &NomatronProvider{}

	dataSources := p.DataSources(context.Background())

	if len(dataSources) != 18 {
		t.Fatalf("expected 18 data sources, got %d", len(dataSources))
	}

	agentDS := dataSources[0]()
	if agentDS == nil {
		t.Fatal("expected first data source factory to return a non-nil data source")
	}

	if _, ok := agentDS.(*AgentDataSource); !ok {
		t.Fatalf("expected first data source to be *AgentDataSource, got %T", agentDS)
	}

	applicationDS := dataSources[1]()
	if applicationDS == nil {
		t.Fatal("expected second data source factory to return a non-nil data source")
	}

	if _, ok := applicationDS.(*ApplicationDataSource); !ok {
		t.Fatalf("expected second data source to be *ApplicationDataSource, got %T", applicationDS)
	}

	environmentDS := dataSources[2]()
	if environmentDS == nil {
		t.Fatal("expected third data source factory to return a non-nil data source")
	}

	if _, ok := environmentDS.(*EnvironmentDataSource); !ok {
		t.Fatalf("expected third data source to be *EnvironmentDataSource, got %T", environmentDS)
	}

	githubAppIntegrationDS := dataSources[3]()
	if githubAppIntegrationDS == nil {
		t.Fatal("expected fourth data source factory to return a non-nil data source")
	}

	if _, ok := githubAppIntegrationDS.(*GitHubAppIntegrationDataSource); !ok {
		t.Fatalf("expected fourth data source to be *GitHubAppIntegrationDataSource, got %T", githubAppIntegrationDS)
	}

	organizationGitHubAppIntegrationDS := dataSources[4]()
	if organizationGitHubAppIntegrationDS == nil {
		t.Fatal("expected fifth data source factory to return a non-nil data source")
	}

	if _, ok := organizationGitHubAppIntegrationDS.(*OrganizationGitHubAppIntegrationDataSource); !ok {
		t.Fatalf("expected fifth data source to be *OrganizationGitHubAppIntegrationDataSource, got %T", organizationGitHubAppIntegrationDS)
	}

	jobDS := dataSources[5]()
	if jobDS == nil {
		t.Fatal("expected sixth data source factory to return a non-nil data source")
	}

	if _, ok := jobDS.(*JobDataSource); !ok {
		t.Fatalf("expected sixth data source to be *JobDataSource, got %T", jobDS)
	}

	nomadClusterDS := dataSources[6]()
	if nomadClusterDS == nil {
		t.Fatal("expected seventh data source factory to return a non-nil data source")
	}

	if _, ok := nomadClusterDS.(*NomadClusterDataSource); !ok {
		t.Fatalf("expected seventh data source to be *NomadClusterDataSource, got %T", nomadClusterDS)
	}

	oidcGroupMappingDS := dataSources[7]()
	if oidcGroupMappingDS == nil {
		t.Fatal("expected eighth data source factory to return a non-nil data source")
	}

	if _, ok := oidcGroupMappingDS.(*OIDCGroupMappingDataSource); !ok {
		t.Fatalf("expected eighth data source to be *OIDCGroupMappingDataSource, got %T", oidcGroupMappingDS)
	}

	oidcProviderDS := dataSources[8]()
	if oidcProviderDS == nil {
		t.Fatal("expected ninth data source factory to return a non-nil data source")
	}

	if _, ok := oidcProviderDS.(*OIDCProviderDataSource); !ok {
		t.Fatalf("expected ninth data source to be *OIDCProviderDataSource, got %T", oidcProviderDS)
	}

	organizationNomadClusterDS := dataSources[9]()
	if organizationNomadClusterDS == nil {
		t.Fatal("expected tenth data source factory to return a non-nil data source")
	}

	if _, ok := organizationNomadClusterDS.(*OrganizationNomadClusterDataSource); !ok {
		t.Fatalf("expected tenth data source to be *OrganizationNomadClusterDataSource, got %T", organizationNomadClusterDS)
	}

	groupMemberDS := dataSources[10]()
	if groupMemberDS == nil {
		t.Fatal("expected eleventh data source factory to return a non-nil data source")
	}

	if _, ok := groupMemberDS.(*GroupMemberDataSource); !ok {
		t.Fatalf("expected eleventh data source to be *GroupMemberDataSource, got %T", groupMemberDS)
	}

	groupDS := dataSources[11]()
	if groupDS == nil {
		t.Fatal("expected twelfth data source factory to return a non-nil data source")
	}

	if _, ok := groupDS.(*GroupDataSource); !ok {
		t.Fatalf("expected twelfth data source to be *GroupDataSource, got %T", groupDS)
	}

	organizationMemberDS := dataSources[12]()
	if organizationMemberDS == nil {
		t.Fatal("expected thirteenth data source factory to return a non-nil data source")
	}

	if _, ok := organizationMemberDS.(*OrganizationMemberDataSource); !ok {
		t.Fatalf("expected thirteenth data source to be *OrganizationMemberDataSource, got %T", organizationMemberDS)
	}

	organizationDS := dataSources[13]()
	if organizationDS == nil {
		t.Fatal("expected fourteenth data source factory to return a non-nil data source")
	}

	if _, ok := organizationDS.(*OrganizationDataSource); !ok {
		t.Fatalf("expected fourteenth data source to be *OrganizationDataSource, got %T", organizationDS)
	}

	roleDS := dataSources[14]()
	if roleDS == nil {
		t.Fatal("expected fifteenth data source factory to return a non-nil data source")
	}

	if _, ok := roleDS.(*RoleDataSource); !ok {
		t.Fatalf("expected fifteenth data source to be *RoleDataSource, got %T", roleDS)
	}

	serviceAccountsDS := dataSources[15]()
	if serviceAccountsDS == nil {
		t.Fatal("expected sixteenth data source factory to return a non-nil data source")
	}

	if _, ok := serviceAccountsDS.(*ServiceAccountDataSource); !ok {
		t.Fatalf("expected sixteenth data source to be *ServiceAccountDataSource, got %T", serviceAccountsDS)
	}

	usersDS := dataSources[16]()
	if usersDS == nil {
		t.Fatal("expected seventeenth data source factory to return a non-nil data source")
	}

	if _, ok := usersDS.(*UsersDataSource); !ok {
		t.Fatalf("expected seventeenth data source to be *UsersDataSource, got %T", usersDS)
	}

	jobApprovalPolicyDS := dataSources[17]()
	if jobApprovalPolicyDS == nil {
		t.Fatal("expected eighteenth data source factory to return a non-nil data source")
	}

	if _, ok := jobApprovalPolicyDS.(*JobApprovalPolicyDataSource); !ok {
		t.Fatalf("expected eighteenth data source to be *JobApprovalPolicyDataSource, got %T", jobApprovalPolicyDS)
	}
}

func TestNomatronProvider_Actions(t *testing.T) {
	t.Parallel()

	p := &NomatronProvider{}

	actions := p.Actions(context.Background())

	if len(actions) != 4 {
		t.Fatalf("expected 4 actions, got %d", len(actions))
	}

	destroyAppJobsAction := actions[0]()
	if destroyAppJobsAction == nil {
		t.Fatal("expected first action factory to return a non-nil action")
	}

	if _, ok := destroyAppJobsAction.(*DestroyAppJobsAction); !ok {
		t.Fatalf("expected first action to be *DestroyAppJobsAction, got %T", destroyAppJobsAction)
	}

	destroyJobAction := actions[1]()
	if destroyJobAction == nil {
		t.Fatal("expected second action factory to return a non-nil action")
	}

	if _, ok := destroyJobAction.(*DestroyJobAction); !ok {
		t.Fatalf("expected second action to be *DestroyJobAction, got %T", destroyJobAction)
	}

	importNomadJobAction := actions[2]()
	if importNomadJobAction == nil {
		t.Fatal("expected third action factory to return a non-nil action")
	}

	if _, ok := importNomadJobAction.(*ImportNomadJobAction); !ok {
		t.Fatalf("expected third action to be *ImportNomadJobAction, got %T", importNomadJobAction)
	}

	speculativePlanAction := actions[3]()
	if speculativePlanAction == nil {
		t.Fatal("expected fourth action factory to return a non-nil action")
	}

	if _, ok := speculativePlanAction.(*SpeculativePlanAction); !ok {
		t.Fatalf("expected fourth action to be *SpeculativePlanAction, got %T", speculativePlanAction)
	}
}

func TestNomatronProvider_EphemeralResources(t *testing.T) {
	t.Parallel()

	p := &NomatronProvider{}

	ephemeralResources := p.EphemeralResources(context.Background())

	if len(ephemeralResources) != 2 {
		t.Fatalf("expected 2 ephemeral resources, got %d", len(ephemeralResources))
	}

	agentTokenResource := ephemeralResources[0]()
	if agentTokenResource == nil {
		t.Fatal("expected first ephemeral resource factory to return a non-nil resource")
	}

	if _, ok := agentTokenResource.(*AgentTokenEphemeralResource); !ok {
		t.Fatalf("expected first ephemeral resource to be *AgentTokenEphemeralResource, got %T", agentTokenResource)
	}

	serviceAccountTokenResource := ephemeralResources[1]()
	if serviceAccountTokenResource == nil {
		t.Fatal("expected second ephemeral resource factory to return a non-nil resource")
	}

	if _, ok := serviceAccountTokenResource.(*ServiceAccountTokenEphemeralResource); !ok {
		t.Fatalf("expected second ephemeral resource to be *ServiceAccountTokenEphemeralResource, got %T", serviceAccountTokenResource)
	}
}

func TestNomatronProvider_Functions(t *testing.T) {
	t.Parallel()

	p := &NomatronProvider{}

	functions := p.Functions(context.Background())

	if len(functions) != 0 {
		t.Fatalf("expected no functions, got %d", len(functions))
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	factory := New("test")
	if factory == nil {
		t.Fatal("expected provider factory to be non-nil")
	}

	p := factory()
	if p == nil {
		t.Fatal("expected provider instance to be non-nil")
	}

	nomatronProvider, ok := p.(*NomatronProvider)
	if !ok {
		t.Fatalf("expected provider to be *NomatronProvider, got %T", p)
	}

	if nomatronProvider.version != "test" {
		t.Fatalf("expected version %q, got %q", "test", nomatronProvider.version)
	}
}

func TestConfigureClient_HTTP(t *testing.T) {
	t.Parallel()

	data := NomatronProviderModel{
		Address:   types.StringValue("localhost:4649"),
		Token:     types.StringValue("service-account-token"),
		TlsCert:   types.StringNull(),
		TlsKey:    types.StringNull(),
		TlsCaCert: types.StringNull(),
	}

	client, err := configureClient(data)
	if err != nil {
		t.Fatalf("configureClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}

	rawClient, ok := client.ClientInterface.(*sdk.Client)
	if !ok {
		t.Fatalf("expected ClientInterface to be *sdk.Client, got %T", client.ClientInterface)
	}

	if rawClient.Server != "http://localhost:4649/api/v1/" {
		t.Fatalf("expected server %q, got %q", "http://localhost:4649/api/v1/", rawClient.Server)
	}

	httpClient, ok := rawClient.Client.(*http.Client)
	if !ok {
		t.Fatalf("expected SDK http client to be *http.Client, got %T", rawClient.Client)
	}

	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatalf("expected *http.Transport, got %T", httpClient.Transport)
	}

	if transport.TLSClientConfig != nil {
		if len(transport.TLSClientConfig.Certificates) != 0 {
			t.Fatalf("expected no client certificates for plain HTTP")
		}
		if transport.TLSClientConfig.RootCAs != nil {
			t.Fatalf("expected no custom root CAs for plain HTTP")
		}
	}

	if len(rawClient.RequestEditors) != 1 {
		t.Fatalf("expected 1 request editor, got %d", len(rawClient.RequestEditors))
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	if err := rawClient.RequestEditors[0](context.Background(), req); err != nil {
		t.Fatalf("request editor error = %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "Bearer service-account-token" {
		t.Fatalf("expected bearer token header, got %q", got)
	}
}

func TestConfigureClient_HTTPSWhenTLSMaterialPresent(t *testing.T) {
	t.Parallel()

	tlsMaterial := newTestTLSMaterial(t)

	data := NomatronProviderModel{
		Address:   types.StringValue("localhost:4649"),
		Token:     types.StringValue("service-account-token"),
		TlsCert:   types.StringValue(tlsMaterial.clientCertPEM),
		TlsKey:    types.StringValue(tlsMaterial.clientKeyPEM),
		TlsCaCert: types.StringValue(tlsMaterial.caCertPEM),
	}

	client, err := configureClient(data)
	if err != nil {
		t.Fatalf("configureClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}

	rawClient, ok := client.ClientInterface.(*sdk.Client)
	if !ok {
		t.Fatalf("expected ClientInterface to be *sdk.Client, got %T", client.ClientInterface)
	}

	if rawClient.Server != "https://localhost:4649/api/v1/" {
		t.Fatalf("expected server %q, got %q", "https://localhost:4649/api/v1/", rawClient.Server)
	}

	httpClient, ok := rawClient.Client.(*http.Client)
	if !ok {
		t.Fatalf("expected SDK http client to be *http.Client, got %T", rawClient.Client)
	}

	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatalf("expected *http.Transport, got %T", httpClient.Transport)
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLS client config to be set")
	}

	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected TLS min version %v, got %v", tls.VersionTLS12, transport.TLSClientConfig.MinVersion)
	}

	if len(transport.TLSClientConfig.Certificates) != 1 {
		t.Fatalf("expected 1 client certificate, got %d", len(transport.TLSClientConfig.Certificates))
	}

	if transport.TLSClientConfig.RootCAs == nil {
		t.Fatal("expected RootCAs to be set")
	}
}

func TestConfigureClient_InvalidTLS(t *testing.T) {
	t.Parallel()

	data := NomatronProviderModel{
		Address: types.StringValue("localhost:4649"),
		Token:   types.StringValue("service-account-token"),
		TlsCert: types.StringValue("not-a-cert"),
		TlsKey:  types.StringValue("not-a-key"),
	}

	client, err := configureClient(data)
	if err == nil {
		t.Fatalf("expected configureClient() to fail, got client=%v", client)
	}
}

func TestConfigureClient_ExistingSchemeIsPreserved(t *testing.T) {
	t.Parallel()

	data := NomatronProviderModel{
		Address:   types.StringValue("https://nomatron.example.com/custom/base"),
		Token:     types.StringValue("service-account-token"),
		TlsCert:   types.StringNull(),
		TlsKey:    types.StringNull(),
		TlsCaCert: types.StringNull(),
	}

	client, err := configureClient(data)
	if err != nil {
		t.Fatalf("configureClient() error = %v", err)
	}

	rawClient, ok := client.ClientInterface.(*sdk.Client)
	if !ok {
		t.Fatalf("expected ClientInterface to be *sdk.Client, got %T", client.ClientInterface)
	}

	if rawClient.Server != "https://nomatron.example.com/custom/base/" {
		t.Fatalf("expected existing scheme/path to be preserved, got %q", rawClient.Server)
	}
}

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	stringAttr, ok := attr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected attribute %q to be schema.StringAttribute, got %T", name, attr)
	}

	if stringAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, stringAttr.Required)
	}

	if stringAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, stringAttr.Optional)
	}
}

type testTLSMaterial struct {
	caCertPEM     string
	clientCertPEM string
	clientKeyPEM  string
}

func newTestTLSMaterial(t *testing.T) testTLSMaterial {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey(ca): %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate(ca): %v", err)
	}

	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("ParseCertificate(ca): %v", err)
	}

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey(client): %v", err)
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "Test Client",
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate(client): %v", err)
	}

	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caDER,
	})

	clientCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientDER,
	})

	clientKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(clientKey),
	})

	return testTLSMaterial{
		caCertPEM:     string(caCertPEM),
		clientCertPEM: string(clientCertPEM),
		clientKeyPEM:  string(clientKeyPEM),
	}
}
