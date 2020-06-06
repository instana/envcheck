package cluster_test

import (
	"testing"

	v1 "k8s.io/api/apps/v1"

	"github.com/instana/envcheck/cluster"
)

func Test_DaemonConfig_Namespace_should_change_manifests_namespace(t *testing.T) {
	config := cluster.DaemonConfig{Namespace: "bloop"}
	resource := cluster.Daemon(config)
	if resource.Namespace != "bloop" {
		t.Errorf("Namespace=%v, want <bloop>", resource.Namespace)
	}
}

func Test_DaemonConfig_Version_should_change_manifests_version(t *testing.T) {
	config := cluster.DaemonConfig{Version: "1234"}
	resource := cluster.Daemon(config)
	actual := versionLabel(resource)
	if actual != "1234" {
		t.Errorf("label version=%v, want <1234>", actual)
	}
}

func Test_DaemonConfig_Image_should_change_manifests_image(t *testing.T) {
	config := cluster.DaemonConfig{Image: "instana/daemon:latest"}
	resource := cluster.Daemon(config)
	actual := image(resource)
	if actual != "instana/daemon:latest" {
		t.Errorf("label version=%v, want <instana/daemon:latest>", actual)
	}
}

func Test_PingerConfig_Namespace_should_change_manifests_namespace(t *testing.T) {
	config := cluster.PingerConfig{Namespace: "foobar"}
	resource := cluster.Pinger(config)
	actual := resource.Namespace
	if actual != "foobar" {
		t.Errorf("resource.Namespace=%v, want <foobar>", actual)
	}
}

func Test_PingerConfig_Version_should_change_manifests_version(t *testing.T) {
	config := cluster.PingerConfig{Version: "3456"}
	resource := cluster.Pinger(config)
	actual := versionLabel(resource)
	if actual != "3456" {
		t.Errorf("label version=%v, want <3456>", actual)
	}
}

func Test_PingerConfig_Image_should_change_manifests_image(t *testing.T) {
	config := cluster.PingerConfig{Image: "instana/pinger:latest"}
	resource := cluster.Pinger(config)
	actual := image(resource)
	if actual != "instana/pinger:latest" {
		t.Errorf("label version=%v, want <instana/pinger:latest>", actual)
	}
}

func versionLabel(resource *v1.DaemonSet) string {
	return resource.Spec.Template.ObjectMeta.Labels[cluster.LabelVersion]
}

func image(resource *v1.DaemonSet) string {
	return resource.Spec.Template.Spec.Containers[0].Image
}
