package client

import (
	"context"
	"log"

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

type ClusterObjectList struct {
	client.ObjectList
	AddItems func(client.ObjectList) []client.Object
}

// Implements ClusterObject
type Object struct {
	client.Object
	Cluster client.ObjectKey
}

func (o Object) GetCluster() client.ObjectKey {
	return o.Cluster
}

func (gcl *GlobalClient) List(ctx context.Context, obj ClusterObjectList, opts ...client.ListOption) ([]ClusterObject, error) {
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
		oo := obj.ObjectList
		if err := cl.List(ctx, oo, opts...); err != nil {
			errors = append(errors, err)
			continue
		}
		for _, o := range obj.AddItems(oo) {
			objects = append(objects, Object{Object: o, Cluster: nsn})
		}
	}

	if len(errors) > 0 {
		log.Printf("ignoring %d found errors", len(errors))
	}
	return objects, nil
}
