package featureaware

import (
	"context"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal/server"
	"github.com/packagewjx/workload-classifier/pkg/client"
	"github.com/packagewjx/workload-classifier/pkg/core"
	server2 "github.com/packagewjx/workload-classifier/pkg/server"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	"time"
)

var _ framework.FilterPlugin = &featureAwarePlugin{}
var _ framework.ScorePlugin = &featureAwarePlugin{}

type featureAwarePlugin struct {
	client server2.API
}

func (f *featureAwarePlugin) Score(ctx context.Context, state *framework.CycleState, p *corev1.Pod, nodeName string) (int64, *framework.Status) {
	panic("implement me")
}

func (f *featureAwarePlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func New(_ runtime.Object, _ framework.FrameworkHandle) (framework.Plugin, error) {
	return &featureAwarePlugin{
		client: client.NewApiClient(),
	}, nil
}

func (f *featureAwarePlugin) Name() string {
	return "FeatureAware"
}

func (f *featureAwarePlugin) Filter(_ context.Context, _ *framework.CycleState, pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	if pod.Status.QOSClass == corev1.PodQOSBestEffort {
		// 离线任务处理逻辑
		cpuAllocatable := float32(0)
		memAllocatable := float32(0)
		sectionIdx := time.Now().Unix() % core.DayLength / core.SectionLength

		podCpuRequest, podMemRequest := podTotalRequest(pod)

		// 计算所有在线服务的空闲CPU
		for _, info := range nodeInfo.Pods {
			p := info.Pod
			if p.Status.QOSClass == corev1.PodQOSBestEffort {
				continue
			}

			appName := p.Name
			for _, reference := range p.OwnerReferences {
				switch reference.Kind {
				case server.KindDaemonSet, server.KindStatefulSet, server.KindReplicaSet:
					appName = reference.Name
				default:
					continue
				}
				break
			}

			characteristics, err := f.client.QueryAppCharacteristics(server2.AppName{
				Name:      appName,
				Namespace: p.Namespace,
			})
			if err != nil {
				fmt.Printf("获取%s名称空间的%s的Pod的分类信息失败：%v\n", p.Namespace, p.Name, err)
				continue
			}

			cpu, mem := podTotalRequest(p)

			// 可能没有赋值，此时将会默认为节点的总值
			if cpu == 0 {
				cpu = float32(nodeInfo.Allocatable.MilliCPU) / 1000
			}
			if mem == 0 {
				mem = float32(nodeInfo.Allocatable.Memory)
			}

			data := characteristics.SectionData[sectionIdx]
			cpuAllocatable += cpu - data.CpuMax
			memAllocatable += mem - data.MemMax
		}

		cpuAllocatable += float32(nodeInfo.Allocatable.MilliCPU) / 1000
		memAllocatable += float32(nodeInfo.Allocatable.Memory)
		if cpuAllocatable < podCpuRequest || memAllocatable < podMemRequest {
			return framework.NewStatus(framework.Unschedulable,
				fmt.Sprintf("空闲可用CPU：%f，空闲可用内存：%f。需要CPU：%f，需要内存：%f",
					cpuAllocatable, memAllocatable, podCpuRequest, podMemRequest))
		}
	} else {
		// 在线服务处理逻辑空，资源足够即可，这一点在其他插件得到验证
	}

	return framework.NewStatus(framework.Success)
}

func podTotalRequest(p *corev1.Pod) (cpu, mem float32) {
	for _, container := range p.Spec.Containers {
		cpu += float32(container.Resources.Requests.Cpu().MilliValue()) / 1000
		mem += float32(container.Resources.Requests.Memory().Value())
	}
	return
}
