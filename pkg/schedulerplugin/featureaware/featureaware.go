package featureaware

import (
	"context"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal/server"
	"github.com/packagewjx/workload-classifier/pkg/client"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/packagewjx/workload-classifier/pkg/metricsclient"
	server2 "github.com/packagewjx/workload-classifier/pkg/server"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	"time"
)

var _ framework.FilterPlugin = &featureAwarePlugin{}
var _ framework.ScorePlugin = &featureAwarePlugin{}

const PluginName = "FeatureAware"

type featureAwarePlugin struct {
	handle        framework.FrameworkHandle
	client        server2.API
	metricsClient metricsclient.Client
}

func (f *featureAwarePlugin) Score(_ context.Context, _ *framework.CycleState, _ *corev1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := f.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil || nodeInfo == nil {
		return 0, framework.NewStatus(framework.Unschedulable, "获取NodeInfo出错")
	}

	utilization, err := nodeUtilization(nodeInfo, f.metricsClient)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("计算使用率错误：%v", err))
	}

	// 利用率越低，分数越高
	return 100 - int64(utilization*100), framework.NewStatus(framework.Success)
}

func (f *featureAwarePlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func New(_ runtime.Object, handle framework.FrameworkHandle) (framework.Plugin, error) {
	return &featureAwarePlugin{
		client:        client.NewApiClient(),
		handle:        handle,
		metricsClient: metricsclient.NewHttpMetricsClient(metricsclient.DefaultKubeApiServerBaseUrl),
	}, nil
}

func (f *featureAwarePlugin) Name() string {
	return PluginName
}

func (f *featureAwarePlugin) Filter(_ context.Context, _ *framework.CycleState, pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	if pod.Status.QOSClass == corev1.PodQOSBestEffort {
		// 离线任务处理逻辑
		cpuIdle := float32(0)
		memIdle := float32(0)
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
			cpuIdle += cpu - data.CpuMax
			memIdle += mem - data.MemMax
		}

		cpuAllocatable := cpuIdle + float32(nodeInfo.Allocatable.MilliCPU-nodeInfo.Requested.MilliCPU)/1000
		memAllocatable := memIdle + float32(nodeInfo.Allocatable.Memory-nodeInfo.Requested.Memory)

		if cpuAllocatable < podCpuRequest || memAllocatable < podMemRequest {
			return framework.NewStatus(framework.Unschedulable,
				fmt.Sprintf("资源不足。可分配CPU：%f，可分配内存：%f。需要CPU：%f，需要内存：%f",
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

// 计算节点的总资源利用率。返回值范围0到1
func nodeUtilization(nodeInfo *framework.NodeInfo, metricsClient metricsclient.Client) (float32, error) {
	allocatable := nodeInfo.Allocatable

	// 获取节点的监控数据
	metrics, err := metricsClient.QueryNodeMetrics(nodeInfo.Node().Name)
	if err != nil {
		return 0, err
	}

	usage := metrics.Usage

	util := (float32(usage.Cpu().MilliValue())/float32(allocatable.MilliCPU) + float32(usage.Memory().Value())/float32(allocatable.Memory)) / 2

	return util, nil
}
