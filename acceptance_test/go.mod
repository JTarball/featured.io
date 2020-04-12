module github.com/featured.io/acceptance_test

// IMPORTANT NOTE: terratest doesnt support v1.18.1 k8s
// therefore we need to set to v1.17 for now here

go 1.14

require (
	github.com/gruntwork-io/gruntwork-cli v0.5.1
	github.com/gruntwork-io/terratest v0.26.3
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.5.1
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
)
