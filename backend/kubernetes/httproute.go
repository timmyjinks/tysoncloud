package kubernetes

import (
	"context"

	"github.com/timmyjinks/tysoncloud/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	v1 "sigs.k8s.io/gateway-api/applyconfiguration/apis/v1"
)

func (svc *KubernetesService) CreateHTTPRoute(ctx context.Context, resource Resource) error {
	httpRoute := &v1.HTTPRouteApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			Kind:       util.StringPtr("HTTPRoute"),
			APIVersion: util.StringPtr("gateway.networking.k8s.io/v1"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Namespace: util.StringPtr(resource.Namespace),
			Name:      util.StringPtr(resource.Name),
		},
		Spec: &v1.HTTPRouteSpecApplyConfiguration{
			CommonRouteSpecApplyConfiguration: v1.CommonRouteSpecApplyConfiguration{
				ParentRefs: []v1.ParentReferenceApplyConfiguration{
					{
						Name:      (*gatewayv1.ObjectName)(util.StringPtr("tysoncloud-gateway")),
						Namespace: (*gatewayv1.Namespace)(util.StringPtr("tc-system")),
					},
				},
			},
			Hostnames: []gatewayv1.Hostname{
				gatewayv1.Hostname(resource.Hostname),
			},
			Rules: []v1.HTTPRouteRuleApplyConfiguration{
				{
					Matches: []v1.HTTPRouteMatchApplyConfiguration{
						{
							Path: &v1.HTTPPathMatchApplyConfiguration{
								Type:  (*gatewayv1.PathMatchType)(util.StringPtr("PathPrefix")),
								Value: util.StringPtr("/"),
							},
						},
					},
					BackendRefs: []v1.HTTPBackendRefApplyConfiguration{
						{
							BackendRefApplyConfiguration: v1.BackendRefApplyConfiguration{
								BackendObjectReferenceApplyConfiguration: v1.BackendObjectReferenceApplyConfiguration{
									Name: (*gatewayv1.ObjectName)(util.StringPtr(resource.Name)),
									Port: util.IntPtr(resource.Port),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := svc.gatewayClient.GatewayV1().HTTPRoutes(resource.Namespace).Apply(ctx, httpRoute, metav1.ApplyOptions{
		FieldManager: "tysoncloud",
	})
	if err != nil {
		return err
	}
	return nil
}

func (d *KubernetesService) DeleteHTTPRoute(ctx context.Context, resource Resource) error {
	return d.gatewayClient.GatewayV1().HTTPRoutes(resource.Namespace).Delete(ctx, resource.Name, metav1.DeleteOptions{})
}
