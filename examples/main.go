package main

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	capigcl "github.com/pvsune/capi-global-client"
)

var scheme = runtime.NewScheme()

func init() {
	_ = capiv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
}

func main() {
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("unable to create manager: %s", err)
	}

	ctx := context.TODO()
	go mgr.Start(ctx)
	if !mgr.GetCache().WaitForCacheSync(ctx) {
		log.Fatalf("cannot sync cache")
	}

	// global client supports indexes for use with client.Options
	gcl := capigcl.GlobalClient{
		Manager: mgr,
		Indexes: []remote.Index{{
			Object: &corev1.Pod{},
			Field:  "metadata.name",
			ExtractValue: func(o client.Object) []string {
				return []string{o.GetName()}
			},
		}},
	}

	// it's impossible to get concrete type from client.ObjectList
	// global client will call AddItems() to add in response
	obj := capigcl.ClusterObjectList{
		ObjectList: &corev1.PodList{},
		AddItems: func(pods client.ObjectList) (items []client.Object) {
			for _, o := range pods.(*corev1.PodList).Items {
				items = append(items, o.DeepCopy())
			}
			return
		},
	}

	objects, err := gcl.List(ctx, obj, client.MatchingFields{"metadata.name": "nginx"})
	if err != nil {
		log.Fatalf("cannot list objects: %s", err)
	}

	log.Printf("found %d objects", len(objects))
	for _, o := range objects {
		log.Println(o.GetName(), o.GetCluster())
	}
}
