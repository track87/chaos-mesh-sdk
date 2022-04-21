// Package sdk
// marsdong 2022/4/21
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NotFoundErr = fmt.Errorf("resource not found")
	scheme      = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
}

// Client define APi sets fro chaos mesh
type Client interface {
	// CreateExperiment
	// @Description 创建特定类型的实验
	// @Author marsdong 2022-04-20 18:52:55
	// @param ctx
	// @param kind 实验类型
	// @param chaos 实验结构体，根据实验类型
	// @return *Experiment
	// @return error
	CreateExperiment(ctx context.Context, kind string, chaos interface{}) (*Experiment, error)
	// DeleteExperiment
	// @Description 删除实验， 该操作会导致实验终止
	// @Author marsdong 2022-04-20 18:54:17
	// @param ctx
	// @param namespace 实验所在的命名空间
	// @param name 实验名称
	// @param kind 实验类型
	// @return error 如果不存在返回NotFoundErr
	DeleteExperiment(ctx context.Context, namespace, name, kind string) error
	// DescribeExperimentWithEvents
	// @Description  查询实验以及关联的events
	// @Author marsdong 2022-04-20 18:55:02
	// @param ctx
	// @param namespace 实验所在的命名空间
	// @param name 实验名称
	// @param kind 实验类型
	// @return *Experiment 实验信息，包含events
	// @return error 如果不存在返回NotFoundErr
	DescribeExperimentWithEvents(ctx context.Context, namespace, name, kind string) (*Experiment, error)
	// DescribeExperiment
	// @Description  查询实验
	// @Author marsdong 2022-04-20 18:55:02
	// @param ctx
	// @param namespace 实验所在的命名空间
	// @param name 实验名称
	// @param kind 实验类型
	// @return *Experiment 实验信息
	// @return error 如果不存在返回NotFoundErr
	DescribeExperiment(ctx context.Context, namespace, name, kind string) (*Experiment, error)
	// ListExperiments
	// @Description 查询某种类型实验的集合
	// @Author marsdong 2022-04-20 18:58:38
	// @param ctx
	// @param kind 实验类型
	// @return []*Experiment
	// @return error 如果数据未空，不返回错误
	ListExperiments(ctx context.Context, kind string) ([]*Experiment, error)
	// ListEvents
	// @Description 查询实验事件集合
	// @Author marsdong 2022-04-20 18:59:46
	// @param ctx
	// @param experimentUid 实验的唯一ID
	// @param eventType 事件类型
	// @param reason
	// @return []v1.Event
	// @return error
	ListEvents(ctx context.Context, experimentUid, eventType, reason string) ([]v1.Event, error)
}

type client struct {
	kubeCli pkgclient.Client
}

// CreateExperiment 创建实验
func (c *client) CreateExperiment(ctx context.Context, kind string, chaos interface{}) (*Experiment, error) {
	chaosKind, ok := v1alpha1.AllKinds()[kind]
	if !ok {
		return nil, fmt.Errorf("not support chaos kind '%s'", kind)
	}

	object := chaosKind.SpawnObject()
	reflect.ValueOf(object).Elem().FieldByName("ObjectMeta").Set(reflect.ValueOf(metav1.ObjectMeta{}))

	bytes, err := json.Marshal(chaos)
	if err != nil {
		return nil, fmt.Errorf("failed marshal chaos, %s", err.Error())
	}
	if err = json.Unmarshal(bytes, object); err != nil {
		return nil, fmt.Errorf("failed unmarshal chaos, %s", err.Error())
	}

	if err = c.kubeCli.Create(ctx, object); err != nil {
		return nil, fmt.Errorf("failed create chaos, %s", err.Error())
	}
	return c.DescribeExperiment(ctx, object.GetNamespace(), object.GetName(), kind)
}

// DeleteExperiment 删除实验
func (c *client) DeleteExperiment(ctx context.Context, namespace, name, kind string) error {
	chaosKind, exists := v1alpha1.AllKinds()[kind]
	if !exists {
		return fmt.Errorf("unknwon chaos kind '%s'", kind)
	}

	chaos := chaosKind.SpawnObject()
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}
	if err := c.kubeCli.Get(ctx, namespacedName, chaos); err != nil {
		if isNotFound(err) {
			return NotFoundErr
		}
		return fmt.Errorf("failed get chaos, %s", err.Error())
	}

	if err := c.kubeCli.Delete(ctx, chaos); err != nil {
		return fmt.Errorf("failed delete chaos, %s", err.Error())
	}
	return nil
}

// DescribeExperimentWithEvents 查询实验，包含实验事件集合
func (c *client) DescribeExperimentWithEvents(ctx context.Context, namespace, name, kind string) (*Experiment, error) {
	experiment, err := c.DescribeExperiment(ctx, namespace, name, kind)
	if err != nil {
		return nil, err
	}

	events, err := listExperimentEvents(c.kubeCli, experiment.UID, "", "")
	if err != nil {
		return nil, err
	}
	experiment.Events = events
	return experiment, nil
}

// DescribeExperiment 查询实验
func (c *client) DescribeExperiment(ctx context.Context, namespace, name, kind string) (*Experiment, error) {
	chaosKind, exists := v1alpha1.AllKinds()[kind]
	if !exists {
		return nil, fmt.Errorf("unknwon chaos kind '%s'", kind)
	}

	chaos := chaosKind.SpawnObject()
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}
	if err := c.kubeCli.Get(ctx, namespacedName, chaos); err != nil {
		if isNotFound(err) {
			return nil, NotFoundErr
		}
		return nil, err
	}

	experiment := &Experiment{
		Namespace: reflect.ValueOf(chaos).MethodByName("GetNamespace").Call(nil)[0].String(),
		Name:      reflect.ValueOf(chaos).MethodByName("GetName").Call(nil)[0].String(),
		Kind:      kind,
		UID:       reflect.ValueOf(chaos).MethodByName("GetUID").Call(nil)[0].String(),
		Created: reflect.ValueOf(chaos).MethodByName("GetCreationTimestamp").
			Call(nil)[0].Interface().(metav1.Time).Format(time.RFC3339),
		Status: getChaosStatus(chaos.(v1alpha1.InnerObject)),
	}
	return experiment, nil
}

// ListExperiments 检索实验
func (c *client) ListExperiments(ctx context.Context, kind string) ([]*Experiment, error) {
	chaosKind, exists := v1alpha1.AllKinds()[kind]
	if !exists {
		return nil, fmt.Errorf("unknwon chaos kind '%s'", kind)
	}

	list := chaosKind.SpawnList()
	listOptions := &pkgclient.ListOptions{Namespace: "chaos-testing"}
	if err := c.kubeCli.List(ctx, list, listOptions); err != nil {
		return nil, err
	}

	experimentList := make([]*Experiment, 0)
	for _, item := range list.GetItems() {
		chaosName := item.GetName()
		experimentList = append(experimentList, &Experiment{
			Namespace: item.GetNamespace(),
			Name:      chaosName,
			Kind:      item.GetObjectKind().GroupVersionKind().Kind,
			UID:       string(item.GetUID()),
			Created:   item.GetCreationTimestamp().Format(time.RFC3339),
			Status:    getChaosStatus(item.(v1alpha1.InnerObject)),
		})
	}
	return experimentList, nil
}

// ListEvents 检索实验的事件
func (c *client) ListEvents(ctx context.Context, experimentUid, eventType, reason string) ([]v1.Event, error) {
	return listExperimentEvents(c.kubeCli, experimentUid, eventType, reason)
}

func listExperimentEvents(cli pkgclient.Client, experimentUid, eventType, reason string) ([]v1.Event, error) {
	list := &v1.EventList{}
	filter := map[string]string{
		"involvedObject.uid": experimentUid,
	}
	if eventType != "" {
		filter["type"] = eventType
	}
	if reason != "" {
		filter["reason"] = reason
	}
	options := &pkgclient.ListOptions{
		Raw: &metav1.ListOptions{
			FieldSelector: labels.SelectorFromSet(filter).String(),
		},
	}
	if err := cli.List(context.Background(), list, options); err != nil {
		return nil, err
	}

	return list.Items, nil
}

func getChaosStatus(obj v1alpha1.InnerObject) *v1alpha1.ChaosStatus {
	return obj.GetStatus()
}

// NewClient
// @Description create chaos mesh client
// @Author marsdong 2022-04-20 19:09:53
// @param logger
// @return Client
// @return error
func NewClient() (Client, error) {
	cfg := ctrl.GetConfigOrDie()
	cli, err := pkgclient.New(cfg, pkgclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}
	return &client{
		kubeCli: cli,
	}, nil
}

// NewClientOrDie
// @Description create chaos mesh client
// panic if error happens
// @Author marsdong 2022-04-21 15:14:28
// @return Client
func NewClientOrDie() Client {
	cli, err := NewClient()
	if err != nil {
		panic(err)
	}
	return cli
}

func isNotFound(err error) bool {
	statusErr, ok := err.(*errors.StatusError)
	if !ok {
		return false
	}
	return statusErr.ErrStatus.Code == http.StatusNotFound
}

