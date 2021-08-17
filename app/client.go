package app

import (
	"os/exec"

	"github.com/hashicorp/go-hclog"
	hcplugin "github.com/hashicorp/go-plugin"
	"github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

func clientConfig(commandName string, commandArgs []string) *hcplugin.ClientConfig {
	return &hcplugin.ClientConfig{
		HandshakeConfig:  framework.Handshake(),
		AllowedProtocols: []hcplugin.Protocol{hcplugin.ProtocolGRPC},
		Plugins: map[string]hcplugin.Plugin{
			string(framework.PluginKindObjectStore):  framework.NewObjectStorePlugin(framework.ClientLogger(nil)),
			string(framework.PluginKindPluginLister): &framework.PluginListerPlugin{},
		},
		Logger: hclog.Default(),
		Cmd:    exec.Command(commandName, commandArgs...),
	}
}

// client creates a new go-plugin Client with support for all of Velero's plugin kinds (BackupItemAction, VolumeSnapshotter,
// ObjectStore, PluginLister, RestoreItemAction).
func Client() *hcplugin.Client {
	return hcplugin.NewClient(clientConfig("/plugins/velero-plugin-for-aws", nil))
}
