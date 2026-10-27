package kubernetes

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	gatewayfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"
)

const testNS = "proj-1"

func newTestService(t *testing.T, objects ...runtime.Object) (*KubernetesService, *fake.Clientset, *gatewayfake.Clientset, *dynamicfake.FakeDynamicClient) {
	t.Helper()
	cs := fake.NewClientset(objects...)
	gw := gatewayfake.NewClientset()
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	return NewKubernetesServiceWithClients("10.0.0.1", cs, gw, dyn), cs, gw, dyn
}

func TestCreateDeployment(t *testing.T) {
	ctx := context.Background()
	svc, cs, _, _ := newTestService(t)

	res := Resource{Namespace: testNS, Name: "web", Image: "nginx:1.27", Port: 8080}
	if err := svc.CreateDeployment(ctx, res); err != nil {
		t.Fatalf("CreateDeployment: %v", err)
	}

	d, err := cs.AppsV1().Deployments(testNS).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if d.Labels["app"] != "web" || d.Spec.Selector.MatchLabels["app"] != "web" || d.Spec.Template.Labels["app"] != "web" {
		t.Fatalf("app label/selector mismatch: %+v", d)
	}
	if d.Annotations["reloader.stakater.com/auto"] != "true" {
		t.Fatalf("missing reloader annotation: %v", d.Annotations)
	}
	if _, err := time.Parse(time.RFC3339, d.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]); err != nil {
		t.Fatalf("restartedAt annotation not RFC3339: %v", d.Spec.Template.Annotations)
	}

	c := d.Spec.Template.Spec.Containers[0]
	if c.Name != "web" || c.Image != "nginx:1.27" {
		t.Fatalf("container = %s/%s", c.Name, c.Image)
	}
	if c.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Fatalf("pull policy = %s, want IfNotPresent", c.ImagePullPolicy)
	}
	if len(c.Ports) != 1 || c.Ports[0].ContainerPort != 8080 || c.Ports[0].Protocol != corev1.ProtocolTCP {
		t.Fatalf("ports = %+v", c.Ports)
	}
	if c.Resources.Limits.Cpu().String() != "500m" || c.Resources.Limits.Memory().String() != "1Gi" {
		t.Fatalf("limits = %v", c.Resources.Limits)
	}
	if c.Resources.Requests.Cpu().String() != "100m" || c.Resources.Requests.Memory().String() != "100Mi" {
		t.Fatalf("requests = %v", c.Resources.Requests)
	}
	if len(c.EnvFrom) != 0 {
		t.Fatalf("EnvFrom set without env: %+v", c.EnvFrom)
	}
}

func TestCreateDeploymentLatestAndEnv(t *testing.T) {
	ctx := context.Background()
	svc, cs, _, _ := newTestService(t)

	res := Resource{Namespace: testNS, Name: "api", Image: "registry/api:latest", Port: 3000, Env: map[string][]byte{"A": []byte("1")}}
	if err := svc.CreateDeployment(ctx, res); err != nil {
		t.Fatalf("CreateDeployment: %v", err)
	}
	d, err := cs.AppsV1().Deployments(testNS).Get(ctx, "api", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	c := d.Spec.Template.Spec.Containers[0]
	if c.ImagePullPolicy != corev1.PullAlways {
		t.Fatalf("pull policy = %s, want Always for :latest", c.ImagePullPolicy)
	}
	if len(c.EnvFrom) != 1 || c.EnvFrom[0].SecretRef == nil || c.EnvFrom[0].SecretRef.Name != "api" {
		t.Fatalf("EnvFrom = %+v, want secretRef api", c.EnvFrom)
	}
}

func TestCreateService(t *testing.T) {
	ctx := context.Background()
	svc, cs, _, _ := newTestService(t)

	if err := svc.CreateService(ctx, Resource{Namespace: testNS, Name: "web", Port: 8080}); err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	s, err := cs.CoreV1().Services(testNS).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get service: %v", err)
	}
	if s.Spec.Selector["app"] != "web" {
		t.Fatalf("selector = %v", s.Spec.Selector)
	}
	if len(s.Spec.Ports) != 1 || s.Spec.Ports[0].Port != 8080 || s.Spec.Ports[0].TargetPort.IntVal != 8080 {
		t.Fatalf("ports = %+v", s.Spec.Ports)
	}

	if err := svc.DeleteService(ctx, Resource{Namespace: testNS, Name: "web"}); err != nil {
		t.Fatalf("DeleteService: %v", err)
	}
	if _, err := cs.CoreV1().Services(testNS).Get(ctx, "web", metav1.GetOptions{}); err == nil {
		t.Fatalf("service still exists after delete")
	}
}

func TestSecretRoundTrip(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _ := newTestService(t)
	res := Resource{Namespace: testNS, Name: "web", Env: map[string][]byte{
		"API_URL": []byte("https://x\n"),
		"PORT":    []byte("8080\r\n"),
	}}

	if err := svc.CreateSecret(ctx, res); err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	env, err := svc.GetSecret(ctx, res)
	if err != nil {
		t.Fatalf("GetSecret: %v", err)
	}
	if env["API_URL"] != "https://x" || env["PORT"] != "8080" || len(env) != 2 {
		t.Fatalf("env = %q, want trailing newlines trimmed", env)
	}

	if err := svc.DeleteSecret(ctx, res); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	env, err = svc.GetSecret(ctx, res)
	if err != nil {
		t.Fatalf("GetSecret after delete: %v", err)
	}
	if env == nil || len(env) != 0 {
		t.Fatalf("missing secret env = %v, want empty non-nil map", env)
	}
}

func TestHPA(t *testing.T) {
	ctx := context.Background()
	svc, cs, _, _ := newTestService(t)
	res := Resource{Namespace: testNS, Name: "web"}

	if err := svc.CreateHPA(ctx, res); err != nil {
		t.Fatalf("CreateHPA: %v", err)
	}
	h, err := cs.AutoscalingV2().HorizontalPodAutoscalers(testNS).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get hpa: %v", err)
	}
	if h.Spec.ScaleTargetRef.Kind != "Deployment" || h.Spec.ScaleTargetRef.Name != "web" {
		t.Fatalf("scale target = %+v", h.Spec.ScaleTargetRef)
	}
	if *h.Spec.MinReplicas != 1 || h.Spec.MaxReplicas != 10 {
		t.Fatalf("replicas = %d..%d, want 1..10", *h.Spec.MinReplicas, h.Spec.MaxReplicas)
	}
	if *h.Spec.Metrics[0].Resource.Target.AverageUtilization != 50 {
		t.Fatalf("cpu target = %d, want 50", *h.Spec.Metrics[0].Resource.Target.AverageUtilization)
	}

	if err := svc.DeleteHPA(ctx, res); err != nil {
		t.Fatalf("DeleteHPA: %v", err)
	}
	if err := svc.DeleteHPA(ctx, res); err != nil {
		t.Fatalf("DeleteHPA on missing hpa = %v, want nil", err)
	}
}

func TestNamespace(t *testing.T) {
	ctx := context.Background()
	svc, cs, _, _ := newTestService(t)

	if err := svc.CreateNamespace(ctx, testNS); err != nil {
		t.Fatalf("CreateNamespace: %v", err)
	}
	ns, err := cs.CoreV1().Namespaces().Get(ctx, testNS, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get namespace: %v", err)
	}
	if ns.Labels["managed-by"] != "tysoncloud" {
		t.Fatalf("labels = %v", ns.Labels)
	}

	if err := svc.DeleteNamespace(ctx, testNS); err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}
	if _, err := cs.CoreV1().Namespaces().Get(ctx, testNS, metav1.GetOptions{}); err == nil {
		t.Fatalf("namespace still exists after delete")
	}
}

func TestCreateNetworkPolicy(t *testing.T) {
	ctx := context.Background()
	svc, cs, _, _ := newTestService(t)

	if err := svc.CreateNetworkPolicy(ctx, testNS, "10.0.0.1"); err != nil {
		t.Fatalf("CreateNetworkPolicy: %v", err)
	}

	main, err := cs.NetworkingV1().NetworkPolicies(testNS).Get(ctx, testNS, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get main policy: %v", err)
	}
	if len(main.Spec.PolicyTypes) != 2 {
		t.Fatalf("policy types = %v, want Ingress+Egress", main.Spec.PolicyTypes)
	}
	if got := main.Spec.Ingress[0].From[1].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"]; got != "tc-system" {
		t.Fatalf("ingress namespace = %q, want tc-system", got)
	}

	// The public-internet egress rule (80/443) must be constrained to an
	// IPBlock that excludes link-local and RFC1918, so tenants can't reach
	// the metadata endpoint, the API server or other in-cluster Services.
	var publicEgress *netv1.NetworkPolicyEgressRule
	for i := range main.Spec.Egress {
		e := &main.Spec.Egress[i]
		for _, p := range e.Ports {
			if p.Port != nil && p.Port.IntVal == 443 {
				publicEgress = e
			}
		}
	}
	if publicEgress == nil {
		t.Fatal("no egress rule allowing 443")
	}
	if len(publicEgress.To) != 1 || publicEgress.To[0].IPBlock == nil {
		t.Fatalf("public egress is not restricted to an IPBlock: %+v", publicEgress.To)
	}
	if publicEgress.To[0].IPBlock.CIDR != "0.0.0.0/0" {
		t.Fatalf("public egress CIDR = %q, want 0.0.0.0/0", publicEgress.To[0].IPBlock.CIDR)
	}
	for _, want := range []string{"169.254.0.0/16", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		if !slices.Contains(publicEgress.To[0].IPBlock.Except, want) {
			t.Fatalf("public egress except %v missing %s", publicEgress.To[0].IPBlock.Except, want)
		}
	}

	db, err := cs.NetworkingV1().NetworkPolicies(testNS).Get(ctx, testNS+"-database", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get database policy: %v", err)
	}
	if db.Spec.PodSelector.MatchLabels["app.kubernetes.io/component"] != "database" {
		t.Fatalf("db pod selector = %v", db.Spec.PodSelector.MatchLabels)
	}
	egress := db.Spec.Egress[0]
	if egress.To[0].IPBlock.CIDR != "10.0.0.1/32" || egress.Ports[0].Port.IntVal != 6443 {
		t.Fatalf("db egress = %+v", egress)
	}
}

func TestCreateHTTPRoute(t *testing.T) {
	ctx := context.Background()
	svc, _, gw, _ := newTestService(t)

	res := Resource{Namespace: testNS, Name: "web", Hostname: "web.example.com", Port: 8080}
	if err := svc.CreateHTTPRoute(ctx, res); err != nil {
		t.Fatalf("CreateHTTPRoute: %v", err)
	}
	route, err := gw.GatewayV1().HTTPRoutes(testNS).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get route: %v", err)
	}
	if len(route.Spec.Hostnames) != 1 || route.Spec.Hostnames[0] != "web.example.com" {
		t.Fatalf("hostnames = %v", route.Spec.Hostnames)
	}
	parent := route.Spec.ParentRefs[0]
	if parent.Name != "tysoncloud-gateway" || *parent.Namespace != "tc-system" {
		t.Fatalf("parent = %+v", parent)
	}
	backend := route.Spec.Rules[0].BackendRefs[0]
	if backend.Name != "web" || *backend.Port != 8080 {
		t.Fatalf("backend = %s:%d", backend.Name, *backend.Port)
	}

	if err := svc.DeleteHTTPRoute(ctx, res); err != nil {
		t.Fatalf("DeleteHTTPRoute: %v", err)
	}
}

func TestPVCAttachDetach(t *testing.T) {
	ctx := context.Background()
	existing := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: testNS},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "nginx"}}},
			},
		},
	}
	svc, cs, _, _ := newTestService(t, existing)
	res := Resource{Namespace: testNS, Name: "web", StorageGB: 3, MountPath: "/data"}

	if err := svc.CreatePVC(ctx, res); err != nil {
		t.Fatalf("CreatePVC: %v", err)
	}
	pvc, err := cs.CoreV1().PersistentVolumeClaims(testNS).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get pvc: %v", err)
	}
	if got := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; got.String() != "3Gi" {
		t.Fatalf("storage = %s, want 3Gi", got.String())
	}

	if err := svc.AttachPVCToDeployment(ctx, res); err != nil {
		t.Fatalf("AttachPVCToDeployment: %v", err)
	}
	d, _ := cs.AppsV1().Deployments(testNS).Get(ctx, "web", metav1.GetOptions{})
	vols := d.Spec.Template.Spec.Volumes
	mounts := d.Spec.Template.Spec.Containers[0].VolumeMounts
	if len(vols) != 1 || vols[0].Name != "vol-web" || vols[0].PersistentVolumeClaim.ClaimName != "web" {
		t.Fatalf("volumes = %+v", vols)
	}
	if len(mounts) != 1 || mounts[0].Name != "vol-web" || mounts[0].MountPath != "/data" {
		t.Fatalf("mounts = %+v", mounts)
	}

	if err := svc.DetachPVCToDeployment(ctx, res); err != nil {
		t.Fatalf("DetachPVCToDeployment: %v", err)
	}
	d, _ = cs.AppsV1().Deployments(testNS).Get(ctx, "web", metav1.GetOptions{})
	if len(d.Spec.Template.Spec.Volumes) != 0 || len(d.Spec.Template.Spec.Containers[0].VolumeMounts) != 0 {
		t.Fatalf("volumes not detached: %+v", d.Spec.Template.Spec)
	}

	if err := svc.DeletePVC(ctx, res); err != nil {
		t.Fatalf("DeletePVC: %v", err)
	}
}

func TestAttachPVCMissingDeployment(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	if err := svc.AttachPVCToDeployment(context.Background(), Resource{Namespace: testNS, Name: "nope"}); err == nil {
		t.Fatalf("expected error attaching to missing deployment")
	}
}

func TestCreatePostgresDatabase(t *testing.T) {
	tests := []struct {
		name      string
		storageGB int32
		wantSize  string
	}{
		{"default storage", 0, "5Gi"},
		{"negative storage", -1, "5Gi"},
		{"explicit storage", 20, "20Gi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, dyn := newTestService(t)

			var applied map[string]any
			dyn.PrependReactor("patch", "clusters", func(action k8stesting.Action) (bool, runtime.Object, error) {
				patch := action.(k8stesting.PatchAction)
				if err := json.Unmarshal(patch.GetPatch(), &applied); err != nil {
					t.Fatalf("decode patch: %v", err)
				}
				return true, &unstructured.Unstructured{Object: applied}, nil
			})

			if err := svc.CreatePostgresDatabase(context.Background(), Resource{Namespace: testNS, Name: "db", StorageGB: tt.storageGB}); err != nil {
				t.Fatalf("CreatePostgresDatabase: %v", err)
			}

			u := unstructured.Unstructured{Object: applied}
			if u.GetKind() != "Cluster" || u.GetAPIVersion() != "postgresql.cnpg.io/v1" || u.GetName() != "db" {
				t.Fatalf("applied object = %s %s %s", u.GetAPIVersion(), u.GetKind(), u.GetName())
			}
			size, _, _ := unstructured.NestedString(applied, "spec", "storage", "size")
			if size != tt.wantSize {
				t.Fatalf("storage size = %q, want %q", size, tt.wantSize)
			}
			component, _, _ := unstructured.NestedString(applied, "spec", "inheritedMetadata", "labels", "app.kubernetes.io/component")
			if component != "database" {
				t.Fatalf("component label = %q, want database (network policy relies on it)", component)
			}
			db, _, _ := unstructured.NestedString(applied, "spec", "bootstrap", "initdb", "database")
			if db != "app" {
				t.Fatalf("initdb database = %q, want app", db)
			}
		})
	}
}

func TestGetDeploymentLogsNoPods(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	err := svc.GetDeploymentLogs(context.Background(), Resource{Namespace: testNS, Name: "web"}, make(chan string, 1))
	if err == nil || !strings.Contains(err.Error(), "no pods found") {
		t.Fatalf("err = %v, want no pods found", err)
	}
}

func TestGetDeploymentDiagnosticLogs(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-abc", Namespace: testNS, Labels: map[string]string{"app": "web"}},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "web",
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff", Message: "back-off"}},
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1, Message: "boom"},
					},
				},
				{
					Name:  "sidecar",
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "Ignored"}},
				},
			},
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "migrate",
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 2}},
				},
			},
		},
	}
	svc, _, _, _ := newTestService(t, pod)

	ctx, cancel := context.WithCancel(context.Background())
	lines := make(chan string, 10)
	done := make(chan error, 1)
	go func() { done <- svc.GetDeploymentDiagnosticLogs(ctx, Resource{Namespace: testNS, Name: "web"}, lines) }()

	var got []string
	for range 3 {
		select {
		case l := <-lines:
			got = append(got, l)
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for diagnostics, got %q", got)
		}
	}
	cancel()
	if err := <-done; err != context.Canceled {
		t.Fatalf("err = %v, want context.Canceled", err)
	}

	want := []string{
		"container waiting: CrashLoopBackOff — back-off",
		"last termination: Error (exit 1) — boom",
		"init container migrate terminated: Error (exit 2) — ",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
	select {
	case extra := <-lines:
		t.Fatalf("unexpected extra line %q (sidecar should be skipped)", extra)
	default:
	}
}

func TestGetDeploymentDiagnosticLogsNoPods(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	lines := make(chan string, 1)
	if err := svc.GetDeploymentDiagnosticLogs(context.Background(), Resource{Namespace: testNS, Name: "web"}, lines); err == nil {
		t.Fatalf("expected error with no pods")
	}
	if l := <-lines; !strings.HasPrefix(l, "no pods found for web") {
		t.Fatalf("line = %q", l)
	}
}
