/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	config "github.com/openshift/api/config/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/openshift/windows-machine-config-operator/pkg/cluster"
	"github.com/openshift/windows-machine-config-operator/pkg/registries"
)

//+kubebuilder:rbac:groups="config.openshift.io",resources=imagedigestmirrorsets,verbs=get;list;watch
//+kubebuilder:rbac:groups="config.openshift.io",resources=imagetagmirrorsets,verbs=get;list;watch

const (
	// RegistryController is the name of this controller in logs and other outputs.
	RegistryController = "registry"
)

// registryReconciler holds the info required to reconcile image registry settings on Windows nodes
type registryReconciler struct {
	instanceReconciler
}

// NewRegistryReconciler returns a pointer to a new registryReconciler
func NewRegistryReconciler(mgr manager.Manager, clusterConfig cluster.Config,
	watchNamespace string) (*registryReconciler, error) {
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("error creating kubernetes clientset: %w", err)
	}

	return &registryReconciler{
		instanceReconciler: instanceReconciler{
			client:             mgr.GetClient(),
			log:                ctrl.Log.WithName("controllers").WithName(RegistryController),
			k8sclientset:       clientset,
			clusterServiceCIDR: clusterConfig.Network().GetServiceCIDR(),
			watchNamespace:     watchNamespace,
			recorder:           mgr.GetEventRecorderFor(RegistryController),
		},
	}, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which reads that state of the cluster for objects
// related to image registry config and aims to move the current state of the cluster closer to the desired state.
func (r *registryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	r.log = r.log.WithValues(RegistryController, req.NamespacedName)

	// List all IDMS/ITMS resources
	imageDigestMirrorSetList := &config.ImageDigestMirrorSetList{}
	if err = r.client.List(ctx, imageDigestMirrorSetList); err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting IDMS list: %w", err)
	}
	imageTagMirrorSetList := &config.ImageTagMirrorSetList{}
	if err = r.client.List(ctx, imageTagMirrorSetList); err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting ITMS list: %w", err)
	}

	_ = registries.NewRegistryConfig(imageDigestMirrorSetList.Items, imageTagMirrorSetList.Items)
	// TODO: transfer generated config files to Windows nodes as part of WINC-1222

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *registryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&config.ImageDigestMirrorSet{}).
		Watches(&config.ImageTagMirrorSet{}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
