package broker

type Options struct {
	HelmRepoUrl               string
	CatalogPath               string
	DefaultNamespace          string
	ServiceCatalogEnabledOnly bool
}
