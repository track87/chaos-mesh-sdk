// Package sdk
// marsdong 2022/4/21
package sdk

import (
	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)
// Experiment defines the information of an experiment.
type Experiment struct {
	Namespace string                `json:"namespace"`
	Name      string                `json:"name"`
	Kind      string                `json:"kind"`
	UID       string                `json:"uid"`
	Created   string                `json:"created_at"`
	Status    *v1alpha1.ChaosStatus `json:"status"`
	Events    []v1.Event            `json:"events"`
}
