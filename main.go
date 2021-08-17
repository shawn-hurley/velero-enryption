// main register a plugin for object storage that will provide for encryption

// the plugin will be responsible for first verifying that a plugin exists for
// the underlying object store i.e aws
// so for velero.io/encryption-aws we will look for aws object store plugin.
// if found, we will state "we also support encryption-aws"

// Deal with encryption key for now, lets just assume user provides ref to a secret that contains it
// that should be mounted onto the pod /encryption/key

package main

import (
	"github.com/hashicorp/go-hclog"
	"github.com/konveyor/encryption-object-store-proxy/app"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

func main() {

	log := hclog.Default()
	r := registry{
		pluginsByID:   make(map[kindAndName]framework.PluginIdentifier),
		pluginsByKind: make(map[framework.PluginKind][]framework.PluginIdentifier),
	}

	c := app.Client()
	client, err := c.Client()
	if err != nil {
		panic(err)
	}
	plugin, err := client.Dispense(framework.PluginKindPluginLister.String())
	if err != nil {
		panic(err)
	}

	lister, ok := plugin.(framework.PluginLister)
	if !ok {
		panic("error with casting to lister")
	}

	plugins, err := lister.ListPlugins()
	if err != nil {
		panic(err)
	}

	for _, p := range plugins {
		r.register(p)
	}

	s := framework.NewServer().BindFlags(pflag.CommandLine)

	if pluginIdentifier, ok := r.pluginsByID[kindAndName{kind: framework.PluginKindObjectStore, name: "velero.io/aws"}]; ok {
		log.Info("Regigistering Plugin", "plugin", pluginIdentifier)

		s.RegisterObjectStore("velero.io/encryption-aws", func(_ logrus.FieldLogger) (interface{}, error) {
			log.Info("Creating new Plugin for id", "plugin", pluginIdentifier)
			return app.New(pluginIdentifier), nil
		})
	}
	s.Serve()
}

type kindAndName struct {
	kind framework.PluginKind
	name string
}

type registry struct {
	pluginsByID   map[kindAndName]framework.PluginIdentifier
	pluginsByKind map[framework.PluginKind][]framework.PluginIdentifier
}

// register registers a PluginIdentifier with the registry.
func (r *registry) register(id framework.PluginIdentifier) error {
	key := kindAndName{kind: id.Kind, name: id.Name}
	if existing, found := r.pluginsByID[key]; found {
		return errors.Errorf("duplicate plugin: %v, %v", existing, key)
	}

	// no need to pass list of existing plugins since the check if this exists was done above
	if err := framework.ValidatePluginName(id.Name, nil); err != nil {
		return errors.Errorf("invalid plugin name %q: %s", id.Name, err)
	}

	r.pluginsByID[key] = id
	r.pluginsByKind[id.Kind] = append(r.pluginsByKind[id.Kind], id)

	return nil
}
