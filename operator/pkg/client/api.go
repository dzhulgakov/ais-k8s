// Package client contains wrapper for k8s client
/*
 * Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
 */

package client

import (
	"context"
	"time"

	apiv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	aisv1 "github.com/ais-operator/api/v1alpha1"
)

type (
	K8SClient struct {
		client.Client
		scheme *runtime.Scheme
	}
)

func NewClientFromMgr(mgr manager.Manager) *K8SClient {
	return &K8SClient{
		mgr.GetClient(),
		mgr.GetScheme(),
	}
}

//////////////////////////////////////////
//             Get resources            //
/////////////////////////////////////////

func (c *K8SClient) GetAIStoreCR(ctx context.Context, name types.NamespacedName) (*aisv1.AIStore, error) {
	aistore := &aisv1.AIStore{}
	err := c.Get(ctx, name, aistore)
	return aistore, err
}

func (c *K8SClient) GetStatefulSet(ctx context.Context, name types.NamespacedName) (*apiv1.StatefulSet, error) {
	ss := &apiv1.StatefulSet{}
	err := c.Get(ctx, name, ss)
	return ss, err
}

func (c *K8SClient) GetServiceByName(ctx context.Context, name types.NamespacedName) (*corev1.Service, error) {
	svc := &corev1.Service{}
	err := c.Get(ctx, name, svc)
	return svc, err
}

func (c *K8SClient) GetCMByName(ctx context.Context, name types.NamespacedName) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := c.Get(ctx, name, cm)
	return cm, err
}

func (c *K8SClient) GetPodByName(ctx context.Context, name types.NamespacedName) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := c.Get(ctx, name, pod)
	return pod, err
}

func (c *K8SClient) GetAIStoreConfCR(ctx context.Context, name types.NamespacedName) (*aisv1.AISConfig, error) {
	cr := &aisv1.AISConfig{}
	err := c.Get(ctx, name, cr)
	return cr, err
}

////////////////////////////////////////
//      create/update resources      //
//////////////////////////////////////

func (c *K8SClient) UpdateStatefulSetReplicas(ctx context.Context, name types.NamespacedName, size int32) (updated bool, err error) {
	ss, err := c.GetStatefulSet(ctx, name)
	if err != nil {
		return
	}
	updated = *ss.Spec.Replicas != size
	if !updated {
		return
	}
	ss.Spec.Replicas = &size
	err = c.Update(ctx, ss)
	return
}

func (c *K8SClient) CreateResourceIfNotExists(ctx context.Context, owner *aisv1.AIStore, res client.Object) (exists bool, err error) {
	if owner != nil {
		if err = controllerutil.SetControllerReference(owner, res, c.scheme); err != nil {
			return
		}
		res.SetNamespace(owner.Namespace)
	}

	err = c.Create(ctx, res)
	exists = err != nil && apierrors.IsAlreadyExists(err)
	if exists {
		err = nil
	}
	return
}

func (c *K8SClient) UpdateIfExists(ctx context.Context, res client.Object) error {
	err := c.Update(ctx, res)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

/////////////////////////////////
//       Delete resources      //
////////////////////////////////

// DeleteResourceIfExists deletes an existing resource. It doesn't fail if the resource does not exist
func (c *K8SClient) DeleteResourceIfExists(context context.Context, obj client.Object) error {
	err := c.Delete(context, obj)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (c *K8SClient) DeleteServiceIfExists(ctx context.Context, name types.NamespacedName) error {
	svc := &corev1.Service{}
	svc.SetName(name.Name)
	svc.SetNamespace(name.Namespace)
	return c.DeleteResourceIfExists(ctx, svc)
}

func (c *K8SClient) DeleteAllServicesIfExists(ctx context.Context, namespace string, labels client.MatchingLabels) (err error) {
	err = c.DeleteAllOf(ctx, &corev1.Service{}, client.InNamespace(namespace), labels)
	if err != nil && apierrors.IsNotFound(err) {
		err = nil
	}
	return
}

func (c *K8SClient) DeleteStatefulSetIfExists(ctx context.Context, name types.NamespacedName) error {
	ss := &apiv1.StatefulSet{}
	ss.SetName(name.Name)
	ss.SetNamespace(name.Namespace)
	return c.DeleteResourceIfExists(ctx, ss)
}

func (c *K8SClient) DeleteConfigMapIfExists(ctx context.Context, name types.NamespacedName) error {
	ss := &corev1.ConfigMap{}
	ss.SetName(name.Name)
	ss.SetNamespace(name.Namespace)
	return c.DeleteResourceIfExists(ctx, ss)
}

func (c *K8SClient) DeletePodIfExists(ctx context.Context, name types.NamespacedName) error {
	pod := &corev1.Pod{}
	pod.SetName(name.Name)
	pod.SetNamespace(name.Namespace)
	return c.DeleteResourceIfExists(ctx, pod)
}

func (c *K8SClient) WaitForPodReady(ctx context.Context, name types.NamespacedName, timeout time.Duration) error {
	var (
		retryInterval   = 3 * time.Second
		ctxBack, cancel = context.WithTimeout(ctx, timeout)
		pod             *corev1.Pod
		err             error
	)
	defer cancel()
	for {
		pod, err = c.GetPodByName(ctx, name)
		if err != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodRunning {
			return nil
		}
		time.Sleep(retryInterval)
		select {
		case <-ctxBack.Done():
			return ctxBack.Err()
		default:
			break
		}
	}
}