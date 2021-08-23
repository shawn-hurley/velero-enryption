package app

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	hcplugin "github.com/hashicorp/go-plugin"
	"github.com/konveyor/encryption-object-store-proxy/crypto"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/framework"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
)

type EncryptedAwsObjectStoreType struct {
	ID     framework.PluginIdentifier
	logger hclog.Logger
	client *hcplugin.Client
}

var _ velero.ObjectStore = &EncryptedAwsObjectStoreType{}

func (e *EncryptedAwsObjectStoreType) New(_ logrus.FieldLogger) (interface{}, error) {
	e.logger = hclog.Default()

	e.client = hcplugin.NewClient(clientConfig(e.ID.Command, nil))

	return e, nil
}

func New(id framework.PluginIdentifier) velero.ObjectStore {
	return &EncryptedAwsObjectStoreType{
		ID:     id,
		client: hcplugin.NewClient(clientConfig(id.Command, nil)),
		logger: hclog.Default(),
	}
}

func (e *EncryptedAwsObjectStoreType) getObjectStoreClient() (velero.ObjectStore, error) {
	e.logger.Info("getting client From client config")
	c, err := e.client.Client()
	if err != nil {
		e.logger.Info("getting client From client config", "err", err)
		return nil, err
	}

	e.logger.Info("Dispensing client to run aws plugin")
	plugin, err := c.Dispense(framework.PluginKindObjectStore.String())
	if err != nil {
		e.logger.Info("Dispensing client to run aws plugin", "err", err)
		return nil, err
	}

	e.logger.Info("Attempting to cast plugin to ClientDispenser", "pluign", plugin)
	cd, ok := plugin.(framework.ClientDispenser)
	if !ok {
		e.logger.Info("error casting plugin to ClientDispenser", "pluign", plugin, "type", reflect.TypeOf(plugin))
		return nil, fmt.Errorf("unable to get object store")
	}

	e.logger.Info("Attempting to get Client For", "name", e.ID.Name)
	objStore, ok := cd.ClientFor(e.ID.Name).(velero.ObjectStore)
	if !ok {
		e.logger.Info("error casting plugin to ObjectStore", "pluign", objStore, "type", reflect.TypeOf(objStore))
		return nil, fmt.Errorf("unable to get object store")
	}

	return objStore, nil
}

func (e *EncryptedAwsObjectStoreType) Init(config map[string]string) error {
	e.logger.Info("Calling init", "config", config, "id", e.ID)
	o, err := e.getObjectStoreClient()
	if err != nil {
		e.logger.Info("gettingObjectStoreClient Error", "err", err)
		return err
	}
	e.logger.Info("forwarding to plugin", "plugin", e.ID)
	return o.Init(config)
}

func (e *EncryptedAwsObjectStoreType) PutObject(bucket, key string, body io.Reader) error {
	o, err := e.getObjectStoreClient()
	if err != nil {
		return err
	}
	if encryctObjects(key) {
		e.logger.Info("encrypting data", "plugin", e.ID)
		encryptReader, err := crypto.NewStreamEncrypter([]byte("testing123456789"), []byte("testing"), body)
		if err != nil {
			return err
		}
		buf := &bytes.Buffer{}
		err = encryptReader.EncryptData(buf)
		if err != nil {
			return err
		}
		e.logger.Info("encrypting data after", "plugin", e.ID, "hash", encryptReader.Mac.Sum(nil))
		body = buf
	}

	e.logger.Info("forwarding to plugin", "plugin", e.ID)
	return o.PutObject(bucket, key, body)
}

// ObjectExists checks if there is an object with the given key in the object storage bucket.
func (e *EncryptedAwsObjectStoreType) ObjectExists(bucket, key string) (bool, error) {
	e.logger.Info("ObjectExists Call", "bucket", bucket, "key", key)
	o, err := e.getObjectStoreClient()
	if err != nil {
		return false, err
	}
	e.logger.Info("forwarding to plugin", "plugin", e.ID)
	return o.ObjectExists(bucket, key)
}

func (e *EncryptedAwsObjectStoreType) GetObject(bucket, key string) (io.ReadCloser, error) {
	e.logger.Info("GetObject Call", "bucket", bucket, "key", key)
	o, err := e.getObjectStoreClient()
	if err != nil {
		return nil, err
	}
	e.logger.Info("forwarding to plugin", "plugin", e.ID)
	closer, err := o.GetObject(bucket, key)
	if err != nil {
		e.logger.Info("Failure to forward", "error", err, "id", e.ID)
		return closer, err
	}
	if encryctObjects(key) {
		e.logger.Info("Decrypting data", "plugin", e.ID)
		decryptReader, err := crypto.NewStreamDecrypter([]byte("testing123456789"), []byte("testing"), closer)
		if err != nil {
			e.logger.Info("error decrypting data", "error", e.ID, "plugin", e.ID)
			return nil, err
		}
		b := &bytes.Buffer{}
		err = decryptReader.DecryptData(b)
		if err != nil {
			return nil, err
		}
		closer = nopCloser{Reader: b}
	}
	return closer, err
}

func (e *EncryptedAwsObjectStoreType) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {
	e.logger.Info("ListCommonPrefixes Call", "bucket", bucket, "prefix", prefix, "delimiter", delimiter)
	o, err := e.getObjectStoreClient()
	if err != nil {
		return nil, err
	}
	e.logger.Info("forwarding to plugin", "plugin", e.ID)
	return o.ListCommonPrefixes(bucket, prefix, delimiter)
}

func (e *EncryptedAwsObjectStoreType) ListObjects(bucket, prefix string) ([]string, error) {
	e.logger.Info("ListObjects Call", "bucket", bucket, "prefix", prefix)
	o, err := e.getObjectStoreClient()
	if err != nil {
		return nil, err
	}
	e.logger.Info("forwarding to plugin", "plugin", e.ID)
	return o.ListObjects(bucket, prefix)
}

func (e *EncryptedAwsObjectStoreType) DeleteObject(bucket, key string) error {
	e.logger.Info("DeleteObject Call", "bucket", bucket, "key", key)
	o, err := e.getObjectStoreClient()
	if err != nil {
		return err
	}
	e.logger.Info("forwarding to plugin", "plugin", e.ID)
	return o.DeleteObject(bucket, key)
}

func (e *EncryptedAwsObjectStoreType) CreateSignedURL(bucket, key string, ttl time.Duration) (string, error) {
	e.logger.Info("CreateSignedURL Call", "bucket", bucket, "key", key)
	o, err := e.getObjectStoreClient()
	if err != nil {
		return "", err
	}
	e.logger.Info("forwarding to plugin", "plugin", e.ID)
	return o.CreateSignedURL(bucket, key, ttl)
}

const PodVolumeBackups = "podvolumebackups.json.gz"
const VolumeContents = "tar.gz"

func encryctObjects(key string) bool {
	if strings.Contains(key, PodVolumeBackups) || strings.Contains(key, VolumeContents) {
		return true
	}
	return false
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
