// Package manifests deals with creating manifests for all manifests to be installed for the cluster
package manifests

import (
	"bytes"
	"encoding/base64"
	"path/filepath"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/manifests/content/bootkube"
	"github.com/openshift/installer/pkg/asset/tls"
)

const (
	manifestDir = "manifests"
)

// Manifests generates the dependent operator config.yaml files
type Manifests struct {
	kubeSysConfig  *configurationObject
	tectonicConfig *configurationObject
	files          []*asset.File
}

var _ asset.WritableAsset = (*Manifests)(nil)

type genericData map[string]string

// Name returns a human friendly name for the operator
func (m *Manifests) Name() string {
	return "Common Manifests"
}

// Dependencies returns all of the dependencies directly needed by a
// Manifests asset.
func (m *Manifests) Dependencies() []asset.Asset {
	return []asset.Asset{
		&installconfig.InstallConfig{},
		&KubeCoreOperator{},
		&networkOperator{},
		&kubeAddonOperator{},
		&machineAPIOperator{},
		&Tectonic{},
		&tls.RootCA{},
		&tls.EtcdCA{},
		&tls.IngressCertKey{},
		&tls.KubeCA{},
		&tls.AggregatorCA{},
		&tls.ServiceServingCA{},
		&tls.ClusterAPIServerCertKey{},
		&tls.EtcdClientCertKey{},
		&tls.APIServerCertKey{},
		&tls.OpenshiftAPIServerCertKey{},
		&tls.APIServerProxyCertKey{},
		&tls.MCSCertKey{},
		&tls.KubeletCertKey{},
		&tls.ServiceAccountKeyPair{},
		&kubeconfig.Admin{},
		&machine.Worker{},
	}
}

// Generate generates the respective operator config.yml files
func (m *Manifests) Generate(dependencies asset.Parents) error {
	kco := &KubeCoreOperator{}
	no := &networkOperator{}
	addon := &kubeAddonOperator{}
	mao := &machineAPIOperator{}
	installConfig := &installconfig.InstallConfig{}
	dependencies.Get(kco, no, addon, mao, installConfig)

	// kco+no+mao go to kube-system config map
	m.kubeSysConfig = configMap("kube-system", "cluster-config-v1", genericData{
		"kco-config":     string(kco.Files()[0].Data),
		"network-config": string(no.Files()[0].Data),
		"install-config": string(installConfig.Files()[0].Data),
		"mao-config":     string(mao.Files()[0].Data),
	})
	kubeSysConfigData, err := yaml.Marshal(m.kubeSysConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create kube-system/cluster-config-v1 configmap")
	}

	// addon goes to openshift system
	m.tectonicConfig = configMap("tectonic-system", "cluster-config-v1", genericData{
		"addon-config": string(addon.Files()[0].Data),
	})
	tectonicConfigData, err := yaml.Marshal(m.tectonicConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create tectonic-system/cluster-config-v1 configmap")
	}

	m.files = []*asset.File{
		{
			Filename: filepath.Join(manifestDir, "cluster-config.yaml"),
			Data:     kubeSysConfigData,
		},
		{
			Filename: filepath.Join("tectonic", "cluster-config.yaml"),
			Data:     tectonicConfigData,
		},
	}
	m.files = append(m.files, m.generateBootKubeManifests(dependencies)...)

	return nil
}

// Files returns the files generated by the asset.
func (m *Manifests) Files() []*asset.File {
	return m.files
}

func (m *Manifests) generateBootKubeManifests(dependencies asset.Parents) []*asset.File {
	installConfig := &installconfig.InstallConfig{}
	aggregatorCA := &tls.AggregatorCA{}
	apiServerCertKey := &tls.APIServerCertKey{}
	apiServerProxyCertKey := &tls.APIServerProxyCertKey{}
	clusterAPIServerCertKey := &tls.ClusterAPIServerCertKey{}
	etcdCA := &tls.EtcdCA{}
	etcdClientCertKey := &tls.EtcdClientCertKey{}
	kubeCA := &tls.KubeCA{}
	mcsCertKey := &tls.MCSCertKey{}
	openshiftAPIServerCertKey := &tls.OpenshiftAPIServerCertKey{}
	adminKubeconfig := &kubeconfig.Admin{}
	rootCA := &tls.RootCA{}
	serviceAccountKeyPair := &tls.ServiceAccountKeyPair{}
	serviceServingCA := &tls.ServiceServingCA{}
	workerIgnition := &machine.Worker{}
	dependencies.Get(
		installConfig,
		aggregatorCA,
		apiServerCertKey,
		apiServerProxyCertKey,
		clusterAPIServerCertKey,
		etcdCA,
		etcdClientCertKey,
		kubeCA,
		mcsCertKey,
		openshiftAPIServerCertKey,
		adminKubeconfig,
		rootCA,
		serviceAccountKeyPair,
		serviceServingCA,
		workerIgnition,
	)

	templateData := &bootkubeTemplateData{
		AggregatorCaCert:                base64.StdEncoding.EncodeToString(aggregatorCA.Cert()),
		AggregatorCaKey:                 base64.StdEncoding.EncodeToString(aggregatorCA.Key()),
		ApiserverCert:                   base64.StdEncoding.EncodeToString(apiServerCertKey.Cert()),
		ApiserverKey:                    base64.StdEncoding.EncodeToString(apiServerCertKey.Key()),
		ApiserverProxyCert:              base64.StdEncoding.EncodeToString(apiServerProxyCertKey.Cert()),
		ApiserverProxyKey:               base64.StdEncoding.EncodeToString(apiServerProxyCertKey.Key()),
		Base64encodeCloudProviderConfig: "", // FIXME
		ClusterapiCaCert:                base64.StdEncoding.EncodeToString(clusterAPIServerCertKey.Cert()),
		ClusterapiCaKey:                 base64.StdEncoding.EncodeToString(clusterAPIServerCertKey.Key()),
		EtcdCaCert:                      base64.StdEncoding.EncodeToString(etcdCA.Cert()),
		EtcdClientCert:                  base64.StdEncoding.EncodeToString(etcdClientCertKey.Cert()),
		EtcdClientKey:                   base64.StdEncoding.EncodeToString(etcdClientCertKey.Key()),
		KubeCaCert:                      base64.StdEncoding.EncodeToString(kubeCA.Cert()),
		KubeCaKey:                       base64.StdEncoding.EncodeToString(kubeCA.Key()),
		McsTLSCert:                      base64.StdEncoding.EncodeToString(mcsCertKey.Cert()),
		McsTLSKey:                       base64.StdEncoding.EncodeToString(mcsCertKey.Key()),
		OidcCaCert:                      base64.StdEncoding.EncodeToString(kubeCA.Cert()),
		OpenshiftApiserverCert:          base64.StdEncoding.EncodeToString(openshiftAPIServerCertKey.Cert()),
		OpenshiftApiserverKey:           base64.StdEncoding.EncodeToString(openshiftAPIServerCertKey.Key()),
		OpenshiftLoopbackKubeconfig:     base64.StdEncoding.EncodeToString(adminKubeconfig.Files()[0].Data),
		PullSecret:                      base64.StdEncoding.EncodeToString([]byte(installConfig.Config.PullSecret)),
		RootCaCert:                      base64.StdEncoding.EncodeToString(rootCA.Cert()),
		ServiceaccountKey:               base64.StdEncoding.EncodeToString(serviceAccountKeyPair.Private()),
		ServiceaccountPub:               base64.StdEncoding.EncodeToString(serviceAccountKeyPair.Public()),
		ServiceServingCaCert:            base64.StdEncoding.EncodeToString(serviceServingCA.Cert()),
		ServiceServingCaKey:             base64.StdEncoding.EncodeToString(serviceServingCA.Key()),
		TectonicNetworkOperatorImage:    "quay.io/coreos/tectonic-network-operator-dev:3b6952f5a1ba89bb32dd0630faddeaf2779c9a85",
		WorkerIgnConfig:                 base64.StdEncoding.EncodeToString(workerIgnition.Files()[0].Data),
		CVOClusterID:                    installConfig.Config.ClusterID,
	}

	assetData := map[string][]byte{
		"cluster-apiserver-certs.yaml":          applyTemplateData(bootkube.ClusterApiserverCerts, templateData),
		"ign-config.yaml":                       applyTemplateData(bootkube.IgnConfig, templateData),
		"kube-apiserver-secret.yaml":            applyTemplateData(bootkube.KubeApiserverSecret, templateData),
		"kube-cloud-config.yaml":                applyTemplateData(bootkube.KubeCloudConfig, templateData),
		"kube-controller-manager-secret.yaml":   applyTemplateData(bootkube.KubeControllerManagerSecret, templateData),
		"machine-config-server-tls-secret.yaml": applyTemplateData(bootkube.MachineConfigServerTLSSecret, templateData),
		"openshift-apiserver-secret.yaml":       applyTemplateData(bootkube.OpenshiftApiserverSecret, templateData),
		"pull.json":                             applyTemplateData(bootkube.Pull, templateData),
		"tectonic-network-operator.yaml":        applyTemplateData(bootkube.TectonicNetworkOperator, templateData),
		"cvo-overrides.yaml":                    applyTemplateData(bootkube.CVOOverrides, templateData),

		"01-tectonic-namespace.yaml":                       []byte(bootkube.TectonicNamespace),
		"02-ingress-namespace.yaml":                        []byte(bootkube.IngressNamespace),
		"03-openshift-web-console-namespace.yaml":          []byte(bootkube.OpenshiftWebConsoleNamespace),
		"04-openshift-machine-config-operator.yaml":        []byte(bootkube.OpenshiftMachineConfigOperator),
		"05-openshift-cluster-api-namespace.yaml":          []byte(bootkube.OpenshiftClusterAPINamespace),
		"app-version-kind.yaml":                            []byte(bootkube.AppVersionKind),
		"app-version-mao.yaml":                             []byte(bootkube.AppVersionMao),
		"app-version-tectonic-network.yaml":                []byte(bootkube.AppVersionTectonicNetwork),
		"machine-config-operator-01-images-configmap.yaml": []byte(bootkube.MachineConfigOperator01ImagesConfigmap),
		"operatorstatus-crd.yaml":                          []byte(bootkube.OperatorstatusCrd),
	}

	files := make([]*asset.File, 0, len(assetData))
	for name, data := range assetData {
		files = append(files, &asset.File{
			Filename: filepath.Join(manifestDir, name),
			Data:     data,
		})
	}

	return files
}

func applyTemplateData(template *template.Template, templateData interface{}) []byte {
	buf := &bytes.Buffer{}
	if err := template.Execute(buf, templateData); err != nil {
		panic(err)
	}
	return buf.Bytes()
}