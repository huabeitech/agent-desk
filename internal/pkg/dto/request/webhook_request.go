package request

type OrgSyncEventData struct {
	// Organization fields
	OrgID        string `json:"org_id"`
	ID           string `json:"id,omitempty"`
	GlobalOrgID  string `json:"global_org_id,omitempty"`
	OrgName      string `json:"org_name"`
	Name         string `json:"name,omitempty"`
	Slug         string `json:"slug,omitempty"`
	UserID       string `json:"user_id"`
	UserEmail    string `json:"user_email"`
	BillingEmail string `json:"billingEmail,omitempty"`
	UserName     string `json:"user_name"`
	Role         string `json:"role"`
	Plan         string `json:"plan"`

	// Company fields
	CRMCompanyID      string `json:"crm_company_id,omitempty"`
	DeskCompanyID     string `json:"desk_company_id,omitempty"`
	CompanyID         string `json:"company_id,omitempty"`
	DomainName        string `json:"domain_name,omitempty"`
	Domain            string `json:"domain,omitempty"`
	Address           string `json:"address,omitempty"`
	Tier              string `json:"tier,omitempty"`
	TaxCode           string `json:"tax_code,omitempty"`
	AccountOwnerEmail string `json:"account_owner_email,omitempty"`

	// Customer fields
	CRMPersonID    string `json:"crm_person_id,omitempty"`
	DeskCustomerID string `json:"desk_customer_id,omitempty"`
	Email          string `json:"email,omitempty"`
	Phone          string `json:"phone,omitempty"`
	AvatarURL      string `json:"avatar_url,omitempty"`
	JobTitle       string `json:"job_title,omitempty"`
	CompanyName    string `json:"company_name,omitempty"`
	Source         string `json:"source,omitempty"`
}

type OrgSyncWebhookRequest struct {
	Event     string           `json:"event"`
	Timestamp string           `json:"timestamp"`
	Data      OrgSyncEventData `json:"data"`
}

type DOSOrgSyncEventData = OrgSyncEventData
type DOSOrgSyncWebhookRequest = OrgSyncWebhookRequest
