// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package resourcesmisc

import (
	"fmt"

	kcv1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apis/kappctrl/v1alpha1"
	ctlres "github.com/vmware-tanzu/carvel-kapp/pkg/kapp/resources"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	kcv1alpha1.AddToScheme(scheme.Scheme)
}

type KappctrlK14sIoV1alpha1App struct {
	resource ctlres.Resource
}

func NewKappctrlK14sIoV1alpha1App(resource ctlres.Resource) *KappctrlK14sIoV1alpha1App {
	matcher := ctlres.APIVersionKindMatcher{
		APIVersion: kcv1alpha1.SchemeGroupVersion.String(),
		Kind:       "App",
	}
	if matcher.Matches(resource) {
		return &KappctrlK14sIoV1alpha1App{resource}
	}
	return nil
}

func (s KappctrlK14sIoV1alpha1App) IsDoneApplying() DoneApplyState {
	app := kcv1alpha1.App{}

	err := s.resource.AsTypedObj(&app)
	if err != nil {
		return DoneApplyState{Done: true, Successful: false, Message: fmt.Sprintf(
			"Error: Failed obj conversion: %s", err)}
	}

	if app.Generation != app.Status.ObservedGeneration {
		return DoneApplyState{Done: false, Message: fmt.Sprintf(
			"Waiting for generation %d to be observed", app.Generation)}
	}

	for _, cond := range app.Status.Conditions {
		errorMsg := app.Status.UsefulErrorMessage
		if errorMsg == "" {
			errorMsg = cond.Message
		}
		switch {
		case cond.Type == kcv1alpha1.Reconciling && cond.Status == corev1.ConditionTrue:
			return DoneApplyState{Done: false, Message: "Reconciling"}

		case cond.Type == kcv1alpha1.ReconcileFailed && cond.Status == corev1.ConditionTrue:
			return DoneApplyState{Done: true, Successful: false, Message: fmt.Sprintf(
				"Reconcile failed: %s (message: %s)", cond.Reason, errorMsg)}

		case cond.Type == kcv1alpha1.DeleteFailed && cond.Status == corev1.ConditionTrue:
			return DoneApplyState{Done: true, Successful: false, Message: fmt.Sprintf(
				"Delete failed: %s (message: %s)", cond.Reason, errorMsg)}
		}
	}

	deletingRes := NewDeleting(s.resource)
	if deletingRes != nil {
		return deletingRes.IsDoneApplying()
	}

	return DoneApplyState{Done: true, Successful: true}
}
