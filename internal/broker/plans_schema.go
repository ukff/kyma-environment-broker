package broker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/networking"
)

type RootSchema struct {
	Schema string `json:"$schema"`
	Type
	Properties interface{} `json:"properties"`
	Required   []string    `json:"required"`

	// Specified to true enables form view on website
	ShowFormView bool `json:"_show_form_view"`
	// Specifies in what order properties will be displayed on the form
	ControlsOrder []string `json:"_controlsOrder"`
}

type ProvisioningProperties struct {
	UpdateProperties

	Name        NameType        `json:"name"`
	ShootName   *Type           `json:"shootName,omitempty"`
	ShootDomain *Type           `json:"shootDomain,omitempty"`
	Region      *Type           `json:"region,omitempty"`
	Networking  *NetworkingType `json:"networking,omitempty"`
	Modules     *Modules        `json:"modules,omitempty"`
}

type UpdateProperties struct {
	Kubeconfig     *Type     `json:"kubeconfig,omitempty"`
	AutoScalerMin  *Type     `json:"autoScalerMin,omitempty"`
	AutoScalerMax  *Type     `json:"autoScalerMax,omitempty"`
	OIDC           *OIDCType `json:"oidc,omitempty"`
	Administrators *Type     `json:"administrators,omitempty"`
	MachineType    *Type     `json:"machineType,omitempty"`
}

func (up *UpdateProperties) IncludeAdditional() {
	up.OIDC = NewOIDCSchema()
	up.Administrators = AdministratorsProperty()
}

type NetworkingProperties struct {
	Nodes Type `json:"nodes"`
}

type NetworkingType struct {
	Type
	Properties NetworkingProperties `json:"properties"`
	Required   []string             `json:"required"`
}

type OIDCProperties struct {
	ClientID       Type `json:"clientID"`
	GroupsClaim    Type `json:"groupsClaim"`
	IssuerURL      Type `json:"issuerURL"`
	SigningAlgs    Type `json:"signingAlgs"`
	UsernameClaim  Type `json:"usernameClaim"`
	UsernamePrefix Type `json:"usernamePrefix"`
}

type OIDCType struct {
	Type
	Properties OIDCProperties `json:"properties"`
	Required   []string       `json:"required"`
}

type Type struct {
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Minimum     int    `json:"minimum,omitempty"`
	Maximum     int    `json:"maximum,omitempty"`
	MinLength   int    `json:"minLength,omitempty"`
	MaxLength   int    `json:"maxLength,omitempty"`

	// Regex pattern to match against string type of fields.
	// If not specified for strings user can pass empty string with whitespaces only.
	Pattern         string            `json:"pattern,omitempty"`
	Default         interface{}       `json:"default,omitempty"`
	Example         interface{}       `json:"example,omitempty"`
	Enum            []interface{}     `json:"enum,omitempty"`
	EnumDisplayName map[string]string `json:"_enumDisplayName,omitempty"`
	Items           *Type             `json:"items,omitempty"`
	AdditionalItems interface{}       `json:"additionalItems,omitempty"`
	UniqueItems     interface{}       `json:"uniqueItems,omitempty"`
	ReadOnly        interface{}       `json:"readOnly,omitempty"`
}

type NameType struct {
	Type
	BTPdefaultTemplate BTPdefaultTemplate `json:"_BTPdefaultTemplate,omitempty"`
}

type BTPdefaultTemplate struct {
	Elements  []string `json:"elements,omitempty"`
	Separator string   `json:"separator,omitempty"`
}

type Modules struct {
	Type
	ControlsOrder []string      `json:"_controlsOrder,omitempty"`
	OneOf         []interface{} `json:"oneOf,omitempty"`
}

type ModulesDefault struct {
	Type
	Properties ModulesDefaultProperties `json:"properties,omitempty"`
}

type ModulesDefaultProperties struct {
	UseDefault Type `json:"useDefault,omitempty"`
}

type ModulesCustom struct {
	Type
	Properties ModulesCustomProperties `json:"properties,omitempty"`
}

type ModulesCustomProperties struct {
	List ModulesCustomList `json:"list,omitempty"`
}

type ModulesCustomList struct {
	Type
	Items ModulesCustomListItems `json:"items,omitempty"`
}

type ModulesCustomListItems struct {
	Type
	ControlsOrder []string                         `json:"_controlsOrder,omitempty"`
	Properties    ModulesCustomListItemsProperties `json:"properties,omitempty"`
}

type ModulesCustomListItemsProperties struct {
	Name                 Type `json:"name,omitempty"`
	Channel              Type `json:"channel,omitempty"`
	CustomResourcePolicy Type `json:"customResourcePolicy,omitempty"`
}

func NewModulesSchema() *Modules {
	return &Modules{
		Type: Type{
			Type:        "object",
			Description: "Use default modules or provide your custom list of modules.",
		},
		ControlsOrder: []string{"useDefault", "modules"},
		OneOf: []any{
			ModulesDefault{
				Type: Type{
					Type:        "object",
					Title:       "Default",
					Description: "Check the default modules at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud.\n",
				},
				Properties: ModulesDefaultProperties{
					Type{
						Type:        "boolean",
						Description: `"Select a module’s technical key from the list available at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud. You can only use a module’s technical key once."`,
						Default:     true,
						ReadOnly:    true,
					},
				},
			},
			ModulesCustom{
				Type: Type{
					Type:        "object",
					Title:       "Custom",
					Description: "Check the default modules at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud.\n",
				},
				Properties: ModulesCustomProperties{
					ModulesCustomList{
						Type: Type{
							Type:        "array",
							UniqueItems: true,
							Description: `"Select a module’s technical key from the list available at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud. You can only use a module’s technical key once."`,
						},
						Items: ModulesCustomListItems{
							Type: Type{
								Type: "object",
							},
							ControlsOrder: []string{"name", "channel", "customResourcePolicy"},
							Properties: ModulesCustomListItemsProperties{
								Name: Type{
									Type:        "string",
									Title:       "name",
									MinLength:   1,
									Description: `"Select a module’s technical key from the list available at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud. You can only use a module’s technical key once."`,
								},
								Channel: Type{
									Type:        "string",
									Default:     "regular",
									Description: "Select your preferred release channel.\n",
									Enum:        ToInterfaceSlice([]string{"regular", "fast"}),
									EnumDisplayName: map[string]string{
										"regular": "Regular - default version\n",
										"fast":    "Fast - latest version",
									},
								},
								CustomResourcePolicy: Type{
									Type:        "string",
									Description: "Select your preferred CustomResourcePolicy setting.",
									Default:     "CreateAndDelete",
									Enum:        ToInterfaceSlice([]string{"CreateAndDelete", "Ignore"}),
									EnumDisplayName: map[string]string{
										"CreateAndDelete": "CreateAndDelete - default module resource is created or deleted.",
										"Ignore":          "Ignore - module resource is not created.",
									},
								},
							},
						},
					}},
			}},
	}
}

func NameProperty() NameType {
	return NameType{
		Type: Type{
			Type:  "string",
			Title: "Cluster Name",
			// Allows for all alphanumeric characters and '-'
			Pattern:   "^[a-zA-Z0-9-]*$",
			MinLength: 1,
		},
		BTPdefaultTemplate: BTPdefaultTemplate{
			Elements: []string{"saSubdomain"},
		},
	}
}

func KubeconfigProperty() *Type {
	return &Type{
		Type:  "string",
		Title: "Kubeconfig contents",
	}
}

func ShootNameProperty() *Type {
	return &Type{
		Type:      "string",
		Title:     "Shoot name",
		Pattern:   "^[a-zA-Z0-9-]*$",
		MinLength: 1,
	}
}

func ShootDomainProperty() *Type {
	return &Type{
		Type:      "string",
		Title:     "Shoot domain",
		Pattern:   "^[a-zA-Z0-9-\\.]*$",
		MinLength: 1,
	}
}

// NewProvisioningProperties creates a new properties for different plans
// Note that the order of properties will be the same in the form on the website
func NewProvisioningProperties(machineTypesDisplay map[string]string, machineTypes, regions []string, update bool) ProvisioningProperties {

	properties := ProvisioningProperties{
		UpdateProperties: UpdateProperties{
			AutoScalerMin: &Type{
				Type:        "integer",
				Minimum:     2,
				Default:     3,
				Description: "Specifies the minimum number of virtual machines to create",
			},
			AutoScalerMax: &Type{
				Type:        "integer",
				Minimum:     2,
				Maximum:     80,
				Default:     20,
				Description: "Specifies the maximum number of virtual machines to create",
			},
			MachineType: &Type{
				Type:            "string",
				Enum:            ToInterfaceSlice(machineTypes),
				EnumDisplayName: machineTypesDisplay,
			},
		},
		Name: NameProperty(),
		Region: &Type{
			Type: "string",
			Enum: ToInterfaceSlice(regions),
		},
		Networking: NewNetworkingSchema(),
		Modules:    NewModulesSchema(),
	}

	if update {
		properties.AutoScalerMax.Default = nil
		properties.AutoScalerMin.Default = nil
	}

	return properties
}

func NewNetworkingSchema() *NetworkingType {
	seedCIDRs := strings.Join(networking.GardenerSeedCIDRs, ", ")
	return &NetworkingType{
		Type: Type{Type: "object", Description: "Networking configuration. These values are immutable and cannot be updated later."},
		Properties: NetworkingProperties{
			Nodes: Type{Type: "string", Title: "CIDR range for nodes", Description: fmt.Sprintf("CIDR range for nodes, must not overlap with the following CIDRs: %s, %s, %s", networking.DefaultPodsCIDR, networking.DefaultServicesCIDR, seedCIDRs),
				Default: networking.DefaultNodesCIDR},
		},
		Required: []string{"nodes"},
	}
}

func NewOIDCSchema() *OIDCType {
	return &OIDCType{
		Type: Type{Type: "object", Description: "OIDC configuration"},
		Properties: OIDCProperties{
			ClientID:       Type{Type: "string", Description: "The client ID for the OpenID Connect client."},
			IssuerURL:      Type{Type: "string", Description: "The URL of the OpenID issuer, only HTTPS scheme will be accepted."},
			GroupsClaim:    Type{Type: "string", Description: "If provided, the name of a custom OpenID Connect claim for specifying user groups."},
			UsernameClaim:  Type{Type: "string", Description: "The OpenID claim to use as the user name."},
			UsernamePrefix: Type{Type: "string", Description: "If provided, all usernames will be prefixed with this value. If not provided, username claims other than 'email' are prefixed by the issuer URL to avoid clashes. To skip any prefixing, provide the value '-' (dash character without additional characters)."},
			SigningAlgs: Type{
				Type: "array",
				Items: &Type{
					Type: "string",
				},
				Description: "Comma separated list of allowed JOSE asymmetric signing algorithms, for example, RS256, ES256",
			},
		},
		Required: []string{"clientID", "issuerURL"},
	}
}

func NewSchemaWithOnlyNameRequired(properties interface{}, update bool) *RootSchema {
	return NewSchemaForOwnCluster(properties, update, []string{"name"})
}

func NewSchemaForOwnCluster(properties interface{}, update bool, required []string) *RootSchema {
	schema := &RootSchema{
		Schema: "http://json-schema.org/draft-04/schema#",
		Type: Type{
			Type: "object",
		},
		Properties:   properties,
		ShowFormView: true,
		Required:     required,
	}

	if update {
		schema.Required = []string{}
	}

	return schema
}

func unmarshalOrPanic(from, to interface{}) interface{} {
	if from != nil {
		marshaled := Marshal(from)
		err := json.Unmarshal(marshaled, to)
		if err != nil {
			panic(err)
		}
	}
	return to
}

func DefaultControlsOrder() []string {
	return []string{"name", "kubeconfig", "shootName", "shootDomain", "region", "machineType", "autoScalerMin", "autoScalerMax", "zonesCount", "networking", "oidc", "administrators"}
}

func ToInterfaceSlice(input []string) []interface{} {
	interfaces := make([]interface{}, len(input))
	for i, item := range input {
		interfaces[i] = item
	}
	return interfaces
}

func AdministratorsProperty() *Type {
	return &Type{
		Type:        "array",
		Title:       "Administrators",
		Description: "Specifies the list of runtime administrators",
		Items: &Type{
			Type: "string",
		},
	}
}
