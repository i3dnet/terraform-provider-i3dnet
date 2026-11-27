package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"terraform-provider-i3dnet/internal/cdn_api"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                   = (*cdnResource)(nil)
	_ resource.ResourceWithConfigure      = (*cdnResource)(nil)
	_ resource.ResourceWithImportState    = (*cdnResource)(nil)
	_ resource.ResourceWithModifyPlan     = (*cdnResource)(nil)
	_ resource.ResourceWithValidateConfig = (*cdnResource)(nil)
)

const (
	// cdnCNAMETarget is where every hostname of a CDN resource points.
	cdnCNAMETarget = "static.cdn-v2.i3d.net."

	cdnDefaultCreateTimeout = 10 * time.Minute
	cdnDefaultDeleteTimeout = 10 * time.Minute

	// cdnMinHealthcheckIntervalMs is the shortest health check interval the
	// CDN API accepts.
	cdnMinHealthcheckIntervalMs = 15000
)

// cdnDeleteRetryInterval is how long Delete waits before asking the API again
// to delete a resource whose cleared settings it does not see yet. A variable
// so tests can shorten it.
var cdnDeleteRetryInterval = 5 * time.Second

func NewCDNResource() resource.Resource {
	return &cdnResource{}
}

type cdnResource struct {
	client *cdn_api.Client
}

type cdnResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	Name                types.String   `tfsdk:"name"`
	GroupCode           types.String   `tfsdk:"group_code"`
	InitiativeCode      types.String   `tfsdk:"initiative_code"`
	Hostnames           types.List     `tfsdk:"hostnames"`
	Strategy            types.String   `tfsdk:"strategy"`
	Origins             types.List     `tfsdk:"origins"`
	Flags               types.Object   `tfsdk:"flags"`
	PredefinedAllowlist types.String   `tfsdk:"predefined_allowlist"`
	CNAMETarget         types.String   `tfsdk:"cname_target"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

type cdnHostnameModel struct {
	FQDN  types.String `tfsdk:"fqdn"`
	SSLCN types.String `tfsdk:"ssl_cn"`
}

type cdnOriginModel struct {
	FQDN        types.String `tfsdk:"fqdn"`
	Port        types.Int64  `tfsdk:"port"`
	SSL         types.Bool   `tfsdk:"ssl"`
	VerifySSL   types.Bool   `tfsdk:"verify_ssl"`
	Location    types.String `tfsdk:"location"`
	DNSRecord   types.String `tfsdk:"dns_record"`
	Healthcheck types.Object `tfsdk:"healthcheck"`
}

type cdnHealthcheckModel struct {
	Path               types.String `tfsdk:"path"`
	Host               types.String `tfsdk:"host"`
	ExpectedStatusCode types.Int64  `tfsdk:"expected_status_code"`
	IntervalMs         types.Int64  `tfsdk:"interval_ms"`
	TimeoutMs          types.Int64  `tfsdk:"timeout_ms"`
	MinValidChecks     types.Int64  `tfsdk:"min_valid_checks"`
}

type cdnFlagsModel struct {
	Enabled             types.Bool `tfsdk:"enabled"`
	ForceHTTPS          types.Bool `tfsdk:"force_https"`
	AuthPassThrough     types.Bool `tfsdk:"auth_pass_through"`
	OverrideBackendHost types.Bool `tfsdk:"override_backend_host"`
	OverrideHealthHost  types.Bool `tfsdk:"override_health_host"`
	UseI3DStaticDNS     types.Bool `tfsdk:"use_i3d_static_dns"`
}

var (
	cdnHostnameAttrTypes = map[string]attr.Type{
		"fqdn":   types.StringType,
		"ssl_cn": types.StringType,
	}
	cdnHealthcheckAttrTypes = map[string]attr.Type{
		"path":                 types.StringType,
		"host":                 types.StringType,
		"expected_status_code": types.Int64Type,
		"interval_ms":          types.Int64Type,
		"timeout_ms":           types.Int64Type,
		"min_valid_checks":     types.Int64Type,
	}
	cdnOriginAttrTypes = map[string]attr.Type{
		"fqdn":        types.StringType,
		"port":        types.Int64Type,
		"ssl":         types.BoolType,
		"verify_ssl":  types.BoolType,
		"location":    types.StringType,
		"dns_record":  types.StringType,
		"healthcheck": types.ObjectType{AttrTypes: cdnHealthcheckAttrTypes},
	}
	cdnFlagsAttrTypes = map[string]attr.Type{
		"enabled":               types.BoolType,
		"force_https":           types.BoolType,
		"auth_pass_through":     types.BoolType,
		"override_backend_host": types.BoolType,
		"override_health_host":  types.BoolType,
		"use_i3d_static_dns":    types.BoolType,
	}
)

// cdnDefaultFlags are the flags of a resource whose configuration has no
// flags block.
var cdnDefaultFlags = cdn_api.Flags{
	Enabled:             true,
	ForceHTTPS:          true,
	AuthPassThrough:     false,
	OverrideBackendHost: true,
	OverrideHealthHost:  true,
	UseI3DStaticDNS:     false,
}

func (r *cdnResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = cdnClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *cdnResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_resource"
}

func (r *cdnResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an i3D.net CDN resource: a set of public hostnames that the CDN serves from one or more origins.\n\n" +
			"The resource manages the hostnames, the origins, the flags and an optional predefined allowlist. " +
			"Every setting is read back from the CDN API, so a plan shows changes made outside Terraform, for example in the CDN dashboard, and an apply puts them back.\n\n" +
			"Point each hostname at the CDN with a CNAME record to `cname_target`.\n\n" +
			"The provider needs a CDN API key (`cdn_api_key`) for this resource. The key belongs to one group and initiative, and the resource must use the same `group_code` and `initiative_code`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "UID of the CDN resource.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the CDN resource, as shown in the CDN dashboard. 1 to 50 characters.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 50)},
			},
			"group_code": schema.StringAttribute{
				MarkdownDescription: "Code of the CDN group the resource belongs to. Must be the group of the provider's CDN API key. Cannot be changed after creation.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"initiative_code": schema.StringAttribute{
				MarkdownDescription: "Code of the CDN initiative the resource belongs to. Must be the initiative of the provider's CDN API key. Cannot be changed after creation.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"hostnames": schema.ListNestedAttribute{
				MarkdownDescription: "Public hostnames the CDN serves.",
				Required:            true,
				Validators:          []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"fqdn": schema.StringAttribute{
							MarkdownDescription: "The hostname, without a trailing dot.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 200),
								stringvalidator.RegexMatches(noTrailingDot, "must not end with a dot"),
							},
						},
						"ssl_cn": schema.StringAttribute{
							MarkdownDescription: "Name of the edge TLS certificate that covers `fqdn`. " +
								"The API does not check that the certificate exists, so a wrong name shows up only as failing HTTPS.",
							Required:   true,
							Validators: []validator.String{stringvalidator.LengthBetween(1, 200)},
						},
					},
				},
			},
			"strategy": schema.StringAttribute{
				MarkdownDescription: "How the edges pick an origin: `HASH`, `FALLBACK` or `RANDOM`. Defaults to `HASH`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("HASH"),
				Validators:          []validator.String{stringvalidator.OneOf("HASH", "FALLBACK", "RANDOM")},
			},
			"origins": schema.ListNestedAttribute{
				MarkdownDescription: "Origins the edges fetch content from.",
				Required:            true,
				Validators:          []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"fqdn": schema.StringAttribute{
							MarkdownDescription: "Host name or IP address of the origin.",
							Required:            true,
							Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"port": schema.Int64Attribute{
							MarkdownDescription: "Port of the origin. Defaults to `443`.",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(443),
							Validators:          []validator.Int64{int64validator.Between(1, 65535)},
						},
						"ssl": schema.BoolAttribute{
							MarkdownDescription: "Whether the edges connect to the origin over TLS. Defaults to `true`.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(true),
						},
						"verify_ssl": schema.BoolAttribute{
							MarkdownDescription: "Whether the edges verify the origin's TLS certificate. Defaults to `true`.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(true),
						},
						"location": schema.StringAttribute{
							MarkdownDescription: "Region of the origin: `EU`, `NAW`, `NAE` or `APAC`. Defaults to `EU`.",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("EU"),
							Validators:          []validator.String{stringvalidator.OneOf("EU", "NAW", "NAE", "APAC")},
						},
						"dns_record": schema.StringAttribute{
							MarkdownDescription: "How the origin's address is resolved: `A` or `SRV`. Defaults to `A`.",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("A"),
							Validators:          []validator.String{stringvalidator.OneOf("A", "SRV")},
						},
						"healthcheck": schema.SingleNestedAttribute{
							MarkdownDescription: "How the edges check that the origin is healthy.",
							Required:            true,
							Attributes: map[string]schema.Attribute{
								"path": schema.StringAttribute{
									MarkdownDescription: "Path the health check requests, for example `/healthz`.",
									Required:            true,
									Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
								},
								"host": schema.StringAttribute{
									MarkdownDescription: "Base URL the health check requests `path` from. " +
										"When not set, it is the origin's own URL: `https://<fqdn>`, or `http://<fqdn>` when `ssl` is false.",
									Optional:   true,
									Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
								},
								"expected_status_code": schema.Int64Attribute{
									MarkdownDescription: "HTTP status a healthy origin returns. Defaults to `200`.",
									Optional:            true,
									Computed:            true,
									Default:             int64default.StaticInt64(200),
									Validators:          []validator.Int64{int64validator.Between(100, 599)},
								},
								"interval_ms": schema.Int64Attribute{
									MarkdownDescription: "Milliseconds between checks. At least `15000`, the default.",
									Optional:            true,
									Computed:            true,
									Default:             int64default.StaticInt64(cdnMinHealthcheckIntervalMs),
									Validators:          []validator.Int64{int64validator.AtLeast(cdnMinHealthcheckIntervalMs)},
								},
								"timeout_ms": schema.Int64Attribute{
									MarkdownDescription: "Milliseconds a check may take. Defaults to `2000`.",
									Optional:            true,
									Computed:            true,
									Default:             int64default.StaticInt64(2000),
									Validators:          []validator.Int64{int64validator.AtLeast(1)},
								},
								"min_valid_checks": schema.Int64Attribute{
									MarkdownDescription: "Number of passing checks after which the origin counts as healthy. Defaults to `2`.",
									Optional:            true,
									Computed:            true,
									Default:             int64default.StaticInt64(2),
									Validators:          []validator.Int64{int64validator.Between(1, 255)},
								},
							},
						},
					},
				},
			},
			"flags": schema.SingleNestedAttribute{
				MarkdownDescription: "Switches for the resource. Every flag has a default, so the block is optional.",
				Optional:            true,
				Computed:            true,
				Default:             objectdefault.StaticValue(cdnFlagsObject(cdnDefaultFlags)),
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether the CDN serves the resource. Defaults to `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(cdnDefaultFlags.Enabled),
					},
					"force_https": schema.BoolAttribute{
						MarkdownDescription: "Whether the edges redirect HTTP requests to HTTPS. Defaults to `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(cdnDefaultFlags.ForceHTTPS),
					},
					"auth_pass_through": schema.BoolAttribute{
						MarkdownDescription: "Whether the edges forward the `Authorization` request header to the origin. Defaults to `false`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(cdnDefaultFlags.AuthPassThrough),
					},
					"override_backend_host": schema.BoolAttribute{
						MarkdownDescription: "Whether the edges send the origin's own `fqdn` as the `Host` header instead of the public hostname. Defaults to `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(cdnDefaultFlags.OverrideBackendHost),
					},
					"override_health_host": schema.BoolAttribute{
						MarkdownDescription: "Whether health checks send the origin's own `fqdn` as the `Host` header. Defaults to `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(cdnDefaultFlags.OverrideHealthHost),
					},
					"use_i3d_static_dns": schema.BoolAttribute{
						MarkdownDescription: "Whether the resource uses i3D.net's static DNS. Defaults to `false`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(cdnDefaultFlags.UseI3DStaticDNS),
					},
				},
			},
			"predefined_allowlist": schema.StringAttribute{
				MarkdownDescription: "Key of a predefined allowlist that limits who can reach the resource. " +
					"The `i3dnet_cdn_predefined_allowlists` data source lists the keys. When not set, the resource is open to everyone. " +
					"When set, Terraform sends the allowlist to the CDN API before the hostnames and flags, and removes it only after them.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"cname_target": schema.StringAttribute{
				MarkdownDescription: "Target for each hostname's CNAME record.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

// ValidateConfig rejects hostnames that are listed twice.
func (r *cdnResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var hostnames types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("hostnames"), &hostnames)...)
	if resp.Diagnostics.HasError() || hostnames.IsNull() || hostnames.IsUnknown() {
		return
	}

	var models []cdnHostnameModel
	resp.Diagnostics.Append(hostnames.ElementsAs(ctx, &models, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	seen := map[string]bool{}
	for i, h := range models {
		if h.FQDN.IsNull() || h.FQDN.IsUnknown() {
			continue
		}
		fqdn := h.FQDN.ValueString()
		if seen[fqdn] {
			resp.Diagnostics.AddAttributeError(
				path.Root("hostnames").AtListIndex(i).AtName("fqdn"),
				"Duplicate hostname",
				fmt.Sprintf("%q is listed more than once.", fqdn),
			)
		}
		seen[fqdn] = true
	}
}

// ModifyPlan refuses to move an existing resource to another group or
// initiative. A CDN API key belongs to one group and initiative, so after a
// move the key could no longer change or delete the resource.
func (r *cdnResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	for _, attribute := range []string{"group_code", "initiative_code"} {
		var planned, current types.String
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(attribute), &planned)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(attribute), &current)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if planned.IsUnknown() || planned.Equal(current) {
			continue
		}

		resp.Diagnostics.AddAttributeError(
			path.Root(attribute),
			"Cannot move a CDN resource to another group or initiative",
			fmt.Sprintf("The resource belongs to %s %q in the CDN API, and the configuration says %q. "+
				"Terraform does not move resources between groups or initiatives, because the CDN API key could no longer manage the resource afterwards. "+
				"Set %s back to %q, or move the resource in the CDN dashboard and then manage it with a key for its new group and initiative.",
				attribute, current.ValueString(), planned.ValueString(), attribute, current.ValueString()),
		)
	}
}

func (r *cdnResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cdnResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, cdnDefaultCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	desired, diags := plan.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := r.client.CreateResource(ctx, cdn_api.CreateResourceRequest{
		Name:           plan.Name.ValueString(),
		GroupCode:      plan.GroupCode.ValueString(),
		InitiativeCode: plan.InitiativeCode.ValueString(),
	})
	if err != nil {
		detail := err.Error()
		if createOutcomeUnknown(err) {
			detail += fmt.Sprintf("\n\nThe CDN API may have created the resource anyway. Before you apply again, look in the CDN dashboard "+
				"for a resource named %q. If it exists, import it (terraform import <address> <uid>) or delete it, so a second one is not created.",
				plan.Name.ValueString())
		}
		resp.Diagnostics.AddError("Error creating CDN resource", detail)

		return
	}

	// Save the resource before configuring it. If a later step fails, Terraform
	// keeps it in the state as tainted and replaces it on the next apply, instead
	// of losing track of it.
	plan.ID = types.StringValue(uid)
	plan.CNAMETarget = types.StringValue(cdnCNAMETarget)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.apply(ctx, uid, nil, desired)...)
}

func (r *cdnResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cdnResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid := state.ID.ValueString()

	current, err := r.client.GetResource(ctx, uid)
	if cdn_api.IsNotFound(err) {
		// The API also answers 404 when it cannot read its database. Forget the
		// resource only when the resource listing confirms it is gone, or the
		// next apply would create a second one while the first still serves.
		exists, existsErr := r.client.ResourceExists(ctx, uid)
		if existsErr != nil {
			resp.Diagnostics.AddError("Error reading CDN resource",
				fmt.Sprintf("%s\n\nCould not confirm that the resource is gone: %s", err, existsErr))
			return
		}
		if !exists {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading CDN resource",
			fmt.Sprintf("%s\n\nThe resource is still listed by the CDN API, so this is likely a temporary API error. Try again later.", err))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading CDN resource", err.Error())
		return
	}

	settings, err := r.readSettings(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading CDN resource", err.Error())
		return
	}

	state.ID = types.StringValue(current.UID)
	if current.UID == "" {
		state.ID = types.StringValue(uid)
	}
	state.Name = types.StringValue(current.Name)
	state.GroupCode = types.StringValue(current.GroupCode)
	state.InitiativeCode = types.StringValue(current.InitiativeCode)
	state.CNAMETarget = types.StringValue(cdnCNAMETarget)

	resp.Diagnostics.Append(state.setSettings(ctx, settings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cdnResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state cdnResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid := state.ID.ValueString()

	desired, diags := plan.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	current, diags := state.toAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Name.Equal(state.Name) {
		if err := r.client.RenameResource(ctx, uid, plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error renaming CDN resource", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(r.apply(ctx, uid, &current, desired)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID
	plan.CNAMETarget = types.StringValue(cdnCNAMETarget)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cdnResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cdnResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, cdnDefaultDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if err := r.client.DeleteResource(ctx, state.ID.ValueString(), cdnDeleteRetryInterval); err != nil {
		resp.Diagnostics.AddError("Error deleting CDN resource", err.Error())
	}
}

func (r *cdnResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// cdnSettings are the settings of a resource this provider manages, in API form.
type cdnSettings struct {
	Backend             cdn_api.Backend
	Hostnames           []cdn_api.Hostname
	Flags               cdn_api.Flags
	PredefinedAllowlist string
}

func (r *cdnResource) readSettings(ctx context.Context, uid string) (cdnSettings, error) {
	var s cdnSettings

	backend, err := r.client.GetBackend(ctx, uid)
	if err != nil {
		return s, err
	}
	s.Backend = *backend

	s.Hostnames, err = r.client.GetHostnames(ctx, uid)
	if err != nil {
		return s, err
	}

	flags, err := r.client.GetFlags(ctx, uid)
	if err != nil {
		return s, err
	}
	s.Flags = *flags

	s.PredefinedAllowlist, err = r.client.GetPredefinedAllowList(ctx, uid)
	if err != nil {
		return s, err
	}

	return s, nil
}

// apply writes the settings that differ from current (all of them when current
// is nil). The API accepts an allowlist before the hostnames and flags that
// publish the resource, and loses it only after them; a failure in between
// leaves the resource restricted rather than open.
func (r *cdnResource) apply(ctx context.Context, uid string, current *cdnSettings, desired cdnSettings) diag.Diagnostics {
	var diags diag.Diagnostics

	changed := func(differs bool) bool { return current == nil || differs }

	fail := func(what string, err error) diag.Diagnostics {
		diags.AddError("Error configuring CDN resource", fmt.Sprintf("setting the %s of resource %s: %s", what, uid, err))
		return diags
	}

	if desired.PredefinedAllowlist != "" && changed(current != nil && current.PredefinedAllowlist != desired.PredefinedAllowlist) {
		if err := r.client.PutPredefinedAllowList(ctx, uid, desired.PredefinedAllowlist); err != nil {
			return fail("predefined allowlist", err)
		}
	}

	if changed(current != nil && !backendEqual(current.Backend, desired.Backend)) {
		if err := r.client.PutBackend(ctx, uid, desired.Backend); err != nil {
			return fail("origins", err)
		}
	}

	if changed(current != nil && !hostnamesEqual(current.Hostnames, desired.Hostnames)) {
		if err := r.client.PutHostnames(ctx, uid, desired.Hostnames); err != nil {
			return fail("hostnames", err)
		}
	}

	if changed(current != nil && current.Flags != desired.Flags) {
		if err := r.client.PutFlags(ctx, uid, desired.Flags); err != nil {
			return fail("flags", err)
		}
	}

	if desired.PredefinedAllowlist == "" && current != nil && current.PredefinedAllowlist != "" {
		if err := r.client.DeleteSetting(ctx, uid, "predefined-allowlist"); err != nil {
			return fail("predefined allowlist", err)
		}
	}

	return diags
}

func backendEqual(a, b cdn_api.Backend) bool {
	if a.Strategy != b.Strategy || len(a.Origins) != len(b.Origins) {
		return false
	}

	for i := range a.Origins {
		x, y := a.Origins[i], b.Origins[i]
		if x.FQDN != y.FQDN || x.Port != y.Port || x.SSL != y.SSL || x.VerifySSL != y.VerifySSL ||
			x.Location != y.Location || x.DNSRecord != y.DNSRecord {
			return false
		}
		if (x.Healthcheck == nil) != (y.Healthcheck == nil) {
			return false
		}
		if x.Healthcheck != nil && *x.Healthcheck != *y.Healthcheck {
			return false
		}
	}

	return true
}

func hostnamesEqual(a, b []cdn_api.Hostname) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// defaultHealthcheckHost is the health check base URL of an origin whose
// configuration does not set one.
func defaultHealthcheckHost(fqdn string, ssl bool) string {
	if ssl {
		return "https://" + fqdn
	}

	return "http://" + fqdn
}

// toAPI converts the model's settings to their API form.
func (m cdnResourceModel) toAPI(ctx context.Context) (cdnSettings, diag.Diagnostics) {
	var (
		s     cdnSettings
		diags diag.Diagnostics
	)

	s.Backend.Strategy = m.Strategy.ValueString()

	var origins []cdnOriginModel
	diags.Append(m.Origins.ElementsAs(ctx, &origins, false)...)
	for _, o := range origins {
		var hc cdnHealthcheckModel
		diags.Append(o.Healthcheck.As(ctx, &hc, objectAsOptions)...)

		host := hc.Host.ValueString()
		if hc.Host.IsNull() {
			host = defaultHealthcheckHost(o.FQDN.ValueString(), o.SSL.ValueBool())
		}

		s.Backend.Origins = append(s.Backend.Origins, cdn_api.Origin{
			FQDN:      o.FQDN.ValueString(),
			Port:      o.Port.ValueInt64(),
			SSL:       o.SSL.ValueBool(),
			VerifySSL: o.VerifySSL.ValueBool(),
			Location:  o.Location.ValueString(),
			DNSRecord: o.DNSRecord.ValueString(),
			Healthcheck: &cdn_api.Healthcheck{
				Host:               host,
				Path:               hc.Path.ValueString(),
				ExpectedStatusCode: hc.ExpectedStatusCode.ValueInt64(),
				TimeoutMs:          hc.TimeoutMs.ValueInt64(),
				IntervalMs:         hc.IntervalMs.ValueInt64(),
				MinValidChecks:     hc.MinValidChecks.ValueInt64(),
			},
		})
	}

	var hostnames []cdnHostnameModel
	diags.Append(m.Hostnames.ElementsAs(ctx, &hostnames, false)...)
	for _, h := range hostnames {
		s.Hostnames = append(s.Hostnames, cdn_api.Hostname{FQDN: h.FQDN.ValueString(), SSLCN: h.SSLCN.ValueString()})
	}

	var flags cdnFlagsModel
	diags.Append(m.Flags.As(ctx, &flags, objectAsOptions)...)
	s.Flags = cdn_api.Flags{
		Enabled:             flags.Enabled.ValueBool(),
		ForceHTTPS:          flags.ForceHTTPS.ValueBool(),
		AuthPassThrough:     flags.AuthPassThrough.ValueBool(),
		OverrideBackendHost: flags.OverrideBackendHost.ValueBool(),
		OverrideHealthHost:  flags.OverrideHealthHost.ValueBool(),
		UseI3DStaticDNS:     flags.UseI3DStaticDNS.ValueBool(),
	}

	s.PredefinedAllowlist = m.PredefinedAllowlist.ValueString()

	return s, diags
}

// setSettings stores settings read from the API in the model. A health check
// host equal to the origin's default stays unset when the model had it unset,
// so a configuration that leaves it out does not show a difference.
func (m *cdnResourceModel) setSettings(ctx context.Context, s cdnSettings) diag.Diagnostics {
	var diags diag.Diagnostics

	// Which origins had no health check host before, by position.
	var priorOrigins []cdnOriginModel
	if !m.Origins.IsNull() && !m.Origins.IsUnknown() {
		diags.Append(m.Origins.ElementsAs(ctx, &priorOrigins, false)...)
	}
	hostUnset := func(i int) bool {
		if i >= len(priorOrigins) {
			return true
		}
		var hc cdnHealthcheckModel
		if priorOrigins[i].Healthcheck.IsNull() || priorOrigins[i].Healthcheck.IsUnknown() {
			return true
		}
		diags.Append(priorOrigins[i].Healthcheck.As(ctx, &hc, objectAsOptions)...)
		return hc.Host.IsNull()
	}

	strategy := s.Backend.Strategy
	if strategy == "" || strategy == "NO_STRATEGY" {
		m.Strategy = types.StringNull()
	} else {
		m.Strategy = types.StringValue(strategy)
	}

	origins := make([]attr.Value, 0, len(s.Backend.Origins))
	for i, o := range s.Backend.Origins {
		hcValues := map[string]attr.Value{
			"path":                 types.StringNull(),
			"host":                 types.StringNull(),
			"expected_status_code": types.Int64Null(),
			"interval_ms":          types.Int64Null(),
			"timeout_ms":           types.Int64Null(),
			"min_valid_checks":     types.Int64Null(),
		}
		if o.Healthcheck != nil {
			hc := o.Healthcheck
			host := types.StringValue(hc.Host)
			if hc.Host == "" || (hc.Host == defaultHealthcheckHost(o.FQDN, o.SSL) && hostUnset(i)) {
				host = types.StringNull()
			}
			hcValues = map[string]attr.Value{
				"path":                 types.StringValue(hc.Path),
				"host":                 host,
				"expected_status_code": types.Int64Value(hc.ExpectedStatusCode),
				"interval_ms":          types.Int64Value(hc.IntervalMs),
				"timeout_ms":           types.Int64Value(hc.TimeoutMs),
				"min_valid_checks":     types.Int64Value(hc.MinValidChecks),
			}
		}
		healthcheck, d := types.ObjectValue(cdnHealthcheckAttrTypes, hcValues)
		diags.Append(d...)

		origin, d := types.ObjectValue(cdnOriginAttrTypes, map[string]attr.Value{
			"fqdn":        types.StringValue(o.FQDN),
			"port":        types.Int64Value(o.Port),
			"ssl":         types.BoolValue(o.SSL),
			"verify_ssl":  types.BoolValue(o.VerifySSL),
			"location":    types.StringValue(o.Location),
			"dns_record":  types.StringValue(o.DNSRecord),
			"healthcheck": healthcheck,
		})
		diags.Append(d...)
		origins = append(origins, origin)
	}
	originList, d := types.ListValue(types.ObjectType{AttrTypes: cdnOriginAttrTypes}, origins)
	diags.Append(d...)
	m.Origins = originList

	hostnames := make([]attr.Value, 0, len(s.Hostnames))
	for _, h := range s.Hostnames {
		hostname, d := types.ObjectValue(cdnHostnameAttrTypes, map[string]attr.Value{
			"fqdn":   types.StringValue(h.FQDN),
			"ssl_cn": types.StringValue(h.SSLCN),
		})
		diags.Append(d...)
		hostnames = append(hostnames, hostname)
	}
	hostnameList, d := types.ListValue(types.ObjectType{AttrTypes: cdnHostnameAttrTypes}, hostnames)
	diags.Append(d...)
	m.Hostnames = hostnameList

	m.Flags = cdnFlagsObject(s.Flags)

	if s.PredefinedAllowlist == "" {
		m.PredefinedAllowlist = types.StringNull()
	} else {
		m.PredefinedAllowlist = types.StringValue(s.PredefinedAllowlist)
	}

	return diags
}

func cdnFlagsObject(f cdn_api.Flags) types.Object {
	return types.ObjectValueMust(cdnFlagsAttrTypes, map[string]attr.Value{
		"enabled":               types.BoolValue(f.Enabled),
		"force_https":           types.BoolValue(f.ForceHTTPS),
		"auth_pass_through":     types.BoolValue(f.AuthPassThrough),
		"override_backend_host": types.BoolValue(f.OverrideBackendHost),
		"override_health_host":  types.BoolValue(f.OverrideHealthHost),
		"use_i3d_static_dns":    types.BoolValue(f.UseI3DStaticDNS),
	})
}

// createOutcomeUnknown reports whether a failed create may still have created
// the resource.
func createOutcomeUnknown(err error) bool {
	return errors.Is(err, cdn_api.ErrCreateOutcomeUnknown)
}

// objectAsOptions converts objects whose attributes may be unknown or null
// without failing.
var objectAsOptions = basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}

// noTrailingDot matches a host name that does not end with a dot.
var noTrailingDot = regexp.MustCompile(`[^.]$`)
