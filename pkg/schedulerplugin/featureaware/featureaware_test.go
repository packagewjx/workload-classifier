package featureaware

import (
	"context"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal/server"
	"github.com/packagewjx/workload-classifier/pkg/core"
	server2 "github.com/packagewjx/workload-classifier/pkg/server"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/events"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	metrics "k8s.io/metrics/pkg/apis/metrics/v1alpha1"
	"testing"
)

const nodeCpuCapacity = 4
const nodeMemCapacity = 2000
const namespaceTest = "test"

func TestFilterWithNormalRequirement(t *testing.T) {
	plugin, _ := New(nil, nil)
	featurePlugin := plugin.(*featureAwarePlugin)

	schedulePod := &corev1.Pod{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      "to-be-schedule",
			Namespace: namespaceTest,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
							corev1.ResourceMemory: *resource.NewQuantity(500, resource.BinarySI),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			QOSClass: corev1.PodQOSBestEffort,
		},
	}

	requestList := []requirement{
		{
			cpu: 1,
			mem: 1000,
		},
		{
			cpu: 1,
			mem: 300,
		},
		{
			cpu: 1,
			mem: 200,
		},
	}

	requirementMap := make(map[string]requirement)

	var nodePods []*framework.PodInfo
	nodePods = makePods(requestList)

	featurePlugin.client = &fakeApi{requirementMap: requirementMap}

	nodeInfo := makeNodeInfo(nodePods, nodeCpuCapacity, nodeMemCapacity)

	result := featurePlugin.Filter(context.Background(), framework.NewCycleState(), schedulePod, nodeInfo)
	assert.Condition(t, func() (success bool) {
		return result.Code() == framework.Success
	})
}

func TestFilterWithIdleResource(t *testing.T) {
	requestList := []requirement{
		{
			cpu: 1,
			mem: 1000,
		},
		{
			cpu: 1,
			mem: 300,
		},
		{
			cpu: 1,
			mem: 200,
		},
	}

	actualUseList := []requirement{
		{
			cpu: 0.5,
			mem: 800,
		},
		{
			cpu: 0.2,
			mem: 200,
		},
		{
			cpu: 0.8,
			mem: 200,
		},
	}

	requirementMap := make(map[string]requirement)
	nodePods := makePods(requestList)
	for i := 0; i < len(nodePods); i++ {
		requirementMap[nodePods[i].Pod.Name] = actualUseList[i]
	}

	schedulePod := &corev1.Pod{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      "to-be-schedule",
			Namespace: namespaceTest,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewQuantity(2, resource.DecimalSI),
							corev1.ResourceMemory: *resource.NewQuantity(800, resource.BinarySI),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			QOSClass: corev1.PodQOSBestEffort,
		},
	}

	nodeInfo := makeNodeInfo(nodePods, nodeCpuCapacity, nodeMemCapacity)

	plugin, _ := New(nil, nil)
	featureAware := plugin.(*featureAwarePlugin)
	featureAware.client = &fakeApi{requirementMap: requirementMap}

	result := featureAware.Filter(context.Background(), framework.NewCycleState(), schedulePod, nodeInfo)
	assert.Condition(t, func() (success bool) {
		return result.Code() == framework.Success
	})

	schedulePod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewQuantity(2, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(1000, resource.BinarySI),
	}
	result = featureAware.Filter(context.Background(), framework.NewCycleState(), schedulePod, nodeInfo)
	assert.Condition(t, func() (success bool) {
		return result.Code() == framework.Unschedulable
	})
	println(result.Message())
}

func makePods(requestList []requirement) []*framework.PodInfo {
	nodePods := make([]*framework.PodInfo, 0, len(requestList))
	for i, s := range requestList {
		name := fmt.Sprintf("test-%d", i)
		nodePods = append(nodePods, &framework.PodInfo{
			Pod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespaceTest,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       server.KindReplicaSet,
							Name:       name,
						},
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewQuantity(int64(s.cpu), resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(int64(s.mem), resource.BinarySI),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					QOSClass: corev1.PodQOSGuaranteed,
				},
			},
		})
	}
	return nodePods
}

func TestNodeUtilization(t *testing.T) {
	nodeInfo := &framework.NodeInfo{}
	nodeInfo.SetNode(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewMilliQuantity(4000, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(2000, resource.BinarySI),
			},
		},
	})
	fakeClient := &fakeMetricsClient{
		nodeCpu: 2,
		nodeMem: 2000,
	}

	result, _ := nodeUtilization(nodeInfo, fakeClient)
	assert.Equal(t, float32(0.75), result)
}

func TestScore(t *testing.T) {
	plugin, _ := New(nil, nil)
	featurePlugin := plugin.(*featureAwarePlugin)
	featurePlugin.handle = &fakeHandle{
		lister: &fakeSharedLister{
			nodeInfoLister: &fakeNodeInfoLister{
				node: &corev1.Node{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       corev1.NodeSpec{},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewQuantity(4, resource.DecimalSI),
							corev1.ResourceMemory: *resource.NewQuantity(4000, resource.BinarySI),
						},
					},
				},
			}},
	}
	featurePlugin.metricsClient = fakeMetricsClient{
		nodeCpu: 2,
		nodeMem: 2000,
	}

	score, status := featurePlugin.Score(context.Background(), framework.NewCycleState(), &corev1.Pod{}, "test")
	assert.Condition(t, func() (success bool) {
		return status.Code() == framework.Success
	})

	assert.Equal(t, int64(50), score)
}

func makeNodeInfo(pods []*framework.PodInfo, nodeCpu int64, nodeMem int64) *framework.NodeInfo {
	cpuRequest := int64(0)
	memRequest := int64(0)
	for _, pod := range pods {
		for _, container := range pod.Pod.Spec.Containers {
			cpuRequest += container.Resources.Requests.Cpu().Value()
			memRequest += container.Resources.Requests.Memory().Value()
		}
	}

	nodeInfo := &framework.NodeInfo{
		Pods:             pods,
		PodsWithAffinity: nil,
		UsedPorts:        nil,
		Requested: &framework.Resource{
			MilliCPU: 1000 * cpuRequest,
			Memory:   memRequest,
		},
		NonZeroRequested: &framework.Resource{
			MilliCPU: 1000 * cpuRequest,
			Memory:   memRequest,
		},
		Allocatable: &framework.Resource{
			MilliCPU: 1000 * nodeCpu,
			Memory:   nodeMem,
		},
	}
	return nodeInfo
}

type requirement struct {
	cpu float32
	mem float32
}

type fakeApi struct {
	requirementMap map[string]requirement
}

func (f *fakeApi) QueryAppCharacteristics(appName server2.AppName) (*server2.AppCharacteristics, error) {
	req := f.requirementMap[appName.Name]
	sectionData := make([]*core.SectionData, core.NumSections)
	for i := 0; i < len(sectionData); i++ {
		sectionData[i] = &core.SectionData{
			CpuAvg: req.cpu,
			CpuMax: req.cpu,
			CpuMin: req.cpu,
			CpuP50: req.cpu,
			CpuP90: req.cpu,
			CpuP99: req.cpu,
			MemAvg: req.mem,
			MemMax: req.mem,
			MemMin: req.mem,
			MemP50: req.mem,
			MemP90: req.mem,
			MemP99: req.mem,
		}
	}
	return &server2.AppCharacteristics{
		AppName:     appName,
		SectionData: sectionData,
	}, nil
}

func (f *fakeApi) ReCluster() {
	panic("implement me")
}

type fakeMetricsClient struct {
	nodeCpu int64
	nodeMem int64
}

func (f fakeMetricsClient) QueryAllPodMetrics() (*metrics.PodMetricsList, error) {
	panic("implement me")
}

func (f fakeMetricsClient) QueryNodeMetrics(_ string) (*metrics.NodeMetrics, error) {
	return &metrics.NodeMetrics{
		Usage: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewQuantity(f.nodeCpu, resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(f.nodeMem, resource.BinarySI),
		},
	}, nil
}

type fakeHandle struct {
	lister framework.SharedLister
}

func (f *fakeHandle) SnapshotSharedLister() framework.SharedLister {
	return f.lister
}

func (f fakeHandle) IterateOverWaitingPods(callback func(framework.WaitingPod)) {
	panic("implement me")
}

func (f fakeHandle) GetWaitingPod(uid types.UID) framework.WaitingPod {
	panic("implement me")
}

func (f fakeHandle) RejectWaitingPod(uid types.UID) {
	panic("implement me")
}

func (f fakeHandle) ClientSet() kubernetes.Interface {
	panic("implement me")
}

func (f fakeHandle) EventRecorder() events.EventRecorder {
	panic("implement me")
}

func (f fakeHandle) SharedInformerFactory() informers.SharedInformerFactory {
	panic("implement me")
}

func (f fakeHandle) PreemptHandle() framework.PreemptHandle {
	panic("implement me")
}

type fakeSharedLister struct {
	nodeInfoLister framework.NodeInfoLister
}

func (f *fakeSharedLister) NodeInfos() framework.NodeInfoLister {
	return f.nodeInfoLister
}

type fakeNodeInfoLister struct {
	node *corev1.Node
}

func (f fakeNodeInfoLister) List() ([]*framework.NodeInfo, error) {
	panic("implement me")
}

func (f fakeNodeInfoLister) HavePodsWithAffinityList() ([]*framework.NodeInfo, error) {
	panic("implement me")
}

func (f *fakeNodeInfoLister) Get(nodeName string) (*framework.NodeInfo, error) {
	res := &framework.NodeInfo{}
	res.SetNode(f.node)
	return res, nil
}
