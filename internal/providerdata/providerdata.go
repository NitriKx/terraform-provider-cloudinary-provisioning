// Package providerdata defines the shared data type passed from the provider
// to all resources and data sources via Configure().
package providerdata

import "github.com/cloudinary/account-provisioning-go/cldprovisioning"

// ProviderData carries the Cloudinary Provisioning API client together with
// the credentials that were used to configure it. Resources and data sources
// receive a *ProviderData via their Configure() call.
type ProviderData struct {
	Client    *cldprovisioning.CldProvisioning
	APIKey    string
	AccountID string
}
