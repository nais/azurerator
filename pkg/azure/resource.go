package azure

// Resource contains metadata that identifies a resource (e.g. User, Groups, Application, or Service Principal) within Azure AD.
type Resource struct {
	Name          string        `json:"name"`
	ClientId      string        `json:"clientId"`
	ObjectId      string        `json:"-"`
	PrincipalType PrincipalType `json:"-"`
}
