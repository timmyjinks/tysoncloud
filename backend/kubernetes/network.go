package kubernetes

import (
	"context"

	"github.com/timmyjinks/tysoncloud/util"
	corev1 "k8s.io/api/core/v1"

	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	appmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	v1 "k8s.io/client-go/applyconfigurations/networking/v1"
)

func (d *KubernetesService) CreateNetworkPolicy(ctx context.Context, namespace, clusterIP string) error {
	_, err := d.clientset.NetworkingV1().NetworkPolicies(namespace).Apply(context.TODO(), &v1.NetworkPolicyApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			Kind:       util.StringPtr("NetworkPolicy"),
			APIVersion: util.StringPtr("networking.k8s.io/v1"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Name:      &namespace,
			Namespace: &namespace,
		},
		Spec: &v1.NetworkPolicySpecApplyConfiguration{
			PolicyTypes: []netv1.PolicyType{
				"Ingress",
				"Egress",
			},
			Ingress: []v1.NetworkPolicyIngressRuleApplyConfiguration{
				{
					From: []v1.NetworkPolicyPeerApplyConfiguration{
						{
							PodSelector: &appmetav1.LabelSelectorApplyConfiguration{},
						},
						{
							NamespaceSelector: &appmetav1.LabelSelectorApplyConfiguration{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "tc-system",
								},
							},
						},
					},
				},
			},
			Egress: []v1.NetworkPolicyEgressRuleApplyConfiguration{
				{
					To: []v1.NetworkPolicyPeerApplyConfiguration{
						{

							PodSelector: &appmetav1.LabelSelectorApplyConfiguration{},
						},
					},
				},
				{
					To: []v1.NetworkPolicyPeerApplyConfiguration{
						{
							NamespaceSelector: &appmetav1.LabelSelectorApplyConfiguration{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "kube-system",
								},
							},
						},
					},
					Ports: []v1.NetworkPolicyPortApplyConfiguration{
						{
							Protocol: (*corev1.Protocol)(util.StringPtr("UDP")),
							Port:     &intstr.IntOrString{IntVal: 53},
						},
						{
							Protocol: (*corev1.Protocol)(util.StringPtr("TCP")),
							Port:     &intstr.IntOrString{IntVal: 53},
						},
					},
				},
				{
					Ports: []v1.NetworkPolicyPortApplyConfiguration{
						{
							Protocol: (*corev1.Protocol)(util.StringPtr("TCP")),
							Port:     &intstr.IntOrString{IntVal: 80},
						},
						{
							Protocol: (*corev1.Protocol)(util.StringPtr("TCP")),
							Port:     &intstr.IntOrString{IntVal: 443},
						},
					},
				},
			},
		},
	}, metav1.ApplyOptions{FieldManager: "tysoncloud"})
	if err != nil {
		return err
	}

	if _, err := d.clientset.NetworkingV1().NetworkPolicies(namespace).Apply(context.TODO(), &v1.NetworkPolicyApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			Kind:       util.StringPtr("NetworkPolicy"),
			APIVersion: util.StringPtr("networking.k8s.io/v1"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Name:      util.StringPtr(namespace + "-database"),
			Namespace: &namespace,
		},
		Spec: &v1.NetworkPolicySpecApplyConfiguration{
			PolicyTypes: []netv1.PolicyType{
				"Egress",
			},
			PodSelector: &appmetav1.LabelSelectorApplyConfiguration{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": "database",
				},
			},
			Egress: []v1.NetworkPolicyEgressRuleApplyConfiguration{
				{
					To: []v1.NetworkPolicyPeerApplyConfiguration{
						{
							IPBlock: &v1.IPBlockApplyConfiguration{
								CIDR: util.StringPtr(clusterIP + "/32"),
							},
						},
					},
					Ports: []v1.NetworkPolicyPortApplyConfiguration{
						{
							Protocol: (*corev1.Protocol)(util.StringPtr("TCP")),
							Port:     &intstr.IntOrString{IntVal: 6443},
						},
					},
				},
			},
		},
	}, metav1.ApplyOptions{FieldManager: "tysoncloud"}); err != nil {
		return err
	}
	return nil
}

func (d *KubernetesService) DeleteNetwork(ctx context.Context, resource Resource) error {
	return d.clientset.NetworkingV1().NetworkPolicies(resource.Namespace).Delete(ctx, resource.Name, metav1.DeleteOptions{})
}
