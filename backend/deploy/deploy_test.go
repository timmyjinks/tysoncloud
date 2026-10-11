package deploy

import (
	"context"
	"errors"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	gatewayfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"

	"github.com/timmyjinks/tysoncloud/kubernetes"
)

const testNS = "proj-1"

func newTestDeploy(t *testing.T, objects ...runtime.Object) (*DeployService, *fake.Clientset, *gatewayfake.Clientset) {
	t.Helper()
	cs := fake.NewClientset(objects...)
	gw := gatewayfake.NewClientset()
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	return NewDeployService(kubernetes.NewKubernetesServiceWithClients("10.0.0.1", cs, gw, dyn)), cs, gw
}

func TestServiceToResource(t *testing.T) {
	s := Service{Namespace: testNS, Name: "web", Hostname: "web.example.com", Env: map[string][]byte{"A": []byte("1")}, Image: "nginx", Port: 80}
	r := ServiceToResource(s)
	if r.Namespace != s.Namespace || r.Name != s.Name || r.Hostname != s.Hostname || r.Image != s.Image || r.Port != s.Port || string(r.Env["A"]) != "1" {
		t.Fatalf("resource = %+v, want fields copied from %+v", r, s)
	}
}

func TestCreateServiceCreatesAllResources(t *testing.T) {
	ctx := context.Background()
	d, cs, gw := newTestDeploy(t)
	s := Service{Namespace: testNS, Name: "web", Hostname: "web.example.com", Image: "nginx", Port: 80, Env: map[string][]byte{"A": []byte("1")}}

	if err := d.CreateService(ctx, s); err != nil {
		t.Fatalf("CreateService: %v", err)
	}

	if _, err := cs.CoreV1().Services(testNS).Get(ctx, "web", metav1.GetOptions{}); err != nil {
		t.Errorf("k8s service missing: %v", err)
	}
	if _, err := cs.CoreV1().Secrets(testNS).Get(ctx, "web", metav1.GetOptions{}); err != nil {
		t.Errorf("secret missing: %v", err)
	}
	if _, err := cs.AppsV1().Deployments(testNS).Get(ctx, "web", metav1.GetOptions{}); err != nil {
		t.Errorf("deployment missing: %v", err)
	}
	if _, err := cs.AutoscalingV2().HorizontalPodAutoscalers(testNS).Get(ctx, "web", metav1.GetOptions{}); err != nil {
		t.Errorf("hpa missing: %v", err)
	}
	if _, err := gw.GatewayV1().HTTPRoutes(testNS).Get(ctx, "web", metav1.GetOptions{}); err != nil {
		t.Errorf("httproute missing: %v", err)
	}

	env, err := d.GetServiceEnv(ctx, s)
	if err != nil || env["A"] != "1" {
		t.Fatalf("GetServiceEnv = %v, %v", env, err)
	}
}

func TestCreateServiceWithoutEnvRemovesSecret(t *testing.T) {
	ctx := context.Background()
	stale := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: testNS}, Data: map[string][]byte{"OLD": []byte("x")}}
	d, cs, _ := newTestDeploy(t, stale)

	if err := d.CreateService(ctx, Service{Namespace: testNS, Name: "web", Image: "nginx", Port: 80}); err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	if _, err := cs.CoreV1().Secrets(testNS).Get(ctx, "web", metav1.GetOptions{}); err == nil {
		t.Fatalf("stale secret not removed when env is empty")
	}

	// No secret to delete the second time around must not fail.
	if err := d.CreateService(ctx, Service{Namespace: testNS, Name: "web", Image: "nginx", Port: 80}); err != nil {
		t.Fatalf("CreateService with no existing secret: %v", err)
	}
}

func TestCreateServiceStopsOnError(t *testing.T) {
	ctx := context.Background()
	d, cs, _ := newTestDeploy(t)
	cs.PrependReactor("patch", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("deployment apply failed")
	})

	err := d.CreateService(ctx, Service{Namespace: testNS, Name: "web", Image: "nginx", Port: 80})
	if err == nil || err.Error() != "deployment apply failed" {
		t.Fatalf("err = %v, want deployment apply failed", err)
	}
	if _, err := cs.AutoscalingV2().HorizontalPodAutoscalers(testNS).Get(ctx, "web", metav1.GetOptions{}); err == nil {
		t.Fatalf("hpa created after deployment failure")
	}
}

func TestDeleteServiceIsIdempotent(t *testing.T) {
	ctx := context.Background()
	d, cs, gw := newTestDeploy(t)
	s := Service{Namespace: testNS, Name: "web", Hostname: "web.example.com", Image: "nginx", Port: 80, Env: map[string][]byte{"A": []byte("1")}}

	if err := d.CreateService(ctx, s); err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	if err := d.DeleteService(ctx, s); err != nil {
		t.Fatalf("DeleteService: %v", err)
	}
	if _, err := cs.AppsV1().Deployments(testNS).Get(ctx, "web", metav1.GetOptions{}); err == nil {
		t.Errorf("deployment still exists")
	}
	if _, err := cs.CoreV1().Services(testNS).Get(ctx, "web", metav1.GetOptions{}); err == nil {
		t.Errorf("k8s service still exists")
	}
	if _, err := gw.GatewayV1().HTTPRoutes(testNS).Get(ctx, "web", metav1.GetOptions{}); err == nil {
		t.Errorf("httproute still exists")
	}

	if err := d.DeleteService(ctx, s); err != nil {
		t.Fatalf("second DeleteService = %v, want nil when everything is already gone", err)
	}
}

func TestDeleteServicePropagatesRealErrors(t *testing.T) {
	d, cs, _ := newTestDeploy(t)
	cs.PrependReactor("delete", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("api down")
	})
	if err := d.DeleteService(context.Background(), Service{Namespace: testNS, Name: "web"}); err == nil {
		t.Fatalf("expected non-NotFound error to propagate")
	}
}

func TestCreateProject(t *testing.T) {
	ctx := context.Background()
	d, cs, _ := newTestDeploy(t)

	if err := d.CreateProject(ctx, testNS); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if _, err := cs.CoreV1().Namespaces().Get(ctx, testNS, metav1.GetOptions{}); err != nil {
		t.Fatalf("namespace missing: %v", err)
	}
	policies, _ := cs.NetworkingV1().NetworkPolicies(testNS).List(ctx, metav1.ListOptions{})
	if len(policies.Items) != 2 {
		t.Fatalf("network policies = %d, want 2", len(policies.Items))
	}

	if err := d.DeleteProject(ctx, testNS); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := cs.CoreV1().Namespaces().Get(ctx, testNS, metav1.GetOptions{}); err == nil {
		t.Fatalf("namespace still exists after DeleteProject")
	}
}

func TestCreateProjectRollsBackNamespace(t *testing.T) {
	ctx := context.Background()
	d, cs, _ := newTestDeploy(t)
	cs.PrependReactor("patch", "networkpolicies", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("policy failed")
	})

	if err := d.CreateProject(ctx, testNS); err == nil {
		t.Fatalf("expected error when network policy fails")
	}
	if _, err := cs.CoreV1().Namespaces().Get(ctx, testNS, metav1.GetOptions{}); err == nil {
		t.Fatalf("namespace not cleaned up after network policy failure")
	}
}

func TestDatabaseUnsupportedEngine(t *testing.T) {
	d, _, _ := newTestDeploy(t)
	db := Database{Namespace: testNS, Name: "db", Engine: "mysql"}
	if err := d.CreateDatabase(context.Background(), db); err == nil {
		t.Fatalf("CreateDatabase accepted unsupported engine")
	}
	if err := d.DeleteDatabase(context.Background(), db); err == nil {
		t.Fatalf("DeleteDatabase accepted unsupported engine")
	}
}

func TestVolumeAttachDetach(t *testing.T) {
	ctx := context.Background()
	existing := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: testNS},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "web"}}},
			},
		},
	}
	d, cs, _ := newTestDeploy(t, existing)
	s := Service{Namespace: testNS, Name: "web"}

	if err := d.AttachVolume(ctx, s, Volume{MountPath: "/data", StorageGB: 2}); err != nil {
		t.Fatalf("AttachVolume: %v", err)
	}
	if _, err := cs.CoreV1().PersistentVolumeClaims(testNS).Get(ctx, "web", metav1.GetOptions{}); err != nil {
		t.Fatalf("pvc missing: %v", err)
	}
	dep, _ := cs.AppsV1().Deployments(testNS).Get(ctx, "web", metav1.GetOptions{})
	if mounts := dep.Spec.Template.Spec.Containers[0].VolumeMounts; len(mounts) != 1 || mounts[0].MountPath != "/data" {
		t.Fatalf("mounts = %+v", mounts)
	}

	if err := d.DetachVolume(ctx, s); err != nil {
		t.Fatalf("DetachVolume: %v", err)
	}
	if _, err := cs.CoreV1().PersistentVolumeClaims(testNS).Get(ctx, "web", metav1.GetOptions{}); err == nil {
		t.Fatalf("pvc still exists after detach")
	}
}
