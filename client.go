package client

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	capiv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type GlobalClient struct {
	manager.Manager
	Indexes []remote.Index
}

type ClusterObject interface {
	client.Object
	GetCluster() client.ObjectKey
}

type clusterObject struct {
	client.Object
	cluster client.ObjectKey
}

func (co clusterObject) GetCluster() client.ObjectKey {
	return co.cluster
}

func (gcl *GlobalClient) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) ([]ClusterObject, error) {
	clusters := &capiv1.ClusterList{}
	if err := gcl.GetClient().List(ctx, clusters); err != nil {
		return nil, err
	}

	t, err := remote.NewClusterCacheTracker(gcl.Manager, remote.ClusterCacheTrackerOptions{Indexes: gcl.Indexes})
	if err != nil {
		return nil, err
	}

	var (
		errors  = []error{}
		objects = []ClusterObject{}
	)
	for _, cluster := range clusters.Items {
		log.Printf("listing from cluster: %s", cluster.GetName())
		nsn := client.ObjectKeyFromObject(&cluster)
		cl, err := t.GetClient(ctx, nsn)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if err := cl.List(ctx, obj, opts...); err != nil {
			errors = append(errors, err)
			continue
		}
		// we really need to get concrete type here
		switch object := obj.(type) {
		case *corev1.PodList:
			for _, o := range object.Items {
				objects = append(objects, clusterObject{Object: o.DeepCopy(), cluster: nsn})
			}
		case *corev1.ServiceList:
			for _, o := range object.Items {
				objects = append(objects, clusterObject{Object: o.DeepCopy(), cluster: nsn})
			}
		}
	}

	// caller shouldn't use this, use returned value instead
	obj = nil

	if len(errors) > 0 {
		log.Printf("ignoring %d found errors", len(errors))
	}
	return objects, nil
}
