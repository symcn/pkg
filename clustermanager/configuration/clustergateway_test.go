package configuration

import (
	"os"
	"testing"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func TestNewClusterCfgManagerWithGateway(t *testing.T) {
	home, _ := os.UserHomeDir()
	cfg, err := clientcmd.BuildConfigFromFlags("", home+"/.kube/config")
	if err != nil {
		t.Error(err)
		return
	}
	dyanamicInterface, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Error(err)
		return
	}

	// scheme := runtime.NewScheme()
	// clustetgatewayv1aplpha1.AddToScheme(scheme)
	// cg := &clustetgatewayv1aplpha1.ClusterGateway{}
	// // o, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&clustetgatewayv1aplpha1.ClusterGateway{
	// //     TypeMeta: metav1.TypeMeta{
	// //         APIVersion: cg.GetGroupVersionResource().Group + "/" + cg.GetGroupVersionResource().Version,
	// //         Kind:       "ClusterGateway",
	// //     },
	// //     ObjectMeta: metav1.ObjectMeta{
	// //         Name: "cluster1",
	// //     },
	// //     Spec: clustetgatewayv1aplpha1.ClusterGatewaySpec{
	// //         Access: clustetgatewayv1aplpha1.ClusterAccess{
	// //             Endpoint: &clustetgatewayv1aplpha1.ClusterEndpoint{
	// //                 Type: clustetgatewayv1aplpha1.ClusterEndpointType("ClusterProxy"),
	// //             },
	// //             Credential: &clustetgatewayv1aplpha1.ClusterAccessCredential{
	// //                 Type:                clustetgatewayv1aplpha1.CredentialTypeServiceAccountToken,
	// //                 ServiceAccountToken: "111111111111111111",
	// //             },
	// //         },
	// //     },
	// // })
	// // obj := &unstructured.Unstructured{}
	// // runtime.DefaultUnstructuredConverter.FromUnstructured(o, obj)
	// dyanamicInterface := fake.NewSimpleDynamicClientWithCustomListKinds(
	//     scheme,
	//     map[schema.GroupVersionResource]string{
	//         cg.GetGroupVersionResource(): "ClusterGatewayList",
	//     },
	//     // &unstructured.UnstructuredList{
	//     //     Object: map[string]interface{}{
	//     //         "apiVersion": cg.GetGroupVersionResource().Group + "/" + cg.GetGroupVersionResource().Version,
	//     //         "kind":       "ClusterGatewayList",
	//     //     },
	//     //     Items: []unstructured.Unstructured{*obj},
	//     // },
	//     &clustetgatewayv1aplpha1.ClusterGatewayList{
	//         TypeMeta: metav1.TypeMeta{
	//             APIVersion: cg.GetGroupVersionResource().Group + "/" + cg.GetGroupVersionResource().Version,
	//             Kind:       "ClusterGatewayList",
	//         },
	//         Items: []clustetgatewayv1aplpha1.ClusterGateway{
	//             {
	//                 TypeMeta: metav1.TypeMeta{
	//                     APIVersion: cg.GetGroupVersionResource().Group + "/" + cg.GetGroupVersionResource().Version,
	//                     Kind:       "ClusterGateway",
	//                 },
	//                 ObjectMeta: metav1.ObjectMeta{
	//                     Name: "cluster1",
	//                 },
	//                 Spec: clustetgatewayv1aplpha1.ClusterGatewaySpec{
	//                     Access: clustetgatewayv1aplpha1.ClusterAccess{
	//                         Endpoint: &clustetgatewayv1aplpha1.ClusterEndpoint{
	//                             Type: clustetgatewayv1aplpha1.ClusterEndpointType("ClusterProxy"),
	//                         },
	//                         Credential: &clustetgatewayv1aplpha1.ClusterAccessCredential{
	//                             Type:                clustetgatewayv1aplpha1.CredentialTypeServiceAccountToken,
	//                             ServiceAccountToken: "111111111111111111",
	//                         },
	//                     },
	//                 },
	//             },
	//         },
	//     },
	// )
	cfgManager := NewClusterCfgManagerWithGateway(dyanamicInterface, BuildDefaultClusterCfgInfo("demo"))
	list, err := cfgManager.GetAll()
	if err != nil {
		t.Error(err)
		return
	}

	for _, cluster := range list {
		t.Logf("found cluster: %s", cluster.GetName())
	}
}
